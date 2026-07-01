#!/usr/bin/env bash
# Деплой locator_go на сервер (тот же SSH, что lctr_app/scripts/ci/deploy-to-server.sh).
set -euo pipefail

: "${DEPLOY_SSH_HOST:?DEPLOY_SSH_HOST}"
: "${DEPLOY_SSH_USER:?DEPLOY_SSH_USER}"
: "${DEPLOY_SSH_KEY_FILE:?DEPLOY_SSH_KEY_FILE}"

DEPLOY_SSH_PORT="${DEPLOY_SSH_PORT:-22}"
REMOTE_DIR="${LOCATOR_GO_REMOTE_DIR:-/root/locator_go}"

SSH_BASE=(
  -i "$DEPLOY_SSH_KEY_FILE"
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o BatchMode=yes
  -o ConnectTimeout=30
  -o ServerAliveInterval=15
  -o ServerAliveCountMax=4
  -o TCPKeepAlive=yes
)
SSH=(ssh "${SSH_BASE[@]}" -p "$DEPLOY_SSH_PORT" "${DEPLOY_SSH_USER}@${DEPLOY_SSH_HOST}")

echo "Deploy dir: $REMOTE_DIR"
echo "SSH port:   $DEPLOY_SSH_PORT"

echo "SSH preflight..."
"${SSH[@]}" "echo SSH_OK && hostname"

echo "Pull + docker compose..."
"${SSH[@]}" bash -s "$REMOTE_DIR" <<'REMOTE'
set -euo pipefail
cd "$1"
git pull origin main
if docker compose version >/dev/null 2>&1; then
  docker compose up --build -d
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose up --build -d
else
  echo "docker compose not found" >&2
  exit 1
fi
docker compose ps 2>/dev/null || docker-compose ps
REMOTE

echo "Deploy finished."
