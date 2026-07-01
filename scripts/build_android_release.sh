#!/usr/bin/env bash
# Сборка Android-приложения (lctr_app) и публикация APK для OTA.
# Репозиторий по умолчанию: /root/lctr_app (рядом с locator_go).
set -euo pipefail
cd "$(dirname "$0")/.."

REPO_ROOT="$(pwd)"
LCTR_APP_DIR="${LCTR_APP_DIR:-/root/lctr_app}"
LCTR_APP_REPO="${LCTR_APP_REPO:-https://github.com/deniman23/lctr_app.git}"
BASE_URL="${BASE_URL:-http://87.232.65.52:8080}"
ANDROID_HOME="${ANDROID_HOME:-/opt/android-sdk}"
export ANDROID_SDK_ROOT="$ANDROID_HOME"
export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$PATH"

VERSION_CODE="${VERSION_CODE:-}"
VERSION_NAME="${VERSION_NAME:-}"
CHANGELOG="${CHANGELOG:-Сборка с сервера $(hostname -I | awk '{print $1}')}"

if [[ ! -d "$LCTR_APP_DIR/.git" ]]; then
  echo "Клонирую $LCTR_APP_REPO → $LCTR_APP_DIR"
  git clone "$LCTR_APP_REPO" "$LCTR_APP_DIR"
else
  echo "Обновляю $LCTR_APP_DIR"
  # SSH-ключ на сервере может отсутствовать — fetch через HTTPS
  git -C "$LCTR_APP_DIR" fetch https://github.com/deniman23/lctr_app.git main
  git -C "$LCTR_APP_DIR" reset --hard FETCH_HEAD
fi

if [[ -n "$VERSION_CODE" ]] || [[ -n "$VERSION_NAME" ]]; then
  python3 - <<PY
from pathlib import Path
p = Path("$LCTR_APP_DIR/app/version.properties")
lines = p.read_text().splitlines()
out = []
for line in lines:
    if line.startswith("versionCode=") and "$VERSION_CODE":
        out.append(f"versionCode=$VERSION_CODE")
    elif line.startswith("versionName=") and "$VERSION_NAME":
        out.append(f"versionName=$VERSION_NAME")
    else:
        out.append(line)
p.write_text("\n".join(out) + "\n")
print(p.read_text())
PY
fi

if [[ -n "${ANDROID_KEYSTORE_BASE64:-}" ]]; then
  ANDROID_KEYSTORE_PATH="${ANDROID_KEYSTORE_PATH:-$LCTR_APP_DIR/keystore/lctr-release.jks}"
  mkdir -p "$(dirname "$ANDROID_KEYSTORE_PATH")"
  echo "$ANDROID_KEYSTORE_BASE64" | base64 -d > "$ANDROID_KEYSTORE_PATH"
  export ANDROID_KEYSTORE_PATH
fi

KEYSTORE="${ANDROID_KEYSTORE_PATH:-$LCTR_APP_DIR/keystore/lctr-release.jks}"
BUILD_TYPE="debug"
GRADLE_TASK=":app:assembleDebug"

if [[ -f "$KEYSTORE" ]] && [[ -n "${ANDROID_KEYSTORE_PASSWORD:-}" ]]; then
  export ANDROID_KEYSTORE_PATH="$KEYSTORE"
  export ANDROID_KEY_ALIAS="${ANDROID_KEY_ALIAS:-lctr}"
  export ANDROID_KEY_PASSWORD="${ANDROID_KEY_PASSWORD:-$ANDROID_KEYSTORE_PASSWORD}"
  BUILD_TYPE="release"
  GRADLE_TASK=":app:assembleRelease"
  echo "Сборка release (подпись: $KEYSTORE)"
else
  echo "Release keystore не найден — собираю debug (для OTA на телефоне с release нужен lctr-release.jks)"
fi

cd "$LCTR_APP_DIR"
./gradlew "$GRADLE_TASK" \
  -PlocatorApiBase="${BASE_URL}" \
  -PlocatorApiUrl="${BASE_URL}/api/location" \
  --no-daemon

if [[ "$BUILD_TYPE" == "release" ]]; then
  APK=$(find app/build/outputs/apk/release -name "*.apk" ! -name "*-unsigned.apk" 2>/dev/null | head -1)
  [[ -z "$APK" ]] && APK=$(find app/build/outputs/apk/release -name "*.apk" | head -1)
else
  APK=$(find app/build/outputs/apk/debug -name "*.apk" | head -1)
fi
[[ -n "$APK" && -f "$APK" ]] || { echo "APK не найден после сборки" >&2; exit 1; }
APK="$(realpath "$APK")"

echo "APK: $APK ($BUILD_TYPE)"
cd "$REPO_ROOT"

export APK_RELEASES_DIR="${APK_RELEASES_DIR:-$REPO_ROOT/backend/static/releases}"
mkdir -p "$APK_RELEASES_DIR" /var/www/locator_go/static/releases 2>/dev/null || true

BASE_URL="$BASE_URL" ./scripts/publish_apk_release.sh "$APK" "$CHANGELOG"

# Синхронизация в оба каталога releases
PUBLISHED=$(python3 -c "import json; print(json.load(open('$APK_RELEASES_DIR/manifest.json'))['filename'])")
for dir in "$APK_RELEASES_DIR" /var/www/locator_go/static/releases; do
  [[ -d "$dir" ]] || continue
  real_dir="$(realpath "$dir")"
  real_pub="$(realpath "$APK_RELEASES_DIR")"
  [[ "$real_dir" == "$real_pub" ]] && continue
  cp -f "$APK_RELEASES_DIR/$PUBLISHED" "$dir/"
  cp -f "$APK_RELEASES_DIR/manifest.json" "$dir/"
done

echo ""
echo "Опубликовано: $BASE_URL/static/releases/$PUBLISHED"
echo "OTA на user 1:"
curl -sS -X POST "${BASE_URL}/api/admin/releases/publish-update/1" \
  -H "X-API-Key: ${LOCATOR_ADMIN_API_KEY:-change_me}" \
  -H "Content-Type: application/json" | python3 -m json.tool 2>/dev/null || true
