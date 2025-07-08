# Stage 1: Builder stage
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Копируем файлы модулей
COPY go.mod go.sum ./

# Исправляем версию в go.mod: убираем патч-версию (например, "go 1.24.3" → "go 1.24")
RUN sed -i 's/go 1\.24\.3/go 1.24/' go.mod

# Загружаем зависимости
RUN go mod download

# Копируем весь исходный код
COPY . .

# Устанавливаем утилиту Goose (для управления миграциями)
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Собираем приложение (бинарник)
# Если ваш основной файл main.go находится в корне проекта, используем ".", а не ./cmd/locator
RUN go build -o locator .

# Stage 2: Финальный образ
FROM alpine:latest
RUN apk add --no-cache ca-certificates bash

# Создаем папку для хранения QR‑кодов
RUN mkdir -p /static/qrcode

# Копируем бинарник приложения из builder
COPY --from=builder /app/locator /locator
# Копируем бинарник goose
COPY --from=builder /go/bin/goose /usr/local/bin/goose
# Копируем миграционные файлы (убедитесь, что каталог "migrations" содержит ваши файлы миграций)
COPY migrations /migrations

# Переменная окружения для подключения к БД
ENV DATABASE_URL="postgres://admin:admin@db:5432/locator_db?sslmode=disable"

# CMD: Сначала применяем миграции, затем запускаем приложение
CMD ["bash", "-c", "goose -dir /migrations postgres \"$DATABASE_URL\" up && /locator"]