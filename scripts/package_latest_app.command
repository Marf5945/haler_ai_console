#!/bin/zsh
set -euo pipefail

SCRIPT_DIR="${0:A:h}"
PROJECT="${SCRIPT_DIR:h}"
export PATH="$PATH:/usr/local/go/bin:/opt/homebrew/bin:$HOME/go/bin"
APP="$PROJECT/build/bin/HaLer AI Console.app"
LOG_DIR="$PROJECT/build/logs"
LOG_FILE="$LOG_DIR/package-latest-$(date '+%Y%m%d-%H%M%S').log"
BUILD_HELPER="$SCRIPT_DIR/wails_build_darwin.sh"

mkdir -p "$LOG_DIR"
exec > >(tee "$LOG_FILE") 2>&1

finish() {
  local code=$?
  echo
  if [[ $code -eq 0 ]]; then
    echo "Done. Latest app:"
    echo "$APP"
    echo
    echo "Log:"
    echo "$LOG_FILE"
  else
    echo "Build failed with exit code $code."
    echo "Log:"
    echo "$LOG_FILE"
  fi
  echo
  echo "Press Enter to close this window."
  read -r _
  exit $code
}
trap finish EXIT

echo "Packaging HaLer AI Console"
echo "Project: $PROJECT"
echo "Started: $(date '+%Y-%m-%d %H:%M:%S')"
echo

if [[ ! -d "$PROJECT" ]]; then
  echo "Project folder not found: $PROJECT"
  exit 1
fi

if [[ ! -f "$BUILD_HELPER" ]]; then
  echo "Build helper not found: $BUILD_HELPER"
  exit 1
fi

cd "$PROJECT"

echo "Step 1/2: dependency checks and package build"
bash "$BUILD_HELPER"
echo

if [[ ! -d "$APP" ]]; then
  echo "Build completed but app bundle was not found: $APP"
  exit 1
fi

echo "Step 2/2: reopening exact packaged app"
echo "Closing running HaLer AI Console windows first to avoid showing an older in-memory app."
osascript -e 'tell application "HaLer AI Console" to quit' >/dev/null 2>&1 || true
sleep 1
open -n "$APP"
echo "Finished: $(date '+%Y-%m-%d %H:%M:%S')"
