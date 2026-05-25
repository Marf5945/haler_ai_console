// memory/manifest.go — Memory Manifest + Hash Chain（§18.5）。
// 每筆寫入記錄前一筆的 SHA-256，形成 append-only hash chain。
// ManifestHash 用於 DAG Resume Guard 比對。
package memory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

// ──────────────────────────────────────────────
// Manifest 結構
// ──────────────────────────────────────────────

// MemoryManifest 記憶管線的元資料。
type MemoryManifest struct {
	CurrentHash  string   `json:"current_hash"`   // 當前 hash chain 的最新值
	PreviousHash string   `json:"previous_hash"`  // 上一筆 hash
	EntryCount   int      `json:"entry_count"`     // 累計寫入筆數
	LastRotation string   `json:"last_rotation"`   // 最近一次輪轉時間
	ArchiveFiles []string `json:"archive_files"`   // 歸檔檔案清單
}

// NewManifest 建立空的 manifest。
func NewManifest() *MemoryManifest {
	return &MemoryManifest{
		CurrentHash:  "genesis",
		PreviousHash: "",
		EntryCount:   0,
		ArchiveFiles: []string{},
	}
}

// ──────────────────────────────────────────────
// Hash Chain 操作
// ──────────────────────────────────────────────

// AppendHash 將新寫入的 entry 加入 hash chain。
// newHash = SHA-256(previousHash + entry)
func (m *MemoryManifest) AppendHash(entry string) {
	m.PreviousHash = m.CurrentHash
	combined := m.PreviousHash + entry
	hash := sha256.Sum256([]byte(combined))
	m.CurrentHash = fmt.Sprintf("%x", hash)
	m.EntryCount++
}

// ComputeHash 回傳當前 manifest 的 hash。
// 用於 DAG Resume Guard 比對。
func (m *MemoryManifest) ComputeHash() string {
	return m.CurrentHash
}

// ──────────────────────────────────────────────
// 檔案讀寫
// ──────────────────────────────────────────────

// LoadManifest 從檔案載入 manifest。
func LoadManifest(path string) (*MemoryManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewManifest(), nil
		}
		return nil, err
	}

	var manifest MemoryManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return NewManifest(), nil
	}
	if manifest.ArchiveFiles == nil {
		manifest.ArchiveFiles = []string{}
	}
	return &manifest, nil
}

// SaveManifest 將 manifest 寫入檔案。
func SaveManifest(path string, manifest *MemoryManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
