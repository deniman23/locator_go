#!/bin/bash
# Обновляет manifest.json по уже залитому APK (без копирования).
# Пример:
#   ./scripts/sync_release_manifest.sh /var/www/locator/static/releases/locator-1.0.4-6.apk
set -euo pipefail
cd "$(dirname "$0")/.."

APK_PATH="${1:?Укажите путь к APK}"
BASE_URL="${BASE_URL:-http://178.172.235.51:8080}"
RELEASES_DIR="$(dirname "$(realpath "$APK_PATH")")"
MANIFEST="$RELEASES_DIR/manifest.json"
APK_ABS="$(realpath "$APK_PATH")"
REPO_ROOT="$(pwd)"

META=$(docker run --rm \
  -v "$REPO_ROOT/backend:/app" \
  -v "$(dirname "$APK_ABS"):/apkhost" \
  -w /app golang:1.24-alpine \
  go run ./cmd/apkinfo "/apkhost/$(basename "$APK_ABS")")

VERSION_NAME=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin)['VersionName'])")
VERSION_CODE=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin)['VersionCode'])")
PACKAGE=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin).get('PackageName',''))")
FILENAME="$(basename "$APK_ABS")"
SHA256=$(sha256sum "$APK_ABS" | awk '{print $1}')

python3 - <<PY
import json
m = {
  "version_name": "$VERSION_NAME",
  "version_code": int($VERSION_CODE),
  "package_name": "$PACKAGE",
  "filename": "$FILENAME",
  "sha256": "$SHA256",
  "force": False,
  "changelog": "Synced from $FILENAME",
  "url": "$BASE_URL/static/releases/$FILENAME",
}
open("$MANIFEST", "w").write(json.dumps(m, indent=2) + "\n")
print(json.dumps(m, indent=2))
PY

echo "manifest: $MANIFEST"
