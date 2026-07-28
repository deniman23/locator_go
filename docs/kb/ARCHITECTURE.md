# ARCHITECTURE — locator_go

**Updated (UTC):** 2026-07-28T11:59:44Z  
**Status:** bootstrap

## Stack

| Layer | Path | Notes |
|-------|------|--------|
| Backend | `backend/` | Go 1.24, Gin, GORM, Postgres, RabbitMQ |
| Frontend | `frontend/` | React 19, MUI, Vite, Leaflet/Mapbox |
| Compose | `docker-compose.yml` | backend `:8080`, frontend `:3000`, db `127.0.0.1:5433`, rabbitmq |
| Deploy | `deploy.sh` + cron pull | push `main` → auto-deploy |

## Layering (backend)

`router` → `controllers` → `service` → `dao` (+ `models`, `migrations`)

HTTP routes только в `backend/router/routes.go`. Не изобретать эндпоинты вне роутера.

## Auth (high level)

- Часть device/user: Basic на `/api`
- Admin: API-key middleware
- Public: `GET /healthz`, `GET /api/app/release/latest`

## UI client

`frontend/src/services/api.ts` — единая точка API для админки.

## Deeper evidence

Свежие findings и риски: `docs/knowledge-base/audits/*/99-consolidated-report.md`.
