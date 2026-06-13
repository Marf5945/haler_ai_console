#!/usr/bin/env bash
set -euo pipefail

# Keep: helper for finding the current debug trace monitor URL. A stale link
# means the app is stopped, not that the monitor/debugtrace code is unused.
APP_DATA="${HOME}/Library/Application Support/ai-console"
LINK_TXT="${APP_DATA}/debug/monitor_link.txt"
LINK_JSON="${APP_DATA}/debug/monitor_link.json"

if [[ -s "$LINK_TXT" ]]; then
  cat "$LINK_TXT"
  exit 0
fi

if [[ -s "$LINK_JSON" ]]; then
  sed -n 's/.*"url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$LINK_JSON" | head -n 1
  exit 0
fi

cat >&2 <<MSG
監視頁連結尚未建立。
請先啟動新版 AI Console，或用：
  cd <專案目錄> && wails dev
MSG
exit 1
