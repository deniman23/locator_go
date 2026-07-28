# Аудит бизнес-анализа: Locator — полный отчёт по домену

**Дата:** 28 июля 2026  
**Аудитор:** business-analyst subagent  
**Версия:** MVP health-check, read-only

---

## 1. Scope & Method — что проверялось

### Охват

| Область | Файлы / пути | Статус |
|---|---|---|
| Бизнес-правила геозон | `backend/service/geofence_*.go`, `visit_event_processor.go` | ✅ Проверено |
| Модель данных | `backend/models/`, `backend/migrations/` | ✅ Проверено |
| Авторизация и роли | `backend/middleware/middleware.go`, `backend/service/user_service.go` | ✅ Проверено |
| API-контракт | `backend/router/routes.go`, `backend/controllers/` | ✅ Проверено |
| Сервисный слой | `backend/service/` (все файлы) | ✅ Проверено |
| DAO-слой | `backend/dao/` | Частично |
| Фронтенд (admin UI) | `frontend/src/pages/`, `frontend/src/context/AuthContext.tsx`, `frontend/src/services/api.ts` | ✅ Проверено |
| Инфраструктура | `docker-compose.yml`, `backend/Dockerfile`, `deploy.sh` | ✅ Проверено |
| Документация | `docs/TESTING.md`, `docs/AGENT_PHONE_SETUP.md`, `AGENTS.md` | ✅ Проверено |
| Миграции БД | `backend/migrations/` (8 файлов) | ✅ Проверено |
| Процесс деплоя | `deploy.sh`, `backend/entrypoint.sh` | ✅ Проверено |

### Метод

Статический анализ кода с помощью прямого чтения файлов (CodeGraph использовался для ориентирования по архитектуре). Проверка: модели данных ↔ миграции ↔ бизнес-правила ↔ API-контракт ↔ UI. Секреты не извлекались.

---

## 2. Находки по уровням серьёзности

---

### 🔴 BLOCKER

---

#### BA-001 — Линейный bcrypt-скан на каждый HTTP-запрос (O(N × bcrypt))

**Файл:** `backend/service/user_service.go`, функция `AuthenticateUser`  
**Доказательство:**
```
// строки 89-121
users, err := svc.DAO.GetAll()      // SELECT * FROM users
for _, user := range users {
    if err := bcrypt.CompareHashAndPassword(...) {  // ~50–100 мс на итерацию
```
Каждый защищённый endpoint (все маршруты через `BasicAuthMiddleware` и `APIKeyAuthMiddleware`) выполняет полный `SELECT * FROM users`, затем bcrypt-сравнение для каждого пользователя по очереди. При 20 пользователях — 1–2 с на запрос; при 100 пользователях — 5–10 с. Android-приложение опрашивает `GET /device/poll` каждые ~15 с с каждого устройства.

**Бизнес-воздействие:** Критическое. Деградация под нагрузкой превращается в самоподдерживающийся DoS: чем медленнее ответы, тем больше запросов накапливается в очереди. При 10 устройствах и 15-секундном poll-интервале сервер обрабатывает ~40 bcrypt-операций/мин. При 50 устройствах — ~200/мин, что делает систему практически неотзывчивой для других запросов.

**Рекомендация:** Добавить уникальный индекс на `api_key` и хранить в `api_key` **хэш**, доступный для поиска. Либо перейти на lookup по prefix-токену: первые N символов (plaintext prefix) → lookup по prefix в БД → bcrypt только для найденной записи. Итого: 1 SELECT + 1 bcrypt вместо N bcrypt.

---

#### BA-002 — Пароль базы данных вшит в образ Docker

**Файл:** `backend/Dockerfile`, строки ~45–48  
**Доказательство:**
```dockerfile
ENV \
    DB_PASSWORD=***REDACTED*** \
    ...
```
Пароль присутствует в git-истории и будет включён в каждый собранный Docker-образ, опубликованный в реестре.

**Бизнес-воздействие:** Критическое. Любой, у кого есть доступ к репозиторию или Docker-образу, получает учётные данные БД. При компрометации — полный доступ ко всем данным о местоположении пользователей.

**Рекомендация:** Удалить `DB_PASSWORD` из `ENV` в Dockerfile. Обеспечить передачу секрета **исключительно** через runtime env (`.env` файл или Secrets-менеджер). Обязательно инвалидировать текущий пароль в продуктовой БД.

---

### 🟠 HIGH

---

#### BA-003 — Двойная стратегия миграции: Goose + GORM AutoMigrate работают одновременно

**Файлы:** `backend/config/bootstrap/initializer.go` (строка 35 `AutoMigrate`), `backend/Dockerfile` (строка ~52, `goose ... up`)  
**Доказательство:** Entrypoint запускает `goose` (SQL-миграции) перед стартом бинарника. Затем `InitializeApp` вызывает `dbConn.AutoMigrate(...)`. Это означает, что схема определяется в двух местах одновременно.

Пример потенциального конфликта: миграция `20250501000000_create_user_table.sql.sql` создаёт `qr_code VARCHAR(255) NOT NULL`, но GORM-модель имеет `gorm:"type:text"` — различие в типе колонки. AutoMigrate может попытаться изменить тип, что в PostgreSQL может завершиться ошибкой или блокировкой таблицы.

**Бизнес-воздействие:** Высокое. Непредсказуемое поведение при деплое: сбой AutoMigrate → сервер не стартует; конкурентное изменение схемы → потеря данных на prod. Также: имя файла `20250501000000_create_user_table.sql.sql` (двойное `.sql`) нарушает соглашение goose, что может привести к его пропуску.

**Рекомендация:** Выбрать одну стратегию. Рекомендуется оставить **только Goose** (явный контроль, возможность отката). Убрать `AutoMigrate` из `initializer.go`. Переименовать файл с двойным расширением.

---

#### BA-004 — Отсутствуют DELETE-эндпоинты для чекпоинтов и деактивация пользователей

**Файл:** `backend/router/routes.go`  
**Доказательство:**
```go
checkpointGroup.GET("/", ...)
checkpointGroup.POST("/", ...)
checkpointGroup.PUT("/:id", ...)
// DELETE /:id — ОТСУТСТВУЕТ

userGroup.POST("/", ...)
userGroup.PUT("/:id", ...)
userGroup.GET("/:id", ...)
// DELETE / деактивация — ОТСУТСТВУЕТ
```
Модель `User` не содержит полей `is_active`, `disabled_at`, `deleted_at` (`backend/models/user.go`).

**Бизнес-воздействие:** Высокое. Операционные блокеры:
- Нет возможности удалить ошибочно созданный или устаревший чекпоинт — накапливается «мусор», влияющий на visit-детектор (геозона срабатывает для всех существующих чекпоинтов на каждое событие локации).
- Нет способа деактивировать уволенного сотрудника: устройство продолжает присылать координаты, система фиксирует визиты.

**Рекомендация:**
- Добавить `DELETE /checkpoint/:id` (проверить наличие активных визитов, при необходимости каскадно завершить).
- Добавить `is_active bool` в модель `User`, `PATCH /admin/users/:id/deactivate`. Заблокировать аутентификацию для `is_active=false` в `AuthenticateUser`.

---

#### BA-005 — Plaintext API-ключ постоянно хранится в QR-PNG на диске сервера

**Файл:** `backend/service/user_service.go`, функция `writeUserQRCode`  
**Доказательство:**
```go
qrContent := fmt.Sprintf(`{"user_id": %d, "api_key": "%s", "api_base_url": "%s"}`,
    userID, plainKey, apiBase)
qrFilePath := fmt.Sprintf("static/qrcode/%d.png", userID)
qrcode.WriteFile(qrContent, qrcode.Medium, 256, qrFilePath)
```
QR-файл доступен по публичному URL `GET /static/qrcode/:id.png` (маршрут `router.Static("/static", "./static")`). Несмотря на `no-cache`-заголовки, файл физически существует на диске и доступен всем, кто знает URL или имеет доступ к volume.

**Бизнес-воздействие:** Высокое. Любой пользователь с доступом к файловой системе сервера или volume получает plaintext ключи всех пользователей. При компрометации одного из них злоумышленник может имитировать устройство и вводить поддельные координаты.

**Рекомендация:** Не хранить plaintext ключ в QR-файле дольше, чем необходимо. Рассмотреть: генерация QR on-demand (не на диск), с подписанным временным URL. Как минимум — ограничить доступ к `static/qrcode/` admin-only middleware.

---

#### BA-006 — Состояние геозоны хранится только в памяти процесса

**Файл:** `backend/service/geofence_state.go`  
**Доказательство:**
```go
type geofenceStateStore struct {
    mu    sync.Mutex
    items map[string]*geofencePendingState  // только in-memory
}
```
`geofencePendingState` содержит `pendingEnterSince` и `pendingExitSince` — таймеры grace-периода для входа/выхода из чекпоинта. При перезапуске контейнера (деплой, краш) все pending-состояния теряются.

**Бизнес-воздействие:** Высокое. После каждого деплоя (push → server → `docker compose up --build`) система «забывает» все незавершённые переходы. Пользователи, находившиеся в зоне чекпоинта в момент рестарта, не получат визита (или получат дублирующий). В системе мониторинга посещений детского заведения или рабочего объекта — это прямая потеря данных о присутствии.

**Рекомендация:** Персистировать pending-состояния в PostgreSQL или Redis (простая таблица `geofence_pending_states(user_id, checkpoint_id, pending_enter_since, pending_exit_since)`). Загружать при старте сервиса.

---

#### BA-007 — `CheckUserInCheckpoint` использует устаревшую последнюю позицию, а не текущую

**Файл:** `backend/controllers/checkpoint_controller.go`, функция `CheckUserInCheckpoint`  
**Доказательство:**
```go
loc, err := cc.LocationService.GetLocation(userID)  // ПОСЛЕДНЯЯ известная точка
// ...
event := models.LocationEvent{..., OccurredAt: time.Now()}  // но время — СЕЙЧАС
```
Эндпоинт `GET /checkpoint/check` извлекает последнюю GPS-точку пользователя (которая может быть получена часы или дни назад) и публикует событие в RabbitMQ с текущим временем `OccurredAt`. Обработчик `VisitEventProcessor` создаст или закроет визит на основе этих устаревших координат.

**Бизнес-воздействие:** Высокое. Вызов этого endpoint из UI (кнопка «Проверить» в интерфейсе) может создать фантомный визит — пользователь давно ушёл, но система зафиксирует его присутствие в текущий момент.

**Рекомендация:** Добавить проверку свежести `loc` (`age_seconds <= threshold`). Если координата устарела — вернуть `409 Conflict` с полем `age_seconds` вместо публикации события. Либо исключить этот эндпоинт из UI и полагаться только на автоматический RabbitMQ-pipeline.

---

#### BA-008 — Отсутствует rate limiting для аутентификации и device poll

**Файл:** `backend/middleware/middleware.go`, `backend/router/routes.go`  
**Доказательство:** Ни в middleware, ни в роутере нет ни одного упоминания rate-limiting, throttling или IP-based quota. Эндпоинты `POST /location`, `GET /device/poll`, `POST /device/report` доступны любому клиенту без ограничений.

**Бизнес-воздействие:** Высокое. В сочетании с BA-001 (bcrypt-линейный скан) любой неавторизованный клиент с неверным ключом может исчерпать CPU сервера путём параллельного спама запросами.

**Рекомендация:** Добавить middleware с rate limiting на уровне IP (gin-contrib/ratelimit или аналог). Критически важно для `/api/location` (periodic от устройств) и для аутентификационных проверок.

---

### 🟡 MEDIUM

---

#### BA-009 — Таблица `location_events` создана, но не используется (dead table)

**Файл:** `backend/migrations/20250611090001_create_location_events_table.sql`  
**Доказательство:** Миграция создаёт таблицу с полем `processed BOOLEAN`. Модель `LocationEvent` (`backend/models/event.go`) используется **только** как RabbitMQ-сообщение — нигде в DAOs нет записи в эту таблицу. Поиск по всему backend/ не нашёл инсертов в `location_events`.

**Бизнес-воздействие:** Среднее. Путаница для новых разработчиков (ожидают persistence, не находят). Потенциал для дивергенции при будущем расширении: кто-то добавит запись в таблицу без понимания контракта.

**Рекомендация:** Либо начать использовать таблицу для персистирования событий (решает BA-006), либо добавить миграцию `DROP TABLE location_events` и удалить упоминания. Задокументировать архитектурное решение в `AGENTS.md`.

---

#### BA-010 — Жёстко закодированный часовой пояс `Europe/Minsk` в бизнес-логике

**Файлы:** `backend/service/location_service.go` (строка `loc, err := time.LoadLocation("Europe/Minsk")`), `backend/service/visit_service.go` (функция `parseVisitQueryRange`)  
**Доказательство:**
```go
loc, err := time.LoadLocation("Europe/Minsk")
// ...
log.Fatalf("Ошибка загрузки временной зоны Europe/Minsk: %v", err)
```

**Бизнес-воздействие:** Среднее. Если сервер деплоится в другом регионе или пользователи находятся в другом часовом поясе — все временны́е границы визитов и фильтры отображаются некорректно. Исторические данные невозможно перенести без пересчёта всех временны́х меток.

**Рекомендация:** Вынести TZ в env-переменную `APP_TIMEZONE` (по умолчанию `Europe/Minsk`). Сохранять все временны́е метки в UTC (PostgreSQL `TIMESTAMP WITH TIME ZONE`). Применять TZ только в presentation-слое.

---

#### BA-011 — Алгоритм `filterSignificantLocations` может сделать пользователя невидимым на карте

**Файл:** `backend/service/location_service.go`, функция `filterSignificantLocations`  
**Доказательство:**
```go
const (
    maxDistance = 100.0            // кластер в 100 м
    minDuration = 15 * time.Minute // минимум 15 мин стоянки
    minPoints   = 3                // минимум 3 точки
)
```
Если пользователь передвигается непрерывно (без остановок на 15+ мин), алгоритм кластеризации не находит кластеров и возвращает `getRepresentativePoints` — максимум 10 точек, с минимальным расстоянием 500 м. При `GET /location/` это единственный ответ.

**Бизнес-воздействие:** Среднее. Основная бизнес-задача системы — мониторинг местоположения. Пользователь в движении может полностью исчезнуть с карты, либо его трек будет очень разрежённым. Для администратора, отслеживающего маршрут, это непосредственный провал цели продукта.

**Рекомендация:** Для режима "live tracking" всегда включать параметр `raw=true` в запросах карты. Документировать разницу между `/location/?raw=true` (все точки) и `/location/` (значимые кластеры). В UI по умолчанию использовать `raw=true` для текущего трека.

---

#### BA-012 — Нет пагинации для `GET /visits/` и `GET /location/`

**Файлы:** `backend/controllers/visit_controller.go`, `backend/controllers/location_controller.go`  
**Доказательство:** `GetVisitsByFilters` возвращает весь результат `DAO.GetVisits(...)` без `LIMIT/OFFSET`. Нет параметров `page`, `per_page`, `limit`, `cursor`.

**Бизнес-воздействие:** Среднее. При работе системы месяц–год таблица `locations` и `visits` может содержать миллионы записей. Один запрос без временного фильтра вернёт всё в памяти. Риск OOM-ошибки на сервере и зависания UI.

**Рекомендация:** Добавить `limit` (default: 500, max: 5000) и `offset` или cursor-pagination для `/visits/` и `/location/`. Добавить валидацию: если не передан `from`/`to`, обязательно требовать `limit`.

---

#### BA-013 — Plaintext API-ключ логируется при создании пользователя

**Файл:** `backend/seed/seed.go`, `backend/service/user_service.go`  
**Доказательство:**
```go
// seed.go
log.Printf("Дефолтный администратор успешно создан: %s (ID: %d). Plain API ключ: %s",
    user.Name, user.ID, plainKey)
```
Plaintext ключ попадает в `logs/app.log` (через `config.InitLogger`). Docker-volume `./backend/logs:/app/logs` монтируется на хост.

**Бизнес-воздействие:** Среднее. Доступ к лог-файлу раскрывает API-ключ администратора. При ротации ключа через `regenerate-qr` новый ключ также не логируется, но initial seed — логируется всегда.

**Рекомендация:** Удалить `plainKey` из лог-сообщения в seed. Заменить на: `log.Printf("Дефолтный администратор создан: ID=%d. API ключ выдан, не хранится в логах.", user.ID)`.

---

#### BA-014 — Нет возможности ручной коррекции визитов

**Файлы:** `backend/router/routes.go`, `backend/controllers/visit_controller.go`  
**Доказательство:**
```go
visitGroup.GET("/", visitController.GetVisitsByFilters)  // только чтение
// PUT, DELETE, PATCH — ОТСУТСТВУЮТ
```
Визиты создаются и закрываются **исключительно** автоматически через RabbitMQ-pipeline. Нет ни одного эндпоинта для корректировки.

**Бизнес-воздействие:** Среднее. GPS-дрейф у границы чекпоинта, временное выключение телефона, офлайн-очередь с устаревшим `captured_at` — всё это может создавать ложные или некорректные визиты. Без ручной коррекции оператор не может исправить данные для отчётности.

**Рекомендация:** Добавить `PUT /visits/:id` (изменить `start_at`, `end_at`, `duration`) и `DELETE /visits/:id` с ролью admin. Логировать изменения (audit trail).

---

#### BA-015 — Отсутствует механизм уведомлений/алертов

**Файлы:** `backend/service/`, `backend/controllers/` — полный поиск**  
**Доказательство:** Отсутствует любое упоминание webhook, notification, push, alert. Система фиксирует визиты, но никак не уведомляет об их начале или окончании.

**Бизнес-воздействие:** Среднее. Основные use-cases системы (родитель следит за ребёнком у школы, работодатель — за сотрудником на объекте) предполагают получение сигнала в реальном времени, а не постфактум через интерфейс. Текущая система требует активного мониторинга — значительная потеря ценности продукта.

**Рекомендация:** Добавить webhook-callback (URL, configurable per checkpoint) при входе/выходе из зоны. MVP: POST запрос на заданный URL с payload `{user_id, checkpoint_id, event: "enter"|"exit", occurred_at}`.

---

### 🔵 LOW

---

#### BA-016 — Три дублирующих реализации функции `haversineDistance`

**Файлы:**  
- `backend/service/checkpoint_service.go` — `haversineDistance(lat1, lon1, lat2, lon2)`  
- `backend/service/location_service.go` — `haversineDistance(lat1, lon1, lat2, lon2)`  
- `backend/service/track_filter.go` — `haversineDistanceM(lat1, lon1, lat2, lon2)`

**Рекомендация:** Вынести в `backend/internal/geo/haversine.go`. Снижает риск расхождения реализаций при изменении логики (например, переход к формуле Vincenty для точных расчётов).

---

#### BA-017 — `manifest.json` бэкается в Docker-образ, затрудняя OTA-процесс

**Файл:** `backend/Dockerfile`, строка `COPY static/releases/manifest.json /app/static/releases/manifest.json`

При каждом новом релизе APK нужно пересобирать Docker-образ. Если APK обновляется через `sync_release_manifest.sh` на сервере напрямую, образ и manifest расходятся. Volume `${APK_RELEASES_DIR}` монтируется в runtime — manifest в volume и в образе могут быть разными файлами.

**Рекомендация:** Убрать `COPY manifest.json` из Dockerfile. Manifest всегда должен приходить через монтируемый volume.

---

#### BA-018 — Нет индекса на `(user_id, start_at)` в таблице `visits`

**Файл:** `backend/migrations/20250611090000_create_visits_table.sql`  
**Доказательство:** Миграция создаёт только FK-constraints. Нет `CREATE INDEX` для наиболее частого паттерна запроса: `WHERE user_id = X AND start_at BETWEEN a AND b`.

**Рекомендация:** Добавить миграцию `CREATE INDEX idx_visits_user_start ON visits(user_id, start_at)`.

---

#### BA-019 — `capturedAt` с возрастом > 90 дней молча отклоняется

**Файл:** `backend/service/location_service.go`, функция `ParseCapturedAt`  
**Доказательство:**
```go
const capturedAtMaxAge = 90 * 24 * time.Hour
if t.Before(now.Add(-capturedAtMaxAge)) {
    return nil, fmt.Errorf("captured_at слишком старый (более 90 дней)")
}
```
Устройство, которое было оффлайн 91+ день (потерянный телефон, замена аккумулятора), при восстановлении соединения получит отказ для всех буферизованных точек. Бизнес-логика не документирует это ограничение для клиентского приложения.

**Рекомендация:** Документировать ограничение в `AGENT_PHONE_SETUP.md`. Рассмотреть увеличение до 180 дней или введение env-переменной `CAPTURED_AT_MAX_AGE_DAYS`.

---

#### BA-020 — Публичный OSRM используется по умолчанию для маршрутизации

**Файл:** `docker-compose.yml`  
**Доказательство:**
```yaml
ROUTING_BASE_URL: ${ROUTING_BASE_URL:-https://router.project-osrm.org}
```
Если оператор не установил свой OSRM, координаты пользователей отправляются на сторонний публичный сервис.

**Рекомендация:** Документировать это явно в `AGENTS.md` и `docker-compose.yml`. Добавить предупреждение при старте, если используется публичный OSRM (для производственного режима `GIN_MODE=release`).

---

### ℹ️ INFO

---

#### BA-021 — Нет формального API-контракта (OpenAPI/Swagger) между backend и Android-клиентом

**Файлы:** `backend/router/routes.go`, `docs/AGENT_PHONE_SETUP.md`  
API-контракт описан только в неформальной таблице в `AGENT_PHONE_SETUP.md`. Нет OpenAPI-спецификации. Синхронизация `locator_go` и `lctr_app` (отдельный репозиторий) при изменениях контракта — ручная.

---

#### BA-022 — RabbitMQ работает с дефолтными учётными данными `guest/guest`

**Файл:** `docker-compose.yml`, `backend/config/bootstrap/initializer.go`  
Допустимо для разработки, но должно быть явно задокументировано как обязательное изменение при продовом деплое.

---

#### BA-023 — Нет политики хранения данных локации (data retention)

**Файлы:** `backend/dao/`, `backend/service/` — нет schedules, cleanup jobs  
Таблица `locations` растёт неограниченно. Нет ни одного scheduled job, cron или TTL для очистки старых точек. Через год эксплуатации — потенциальные проблемы с производительностью и дисковым пространством.

---

## 3. Приоритизированный план устранения

| Приоритет | ID | Действие | Owner | Сложность |
|---|---|---|---|---|
| P0 | BA-002 | Удалить `DB_PASSWORD` из Dockerfile, инвалидировать текущий пароль | DevOps | S |
| P0 | BA-001 | Рефакторинг `AuthenticateUser`: lookup по prefix/hash вместо full-scan | Backend | M |
| P1 | BA-003 | Выбрать одну стратегию миграции (рекомендуется Goose only); убрать `AutoMigrate` | Backend | M |
| P1 | BA-005 | Ограничить доступ к `static/qrcode/` admin-only или перейти на on-demand QR | Backend+Security | M |
| P1 | BA-008 | Добавить rate limiting middleware (особенно для auth и device poll) | Backend | S |
| P2 | BA-004 | Добавить `DELETE /checkpoint/:id`, `is_active` для пользователей + деактивация | Backend | M |
| P2 | BA-006 | Персистировать geofence pending-состояния в БД | Backend | M |
| P2 | BA-013 | Убрать plaintext ключ из seed-логов | Backend | XS |
| P2 | BA-007 | Валидировать свежесть локации в `CheckUserInCheckpoint` | Backend | S |
| P3 | BA-012 | Добавить пагинацию в `GET /visits/` и `GET /location/` | Backend | M |
| P3 | BA-015 | MVP webhook-уведомлений при входе/выходе из чекпоинта | Backend | L |
| P3 | BA-014 | Добавить `PUT`/`DELETE /visits/:id` (admin-only) | Backend | S |
| P3 | BA-018 | Добавить индекс `(user_id, start_at)` на таблицу visits | Backend | XS |
| P4 | BA-009 | Убрать или начать использовать `location_events` таблицу | Backend | S |
| P4 | BA-011 | Документировать `raw=true` и использовать его по умолчанию в live-tracking | Frontend+Backend | S |
| P4 | BA-010 | Вынести TZ в env-переменную | Backend | S |
| P4 | BA-016 | Вынести `haversine` в `internal/geo` пакет | Backend | XS |
| P4 | BA-017 | Убрать `COPY manifest.json` из Dockerfile | Backend+DevOps | XS |
| P5 | BA-019 | Документировать ограничение `capturedAt` 90 дней | Docs | XS |
| P5 | BA-020 | Добавить предупреждение при использовании публичного OSRM в prod | Backend | XS |
| P5 | BA-021 | Сгенерировать OpenAPI-spec | Backend | M |
| P5 | BA-022 | Документировать обязательную смену RabbitMQ credentials | Docs | XS |
| P5 | BA-023 | Спроектировать и реализовать data retention policy для `locations` | Backend+DBA | M |

---

## 4. Критерии приёмки (Acceptance Checks)

### BA-001 (bcrypt-scan)
- [ ] `GET /device/poll` с любым ключом отвечает < 200 мс при 100 пользователях в БД
- [ ] Нагрузочный тест: 50 параллельных auth-запросов выполняются без деградации
- [ ] Неверный ключ получает `401` за < 100 мс

### BA-002 (пароль в Dockerfile)
- [ ] `git grep DB_PASSWORD backend/Dockerfile` не возвращает hardcoded значения
- [ ] `docker inspect locator-backend | grep DB_PASSWORD` не раскрывает пароль
- [ ] Старый пароль `***REDACTED***` не принимается PostgreSQL после ротации

### BA-003 (двойные миграции)
- [ ] `AutoMigrate` удалён из `initializer.go`
- [ ] Чистый запуск на пустой БД: goose применяет все миграции без ошибок
- [ ] `backend/migrations/20250501000000_create_user_table.sql.sql` переименован в `...sql`

### BA-004 (DELETE checkpoint / деактивация user)
- [ ] `DELETE /checkpoint/:id` возвращает `200` при удалении; `404` если не существует
- [ ] Удаление чекпоинта с активным визитом — `409 Conflict` с описанием
- [ ] `PATCH /admin/users/:id/deactivate` — устройство с деактивированным ключом получает `401`
- [ ] Деактивированный пользователь не отображается в списке активных устройств в UI

### BA-005 (QR-ключ на диске)
- [ ] `GET /static/qrcode/:id.png` без admin-ключа возвращает `401` или `404`
- [ ] Либо QR генерируется on-demand и не хранится в файловой системе

### BA-006 (geofence state persistence)
- [ ] После перезапуска backend активный визит, открытый до рестарта, продолжается (не дублируется)
- [ ] Pending-состояния восстанавливаются из БД при старте

### BA-007 (CheckUserInCheckpoint)
- [ ] Вызов с `age_seconds > threshold` возвращает `409` с полем `age_seconds` и `stale: true`
- [ ] Вызов со свежей координатой (< 5 мин) публикует событие нормально

### BA-012 (пагинация)
- [ ] `GET /visits/?limit=100&offset=0` работает; превышение `limit=5001` → `400 Bad Request`
- [ ] `GET /visits/` без `from`/`to` и без `limit` → `400 Bad Request` с описанием

### BA-013 (plaintext лог)
- [ ] `grep -i "plain api" logs/app.log` после создания admin — ничего не находит

---

## 5. Допущения и вне скоупа

### Допущения
- Анализ проведён на основе статического чтения кода; продовая БД и runtime-конфиг не проверялись.
- Android-клиент (`lctr_app`) находится в отдельном репозитории — его внутренняя логика не анализировалась.
- Целевое число пользователей не задано; оценки производительности (BA-001) основаны на типичном диапазоне 10–200 устройств.
- Данные о реальном трафике и объёме БД не доступны; выводы о производительности — расчётные.

### Вне скоупа
- Проверка безопасности Android-приложения (Device Owner mode, ADB-уязвимости) — зона security-engineer.
- Тестирование с реальными устройствами и GPS-данными.
- Финансовый ROI (отсутствует данные о стоимости инцидентов и объёме операций).
- Compliance / GDPR-анализ (обработка геолокации несовершеннолетних / сотрудников) — требует отдельного юридического аудита.
- Оценка производительности OSRM-маршрутизации и Mapbox/Leaflet на клиенте.

---

*Подготовлено: [BA](fab9f275-8128-4467-a2c6-ae6261fddd3f). Дополнение от [повторного BA](167475f1-3732-4e24-968f-0b04e2382c63): глобальный scope чекпоинтов (нет привязки к user → все зоны на всех пользователей); нет realtime-уведомлений enter/exit; reconciliation открытых визитов после рестарта; retention GPS.*


---
*Источник: [BA](fab9f275-8128-4467-a2c6-ae6261fddd3f). Дополнение [повторного BA](167475f1-3732-4e24-968f-0b04e2382c63): глобальный scope чекпоинтов; нет realtime enter/exit уведомлений; reconciliation открытых визитов после рестарта; GPS retention.*
