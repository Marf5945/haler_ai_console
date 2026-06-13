package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveFullRunLocked_RoundTripAndAtomic(t *testing.T) {
	tmp := t.TempDir()
	run := &DAGRun{
		ID:           "dag-atomic-1",
		Status:       "running",
		Revision:     3,
		ActiveNodeID: "n1",
		GoalContract: &GoalContract{GoalSummary: "x", OutputType: "file"},
		Nodes: []DAGNode{
			{ID: "n1", Action: "grep_search", Status: StatusSucceeded, ResultSummary: "ok"},
		},
	}
	if err := SaveFullRunLocked(tmp, run); err != nil {
		t.Fatalf("locked save failed: %v", err)
	}

	loaded, err := LoadFullRun(tmp, run.ID)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Revision != 3 {
		t.Errorf("revision not preserved, got %d", loaded.Revision)
	}
	if loaded.GoalContract == nil || loaded.GoalContract.GoalSummary != "x" {
		t.Errorf("goal contract not preserved: %+v", loaded.GoalContract)
	}
	if len(loaded.Nodes) != 1 || loaded.Nodes[0].ID != "n1" {
		t.Errorf("nodes not preserved")
	}

	// 不可留下 .tmp 暫存檔（rename 應已清掉）。
	entries, _ := os.ReadDir(filepath.Join(tmp, "dag_runs"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}
