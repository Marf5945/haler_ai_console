// dag/index_test.go — DAG Runs Index 測試。
package dag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendAndListDAGRuns(t *testing.T) {
	tmpDir := t.TempDir()

	// T-DI-01：CreateRun → index 新增 entry
	err := AppendRunIndex(tmpDir, DAGRunSummary{
		RunID:     "dag-001",
		Status:    "running",
		StartedAt: "2026-05-17T10:00:00Z",
		NodeCount: 3,
	})
	if err != nil {
		t.Fatalf("AppendRunIndex failed: %v", err)
	}

	err = AppendRunIndex(tmpDir, DAGRunSummary{
		RunID:     "dag-002",
		Status:    "failed",
		StartedAt: "2026-05-17T11:00:00Z",
		NodeCount: 2,
	})
	if err != nil {
		t.Fatalf("AppendRunIndex failed: %v", err)
	}

	// ListDAGRuns 不篩選
	runs := ListDAGRuns(tmpDir, 10, "")
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	// 最新在前
	if runs[0].RunID != "dag-002" {
		t.Errorf("expected dag-002 first, got %s", runs[0].RunID)
	}

	// T-DL-02：statusFilter="failed" → 只回傳失敗的
	failedRuns := ListDAGRuns(tmpDir, 10, "failed")
	if len(failedRuns) != 1 {
		t.Fatalf("expected 1 failed run, got %d", len(failedRuns))
	}
	if failedRuns[0].RunID != "dag-002" {
		t.Errorf("expected dag-002, got %s", failedRuns[0].RunID)
	}

	// T-DL-01：limit=1 → 最多回傳 1 筆
	limited := ListDAGRuns(tmpDir, 1, "")
	if len(limited) != 1 {
		t.Fatalf("expected 1 run with limit=1, got %d", len(limited))
	}

	// T-DL-03：空 index → 回傳空陣列
	emptyDir := t.TempDir()
	empty := ListDAGRuns(emptyDir, 10, "")
	if len(empty) != 0 {
		t.Errorf("expected empty list, got %d", len(empty))
	}
}

func TestUpdateRunIndex(t *testing.T) {
	tmpDir := t.TempDir()

	AppendRunIndex(tmpDir, DAGRunSummary{
		RunID:     "dag-upd",
		Status:    "running",
		StartedAt: "2026-05-17T10:00:00Z",
		NodeCount: 5,
	})

	// T-DI-02：UpdateNodeStatus → index entry 更新
	err := UpdateRunIndex(tmpDir, "dag-upd", "failed", "2026-05-17T10:05:00Z", 300000, 3, 2, "timeout on node-3")
	if err != nil {
		t.Fatalf("UpdateRunIndex failed: %v", err)
	}

	runs := ListDAGRuns(tmpDir, 10, "")
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Status != "failed" {
		t.Errorf("expected failed, got %s", runs[0].Status)
	}
	if runs[0].FailedNodeCount != 2 {
		t.Errorf("expected 2 failed nodes, got %d", runs[0].FailedNodeCount)
	}
	if runs[0].NodeCount != 3 {
		t.Errorf("expected 3 nodes, got %d", runs[0].NodeCount)
	}
	if runs[0].DurationMs != 300000 {
		t.Errorf("expected 300000ms, got %d", runs[0].DurationMs)
	}
}

func TestUpdateRunIndex_ErrorSummaryTruncation(t *testing.T) {
	tmpDir := t.TempDir()

	AppendRunIndex(tmpDir, DAGRunSummary{
		RunID:     "dag-trunc",
		Status:    "running",
		StartedAt: "2026-05-17T10:00:00Z",
		NodeCount: 1,
	})

	// T-DI-04：error_summary 超過 120 字元被截斷
	longError := ""
	for i := 0; i < 200; i++ {
		longError += "x"
	}

	UpdateRunIndex(tmpDir, "dag-trunc", "failed", "", 0, 1, 1, longError)

	runs := ListDAGRuns(tmpDir, 10, "")
	if len([]rune(runs[0].ErrorSummary)) > 120 {
		t.Errorf("error_summary not truncated: len=%d", len([]rune(runs[0].ErrorSummary)))
	}
}

func TestListDAGRuns_IndexMissing_Rebuild(t *testing.T) {
	// T-DI-03：index 遺失 → 回傳空（readdir 重建留作 v2）
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "dag"), 0700)

	runs := ListDAGRuns(tmpDir, 10, "")
	if runs == nil {
		t.Error("expected non-nil (empty slice)")
	}
}
