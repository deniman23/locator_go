# Подсказка для агентов: голый телефон → связь с сервером → OTA

Чеклист для Cursor/агентов при онбординге нового Android-устройства в Locator.
Два репозитория: **locator_go** (сервер, API, админка) и **lctr_app** (Android).

---

## Контекст (прод, обновлять при смене сервера)

| Параметр | Значение |
|----------|----------|
| Сервер API | `http://87.232.65.52:8080` |
| Админка (веб) | `http://87.232.65.52:3000` |
| SSH / код на сервере | `/root/locator_go`, `/root/lctr_app` |
| Package Android | `com.example.lctr_app` |
| Версия APK (смотреть в репо) | `lctr_app/app/version.properties` |
| Админ API key (из `.env`) | `DEFAULT_ADMIN_API_KEY` (часто `change_me`) |
| Poll интервал на телефоне | ~15 с (`LOCATOR_POLL_INTERVAL_MS`) |
| GPS интервал | ~300 с (`location_interval_seconds`) |

**Важно:** «рут» в этом проекте — это **Device Owner** (корпоративный режим через ADB), а не Magisk/su. Без Device Owner нет тихой OTA и автоправ на фон.

---

## Фаза 0. Подготовка на ПК агента

```bash
adb devices -l          # статус device, не unauthorized
# при unauthorized — подтвердить RSA на экране телефона
```

Скрипт сбора диагностики с ПК (телефон по USB):

```bash
/root/locator_go/scripts/diagnose_phone.sh [SERIAL]
```

На сервере — health бэкенда:

```bash
curl -s http://127.0.0.1:8080/healthz
docker compose -f /root/locator_go/docker-compose.yml ps
```

---

## Фаза 1. Голый телефон → прошивка / сброс → Device Owner

### 1.1 Сброс (обязательно для Device Owner)

1. **Заводской сброс** (Settings → сброс) или `adb reboot recovery` + wipe.
2. Пройти мастер настройки **без** Google-аккаунта (или минимально).
3. Включить **Отладку по USB**.
4. **Не** ставить сторонние лаунчеры до назначения Device Owner.

Device Owner можно выдать **только** на «чистом» устройстве без других профилей/аккаунтов.

### 1.2 Установка APK

**Release (прод, OTA):**

```bash
# с сервера или после CI
adb install -r /root/locator_go/backend/static/releases/locator-latest.apk
# или конкретная версия:
adb install -r /root/locator_go/backend/static/releases/locator-1.0.12-13.apk
```

**Debug** — только для разработки; поверх release без снятия DO не обновить.

Проверка версии:

```bash
adb shell dumpsys package com.example.lctr_app | grep -E 'versionName=|versionCode='
```

### 1.3 Device Owner (не root/Magisk)

```bash
adb shell dpm set-device-owner com.example.lctr_app/.corporate.CorporateDeviceAdminReceiver
```

Ожидается: `Success: Device owner set to ...`

Проверка:

```bash
adb shell dpm list-owners
adb shell dumpsys device_policy | grep -i "device owner" | head -5
```

В приложении на экране настройки должно быть **Device Owner: да**.

Снять DO (только debug / перед сменой подписи):

```bash
adb shell am start -n com.example.lctr_app/.MainActivity --ez clear_device_owner true
# только DEBUG-сборка; иначе снова factory reset
```

### 1.4 Разрешения и обязательные отключения (схема прошивки)

При Device Owner приложение само применяет политики (`DeviceOwnerManager.applyMandatoryProvisioningPolicies` / `suppressTrackingNotifications`):

| Политика | Значение | Зачем |
|----------|----------|-------|
| Геолокация (fine/coarse/background) | **GRANTED** (один раз) | Трекинг без запросов пользователю |
| `POST_NOTIFICATIONS` | **DENIED** | Без служебных push и спама от приложения |
| Перевыдача geo на каждом wake | **запрещена** | Иначе Samsung/Android шлёт «В вашей организации … разрешено использование геолокации» каждые N минут |
| Батарея | whitelist (Doze) | Фон не убивается |
| Лаунчер | скрыто | Маскировка под системное приложение |
| Удаление | заблокировано | Только через ADB / factory reset |

**Важно:** полностью убрать системный индикатор foreground location service **нельзя** — это требование Android. Но уведомление «В вашей организации …» должно исчезнуть после OTA **>= 1.0.30** (не перевыдавать разрешения).

Проверка на телефоне (ADB):

```bash
adb shell dumpsys device_policy | grep -A2 POST_NOTIFICATIONS
adb shell dumpsys package com.example.lctr_app | grep POST_NOTIFICATIONS
```

В health-отчёте: `corporate.app_notifications_suppressed = true`.

Если на конкретной прошивке уведомления «приложение использует геолокацию» всё равно всплывают:
`Настройки → Конфиденциальность → Местоположение → Уведомления об использовании` → **Выкл** (Samsung).
Либо дождаться OTA с `suppressTrackingNotifications` (Device Owner применяет автоматически).

---

## Фаза 2. Привязка к серверу (QR / config)

### 2.1 Создать пользователя (если новый)

Админка → **Пользователи** → создать, или API:

```bash
API_KEY=change_me
curl -s -X POST http://87.232.65.52:8080/api/users/ \
  -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
  -d '{"name":"phone-01","is_admin":false}'
```

### 2.2 QR-код

Админка → пользователь → **QR-код** (или **Перегенерировать QR** после смены IP).

В QR JSON:

```json
{"user_id": N, "api_key": "...", "api_base_url": "http://87.232.65.52:8080"}
```

**Частая ошибка:** старый QR с `localhost:8080` или без `api_base_url` → `post_failed`, очередь offline.

Принудительно на устройство (если уже онлайн):

```bash
curl -s -X POST "http://87.232.65.52:8080/api/admin/users/1/regenerate-qr" \
  -H "X-API-Key: change_me" -H "Content-Type: application/json" \
  -d '{"api_key":"change_me","push_to_device":true}'
```

Команда `config_update` уйдёт в poll.

### 2.3 На телефоне

1. Открыть **Locator** → сканировать QR.
2. Запустить службу (кнопка в приложении или автостарт при DO).
3. Убедиться, что в отчёте `api_base_url` = прод-сервер, не localhost.

---

## Фаза 3. Точки API — что проверять (чеклист)

Все запросы с телефона: заголовок **`X-API-Key: <ключ пользователя>`**.

| # | Метод | Путь | Кто вызывает | Ожидание |
|---|--------|------|--------------|----------|
| 1 | GET | `/healthz` | агент | `{"status":"ok"}` |
| 2 | GET | `/api/users/me` | приложение | 200, `id`, `name` |
| 3 | POST | `/api/location` | LocationService | 200, запись в БД |
| 4 | GET | `/api/device/poll` | каждые ~15 с | 204 или JSON с `command` |
| 5 | POST | `/api/device/report` | health ~20 мин | 200 |
| 6 | POST | `/api/device/command/ack` | после команд | 200 |
| 7 | GET | `/api/users/:id/health` | админ | `healthy`, `issues`, `report` |
| 8 | POST | `/api/admin/users/:id/commands` | админ | `health_check`, `config_update` |
| 9 | POST | `/api/admin/releases/publish-update/:user_id` | админ | OTA `app_update` |
| 10 | GET | `/api/app/release/latest` | приложение | manifest OTA |

Проверка auth с ПК:

```bash
curl -s -H "X-API-Key: change_me" http://87.232.65.52:8080/api/users/me | jq .
```

Poll (эмуляция телефона):

```bash
curl -s -D- -H "X-API-Key: USER_KEY" http://87.232.65.52:8080/api/device/poll
# 204 = нет команд; 200 + command = есть команда
```

---

## Фаза 4. Тестовая отправка координат и сверка на сервере

### 4.1 POST с ПК (как телефон)

```bash
API_KEY=change_me   # или ключ пользователя телефона
TS=$(python3 -c "import time; print(int(time.time()*1000))")

curl -s -X POST http://87.232.65.52:8080/api/location \
  -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
  -d "{
    \"latitude\": 53.92684,
    \"longitude\": 27.69514,
    \"source\": \"manual_test\",
    \"timestamp\": $TS
  }" | jq .
```

Ожидание: JSON с `id`, `user_id`, `latitude`, `longitude`, `captured_at`.

### 4.2 Сверка в PostgreSQL

```bash
docker exec locator-db psql -U locator_user -d locator_db -c "
  SELECT id, user_id, latitude, longitude, captured_at, created_at, source
  FROM locations WHERE user_id = 1
  ORDER BY created_at DESC LIMIT 5;"
```

Координаты теста должны совпасть (с точностью до округления).

### 4.3 Сверка на карте

Админка → карта → период «сегодня» → пользователь → **Применить интервал**.
Статус: `загр. N` точек, GPS онлайн в **Пользователи**.

### 4.4 Визиты в чекпоинте

Визиты создаёт **consumer RabbitMQ** (`location_events`). Без consumer очередь копится, визитов нет.

```bash
docker exec locator-rabbitmq rabbitmqctl list_queues name messages consumers
# location_events: consumers >= 1, messages ~ 0

docker exec locator-db psql -U locator_user -d locator_db -c "
  SELECT id, checkpoint_id, start_at, end_at FROM visits
  WHERE user_id = 1 ORDER BY start_at DESC LIMIT 5;"
```

Вход в зону чекпоинта: **~30 с** устойчивого GPS в радиусе (`GEOFENCE_ENTER_GRACE_SECONDS`).

Полный скрипт API-тестов:

```bash
BASE_URL=http://87.232.65.52:8080 API_KEY=change_me USER_ID=1 \
  /root/locator_go/scripts/test_ota_and_captured_at.sh
```

---

## Фаза 5. Плановые проверки (poll, health, GPS)

### 5.1 Админка → Пользователи

| Кнопка | Команда | Что ждать |
|--------|---------|-----------|
| **GPS** | `location_request` | свежие координаты на карте |
| **Диагностика** | `health_check` | отчёт в колонке «Устройство» |
| **Обновление** | `app_update` | OTA (см. фаза 6) |

### 5.2 API health

```bash
curl -s -H "X-API-Key: change_me" \
  http://87.232.65.52:8080/api/users/1/health | jq .
```

Здорово, если `healthy: true`, `issues: []`.

Типичные `issues`:

| issue | Причина | Действие |
|-------|---------|----------|
| `post_failed` | последний POST не 200 | смотреть `last_post_error` (часто `localhost`) |
| `offline_queue_large` | очередь ≥ 20 | `config_update` с правильным `api_base_url` |
| `background_stopped` | LocationService не работает | DO + старт службы |
| `location_permission_not_always` | нет фона GPS | выдать «Всегда» |
| `app_update_failed` | OTA ошибка | logcat, manifest, sha256 |

### 5.3 Плановый poll

Телефон сам: `GET /api/device/poll` каждые ~15 с.
Проверка в отчёте: `poll.last_poll_at` свежий, `last_poll_status` 204 или 200.

### 5.4 Плановый GPS

Каждые ~300 с новая точка `source: periodic` (если не на паузе).
В health: `location.last_post_at` не старше ~2× интервала.

---

## Фаза 6. OTA-обновления

### 6.1 Manifest на сервере

```bash
curl -s http://87.232.65.52:8080/static/releases/manifest.json | jq .
curl -s http://87.232.65.52:8080/api/app/release/latest | jq .
```

Поля: `version_code`, `version_name`, `sha256`, `url`.

### 6.2 Отправить обновление на устройство

```bash
curl -s -X POST http://87.232.65.52:8080/api/admin/releases/publish-update/1 \
  -H "X-API-Key: change_me" | jq .
```

Или кнопка **Обновление** в админке.

### 6.3 Условия успешной OTA

- `version_code` в manifest **>** на устройстве
- APK **release** подпись совпадает с установленной
- **Device Owner** для тихой установки
- Телефон online (poll получает `app_update`)
- После установки — `command/ack` success (см. `device_commands` в БД)

### 6.4 Проверка версии после OTA

```bash
adb shell dumpsys package com.example.lctr_app | grep -E 'versionName=|versionCode='
```

Сверить с `manifest.json`.

Подробный runbook релиза: **`lctr_app/docs/PREPARE-UPDATE.md`**.

Сборка на сервере вручную:

```bash
/root/locator_go/scripts/build_android_release.sh
```

---

## Фаза 7. Диагностика при сбоях

### Логи

```bash
# backend
docker logs locator-backend --tail 100

# визиты / RabbitMQ
docker logs locator-backend 2>&1 | grep -i ProcessEvent | tail -20
docker exec locator-rabbitmq rabbitmqctl list_queues

# телефон
adb logcat -d | grep -iE 'LocationService|AppUpdate|LocatorHttp|DeviceOwner' | tail -50
```

### Принудительный старт службы

```bash
adb shell am start-foreground-service -n com.example.lctr_app/.LocationService
# или из diagnose_phone.sh
```

### Диск сервера полный

```bash
/root/locator_go/scripts/cleanup-disk.sh
df -h /
```

### Бэкап БД

```bash
/root/locator_go/scripts/backup-db.sh
```

---

## Порядок работы агента (краткий)

1. `healthz` + `docker compose ps` на сервере.
2. Сброс телефона → `adb install` release APK → `dpm set-device-owner`.
3. Создать пользователя → QR → скан → проверить `api/users/me`.
4. POST тестовой точки → сверить `locations` в БД и на карте.
5. Подождать periodic GPS / нажать **GPS** в админке.
6. **Диагностика** → `healthy`, нет `post_failed` / `offline_queue_large`.
7. Стоять в чекпоинте 1–2 интервала GPS → проверить `visits` и RabbitMQ consumer.
8. `publish-update` → версия на телефоне выросла.
9. Весь вывод `diagnose_phone.sh` сохранить в тикет при эскалации.

---

## Связанные файлы

| Файл | Назначение |
|------|------------|
| `scripts/diagnose_phone.sh` | дамп состояния телефона по USB |
| `scripts/test_ota_and_captured_at.sh` | API: location, backfill, OTA |
| `scripts/build_android_release.sh` | сборка APK на сервере |
| `scripts/pull-and-deploy.sh` | автодеплой locator_go |
| `lctr_app/docs/PREPARE-UPDATE.md` | релиз и CI Android |
| `.env` / `.env.example` | `BASE_URL`, `DEFAULT_ADMIN_API_KEY` |

---

*Обновляйте IP/URL в таблице «Контекст» при миграции сервера.*
