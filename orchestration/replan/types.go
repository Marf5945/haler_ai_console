package replan

import (
	"sort"
	"strings"
)

// ──────────────────────────────────────────────
// 列舉
// ──────────────────────────────────────────────

// Decision 是 Gate 的裁決結果。
type Decision string

const (
	DecisionSilent Decision = "silent" // 全 low、同目標：自動套用，不打擾使用者
	DecisionReview Decision = "review" // 整條新 tail 進 review，交人核准
	DecisionStop   Decision = "stop"   // 連續無進展達上限或踩到 critical：停下走 recovery
)

// Stage 驅動 status rail 的升級顯示（越接近上限越明顯）。
type Stage string

const (
	StageSilentNotice Stage = "silent_notice" // 第 1-2 次：低調一行
	StageAdjusting    Stage = "adjusting"     // 第 3-4 次：顯示「正在調整路線 (n/5)」
	StageStop         Stage = "stop"          // 第 5 次：停下交人
)

// FailureCategory 是執行層轉出的結構化失敗類型。
// replan policy 只吃這個 enum，不靠自然語言猜測。
type FailureCategory string

const (
	FailureNoResults     FailureCategory = "no_results"     // 搜尋/查找無命中
	FailurePathNotFound  FailureCategory = "path_not_found" // 路徑不存在
	FailureTruncated     FailureCategory = "truncated"      // 結果被截斷、資訊不足
	FailureAmbiguous     FailureCategory = "ambiguous"      // 結果模糊、需換策略
	FailureOutsideScope  FailureCategory = "outside_scope"  // 超出 GoalContract 範圍（全域 deny）
	FailureSensitivePath FailureCategory = "sensitive_path" // 觸及敏感路徑（全域 deny）
)

// globallyDenied 是永遠不可 silent 的失敗類型，一律 review。
var globallyDenied = map[FailureCategory]bool{
	FailureOutsideScope:  true,
	FailureSensitivePath: true,
}

// IsGloballyDenied 回報該失敗類型是否禁止 silent replan。
func IsGloballyDenied(c FailureCategory) bool { return globallyDenied[c] }

// ProposalIntent 是模型對自身提案的「宣稱」——只能用來收緊（宣稱改範圍→review），
// 不能用來放寬（宣稱同目標仍要 Go 自行驗證）。
type ProposalIntent string

const (
	IntentSameGoalPath ProposalIntent = "same_goal_path_adjustment" // 同目標換路
	IntentScopeChange  ProposalIntent = "scope_change"              // 改變目標/產出
)

// ──────────────────────────────────────────────
// read-only allowlist（v1 硬天花板，編譯在 Go）
// ──────────────────────────────────────────────

// eligibleReadOnlyActions 是 v1 允許 silent replan 的唯一動作集合。
// 不放外部 manifest：新增工具必須改這裡並走 build/review，攻擊面歸零。
var eligibleReadOnlyActions = map[string]bool{
	// 真實註冊的本機唯讀動作詞（對齊 actionTags；網路不在此=一律 review）
	"搜尋":   true, // 找本機檔案
	"本機搜尋": true,
	"查找":   true,
	"查看":   true, // 看檔案
	"讀取":   true,
	"列出":   true, // 列出目錄
	// 註：查詢=查已保存操作目錄（非檔案），預設不 silent、走 review，故不列於此
	// 英文工具名（過渡期保留，供既有測試與舊路徑）
	"grep_search":    true,
	"glob":           true,
	"list_directory": true,
	"read_file":      true,
	"read":           true,
}

// IsReadOnlyEligible 回報動作是否在 v1 silent allowlist 內。
func IsReadOnlyEligible(action string) bool {
	return eligibleReadOnlyActions[strings.ToLower(strings.TrimSpace(action))]
}

// ──────────────────────────────────────────────
// 模型提案結構
// ──────────────────────────────────────────────

// ProposedNode 是模型提議的單一新 tail 節點。
type ProposedNode struct {
	Title          string   `json:"title,omitempty"`
	Action         string   `json:"action"`
	Target         string   `json:"target,omitempty"`
	ModelRiskClass string   `json:"model_risk_class,omitempty"` // 模型自報，僅作提示
	Dependencies   []string `json:"dependencies,omitempty"`
}

// ReplanProposal 是 LLM Proposer 的結構化輸出。
// same_goal / expected_goal_hash / confidence / intent 都只是「說明」，不是「證明」。
type ReplanProposal struct {
	Reason           string         `json:"reason"`
	Intent           ProposalIntent `json:"intent"`
	ProposedTail     []ProposedNode `json:"proposed_tail"`
	ExpectedGoalHash string         `json:"expected_goal_hash,omitempty"` // 模型主張，Go 不採信
	Confidence       float64        `json:"confidence,omitempty"`         // 提示，只能往保守傾
}

// EligibleReplanActions 回傳 v1 silent allowlist 的動作清單（排序穩定）。
// 提供給 Proposer，讓模型只提 eligible 工具、減少必被否決的廢提案。
func EligibleReplanActions() []string {
	out := make([]string, 0, len(eligibleReadOnlyActions))
	for a := range eligibleReadOnlyActions {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
