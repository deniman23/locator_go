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

# BuildKit/buildx: кэш с лимитом диска (см. scripts/docker-build.sh)
chmod +x scripts/docker-build.sh
./scripts/docker-build.sh up

echo "Деплой завершён!"