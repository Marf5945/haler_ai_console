#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

find . -type f -name '*_test.go' -not -path './.git/*' -delete
rm -rf frontend/src/test adapter/visual_learning/testdata frontend/node_modules
rm -f assets/models/__dummy_test__ test_dialogues_1000chars_10sets.md
rm -f adapter/visual_learning/YOLOX_MIGRATION.md build/darwin/Info.dev.plist
rm -f scripts/vision_organ_smoke.sh
find . -type d -empty -not -path './.git/*' -delete

scan_common=(--hidden --glob '!**/.git/**' --glob '!**/node_modules/**' --glob '!frontend/package-lock.json')

fail=0
check_zero() {
  local label="$1"
  local pattern="$2"
  if rg -n "${scan_common[@]}" "$pattern" . >/tmp/haler_clean_scan.txt; then
    echo "FAIL: $label"
    sed -n '1,120p' /tmp/haler_clean_scan.txt
    fail=1
  else
    echo "OK: $label"
  fi
}

monitor_port='487''65'
monitor_pattern='127[.]0[.]0[.]1[:]'"$monitor_port"'|localhost[:]'"$monitor_port"'|'"$monitor_port"
posix_home='/User''s/'
windows_home='C:\\User''s\\'
desktop_trace='Desk''top/ui_''console'
desktop_dir='Desk''top[/\\]'
project_trace='ui_console_wa''ils'
path_pattern="$posix_home|$windows_home|$desktop_trace|$desktop_dir|$project_trace"
secret_pattern='(^|[^A-Za-z0-9_-])sk-(proj-)?[A-Za-z0-9_-]{32,}|ghp_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,}|xox[baprs]-[A-Za-z0-9-]{10,}|AKIA[0-9A-Z]{16}'

check_zero "monitor URL" "$monitor_pattern"
check_zero "device-local paths" "$path_pattern"
check_zero "raw secret patterns" "$secret_pattern"

if find . -type f \( -name '*_test.go' -o -path './frontend/src/test/*' -o -path '*/testdata/*' -o -name '__dummy_test__' -o -name 'test_dialogues*' \) -not -path './.git/*' | grep -q .; then
  echo "FAIL: test files remain"
  find . -type f \( -name '*_test.go' -o -path './frontend/src/test/*' -o -path '*/testdata/*' -o -name '__dummy_test__' -o -name 'test_dialogues*' \) -not -path './.git/*' | sed -n '1,120p'
  fail=1
else
  echo "OK: test files removed"
fi

exit "$fail"
