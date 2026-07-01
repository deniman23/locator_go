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
docker compose up --build -d
echo "[$(date -Is)] Done"
