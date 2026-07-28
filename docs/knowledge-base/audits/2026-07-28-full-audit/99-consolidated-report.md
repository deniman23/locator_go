# Consolidated Full Audit — Locator

**Дата:** 2026-07-28  
**Репозиторий:** `/root/locator_go`  
**Ограничение пользователя:** общая проверка, **не затирать данные** — только append KB markdown  

---

## 1. Scope & method

### Scope
Кросс-доменный health-check: BA, UX/UI, Frontend, Backend, QA, DevOps, Security (readonly).  
Стек: Go 1.24 (Gin/GORM/Postgres/RabbitMQ) + React 19/Vite admin + Android connector.

### Method
1. Bootstrap KB (`README` / `INDEX` / `AGENTS`) — созданы с нуля (ранее отсутствовали).  
2. Параллельный запуск 7 специалистов (Task).  
3. CodeGraph (`codegraph_explore`) для auth, routes, visits, device commands.  
4. Синтез оркестратора → domain files + этот отчёт.  
5. **Не** трогались: application source, DB, `.env` values, docker volumes, prod configs.

### Completeness note
Все 7 domain-отчётов заполнены ответами специалистов (поздние BA/BE/DevOps/Security дописаны после завершения субагентов). Секретные значения в KB заредact'аны.

### Severity legend
`blocker/critical` → `high` → `medium` → `low` → `info`

---

## 2. Executive summary (health by domain)

| Domain | Health | One-liner |
|--------|--------|-----------|
| Business Analyst | 🟡 | MVP сценарии есть; нет formal BRD визитов/QR/device policy |
| UX/UI Designer | 🟡 | Desktop usable; mobile map broken; visits/checkpoint UX gaps |
| Frontend Developer | 🟡 | Build green; security/UX debt (key in DOM, dead deps, a11y) |
| Backend Developer | 🔴 | Auth O(N)×bcrypt; public QR; migration dualism; RMQ retry risk |
| QA Tester | 🔴 | Strong unit geofence; weak device/RMQ/e2e-PR/deploy coupling |
| DevOps Engineer | 🔴 | Secrets in image/git; exposed AMQP; deploy≠CI; no rollback |
| Security Engineer | 🔴 | CRITICAL QR leak + Dockerfile/.env secrets; HTTP; raw commands |

**Overall:** 🔴 **Production NO-GO** until P0 security/delivery closed. Desktop MVP demo — условно допустим в доверенной сети после ротации ключей и закрытия static QR.

---

## 3. Cross-cutting risks (≥2 domains)

| Theme | Domains | Evidence anchors |
|-------|---------|------------------|
| **Публичный QR = plaintext API key** | Sec, BE, BA, FE, DevOps | `writeUserQRCode`, `router.Static("/static")`, SEC-C01, BE-H1, BA-H2 |
| **Секреты в git/Dockerfile** | Sec, DevOps | SEC-C02/C03, DO-B1 |
| **Auth scalability / DoS** | Sec, BE, QA | `AuthenticateUser` GetAll+bcrypt, SEC-H03, BE-B1 |
| **HTTP / no TLS + hard-coded IP** | Sec, DevOps, BE | compose ports, device_controller BASE_URL fallback |
| **Device command power without gates** | Sec, BA, BE, QA | raw `PostAdminUserCommand`, weak tests H2 |
| **Deploy without quality gate** | DevOps, QA | `main.yml` echo vs cron pull; e2e only on main |
| **RabbitMQ exposure / poison** | DevOps, Sec, BE | `5672:5672`; requeue without DLQ |
| **Mobile / field admin UX** | UX, FE, QA | Map sidebar always open; weak e2e |
| **Observability readiness** | DevOps, BE | healthz ≠ deps; no readyz |

---

## 4. Unified prioritized backlog

### P0 — сделать до следующего prod deploy

| ID | Action | Owner |
|----|--------|-------|
| P0-1 | Убрать QR PNG из публичного Static; отдавать только под auth; **ротировать все API keys** | BE + Sec + DevOps |
| P0-2 | Удалить секреты из `backend/Dockerfile`; `git rm --cached backend/.env`; scrub/rotate credentials | DevOps + Sec |
| P0-3 | Переписать `AuthenticateUser` (O(1) lookup, не GetAll×bcrypt) + rate limit 401 | BE |
| P0-4 | Bind RabbitMQ (и OSRM) на `127.0.0.1`; запрет default `change_me`/`guest` в release | DevOps + BE |
| P0-5 | TLS termination перед `:8080`/`:3000`; убрать hard-coded prod IP fallback | DevOps + BE |

### P1 — следующий спринт

| ID | Action | Owner |
|----|--------|-------|
| P1-1 | Deploy gate: только после green CI; post-deploy health + rollback по SHA | DevOps |
| P1-2 | Запретить/валидировать raw admin `config_update`/`app_update` (только builders) | BE + Sec |
| P1-3 | Unit tests DeviceCommand Poll/Ack/expire/wrong-user; integration location→visit | QA + BE |
| P1-4 | E2E smoke на PR + lint в CI | QA + DevOps |
| P1-5 | Mobile map: collapsible sidebar + nav drawer | UX + FE |
| P1-6 | Visits filters by name; checkpoint map picker + validation | UX + FE |
| P1-7 | BRD visit lifecycle + soft-delete checkpoints | BA + BE |
| P1-8 | Pagination/limits на locations; `/readyz` | BE |

### P2 — backlog

| ID | Action | Owner |
|----|--------|-------|
| P2-1 | Graceful shutdown; RMQ DLQ | BE |
| P2-2 | Single migration strategy (goose only in prod) | BE + DevOps |
| P2-3 | FE: ErrorBoundary, axios 401 interceptor, drop unused map deps, security headers | FE |
| P2-4 | DB backup/restore runbook; non-root container | DevOps |
| P2-5 | Retention policy PII; device offline runbook | BA + Sec |
| P2-6 | Modal a11y; copy key UX; design tokens | UX + FE |
| P2-7 | govulncheck / npm audit / gitleaks в CI | Sec + DevOps |

---

## 5. Quick wins (≤1 day each)

1. Bind `127.0.0.1:5672:5672` в compose  
2. Удалить hard-coded IP fallback → require `BASE_URL`  
3. Отключить `router.Static` для `qrcode/` или закрыть auth proxy  
4. Добавить lint job в `tests.yml`  
5. Fail-fast seed/compose если password/`API_KEY` == `change_me` в release  
6. FE: не показывать полный API key в DOM (mask + copy)  
7. Документ `PRODUCT.md` / visit lifecycle 1-pager  

---

## 6. Recommended next slices

1. **Security hardening slice** (P0-1…P0-5) + peer review security-engineer  
2. **Delivery slice** (CI gate, rollback, lint/e2e on PR)  
3. **Device reliability slice** (command tests, validated config only, offline runbook)  
4. **Admin UX slice** (mobile map, visits filters, checkpoint picker)  
5. Повторный `/kb-study` или точечный re-audit Sec+DevOps через 1–2 недели  

---

## 7. Links to domain reports

| # | Domain | Path |
|---|--------|------|
| 00 | Executive | [00-executive-summary.md](./00-executive-summary.md) |
| 01 | Business Analyst | [01-business-analyst.md](./01-business-analyst.md) |
| 02 | UX/UI | [02-ux-ui-designer.md](./02-ux-ui-designer.md) |
| 03 | Frontend | [03-frontend-developer.md](./03-frontend-developer.md) |
| 04 | Backend | [04-backend-developer.md](./04-backend-developer.md) |
| 05 | QA | [05-qa-tester.md](./05-qa-tester.md) |
| 06 | DevOps | [06-devops-engineer.md](./06-devops-engineer.md) |
| 07 | Security | [07-security-engineer.md](./07-security-engineer.md) |

---

## 8. Residual risks (после P0)

Даже после закрытия P0 останутся: MitM если TLS отложен; XSS→sessionStorage key; OTA supply-chain зависит от Android sha256; отсутствие backup; admin raw power до P1-2; mobile UX до P1-5.

## 9. Data safety confirmation

- Создана **новая** папка `audits/2026-07-28-full-audit/` (ранее отчётов не было).  
- Bootstrap `README.md` / `INDEX.md` / `AGENTS.md` созданы (файлов не существовало).  
- **Не** перезаписывались application code, БД, volumes, production configs.  
- Секретные значения в отчёты **не** копировались.

### Примечание оркестратора
Поздние повторные прогоны DevOps/FE добавлены как секции «Дополнение» в `06-devops-engineer.md` и `03-frontend-developer.md` (без перезаписи основных отчётов).
