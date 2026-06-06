// drift.go — drift 判定（TASK 31 / Phase 1.4 低階；Phase 1.5 補 expected 比對）。
// 不變式 3：actual 必須來自「raw parse 後的 EvalStep」，不可用 sanitized summary。
// 不變式：suspected_injection 直接吃 controlseal 的結果，不自建偵測（避免兩套打架）。
package skill_eval

import "ui_console/shared/controlseal"

// DriftKind 列舉至少包含的 drift 類型。
type DriftKind string

const (
	DriftExpectedNextMismatch DriftKind = "expected_next_mismatch" // Phase 1.5（需 expected）
	DriftActionMismatch       DriftKind = "action_mismatch"        // Phase 1.5（需 expected）
	DriftTargetMismatch       DriftKind = "target_mismatch"        // Phase 1.5（需 expected）
	DriftExecutorMismatch     DriftKind = "executor_mismatch"      // Phase 1.4（不需 expected）
	DriftSuspectedInjection   DriftKind = "suspected_injection"    // Phase 1.4，獨立分類
)

// 風險等級：low 只記錄；high 阻擋/暫停/review。
const (
	RiskLow  = "low"
	RiskHigh = "high"
)

// DriftEvent 是一筆 drift 記錄。
type DriftEvent struct {
	Kind     DriftKind `json:"kind"`
	Risk     string    `json:"risk"`
	Expected EvalStep  `json:"expected,omitempty"`
	Actual   EvalStep  `json:"actual,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Blocked  bool      `json:"blocked"` // high risk → true（呼叫端據此阻擋/送 review）
}

// EvaluateLowLevel 不需要 expected_chain，builtin / 無 expected 的 skill 也能跑（不變式：nil 不壞）。
//   - san：對「同一份 raw LLM 輸出」呼叫 controlseal.SanitizeForLLM 的結果，提供 injection 訊號。
//   - wantExec / gotExec：規劃指定的 executor 與實際使用的 executor。
func EvaluateLowLevel(actual EvalStep, san controlseal.SanitizedResult, wantExec, gotExec string) []DriftEvent {
	var out []DriftEvent

	// suspected_injection：來源是 controlseal，不是自製偵測。
	if san.HasFakeSeal {
		out = append(out, DriftEvent{
			Kind:    DriftSuspectedInjection,
			Risk:    RiskHigh,
			Actual:  actual,
			Reason:  "controlseal 偵測到偽命令印章（fake seal）",
			Blocked: true, // high risk：阻擋/送 review
		})
	}

	// executor_mismatch：規劃與實際 executor 不一致。
	if wantExec != "" && gotExec != "" && wantExec != gotExec {
		out = append(out, DriftEvent{
			Kind:    DriftExecutorMismatch,
			Risk:    RiskLow,
			Actual:  actual,
			Reason:  "規劃 executor=" + wantExec + " 實際=" + gotExec,
			Blocked: false, // low risk：只記錄
		})
	}
	return out
}
