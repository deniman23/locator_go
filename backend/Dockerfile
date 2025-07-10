# Stage 1: сборка вашего Go-бинаря и goose
FROM golang:1.24-alpine AS builder
WORKDIR /app

# модули
COPY go.mod go.sum ./
RUN go mod download

# исходники
COPY . .

# утилита миграций
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# собираем основное приложение
RUN go build -o locator .



# Stage 2: финальный образ
FROM alpine:3.18
# C-утилита su-exec вместо gosu, плюс bash и сертификаты
RUN apk add --no-cache ca-certificates bash su-exec

# для wget (если нужно) и папки QR
RUN apk add --no-cache wget \
 && mkdir -p /static/qrcode

# копируем бины и миграции
COPY --from=builder /app/locator       /locator
COPY --from=builder /go/bin/goose      /usr/local/bin/goose
COPY migrations                        /migrations

# строка подключения
ENV DATABASE_URL="postgres://locator_user:1Y8OEqzBLg@db:5432/locator_db?sslmode=disable"

# запускаем миграции, а затем приложение под postgres:postgres
CMD ["bash", "-c", \
     "goose -dir /migrations postgres \"$DATABASE_URL\" up && exec su-exec postgres /locator"]
