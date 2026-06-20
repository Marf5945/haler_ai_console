#!/bin/zsh
set -euo pipefail

SCRIPT_DIR="${0:A:h}"
PROJECT="${AI_CONSOLE_PROJECT:-${SCRIPT_DIR:h}}"
export PATH="$PATH:/usr/local/go/bin:/opt/homebrew/bin:$HOME/go/bin"
APP_NAME="HaLer AI Console"
APP="$PROJECT/build/bin/${APP_NAME}.app"
APP_EXEC="$APP/Contents/MacOS/$APP_NAME"
LOG_DIR="$PROJECT/build/logs"
LOG_FILE="$LOG_DIR/package-latest-$(date '+%Y%m%d-%H%M%S').log"
BUILD_HELPER="$PROJECT/scripts/wails_build_darwin.sh"

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

close_running_app() {
  if [[ ! -x "$APP_EXEC" ]]; then
    return 0
  fi
  local pids=()
  while IFS= read -r pid; do
    [[ -n "$pid" ]] && pids+=("$pid")
  done < <(pgrep -f "$APP_EXEC" 2>/dev/null || true)
  if [[ ${#pids[@]} -eq 0 ]]; then
    return 0
  fi
  echo "Closing ${#pids[@]} running ${APP_NAME} process(es) from this build path."
  kill "${pids[@]}" 2>/dev/null || true
  sleep 1
}

echo "Packaging $APP_NAME"
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

if [[ "${OPEN_AFTER_BUILD:-1}" == "0" || "${OPEN_AFTER_BUILD:-1}" == "false" ]]; then
  echo "Step 2/2: skipping app reopen because OPEN_AFTER_BUILD=${OPEN_AFTER_BUILD:-1}."
  echo "Finished: $(date '+%Y-%m-%d %H:%M:%S')"
  exit 0
fi

echo "Step 2/2: reopening exact packaged app"
echo "Closing running $APP_NAME from the exact build path to avoid showing an older in-memory app."
close_running_app
open -n "$APP"
echo "Finished: $(date '+%Y-%m-%d %H:%M:%S')"
