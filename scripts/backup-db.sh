#!/usr/bin/env bash
# Бэкап PostgreSQL в каталог backups/ (gzip). Хранит последние KEEP_DAYS дней.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups}"
KEEP_DAYS="${KEEP_DAYS:-14}"
CONTAINER="${DB_CONTAINER:-locator-db}"
ENV_FILE="${ENV_FILE:-$ROOT/.env}"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "$ENV_FILE"
  set +a
fi

DB_USER="${DB_USER:-locator_user}"
DB_NAME="${DB_NAME:-locator_db}"

mkdir -p "$BACKUP_DIR"
STAMP="$(date +%Y%m%d_%H%M%S)"
OUT="$BACKUP_DIR/${DB_NAME}_${STAMP}.sql.gz"

echo "[$(date -Is)] [backup-db] $DB_NAME -> $OUT"

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "[$(date -Is)] [backup-db] ERROR: контейнер $CONTAINER не запущен" >&2
  exit 1
fi

docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" --no-owner --no-acl | gzip -9 > "$OUT"

SIZE=$(du -h "$OUT" | awk '{print $1}')
echo "[$(date -Is)] [backup-db] OK ($SIZE)"

# Удаляем бэкапы старше KEEP_DAYS
find "$BACKUP_DIR" -maxdepth 1 -type f -name "${DB_NAME}_*.sql.gz" -mtime +"$KEEP_DAYS" -delete 2>/dev/null || true
echo "[$(date -Is)] [backup-db] Храним бэкапы за последние ${KEEP_DAYS} дней"
