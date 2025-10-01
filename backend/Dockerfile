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
    wget

# создаем рабочую директорию
WORKDIR /app

# создаем директории для статики
RUN mkdir -p /app/static/qrcode && \
    chmod -R 755 /app/static

# копируем миграции, goose и приложение
COPY --from=builder /app/locator      /app/locator
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

# копируем entrypoint.sh из корня проекта
COPY ./entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]