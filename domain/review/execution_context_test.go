// review/execution_context_test.go — TASKS_1_7 後端驗收測試
package review

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── 驗收 1：target 順序不同但內容相同時 hash 一致 ──
func TestTargetHashSet_OrderIndependent(t *testing.T) {
	a := []TargetEntry{
		{Operation: "delete", Target: "/data/a.json", AffectedScope: "project"},
		{Operation: "delete", Target: "/data/b.json", AffectedScope: "project"},
	}
	b := []TargetEntry{
		{Operation: "delete", Target: "/data/b.json", AffectedScope: "project"},
		{Operation: "delete", Target: "/data/a.json", AffectedScope: "project"},
	}
	hashA := ComputeTargetHashSet(a)
	hashB := ComputeTargetHashSet(b)
	if hashA != hashB {
		t.Errorf("同內容不同順序 hash 應一致: %s != %s", hashA, hashB)
	}
}

// ── 驗收 2：target 內容改變 hash 不同 ──
func TestTargetHashSet_ContentChange(t *testing.T) {
	original := []TargetEntry{{Operation: "delete", Target: "/data/a.json", AffectedScope: "project"}}
	modified := []TargetEntry{{Operation: "delete", Target: "/data/a_modified.json", AffectedScope: "project"}}
	h1 := ComputeTargetHashSet(original)
	h2 := ComputeTargetHashSet(modified)
	if h1 == h2 {
		t.Error("內容改變後 hash 應不同")
	}
}

// ── 驗收 3：Step 1 後 3 秒內 Resolve 必須失敗 ──
func TestResolve_CooldownEnforced(t *testing.T) {
	svc := NewService()
	card := svc.AddCard(CardParams{
		RiskClass:   "security_boundary_rewrite",
		Operation:   "test_op",
		Target:      "test_target",
		Reason:      "test",
		AcceptLabel: "OK",
		RejectLabel: "Cancel",
		AcceptEffect: "effect",
		RejectEffect: "none",
	})

	// Step 1
	err := svc.DualStepConfirmStep1(card.ID, "scope", "risk", "tool")
	if err != nil {
		t.Fatalf("Step 1 should succeed: %v", err)
	}

	// 立即 Resolve 應該失敗（冷卻未到）
	err = svc.Resolve(card.ID)
	if err == nil {
		t.Error("Resolve within cooldown should fail")
	}
}

// ── 驗收 4：Step 1 後修改 policy 檔案，Step 2 必須失敗並 invalidate ──
func TestResolve_HashMismatchInvalidates(t *testing.T) {
	// 建立臨時專案目錄
	tmpDir := t.TempDir()
	riskDir := filepath.Join(tmpDir, "risk_policy")
	toolDir := filepath.Join(tmpDir, "tool_registry")
	os.MkdirAll(riskDir, 0o700)
	os.MkdirAll(toolDir, 0o700)

	policyFile := filepath.Join(riskDir, "policy.json")
	registryFile := filepath.Join(toolDir, "registry.json")
	os.WriteFile(policyFile, []byte(`{"version": 1}`), 0o600)
	os.WriteFile(registryFile, []byte(`{"tools": []}`), 0o600)

	svc := NewService()
	card := svc.AddCard(CardParams{
		RiskClass:    "security_boundary_rewrite",
		Operation:    "test_op",
		Target:       "test_target",
		Reason:       "test",
		AcceptLabel:  "OK",
		RejectLabel:  "Cancel",
		AcceptEffect: "effect",
		RejectEffect: "none",
	})

	// Step 1（hash 從檔案計算）
	err := svc.DualStepConfirmStep1(card.ID,
		computeFileHashForReview(policyFile),   // scopeHash 模擬
		computeFileHashForReview(policyFile),   // riskPolicyHash
		computeFileHashForReview(registryFile), // toolRegistryHash
	)
	if err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}

	// 修改 policy.json
	os.WriteFile(policyFile, []byte(`{"version": 2, "changed": true}`), 0o600)

	// 等待冷卻期
	time.Sleep(3100 * time.Millisecond)

	// Step 2 應失敗（hash 不一致）
	err = svc.Resolve(card.ID, tmpDir)
	if err != ErrReviewContextChanged {
		t.Errorf("Expected ErrReviewContextChanged, got: %v", err)
	}

	// 確認 card 已 invalidated
	c, _ := svc.GetCard(card.ID)
	if c.DualStepState == nil || !c.DualStepState.Invalidated {
		t.Error("Card should be invalidated after hash mismatch")
	}
}
