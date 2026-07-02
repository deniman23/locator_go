#!/usr/bin/env bash
# Безопасная очистка диска: Docker build cache, неиспользуемые образы, старые логи.
# НЕ трогает: volumes (БД), контейнеры, backups/, APK releases.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOG_TAG="[cleanup-disk]"

log() { echo "$(date -Is) $LOG_TAG $*"; }

log "Старт. Диск до: $(df -h / | awk 'NR==2 {print $3"/"$2" ("$5")"}')"

# Только build cache — самый большой потребитель после сборок
if command -v docker >/dev/null 2>&1; then
  log "docker builder prune..."
  docker builder prune -af 2>&1 | tail -1 || true

  log "docker image prune (только неиспользуемые образы, volumes не трогаем)..."
  docker image prune -af 2>&1 | tail -1 || true
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

log "Готово. Диск после: $(df -h / | awk 'NR==2 {print $3"/"$2" ("$5")"}')"
