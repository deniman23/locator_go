# syntax=docker/dockerfile:1
# Stage 1: собираем приложение и goose (кэш go mod / build через BuildKit)
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@v3.27.2

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o locator .

# Stage 2: финальный образ
FROM alpine:3.18

RUN apk add --no-cache \
    ca-certificates \
    bash \
    su-exec \
    postgresql-client \
    wget \
    tzdata

WORKDIR /app

RUN mkdir -p /app/static/qrcode /app/static/releases && chmod -R 755 /app/static

COPY --from=builder /app/locator /app/locator
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations
# APK монтируется volume в compose; в образ — только manifest
COPY static/releases/manifest.json /app/static/releases/manifest.json

# Non-secret defaults only; DB_PASSWORD MUST be supplied at runtime via compose env / secret.
# No password default is intentional — container fails fast rather than starting with a baked secret.
ENV \
    DB_HOST=db \
    DB_PORT=5432 \
    DB_NAME=locator_db \
    DB_USER=locator_user \
    DB_SSLMODE=disable

CMD ["bash", "-c", "\
    if [ -z \"$DB_PASSWORD\" ]; then \
      echo '❌ DB_PASSWORD is not set — refusing to start. Set it via compose environment or a secret.'; \
      exit 1; \
    fi; \
    until pg_isready -h $DB_HOST -U $DB_USER -d $DB_NAME; do \
      echo '⏳ waiting for postgres…'; sleep 2; \
    done; \
    goose -dir /migrations postgres \"user=$DB_USER password=$DB_PASSWORD host=$DB_HOST port=$DB_PORT dbname=$DB_NAME sslmode=$DB_SSLMODE\" up; \
    mkdir -p /app/static/qrcode && chmod 755 /app/static/qrcode; \
    exec /app/locator \
"]
