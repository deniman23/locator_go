#!/bin/bash
# Публикует APK в каталог releases и обновляет manifest.json.
# versionCode/versionName — из AndroidManifest внутри APK.
#
# На сервере:
#   export APK_RELEASES_DIR=/var/www/locator/static/releases
#   ./scripts/publish_apk_release.sh /var/www/locator/static/releases/locator-1.0.4-6.apk
set -euo pipefail
cd "$(dirname "$0")/.."

APK_SRC="${1:?Укажите путь к APK}"
CHANGELOG="${2:-}"
BASE_URL="${BASE_URL:-http://87.232.65.52:8080}"

if [[ -d /var/www/locator_go/static/releases ]]; then
  DEFAULT_RELEASES="/var/www/locator_go/static/releases"
elif [[ -d /var/www/locator/static/releases ]]; then
  DEFAULT_RELEASES="/var/www/locator/static/releases"
else
  DEFAULT_RELEASES="backend/static/releases"
fi
RELEASES_DIR="${APK_RELEASES_DIR:-$DEFAULT_RELEASES}"
MANIFEST="$RELEASES_DIR/manifest.json"

if [[ ! -f "$APK_SRC" ]]; then
  echo "APK не найден: $APK_SRC" >&2
  exit 1
fi

mkdir -p "$RELEASES_DIR"
APK_ABS="$(realpath "$APK_SRC")"
REPO_ROOT="$(pwd)"

META=$(docker run --rm \
  -v "$REPO_ROOT/backend:/app" \
  -v "$(dirname "$APK_ABS"):/apkhost" \
  -w /app golang:1.24-alpine \
  go run ./cmd/apkinfo "/apkhost/$(basename "$APK_ABS")")

VERSION_NAME=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin)['VersionName'])")
VERSION_CODE=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin)['VersionCode'])")
PACKAGE=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin).get('PackageName',''))")

# locator-1.0.4-6.apk
FILENAME="locator-${VERSION_NAME}-${VERSION_CODE}.apk"
cp "$APK_ABS" "$RELEASES_DIR/$FILENAME"
SHA256=$(sha256sum "$RELEASES_DIR/$FILENAME" | awk '{print $1}')
if [[ -z "$CHANGELOG" ]]; then
  CHANGELOG="Release ${VERSION_NAME} (${VERSION_CODE})"
fi

python3 - <<PY
import json
m = {
  "version_name": "$VERSION_NAME",
  "version_code": int($VERSION_CODE),
  "package_name": "$PACKAGE",
  "filename": "$FILENAME",
  "sha256": "$SHA256",
  "force": False,
  "changelog": """$CHANGELOG""",
  "url": "$BASE_URL/static/releases/$FILENAME",
}
open("$MANIFEST", "w").write(json.dumps(m, indent=2) + "\n")
print(json.dumps(m, indent=2))
PY

echo ""
echo "Файл: $RELEASES_DIR/$FILENAME"
echo "URL:  $BASE_URL/static/releases/$FILENAME"

if [[ -f docker-compose.yml ]]; then
  docker compose up -d backend 2>/dev/null || docker-compose up -d backend 2>/dev/null || true
fi
