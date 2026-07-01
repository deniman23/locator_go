#!/usr/bin/env bash
# Полный цикл: timestamp API, backfill, OTA manifest, publish-update.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-change_me}"
USER_ID="${USER_ID:-1}"

hdr=(-H "Content-Type: application/json" -H "X-API-Key: ${API_KEY}")

echo "=== 1. POST с timestamp (как Android Location.getTime()) ==="
TS=$(python3 -c "from datetime import datetime,timezone; print(int(datetime(2026,7,1,9,0,tzinfo=timezone.utc).timestamp()*1000))")
curl -sS -X POST "${BASE_URL}/api/location" "${hdr[@]}" -d "{
  \"latitude\": 53.88586,
  \"longitude\": 27.51026,
  \"source\": \"periodic\",
  \"timestamp\": ${TS}
}" | python3 -m json.tool
echo

echo "=== 2. Backfill captured_at (dry-run) user=${USER_ID} ==="
curl -sS -X POST "${BASE_URL}/api/admin/locations/backfill-captured-at?user_id=${USER_ID}&dry_run=true" "${hdr[@]}" | python3 -m json.tool
echo

echo "=== 3. Backfill captured_at (apply) user=${USER_ID} ==="
curl -sS -X POST "${BASE_URL}/api/admin/locations/backfill-captured-at?user_id=${USER_ID}" "${hdr[@]}" | python3 -m json.tool
echo

echo "=== 4. Latest release manifest ==="
curl -sS "${BASE_URL}/api/app/release/latest" | python3 -m json.tool
echo

echo "=== 5. OTA app_update command → user ${USER_ID} ==="
curl -sS -X POST "${BASE_URL}/api/admin/releases/publish-update/${USER_ID}" "${hdr[@]}" | python3 -m json.tool
echo

echo "Готово."
