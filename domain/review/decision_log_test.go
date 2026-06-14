package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ui_console/audit_log"
	"ui_console/domain/risk"
)

func tmpDataRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "review_log_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// TestWriteDecisionLogJSONLFormat verifies that resolving a high-risk card
// produces a valid JSONL entry in review_decision_log.jsonl.
func TestWriteDecisionLogJSONLFormat(t *testing.T) {
	dataRoot := tmpDataRoot(t)
	svc := NewServiceWithDataRoot(dataRoot)

	// Add and resolve a high-risk card (only high_non_destructive+ gets logged)
	card := svc.AddCard(CardParams{
		RiskClass:   risk.HighNonDestructive,
		Operation:   "install_package",
		Target:      "pkg:dangerous-lib",
		Reason:      "安裝外部套件",
		AcceptLabel: "確認",
		RejectLabel: "取消",
	})
	if err := svc.Resolve(card.ID); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Read back the decision log
	logPath := filepath.Join(dataRoot, "review", "review_decision_log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := audit_log.SplitLines(data)
	var entries []decisionLogEntry
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e decisionLogEntry
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("invalid JSONL: %v\nline: %s", err, line)
		}
		entries = append(entries, e)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ReviewID != card.ID {
		t.Errorf("ReviewID: expected %s, got %s", card.ID, entries[0].ReviewID)
	}
	if entries[0].RiskClass != risk.HighNonDestructive {
		t.Errorf("RiskClass: expected %s, got %s", risk.HighNonDestructive, entries[0].RiskClass)
	}
	if entries[0].Decision != "accepted" {
		t.Errorf("Decision: expected accepted, got %s", entries[0].Decision)
	}
	if entries[0].Operation != "install_package" {
		t.Errorf("Operation: expected install_package, got %s", entries[0].Operation)
	}
}

// TestWriteDecisionLogFilePermission verifies the log file is 0644.
func TestWriteDecisionLogFilePermission(t *testing.T) {
	dataRoot := tmpDataRoot(t)
	svc := NewServiceWithDataRoot(dataRoot)

	card := svc.AddCard(CardParams{
		RiskClass:   risk.HighNonDestructive,
		Operation:   "test_perm",
		Target:      "target",
		Reason:      "test",
		AcceptLabel: "ok",
		RejectLabel: "cancel",
	})
	svc.Resolve(card.ID)

	logPath := filepath.Join(dataRoot, "review", "review_decision_log.jsonl")
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	// SEC-W07 第二刀（2026-05-24）：P-6 拿掉 audit_log WithFilePermission(0o644) override，
	// 改用 framework 預設 0o600。decision log 為私有稽核資料，無外部讀取需求。
	if perm != 0o600 {
		t.Errorf("expected file permission 0600 (post SEC-W07), got %04o", perm)
	}
}

// TestWriteDecisionLogDirPermission verifies the review directory is 0755.
func TestWriteDecisionLogDirPermission(t *testing.T) {
	dataRoot := tmpDataRoot(t)
	svc := NewServiceWithDataRoot(dataRoot)

	card := svc.AddCard(CardParams{
		RiskClass:   risk.HighNonDestructive,
		Operation:   "test_dir_perm",
		Target:      "target",
		Reason:      "test",
		AcceptLabel: "ok",
		RejectLabel: "cancel",
	})
	svc.Resolve(card.ID)

	reviewDir := filepath.Join(dataRoot, "review")
	info, err := os.Stat(reviewDir)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0o755 {
		t.Errorf("expected dir permission 0755, got %04o", perm)
	}
}

// TestWriteDecisionLogMediumSkipped verifies that medium-risk cards
// do NOT produce a decision log entry (§5.4: only high+ is logged).
func TestWriteDecisionLogMediumSkipped(t *testing.T) {
	dataRoot := tmpDataRoot(t)
	svc := NewServiceWithDataRoot(dataRoot)

	card := svc.AddCard(CardParams{
		RiskClass:   risk.Medium,
		Operation:   "low_risk_op",
		Target:      "target",
		Reason:      "test",
		AcceptLabel: "ok",
		RejectLabel: "cancel",
	})
	svc.Resolve(card.ID)

	logPath := filepath.Join(dataRoot, "review", "review_decision_log.jsonl")
	_, err := os.Stat(logPath)
	if !os.IsNotExist(err) {
		t.Error("medium-risk resolution should not create decision log")
	}
}

// TestWriteDecisionLogSecurityBoundaryRewriteAlsoWritesSecLog verifies
// security_boundary_rewrite creates entries in both decision and security logs.
func TestWriteDecisionLogSecurityBoundaryRewriteAlsoWritesSecLog(t *testing.T) {
	dataRoot := tmpDataRoot(t)
	svc := NewServiceWithDataRoot(dataRoot)

	// Directly invoke writeDecisionLog with a resolved security_boundary_rewrite card
	// (same package access). This bypasses the dual-step cooldown for testing the log path.
	card := Card{
		ID:               "test-sec-card",
		RiskClass:        risk.SecurityBoundaryRewrite,
		Operation:        "modify_risk_policy",
		Target:           "risk_policy:main",
		Resolved:         true,
		ResolvedAt:       nowRFC3339(),
		RequiresDualStep: true,
	}
	svc.writeDecisionLog(card)

	// Verify both logs exist
	decisionLogPath := filepath.Join(dataRoot, "review", "review_decision_log.jsonl")
	if _, err := os.Stat(decisionLogPath); os.IsNotExist(err) {
		t.Error("decision log should be created")
	}

	secLogPath := filepath.Join(dataRoot, "review", "security_change_log.jsonl")
	if _, err := os.Stat(secLogPath); os.IsNotExist(err) {
		t.Error("security_boundary_rewrite should also write security_change_log.jsonl")
	}
}
