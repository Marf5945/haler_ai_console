package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// GoalContract 在 plan 階段建立，描述任務目標的不可變骨架。
// Bounded Replan 用它判定「replan 後的 tail 是否仍服務原目標」。
//
// 注意：資料型別刻意放在 dag（與 DAGRun 一起持久化），行為（StillSatisfiedBy /
// OutputSatisfiedBy，需引用 replan 的提案型別）留在 replan，避免 dag→replan 循環。
type GoalContract struct {
	GoalSummary         string   `json:"goal_summary"`
	OutputType          string   `json:"output_type,omitempty"`      // 例：file / items / answer
	OutputPredicate     string   `json:"output_predicate,omitempty"` // 機器可驗的完成條件（選填）
	Scope               string   `json:"scope,omitempty"`            // v1：路徑前綴，限制 tail 目標範圍
	ImmutableConditions []string `json:"immutable_conditions,omitempty"`
}

// IsZero 判斷 contract 是否未建立（legacy run）。
func (c GoalContract) IsZero() bool {
	return c.GoalSummary == "" && c.OutputType == "" && c.Scope == ""
}

// Hash 由 Go 自行計算，作為 contract 身分；不採信模型傳來的 expected_goal_hash。
func (c GoalContract) Hash() string {
	h := sha256.New()
	h.Write([]byte(strings.Join([]string{
		c.GoalSummary, c.OutputType, c.OutputPredicate, c.Scope,
		strings.Join(c.ImmutableConditions, ","),
	}, "\x1f")))
	return hex.EncodeToString(h.Sum(nil))
}

// NewGoalContractFromPlan 從使用者目標與正規化後的 plan 衍生一個 GoalContract。
// v1 先抓住「目標摘要」確保非 legacy；OutputType / Scope / OutputPredicate
// 的更精準推導留待下一輪（沒有時 Gate 會保守地走 review）。
func NewGoalContractFromPlan(goalSummary string, plan TaskPlan) GoalContract {
	return GoalContract{
		GoalSummary: strings.TrimSpace(goalSummary),
	}
}
