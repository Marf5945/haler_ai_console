// memory/memory_ops.go — memory_ops.jsonl 雙鏈 Hash Producer（§23 v4.0 Memory Guard）。
//
// 設計：
//   - 每次對 talk_full.md 的寫入都產生一筆 MemoryOpEntry，寫入 memory_ops.jsonl
//   - 雙鏈 hash：(1) talk_full 修改鏈 (2) memory_ops log 自身鏈
//   - 檔案以 O_APPEND|O_CREATE|O_WRONLY + 0600 權限寫入
//   - 首筆 prev_memory_op_hash = ""
//   - talk_full.md 不存在時 before_hash = SHA-256 of empty file
//
// 安全性：
//   - memory_ops 寫入失敗 → 回傳 error（fail closed）
//   - 外部呼叫者須在 p.mu.Lock() 保護下呼叫 appendMemoryOp
package memory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════════════════
// 常數
// ══════════════════════════════════════════════════════════════════════════════

const (
	// FileMemoryOps 是 memory_ops.jsonl 的檔名。
	FileMemoryOps = "memory_ops.jsonl"

	// memoryOpsSchemaVersion 與 spec_patch_checker contract 一致。
	memoryOpsSchemaVersion = "v4.0-memory-guard-1"

	// EmptyFileSHA256 是空檔案（0 bytes）的 SHA-256 hex。
	EmptyFileSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// ══════════════════════════════════════════════════════════════════════════════
// MemoryOpEntry — 每筆 memory_ops.jsonl 的資料結構
// ══════════════════════════════════════════════════════════════════════════════

// MemoryOpEntry 代表 memory_ops.jsonl 中的一行 JSON。
// 用於事後驗證 talk_full.md 是否被繞過 pipeline 改動。
type MemoryOpEntry struct {
	SchemaVersion      string `json:"schema_version"`        // 固定 "v4.0-memory-guard-1"
	EventType          string `json:"event_type"`            // "memory_write_event"
	Target             string `json:"target"`                // "memory/talk_full.md"
	Operation          string `json:"operation"`             // "append" | "rotate" | "delete_sentences"
	TalkFullHashBefore string `json:"talk_full_hash_before"` // 寫入前即時讀檔 SHA-256
	TalkFullHashAfter  string `json:"talk_full_hash_after"`  // 寫入後即時讀檔 SHA-256
	PrevMemoryOpHash   string `json:"prev_memory_op_hash"`   // 前一筆 entry 的 memory_op_hash
	MemoryOpHash       string `json:"memory_op_hash"`        // 當前 entry 全欄位（不含自身）的 SHA-256
	Timestamp          string `json:"timestamp"`             // RFC3339
}

// ══════════════════════════════════════════════════════════════════════════════
// Hash 計算輔助
// ══════════════════════════════════════════════════════════════════════════════

// computeFileSHA256 計算檔案內容的 SHA-256 hex。
// 檔案不存在時回傳 EmptyFileSHA256（空檔案 hash）。
func computeFileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return EmptyFileSHA256, nil
		}
		return "", fmt.Errorf("讀取檔案計算 hash 失敗: %w", err)
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// computeEntryHash 計算 MemoryOpEntry 的 hash（不含 memory_op_hash 欄位本身）。
// 使用 json.Marshal 確保 deterministic（Go struct 欄位宣告順序）。
func computeEntryHash(entry MemoryOpEntry) string {
	// 暫存並清空 memory_op_hash，對剩餘欄位做 hash
	saved := entry.MemoryOpHash
	entry.MemoryOpHash = ""

	data, _ := json.Marshal(entry) // Go struct → deterministic JSON
	hash := sha256.Sum256(data)

	entry.MemoryOpHash = saved // 恢復（雖然此處 entry 是值傳遞）
	return fmt.Sprintf("%x", hash)
}

// ══════════════════════════════════════════════════════════════════════════════
// 讀取最後一筆 entry 的 memory_op_hash
// ══════════════════════════════════════════════════════════════════════════════

// readLastMemoryOpHash 讀取 memory_ops.jsonl 最後一行的 memory_op_hash。
// 若檔案不存在或為空，回傳 ""（首筆 entry 的 prev 為空）。
func readLastMemoryOpHash(opsPath string) (string, error) {
	data, err := os.ReadFile(opsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("讀取 memory_ops.jsonl 失敗: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", nil
	}

	// 取最後一行（非空）
	lines := strings.Split(content, "\n")
	lastLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastLine = lines[i]
			break
		}
	}
	if lastLine == "" {
		return "", nil
	}

	// 解析 memory_op_hash 欄位
	var partial struct {
		MemoryOpHash string `json:"memory_op_hash"`
	}
	if err := json.Unmarshal([]byte(lastLine), &partial); err != nil {
		return "", fmt.Errorf("解析最後一筆 memory_ops entry 失敗: %w", err)
	}

	return partial.MemoryOpHash, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// 核心寫入函式
// ══════════════════════════════════════════════════════════════════════════════

// appendMemoryOp 寫入一筆 MemoryOpEntry 到 memory_ops.jsonl。
// 呼叫者須確保已持有 p.mu.Lock()。
//
// 參數：
//   - rootDir: memory 目錄根路徑
//   - operation: "append" | "rotate" | "delete_sentences"
//   - talkFullHashBefore: talk_full.md 寫入前的 SHA-256
//   - talkFullHashAfter: talk_full.md 寫入後的 SHA-256
//
// 回傳：
//   - error: 寫入失敗時回傳 error（fail closed）
func appendMemoryOp(rootDir, operation, talkFullHashBefore, talkFullHashAfter string) error {
	opsPath := filepath.Join(rootDir, FileMemoryOps)

	// ── 讀取前一筆的 memory_op_hash ──
	prevHash, err := readLastMemoryOpHash(opsPath)
	if err != nil {
		return fmt.Errorf("memory_ops: 無法讀取前一筆 hash: %w", err)
	}

	// ── 建立 entry（不含 memory_op_hash）──
	entry := MemoryOpEntry{
		SchemaVersion:      memoryOpsSchemaVersion,
		EventType:          "memory_write_event",
		Target:             "memory/talk_full.md",
		Operation:          operation,
		TalkFullHashBefore: talkFullHashBefore,
		TalkFullHashAfter:  talkFullHashAfter,
		PrevMemoryOpHash:   prevHash,
		MemoryOpHash:       "", // 待計算
		Timestamp:          time.Now().Format(time.RFC3339),
	}

	// ── 計算 memory_op_hash ──
	entry.MemoryOpHash = computeEntryHash(entry)

	// ── 序列化為 JSONL 一行 ──
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("memory_ops: JSON 序列化失敗: %w", err)
	}
	line = append(line, '\n')

	// ── 寫入 memory_ops.jsonl（O_APPEND + 0600）──
	f, err := os.OpenFile(opsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("memory_ops: 開啟檔案失敗: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("memory_ops: 寫入失敗: %w", err)
	}

	return nil
}
