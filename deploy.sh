#!/bin/bash
set -eo pipefail

# Корень проекта на сервере
if [[ -d /var/www/locator_go ]] && [[ -f /var/www/locator_go/docker-compose.yml ]]; then
  cd /var/www/locator_go
elif [[ -d /var/www/locator ]] && [[ -f /var/www/locator/docker-compose.yml ]]; then
  cd /var/www/locator
else
  cd "$(dirname "$0")"
fi
mkdir -p "${APK_RELEASES_DIR:-/var/www/locator_go/static/releases}"

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