#!/usr/bin/env bash
# 總驗收腳本：格式 / vet / build / test，外加本批修正的行為檢查。
# 用法（從任何地方都可執行）： ./verify.sh   或   bash verify.sh
set -uo pipefail

# 專案根目錄：由此腳本所在位置推導，雙擊或從別處執行都不會跑錯位置。
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
PROJECT_DIR="$SCRIPT_DIR"
cd "$PROJECT_DIR" || { echo "❌ 找不到專案目錄：$PROJECT_DIR"; exit 1; }

# 確保 go 在 PATH（macOS 常見安裝位置：官方 pkg / Homebrew / GOPATH bin）。
export PATH="$PATH:/usr/local/go/bin:/opt/homebrew/bin:$HOME/go/bin"
if ! command -v go >/dev/null 2>&1; then
  echo "❌ 找不到 go 指令。請確認已安裝 Go，或把它的 bin 目錄加進 PATH。"
  echo "   目前 PATH：$PATH"
  exit 1
fi

export GOCACHE="${GOCACHE:-/private/tmp/go-build-cache-fresh}"
echo "專案：$PROJECT_DIR"
echo "go： $(command -v go)  ($(go version 2>/dev/null))"

pass=0; fail=0
ok(){ echo "  ✅ $1"; pass=$((pass+1)); }
no(){ echo "  ❌ $1"; fail=$((fail+1)); }
hr(){ echo; echo "▍$1"; }

hr "1/5 gofmt（格式）"
if go fmt ./... 2>&1 | sed 's/^/    /'; then ok "格式化完成"; else no "gofmt 失敗"; fi

hr "2/5 go vet"
if go vet ./... 2>&1 | sed 's/^/    /'; then ok "vet 乾淨"; else no "vet 有問題"; fi

hr "3/5 go build ./..."
if go build ./... 2>&1 | sed 's/^/    /'; then ok "build 成功"; else no "build 失敗"; fi

hr "4/5 go test ./..."
if go test ./... 2>&1 | sed 's/^/    /'; then ok "全部測試通過"; else no "有測試失敗"; fi

hr "5/5 行為錨點（grep 確認本批修正在位）"
chk(){ if grep -q "$2" "$1" 2>/dev/null; then ok "$3"; else no "$3（找不到 $2）"; fi; }
chk tool_readiness.go 'containsLocationHint(userText)'                "地點誤問：readiness 看原始訊息"
chk shared/websearch/service.go 'func queryLanguage'                  "語言感知排序：queryLanguage 在位"
chk shared/websearch/service.go 'rankByAuthority(req.Query'           "語言感知排序：Search 帶 query"
chk shared/controlseal/seal.go 'for attempt := 0'                     "controlseal：rand 重試退化"
chk replan_binding.go 'buildReplanRepairPrompt'                       "replan：修復重試在位"

echo
echo "════════════════════════════════"
echo "通過 $pass 項，失敗 $fail 項"
[ "$fail" -eq 0 ] && echo "🎉 全綠" || echo "⚠️  有項目要看上面 ❌"
exit "$fail"
