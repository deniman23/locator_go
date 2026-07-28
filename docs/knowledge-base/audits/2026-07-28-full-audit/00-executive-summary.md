# Executive Summary — Full Audit Locator

**Дата:** 2026-07-28  
**Папка:** `docs/knowledge-base/audits/2026-07-28-full-audit/`  
**Режим:** append-only KB; application/DB/.env не изменялись

## Вердикт

Locator как **desktop MVP мониторинга** функционально связан (карта → визиты → чекпоинты → устройства), но для production **NO-GO** до закрытия критичных AppSec и delivery gaps: публичные QR с API-ключами, секреты в Dockerfile/git, O(N)×bcrypt auth, HTTP без TLS, deploy без CI gate.

## Здоровье по доменам

| Домен | Статус | Одна строка |
|-------|--------|-------------|
| Business Analyst | 🟡 | MVP ясен, но нет BRD визитов/enrollment и политики device commands |
| UX/UI | 🟡 | Desktop OK; mobile map непригоден; visits filters и checkpoint picker слабые |
| Frontend | 🟡 | Сборка чистая; ключ в DOM после regenerate; мёртвые deps; мало a11y |
| Backend | 🔴 | Auth DoS O(N)×bcrypt; public QR; dual migrations; RMQ poison loop risk |
| QA | 🔴 | Unit geofence сильный; device/RMQ/e2e-on-PR/deploy gate — дыры |
| DevOps | 🔴 | Секреты в артефактах; RabbitMQ exposed; deploy≠CI; нет rollback |
| Security | 🔴 | CRITICAL: QR static, Dockerfile secrets, tracked `.env` |

## Top 5 срочных действий

1. Закрыть публичный `/static/qrcode/*.png` и ротировать ключи  
2. Убрать секреты из `backend/Dockerfile` и `git`-tracked `backend/.env`, ротация  
3. Переписать `AuthenticateUser` без полного скана + bcrypt×N  
4. TLS + bind RabbitMQ на localhost + запрет дефолтов `change_me`  
5. Связать deploy с зелёным CI и добавить post-deploy health/rollback  

Полный backlog и cross-cutting: [`99-consolidated-report.md`](./99-consolidated-report.md).

## Источники отчётов

| Файл | Источник |
|------|----------|
| 01 BA | Оркестратор (evidence), BA-agent не завершил markdown |
| 02 UX | Специалист ux-ui-designer |
| 03 FE | Специалист frontend-developer |
| 04 BE | Оркестратор по evidence + partial BE-agent |
| 05 QA | Специалист qa-tester |
| 06 DevOps | Оркестратор по evidence |
| 07 Security | Специалист security-engineer |

## Дополнение

Domain-отчёты BA / Backend / DevOps / Security обновлены полными ответами специалистов после завершения субагентов (append в ту же папку аудита; код приложения не менялся).
