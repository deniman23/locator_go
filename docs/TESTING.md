# Testing Locator

## Pyramid

| Layer | Command | Notes |
|-------|---------|-------|
| Unit (Go) | `make test-unit-go` | stdlib `testing`; excludes `integration/` |
| Unit (FE) | `make test-unit-fe` | Vitest for `frontend/src/utils/*` |
| Integration | `make test-integration` | Needs Postgres; set `INTEGRATION_TEST=1` |
| E2E | `make test-e2e` | Playwright; needs running frontend + backend |

## Unit

```bash
make test-unit
# or separately:
cd backend && go test $(go list ./... | grep -v '/integration')
cd frontend && npm run test:run
```

Shared Go fixtures: `backend/internal/testutil/`.

Frontend track filters mirror Go cases in `backend/service/track_filter_test.go`.

## Integration

Requires a **dedicated** Postgres database. The harness `TRUNCATE`s all domain tables — never point it at production `locator_db`.

```bash
docker compose up -d db
# one-time: create isolated test DB
docker exec locator-db psql -U locator_user -d postgres -c "CREATE DATABASE locator_db_test OWNER locator_user;"

export INTEGRATION_TEST=1
export DB_HOST=127.0.0.1 DB_PORT=5433
export DB_USER=locator_user DB_PASSWORD=change_me DB_NAME=locator_db_test DB_SSLMODE=disable
make test-integration
```

Default `DB_NAME` is `locator_db_test`. Using `locator_db` fails unless `ALLOW_PROD_DB_WIPE=1` (CI ephemeral Postgres only).

Without Postgres the suite **skips** (does not fail). CI `integration` job always has Postgres.

Harness: `backend/integration/` — httptest against full Gin router, noop RabbitMQ publisher.

## E2E (Playwright)

```bash
# stack up (compose or local backend + vite)
export E2E_BASE_URL=http://localhost:3000   # or preview :4173
export E2E_API_KEY="$DEFAULT_ADMIN_API_KEY"
cd e2e && npm install && npx playwright install chromium
make test-e2e
```

If the frontend is unreachable or `E2E_API_KEY` is empty, smoke tests **skip** (except the login-page render check when the UI is up).

CI runs e2e on pushes to `main` only (after unit + integration).

## CI

Workflow: [`.github/workflows/tests.yml`](../.github/workflows/tests.yml)

- PR / `main`: `unit` + `integration`
- `main` push: + `e2e` against ephemeral Postgres + RabbitMQ + backend + vite preview

Deploy notify remains in `.github/workflows/main.yml`.

## Coverage (soft)

```bash
cd backend && go test $(go list ./... | grep -v '/integration') -coverprofile=coverage.out
cd frontend && npx vitest run --coverage
```

No hard % gate yet; aim for critical packages (`service`, middleware, utils) ≥70% over time.

## Android

Device app lives in `lctr_app` (separate repo). This repo covers device HTTP contracts via integration tests (`/api/device/poll`, `report`, `ack`).
