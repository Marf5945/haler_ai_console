package controlled_trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ui_console/audit_log"
)

func tmpTrustDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "trust_log_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// TestTrustLogAppendAndJSONLFormat writes N entries then reads back to verify
// each line is valid JSON and fields are populated correctly.
func TestTrustLogAppendAndJSONLFormat(t *testing.T) {
	dir := tmpTrustDir(t)
	tl := NewTrustLog(dir)

	entries := []TrustLogEntry{
		{Type: "override_granted", ScopeHash: "scope_a"},
		{Type: "session_started", ScopeHash: "scope_b"},
		{Type: "override_expired", ScopeHash: "scope_c"},
	}
	for _, e := range entries {
		if err := tl.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Read back the JSONL file
	logPath := filepath.Join(dir, "controlled_trust_log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := audit_log.SplitLines(data)
	var parsed []TrustLogEntry
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e TrustLogEntry
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("invalid JSONL line: %v\nline: %s", err, line)
		}
		parsed = append(parsed, e)
	}

	if len(parsed) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(parsed))
	}
	for i, e := range parsed {
		if e.ID == "" {
			t.Errorf("entry %d: ID should not be empty", i)
		}
		if e.Type != entries[i].Type {
			t.Errorf("entry %d: Type expected %s, got %s", i, entries[i].Type, e.Type)
		}
		if e.EntryHash == "" {
			t.Errorf("entry %d: EntryHash should not be empty", i)
		}
		if e.CreatedAt.IsZero() {
			t.Errorf("entry %d: CreatedAt should not be zero", i)
		}
	}
}

// TestTrustLogHashChainIntegrity verifies that each entry's PreviousEntryHash
// links to the prior entry's EntryHash, forming an intact chain.
func TestTrustLogHashChainIntegrity(t *testing.T) {
	dir := tmpTrustDir(t)
	tl := NewTrustLog(dir)

	for i := 0; i < 5; i++ {
		if err := tl.Append(TrustLogEntry{Type: "test", ScopeHash: "s"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	logPath := filepath.Join(dir, "controlled_trust_log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	lines := audit_log.SplitLines(data)
	prevHash := genesisEntryHash
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e TrustLogEntry
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("line %d: parse error: %v", i, err)
		}
		if e.PreviousEntryHash != prevHash {
			t.Errorf("line %d: broken chain link: expected prev=%s, got %s", i, prevHash, e.PreviousEntryHash)
		}
		// Verify hash is self-consistent
		expected := computeEntryHash(e)
		if e.EntryHash != expected {
			t.Errorf("line %d: hash mismatch: expected %s, got %s", i, expected, e.EntryHash)
		}
		prevHash = e.EntryHash
	}
}

// TestTrustLogGenesisHash verifies the first entry links to the genesis hash.
func TestTrustLogGenesisHash(t *testing.T) {
	dir := tmpTrustDir(t)
	tl := NewTrustLog(dir)

	if err := tl.Append(TrustLogEntry{Type: "first"}); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(dir, "controlled_trust_log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := audit_log.SplitLines(data)
	var first TrustLogEntry
	for _, line := range lines {
		if len(line) > 0 {
			json.Unmarshal(line, &first)
			break
		}
	}
	if first.PreviousEntryHash != genesisEntryHash {
		t.Errorf("first entry should link to genesis hash, got %s", first.PreviousEntryHash)
	}
}

// TestTrustLogFilePermission verifies the log file is created with 0600 permission.
func TestTrustLogFilePermission(t *testing.T) {
	dir := tmpTrustDir(t)
	tl := NewTrustLog(dir)

	if err := tl.Append(TrustLogEntry{Type: "perm_test"}); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(dir, "controlled_trust_log.jsonl")
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("expected file permission 0600, got %04o", perm)
	}
}

// TestTrustLogDirPermission verifies the directory is created with 0700.
func TestTrustLogDirPermission(t *testing.T) {
	dir := tmpTrustDir(t)
	subDir := filepath.Join(dir, "nested_trust")
	tl := NewTrustLog(subDir)

	if err := tl.Append(TrustLogEntry{Type: "dir_test"}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0o700 {
		t.Errorf("expected dir permission 0700, got %04o", perm)
	}
}

// TestTrustLogInvariantReject verifies that entries with FinalRiskChanged=true
// or HardRulesModified=true are rejected.
func TestTrustLogInvariantReject(t *testing.T) {
	dir := tmpTrustDir(t)
	tl := NewTrustLog(dir)

	err := tl.Append(TrustLogEntry{Type: "bad", FinalRiskChanged: true})
	if err == nil {
		t.Error("should reject FinalRiskChanged=true")
	}

	err = tl.Append(TrustLogEntry{Type: "bad", HardRulesModified: true})
	if err == nil {
		t.Error("should reject HardRulesModified=true")
	}
}

// TestTrustLogReopen verifies that a TrustLog re-opened from an existing file
// continues the hash chain correctly.
func TestTrustLogReopen(t *testing.T) {
	dir := tmpTrustDir(t)

	// First session: append 3 entries
	tl1 := NewTrustLog(dir)
	for i := 0; i < 3; i++ {
		if err := tl1.Append(TrustLogEntry{Type: "session1"}); err != nil {
			t.Fatal(err)
		}
	}

	// Second session: re-open and append more
	tl2 := NewTrustLog(dir)
	if err := tl2.Append(TrustLogEntry{Type: "session2"}); err != nil {
		t.Fatal(err)
	}

	// Verify full chain integrity
	logPath := filepath.Join(dir, "controlled_trust_log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := audit_log.SplitLines(data)
	prevHash := genesisEntryHash
	count := 0
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e TrustLogEntry
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if e.PreviousEntryHash != prevHash {
			t.Fatalf("broken chain at entry %d", count)
		}
		prevHash = e.EntryHash
		count++
	}
	if count != 4 {
		t.Errorf("expected 4 total entries, got %d", count)
	}
}
