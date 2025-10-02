# Stage 1: собираем приложение и goose
FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
RUN go build -o locator .

# Stage 2: финальный образ
FROM alpine:3.18

# утилиты + tzdata для часовых поясов
RUN apk add --no-cache \
    ca-certificates \
    bash \
    su-exec \
    postgresql-client \
    wget \
    tzdata

WORKDIR /app

# директории под статику
RUN mkdir -p /app/static/qrcode && chmod -R 755 /app/static

# копируем бинарь, goose и миграции
COPY --from=builder /app/locator /app/locator
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations

# передаём DB_* (переопределяются в docker-compose)
ENV \
    DB_HOST=db \
    DB_PORT=5432 \
    DB_NAME=locator_db \
    DB_USER=locator_user \
    DB_PASSWORD=1Y8OEqzBLg \
    DB_SSLMODE=disable


CMD ["bash", "-c", "\
    until pg_isready -h $DB_HOST -U $DB_USER -d $DB_NAME; do \
      echo '⏳ waiting for postgres…'; sleep 2; \
    done; \
    goose -dir /migrations postgres \"user=$DB_USER password=$DB_PASSWORD host=$DB_HOST port=$DB_PORT dbname=$DB_NAME sslmode=$DB_SSLMODE\" up; \
    mkdir -p /app/static/qrcode && chmod 755 /app/static/qrcode; \
    exec /app/locator \
"]