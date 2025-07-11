# Stage 1: собираем приложение и goose
FROM golang:1.24-alpine AS builder
WORKDIR /app

# грузим зависимости
COPY go.mod go.sum ./
RUN go mod download

# копируем код и ставим goose
COPY . .
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# билдим бинарник
RUN go build -o locator .

# Stage 2: финальный образ
FROM alpine:3.18

# шорткат для запуска от non-root и минимальные утилиты
RUN apk add --no-cache \
    ca-certificates \
    bash \
    su-exec \
    postgresql-client \
    wget \
  && mkdir -p /static/qrcode

# копируем миграции, goose и приложение
COPY --from=builder /app/locator      /locator
COPY --from=builder /go/bin/goose     /usr/local/bin/goose
COPY migrations                      /migrations

# передаём DB_* из docker-compose
ENV \
    DB_HOST=db \
    DB_PORT=5432 \
    DB_NAME=locator_db \
    DB_USER=locator_user \
    DB_PASSWORD=1Y8OEqzBLg \
    DB_SSLMODE=disable

# запускаем миграции и приложение
CMD ["bash", "-c", "\
    # ждём, когда Postgres будет готов \
    until pg_isready -h $DB_HOST -U $DB_USER -d $DB_NAME; do \
      echo '⏳ waiting for postgres…'; sleep 2; \
    done; \
    # запускаем goose через DSN, собранный из DB_* \
    goose -dir /migrations postgres \"user=$DB_USER password=$DB_PASSWORD host=$DB_HOST port=$DB_PORT dbname=$DB_NAME sslmode=$DB_SSLMODE\" up; \
    # стартуем приложение под postgres-пользователем \
    exec su-exec postgres /locator \
"]
