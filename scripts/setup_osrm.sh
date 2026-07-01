#!/usr/bin/env bash
# Опционально: свой OSRM с картой Беларуси (15–30 мин, ~2 ГБ на диске).
# По умолчанию Locator использует публичный router.project-osrm.org — этот скрипт не обязателен.
#
# Запуск в фоне: nohup ./scripts/setup_osrm.sh > osrm-setup.log 2>&1 &
# После завершения в .env: ROUTING_BASE_URL=http://osrm:5000
#   ROUTING_MATCH_CHUNK_SIZE=50  ROUTING_MATCH_RADIUS=25
#   docker compose --profile osrm-local up -d osrm
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATA="$ROOT/osrm-data"
PBF="$DATA/belarus-latest.osm.pbf"
OSRM_BASE="$DATA/belarus-latest.osrm"
IMAGE="${OSRM_IMAGE:-ghcr.io/project-osrm/osrm-backend}"

mkdir -p "$DATA"

if [[ -f "${OSRM_BASE}.mldgr" ]]; then
  echo "OSRM уже подготовлен: ${OSRM_BASE}.mldgr"
  exit 0
fi

if [[ ! -f "$PBF" ]]; then
  echo "[1/4] Скачивание OSM Беларуси (~90 МБ)…"
  wget -O "$PBF" https://download.geofabrik.de/europe/belarus-latest.osm.pbf
else
  echo "[1/4] PBF уже есть: $PBF"
fi

echo "[2/4] osrm-extract (5–15 мин)…"
docker run --rm -t -v "$DATA:/data" "$IMAGE" \
  osrm-extract -p /opt/car.lua /data/belarus-latest.osm.pbf

echo "[3/4] osrm-partition…"
docker run --rm -t -v "$DATA:/data" "$IMAGE" \
  osrm-partition /data/belarus-latest.osrm

echo "[4/4] osrm-customize…"
docker run --rm -t -v "$DATA:/data" "$IMAGE" \
  osrm-customize /data/belarus-latest.osrm

echo
echo "Готово. В .env: ROUTING_BASE_URL=http://osrm:5000"
echo "Запуск: cd $ROOT && docker compose --profile osrm-local up -d osrm"
