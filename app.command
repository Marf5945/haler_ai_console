#!/bin/zsh
set -euo pipefail

ROOT="${0:A:h}"
APP_NAME="HaLer AI Console"
APP_PATH="$ROOT/build/bin/${APP_NAME}.app"

if [[ ! -d "$APP_PATH" ]]; then
  echo "[ERROR] App bundle not found."
  echo "Expected: $APP_PATH"
  echo
  echo "Run ./install_macos.sh first, then run app.command again."
  echo
  echo "Press Enter to close this window."
  read -r _
  exit 1
fi

open -n "$APP_PATH"
