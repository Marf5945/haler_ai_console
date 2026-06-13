package replan

import (
	"testing"

	"ui_console/domain/risk"
	"ui_console/orchestration/dag"
)

// 共用：一個帶 scope 的非空 contract。
func baseContract() dag.GoalContract {
	return dag.GoalContract{GoalSummary: "整理研究資料", OutputType: "file", Scope: "proj/"}
}

func TestGate_SilentReadOnlyPathReplan(t *testing.T) {
	in := GateInput{
		Contract: baseContract(),
		Proposal: ReplanProposal{
			Intent:       IntentSameGoalPath,
			Confidence:   0.8,
			ProposedTail: []ProposedNode{{Action: "grep_search", Target: "proj/data"}},
		},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	got := Gate(in)
	if got.Decision != DecisionSilent {
		t.Fatalf("want silent, got %s (%s)", got.Decision, got.Reason)
	}
	if got.Stage != StageSilentNotice {
		t.Errorf("want silent_notice stage, got %s", got.Stage)
	}
}

func TestGate_NonReadOnlyTailReview(t *testing.T) {
	in := GateInput{
		Contract:        baseContract(),
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: []ProposedNode{{Action: "xlsx_write", Target: "proj/out.xlsx"}}},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	if got := Gate(in); got.Decision != DecisionReview {
		t.Fatalf("non read-only tail must review, got %s", got.Decision)
	}
}

func TestGate_ScopeChangeReview(t *testing.T) {
	in := GateInput{
		Contract:        baseContract(),
		Proposal:        ReplanProposal{Intent: IntentScopeChange, ProposedTail: []ProposedNode{{Action: "grep_search", Target: "proj/x"}}},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureAmbiguous,
		Counter:         NewCounter(),
	}
	got := Gate(in)
	if got.Decision != DecisionReview || !got.ScopeReplan {
		t.Fatalf("scope change must review with ScopeReplan, got %s scope=%v", got.Decision, got.ScopeReplan)
	}
}

func TestGate_GlobalDeny(t *testing.T) {
	for _, f := range []FailureCategory{FailureOutsideScope, FailureSensitivePath} {
		in := GateInput{
			Contract:        baseContract(),
			Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: []ProposedNode{{Action: "read_file", Target: "proj/a"}}},
			TriggerNodeRisk: risk.Low,
			Failure:         f,
			Counter:         NewCounter(),
		}
		if got := Gate(in); got.Decision != DecisionReview {
			t.Errorf("%s must be denied to review, got %s", f, got.Decision)
		}
	}
}

func TestGate_TriggerNotLowReview(t *testing.T) {
	in := GateInput{
		Contract:        baseContract(),
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: []ProposedNode{{Action: "read_file", Target: "proj/a"}}},
		TriggerNodeRisk: risk.Medium,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	if got := Gate(in); got.Decision != DecisionReview {
		t.Fatalf("non-low trigger must review, got %s", got.Decision)
	}
}

func TestGate_CounterStop(t *testing.T) {
	c := NewCounter()
	c.ConsecutiveNoProgress = MaxConsecutiveNoProgress // 已達上限
	in := GateInput{
		Contract:        baseContract(),
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: []ProposedNode{{Action: "read_file", Target: "proj/a"}}},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         c,
	}
	if got := Gate(in); got.Decision != DecisionStop {
		t.Fatalf("exhausted budget must stop, got %s", got.Decision)
	}
}

func TestGate_ClassifierVetoCritical(t *testing.T) {
	// read_file 在 allowlist，但目標含 critical 關鍵字 → classifier 否決 → stop。
	in := GateInput{
		Contract:        dag.GoalContract{GoalSummary: "x", OutputType: "answer"}, // 無 scope，避免先被 scope 擋
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: []ProposedNode{{Action: "read_file", Target: "login"}}},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	if got := Gate(in); got.Decision != DecisionStop {
		t.Fatalf("critical veto must stop, got %s (%s)", got.Decision, got.Reason)
	}
}

func TestGate_ClassifierVetoDestructiveReview(t *testing.T) {
	in := GateInput{
		Contract:        dag.GoalContract{GoalSummary: "x", OutputType: "answer"},
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: []ProposedNode{{Action: "read_file", Target: "delete_project"}}},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	if got := Gate(in); got.Decision != DecisionReview {
		t.Fatalf("destructive veto must review, got %s", got.Decision)
	}
}

func TestGate_EmptyTailGiveUpReview(t *testing.T) {
	in := GateInput{
		Contract:        baseContract(), // 無 OutputPredicate → 無法確認完成
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: nil},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	if got := Gate(in); got.Decision != DecisionReview {
		t.Fatalf("empty tail give-up must review, got %s", got.Decision)
	}
}

func TestGate_EmptyTailCompleteSilent(t *testing.T) {
	run := &dag.DAGRun{Nodes: []dag.DAGNode{
		{ID: "n1", Status: dag.StatusSucceeded, OutputRef: "proj/report.md"},
	}}
	in := GateInput{
		Contract:        dag.GoalContract{GoalSummary: "做摘要", OutputType: "file", OutputPredicate: "report.md"},
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: nil},
		Run:             run,
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	if got := Gate(in); got.Decision != DecisionSilent {
		t.Fatalf("empty tail with satisfied output must silent, got %s (%s)", got.Decision, got.Reason)
	}
}

func TestGate_LowConfidenceTightens(t *testing.T) {
	in := GateInput{
		Contract:        baseContract(),
		Proposal:        ReplanProposal{Intent: IntentSameGoalPath, Confidence: 0.3, ProposedTail: []ProposedNode{{Action: "grep_search", Target: "proj/a"}}},
		TriggerNodeRisk: risk.Low,
		Failure:         FailureNoResults,
		Counter:         NewCounter(),
	}
	if got := Gate(in); got.Decision != DecisionReview {
		t.Fatalf("low confidence must tighten to review, got %s", got.Decision)
	}
}

func TestTriggerRiskFor(t *testing.T) {
	// allowlist 動作（即使 classifier 預設 Medium）→ 視為 low，才可能 silent。
	if TriggerRiskFor("grep_search", "proj/a") != risk.Low {
		t.Errorf("grep_search trigger must be low")
	}
	if TriggerRiskFor("read_file", "proj/a") != risk.Low {
		t.Errorf("read_file trigger must be low (allowlist vouches)")
	}
	// 非 allowlist → 用分類器（非 low），擋在 silent 外。
	if TriggerRiskFor("xlsx_write", "proj/out.xlsx") == risk.Low {
		t.Errorf("non-allowlist trigger must not be low")
	}
}
