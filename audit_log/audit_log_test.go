package audit_log

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ──────────────────────────────────────────────
// 測試用型別
// ──────────────────────────────────────────────

type testEntry struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type chainedEntry struct {
	Index        int    `json:"index"`
	Payload      string `json:"payload"`
	PreviousHash string `json:"previous_hash"`
	Hash         string `json:"hash"`
}

func computeTestHash(e chainedEntry) string {
	raw := fmt.Sprintf("%d|%s|%s", e.Index, e.Payload, e.PreviousHash)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func tmpDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "audit_log_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// ──────────────────────────────────────────────
// 基本 Append + ReadAll
// ──────────────────────────────────────────────

func TestAppendAndReadAll(t *testing.T) {
	dir := tmpDir(t)
	log := New[testEntry](filepath.Join(dir, "test.jsonl"))

	for i := 0; i < 5; i++ {
		if err := log.Append(testEntry{ID: fmt.Sprintf("e-%d", i), Message: "hello"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	entries, err := log.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5, got %d", len(entries))
	}
	for i, e := range entries {
		expected := fmt.Sprintf("e-%d", i)
		if e.ID != expected {
			t.Errorf("entry %d: expected ID %s, got %s", i, expected, e.ID)
		}
	}
}

// ──────────────────────────────────────────────
// ReadRecent
// ──────────────────────────────────────────────

func TestReadRecent(t *testing.T) {
	dir := tmpDir(t)
	log := New[testEntry](filepath.Join(dir, "test.jsonl"))

	for i := 0; i < 10; i++ {
		log.Append(testEntry{ID: fmt.Sprintf("e-%d", i)})
	}

	recent, err := log.ReadRecent(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 3 {
		t.Fatalf("expected 3, got %d", len(recent))
	}
	if recent[0].ID != "e-7" {
		t.Errorf("expected e-7, got %s", recent[0].ID)
	}
}

func TestReadRecentMoreThanAvailable(t *testing.T) {
	dir := tmpDir(t)
	log := New[testEntry](filepath.Join(dir, "test.jsonl"))

	log.Append(testEntry{ID: "only"})

	recent, err := log.ReadRecent(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 1 {
		t.Fatalf("expected 1, got %d", len(recent))
	}
}

// ──────────────────────────────────────────────
// Empty / Non-existent
// ──────────────────────────────────────────────

func TestReadAllEmpty(t *testing.T) {
	dir := tmpDir(t)
	log := New[testEntry](filepath.Join(dir, "nonexistent.jsonl"))

	entries, err := log.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil, got %d entries", len(entries))
	}
}

// ──────────────────────────────────────────────
// File & Dir Permissions
// ──────────────────────────────────────────────

func TestDefaultPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not report Unix permission bits reliably")
	}
	dir := tmpDir(t)
	subDir := filepath.Join(dir, "sub")
	log := New[testEntry](filepath.Join(subDir, "test.jsonl"))

	log.Append(testEntry{ID: "perm"})

	// File: default 0600
	fi, err := os.Stat(filepath.Join(subDir, "test.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("file perm: expected 0600, got %04o", fi.Mode().Perm())
	}

	// Dir: default 0700
	di, err := os.Stat(subDir)
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Errorf("dir perm: expected 0700, got %04o", di.Mode().Perm())
	}
}

func TestCustomPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not report Unix permission bits reliably")
	}
	dir := tmpDir(t)
	subDir := filepath.Join(dir, "custom")
	log := New[testEntry](
		filepath.Join(subDir, "test.jsonl"),
		WithFilePermission[testEntry](0o644),
		WithDirPermission[testEntry](0o755),
	)

	log.Append(testEntry{ID: "perm"})

	fi, _ := os.Stat(filepath.Join(subDir, "test.jsonl"))
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("file perm: expected 0644, got %04o", fi.Mode().Perm())
	}

	di, _ := os.Stat(subDir)
	if di.Mode().Perm() != 0o755 {
		t.Errorf("dir perm: expected 0755, got %04o", di.Mode().Perm())
	}
}

// ──────────────────────────────────────────────
// SkipCorruptLines
// ──────────────────────────────────────────────

func TestSkipCorruptLines(t *testing.T) {
	dir := tmpDir(t)
	p := filepath.Join(dir, "corrupt.jsonl")
	log := New[testEntry](p, WithSkipCorruptLines[testEntry]())

	log.Append(testEntry{ID: "good-1"})

	// Inject corrupt line
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write([]byte("NOT JSON AT ALL\n"))
	f.Close()

	log.Append(testEntry{ID: "good-2"})

	entries, err := log.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll should not error with SkipCorruptLines: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 good entries, got %d", len(entries))
	}
}

func TestCorruptLineErrorWithoutSkip(t *testing.T) {
	dir := tmpDir(t)
	p := filepath.Join(dir, "corrupt.jsonl")
	log := New[testEntry](p) // no SkipCorruptLines

	log.Append(testEntry{ID: "good"})

	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write([]byte("CORRUPT\n"))
	f.Close()

	_, err := log.ReadAll()
	if err == nil {
		t.Error("expected parse error for corrupt line")
	}
}

// ──────────────────────────────────────────────
// BeforeAppend hook
// ──────────────────────────────────────────────

func TestBeforeAppendReject(t *testing.T) {
	dir := tmpDir(t)
	log := New[testEntry](
		filepath.Join(dir, "validated.jsonl"),
		WithBeforeAppend[testEntry](func(e *testEntry) error {
			if e.ID == "bad" {
				return fmt.Errorf("rejected: invalid ID")
			}
			return nil
		}),
	)

	if err := log.Append(testEntry{ID: "good"}); err != nil {
		t.Fatalf("good entry rejected: %v", err)
	}
	if err := log.Append(testEntry{ID: "bad"}); err == nil {
		t.Error("bad entry should be rejected")
	}

	entries, _ := log.ReadAll()
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestBeforeAppendMutate(t *testing.T) {
	dir := tmpDir(t)
	log := New[testEntry](
		filepath.Join(dir, "mutated.jsonl"),
		WithBeforeAppend[testEntry](func(e *testEntry) error {
			e.Message = "auto-set"
			return nil
		}),
	)

	log.Append(testEntry{ID: "e1"})

	entries, _ := log.ReadAll()
	if len(entries) != 1 || entries[0].Message != "auto-set" {
		t.Errorf("expected message auto-set, got %+v", entries)
	}
}

// ──────────────────────────────────────────────
// Hash Chain
// ──────────────────────────────────────────────

const testGenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

func TestHashChain(t *testing.T) {
	dir := tmpDir(t)
	log := New[chainedEntry](
		filepath.Join(dir, "chain.jsonl"),
		WithHashChain[chainedEntry](
			testGenesisHash,
			func(entry *chainedEntry, state *ChainState) {
				entry.Index = state.Length
				entry.PreviousHash = state.LastHash
				entry.Hash = computeTestHash(*entry)
			},
			func(entry *chainedEntry, state *ChainState) {
				state.LastHash = entry.Hash
				state.Length++
			},
			func(entry *chainedEntry, state *ChainState) {
				state.LastHash = entry.Hash
				state.Length = entry.Index + 1
			},
		),
	)

	for i := 0; i < 5; i++ {
		if err := log.Append(chainedEntry{Payload: fmt.Sprintf("data-%d", i)}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	// Verify chain integrity
	data, _ := os.ReadFile(filepath.Join(dir, "chain.jsonl"))
	lines := SplitLines(data)
	prevHash := testGenesisHash
	count := 0
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e chainedEntry
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatal(err)
		}
		if e.PreviousHash != prevHash {
			t.Errorf("entry %d: broken link: expected prev %s, got %s", e.Index, prevHash, e.PreviousHash)
		}
		expected := computeTestHash(e)
		if e.Hash != expected {
			t.Errorf("entry %d: hash mismatch", e.Index)
		}
		prevHash = e.Hash
		count++
	}
	if count != 5 {
		t.Errorf("expected 5 entries, got %d", count)
	}
}

func TestHashChainReopen(t *testing.T) {
	dir := tmpDir(t)
	p := filepath.Join(dir, "chain.jsonl")

	makeLog := func() *AppendLog[chainedEntry] {
		return New[chainedEntry](
			p,
			WithHashChain[chainedEntry](
				testGenesisHash,
				func(entry *chainedEntry, state *ChainState) {
					entry.Index = state.Length
					entry.PreviousHash = state.LastHash
					entry.Hash = computeTestHash(*entry)
				},
				func(entry *chainedEntry, state *ChainState) {
					state.LastHash = entry.Hash
					state.Length++
				},
				func(entry *chainedEntry, state *ChainState) {
					state.LastHash = entry.Hash
					state.Length = entry.Index + 1
				},
			),
		)
	}

	// Session 1: write 3 entries
	log1 := makeLog()
	for i := 0; i < 3; i++ {
		log1.Append(chainedEntry{Payload: fmt.Sprintf("s1-%d", i)})
	}

	// Session 2: re-open and write 2 more
	log2 := makeLog()
	for i := 0; i < 2; i++ {
		log2.Append(chainedEntry{Payload: fmt.Sprintf("s2-%d", i)})
	}

	// Verify full chain
	data, _ := os.ReadFile(p)
	lines := SplitLines(data)
	prevHash := testGenesisHash
	count := 0
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e chainedEntry
		json.Unmarshal(line, &e)
		if e.PreviousHash != prevHash {
			t.Errorf("entry %d: broken chain after reopen", e.Index)
		}
		prevHash = e.Hash
		count++
	}
	if count != 5 {
		t.Errorf("expected 5 total, got %d", count)
	}
}

// ──────────────────────────────────────────────
// SplitLines
// ──────────────────────────────────────────────

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"line1\n", 1},
		{"line1\nline2\n", 2},
		{"line1\nline2", 2}, // no trailing newline
		{"\n\nline\n", 3},   // empty lines counted
	}
	for _, tc := range tests {
		lines := SplitLines([]byte(tc.input))
		if len(lines) != tc.expected {
			t.Errorf("SplitLines(%q): expected %d lines, got %d", tc.input, tc.expected, len(lines))
		}
	}
}

// ──────────────────────────────────────────────
// Fixture 相容性（舊格式檔案可讀取）
// ──────────────────────────────────────────────

func TestReadOldFormatFixture(t *testing.T) {
	dir := tmpDir(t)
	p := filepath.Join(dir, "old_format.jsonl")

	// Simulate an old-format file written by the original implementation
	fixture := `{"id":"e-0","message":"old entry 1"}
{"id":"e-1","message":"old entry 2"}
{"id":"e-2","message":"old entry 3"}
`
	os.WriteFile(p, []byte(fixture), 0o600)

	log := New[testEntry](p)
	entries, err := log.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll old format: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3, got %d", len(entries))
	}
	if entries[0].ID != "e-0" || entries[2].Message != "old entry 3" {
		t.Error("old format entries not parsed correctly")
	}
}
