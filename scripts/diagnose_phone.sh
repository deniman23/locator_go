#!/usr/bin/env bash
# Диагностика Locator на телефоне (Device Owner / OTA / LocationService).
# Запуск на ПК, где: adb devices показывает device.
set -euo pipefail

SERIAL="${1:-}"
PKG=com.example.lctr_app
ADB=(adb)
if [[ -n "$SERIAL" ]]; then
  ADB=(adb -s "$SERIAL")
fi

echo "=== adb devices ==="
adb devices -l

if ! "${ADB[@]}" get-state >/dev/null 2>&1; then
  echo "Укажите serial: $0 R9WX202AD5H"
  exit 1
fi

echo
echo "=== устройство ==="
"${ADB[@]}" shell getprop ro.product.model
"${ADB[@]}" shell getprop ro.build.version.release

echo
echo "=== версия Locator ==="
"${ADB[@]}" shell dumpsys package "$PKG" 2>/dev/null | grep -E 'versionName=|versionCode=' | head -2

echo
echo "=== Device Owner / profile owner ==="
"${ADB[@]}" shell dumpsys device_policy 2>/dev/null | grep -E 'Device Owner|Profile Owner|com.example.lctr_app|lctr' | head -20 || true
"${ADB[@]}" shell dpm list-owners 2>/dev/null || true

echo
echo "=== LocationService / AppUpdate (dumpsys services) ==="
"${ADB[@]}" shell dumpsys activity services "$PKG" 2>/dev/null \
  | grep -E 'ServiceRecord|LocationService|AppUpdate|isForeground|startRequested|createTime|lastActivity' \
  | head -30 || echo "(сервисы не найдены — фон мёртв)"

echo
echo "=== разрешения геолокации ==="
"${ADB[@]}" shell dumpsys package "$PKG" 2>/dev/null \
  | grep -A1 'android.permission.ACCESS_.*LOCATION' | head -10

echo
echo "=== battery / оптимизация (если есть) ==="
"${ADB[@]}" shell dumpsys deviceidle 2>/dev/null | grep -i "$PKG" | head -5 || true

echo
echo "=== SharedPreferences (OTA state), run-as ==="
"${ADB[@]}" shell "run-as $PKG sh -c 'ls shared_prefs 2>/dev/null; for f in shared_prefs/*.xml; do echo --- \$f ---; cat \"\$f\"; done'" 2>/dev/null \
  | grep -iE 'app_update|update|pending|sha256|service|poll|api_key' | head -40 \
  || echo "(run-as недоступен — debug-сборка или откройте приложение)"

echo
echo "=== logcat (последние 500 строк, фильтр Locator) ==="
"${ADB[@]}" logcat -d -t 500 2>/dev/null \
  | grep -iE 'LocationService|AppUpdate|InstallResult|DeviceOwner|LocatorHttp|command ack|FATAL EXCEPTION|AndroidRuntime' \
  | tail -80 || echo "(пусто)"

echo
echo "=== принудительный старт LocationService (без OTA) ==="
"${ADB[@]}" shell am startservice -n "$PKG/.LocationService" 2>&1 || true
sleep 3
"${ADB[@]}" shell dumpsys activity services "$PKG" 2>/dev/null | grep LocationService | head -5 || true

echo
echo "Готово. Скопируйте весь вывод в чат."
