#!/usr/bin/env bash
# Проверка POST /api/location с captured_at (офлайн-очередь).
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-change_me}"

post() {
  local body="$1"
  curl -sS -w "\nHTTP %{http_code}\n" -X POST "${BASE_URL}/api/location" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: ${API_KEY}" \
    -d "$body"
}

echo "=== 1. Точка с captured_at в прошлом (как офлайн) ==="
post '{
  "latitude": 53.88586,
  "longitude": 27.51026,
  "source": "periodic",
  "captured_at": "2026-07-01T07:00:00+03:00"
}'

echo
echo "=== 2. Вторая точка +5 мин по captured_at, received сейчас ==="
post '{
  "latitude": 53.88590,
  "longitude": 27.51030,
  "source": "periodic",
  "captured_at": "2026-07-01T07:05:00+03:00"
}'

echo
echo "=== 3. Выброс (13 км за 5 мин captured) — должен skip ==="
post '{
  "latitude": 53.92684,
  "longitude": 27.69516,
  "source": "periodic",
  "captured_at": "2026-07-01T07:10:00+03:00"
}'

echo
echo "=== 4. Возврат домой — должен принять (snap-back к якорю) ==="
post '{
  "latitude": 53.88586,
  "longitude": 27.51026,
  "source": "periodic",
  "captured_at": "2026-07-01T07:15:00+03:00"
}'

echo
echo "=== 5. GET raw за утро 01.07 (Минск) — сортировка по captured_at ==="
curl -sS "${BASE_URL}/api/location/?from=2026-07-01T07:00&to=2026-07-01T08:00&raw=true" \
  -H "X-API-Key: ${API_KEY}" | python3 -m json.tool 2>/dev/null | head -40

echo
echo "Готово."
