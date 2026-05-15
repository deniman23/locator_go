#!/bin/bash
set -eo pipefail

# Переходим в корневую директорию с проектом, где находится docker-compose.yml
cd /root/locator_go

# Обновляем репозиторий с новыми изменениями
git pull origin main

# Пересобираем все контейнеры (в том числе и frontend, и backend) и запускаем их в фоне
# Compose V2: `docker compose` (на многих серверах нет устаревшего бинарника `docker-compose`)
if docker compose version >/dev/null 2>&1; then
  docker compose up --build -d
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose up --build -d
else
  echo "Ошибка: не найден ни «docker compose», ни «docker-compose»" >&2
  exit 1
fi

echo "Деплой завершён!"