#!/usr/bin/env bash
# Сборка через BuildKit: cache mounts в Dockerfile + prune с лимитом (быстрее, диск не забивается).
# Использование:
#   ./scripts/docker-build.sh up          # build + prune + docker compose up -d
#   ./scripts/docker-build.sh build       # только образы
#   ./scripts/docker-build.sh prune       # урезать cache до лимита
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

BUILDX_KEEP_STORAGE="${BUILDX_KEEP_STORAGE:-3gb}"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    echo "docker compose not found" >&2
    exit 1
  fi
}

ensure_buildkit() {
  if docker buildx version >/dev/null 2>&1; then
    docker buildx use default >/dev/null 2>&1 || docker buildx create --use --name default
  fi
}

build_images() {
  ensure_buildkit
  echo "[docker-build] BuildKit cache mounts; prune limit: $BUILDX_KEEP_STORAGE"
  compose build --parallel "$@"
}

prune_build_cache() {
  echo "[docker-build] prune cache (keep-storage=$BUILDX_KEEP_STORAGE)…"
  if docker buildx version >/dev/null 2>&1; then
    docker buildx prune -f --keep-storage "$BUILDX_KEEP_STORAGE" 2>&1 | tail -3 || true
  fi
  docker builder prune -f --keep-storage "$BUILDX_KEEP_STORAGE" 2>&1 | tail -3 || true
  docker image prune -f 2>&1 | tail -1 || true
}

up_services() {
  compose up -d "$@"
}

cmd="${1:-up}"
shift || true

case "$cmd" in
  build)
    build_images "$@"
    prune_build_cache
    ;;
  up)
    build_images
    prune_build_cache
    up_services "$@"
    ;;
  prune)
    prune_build_cache
    ;;
  *)
    echo "Usage: $0 {up|build|prune} [compose args…]" >&2
    exit 1
    ;;
esac

echo "[docker-build] Диск: $(df -h / | awk 'NR==2 {print $3"/"$2" ("$5")"}')"
