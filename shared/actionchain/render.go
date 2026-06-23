package actionchain

import "strings"

// ──────────────────────────────────────────────
// Renderer：wire format 對內、人類語句對外。
// 規則：actionchain 只給程式看；使用者只看 renderer 合成後的人類句。
//   - 未知動作 fail-safe，絕不回退顯示 raw（避免 ㄌ/wire format 外洩、被模仿）。
//   - target 當純文字、截斷後加引號顯示，不二次解析、不執行。
// ──────────────────────────────────────────────

// Phase 區分「準備執行」與「執行中」，用於是否顯示中斷點與省略號。
type Phase string

const (
	PhasePending Phase = "pending" // 送出前：準備…
	PhaseRunning Phase = "running" // 執行中：正在…
)

// statusDisplayMaxRunes 是狀態句裡 target 的顯示截斷長度。
const statusDisplayMaxRunes = 40

// verbPhrase 把動作詞映成人類動詞片語；未知 → 通用安全片語。
func verbPhrase(action string) string {
	switch NormalizeAction(strings.TrimSpace(action)) {
	case "查詢", "搜尋", "查找", "本機搜尋":
		return "搜尋本機資料"
	case "讀取", "查看", "閱讀":
		return "查閱資料"
	case "列出", "列表":
		return "列出項目"
	case "網路":
		return "用網路搜尋"
	case "程式":
		return "準備製作小程式"
	default:
		return "處理" // fail-safe：未知動作不洩 raw
	}
}

// HumanStatus 把一個動作合成使用者可讀的狀態句。
// 例：HumanStatus("網路","甜點食譜",PhaseRunning) → 正在用網路搜尋「甜點食譜」…
func HumanStatus(action, target string, phase Phase) string {
	vp := verbPhrase(action)
	shown := truncateForDisplay(strings.TrimSpace(target))
	var b strings.Builder
	if phase == PhasePending {
		b.WriteString("準備")
	} else {
		b.WriteString("正在")
	}
	b.WriteString(vp)
	if shown != "" {
		b.WriteString("「")
		b.WriteString(shown)
		b.WriteString("」")
	}
	if phase == PhaseRunning {
		b.WriteString("…")
	}
	return b.String()
}

// HumanStatusChain 是 ActionChain 版本的便利包裝。
func HumanStatusChain(chain ActionChain, phase Phase) string {
	return HumanStatus(chain.Action, chain.Target, phase)
}

// NoProposalStatus 是「找不到」的人類句。
func NoProposalStatus() string {
	return "找不到可用的新查找方式。"
}

// truncateForDisplay 截斷 target 顯示長度（rune 計），超長加省略號。
func truncateForDisplay(s string) string {
	runes := []rune(s)
	if len(runes) <= statusDisplayMaxRunes {
		return s
	}
	return string(runes[:statusDisplayMaxRunes]) + "…"
}
