#!/usr/bin/env bash
# Безопасная очистка диска.
# НЕ трогает: docker volumes (БД, QR), работающие контейнеры, backups/, APK releases,
# текущий бинарник cursor-server и самую новую копию cursor-agent.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOG_TAG="[cleanup-disk]"

# На диске 20 ГБ лимит 3gb ничего не освобождал: cache как раз ~3 ГБ.
KEEP_STORAGE="${BUILDX_KEEP_STORAGE:-512mb}"
PRESSURE_KEEP="${BUILDX_KEEP_STORAGE_PRESSURE:-256mb}"
JOURNAL_MAX="${JOURNAL_MAX_SIZE:-200M}"

log() { echo "$(date -Is) $LOG_TAG $*"; }

disk_used_pct() {
  df -P / | awk 'NR==2 { gsub(/%/, "", $5); print $5 }'
}

disk_line() {
  df -h / | awk 'NR==2 { print $3"/"$2" ("$5", free "$4")" }'
}

lock_or_exit() {
  local lock="/var/lock/locator-cleanup.lock"
  exec 9>"$lock"
  if ! flock -n 9; then
    log "Уже идёт другая очистка, выход."
    exit 0
  fi
}

prune_build_cache() {
  local keep="$1"
  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi
  if docker buildx version >/dev/null 2>&1; then
    log "docker buildx prune (keep-storage=$keep)"
    docker buildx prune -f --keep-storage "$keep" 2>&1 | tail -5 || true
  fi
  log "docker builder prune (keep-storage=$keep)"
  docker builder prune -f --keep-storage "$keep" 2>&1 | tail -5 || true
  log "docker image prune (только dangling, образы контейнеров не трогаем)"
  docker image prune -f 2>&1 | tail -3 || true
}

vacuum_journal() {
  if ! command -v journalctl >/dev/null 2>&1; then
    return 0
  fi
  log "journalctl --vacuum-size=$JOURNAL_MAX"
  journalctl --vacuum-size="$JOURNAL_MAX" 2>&1 | tail -5 || true
}

clean_apt_cache() {
  if command -v apt-get >/dev/null 2>&1; then
    log "apt-get clean"
    apt-get clean || true
  fi
}

# Старые распакованные копии IDE. Живой процесс и самый новый каталог остаются.
prune_old_cursor_servers() {
  local base="/root/.cursor-server/bin/linux-x64"
  [[ -d "$base" ]] || return 0
  local live="" newest="" name
  # ps обрезает длинную командную строку, из-за этого живой бинарник выглядел как «none».
  live="$(
    for f in /proc/[0-9]*/cmdline; do
      [[ -r "$f" ]] || continue
      tr '\0' ' ' <"$f" 2>/dev/null || true
      printf '\n'
    done | grep -oE 'cursor-server/bin/linux-x64/[0-9a-f]{40}' | head -1 | sed 's|.*/||'
  )"
  newest="$(find "$base" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %f\n' 2>/dev/null | sort -nr | awk 'NR==1 { print $2 }')"
  local removed=0
  for dir in "$base"/*; do
    [[ -d "$dir" ]] || continue
    name="$(basename "$dir")"
    if [[ -n "$live" && "$name" == "$live" ]]; then
      continue
    fi
    if [[ -n "$newest" && "$name" == "$newest" ]]; then
      continue
    fi
    log "старый cursor-server $name"
    rm -rf "$dir"
    removed=$((removed + 1))
  done
  log "cursor-server: убрано копий $removed (live=${live:-none}, newest=${newest:-none})"
}

# Старые скачанные CLI агента. Остаётся один самый новый каталог версий.
prune_old_cursor_agent_versions() {
  local base="/root/.cursor-server/data/User/globalStorage/anysphere.cursor-agent-worker/agent-cli/.local/share/cursor-agent/versions"
  [[ -d "$base" ]] || return 0
  local name removed=0
  local -a keep_names=()
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    keep_names+=("$name")
    [[ ${#keep_names[@]} -ge 1 ]] && break
  done < <(find "$base" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %f\n' 2>/dev/null | sort -nr | awk '{ print $2 }')
  for dir in "$base"/*; do
    [[ -d "$dir" ]] || continue
    name="$(basename "$dir")"
    local keep=0 kn
    for kn in "${keep_names[@]:-}"; do
      if [[ "$name" == "$kn" ]]; then
        keep=1
        break
      fi
    done
    if [[ "$keep" -eq 1 ]]; then
      continue
    fi
    log "старая версия cursor-agent $name"
    rm -rf "$dir"
    removed=$((removed + 1))
  done
  log "cursor-agent versions: убрано $removed, оставлено ${keep_names[*]:-none}"
}

prune_dev_caches_under_pressure() {
  if [[ -d /root/.cache/ms-playwright ]]; then
    log "удаляю кэш браузеров Playwright (перекачается при e2e)"
    rm -rf /root/.cache/ms-playwright
  fi
  if [[ -d /root/.cache/go-build ]]; then
    log "удаляю кэш go-build"
    rm -rf /root/.cache/go-build
  fi
}

lock_or_exit
log "Старт. Диск до: $(disk_line)"

used="$(disk_used_pct)"
keep="$KEEP_STORAGE"
if [[ "$used" -ge 80 ]]; then
  keep="$PRESSURE_KEEP"
  log "занято ${used}% >= 80, лимит cache снижен до $keep"
fi

prune_build_cache "$keep"
vacuum_journal

if [[ "$used" -ge 80 ]]; then
  clean_apt_cache
fi

if [[ -d "$ROOT/backend/logs" ]]; then
  find "$ROOT/backend/logs" -type f -name '*.log*' -mtime +30 -delete 2>/dev/null || true
fi

for f in /var/log/locator-deploy.log /var/log/locator-backup.log /var/log/locator-cleanup.log; do
  if [[ -f "$f" ]] && [[ $(stat -c%s "$f" 2>/dev/null || echo 0) -gt 5242880 ]]; then
    tail -n 2000 "$f" > "${f}.tmp" && mv "${f}.tmp" "$f"
    log "Урезан $f"
  fi
done

find /tmp -maxdepth 1 -type f -mtime +2 -size +10M -delete 2>/dev/null || true
if [[ -d /root/.gradle/caches ]]; then
  find /root/.gradle/caches -type f -mtime +14 -delete 2>/dev/null || true
fi

prune_old_cursor_servers
prune_old_cursor_agent_versions

used="$(disk_used_pct)"
if [[ "$used" -ge 80 ]]; then
  clean_apt_cache
fi
if [[ "$used" -ge 85 ]]; then
  prune_dev_caches_under_pressure
fi

log "Готово. Диск после: $(disk_line)"
