#!/usr/bin/env bash
# 總驗收腳本：格式 / vet / build / test，外加本批修正的行為檢查。
# 用法（從任何地方都可執行）： ./verify.sh   或   bash verify.sh
set -uo pipefail

# 專案根目錄：以此腳本位置解析，避免寫死個人路徑。
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR" || { echo "❌ 找不到專案目錄：$PROJECT_DIR"; exit 1; }

# 確保 go 在 PATH（macOS 常見安裝位置：官方 pkg / Homebrew / GOPATH bin）。
export PATH="$PATH:/usr/local/go/bin:/opt/homebrew/bin:$HOME/go/bin"
if ! command -v go >/dev/null 2>&1; then
  echo "❌ 找不到 go 指令。請確認已安裝 Go，或把它的 bin 目錄加進 PATH。"
  echo "   目前 PATH：$PATH"
  exit 1
fi

export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/go-build-cache-fresh}"
echo "專案：$PROJECT_DIR"
echo "go： $(command -v go)  ($(go version 2>/dev/null))"

pass=0; fail=0
ok(){ echo "  ✅ $1"; pass=$((pass+1)); }
no(){ echo "  ❌ $1"; fail=$((fail+1)); }
hr(){ echo; echo "▍$1"; }

hr "1/5 gofmt（格式）"
if go fmt ./... 2>&1 | sed 's/^/    /'; then ok "格式化完成"; else no "gofmt 失敗"; fi

hr "2/5 go vet"
# unsafeptr 例外：adapter/visual_learning/*_windows.go 與 native_drag_windows.go 的
# Windows COM 回呼／syscall 以 uintptr 承接 OS 指標再轉回 unsafe.Pointer（this/riid/
# lParam/ppv/ptr 及兩處指標算術）。這是正確且必要的 OS 互操作寫法——指標來自
# GlobalAlloc／COM 非受管記憶體，GC 不會搬移；盲目改寫的風險遠高於警告本身。
# 故僅關閉 unsafeptr 這單一分析器，其餘 vet 檢查（printf／結構標籤／lostcancel…）全保留。
if go vet -unsafeptr=false ./... 2>&1 | sed 's/^/    /'; then ok "vet 乾淨（unsafeptr 例外：Windows interop）"; else no "vet 有問題"; fi

hr "3/5 go build ./..."
if go build ./... 2>&1 | sed 's/^/    /'; then ok "build 成功"; else no "build 失敗"; fi

hr "4/5 go test ./..."
if go test ./... 2>&1 | sed 's/^/    /'; then ok "全部測試通過"; else no "有測試失敗"; fi

hr "5/5 行為錨點（grep 確認本批修正在位）"
chk(){ if grep -q "$2" "$1" 2>/dev/null; then ok "$3"; else no "$3（找不到 $2）"; fi; }
# 反向錨點：驗證某符號「已被移除」仍在位（找到＝反而失敗）。
chkabsent(){ if grep -q "$2" "$1" 2>/dev/null; then no "$3（$2 仍在，應已移除）"; else ok "$3"; fi; }
# 這批把關鍵字驅動的「缺地點就問」臆測（containsLocationHint / isContextSensitiveWebQuery）
# 整個廢除，澄清改由 judge 決定（見 tool_readiness.go 檔首說明）。故這裡驗證的是
# 「該臆測已不存在」，而非舊符號在位——舊錨點 'containsLocationHint(userText)' 已過時。
chkabsent tool_readiness.go 'func containsLocationHint'               "地點誤問臆測已移除，澄清交給 judge"
chk shared/websearch/service.go 'func queryLanguage'                  "語言感知排序：queryLanguage 在位"
chk shared/websearch/service.go 'rankByAuthority(req.Query'           "語言感知排序：Search 帶 query"
chk shared/controlseal/seal.go 'for attempt := 0'                     "controlseal：rand 重試退化"
chk replan_binding.go 'buildReplanRepairPrompt'                       "replan：修復重試在位"

echo
echo "════════════════════════════════"
echo "通過 $pass 項，失敗 $fail 項"
[ "$fail" -eq 0 ] && echo "🎉 全綠" || echo "⚠️  有項目要看上面 ❌"
exit "$fail"
