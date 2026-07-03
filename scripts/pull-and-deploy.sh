#!/usr/bin/env bash
# Автодеплой на сервере: git fetch → pull → docker compose (без SSH из GitHub).
set -euo pipefail

cd /root/locator_go
git fetch origin main
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)
if [[ "$LOCAL" == "$REMOTE" ]]; then
  exit 0
fi
echo "[$(date -Is)] Deploying $LOCAL -> $REMOTE"
git pull origin main

CHANGED=$(git diff --name-only "$LOCAL" "$REMOTE" 2>/dev/null || git diff --name-only HEAD~1 HEAD)
NEEDS_BUILD=false
if echo "$CHANGED" | grep -qE '^(backend/|frontend/|docker-compose\.yml)'; then
  NEEDS_BUILD=true
fi

if [[ "$NEEDS_BUILD" == true ]]; then
  echo "[$(date -Is)] Rebuild: изменились backend/frontend/docker-compose"
  docker compose up --build -d
  # Сборки копят ~1–5 ГБ cache на диске 20 ГБ; чистим сразу после деплоя.
  docker builder prune -af 2>/dev/null | tail -1 || true
  docker image prune -af 2>/dev/null | tail -1 || true
else
  echo "[$(date -Is)] Skip build: только конфиги/доки — docker compose up -d"
  docker compose up -d
fi
echo "[$(date -Is)] Done. Диск: $(df -h / | awk 'NR==2 {print $3"/"$2" ("$5")"}')"
