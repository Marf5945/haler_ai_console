package remote_bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tmpAuditDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "audit_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// TestAuditLogAppendAndJSONLFormat writes entries then reads back
// to verify each line is valid JSONL.
func TestAuditLogAppendAndJSONLFormat(t *testing.T) {
	dir := tmpAuditDir(t)
	al := NewAuditLog(dir)

	entries := []AuditEntry{
		{DispatchID: "d-1", Channel: ChannelTelegram, Mode: ModeNotificationOnly, Outcome: "accepted"},
		{DispatchID: "d-2", Channel: ChannelDiscord, Mode: ModeRemoteReview, Outcome: "rejected"},
		{DispatchID: "d-3", Channel: ChannelLINE, Mode: ModeRemoteTaskSubmit, Outcome: "ignored"},
	}
	for _, e := range entries {
		if err := al.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	result, err := al.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	for i, r := range result {
		if r.DispatchID != entries[i].DispatchID {
			t.Errorf("entry %d: DispatchID expected %s, got %s", i, entries[i].DispatchID, r.DispatchID)
		}
		if r.Outcome != entries[i].Outcome {
			t.Errorf("entry %d: Outcome expected %s, got %s", i, entries[i].Outcome, r.Outcome)
		}
		if r.CreatedAt.IsZero() {
			t.Errorf("entry %d: CreatedAt should be auto-set", i)
		}
	}
}

// TestAuditLogReadRecent verifies ReadRecent returns the last N entries.
func TestAuditLogReadRecent(t *testing.T) {
	dir := tmpAuditDir(t)
	al := NewAuditLog(dir)

	for i := 0; i < 10; i++ {
		al.Append(AuditEntry{DispatchID: fmt.Sprintf("d-%d", i), Outcome: "ok"})
	}

	recent, err := al.ReadRecent(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent, got %d", len(recent))
	}
	// Should be the last 3
	if recent[0].DispatchID != "d-7" {
		t.Errorf("expected d-7, got %s", recent[0].DispatchID)
	}
}

// TestAuditLogSkipCorruptLines verifies that corrupt lines are skipped during ReadAll.
func TestAuditLogSkipCorruptLines(t *testing.T) {
	dir := tmpAuditDir(t)
	al := NewAuditLog(dir)

	// Write a valid entry
	al.Append(AuditEntry{DispatchID: "good-1", Outcome: "ok"})

	// Manually inject a corrupt line
	logPath := filepath.Join(dir, "remote_bridge", auditFileName)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("THIS IS NOT JSON\n"))
	f.Close()

	// Write another valid entry
	al.Append(AuditEntry{DispatchID: "good-2", Outcome: "ok"})

	result, err := al.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	// Should have 2 valid entries, corrupt line skipped
	if len(result) != 2 {
		t.Fatalf("expected 2 valid entries, got %d", len(result))
	}
	if result[0].DispatchID != "good-1" {
		t.Errorf("first entry should be good-1, got %s", result[0].DispatchID)
	}
	if result[1].DispatchID != "good-2" {
		t.Errorf("second entry should be good-2, got %s", result[1].DispatchID)
	}
}

// TestAuditLogFilePermission verifies the log file is created with 0600.
func TestAuditLogFilePermission(t *testing.T) {
	dir := tmpAuditDir(t)
	al := NewAuditLog(dir)

	al.Append(AuditEntry{DispatchID: "perm", Outcome: "ok"})

	logPath := filepath.Join(dir, "remote_bridge", auditFileName)
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("expected file permission 0600, got %04o", perm)
	}
}

// TestAuditLogDirPermission verifies the directory is created with 0700.
func TestAuditLogDirPermission(t *testing.T) {
	dir := tmpAuditDir(t)
	al := NewAuditLog(dir)

	al.Append(AuditEntry{DispatchID: "dir", Outcome: "ok"})

	auditDir := filepath.Join(dir, "remote_bridge")
	info, err := os.Stat(auditDir)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0o700 {
		t.Errorf("expected dir permission 0700, got %04o", perm)
	}
}

// TestAuditLogCreatedAtAutoSet verifies that CreatedAt is auto-set when zero.
func TestAuditLogCreatedAtAutoSet(t *testing.T) {
	dir := tmpAuditDir(t)
	al := NewAuditLog(dir)

	before := time.Now()
	al.Append(AuditEntry{DispatchID: "ts", Outcome: "ok"})
	after := time.Now()

	result, _ := al.ReadAll()
	if len(result) != 1 {
		t.Fatal("expected 1 entry")
	}
	if result[0].CreatedAt.Before(before) || result[0].CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not in expected range [%v, %v]", result[0].CreatedAt, before, after)
	}
}

// TestAuditLogEmptyReadAll verifies ReadAll returns nil for non-existent file.
func TestAuditLogEmptyReadAll(t *testing.T) {
	dir := tmpAuditDir(t)
	al := NewAuditLog(dir)

	result, err := al.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll on empty: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %d entries", len(result))
	}
}

