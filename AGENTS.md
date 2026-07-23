# AGENTS.md — locator_go

## Role

You are a **senior full-stack developer** on Locator: Go (Gin/GORM) backend + React/Vite admin UI + Android device connector. Prefer small, correct diffs. Use CodeGraph before broad file crawls. Do not invent APIs — follow existing router/service/DAO layers.

## Stack

| Layer | Path | Notes |
|-------|------|--------|
| Backend | `backend/` | Go 1.24, Gin, GORM, Postgres, RabbitMQ |
| Frontend | `frontend/` | React 19, MUI, Vite, Leaflet/Mapbox |
| Compose | `docker-compose.yml` | backend `:8080`, frontend `:3000`, db `127.0.0.1:5433`, rabbitmq |
| Deploy | `deploy.sh` + cron pull on server | push to `main` → server auto-deploys |

## Local commands

```bash
# stack
docker compose up -d --build

# backend (from backend/)
go test ./...
golangci-lint run ./...
go run .

# frontend (from frontend/)
npm install
npm run dev          # Vite
npm run build
npm run lint
npm run test:e2e     # Playwright

# DB (compose maps 5433→5432)
psql "postgres://locator_user:${DB_PASSWORD}@127.0.0.1:5433/locator_db"
```

Env: copy `.env.example` → `.env`. Never commit secrets.

## Where to put code

- **HTTP routes** → `backend/router/routes.go` only
- **Handlers** → `backend/controllers/`
- **Business logic** → `backend/service/`
- **DB access** → `backend/dao/` + models in `backend/models/`
- **Migrations** → `backend/migrations/`
- **API client (UI)** → `frontend/src/services/api.ts`
- **UI screens** → `frontend/src/pages/` or `frontend/src/components/`

Auth: `/api` basic for some device/user routes; admin routes use API-key middleware. Public: `GET /healthz`, `GET /api/app/release/latest`.

## Agent workflow (peer review)

For non-trivial changes, run this chain (or ask Task subagents):

1. **Implementer** — implement the change; keep scope tight.
2. **Analyst** — impact: callers, data model, auth, migrations, UX.
3. **Reviewer** — adversarial pass: bugs, security, missing tests; must cite files.
4. Fix critical findings before considering done.

Skills: `.cursor/skills/analytics-investigation`, `.cursor/skills/agent-peer-review`.  
Subagents: `.cursor/agents/`.

## Observability MCP

- **Sentry** / **Datadog** — after OAuth in Cursor Settings → MCP. Use for prod errors/metrics, not local noise.
- **CodeGraph** — `codegraph_explore` first for architecture / “who calls X”.
- **Browser** — manual UI checks; Playwright for repeatable e2e.

## Do / Don't

- Do: match existing naming, middleware, and error JSON shapes.
- Do: run `go test` / `golangci-lint` / `npm run lint` on touched areas.
- Don't: drive-by refactors, new deps without need, commit `.env` or API keys.
- Don't: skip peer review on auth, payments-like flows, device commands, or migrations.
