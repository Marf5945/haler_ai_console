package hookgene

import "math/rand"

// CandidateState 是突變 candidate 的狀態（§3.1.5.18.6）。
type CandidateState string

const (
	StateStaged        CandidateState = "STAGED"
	StatePendingReview CandidateState = "PENDING_REVIEW"
	StateDormant       CandidateState = "DORMANT"
	StateActive        CandidateState = "ACTIVE" // 型別存在；MVP 不允許自動轉入
)

// allowedTransitions 是合法狀態轉移白名單（§3.1.5.18.6）。
// 任何 → ACTIVE 一律禁止（MVP 不自動啟用）；被拒 candidate → DORMANT，
// DORMANT 只能回 PENDING_REVIEW 重新審核（環境改變時可重新提案，但不自動復活）。
var allowedTransitions = map[CandidateState]map[CandidateState]bool{
	StateStaged:        {StatePendingReview: true, StateDormant: true},
	StatePendingReview: {StateDormant: true},
	StateDormant:       {StatePendingReview: true},
}

// CanTransition 回傳 from→to 是否為合法轉移。
func CanTransition(from, to CandidateState) bool {
	return allowedTransitions[from][to]
}

// Candidate 是突變產生的 skill 候選。突變只動 candidate，不修改原 skill。
type Candidate struct {
	SourceSkillID string         // 來源 skill（保留原樣，不被修改）
	Gene          []HookCode     // 被突變的 gene
	State         CandidateState // 只能 STAGED / PENDING_REVIEW / DORMANT
	// NewTools：相對原 skill 新增的工具。review diff 必須顯示，且每個工具仍須走自身 risk gate
	// （§3.1.5.18.6 / H-10）。本套件不執行工具，只攜帶資訊供上層 gate 與 diff 使用。
	NewTools []string
}

// CopyForMutation 複製原 gene 成新 candidate（保留原 skill，狀態起始為 STAGED）。
func CopyForMutation(sourceSkillID string, gene []HookCode) Candidate {
	return Candidate{
		SourceSkillID: sourceSkillID,
		Gene:          append([]HookCode(nil), gene...),
		State:         StateStaged,
	}
}

// MutateGene 隨機改變 candidate 其中一格 hook（§3.1.5.18.6）。
// 只改 gene；填入合理動作與評估由上層在 Learning Mode 進行，且評估涉及真實副作用時
// 必須走一般 replay gate + 使用者確認（見 guard.go 與 §3.1.5.18.6）。
func MutateGene(c *Candidate, r *rand.Rand) {
	if len(c.Gene) == 0 {
		return
	}
	alphabet := []HookCode{HookList, HookInput, HookOutput, HookStandby}
	idx := r.Intn(len(c.Gene))
	c.Gene[idx] = alphabet[r.Intn(len(alphabet))]
}
