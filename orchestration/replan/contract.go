package replan

import (
	"strings"

	"ui_console/orchestration/dag"
)

// ContractStillSatisfiedBy 由 Go 確定性地驗證提案是否仍服務原目標（C4）。
// 保守規則：
//   - legacy（IsZero）→ false，強制 review。
//   - 模型自報 scope_change → false（宣稱改範圍就 review）。
//   - 設了 Scope 時，每個新節點 target 必須落在 Scope 前綴內。
//
// 註：contract 資料型別在 dag，行為留在 replan，避免 dag→replan 套件循環。
func ContractStillSatisfiedBy(c dag.GoalContract, p ReplanProposal) bool {
	if c.IsZero() {
		return false
	}
	if p.Intent == IntentScopeChange {
		return false
	}
	if c.Scope != "" {
		for _, n := range p.ProposedTail {
			t := strings.TrimSpace(n.Target)
			if t != "" && !strings.HasPrefix(t, c.Scope) {
				return false // 目標跑出範圍 = scope replan
			}
		}
	}
	return true
}

// ContractOutputSatisfiedBy 判斷空 tail 是否真的「已完成」而非「放棄」。
// v1 要求機器可驗：須設 OutputPredicate，且有 succeeded 節點的 OutputRef/ResultSummary
// 命中該 predicate；否則回 false → review。
func ContractOutputSatisfiedBy(c dag.GoalContract, run *dag.DAGRun) bool {
	if c.IsZero() || c.OutputPredicate == "" || run == nil {
		return false
	}
	for _, n := range run.Nodes {
		if n.Status != dag.StatusSucceeded {
			continue
		}
		hay := n.OutputRef + "\x1f" + n.ResultSummary
		if strings.Contains(hay, c.OutputPredicate) {
			return true
		}
	}
	return false
}
