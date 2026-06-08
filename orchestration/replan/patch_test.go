package replan

import (
	"errors"
	"testing"

	"ui_console/orchestration/dag"
)

// 一個含 1 succeeded + 2 planned tail 的 run。
func sampleRun() *dag.DAGRun {
	return &dag.DAGRun{
		ID:           "dag-1",
		Revision:     0,
		ActiveNodeID: "n1",
		Nodes: []dag.DAGNode{
			{ID: "n1", Action: "grep_search", Target: "proj/a", Status: dag.StatusSucceeded, ResultSummary: "found"},
			{ID: "n2", Action: "read_file", Target: "proj/b", Status: dag.StatusPlanned, Dependencies: []string{"n1"}},
			{ID: "n3", Action: "read_file", Target: "proj/c", Status: dag.StatusReady},
		},
	}
}

func validPatch(run *dag.DAGRun) TailPatch {
	p := ReplanProposal{ProposedTail: []ProposedNode{{Action: "glob", Target: "proj/*.md"}}}
	return TailPatch{
		ExpectedRevision:     run.Revision,
		ExpectedActiveNodeID: run.ActiveNodeID,
		ExpectedOldTailHash:  ComputeTailHash(PlannedTail(run)),
		NewNodes:             BuildNewNodes(p, run.Revision+1),
	}
}

func TestApplyTailPatch_Success(t *testing.T) {
	run := sampleRun()
	patch := validPatch(run)
	if err := ApplyTailPatch(run, patch); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if run.Revision != 1 {
		t.Errorf("revision should bump to 1, got %d", run.Revision)
	}
	// 應保留 succeeded n1，丟棄舊 tail n2/n3，換成新節點。
	if len(run.Nodes) != 2 {
		t.Fatalf("want 2 nodes (n1 + 1 new), got %d", len(run.Nodes))
	}
	if run.Nodes[0].ID != "n1" {
		t.Errorf("succeeded node must be kept, got %s", run.Nodes[0].ID)
	}
	if run.Nodes[1].ID != "r1_node_1" {
		t.Errorf("new node id must be versioned, got %s", run.Nodes[1].ID)
	}
}

func TestApplyTailPatch_RevisionConflict(t *testing.T) {
	run := sampleRun()
	patch := validPatch(run)
	run.Revision = 5 // 期間被別人推進
	if err := ApplyTailPatch(run, patch); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("want revision conflict, got %v", err)
	}
}

func TestApplyTailPatch_ActiveNodeConflict(t *testing.T) {
	run := sampleRun()
	patch := validPatch(run)
	run.ActiveNodeID = "n2" // 當前節點已改變
	if err := ApplyTailPatch(run, patch); !errors.Is(err, ErrActiveNodeConflict) {
		t.Fatalf("want active node conflict, got %v", err)
	}
}

func TestApplyTailPatch_TailHashConflict(t *testing.T) {
	run := sampleRun()
	patch := validPatch(run)
	run.Nodes[2].Target = "proj/changed" // tail 被動過
	if err := ApplyTailPatch(run, patch); !errors.Is(err, ErrTailHashConflict) {
		t.Fatalf("want tail hash conflict, got %v", err)
	}
}

func TestBuildNewNodes_ClassifierOverridesModelRisk(t *testing.T) {
	// 模型自報 low，但目標含 destructive 關鍵字 → Go 裁定須非 low。
	p := ReplanProposal{ProposedTail: []ProposedNode{
		{Action: "read_file", Target: "delete_project", ModelRiskClass: "low"},
	}}
	nodes := BuildNewNodes(p, 1)
	if nodes[0].RiskClass == "low" {
		t.Fatalf("classifier must override model self-report, got %s", nodes[0].RiskClass)
	}
	if nodes[0].ModelRiskClass != "low" {
		t.Errorf("model self-report should be preserved for reference")
	}
}

func TestTailSignature_OrderInsensitive(t *testing.T) {
	a := []ProposedNode{{Action: "grep_search", Target: "x"}, {Action: "glob", Target: "y"}}
	b := []ProposedNode{{Action: "glob", Target: "y"}, {Action: "grep_search", Target: "x"}}
	if TailSignature(a) != TailSignature(b) {
		t.Errorf("signature must be order-insensitive (A<->B oscillation detection)")
	}
}
