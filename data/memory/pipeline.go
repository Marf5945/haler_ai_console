// memory/pipeline.go — Memory Pipeline 核心管理器（§18）。
// 管理 talk_full → main_memory → deep_memory 三級記憶管線。
// 所有寫入經 redaction 過濾，確保 credential 不外洩。
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// 常數定義
// ──────────────────────────────────────────────

const (
	// 輪轉閾值：500KB（固定大小輪轉策略）
	RotationThresholdBytes = 500 * 1024

	// 核心檔案名
	FileTalkFull   = "talk_full.md"
	FileTalkIndex  = "talk_index.json"
	FileMainMemory = "main_memory.md"
	FileDeepMemory = "deep_memory.md"
	FileDeepThreat = "deep_memory_THREAT.md"
	FileManifest   = "memory_manifest.json"
)

// RotationAction 輪轉動作建議。
type RotationAction string

const (
	RotationNone     RotationAction = "none"     // 無需輪轉
	RotationWarning  RotationAction = "warning"  // 接近閾值（80%）
	RotationRequired RotationAction = "required" // 已達閾值，需立即輪轉
)

// ──────────────────────────────────────────────
// Pipeline 結構
// ──────────────────────────────────────────────

// Pipeline 管理三級記憶管線。
type Pipeline struct {
	mu      sync.Mutex
	rootDir string // data/projects/{project}/memory/
}

// PipelineState 記憶管線的即時狀態。
type PipelineState struct {
	TalkFullSize   int64          `json:"talk_full_size"`
	RotationAction RotationAction `json:"rotation_action"`
	MainMemorySize int64          `json:"main_memory_size"`
	DeepMemorySize int64          `json:"deep_memory_size"`
	ThreatEntries  int            `json:"threat_entries"`
	ManifestHash   string         `json:"manifest_hash"`
	LastRotation   string         `json:"last_rotation"`
}

// NewPipeline 建立記憶管線管理器。
func NewPipeline(projectRoot string) *Pipeline {
	rootDir := filepath.Join(projectRoot, "memory")
	os.MkdirAll(rootDir, 0755)
	return &Pipeline{rootDir: rootDir}
}

// ──────────────────────────────────────────────
// 狀態查詢
// ──────────────────────────────────────────────

// GetState 回傳記憶管線即時狀態。
func (p *Pipeline) GetState() PipelineState {
	p.mu.Lock()
	defer p.mu.Unlock()

	talkSize := fileSize(filepath.Join(p.rootDir, FileTalkFull))
	mainSize := fileSize(filepath.Join(p.rootDir, FileMainMemory))
	deepSize := fileSize(filepath.Join(p.rootDir, FileDeepMemory))

	// 計算威脅條目數
	threatCount := countLines(filepath.Join(p.rootDir, FileDeepThreat))

	// 計算輪轉動作
	action := RotationNone
	if talkSize >= RotationThresholdBytes {
		action = RotationRequired
	} else if talkSize >= RotationThresholdBytes*80/100 {
		action = RotationWarning
	}

	// 讀取 manifest hash
	manifest, _ := LoadManifest(filepath.Join(p.rootDir, FileManifest))
	hash := ""
	if manifest != nil {
		hash = manifest.CurrentHash
	}

	return PipelineState{
		TalkFullSize:   talkSize,
		RotationAction: action,
		MainMemorySize: mainSize,
		DeepMemorySize: deepSize,
		ThreatEntries:  threatCount,
		ManifestHash:   hash,
	}
}

// ──────────────────────────────────────────────
// 寫入（含 redaction）
// ──────────────────────────────────────────────

// AppendTalkEntry 寫入一筆對話記錄到 talk_full.md。
// 自動執行 redaction，確保 credential 不外洩。
// 回傳 redaction 記錄（若有）。
//
// 內部流程（v4.0 雙鏈 hash 整合）：
//  1. Lock
//  2. Redaction
//  3. talk_full_hash_before = SHA-256(read talk_full.md)
//  4. Append talk_full.md（權限 0600）
//  5. talk_full_hash_after = SHA-256(read talk_full.md)
//  6. 讀取 memory_ops.jsonl 最後一行 → prev_memory_op_hash
//  7. 建立 MemoryOpEntry + 計算 memory_op_hash
//  8. Append memory_ops.jsonl（O_APPEND|O_CREATE|O_WRONLY, 0600）
//  9. 更新 manifest
//  10. 回傳 redaction records
//
// 安全性：memory_ops 寫入失敗 → 回傳 error（fail closed）
func (p *Pipeline) AppendTalkEntry(role, text string) ([]RedactionRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// ── Step 1: Redaction 過濾 ──
	cleaned, records := RedactBeforeWrite(text)

	// 格式化 entry
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n## [%s] %s\n%s\n", timestamp, role, cleaned)

	talkPath := filepath.Join(p.rootDir, FileTalkFull)

	// ── Step 2: talk_full_hash_before（寫入前 SHA-256）──
	hashBefore, err := computeFileSHA256(talkPath)
	if err != nil {
		return nil, fmt.Errorf("計算 talk_full hash_before 失敗: %w", err)
	}

	// ── Step 3: Append 到 talk_full.md（權限改為 0600 — 本機隱私資料）──
	f, err := os.OpenFile(talkPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("寫入 talk_full 失敗: %w", err)
	}
	if _, err := f.WriteString(entry); err != nil {
		f.Close()
		return nil, fmt.Errorf("寫入 talk_full 失敗: %w", err)
	}
	f.Close()

	// ── Step 4: talk_full_hash_after（寫入後 SHA-256）──
	hashAfter, err := computeFileSHA256(talkPath)
	if err != nil {
		return nil, fmt.Errorf("計算 talk_full hash_after 失敗: %w", err)
	}

	// ── Step 5: 寫入 memory_ops.jsonl（雙鏈 hash）──
	// Fail closed: 若寫入失敗，AppendTalkEntry 整體回傳 error
	if err := appendMemoryOp(p.rootDir, "append", hashBefore, hashAfter); err != nil {
		return nil, fmt.Errorf("memory_ops 寫入失敗 (fail closed): %w", err)
	}

	// ── Step 6: 更新 manifest hash chain ──
	p.updateManifestAfterWrite(entry)

	return records, nil
}

// RewriteTalkFullForDelete 以授權的 delete_sentences 操作整檔重寫 talk_full.md。
// 用於使用者刪除對話：走正常 pipeline（hash_before/after + memory_ops + manifest），
// 因此完整性驗證視為「合法變更」而非繞過竄改；舊對話資料真正從檔案移除。
// newContent 必須是已移除目標條目、其餘原樣保留的完整檔案內容。
func (p *Pipeline) RewriteTalkFullForDelete(newContent string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	talkPath := filepath.Join(p.rootDir, FileTalkFull)

	// Step 1: 寫入前 SHA-256
	hashBefore, err := computeFileSHA256(talkPath)
	if err != nil {
		return fmt.Errorf("計算 talk_full hash_before 失敗: %w", err)
	}
	// Step 2: 整檔重寫（0600 — 本機隱私資料）
	if err := os.WriteFile(talkPath, []byte(newContent), 0o600); err != nil {
		return fmt.Errorf("重寫 talk_full 失敗: %w", err)
	}
	// Step 3: 寫入後 SHA-256
	hashAfter, err := computeFileSHA256(talkPath)
	if err != nil {
		return fmt.Errorf("計算 talk_full hash_after 失敗: %w", err)
	}
	// Step 4: memory_ops 記一筆 delete_sentences（fail closed）
	if err := appendMemoryOp(p.rootDir, "delete_sentences", hashBefore, hashAfter); err != nil {
		return fmt.Errorf("memory_ops 寫入失敗 (fail closed): %w", err)
	}
	// Step 5: 推進 manifest hash chain（以 delete 標記 + 新 hash 當作 entry）
	p.updateManifestAfterWrite("delete_sentences\n" + hashAfter)
	return nil
}

// ──────────────────────────────────────────────
// 輪轉（固定大小策略）
// ──────────────────────────────────────────────

// CheckRotation 檢查是否需要輪轉。
func (p *Pipeline) CheckRotation() RotationAction {
	size := fileSize(filepath.Join(p.rootDir, FileTalkFull))
	if size >= RotationThresholdBytes {
		return RotationRequired
	}
	if size >= RotationThresholdBytes*80/100 {
		return RotationWarning
	}
	return RotationNone
}

// Rotate 執行 talk_full.md 輪轉。
// 將現有檔案歸檔為 talk_full_YYYYMMDD_HHMMSS.md，建立新的空檔案。
func (p *Pipeline) Rotate() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	srcPath := filepath.Join(p.rootDir, FileTalkFull)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", nil // 沒有檔案需要輪轉
	}

	// 歸檔檔名
	archiveName := fmt.Sprintf("talk_full_%s.md", time.Now().Format("20060102_150405"))
	archiveDir := filepath.Join(p.rootDir, "archive")
	os.MkdirAll(archiveDir, 0755)
	archivePath := filepath.Join(archiveDir, archiveName)

	// 移動檔案
	if err := os.Rename(srcPath, archivePath); err != nil {
		return "", fmt.Errorf("輪轉歸檔失敗: %w", err)
	}

	// 建立新的空 talk_full.md
	header := fmt.Sprintf("# Talk Full — 新建於 %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if err := os.WriteFile(srcPath, []byte(header), 0o600); err != nil {
		return "", fmt.Errorf("建立新 talk_full 失敗: %w", err)
	}

	// 更新 manifest
	manifest, _ := LoadManifest(filepath.Join(p.rootDir, FileManifest))
	if manifest != nil {
		manifest.LastRotation = time.Now().Format(time.RFC3339)
		manifest.ArchiveFiles = append(manifest.ArchiveFiles, archiveName)
		SaveManifest(filepath.Join(p.rootDir, FileManifest), manifest)
	}

	return archivePath, nil
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

// updateManifestAfterWrite 寫入後更新 manifest hash chain。
func (p *Pipeline) updateManifestAfterWrite(entry string) {
	manifestPath := filepath.Join(p.rootDir, FileManifest)
	manifest, err := LoadManifest(manifestPath)
	if err != nil || manifest == nil {
		manifest = NewManifest()
	}
	manifest.AppendHash(entry)
	SaveManifest(manifestPath, manifest)
}

// fileSize 回傳檔案大小，不存在時回傳 0。
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// countLines 計算檔案行數，不存在時回傳 0。
func countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	return count
}
