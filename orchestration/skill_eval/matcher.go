// matcher.go — expected_chain 比對與評分（TASK 31 / Phase 1.5）。
// 不變式 4：比對 next 前，expected 與 actual 兩邊都跑 actionchain.NormalizeNext。
package skill_eval

import (
	"fmt"
	"strings"

	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
)

// MuOptionalPenalty 是「ㄇ 替代 OP step」的固定小扣分（不算 drift）。
const MuOptionalPenalty = -0.1

// DefaultMaxSteps 是 16 格規則的預設上限；超過僅低風險提示，不阻擋。
const DefaultMaxSteps = 16

// MatchResult 是一次 expected 比對的彙整結果。
type MatchResult struct {
	Drifts []DriftEvent `json:"drifts"`
	Score  float64      `json:"score"` // 基準 1.0，ㄇ 替代 OP 每次 -0.1
	Hints  []string     `json:"hints"` // 低風險提示（如超過 16 步）
}

// muTypeActions 是 ㄇ 類動作（詢問/等待/確認）；actual 命中代表模型用 ㄇ 取代了某步。
var muTypeActions = map[string]bool{
	"提問": true, "詢問": true, "澄清": true, "選項": true, "等待": true, "確認": true, "待命": true,
}

// classifyActualCode 從 actual 的 action 推測其 step code；目前只需辨識 ㄇ。
func classifyActualCode(action string) string {
	if muTypeActions[strings.TrimSpace(action)] {
		return "ㄇ"
	}
	return ""
}

// EvaluateAgainstExpected 把 actual 步驟序列對齊 expected_chain 比對。
// exp 為 nil（builtin / 無 expected）時回傳空結果，呼叫端應退回 EvaluateLowLevel。
func EvaluateAgainstExpected(actual []EvalStep, exp *skill_step.ExpectedChain) MatchResult {
	res := MatchResult{Score: 1.0}
	if exp == nil {
		return res // 無 expected：不報 mismatch，交給低階 drift
	}

	// 16 格規則：超過僅低風險提示，不阻擋。
	if maxSteps := effectiveMax(exp.MaxSteps); len(exp.Steps) > maxSteps {
		res.Hints = append(res.Hints,
			fmt.Sprintf("expected_chain 共 %d 步，超過建議上限 %d，建議拆分（低風險）", len(exp.Steps), maxSteps))
	}

	for i, want := range exp.Steps {
		wantNext := actionchain.NormalizeNext(firstNonEmpty(want.Next, actionchain.StandbyNext))

		if i >= len(actual) {
			// 缺步：RE 缺步算 drift；OP 缺步不算。
			if want.Requirement == "RE" {
				res.Drifts = append(res.Drifts, DriftEvent{
					Kind: DriftActionMismatch, Risk: RiskHigh, Blocked: true,
					Expected: expToEval(want), Reason: fmt.Sprintf("缺少必要步驟 #%d", i+1),
				})
			}
			continue
		}
		got := actual[i]
		gotNext := actionchain.NormalizeNext(firstNonEmpty(got.Next, actionchain.StandbyNext))

		if want.Requirement == "OP" {
			// ㄇ 替代 OP：合法，不算 drift，扣 0.1。
			if classifyActualCode(got.Action) == "ㄇ" && want.Action != got.Action {
				res.Score += MuOptionalPenalty
			}
			continue // OP step 不對 action/target/next 報 drift
		}

		// 以下為 RE step 的嚴格比對。
		if !eq(want.Action, got.Action) {
			res.Drifts = append(res.Drifts, drift(DriftActionMismatch, want, got, "action 不符"))
		}
		if !eq(want.Target, got.Target) {
			res.Drifts = append(res.Drifts, drift(DriftTargetMismatch, want, got, "target 不符"))
		}
		if wantNext != gotNext {
			res.Drifts = append(res.Drifts, drift(DriftExpectedNextMismatch, want, got,
				fmt.Sprintf("next 不符（期望 %s 得到 %s）", wantNext, gotNext)))
		}
	}
	return res
}

// ValidateDraft 在 LLM 草稿升 pending_skill 前驗證（TASK 31 / Phase 1.5）。
// 回傳問題清單；空 = 通過。超過 16 步只列為提示性問題（呼叫端可選擇不阻擋）。
func ValidateDraft(exp *skill_step.ExpectedChain) []string {
	var problems []string
	if exp == nil || len(exp.Steps) == 0 {
		return []string{"expected_chain 為空"}
	}
	if len(exp.Steps) > effectiveMax(exp.MaxSteps) {
		problems = append(problems, fmt.Sprintf("步數 %d 超過 %d（建議拆分）", len(exp.Steps), effectiveMax(exp.MaxSteps)))
	}
	for i, st := range exp.Steps {
		if strings.TrimSpace(st.Action) == "" || strings.TrimSpace(st.Target) == "" {
			problems = append(problems, fmt.Sprintf("步驟 #%d action/target 不可為空", i+1))
		}
		if !skill_step.IsValidStepCode(st.Code) {
			problems = append(problems, fmt.Sprintf("步驟 #%d code %q 非法（須 ㄅㄔㄇㄖ）", i+1, st.Code))
		}
		if !skill_step.IsValidRequirement(st.Requirement) {
			problems = append(problems, fmt.Sprintf("步驟 #%d requirement %q 非法（須 OP/RE）", i+1, st.Requirement))
		}
		// reserved tag 不應被當成 skill 的可執行步驟動作（提問/輸出等是 Controller 擁有）。
		if actionchain.IsReservedTag(st.Action) {
			problems = append(problems, fmt.Sprintf("步驟 #%d action %q 是 Controller 保留字，不可作為執行步驟", i+1, st.Action))
		}
	}
	return problems
}

// ── 小工具 ──

func effectiveMax(m int) int {
	if m <= 0 {
		return DefaultMaxSteps
	}
	return m
}
func eq(a, b string) bool { return strings.TrimSpace(a) == strings.TrimSpace(b) }
func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
func expToEval(e skill_step.ExpectedStep) EvalStep {
	return EvalStep{Action: e.Action, Target: e.Target,
		Next: actionchain.NormalizeNext(firstNonEmpty(e.Next, actionchain.StandbyNext)),
		Code: e.Code, Requirement: e.Requirement}
}
func drift(kind DriftKind, want skill_step.ExpectedStep, got EvalStep, reason string) DriftEvent {
	risk, blocked := RiskLow, false
	if want.Requirement == "RE" { // RE 步驟不符視為高風險
		risk, blocked = RiskHigh, true
	}
	return DriftEvent{Kind: kind, Risk: risk, Blocked: blocked,
		Expected: expToEval(want), Actual: got, Reason: reason}
}
