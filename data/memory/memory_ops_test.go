// memory/memory_ops_test.go — memory_ops.jsonl 雙鏈 Hash 單元測試。
// 驗證：hash chain 正確性、檔案權限、fail closed 行為、邊界條件。
package memory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════════════
// Test 1: 首筆 entry 的 prev_memory_op_hash 應為空字串
// ══════════════════════════════════════════════════════════════════════════════

func TestMemoryOps_FirstEntry_PrevHashEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// 直接呼叫 appendMemoryOp（模擬 pipeline 內部）
	err := appendMemoryOp(tmpDir, "append", EmptyFileSHA256, "abc123")
	if err != nil {
		t.Fatalf("appendMemoryOp failed: %v", err)
	}

	// 讀取並驗證
	entry := readLastEntry(t, tmpDir)
	if entry.PrevMemoryOpHash != "" {
		t.Errorf("first entry prev_memory_op_hash should be empty, got %q", entry.PrevMemoryOpHash)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Test 2: 第二筆 entry 的 prev_memory_op_hash == 第一筆的 memory_op_hash
// ══════════════════════════════════════════════════════════════════════════════

func TestMemoryOps_SecondEntry_PrevHashChain(t *testing.T) {
	tmpDir := t.TempDir()

	// 寫入兩筆
	_ = appendMemoryOp(tmpDir, "append", EmptyFileSHA256, "hash1")
	firstEntry := readLastEntry(t, tmpDir)

	_ = appendMemoryOp(tmpDir, "append", "hash1", "hash2")
	secondEntry := readLastEntry(t, tmpDir)

	if secondEntry.PrevMemoryOpHash != firstEntry.MemoryOpHash {
		t.Errorf("second.prev (%s) != first.hash (%s)",
			secondEntry.PrevMemoryOpHash, firstEntry.MemoryOpHash)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Test 3: memory_op_hash 是 entry 自身欄位（不含 memory_op_hash）的 SHA-256
// ══════════════════════════════════════════════════════════════════════════════

func TestMemoryOps_HashIsCorrect(t *testing.T) {
	tmpDir := t.TempDir()

	_ = appendMemoryOp(tmpDir, "append", "before1", "after1")
	entry := readLastEntry(t, tmpDir)

	// 手動重新計算
	verify := entry
	verify.MemoryOpHash = ""
	data, _ := json.Marshal(verify)
	expected := fmt.Sprintf("%x", sha256.Sum256(data))

	if entry.MemoryOpHash != expected {
		t.Errorf("hash mismatch:\n  got:      %s\n  expected: %s", entry.MemoryOpHash, expected)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Test 4: schema_version 正確
// ══════════════════════════════════════════════════════════════════════════════

func TestMemoryOps_SchemaVersion(t *testing.T) {
	tmpDir := t.TempDir()

	_ = appendMemoryOp(tmpDir, "append", "b", "a")
	entry := readLastEntry(t, tmpDir)

	if entry.SchemaVersion != "v4.0-memory-guard-1" {
		t.Errorf("wrong schema version: %s", entry.SchemaVersion)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Test 5: 檔案權限 0600
// ══════════════════════════════════════════════════════════════════════════════

func TestMemoryOps_FilePermission0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not report Unix permission bits reliably")
	}
	tmpDir := t.TempDir()

	_ = appendMemoryOp(tmpDir, "append", "b", "a")

	info, err := os.Stat(filepath.Join(tmpDir, FileMemoryOps))
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected 0600, got %04o", perm)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Test 6: talk_full 不存在時 computeFileSHA256 回傳 EmptyFileSHA256
// ══════════════════════════════════════════════════════════════════════════════

func TestMemoryOps_EmptyFileHash(t *testing.T) {
	hash, err := computeFileSHA256("/nonexistent/path/file.md")
	if err != nil {
		t.Fatalf("should not error for nonexistent: %v", err)
	}
	if hash != EmptyFileSHA256 {
		t.Errorf("expected empty file hash, got %s", hash)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Test 7: AppendTalkEntry 整合測試 — 正確產生 memory_ops entry
// ══════════════════════════════════════════════════════════════════════════════

func TestMemoryOps_AppendTalkEntryIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	pipeline := NewPipeline(tmpDir)

	// 第一次 append
	_, err := pipeline.AppendTalkEntry("user", "Hello world")
	if err != nil {
		t.Fatalf("AppendTalkEntry failed: %v", err)
	}

	// 驗證 memory_ops.jsonl 存在且有一行
	opsPath := filepath.Join(tmpDir, "memory", FileMemoryOps)
	data, err := os.ReadFile(opsPath)
	if err != nil {
		t.Fatalf("read memory_ops failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var entry MemoryOpEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("parse entry failed: %v", err)
	}

	// 驗證 hash_before 是空檔案 hash（因為 talk_full 之前不存在）
	if entry.TalkFullHashBefore != EmptyFileSHA256 {
		t.Errorf("first entry hash_before should be empty file hash, got %s", entry.TalkFullHashBefore)
	}

	// 驗證 hash_after 不為空且不等於 before
	if entry.TalkFullHashAfter == "" || entry.TalkFullHashAfter == EmptyFileSHA256 {
		t.Errorf("hash_after should not be empty after write")
	}

	// 第二次 append — 驗證 chain
	_, err = pipeline.AppendTalkEntry("assistant", "Hi there")
	if err != nil {
		t.Fatalf("second AppendTalkEntry failed: %v", err)
	}

	data, _ = os.ReadFile(opsPath)
	lines = strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var entry2 MemoryOpEntry
	json.Unmarshal([]byte(lines[1]), &entry2)

	// talk_full chain: entry2.hash_before == entry1.hash_after
	if entry2.TalkFullHashBefore != entry.TalkFullHashAfter {
		t.Errorf("talk_full chain broken: entry2.before=%s != entry1.after=%s",
			entry2.TalkFullHashBefore, entry.TalkFullHashAfter)
	}

	// memory_ops chain: entry2.prev == entry1.memory_op_hash
	if entry2.PrevMemoryOpHash != entry.MemoryOpHash {
		t.Errorf("memory_ops chain broken: entry2.prev=%s != entry1.hash=%s",
			entry2.PrevMemoryOpHash, entry.MemoryOpHash)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// 輔助函式
// ══════════════════════════════════════════════════════════════════════════════

func readLastEntry(t *testing.T, dir string) MemoryOpEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, FileMemoryOps))
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	lastLine := lines[len(lines)-1]

	var entry MemoryOpEntry
	if err := json.Unmarshal([]byte(lastLine), &entry); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	return entry
}
