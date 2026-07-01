#!/usr/bin/env bash
# Сборка подписанного APK 1.0.6 из app-debug.apk (apktool) и публикация в releases.
set -euo pipefail
cd "$(dirname "$0")/.."

SRC_APK="${1:-app-debug.apk}"
DECODE_DIR="${DECODE_DIR:-/tmp/lctr_build}"
KEYSTORE="${KEYSTORE:-/tmp/lctr_debug.keystore}"
OUT_APK="/tmp/locator-unsigned.apk"
SIGNED_APK="/tmp/locator-signed.apk"

if [[ ! -f "$SRC_APK" ]]; then
  echo "APK не найден: $SRC_APK" >&2
  exit 1
fi

command -v apktool >/dev/null || { echo "apktool не установлен" >&2; exit 1; }
command -v zipalign >/dev/null || apt-get install -y -qq zipalign >/dev/null
command -v apksigner >/dev/null || apt-get install -y -qq apksigner >/dev/null

rm -rf "$DECODE_DIR"
apktool d -f -o "$DECODE_DIR" "$SRC_APK"

# Версия выше 1.0.5 на устройствах — для OTA
python3 - <<'PY' "$DECODE_DIR/apktool.yml"
import sys, re
path = sys.argv[1]
text = open(path).read()
text = re.sub(r"versionCode: '\d+'", "versionCode: '8'", text)
text = re.sub(r"versionName: [^\n]+", "versionName: 1.0.6", text)
open(path, 'w').write(text)
PY

apktool b "$DECODE_DIR" -o "$OUT_APK"

if [[ ! -f "$KEYSTORE" ]]; then
  keytool -genkey -v -keystore "$KEYSTORE" -storepass android -alias androiddebugkey \
    -keypass android -keyalg RSA -keysize 2048 -validity 10000 \
    -dname "CN=Locator Debug,O=Locator,C=BY" 2>/dev/null
fi

zipalign -f 4 "$OUT_APK" "${OUT_APK%.apk}-aligned.apk"
mv "${OUT_APK%.apk}-aligned.apk" "$OUT_APK"

apksigner sign --ks "$KEYSTORE" --ks-pass pass:android --key-pass pass:android \
  --out "$SIGNED_APK" "$OUT_APK"

echo "Собран: $SIGNED_APK"
export BASE_URL="${BASE_URL:-http://localhost:8080}"
./scripts/publish_apk_release.sh "$SIGNED_APK" "1.0.6: captured_at API, timestamp→captured_at на сервере"
