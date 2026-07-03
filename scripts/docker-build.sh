#!/usr/bin/env bash
# Сборка через BuildKit/buildx: cache mounts в Dockerfile + локальный cache (быстрее, меньше мусора).
# Использование:
#   ./scripts/docker-build.sh up          # build + prune + docker compose up -d
#   ./scripts/docker-build.sh build       # только образы
#   ./scripts/docker-build.sh prune       # урезать cache до лимита
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

BUILDX_CACHE_DIR="${BUILDX_CACHE_DIR:-/var/cache/locator-buildx}"
BUILDX_KEEP_STORAGE="${BUILDX_KEEP_STORAGE:-3gb}"

mkdir -p "$BUILDX_CACHE_DIR"/backend "$BUILDX_CACHE_DIR"/frontend
export BUILDX_CACHE_DIR

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

ensure_buildx() {
  if ! docker buildx version >/dev/null 2>&1; then
    echo "[docker-build] buildx недоступен — обычный BuildKit"
    return 0
  fi
  # default driver: без лишнего buildkit-контейнера на диске 20 ГБ
  if ! docker buildx inspect default >/dev/null 2>&1; then
    docker buildx create --use --name default
  else
    docker buildx use default >/dev/null 2>&1 || true
  fi
}

build_images() {
  ensure_buildx
  echo "[docker-build] cache: $BUILDX_CACHE_DIR (лимит после сборки: $BUILDX_KEEP_STORAGE)"
  compose build --parallel "$@"
}

prune_build_cache() {
  echo "[docker-build] prune cache (keep-storage=$BUILDX_KEEP_STORAGE)…"
  if docker buildx version >/dev/null 2>&1; then
    docker buildx prune -f --keep-storage "$BUILDX_KEEP_STORAGE" 2>&1 | tail -3 || true
  fi
  docker builder prune -f --keep-storage "$BUILDX_KEEP_STORAGE" 2>&1 | tail -3 || true
  # Только dangling-слои; рабочие образы не трогаем
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
