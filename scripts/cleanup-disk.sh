#!/usr/bin/env bash
# Безопасная очистка диска: Docker build cache, неиспользуемые образы, старые логи.
# НЕ трогает: volumes (БД), контейнеры, backups/, APK releases.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOG_TAG="[cleanup-disk]"

log() { echo "$(date -Is) $LOG_TAG $*"; }

log "Старт. Диск до: $(df -h / | awk 'NR==2 {print $3"/"$2" ("$5")"}')"

KEEP_STORAGE="${BUILDX_KEEP_STORAGE:-3gb}"

# BuildKit/buildx cache — урезаем до лимита, не удаляем всё (быстрее следующая сборка)
if command -v docker >/dev/null 2>&1; then
  if docker buildx version >/dev/null 2>&1; then
    log "docker buildx prune (keep-storage=$KEEP_STORAGE)..."
    docker buildx prune -f --keep-storage "$KEEP_STORAGE" 2>&1 | tail -3 || true
  fi

  log "docker builder prune (keep-storage=$KEEP_STORAGE)..."
  docker builder prune -f --keep-storage "$KEEP_STORAGE" 2>&1 | tail -3 || true

  log "docker image prune (только dangling)..."
  docker image prune -f 2>&1 | tail -1 || true
fi

# Старые логи приложения (старше 30 дней)
if [[ -d "$ROOT/backend/logs" ]]; then
  find "$ROOT/backend/logs" -type f -name '*.log*' -mtime +30 -delete 2>/dev/null || true
fi

# Ротация системных логов locator (оставляем последние 5 МБ)
for f in /var/log/locator-deploy.log /var/log/locator-backup.log /var/log/locator-cleanup.log; do
  if [[ -f "$f" ]] && [[ $(stat -c%s "$f" 2>/dev/null || echo 0) -gt 5242880 ]]; then
    tail -n 2000 "$f" > "${f}.tmp" && mv "${f}.tmp" "$f"
    log "Урезан $f"
  fi
done

# Старые временные файлы Gradle/APK (не трогаем свежие сборки)
find /tmp -maxdepth 1 -type f -mtime +2 -size +10M -delete 2>/dev/null || true
if [[ -d /root/.gradle/caches ]]; then
  find /root/.gradle/caches -type f -mtime +14 -delete 2>/dev/null || true
fi

log "Готово. Диск после: $(df -h / | awk 'NR==2 {print $3"/"$2" ("$5")"}')"
