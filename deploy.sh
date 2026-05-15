#!/bin/bash
set -euo pipefail

# Переходим в корневую директорию с проектом, где находится docker-compose.yml
cd /root/locator_go || exit

# Обновляем репозиторий с новыми изменениями
git pull origin main

ENV_FILE="${ENV_FILE:-.env}"
if [ ! -f "$ENV_FILE" ]; then
  echo "Ошибка: нет файла $ENV_FILE — на сервере должен лежать локальный .env с секретами (не коммитится в git)." >&2
  exit 1
fi

# Compose V2: явно указываем файл с секретами (подстановка ${VAR} в compose)
compose_v2_up() {
  docker compose --env-file "$ENV_FILE" up --build -d
}

# Compose V2: `docker compose` (на многих серверах нет устаревшего бинарника `docker-compose`)
if docker compose version >/dev/null 2>&1; then
  compose_v2_up
elif command -v docker-compose >/dev/null 2>&1; then
  # v1 сам читает .env из каталога с compose-файлом; имя должно совпадать с $ENV_FILE
  if [ "$ENV_FILE" != ".env" ]; then
    echo "Ошибка: для docker-compose v1 поддерживается только ENV_FILE=.env" >&2
    exit 1
  fi
  docker-compose up --build -d
else
  echo "Ошибка: не найден ни «docker compose», ни «docker-compose»" >&2
  exit 1
fi

echo "Деплой завершён!"