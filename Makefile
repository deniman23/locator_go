.PHONY: up down backend-test backend-lint frontend-lint frontend-e2e psql status

up:
	docker compose up -d --build

down:
	docker compose down

backend-test:
	cd backend && go test ./...

backend-lint:
	cd backend && golangci-lint run ./...

frontend-lint:
	cd frontend && npm run lint

frontend-e2e:
	cd frontend && npm run test:e2e

psql:
	@psql "postgres://$${DB_USER:-locator_user}:$${DB_PASSWORD}@127.0.0.1:5433/$${DB_NAME:-locator_db}"

status:
	@export PATH="/usr/local/go/bin:$$HOME/go/bin:$$HOME/.local/bin:$$PATH"; \
	go version; node -v; docker compose version; codegraph status 2>/dev/null | head -20 || true
