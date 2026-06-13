#!/bin/zsh
set -euo pipefail

SCRIPT_DIR="${0:A:h}"
PROJECT="$(cd "$SCRIPT_DIR/.." && pwd -P)"
WAILS="${WAILS:-$(command -v wails || true)}"
if [[ -z "$WAILS" && -x "$HOME/go/bin/wails" ]]; then
  WAILS="$HOME/go/bin/wails"
fi
APP="$PROJECT/build/bin/ai-console.app"
LOG_DIR="$PROJECT/build/logs"
LOG_FILE="$LOG_DIR/package-latest-$(date '+%Y%m%d-%H%M%S').log"

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

echo "Packaging AI Console"
echo "Project: $PROJECT"
echo "Started: $(date '+%Y-%m-%d %H:%M:%S')"
echo

if [[ ! -d "$PROJECT" ]]; then
  echo "Project folder not found: $PROJECT"
  exit 1
fi

if [[ -z "$WAILS" || ! -x "$WAILS" ]]; then
  echo "Wails executable not found or not executable."
  echo "Install Wails, add it to PATH, or run with WAILS=/path/to/wails."
  exit 1
fi

cd "$PROJECT"

echo "Step 1/3: frontend build"
(cd frontend && npm run build)
echo

echo "Step 2/3: Wails package build"
"$WAILS" build
echo

if [[ ! -d "$APP" ]]; then
  echo "Build completed but app bundle was not found: $APP"
  exit 1
fi

echo "Step 3/3: reopening exact packaged app"
echo "Closing running ai-console windows first to avoid showing an older in-memory app."
osascript -e 'tell application id "com.wails.ai-console" to quit' >/dev/null 2>&1 || true
sleep 1
open -n "$APP"
echo "Finished: $(date '+%Y-%m-%d %H:%M:%S')"
