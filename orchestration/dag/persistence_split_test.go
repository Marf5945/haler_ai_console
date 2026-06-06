package dag

import "testing"

// V-31-01：critical 與 full 拆檔後不互相覆蓋，關鍵節點不遺失。
func TestCriticalAndFullDoNotCollide(t *testing.T) {
	tmp := t.TempDir()
	run := &DAGRun{
		ID:     "run-x",
		Status: "blocked",
		Nodes: []DAGNode{
			{ID: "n1", Status: StatusSucceeded},
			{ID: "n2", Status: StatusBlocked, BlockReason: "guard"},
			{ID: "n3", Status: StatusFailed, Error: "boom"},
		},
	}
	if err := SaveCriticalNodes(tmp, run); err != nil {
		t.Fatalf("save critical: %v", err)
	}
	// full 後寫，過去會覆蓋 critical；拆檔後兩者並存。
	if err := SaveFullRun(tmp, run); err != nil {
		t.Fatalf("save full: %v", err)
	}
	crit, err := LoadCriticalNodes(tmp, "run-x")
	if err != nil {
		t.Fatalf("load critical: %v", err)
	}
	if len(crit.Nodes) != 2 { // 只有 blocked + failed 為關鍵
		t.Fatalf("關鍵節點應為 2，得到 %d（疑似被 full 覆蓋）", len(crit.Nodes))
	}
	full, err := LoadFullRun(tmp, "run-x")
	if err != nil {
		t.Fatalf("load full: %v", err)
	}
	if len(full.Nodes) != 3 {
		t.Fatalf("完整節點應為 3，得到 %d", len(full.Nodes))
	}
}
