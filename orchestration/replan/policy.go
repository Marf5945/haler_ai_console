package replan

import (
	"ui_console/domain/risk"
	"ui_console/orchestration/dag"
)

// LowConfidenceThreshold 以下視為低信心：只能把 silent 收緊成 review，不能放寬。
const LowConfidenceThreshold = 0.5

// GateInput 是 Gate 裁決所需的全部輸入。
type GateInput struct {
	Contract        dag.GoalContract // Go 持有的目標契約
	Proposal        ReplanProposal   // 模型提案
	Run             *dag.DAGRun      // 當前 run（讀取 succeeded 節點等）
	TriggerNodeRisk risk.RiskClass   // 觸發 replan 的節點本身風險（C1）
	Failure         FailureCategory  // 結構化失敗類型
	Counter         *Counter         // 連續無進展 + 總閘
	Oscillating     bool             // 此提案 tail 是否與先前雷同
}

// GateResult 是裁決結果與供 UX / audit 用的理由。
type GateResult struct {
	Decision      Decision
	Reason        string
	Stage         Stage
	ClassifiedMax risk.RiskClass // 新 tail 經確定性分類的最高風險
	ScopeReplan   bool           // 是否因改變目標而被擋
}

// Gate 是唯一的裁決入口。硬規則順序由嚴到寬，任一條不過即升級。
// 核心原則：模型只提案，Go 裁決；unknown / 無法判斷一律 review。
func Gate(in GateInput) GateResult {
	res := GateResult{Decision: DecisionReview, Stage: StageSilentNotice}

	// 0) 全域 deny：outside_scope / sensitive_path 永不 silent。
	if IsGloballyDenied(in.Failure) {
		res.Reason = "failure category globally denied: " + string(in.Failure)
		return res
	}

	// 1) 觸發節點本身必須 low（C1）。繞過非 low 失敗不可無聲。
	if in.TriggerNodeRisk != "" && in.TriggerNodeRisk != risk.Low {
		res.Reason = "trigger node not low: " + string(in.TriggerNodeRisk)
		return res
	}

	// 2) 計數上限：連續無進展或整趟總閘超標 → 停下交人。
	if in.Counter != nil && in.Counter.ShouldStop() {
		res.Decision = DecisionStop
		res.Stage = StageStop
		res.Reason = "replan budget exhausted"
		return res
	}

	// 3) 空 tail：可能是剪枝（完成）也可能是放棄。只有 output_contract 機器可驗
	//    且已被某 succeeded 節點滿足，才算完成可 silent；否則 review。
	if len(in.Proposal.ProposedTail) == 0 {
		if ContractOutputSatisfiedBy(in.Contract, in.Run) {
			res.Decision = DecisionSilent
			res.Stage = stageForSilent(in.Counter)
			res.Reason = "empty tail: output contract satisfied (pruning)"
			return res
		}
		res.Reason = "empty tail but output contract not satisfied (possible give-up)"
		return res
	}

	// 4) 同目標驗證（C4）。改變產出/範圍一律 review，風險再低也一樣。
	if !ContractStillSatisfiedBy(in.Contract, in.Proposal) {
		res.ScopeReplan = true
		res.Reason = "scope replan: proposal does not preserve goal contract"
		return res
	}

	// 5) 逐節點硬閘：read-only allowlist（准入）+ classifier 否決（盯目標）。
	maxRisk := risk.Low
	for _, n := range in.Proposal.ProposedTail {
		// 5a) 准入：非 read-only allowlist → 整條 tail 進 review（等同 medium+）。
		if !IsReadOnlyEligible(n.Action) {
			res.Reason = "action not read-only eligible: " + n.Action
			return res
		}
		// 5b) 否決：確定性分類盯「目標」是否把操作推進硬停類。
		cls := risk.ClassifyOperation(n.Action, targetsOf(n))
		if risk.IsHigherOrEqual(cls, maxRisk) {
			maxRisk = cls
		}
		if cls == risk.CriticalRuntimeAction {
			res.Decision = DecisionStop
			res.Stage = StageStop
			res.Reason = "classifier veto: critical runtime action on " + n.Target
			res.ClassifiedMax = cls
			return res
		}
		if risk.IsHigherOrEqual(cls, risk.HighNonDestructive) {
			res.Reason = "classifier veto: " + string(cls) + " on " + n.Target
			res.ClassifiedMax = cls
			return res
		}
	}
	res.ClassifiedMax = maxRisk

	// 6) 低信心只能收緊：模型沒把握時，寧可 review。
	if in.Proposal.Confidence > 0 && in.Proposal.Confidence < LowConfidenceThreshold {
		res.Reason = "low confidence, escalate to review"
		return res
	}

	// 通過全部硬閘 → 准許 silent。
	res.Decision = DecisionSilent
	res.Stage = stageForSilent(in.Counter)
	res.Reason = "path replan within read-only ceiling"
	return res
}

// targetsOf 把節點 target 包成 classifier 需要的 slice。
func targetsOf(n ProposedNode) []string {
	if n.Target == "" {
		return nil
	}
	return []string{n.Target}
}

// stageForSilent 依「下一次」的連續無進展數決定 UX 階段。
func stageForSilent(c *Counter) Stage {
	next := 1
	if c != nil {
		next = c.ConsecutiveNoProgress + 1
	}
	return StageFor(next)
}

// TriggerRiskFor 判定「觸發 replan 的失敗節點」風險，供 Gate 的 C1 使用。
// read-only allowlist 的動作視為 low（allowlist 已背書其安全；繞不開分類器預設 Medium
// 否則所有 read-only 失敗都會被擋在 silent 外）；其餘用確定性分類器。
func TriggerRiskFor(action, target string) risk.RiskClass {
	if IsReadOnlyEligible(action) {
		return risk.Low
	}
	return risk.ClassifyOperation(action, []string{target})
}

// StageFor 把連續無進展次數對映到顯示階段（1-2 低調 / 3-4 顯示計數 / >=5 停）。
func StageFor(consecutive int) Stage {
	switch {
	case consecutive >= MaxConsecutiveNoProgress:
		return StageStop
	case consecutive >= 3:
		return StageAdjusting
	default:
		return StageSilentNotice
	}
}
