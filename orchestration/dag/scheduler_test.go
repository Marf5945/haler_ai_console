package dag

import (
	"strings"
	"os"
	"path/filepath"
	"testing"
)

// 測試建立 DAGRun
func TestCreateRun(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewScheduler(tmpDir)

	nodes := []DAGNode{
		{ID: "n1", Operation: "fetch_data", RiskClass: "low"},
		{ID: "n2", Operation: "process", RiskClass: "medium", Dependencies: []string{"n1"}},
	}

	run, err := s.CreateRun(nodes, "test-hash")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if run.ID == "" {
		t.Error("run ID should not be empty")
	}
	if run.Status != "running" {
		t.Errorf("initial status should be running, got %s", run.Status)
	}

	// n1 無依賴 → ready
	if run.Nodes[0].Status != StatusReady {
		t.Errorf("n1 should be ready, got %s", run.Nodes[0].Status)
	}
	// n2 依賴 n1 → planned
	if run.Nodes[1].Status != StatusPlanned {
		t.Errorf("n2 should be planned, got %s", run.Nodes[1].Status)
	}
}

// 測試空節點拒絕
func TestCreateRunEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewScheduler(tmpDir)
	_, err := s.CreateRun(nil, "hash")
	if err == nil {
		t.Error("should reject empty nodes")
	}
}

// 測試節點狀態推進
func TestNodeStatusProgression(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewScheduler(tmpDir)

	nodes := []DAGNode{
		{ID: "n1", Operation: "step1", RiskClass: "low"},
		{ID: "n2", Operation: "step2", RiskClass: "low", Dependencies: []string{"n1"}},
		{ID: "n3", Operation: "step3", RiskClass: "low", Dependencies: []string{"n2"}},
	}

	run, _ := s.CreateRun(nodes, "hash")

	// n1: ready → running → succeeded
	s.UpdateNodeStatus(run.ID, "n1", StatusRunning)
	s.UpdateNodeStatus(run.ID, "n1", StatusSucceeded)

	// n2 應該變為 ready
	updated, _ := s.GetRun(run.ID)
	var n2Status NodeStatus
	for _, n := range updated.Nodes {
		if n.ID == "n2" {
			n2Status = n.Status
		}
	}
	if n2Status != StatusReady {
		t.Errorf("n2 should be ready after n1 succeeded, got %s", n2Status)
	}
}

// 測試 DAGRun 完成
func TestRunCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewScheduler(tmpDir)

	nodes := []DAGNode{
		{ID: "n1", Operation: "only", RiskClass: "low"},
	}

	run, _ := s.CreateRun(nodes, "hash")
	s.UpdateNodeStatus(run.ID, "n1", StatusRunning)
	s.UpdateNodeStatus(run.ID, "n1", StatusSucceeded)

	updated, _ := s.GetRun(run.ID)
	if updated.Status != "completed" {
		t.Errorf("run should be completed, got %s", updated.Status)
	}
}

// 測試 blocked 狀態
func TestSetNodeBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewScheduler(tmpDir)

	nodes := []DAGNode{
		{ID: "n1", Operation: "risky", RiskClass: "high_non_destructive"},
	}

	run, _ := s.CreateRun(nodes, "hash")
	s.SetNodeBlocked(run.ID, "n1", "RESUME_GUARD_HASH_CHANGED")

	updated, _ := s.GetRun(run.ID)
	if updated.Status != "blocked" {
		t.Errorf("run should be blocked, got %s", updated.Status)
	}
	if updated.Nodes[0].BlockReason != "RESUME_GUARD_HASH_CHANGED" {
		t.Error("block reason mismatch")
	}
}

// 測試 failed 狀態
func TestSetNodeFailed(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewScheduler(tmpDir)

	nodes := []DAGNode{
		{ID: "n1", Operation: "fragile", RiskClass: "low"},
	}

	run, _ := s.CreateRun(nodes, "hash")
	s.SetNodeFailed(run.ID, "n1", "connection timeout")

	updated, _ := s.GetRun(run.ID)
	if updated.Status != "failed" {
		t.Errorf("run should be failed, got %s", updated.Status)
	}
}

// 測試僅關鍵節點持久化
func TestSaveCriticalNodes(t *testing.T) {
	tmpDir := t.TempDir()

	run := &DAGRun{
		ID:     "test-run",
		Status: "blocked",
		Nodes: []DAGNode{
			{ID: "n1", Status: StatusSucceeded},
			{ID: "n2", Status: StatusBlocked, BlockReason: "guard changed"},
			{ID: "n3", Status: StatusPlanned},
			{ID: "n4", Status: StatusFailed, Error: "timeout"},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
		GuardHash: "abc123",
	}

	err := SaveCriticalNodes(tmpDir, run)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// 驗證檔案存在
	path := filepath.Join(tmpDir, "dag_runs", "test-run.critical.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("persisted file should exist")
	}

	// 載入並驗證只有關鍵節點
	loaded, err := LoadCriticalNodes(tmpDir, "test-run")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	// 應只有 n2 (blocked) 和 n4 (failed) = 2 個關鍵節點
	if len(loaded.Nodes) != 2 {
		t.Errorf("expected 2 critical nodes, got %d", len(loaded.Nodes))
	}
}

// 測試 ListPersistedRuns
func TestListPersistedRuns(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "dag_runs")
	os.MkdirAll(dir, 0755)

	os.WriteFile(filepath.Join(dir, "run-1.critical.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "run-2.critical.json"), []byte("{}"), 0644)

	ids, err := ListPersistedRuns(tmpDir)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 runs, got %d", len(ids))
	}
}

// 測試 Resume Guard：hash 匹配
func TestResumeGuardSafe(t *testing.T) {
	hashes := GuardHashes{
		SubMemoryHash:   "hash1",
		ToolRegistryHash: "hash2",
		RiskPolicyHash:   "hash3",
		SourceTrustHash:  "hash4",
	}

	run := &DAGRun{GuardHash: hashes.CombinedHash()}
	result := CheckResumeGuard(run, hashes)
	if !result.Safe {
		t.Error("same hashes should be safe")
	}
}

// 測試 Resume Guard：hash 不匹配
func TestResumeGuardUnsafe(t *testing.T) {
	saved := GuardHashes{
		SubMemoryHash:   "hash1",
		ToolRegistryHash: "hash2",
		RiskPolicyHash:   "hash3",
		SourceTrustHash:  "hash4",
	}
	current := GuardHashes{
		SubMemoryHash:   "hash1_changed",
		ToolRegistryHash: "hash2",
		RiskPolicyHash:   "hash3",
		SourceTrustHash:  "hash4",
	}

	run := &DAGRun{GuardHash: saved.CombinedHash()}
	result := CheckResumeGuard(run, current)
	if result.Safe {
		t.Error("different hashes should be unsafe")
	}
	if result.BlockReason != "RESUME_GUARD_HASH_CHANGED" {
		t.Errorf("wrong block reason: %s", result.BlockReason)
	}
}

// 測試詳細 Guard 比對
func TestResumeGuardDetailed(t *testing.T) {
	saved := GuardHashes{
		SubMemoryHash:   "aaa",
		ToolRegistryHash: "bbb",
		RiskPolicyHash:   "ccc",
		SourceTrustHash:  "ddd",
	}
	current := GuardHashes{
		SubMemoryHash:   "aaa",
		ToolRegistryHash: "bbb_changed",
		RiskPolicyHash:   "ccc",
		SourceTrustHash:  "ddd_changed",
	}

	result := CheckResumeGuardDetailed(saved, current)
	if result.Safe {
		t.Error("should be unsafe")
	}
	if len(result.ChangedFields) != 2 {
		t.Errorf("should have 2 changed fields, got %d", len(result.ChangedFields))
	}
}

// 測試 Pre-execution Guard：低風險跳過
func TestPreExecutionGuardLowRisk(t *testing.T) {
	node := &DAGNode{ID: "n1", RiskClass: "low"}
	saved := GuardHashes{}
	result := CheckPreExecutionGuard(node, t.TempDir(), saved)
	if !result.Safe {
		t.Error("low risk should skip pre-execution guard")
	}
}

// 測試 Guard Hash 持久化
func TestGuardHashSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	hashes := GuardHashes{
		SubMemoryHash:   "m1",
		ToolRegistryHash: "t1",
		RiskPolicyHash:   "r1",
		SourceTrustHash:  "s1",
	}

	err := SaveGuardHashes(tmpDir, "run-test", hashes)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadGuardHashes(tmpDir, "run-test")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.SubMemoryHash != "m1" {
		t.Error("hash mismatch after load")
	}
}

// 測試 IsCriticalStatus
func TestIsCriticalStatus(t *testing.T) {
	cases := []struct {
		status   NodeStatus
		critical bool
	}{
		{StatusWaitingReview, true},
		{StatusBlocked, true},
		{StatusFailed, true},
		{StatusSucceeded, false},
		{StatusRunning, false},
		{StatusPlanned, false},
		{StatusReady, false},
	}
	for _, c := range cases {
		if IsCriticalStatus(c.status) != c.critical {
			t.Errorf("IsCriticalStatus(%s) = %v, want %v", c.status, !c.critical, c.critical)
		}
	}
}

// 測試 IsTerminal
func TestIsTerminal(t *testing.T) {
	cases := []struct {
		status   NodeStatus
		terminal bool
	}{
		{StatusSucceeded, true},
		{StatusFailed, true},
		{StatusCancelled, true},
		{StatusSkipped, true},
		{StatusRunning, false},
		{StatusBlocked, false},
	}
	for _, c := range cases {
		if IsTerminal(c.status) != c.terminal {
			t.Errorf("IsTerminal(%s) = %v, want %v", c.status, !c.terminal, c.terminal)
		}
	}
}

// 測試 LoadGuardHashes 偵測 deprecated 舊格式
func TestLoadGuardHashes_DeprecatedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "dag_runs")
	os.MkdirAll(dir, 0755)

	// 寫入含有 main_memory_hash 的舊格式檔案
	oldFormat := `{
		"main_memory_hash": "old_hash",
		"tool_registry_hash": "t1",
		"risk_policy_hash": "r1",
		"source_trust_hash": "s1"
	}`
	os.WriteFile(filepath.Join(dir, "run-old_guard.json"), []byte(oldFormat), 0644)

	// LoadGuardHashes 應回傳明確的 deprecated error
	_, err := LoadGuardHashes(tmpDir, "run-old")
	if err == nil {
		t.Fatal("LoadGuardHashes should return error for deprecated format")
	}
	if !strings.Contains(err.Error(), "deprecated guard format") {
		t.Errorf("error should mention 'deprecated guard format', got: %v", err)
	}
}

// 測試 LoadGuardHashes 正常 v4.0 格式通過
func TestLoadGuardHashes_V4Format(t *testing.T) {
	tmpDir := t.TempDir()
	hashes := GuardHashes{
		SubMemoryHash:    "sm1",
		ToolRegistryHash: "t1",
		RiskPolicyHash:   "r1",
		SourceTrustHash:  "s1",
	}

	err := SaveGuardHashes(tmpDir, "run-new", hashes)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadGuardHashes(tmpDir, "run-new")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.SubMemoryHash != "sm1" {
		t.Errorf("SubMemoryHash mismatch: got %s", loaded.SubMemoryHash)
	}
}
