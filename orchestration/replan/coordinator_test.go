package replan

import (
	"context"
	"errors"
	"testing"

	"ui_console/domain/risk"
	"ui_console/orchestration/dag"
)

// fakeProposer 回傳預設提案 / 錯誤，讓 Coordinator 迴路可離線單測。
type fakeProposer struct {
	proposal ReplanProposal
	err      error
}

func (f fakeProposer) Propose(_ context.Context, _ ProposerContext) (ReplanProposal, error) {
	return f.proposal, f.err
}

func coordRun() *dag.DAGRun {
	return &dag.DAGRun{
		ID:           "dag-c1",
		Revision:     0,
		ActiveNodeID: "n1",
		GoalContract: &dag.GoalContract{GoalSummary: "整理", OutputType: "file", Scope: "proj/"},
		Nodes: []dag.DAGNode{
			{ID: "n1", Status: dag.StatusSucceeded, ResultSummary: "done"},
			{ID: "n2", Action: "read_file", Target: "proj/b", Status: dag.StatusPlanned},
		},
	}
}

func TestCoordinator_SilentApply(t *testing.T) {
	run := coordRun()
	fp := fakeProposer{proposal: ReplanProposal{
		Intent: IntentSameGoalPath, Confidence: 0.8,
		ProposedTail: []ProposedNode{{Action: "grep_search", Target: "proj/data"}},
	}}
	co := NewCoordinator(fp, NewCounter(), nil, 0)
	res := co.Attempt(run, FailureNoResults, risk.Low)

	if res.Decision != DecisionSilent || !res.Applied {
		t.Fatalf("want silent+applied, got %s applied=%v (%s)", res.Decision, res.Applied, res.Reason)
	}
	if run.Revision != 1 {
		t.Errorf("revision should bump, got %d", run.Revision)
	}
	if len(run.Nodes) != 2 || run.Nodes[1].ID != "r1_node_1" {
		t.Errorf("tail not replaced with versioned node: %+v", run.Nodes)
	}
	if co.Counter.ConsecutiveNoProgress != 1 {
		t.Errorf("counter should increment to 1, got %d", co.Counter.ConsecutiveNoProgress)
	}
}

func TestCoordinator_NonReadOnlyReview(t *testing.T) {
	run := coordRun()
	fp := fakeProposer{proposal: ReplanProposal{
		Intent:       IntentSameGoalPath,
		ProposedTail: []ProposedNode{{Action: "xlsx_write", Target: "proj/out.xlsx"}},
	}}
	co := NewCoordinator(fp, NewCounter(), nil, 0)
	res := co.Attempt(run, FailureNoResults, risk.Low)

	if res.Decision != DecisionReview || res.Applied {
		t.Fatalf("want review+not-applied, got %s applied=%v", res.Decision, res.Applied)
	}
	if run.Revision != 0 {
		t.Errorf("review must not mutate run, revision=%d", run.Revision)
	}
	if co.Counter.ConsecutiveNoProgress != 0 {
		t.Errorf("review must not consume counter")
	}
}

func TestCoordinator_CounterStop(t *testing.T) {
	run := coordRun()
	c := NewCounter()
	c.ConsecutiveNoProgress = MaxConsecutiveNoProgress
	fp := fakeProposer{proposal: ReplanProposal{Intent: IntentSameGoalPath, ProposedTail: []ProposedNode{{Action: "grep_search", Target: "proj/x"}}}}
	co := NewCoordinator(fp, c, nil, 0)
	if res := co.Attempt(run, FailureNoResults, risk.Low); res.Decision != DecisionStop {
		t.Fatalf("want stop, got %s", res.Decision)
	}
}

func TestCoordinator_ProposerErrorFailSafe(t *testing.T) {
	run := coordRun()
	fp := fakeProposer{err: errors.New("model timeout")}
	co := NewCoordinator(fp, NewCounter(), nil, 0)
	res := co.Attempt(run, FailureNoResults, risk.Low)
	if res.Decision != DecisionReview || res.Applied {
		t.Fatalf("proposer error must fail-safe to review, got %s applied=%v", res.Decision, res.Applied)
	}
	if run.Revision != 0 {
		t.Errorf("fail-safe must not mutate run")
	}
}

func TestCoordinator_AuditWritten(t *testing.T) {
	run := coordRun()
	audit := NewAuditLog(t.TempDir())
	fp := fakeProposer{proposal: ReplanProposal{
		Intent: IntentSameGoalPath, Confidence: 0.8,
		ProposedTail: []ProposedNode{{Action: "grep_search", Target: "proj/data"}},
	}}
	co := NewCoordinator(fp, NewCounter(), audit, 0)
	_ = co.Attempt(run, FailureNoResults, risk.Low)

	entries, err := audit.ReadAll()
	if err != nil {
		t.Fatalf("read audit failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 audit entry, got %d", len(entries))
	}
	if !entries[0].Silent || entries[0].RunID != "dag-c1" {
		t.Errorf("audit entry wrong: %+v", entries[0])
	}
}
