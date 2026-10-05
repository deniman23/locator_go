#!/usr/bin/env bash
# Автодеплой на сервере: git fetch → pull → docker compose (без SSH из GitHub).
set -euo pipefail

cd /root/locator_go
exec 8>/var/lock/locator-deploy.lock
if ! flock -n 8; then
  echo "[$(date -Is)] Deploy already running"
  exit 0
fi
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

PREV="$LOCAL"
if [[ "$NEEDS_BUILD" == true ]]; then
  echo "[$(date -Is)] Rebuild: изменились backend/frontend/docker-compose"
  chmod +x scripts/docker-build.sh scripts/cleanup-disk.sh
  ./scripts/cleanup-disk.sh || true
  ./scripts/docker-build.sh up
else
  echo "[$(date -Is)] Skip build: только конфиги/доки — docker compose up -d"
  docker compose up -d
fi

if ! curl -fsS --max-time 10 http://127.0.0.1:8080/healthz >/dev/null; then
  echo "[$(date -Is)] healthz failed, откат на $PREV"
  git checkout "$PREV"
  docker compose up -d
  exit 1
fi
echo "[$(date -Is)] Done. Диск: $(df -h / | awk 'NR==2 {print $3"/"$2" ("$5")"}')"
