// Package audit_log provides a generic append-only JSONL log abstraction.
//
// Five modules previously implemented near-identical JSONL append logic:
//   - controlled_trust/trust_log.go
//   - remote_bridge/audit.go
//   - review/service.go (writeDecisionLog)
//   - conversation/memory_ops.go
//   - execution_hook/hash_chain.go
//
// This package extracts the common pattern into a single reusable type:
//   AppendLog[T] with Option-based configuration for per-consumer differences.
//
// Differences handled by Options:
//   - File/dir permissions (0600/0700 vs 0644/0755)
//   - Hash chain linking (trust_log, execution_hook)
//   - Before-append validation (trust_log invariant checks)
//   - Corrupt line skip on read (remote_bridge/audit)
// [TASK 8 完成區塊]
// 此 package 為 TASK 8 (TASKS_1_6_3) 新增的共用 JSONL append-only 日誌抽象。
// 取代 5 個消費者的重複邏輯：trust_log / audit / review / memory_ops / hash_chain。
// 泛型 AppendLog[T] + Option 模式處理差異：權限、hash chain、corrupt line skip、validator。
package audit_log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ──────────────────────────────────────────────
// AppendLog — 泛型 append-only JSONL 日誌
// ──────────────────────────────────────────────

// AppendLog is a generic append-only JSONL log file.
// Thread-safe via internal mutex.
type AppendLog[T any] struct {
	mu   sync.Mutex
	path string
	cfg  config[T]
}

// New creates (or re-opens) an AppendLog at the given path.
// The path should include the filename (e.g. "dir/audit.jsonl").
func New[T any](path string, opts ...Option[T]) *AppendLog[T] {
	l := &AppendLog[T]{
		path: path,
		cfg:  defaultConfig[T](),
	}
	for _, opt := range opts {
		opt(&l.cfg)
	}
	// If hash chain is configured, load the tail hash from existing file
	if l.cfg.hashChain != nil {
		_ = l.loadTailHash()
	}
	return l
}

// Append adds a new entry to the log. Thread-safe.
func (l *AppendLog[T]) Append(entry T) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Run before-append hook (e.g. invariant validation)
	if l.cfg.beforeAppend != nil {
		if err := l.cfg.beforeAppend(&entry); err != nil {
			return err
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(l.path), l.cfg.dirPerm); err != nil {
		return fmt.Errorf("audit_log: mkdir: %w", err)
	}

	// Run hash chain hook (sets previous hash / computes entry hash)
	if l.cfg.hashChain != nil {
		l.cfg.hashChain.beforeWrite(&entry, l.cfg.chainState)
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit_log: marshal: %w", err)
	}

	// Append to file
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, l.cfg.filePerm)
	if err != nil {
		return fmt.Errorf("audit_log: open: %w", err)
	}
	defer f.Close()

	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("audit_log: write: %w", err)
	}

	// Update hash chain state after successful write
	if l.cfg.hashChain != nil {
		l.cfg.hashChain.afterWrite(&entry, l.cfg.chainState)
	}

	return nil
}

// ReadAll reads all entries from the log file.
// Returns nil, nil if the file does not exist.
func (l *AppendLog[T]) ReadAll() ([]T, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("audit_log: read: %w", err)
	}

	var entries []T
	for i, line := range SplitLines(data) {
		if len(line) == 0 {
			continue
		}
		var entry T
		if err := json.Unmarshal(line, &entry); err != nil {
			if l.cfg.skipCorruptLines {
				continue // 跳過損毀行
			}
			return nil, fmt.Errorf("audit_log: parse error at line %d: %w", i, err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ReadRecent reads the last n entries from the log.
func (l *AppendLog[T]) ReadRecent(n int) ([]T, error) {
	all, err := l.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

// Path returns the file path of this log.
func (l *AppendLog[T]) Path() string {
	return l.path
}

// loadTailHash reads the last line to initialise hash chain state.
func (l *AppendLog[T]) loadTailHash() error {
	data, err := os.ReadFile(l.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	lines := SplitLines(data)
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) == 0 {
			continue
		}
		var entry T
		if err := json.Unmarshal(lines[i], &entry); err == nil {
			l.cfg.hashChain.loadTail(&entry, l.cfg.chainState)
			return nil
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// SplitLines — 共用 JSONL 行分割
// ──────────────────────────────────────────────

// SplitLines splits JSONL bytes into individual lines.
// Exported so consumers that need raw access can reuse it.
func SplitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
