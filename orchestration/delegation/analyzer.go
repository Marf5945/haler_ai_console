// delegation/analyzer.go — 動作統計分析器。
// 會話結束後統計主代理直接動作比例，判斷是否應建立新 sub（閾值 40%）。
package delegation

// ──────────────────────────────────────────────
// 常數定義
// ──────────────────────────────────────────────

// SubCreationThreshold 直接動作比例門檻，超過時建議建立新 sub。
const SubCreationThreshold = 0.40

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// ActionRecord 單筆動作記錄。
// Type：sub_delegated（委派給 sub）或 main_direct（主代理直接執行）。
type ActionRecord struct {
	Type      string `json:"type"`      // "sub_delegated" | "main_direct"
	ToolID    string `json:"tool_id"`   // 使用的工具 ID
	Timestamp string `json:"timestamp"` // RFC3339 時間戳
}

// AnalysisResult 動作統計分析結果。
type AnalysisResult struct {
	TotalActions        int     `json:"total_actions"`         // 總動作數
	MainDirectActions   int     `json:"main_direct_actions"`   // 主代理直接執行數
	SubDelegatedActions int     `json:"sub_delegated_actions"` // 委派給 sub 的動作數
	DirectRatio         float64 `json:"direct_ratio"`          // 直接執行比例
	ShouldCreateSub     bool    `json:"should_create_sub"`     // 是否建議建立新 sub
}

// ──────────────────────────────────────────────
// 分析函式
// ──────────────────────────────────────────────

// Analyze 統計動作記錄，計算直接執行比例並判斷是否建立新 sub。
// DirectRatio = MainDirectActions / TotalActions（空記錄時為 0）。
// ShouldCreateSub = DirectRatio >= SubCreationThreshold。
func Analyze(records []ActionRecord) AnalysisResult {
	if len(records) == 0 {
		return AnalysisResult{}
	}

	var mainDirect, subDelegated int
	for _, r := range records {
		switch r.Type {
		case "main_direct":
			mainDirect++
		case "sub_delegated":
			subDelegated++
		}
	}

	total := mainDirect + subDelegated
	if total == 0 {
		return AnalysisResult{TotalActions: len(records)}
	}

	ratio := float64(mainDirect) / float64(total)

	return AnalysisResult{
		TotalActions:        total,
		MainDirectActions:   mainDirect,
		SubDelegatedActions: subDelegated,
		DirectRatio:         ratio,
		ShouldCreateSub:     ratio >= SubCreationThreshold,
	}
}
