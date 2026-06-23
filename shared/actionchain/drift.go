package actionchain

import "strings"

// ──────────────────────────────────────────────
// Drift evaluator（rule 5）：偏離判斷看 parsed step 的槽位，不看 raw 字串。
// 重點不是「格式錯」，而是「workflow drift」——尤其「資訊不足卻跳網路」。
// ──────────────────────────────────────────────

const (
	DriftActionMismatch        = "action_mismatch"
	DriftNextMismatch          = "next_mismatch"
	DriftInsufficientToNetwork = "insufficient_to_network"

	DriftDecisionReview  = "review"
	DriftDecisionAskUser = "ask_user"
)

// Drift 是一筆偏離記錄（供 Gate/Controller 決定 review / 提問）。
type Drift struct {
	Index       int    `json:"index"`
	Type        string `json:"type"`
	Reason      string `json:"reason"`
	Decision    string `json:"decision"`
	MissingSlot string `json:"missing_slot,omitempty"`
}

// isNetworkNext 判斷 next 是否是「接下來要上網執行」。
func isNetworkNext(next string) bool {
	return NormalizeAction(strings.TrimSpace(next)) == "網路"
}

// isInfoGatheringNext 判斷 expected 的 next 是否屬於「先收集資訊 / 問使用者」。
func isInfoGatheringNext(next string) bool {
	switch NormalizeAction(strings.TrimSpace(next)) {
	case "輸出", "提問", StandbyNext:
		return true
	}
	return false
}

// DetectDrift 逐 index 比對 expected_chain 與 actual，只看 parsed 槽位。
// 「資訊不足卻跳網路」標成 insufficient_to_network + ask_user，並推出缺少的槽位。
func DetectDrift(expected, actual []ActionChain) []Drift {
	var drifts []Drift
	n := len(expected)
	if len(actual) < n {
		n = len(actual)
	}
	for i := 0; i < n; i++ {
		e, a := expected[i], actual[i]
		if NormalizeAction(a.Action) != NormalizeAction(e.Action) {
			drifts = append(drifts, Drift{
				Index: i, Type: DriftActionMismatch,
				Reason:   "action changed from " + e.Action + " to " + a.Action,
				Decision: DriftDecisionReview,
			})
			continue
		}
		if NormalizeNext(a.Next) == NormalizeNext(e.Next) {
			continue // 同槽位，無偏離
		}
		if isNetworkNext(a.Next) && isInfoGatheringNext(e.Next) {
			drifts = append(drifts, Drift{
				Index: i, Type: DriftInsufficientToNetwork,
				Reason:      "jumped to 網路 before gathering expected info",
				Decision:    DriftDecisionAskUser,
				MissingSlot: inferMissingSlot(expected, i),
			})
			continue
		}
		drifts = append(drifts, Drift{
			Index: i, Type: DriftNextMismatch,
			Reason:   "next changed from " + e.Next + " to " + a.Next,
			Decision: DriftDecisionReview,
		})
	}
	return drifts
}

// inferMissingSlot 從 expected 後續「請問X」步驟推出缺少的槽位名 X。
func inferMissingSlot(expected []ActionChain, from int) string {
	for j := from; j < len(expected); j++ {
		t := strings.TrimSpace(expected[j].Target)
		if strings.HasPrefix(t, "請問") {
			return strings.TrimSpace(strings.TrimPrefix(t, "請問"))
		}
	}
	return ""
}
