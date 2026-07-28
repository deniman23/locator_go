# DevOps Health-Check Audit — Locator (locator_go)

**Дата:** 2026-07-28  
**Аудитор:** devops-engineer (subagent)  
**Статус сервисов на момент аудита:** все 4 контейнера работают (healthy / up)

---

## 1. Scope & Method

### Охват

| Область | Что проверялось |
|---|---|
| CI/CD | `.github/workflows/main.yml`, `.github/workflows/tests.yml` |
| Контейнеры | `docker-compose.yml`, `backend/Dockerfile`, `frontend/Dockerfile`, `docker inspect` |
| Деплой | `deploy.sh`, `scripts/pull-and-deploy.sh`, `scripts/docker-build.sh`, `scripts/ci/deploy-locator-go.sh`, crontab |
| Секреты / конфиги | `.env.example`, `.gitignore`, ENV-переменные контейнеров (без вывода значений) |
| Наблюдаемость | `GET /healthz`, логи в `backend/logs/`, `/var/log/locator-*.log`, интеграция с Sentry/Datadog |
| Бэкапы | `scripts/backup-db.sh`, каталог `backups/`, расписание cron |
| Надёжность | resource limits, healthcheck-параметры, restart policy, откат миграций |
| Диск | `df -h`, `docker system df` |

### Метод

Статический анализ файлов (`Read`, `Grep`, `Glob`) + живые команды только для чтения (`docker inspect`, `docker ps`, `crontab`, `df`, `curl /healthz`, `ss`). CodeGraph использован для маршрутов `/healthz`. Секретные значения не выводились в отчёт.

---

## 2. Findings

### 🔴 BLOCKER

---

#### B-1 — Хардкод DB_PASSWORD в образе Docker

**Файл:** `backend/Dockerfile`, строка 45

```
ENV \
    ...
    DB_PASSWORD=***REDACTED*** \
    DB_SSLMODE=disable
```

**Доказательство:** Значение подставляется на этапе сборки (`docker build`) и закапывается в каждый слой образа. `docker inspect locator-backend` и `docker history locator-go-backend` раскроют пароль любому, у кого есть доступ к Docker socket или к реестру образов. Пароль также будет в git-истории при любом `docker build --no-cache` в CI с публичным runner-ом.

**Влияние:** Полная компрометация базы данных при утечке образа или артефактов CI/CD.

**Рекомендация:**
1. Немедленно сменить пароль БД на сервере.
2. Убрать `ENV DB_PASSWORD=***REDACTED*** из Dockerfile — переменная уже приходит из `.env` через `docker-compose.yml`. В Dockerfile достаточно документального `ARG`/`ENV` без значения:
   ```dockerfile
   ENV DB_PASSWORD=***REDACTED***
   ```
3. Добавить в `scripts/docker-build.sh` или в CI проверку: `git secrets --scan` или `truffleHog`.

---

### 🟠 HIGH

---

#### H-1 — RabbitMQ открыт на всех интерфейсах с дефолтными учётными данными

**Файлы:** `docker-compose.yml` строка 120; `.env.example` строки 18–20

```yaml
rabbitmq:
  ports:
    - "5672:5672"   # 0.0.0.0:5672 — виден извне
```

```
RABBITMQ_USER=guest
RABBITMQ_PASS=guest
```

**Доказательство:** `ss -tlnp` показывает `0.0.0.0:5672` с процессом `docker-proxy`. RabbitMQ 3.x ограничивает пользователя `guest` только `localhost` внутри самого сервиса — но не на уровне сетевого биндинга хоста. Если в `.env` не переопределены `RABBITMQ_USER/PASS`, учётные данные остаются `guest/guest`.

**Влияние:** Любой, кто имеет доступ к порту 5672 сервера, может подключиться к брокеру, читать/публиковать сообщения, управлять топологией.

**Рекомендация:**
1. Изменить биндинг порта на `127.0.0.1:5672:5672`.
2. Обязательно установить `RABBITMQ_USER` и `RABBITMQ_PASS` в `.env` (не `guest`).
3. В `docker-compose.yml` добавить:
   ```yaml
   environment:
     RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER}
     RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASS}
   ```

---

#### H-2 — Отсутствует TLS/HTTPS на всём стеке

**Файлы:** `frontend/nginx.conf`, `docker-compose.yml`

nginx слушает только `port 80`. Backend — `8080`. Нет конфигурации SSL/TLS, нет проброса через reverse-proxy с сертификатом.

**Доказательство:** `nginx.conf` содержит только блок `listen 80`. `BASE_URL=http://87.232.65.52:8080` — явный HTTP.

**Влияние:** Все данные (координаты, API-ключи в заголовках `X-API-Key`, пароли входа) передаются открытым текстом. QR-коды, APK — по незащищённому каналу.

**Рекомендация:** Добавить перед nginx обратный прокси (nginx/Caddy) с Let's Encrypt. Caddy — наиболее простой вариант для single-server MVP:
```yaml
caddy:
  image: caddy:alpine
  ports: ["80:80", "443:443"]
  volumes:
    - caddy_data:/data
    - ./Caddyfile:/etc/caddy/Caddyfile
```

---

#### H-3 — Нет resource limits ни на одном контейнере

**Файл:** `docker-compose.yml` (весь файл)

`docker inspect locator-backend` → `Memory: 0`, `CpuShares: 0`. Ни один сервис не имеет `mem_limit`, `cpus`, `memswap_limit`.

**Доказательство:** `docker inspect` → `HostConfig.Memory == 0` для всех контейнеров.

**Влияние:** Один «взбесившийся» контейнер (утечка памяти в Go-сервере, бесконечный цикл) может исчерпать RAM/CPU хоста и уронить остальные сервисы. Диск и без того занят на 87%.

**Рекомендация:**
```yaml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M
  db:
    deploy:
      resources:
        limits:
          memory: 256M
  rabbitmq:
    deploy:
      resources:
        limits:
          memory: 256M
  frontend:
    deploy:
      resources:
        limits:
          memory: 64M
```

---

#### H-4 — Диск заполнен на 87% (17 ГБ из 20 ГБ)

**Доказательство:** `df -h /` → `17G / 20G (87%)`. `docker system df` → build cache 3.2 ГБ (полностью reclaimable).

**Влияние:** При следующей `docker build` (cron каждые 2 мин при изменениях) и заполнении оставшихся 2.6 ГБ — все сервисы упадут. Бэкап на этот же диск (~1.2 МБ/день, за 14 дней — ~16 МБ) сейчас не критичен, но при росте данных создаст проблему.

**Рекомендация:**
1. Немедленно: `docker buildx prune -f --keep-storage 1gb` (освободит ~2 ГБ).
2. Уменьшить `BUILDX_KEEP_STORAGE` в `.env` с 3gb до 1.5gb.
3. Добавить мониторинг диска: alert при `df > 80%` (e.g., простой cron-скрипт → Telegram/email).
4. Рассмотреть расширение тома или перенос бэкапов на внешнее хранилище.

---

#### H-5 — Нет сканирования уязвимостей и линтинга в CI

**Файл:** `.github/workflows/tests.yml`

В pipeline `tests.yml` — только unit, integration, e2e тесты. Нет:
- `golangci-lint` (заявлен в AGENTS.md как обязательный)
- `govulncheck` или `trivy`
- `npm audit` для frontend

**Влияние:** Уязвимые зависимости и code-smells попадают в production незамеченными.

**Рекомендация:**
```yaml
- name: golangci-lint
  uses: golangci/golangci-lint-action@v6
  with:
    working-directory: backend

- name: govulncheck
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
  working-directory: backend

- name: npm audit
  run: npm audit --audit-level=high
  working-directory: frontend
```

---

### 🟡 MEDIUM

---

#### M-1 — Бэкапы только локально, нет offsite копии

**Файл:** `scripts/backup-db.sh`

**Доказательство:** `grep remote/s3/rclone backup-db.sh` — пусто. Бэкапы в `backups/` на том же диске, что и БД.

**Влияние:** При отказе физического диска или сервера данные и бэкапы теряются одновременно. RTO/RPO = неопределённо.

**Рекомендация:** Добавить в `backup-db.sh` после gzip:
```bash
# offsite copy (пример с rclone → S3/Backblaze)
rclone copy "$OUT" remote:locator-backups/ --quiet || true
```

---

#### M-2 — Cron-деплой запускается от root каждые 2 минуты без блокировки

**Файл:** `/etc/cron.d/locator` (записи cron)

```
*/2 * * * * root /root/locator_go/scripts/pull-and-deploy.sh >> /var/log/locator-deploy.log 2>&1
```

**Проблемы:**
- Нет lock-файла — два cron-запуска могут наложиться при медленной сборке (>2 мин), что приведёт к гонке `docker compose up` и двойной пересборке.
- Полные права root без необходимости.

**Рекомендация:**
```bash
# В начало pull-and-deploy.sh
exec 200>/var/lock/locator-deploy.lock
flock -n 200 || { echo "already running"; exit 0; }
```

---

#### M-3 — Backend healthcheck не имеет `start_period`; frontend без healthcheck

**Файл:** `docker-compose.yml`

Backend:
```yaml
healthcheck:
  interval: 60s
  timeout: 5s
  retries: 3
  # start_period: отсутствует!
```

Frontend: healthcheck не определён вообще.

**Влияние:** При перезапуске backend Docker начинает считать retries с первой секунды, пока goose-миграции ещё выполняются (~3–10 с). После 3 неудачных попыток за 60×3=180 с контейнер помечается `unhealthy`, но `restart: unless-stopped` не реагирует на это — только `restart: on-failure` и политики orchestration.

**Рекомендация:**
```yaml
backend:
  healthcheck:
    start_period: 30s   # дать время миграциям
    interval: 30s       # чаще для быстрого обнаружения
    timeout: 5s
    retries: 3

frontend:
  healthcheck:
    test: ["CMD-SHELL", "wget -q -O /dev/null http://localhost:80/ || exit 1"]
    interval: 30s
    timeout: 5s
    retries: 3
    start_period: 10s
```

---

#### M-4 — `healthz` не проверяет downstream-зависимости

**Файл:** `backend/router/routes.go` строка 26–28

```go
router.GET("/healthz", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
```

**Влияние:** Контейнер сообщает `healthy` даже при отвалившейся БД или RabbitMQ. Docker healthcheck и внешние мониторы получают ложный `ok`.

**Рекомендация:** Добавить deep health-check:
```go
// /healthz — shallow (для load-balancer)
// /readyz  — deep: ping DB + RabbitMQ
router.GET("/readyz", healthController.DeepCheck)
```

---

#### M-5 — Goose-миграции запускаются автоматически в CMD каждого старта

**Файл:** `backend/Dockerfile` строка 48–55

```bash
CMD ["bash", "-c", "... goose ... up; exec /app/locator"]
```

**Влияние:** При деплое нескольких реплик (или при cron-гонке) два экземпляра могут запустить `goose up` одновременно. Goose использует advisory lock, но при сетевом сбое между lock-acquire и migration-execute можно получить частичное применение. Нет механизма `goose down` / rollback.

**Рекомендация:**
1. Вынести миграции в отдельный `initContainers`-шаг (init-контейнер или отдельный Compose-сервис с `restart: on-failure`).
2. Добавить документированный runbook для `goose down N`.

---

#### M-6 — IP-адрес сервера захардкожен в скриптах и workflow

**Файлы:** `scripts/publish_apk_release.sh` строка 13; `.github/workflows/main.yml` строка 16

```bash
BASE_URL="${BASE_URL:-http://87.232.65.52:8080}"  # publish_apk_release.sh
echo "Сервер 87.232.65.52..."                     # main.yml
```

**Влияние:** При смене IP сервера или переходе на домен — нужно вручную искать и менять в нескольких файлах.

**Рекомендация:** Вынести в `BASE_URL` из `.env`, использовать `${BASE_URL}` без fallback на конкретный IP.

---

#### M-7 — `StrictHostKeyChecking=no` в SSH-скрипте деплоя

**Файл:** `scripts/ci/deploy-locator-go.sh` строки 16–22

```bash
-o StrictHostKeyChecking=no
-o UserKnownHostsFile=/dev/null
```

**Влияние:** MITM-атака возможна при компрометации DNS или сети — деплой пойдёт на подставной сервер.

**Рекомендация:** Использовать `known_hosts` с заранее записанным fingerprint сервера вместо `StrictHostKeyChecking=no`.

---

#### M-8 — Нет мониторинга / alerting (Sentry, Datadog отсутствуют)

**Доказательство:** `grep -r "SENTRY\|DATADOG"` в `backend/` и `frontend/src/` — пусто. Нет ни SDK, ни DSN в `.env.example`.

**Влияние:** Ошибки в production неизвестны до жалобы пользователя. MTTR неизмерим.

**Рекомендация (минимум):**
1. Sentry Go SDK в backend: `github.com/getsentry/sentry-go`.
2. `SENTRY_DSN` в `.env.example` (пустое значение).
3. Простейший alert: cron-скрипт, проверяющий `curl /healthz` и отправляющий уведомление в Telegram/email при сбое.

---

#### M-9 — `DB_SSLMODE=disable` по умолчанию

**Файл:** `.env.example` строка 11; `docker-compose.yml` строка 35

Трафик между backend и postgres идёт без шифрования. В рамках одного Docker-network риск минимален, но при вынесении БД на отдельный хост — критично.

**Рекомендация:** Задокументировать в `.env.example` как потенциальную проблему; при выносе БД — включить `DB_SSLMODE=require`.

---

### 🟢 LOW

---

#### L-1 — В Dockerfile используется `golang:1.25-alpine` (нестабильный тег)

**Файл:** `backend/Dockerfile` строка 3

Go 1.25 не является стабильным релизом на момент аудита. Использование RC/pre-release в production создаёт риск неожиданных изменений.

**Рекомендация:** Зафиксировать на `golang:1.24-alpine` (stable) или `golang:1.24.x-alpine`.

---

#### L-2 — `deploy.sh` fallback `cd "$(dirname "$0")"` без проверки

**Файл:** `deploy.sh` строки 5–11

Если ни `/var/www/locator_go`, ни `/var/www/locator` не существуют, скрипт делает `cd` в директорию самого скрипта — это может быть не корень проекта при вызове из cron.

**Рекомендация:**
```bash
else
  echo "ERROR: не найден каталог проекта" >&2
  exit 1
fi
```

---

#### L-3 — Ротация логов — самописный `tail -n 2000`, не `logrotate`

**Файл:** `scripts/cleanup-disk.sh` строки 36–39

**Влияние:** При ротации `tail -n 2000 > tmp && mv tmp f` — есть окно потери логов, если процесс пишет в файл в этот момент.

**Рекомендация:** Настроить `/etc/logrotate.d/locator` с `copytruncate` и `compress`.

---

#### L-4 — Нет staging-окружения, deploy прямо в production

**Доказательство:** Одна ветка `main`, один сервер, один cron. E2E-тесты запускаются в GitHub Actions на ephemeral окружении, но не на реальном сервере перед деплоем.

**Влияние:** Регрессия на production без промежуточной проверки.

**Рекомендация:** Для MVP — добавить хотя бы `smoke test` после деплоя в `pull-and-deploy.sh`:
```bash
curl -sf http://localhost:8080/healthz || { echo "DEPLOY FAILED HEALTHCHECK"; exit 1; }
```

---

#### L-5 — `CI: main.yml` — пустышка без реальной верификации

**Файл:** `.github/workflows/main.yml`

```yaml
- name: Server auto-deploy
  run: |
    echo "Push принят. Сервер ... подтянет main в течение ~2 мин (cron)."
```

Workflow проходит "зелёным" всегда, не проверяет ни сборку, ни деплой.

**Рекомендация:** Добавить хотя бы `curl` до healthz сервера после deploy или объединить с `tests.yml` в один pipeline.

---

### ℹ️ INFO (позитивные наблюдения)

| # | Наблюдение | Файл/источник |
|---|---|---|
| I-1 | Бэкапы работают: 15 daily дампов, последний сегодня в 03:00, ~1.2 МБ | `backups/`, `/var/log/locator-backup.log` |
| I-2 | Все 4 контейнера `healthy` / `running` (uptime db/rabbitmq — 3 недели) | `docker ps` |
| I-3 | Multi-stage Docker builds: builder → alpine (~111 МБ backend, ~62 МБ frontend) | `Dockerfile` |
| I-4 | BuildKit cache mounts с лимитом 3 ГБ и auto-prune | `scripts/docker-build.sh` |
| I-5 | Smart diff в cron-деплое — пересборка только при изменении `backend/`, `frontend/`, `docker-compose.yml` | `scripts/pull-and-deploy.sh` |
| I-6 | `depends_on` с `condition: service_healthy` для db и rabbitmq | `docker-compose.yml` |
| I-7 | Integration и E2E тесты в CI с реальной Postgres и Playwright | `.github/workflows/tests.yml` |
| I-8 | `ALLOW_PROD_DB_WIPE` guard для CI интеграционных тестов | `tests.yml` строка 66 |
| I-9 | db порт доступен только локально: `127.0.0.1:5433:5432` | `docker-compose.yml` |

---

## 3. Приоритизированный план устранения

### Sprint 0 — Немедленно (до следующего деплоя)

| # | Действие | Файл | Усилие |
|---|---|---|---|
| 1 | **Убрать хардкод пароля из Dockerfile; сменить пароль БД** | `backend/Dockerfile` строка 45 | 30 мин |
| 2 | **Ограничить RabbitMQ: `127.0.0.1:5672:5672`, задать не-guest credentials** | `docker-compose.yml`, `.env` | 30 мин |
| 3 | **Освободить диск: `docker buildx prune -f --keep-storage 1gb`** | CLI | 5 мин |
| 4 | **Добавить flock в pull-and-deploy.sh** | `scripts/pull-and-deploy.sh` | 15 мин |

### Sprint 1 — Эта неделя

| # | Действие | Усилие |
|---|---|---|
| 5 | Добавить `start_period` и resource limits в docker-compose.yml | 1 ч |
| 6 | Добавить `golangci-lint` + `govulncheck` + `npm audit` в CI | 2 ч |
| 7 | Добавить `curl /healthz` smoke-check после деплоя в `pull-and-deploy.sh` | 30 мин |
| 8 | Исправить `start_period` для backend healthcheck; добавить healthcheck для frontend | 30 мин |

### Sprint 2 — Следующие 2 недели

| # | Действие | Усилие |
|---|---|---|
| 9 | Настроить TLS (Caddy + Let's Encrypt) | 3 ч |
| 10 | Добавить offsite-бэкап (rclone → S3/Backblaze) | 2 ч |
| 11 | Внедрить Sentry SDK в backend | 3 ч |
| 12 | Реализовать `/readyz` с проверкой DB + RabbitMQ | 2 ч |
| 13 | Вынести миграции из CMD в init-контейнер | 2 ч |
| 14 | Настроить `logrotate` вместо самописного `tail` | 1 ч |

### Backlog

- Убрать захардкоженный IP из скриптов
- Исправить `StrictHostKeyChecking=no`
- Зафиксировать golang на стабильный тег
- Staging-окружение или pre-deploy smoke
- Алертинг на диск > 80%

---

## 4. Acceptance Checks

После каждого исправления проверить:

```bash
# B-1: пароль ушёл из образа
docker history locator_go-backend --no-trunc | grep -i password
# ожидаемо: пусто

# H-1: RabbitMQ недоступен снаружи
ss -tlnp | grep 5672
# ожидаемо: 127.0.0.1:5672

# H-3: resource limits применились
docker inspect locator-backend | python3 -c "import sys,json; c=json.load(sys.stdin)[0]; print(c['HostConfig']['Memory'])"
# ожидаемо: >0 (например, 536870912 = 512M)

# H-4: диск освобождён
df -h / | awk 'NR==2{print $5}'
# ожидаемо: < 80%

# M-3: backend healthcheck start_period
docker inspect locator-backend | python3 -c "import sys,json; c=json.load(sys.stdin)[0]; print(c['Config']['Healthcheck'])"
# ожидаемо: StartPeriod > 0

# M-4: deep healthz
curl -s http://localhost:8080/readyz
# ожидаемо: {"status":"ok","db":"ok","rabbitmq":"ok"}

# H-5: CI pipeline с lint
# После push в PR — проверить что GitHub Actions запускает golangci-lint шаг

# Бэкап: проверить offsite после добавления rclone
ls -la backups/ && rclone ls remote:locator-backups/
```

---

## 5. Допущения / Вне охвата

| | |
|---|---|
| **Вне охвата** | Аудит прав доступа к GitHub repo; настройки firewall/ufw на хосте; Android-приложение (логика, подписи APK); производительность запросов к БД; тарифы и расходы на хостинг |
| **Допущение** | Единственный сервер (87.232.65.52, 20 ГБ) — production и dev совмещены; отдельного staging нет |
| **Допущение** | `.env` с реальными секретами корректно размещён на сервере (в git не попал — `.gitignore` проверен) |
| **Допущение** | Пароль БД в `backend/Dockerfile` (строка 45) является **дефолтным fallback**, но фактически это тот же пароль, что используется в production runtime — поэтому классифицировано как `blocker` |
| **Стале** | Версии базовых образов (postgres:13-alpine от ноября 2025, rabbitmq:3.13-alpine) — актуальны, но требуют периодического обновления; не проверялся CVE-статус на дату аудита |


---
*Источник: [devops-engineer](384fca94-b4b5-4916-a5e5-68258b7c7895). Живые проверки: docker ps/inspect, df, ss — только read-only.*


---

## Дополнение — повторный прогон ([DevOps](1deb7552-f2ba-4928-9c70-9df46bd27c22))

Уникальные / усиленные пункты (не дублируют основной отчёт):

| Sev | ID | Finding |
|-----|-----|---------|
| high | H-EOL | **EOL / version skew образов:** `postgres:13-alpine` (EOL), runtime `alpine:3.18` (EOL), builder `golang:1.25-alpine` при `go.mod`/`CI` на **1.24.x** — обновить до актуальных stable (`postgres:16`, `alpine:3.21`, `golang:1.24-alpine`). |
| medium | M-ENTRY | `backend/entrypoint.sh` существует, но **не** используется Dockerfile (`CMD` дублирует goose inline) — мёртвый код / расхождение. |
| medium | M-PROXY | `TRUSTED_PROXIES` пустой в compose; уточнить политику Gin proxy trust в release. |
| low | L-MAIN | `main.yml` echo с публичным IP сервера в логах Actions — убрать или secret. |
| info | — | Подтверждает B-1 Dockerfile password, open `5672`/`8080`, no flock, no post-deploy health/rollback, no resource limits, no offsite backup. |

Секреты в дополнении **не** приводятся.
