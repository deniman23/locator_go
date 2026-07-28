# QA Health-Check Audit — Locator

**Режим:** quality audit (MVP-depth)  
**Дата:** 2026-07-28  
**Ограничения:** read-only; тесты не запускались; секреты не извлекались  
**KB:** `docs/knowledge-base/INDEX.md` пуст (предыдущих QA-матриц нет)

---

## 1. Scope & method

| | |
|---|---|
| **Цель** | Оценка тестируемости и риска регрессий критичных путей: auth, device commands, visits/checkpoints, CI/e2e |
| **Стек** | Go backend + React admin + Android (`lctr_app`, отдельный репо — по `docs/TESTING.md`) |
| **Источники** | `backend/**/*_test.go` (19 файлов, ~70 `Test*`), `frontend/**/*.test.ts`, `e2e/`, `frontend/e2e/`, `.github/workflows/tests.yml`, `docs/TESTING.md`, `Makefile`, `backend/router/routes.go`, CodeGraph (`DeviceCommand*`, messaging, visits) |
| **Не делалось** | Запуск suite, покрытие %, прогон Playwright/браузера, Android-репо, prod |

**Пирамида (факт):**

| Слой | Где | CI |
|------|-----|-----|
| Unit Go | `backend/service/*_test.go`, `middleware`, `models` | PR + main (`grep -v integration`) |
| Unit FE | 2 файла utils (`vitest`) | PR + main |
| Integration | `backend/integration/` (Postgres, httptest) | PR + main |
| E2E | `e2e/tests/smoke.spec.ts` (CI); `frontend/e2e/` — отдельный/слабый | **только push на `main`** |
| Lint | `Makefile` (`golangci-lint`, `npm run lint`) | **нет в CI** |

---

## 2. Findings

### blocker

*(нет подтверждённых blocker без runtime-доказательства падения. Ниже — ближайшие к blocker риски релиза.)*

---

### high

#### H1. E2E не на PR; деплой не зависит от Tests
- **Evidence:** `.github/workflows/tests.yml` — job `e2e` с `if: github.event_name == 'push' && github.ref == 'refs/heads/main'`. `.github/workflows/main.yml` — только echo про cron-деплой, без `needs:` на Tests.
- **Quality risk:** UI/auth регрессии уходят в `main` и на сервер до/независимо от e2e; PR зелёный при сломанном логине/картах.
- **Recommendation:** e2e (или урезанный smoke) на PR; деплой только после green Tests (или явный manual gate).

#### H2. DeviceCommand: Ack / Poll / expire / wrong-user почти без unit
- **Evidence:** `device_command_service_test.go` — только `TestEnqueueCommand_invalidType` и проверка map типов. CodeGraph: `Ack` → consumer без unit-cover; DAO `DeviceCommandDAO` без тестов. Интеграция: один happy-path `TestDevice_pollEmptyThenEnqueueAndPoll` (`health_check` → poll → ack ok).
- **Quality risk:** регрессии OTA (`app_update` + progress ack), TTL 15m/48h, `ErrDeviceCommandWrongUser`, fail vs progress — без автообнаружения.
- **Recommendation:** unit с fake DAO: Poll/expire, Ack success/fail/progress, wrong user, location_request + Complete; интеграция для `app_update` progress.

#### H3. RabbitMQ publish→consume→visit не покрыт end-to-end
- **Evidence:** CodeGraph — `Consume`/`Publish`/`RabbitMQClient` без тестов. Integration harness: `noopPub := &messaging.Publisher{}` (`harness.go`). Unit: `visit_process_event_test.go` / geofence — процессор напрямую, без брокера. CI e2e поднимает RabbitMQ, но smoke UI **не** проверяет очередь.
- **Quality risk:** локации пишутся, визиты не появляются — «тихая» поломка критичного бизнес-пути.
- **Recommendation:** узкий integration с Testcontainers/compose RabbitMQ: POST location → сообщение → ProcessEvent → GET visits; или contract-тест Publish/Consume.

#### H4. Ложный смысл `TestVisitEventProcessor_enterExit`
- **Evidence:** `api_test.go` — создаёт checkpoint, GET visits; комментарий: «Process events directly (no RabbitMQ)»; enter/exit **не** вызываются.
- **Quality risk:** ложное ощущение покрытия visit lifecycle на HTTP-слое.
- **Recommendation:** переименовать; добавить реальный enter/exit через `ProcessEvent` + assert визита.

#### H5. Admin UI: почти нет автотестов страниц/auth
- **Evidence:** FE unit только `utils.test.ts`, `locationTrack.test.ts`. E2E CI: login → shell страниц; нет CRUD checkpoint, device commands, users. Компоненты `Login`, `DeviceControlPanel`, `UserManagement`, pages — без тестов.
- **Quality risk:** админские irreversible flows (команды, OTA, users) без регрессионной сетки.
- **Recommendation:** e2e: login fail/success, create checkpoint, enqueue command (API+UI); component/RTL для AuthContext.

---

### medium

#### M1. Два Playwright-контура + рассинхрон Makefile/AGENTS
- **Evidence:** CI → `e2e/` (`docs/TESTING.md`). `Makefile` `frontend-e2e` → `frontend` `test:e2e`. `frontend/e2e/smoke.spec.ts` — только «body visible / <500». AGENTS.md указывает `npm run test:e2e` во frontend.
- **Quality risk:** локально гоняют не тот suite; слабый smoke даёт ложный green.
- **Recommendation:** один канон (`e2e/`); Makefile/AGENTS выровнять; `frontend/e2e` удалить или делегировать.

#### M2. Lint не в CI
- **Evidence:** `Makefile` `backend-lint` / `frontend-lint`; workflows — только tests.
- **Quality risk:** стиль/баги lint не блокируют merge.
- **Recommendation:** добавить lint jobs на PR.

#### M3. HTTP: много admin/device маршрутов без integration
- **Evidence:** `routes.go` — wake, enable-location, device/config, regenerate-qr, publish-update, sync-manifest, backfill, match-route, users CRUD, device/report, location/request — **нет** в `integration/api_test.go`. Checkpoint «CRUD» = create+list, без PUT.
- **Quality risk:** контрактные поломки на admin/device незаметны до ручной проверки.
- **Recommendation:** матрица smoke: report, location_request, checkpoint PUT, user create, publish-update (negative+auth).

#### M4. Нет тестов controllers/dao; интеграция на AutoMigrate
- **Evidence:** 0 `*_test.go` в `controllers/`, `dao/`. Harness `AutoMigrate`, не `backend/migrations/`.
- **Quality risk:** SQL/миграции drift; handler-ошибки только косвенно.
- **Recommendation:** integration прогон миграций; точечные DAO-тесты для command queue ordering.

#### M5. E2E `test.skip` при недоступности UI/ключа
- **Evidence:** `e2e/tests/smoke.spec.ts` — `test.skip` если UI/API key. Документировано в `docs/TESTING.md`.
- **Quality risk:** локально/кривой env → «зелёно» без проверок; в CI ключ задан, риск ниже.
- **Recommendation:** в CI `forbid skip` / fail если skip count > 0.

#### M6. Docs drift: `make test-unit` / `test-e2e` в TESTING.md, в Makefile нет
- **Evidence:** `docs/TESTING.md` vs `Makefile` (только `backend-test`, `frontend-e2e`, …).
- **Quality risk:** онбординг QA ломается.
- **Recommendation:** синхронизировать targets.

#### M7. Android вне репо — контракт device узкий
- **Evidence:** `docs/TESTING.md` — `lctr_app` отдельно; здесь — poll/ack smoke. Нет тестов `device/report`, progress OTA.
- **Quality risk:** рассинхрон клиента и API.
- **Recommendation:** OpenAPI/contract tests + shared fixtures; ссылка на CI `lctr_app`.

---

### low

#### L1. Нет coverage gate
- **Evidence:** `docs/TESTING.md` — soft ≥70%, «No hard % gate».
- **Recommendation:** cover на `service` + middleware в CI (warn → позже fail).

#### L2. Orphan `frontend/e2e/smoke.spec.ts`
- Минимальный smoke, не используется CI.
- **Recommendation:** удалить или не документировать как основной e2e.

#### L3. FE a11y / visual / perf — отсутствуют
- **Recommendation:** 1–2 a11y checks на login/nav в Playwright.

---

### info

#### I1. Сильные стороны (сохранять)
- Auth middleware: missing/invalid/success, admin vs non-admin (`middleware_test.go`).
- User authenticate unit (`user_service_test.go`).
- Глубокие unit: track filter, geofence grace, location quality/create, visits start/end/abandon, ProcessEvent on_demand/exit.
- FE↔Go parity track filters (`locationTrack.test.ts` + `track_filter_test.go`).
- Integration harness: отказ от wipe `locator_db` без `ALLOW_PROD_DB_WIPE` (`harness.go`).
- Документированная пирамида: `docs/TESTING.md`.

#### I2. Объём
- ~70 Go `Test*`, 19 `*_test.go`; FE ~24 `it()` в utils; e2e CI — 4 smoke-сценария.

#### I3. KB пуст
- Повторные аудиты не на что опереться; этот отчёт — baseline.

---

## 3. Prioritized remediation

| Prio | Action | Закрывает |
|------|--------|-----------|
| P0 | E2E (или API smoke) на PR; связать merge/deploy с green Tests | H1 |
| P0 | Unit DeviceCommand: Ack/Poll/expire/wrong-user/app_update progress | H2 |
| P0 | Integration: location publish → visit (RabbitMQ или in-proc + assert) + починить H4 | H3, H4 |
| P1 | E2E: login negative, checkpoint create, device command из UI/API | H5 |
| P1 | Выровнять Playwright/Makefile/AGENTS на `e2e/` | M1, M6 |
| P1 | Lint в CI | M2 |
| P2 | Integration matrix для непроверенных routes | M3 |
| P2 | Fail-on-skip в CI e2e; coverage soft gate | M5, L1 |
| P3 | Contract/Android sync; a11y smoke | M7, L3 |

---

## 4. Acceptance checks

| ID | Тип | Pre | Steps | Expected | AC / риск |
|----|-----|-----|-------|----------|-----------|
| AC-AUTH-01 | API/unit | harness/middleware | me без ключа / неверный / admin / device на admin | 401 / 401 / 200 / 403 | Auth — **частично есть** |
| AC-DEV-01 | API | device+admin | enqueue → poll → ack | команда доставлена, status acked | Happy path — **есть**; типы OTA — **нет** |
| AC-DEV-02 | unit | fake DAO | Ack wrong user; app_update progress; expire TTL | ошибки/статусы по контракту | **gap** |
| AC-VIS-01 | unit | fakes | ProcessEvent inside→visit, exit→end/abandon | визит в БД | **есть** (unit) |
| AC-VIS-02 | integration | Postgres+RMQ | POST location → visits | визит после geo | **gap** |
| AC-CP-01 | API | admin | create+list+update checkpoint | CRUD | create/list **есть**, update **gap** |
| AC-UI-01 | e2e | stack+key | login → dashboard/checkpoints/visits | UI shell | **есть** (main only) |
| AC-UI-02 | e2e | — | неверный ключ | остаёмся на login | **gap** |
| AC-CI-01 | process | PR | unit+integration+lint+(e2e smoke) | обязательный gate | lint/e2e PR **gap** |
| AC-REL-01 | process | main | deploy после Tests | нет деплоя на red | **gap** |

**Release gates (рекомендуемые):**

1. PR: Go unit + FE unit + integration + lint  
2. PR или pre-deploy: e2e smoke login+1 page (без skip)  
3. Перед релизом device: AC-DEV-01 + progress OTA  
4. Перед релизом visits: AC-VIS-01 + AC-VIS-02  

---

## 5. Assumptions / out of scope

- Тесты **не исполнялись** — статусы pass/fail runtime неизвестны; выводы по наличию/смыслу артефактов.
- Android `lctr_app` не аудировался.
- Prod/Sentry/Datadog, нагрузка, ручной exploratory — вне MVP.
- Секреты в CI (`change_me`, e2e keys) — ephemeral; prod secrets не проверялись.
- Предполагается, что CI на GitHub Actions реально используется при PR/push (workflow-файлы есть).

**Вердикт (Go/No-Go качества процесса):** **No-Go как зрелый release gate** для критичных путей device/visits/UI — unit-домен location/geofence сильный, но очередь команд, messaging→visits, e2e-на-PR и связь с деплоем оставляют высокий residual risk.