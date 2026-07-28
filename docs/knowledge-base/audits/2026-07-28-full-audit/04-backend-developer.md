# Аудит бэкенда Locator — Полный доменный отчёт

**Дата:** 28 июля 2026  
**Аудитор:** backend-developer (VoltAgent-enhanced)  
**Режим:** audit / read-only  

---

## 1. Scope & Method

### Охват

| Область | Статус |
|---|---|
| API-дизайн (router → controller → service → dao) | ✅ Проверено |
| Аутентификация / авторизация (middleware) | ✅ Проверено |
| Доменные модели и миграции | ✅ Проверено |
| RabbitMQ: publisher, consumer, processor | ✅ Проверено |
| Надёжность: graceful shutdown, error contracts | ✅ Проверено |
| Индексирование БД и производительность запросов | ✅ Проверено |
| Безопасность QR-кодов и хранения ключей | ✅ Проверено |
| Покрытие тестами | ✅ Проверено |
| Конфигурация и секреты | ✅ Проверено |

### Метод

- Первичная разведка через **CodeGraph MCP** (`codegraph_explore`) — граф символов, blast-radius
- Прямое чтение исходников: `backend/router/routes.go`, `middleware/middleware.go`, `service/user_service.go`, `config/bootstrap/initializer.go`, `config/messaging/`, `controllers/`, `dao/`, `models/`, `migrations/`, `integration/api_test.go`
- Статический анализ потоков данных, auth-путей, DB-запросов
- Без доступа к production-окружению, метрикам и логам

---

## 2. Findings

### 🔴 BLOCKER

---

#### B-01 · O(N) bcrypt-скан при каждом запросе — гарантированный DoS и timing-атака

**Файл:** `backend/service/user_service.go:89–123`

```go
users, err := svc.DAO.GetAll()           // загружает ВСЕх пользователей
for _, user := range users {
    if err := bcrypt.CompareHashAndPassword(…); err == nil {
        matchedUserID = user.ID
        break
    }
}
```

**Контекст:** `AuthenticateUser` вызывается из `BasicAuthMiddleware` и `APIKeyAuthMiddleware` на **каждый входящий запрос**. `DAO.GetAll()` делает `SELECT * FROM users` без WHERE, затем для каждой записи выполняется `bcrypt.CompareHashAndPassword` (намеренно медленная операция, ~100 мс на итерацию при cost=10). При N пользователях латентность middleware = N × ~100 мс.

**Влияние:**
- При 10 пользователях — ~1 с на запрос; при 50 — ~5 с.
- Один легитимный клиент, активно шлющий GPS-точки, может случайно исчерпать пул соединений БД.
- Timing-атака: злоумышленник может определить наличие пользователей по разнице времени ответа.

**Рекомендация:** Добавить в таблицу `users` колонку `api_key_prefix VARCHAR(8) NOT NULL` (первые 8 символов plaintext-ключа после генерации), создать по ней индекс. `AuthenticateUser` выполняет `WHERE api_key_prefix = $1`, получает ≤1 запись, один bcrypt-сравнение. Альтернатива без миграции: хранить отдельную таблицу `api_key_lookup(prefix, user_id)` или перейти на HMAC с lookup по хешу.

---

#### B-02 · QR-коды с API-ключами публично доступны без аутентификации

**Файлы:**
- `backend/service/user_service.go:164` — QR-код кодирует `{"user_id": N, "api_key": "<plaintext>", "api_base_url": "..."}`
- `backend/router/routes.go:39` — `router.Static("/static", "./static")` — без middleware
- `backend/main.go:17` — директория `static/qrcode/` создаётся при старте

**Цепочка атаки:**
1. Злоумышленник перебирает `GET /static/qrcode/1.png`, `GET /static/qrcode/2.png`, ...
2. Парсит QR-изображение → получает plaintext API-ключ любого пользователя.
3. Использует ключ для доступа к API от имени этого пользователя.

**Влияние:** Полная компрометация любого аккаунта без знания ключа. URL предсказуем (`/static/qrcode/{user_id}.png`).

**Рекомендация:** Убрать QR-файлы из-под `router.Static`. Отдавать их только через аутентифицированные эндпоинты (`/api/users/:id/qr-code-file` — уже существует, но параллельно существует и публичный static-путь). Либо не хранить PNG с ключами на диске вообще — генерировать QR в памяти на лету по запросу (аутентифицированному).

---

### 🟠 HIGH

---

#### H-01 · Plaintext API-ключ логируется при создании администратора

**Файл:** `backend/seed/seed.go:51`

```go
log.Printf("Дефолтный администратор успешно создан: %s (ID: %d). Plain API ключ: %s",
    user.Name, user.ID, plainKey)
```

**Влияние:** Plaintext ключ администратора попадает в `logs/app.log`. Если логи собираются централизованно (Docker stdout → Loki, Datadog, etc.), ключ хранится в plaintext в системе логирования. Любой, кто имеет доступ к логам, получает root-доступ к API.

**Рекомендация:** Убрать `plainKey` из лога. Выводить только `"Администратор создан: ID=%d, Name=%s"`. Ключ должен доставляться только через защищённый канал при первом запуске.

---

#### H-02 · Отсутствует rate limiting на всех эндпоинтах

**Файл:** `backend/router/routes.go` — ни один `Group` не имеет rate-limit middleware.

**Влияние:**
- `POST /api/location` — устройство или скрипт может флудить тысячами точек, раздувая таблицу `locations`.
- Auth middleware (из-за B-01) под флудом — гарантированный self-DoS.
- `GET /api/app/release/latest` — публичный эндпоинт без ограничений.

**Рекомендация:** Добавить `golang.org/x/time/rate`-based или `github.com/ulule/limiter` middleware. Минимум: per-IP rate limit на публичные эндпоинты, per-user на `/api/location` (POST) и `/api/device/poll`.

---

#### H-03 · `GET /api/location/` — загрузка всей таблицы без пагинации и лимита

**Файлы:**
- `backend/dao/location_dao.go:62–68` — `GetAll()`: `db.Find(&locations)` без WHERE/LIMIT
- `backend/service/location_service.go:172–189` — `GetLocationsWithoutCache()` вызывает `DAO.GetAll()`

**Влияние:** GPS-трекер пишет точку каждые ~5 минут. За год — ~100K записей на пользователя. Запрос `GET /api/location/` в production вернёт весь массив в память приложения, отдаст клиенту несколько МБ JSON, потребляя память и время. При нескольких одновременных запросах — OOM.

**Рекомендация:** Добавить обязательные параметры `from` / `to` (или `limit` / `offset`) в `GET /api/location/`. Без них возвращать 400 Bad Request или ограничивать окном по умолчанию (например, последние 24 часа).

---

#### H-04 · Двойной middleware — избыточная проверка `IsAdmin` в контроллерах

**Файлы:** `backend/controllers/device_controller.go:168, 210, 236, 282, 331, 377, 430` — каждый admin-обработчик явно проверяет `currentUser.IsAdmin` несмотря на то, что маршруты уже защищены `APIKeyAuthMiddleware`, который проверяет `IsAdmin`.

Само по себе это не уязвимость — дублирующая проверка скорее надёжна. Однако **проблема обратная**: маршруты `GET /api/users/:id`, `PUT /api/users/:id` и другие в `userGroup` (за `APIKeyAuthMiddleware`) позволяют любому admin-пользователю изменять данные **любого** пользователя без проверки принадлежности ресурса. IDOR: `PUT /api/users/2` от admin1 меняет пользователя с ID 2.

**Файл:** `backend/controllers/user_controller.go:92–119` — нет проверки, что `currentUser.ID == id` или явного разграничения.

**Влияние:** Любой администратор может переименовать любого пользователя. В текущей архитектуре это принято по дизайну, но явно не задокументировано.

**Рекомендация:** Задокументировать политику: все admin-маршруты намеренно дают доступ ко всем ресурсам. Убрать дублирующие `IsAdmin`-проверки из контроллеров (или оставить как defence-in-depth с комментарием).

---

### 🟡 MEDIUM

---

#### M-01 · `AutoMigrate` конкурирует с Goose-миграциями — риск schema-drift

**Файл:** `backend/config/bootstrap/initializer.go:35–44`

```go
if err := dbConn.AutoMigrate(
    &models.User{}, &models.Location{}, &models.LocationRequest{},
    &models.DeviceCommand{}, &models.DeviceReport{}, &models.Checkpoint{}, &models.Visit{},
); err != nil { … }
```

**Наблюдение:** Проект содержит 8 goose-миграций (в `backend/migrations/*.sql`) **и** вызывает `AutoMigrate` при каждом запуске. GORM AutoMigrate добавляет колонки, но не удаляет их и не управляет версиями. Это создаёт:
- Рассинхронизацию между `goose_db_version` и реальной схемой
- Невозможность rollback через `goose down`
- Потенциальный конфликт: goose-миграция `20260701120000_add_locations_captured_at.sql` добавляет `captured_at`, а AutoMigrate добавит её снова (безвредно из-за `IF NOT EXISTS`, но путает историю)

**Рекомендация:** Выбрать одну систему миграций. Оптимально — оставить goose, убрать AutoMigrate из `InitializeApp`. Добавить `goose up` в entrypoint контейнера.

---

#### M-02 · RabbitMQ consumer — бесконечный requeue poison-сообщений, нет DLQ

**Файл:** `backend/config/messaging/consumer.go:35–42`

```go
if err := handler(msg.Body); err != nil {
    msg.Nack(false, true) // requeue=true — сообщение вернётся в начало очереди
} else {
    msg.Ack(false)
}
```

**Контекст:** `VisitEventProcessor.ProcessEvent` возвращает ошибку при падении `GetCheckpoints()` или при ошибке DB. Повреждённое JSON-сообщение (`json.Unmarshal` → error) будет nacked и переставлено обратно — бесконечно.

**Влияние:** Одно повреждённое сообщение блокирует очередь `location_events` навсегда (Consumer зависнет в loop), что останавливает все визиты для всех пользователей.

**Рекомендация:** Добавить счётчик попыток (через x-death header или отдельную таблицу). После N попыток — `Nack(false, false)` (не requeue) + переместить в DLQ. Объявить DLQ при инициализации.

---

#### M-03 · Отсутствует graceful shutdown — потеря данных при деплое

**Файл:** `backend/main.go:57–62`

```go
defer app.RMQClient.Close()
if err := app.Router.Run(":8080"); err != nil { … }
```

`app.Router.Run` блокирует навсегда. `defer` сработает только при `log.Fatalf` или панике. SIGTERM (docker stop, k8s rollout) завершит процесс немедленно без:
- Ожидания завершения in-flight HTTP-запросов
- Корректного закрытия RabbitMQ-канала (возможна потеря unacked сообщений)
- Закрытия DB-пула

**Рекомендация:** Использовать `net/http.Server` с `Shutdown(ctx)` и обработчиком `os.Signal`:
```go
srv := &http.Server{Addr: ":8080", Handler: router}
go func() { srv.ListenAndServe() }()
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

---

#### M-04 · Hardcoded production IP в исходном коде

**Файл:** `backend/controllers/device_controller.go:255, 302`

```go
apiBase = "http://87.232.65.52:8080"
```

**Влияние:** IP-адрес продакшн-сервера зашит в бинарник. При смене IP-адреса или переносе — нужен полный rebuild. Утечка топологии сети в публичный репозиторий.

**Рекомендация:** Использовать только `os.Getenv("BASE_URL")` с паником или явной ошибкой при отсутствии переменной в production-режиме. Fallback на `localhost` оставить только для `GIN_MODE=debug`.

---

#### M-05 · Geofence-состояния хранятся in-memory — потеря при рестарте

**Файл:** `backend/service/geofence_state.go:15–38`

```go
type geofenceStateStore struct {
    mu    sync.Mutex
    items map[string]*geofencePendingState
}
```

`pendingEnterSince` / `pendingExitSince` живут в памяти процесса. После рестарта сервиса (деплой, OOM) все отложенные переходы входа/выхода из чекпоинтов теряются. Пользователь, находившийся в grace-периоде входа, не получит открытый визит.

**Влияние:** Средний (логические артефакты, не критический сбой). При непрерывном трекинге следующий GPS-тик восстановит состояние через grace-период.

**Рекомендация:** Приемлемо для MVP. При масштабировании (несколько инстансов) — обязательно перенести в Redis или PostgreSQL.

---

#### M-06 · `Visit.Kind` вычисляется не из БД — непоследовательный API

**Файл:** `backend/models/visit.go:13`

```go
Kind string `gorm:"-" json:"kind,omitempty"`
```

`Kind` не персистируется. Значение `"checkpoint"` или `"outside"` заполняется только в `TravelSegmentService.GetOutsideSegments`. При обычном `GET /api/visits/` без `include_outside=true` поле `kind` отсутствует в ответе. Клиент получает непоследовательный контракт.

**Рекомендация:** Либо добавить `kind` как вычисляемое поле в DAO-слое (`CASE WHEN checkpoint_id = 0 THEN 'outside' ELSE 'checkpoint' END`), либо всегда заполнять его в `GetVisitsByFilters` перед возвратом.

---

#### M-07 · Отсутствует индекс для `GetActiveVisit`

**Файл:** `backend/migrations/20250611090000_create_visits_table.sql` — нет индексов  
**Запрос (DAO):** `WHERE user_id = ? AND checkpoint_id = ? AND end_at IS NULL`

На каждый входящий GPS-тик (через `VisitEventProcessor.ProcessEvent`) для каждого чекпоинта выполняется `GetActiveVisit`. При 10 чекпоинтах и 1 пользователе — 10 запросов на каждое GPS-событие, все — seq scan по `visits`.

**Рекомендация:** Добавить миграцию:
```sql
CREATE INDEX idx_visits_user_checkpoint_active
    ON visits (user_id, checkpoint_id) WHERE end_at IS NULL;
```

---

### 🔵 LOW

---

#### L-01 · Orphaned migration: `location_events` таблица создана, но не используется

**Файл:** `backend/migrations/20250611090001_create_location_events_table.sql`

Таблица `location_events` создаётся (с полями `id`, `user_id`, `checkpoint_id`, `processed`, etc.), но `models.LocationEvent` — это pure Go struct без GORM-маппинга, используется только как сообщение в RabbitMQ. Ни один код не читает/пишет в `location_events` через ORM.

**Рекомендация:** Удалить миграцию или добавить persistence в неё — зависит от будущих требований к аудит-логу событий.

---

#### L-02 · `PostCheckpoint` не валидирует входные данные

**Файл:** `backend/controllers/checkpoint_controller.go:39–56`

```go
var req struct {
    Name      string  `json:"name"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Radius    float64 `json:"radius"`
}
```

Нет `binding:"required"`, нет проверки диапазона (`Latitude` вне `[-90, 90]`, `Radius <= 0`). Можно создать чекпоинт с `Radius=0` — геозона никогда не сработает.

**Рекомендация:** Добавить валидацию в контроллере или сервисе: `Name != ""`, `-90 <= Lat <= 90`, `-180 <= Lon <= 180`, `Radius > 0`.

---

#### L-03 · Нет `DELETE /api/checkpoint/:id`

**Файл:** `backend/router/routes.go:90–96` — только GET, POST, PUT для чекпоинтов.

После создания чекпоинт нельзя удалить через API. При наличии каскадного `ON DELETE CASCADE` в миграции (`fk_checkpoint`) — инфраструктура для удаления есть, эндпоинт отсутствует.

**Рекомендация:** Добавить `DELETE /api/checkpoint/:id` с удалением визитов по каскаду (уже настроен в FK).

---

#### L-04 · `haversineDistance` дублируется в двух сервисах

**Файлы:** `backend/service/checkpoint_service.go:111–124`, `backend/service/location_service.go:485–499`

Идентичная реализация в двух местах.

**Рекомендация:** Вынести в `backend/service/geo.go` (или `backend/internal/geo/`).

---

#### L-05 · `GetLocations` не фильтрует по `user_id` для admin-запросов

**Файл:** `backend/controllers/location_controller.go:257–288`

`GET /api/location/?from=…&to=…` возвращает точки **всех** пользователей в указанном диапазоне. Это может быть намеренным (admin видит всё), но нет UI-подсказки или документации. Если admin ожидает видеть только «своего» пользователя — это логическая ошибка.

**Рекомендация:** Добавить опциональный query-параметр `user_id` для фильтрации; задокументировать текущее поведение в комментарии.

---

#### L-06 · `PostDeviceReport` принимает `map[string]interface{}` без схемы

**Файл:** `backend/controllers/device_controller.go:103–106`

```go
var body map[string]interface{}
if err := ctx.ShouldBindJSON(&body); err != nil { … }
```

Любые данные от устройства принимаются и сохраняются в JSONB. Нет ограничения размера тела запроса. Устройство может прислать 100 МБ JSON и он будет сохранён в `device_reports.report`.

**Рекомендация:** Добавить `MaxBytesReader` на уровне middleware (или в Gin: `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)` — 1 МБ). Опционально — базовую схему валидации.

---

### ℹ️ INFO

---

#### I-01 · Timezone жёстко задана как `Europe/Minsk`

**Файлы:** `backend/service/location_service.go:21`, `backend/service/visit_service.go:177`

Часовой пояс Минска хардкожен в двух сервисах. При смене региона или развёртывании в другой зоне — нужна правка кода.

**Рекомендация:** Вынести в `env: APP_TIMEZONE=Europe/Minsk` и загружать через `os.Getenv`.

---

#### I-02 · `gin.Default()` использует stdout-логгер без структурирования

**Файл:** `backend/router/routes.go:24`

`gin.Default()` включает стандартный текстовый access-логгер. Нет корреляционных ID, нет structured JSON.

**Рекомендация:** Перейти на `gin.New()` + кастомный middleware с `slog`/`zap` для structured logging.

---

#### I-03 · `DB` в `App` объявлена как `interface{}`, теряет типизацию

**Файл:** `backend/config/bootstrap/initializer.go:25`

```go
type App struct {
    Router    *gin.Engine
    DB        interface{} // <-- теряем *gorm.DB
    RMQClient *messaging.RabbitMQClient
}
```

`main.go` не использует `app.DB`, но если потребуется — придётся делать type assertion. Лучше хранить `*gorm.DB` напрямую.

---

#### I-04 · Нет `go.sum` / зависимости не проверены в CI

Не наблюдалось CI-конфигурации с `go mod verify`. Зависимости (`amqp091-go`, `go-qrcode`, etc.) не закреплены хеш-суммой в процессе сборки.

---

#### I-05 · Нет OpenAPI-спецификации

Нет `swagger.yaml` / `openapi.json`. Контракт API существует только в коде. Фронтенд и мобильная команда не имеют формального контракта.

---

## 3. Приоритизированный план устранения

| Приоритет | ID | Заголовок | Трудоёмкость |
|---|---|---|---|
| 🔴 1 | B-01 | O(N) bcrypt-скан: добавить `api_key_prefix` lookup | M (миграция + сервис) |
| 🔴 2 | B-02 | QR-коды: убрать из public static, генерить in-memory | S |
| 🟠 3 | H-01 | Убрать plaintext ключ из seed-лога | XS |
| 🟠 4 | H-02 | Rate limiting middleware (per-IP / per-user) | M |
| 🟠 5 | H-03 | Пагинация / обязательный диапазон для `GET /api/location/` | S |
| 🟡 6 | M-01 | Убрать `AutoMigrate`, перейти на чистый goose | S |
| 🟡 7 | M-02 | DLQ + retry-лимит для RabbitMQ consumer | M |
| 🟡 8 | M-03 | Graceful shutdown (`http.Server.Shutdown`) | S |
| 🟡 9 | M-04 | Убрать hardcoded IP, panic при пустом `BASE_URL` в prod | XS |
| 🟡 10 | M-07 | Индекс `visits(user_id, checkpoint_id) WHERE end_at IS NULL` | XS (миграция) |
| 🔵 11 | L-02 | Валидация Checkpoint (name, lat, lon, radius) | XS |
| 🔵 12 | L-06 | Лимит тела `POST /device/report` | XS |
| 🔵 13 | L-03 | `DELETE /api/checkpoint/:id` | XS |
| ℹ️ 14 | M-06 | `Visit.Kind` в DB или всегда вычислять | S |
| ℹ️ 15 | I-02 | Structured logging с correlation ID | M |

---

## 4. Acceptance Checks

После устранения каждого найденного дефекта предлагаются следующие проверки:

### B-01 (Auth performance)
```bash
# Benchmark: время ответа не должно расти с числом пользователей
go test -bench=BenchmarkAuthenticate -benchtime=10s ./service/...
# Убедиться: 1 SELECT + 1 bcrypt.CompareHashAndPassword
```

### B-02 (QR security)
```bash
# Запрос без auth должен вернуть 401 или 404, НЕ PNG
curl -o /dev/null -w "%{http_code}" http://localhost:8080/static/qrcode/1.png
# Ожидаемый результат: 401 или 404
```

### H-01 (Log secrets)
```bash
grep -n "Plain API" backend/seed/seed.go
# Ожидаемый результат: нет совпадений
```

### H-02 (Rate limiting)
```bash
# 100 запросов подряд должны вернуть 429 после порога
ab -n 200 -c 10 -H "X-API-Key: test" http://localhost:8080/api/location/single
```

### M-01 (AutoMigrate removal)
```bash
grep -n "AutoMigrate" backend/config/bootstrap/initializer.go
# Ожидаемый результат: нет совпадений
cd backend && goose -dir migrations postgres "$DSN" up
```

### M-02 (DLQ)
```bash
# Отправить невалидное JSON-сообщение в очередь
# Убедиться, что оно не повторяется бесконечно:
# Проверить наталкивание в DLQ-очередь через RabbitMQ Management UI
```

### M-03 (Graceful shutdown)
```bash
docker stop locator_backend  # отправляет SIGTERM
# Проверить лог: должно быть "Server stopped gracefully"
# В-flight запросы должны завершиться, не получив EOF
```

### M-07 (Index)
```sql
-- Проверить наличие индекса
SELECT indexname FROM pg_indexes
WHERE tablename = 'visits' AND indexname LIKE '%active%';
```

### Регрессионные тесты
```bash
cd backend && go test ./... -count=1 -timeout 120s
# Все существующие интеграционные тесты должны пройти
```

---

## 5. Допущения и вне области аудита

### Допущения
- Анализ проводился в режиме read-only, без запуска production-окружения или работающей БД.
- Покрытие тестами оценивалось по наличию `_test.go` файлов; инструментальное измерение (`go test -cover`) не выполнялось.
- Аутентификация намеренно простая (X-API-Key + bcrypt) — нет JWT/OAuth2, это принятое проектное решение.
- RabbitMQ-топология (exchange, routing, bindings) не проверялась на запущенном брокере.
- Frontend, Android-клиент и DevOps (docker-compose, deploy.sh) вне области данного отчёта.

### Вне области
- Тестирование на проникновение (penetration testing) — статический анализ только
- Анализ производительности OSRM-интеграции (`location_controller.go:GetMatchedRoute`)
- Оценка надёжности `AppReleaseController` (manifest.json / APK distribution)
- Анализ фронтенда (React/Vite)
- Анализ инфраструктуры (Nginx, TLS, firewall)

---

**Итог:** 2 BLOCKER (auth O(N), QR public), 4 HIGH, 7 MEDIUM, 6 LOW, 5 INFO. Критический путь для production-readiness: B-01 → B-02 → H-01 → H-02 → M-03 (5 задач, ~3–5 дней работы).

The user wants me to synthesize the full audit report now based on all the evidence I've gathered. Let me write the complete markdown report in Russian.


---
*Источник: [backend-developer](0e932dea-cdec-4b5a-9743-fc29dee0479e); кратко подтверждено [повторным BE](fd624294-7249-42c7-84b6-f8c6c16c893a).*
