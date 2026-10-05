#!/usr/bin/env bash
# Свои тайлы карты Беларуси (OpenMapTiles-схема) для контейнера tiles (tileserver-gl).
# Один раз: ~10–30 мин, временно ~2–3 ГБ на диске, итог ~0.5 ГБ в tiles-data/.
#
# Запуск в фоне: nohup ./scripts/setup_tiles.sh > tiles-setup.log 2>&1 &
# После завершения: docker compose up -d tiles && docker compose up -d --build --no-deps frontend
# Обновить карту: удалить tiles-data/belarus.mbtiles и запустить скрипт снова.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATA="$ROOT/tiles-data"
OUT="$DATA/belarus.mbtiles"
IMAGE="${PLANETILER_IMAGE:-ghcr.io/onthegomap/planetiler:latest}"
HEAP="${PLANETILER_HEAP:-2g}"

mkdir -p "$DATA"

if [[ -f "$OUT" ]]; then
  echo "Тайлы уже готовы: $OUT"
  exit 0
fi

echo "[1/2] Planetiler: скачивание OSM Беларуси и сборка тайлов…"
docker run --rm -t \
  -e JAVA_TOOL_OPTIONS="-Xmx$HEAP" \
  -v "$DATA:/data" \
  "$IMAGE" \
  --download --area=belarus \
  --download_dir=/data/sources --tmpdir=/data/tmp \
  --output=/data/belarus.new.mbtiles \
  --force

mv "$DATA/belarus.new.mbtiles" "$OUT"

echo "[2/2] Удаление исходников загрузки (sources/, tmp/)…"
rm -rf "$DATA/sources" "$DATA/tmp"

echo
echo "Готово: $OUT ($(du -h "$OUT" | cut -f1))"
echo "Запуск: cd $ROOT && docker compose up -d tiles"
