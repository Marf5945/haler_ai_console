// project_lifecycle/manifest.go — PurgeManifest（§7.5）。
// 每次 purge 必寫 manifest，記錄清除了什麼、保留了什麼。
// 用於審計追蹤和問題回溯。
package project_lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ──────────────────────────────────────────────
// Manifest 結構
// ──────────────────────────────────────────────

// PurgeManifest 清除操作的完整記錄。
type PurgeManifest struct {
	ManifestID string       `json:"manifest_id"`
	ProjectID  string       `json:"project_id"`
	Trigger    PurgeTrigger `json:"trigger"`
	Timestamp  string       `json:"timestamp"`
	Entries    []PurgeEntry `json:"entries"`
	Summary    PurgeSummary `json:"summary"`
}

// PurgeEntry 單一清除項目記錄。
type PurgeEntry struct {
	Path     string `json:"path"`
	Category string `json:"category"` // auto_safe, forbidden, boundary_review
	Size     int64  `json:"size"`     // bytes（清除前大小）
	Action   string `json:"action"`   // removed, preserved, pending_review
}

// PurgeSummary 清除摘要。
type PurgeSummary struct {
	TotalRemoved   int   `json:"total_removed"`
	TotalPreserved int   `json:"total_preserved"`
	TotalPending   int   `json:"total_pending"`
	FreedBytes     int64 `json:"freed_bytes"`
}

// ──────────────────────────────────────────────
// Manifest 建立
// ──────────────────────────────────────────────

// NewPurgeManifest 建立清除 manifest。
func NewPurgeManifest(projectID string, trigger PurgeTrigger, entries []PurgeEntry) *PurgeManifest {
	summary := PurgeSummary{}
	for _, e := range entries {
		switch e.Action {
		case "removed":
			summary.TotalRemoved++
			summary.FreedBytes += e.Size
		case "preserved":
			summary.TotalPreserved++
		case "pending_review":
			summary.TotalPending++
		}
	}

	return &PurgeManifest{
		ManifestID: fmt.Sprintf("purge-%d", time.Now().UnixNano()),
		ProjectID:  projectID,
		Trigger:    trigger,
		Timestamp:  time.Now().Format(time.RFC3339),
		Entries:    entries,
		Summary:    summary,
	}
}

// ──────────────────────────────────────────────
// 檔案讀寫
// ──────────────────────────────────────────────

// Save 將 manifest 寫入專案目錄。
func (m *PurgeManifest) Save(projectRoot string) error {
	dir := filepath.Join(projectRoot, "purge_manifests")
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", m.ManifestID)
	return os.WriteFile(filepath.Join(dir, filename), data, 0o600)
}

// ListManifests 列出所有清除 manifest。
func ListManifests(projectRoot string) ([]PurgeManifest, error) {
	dir := filepath.Join(projectRoot, "purge_manifests")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var manifests []PurgeManifest
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var m PurgeManifest
		if err := json.Unmarshal(data, &m); err == nil {
			manifests = append(manifests, m)
		}
	}
	return manifests, nil
}

// LoadManifest 載入指定 manifest。
func LoadManifest(projectRoot, manifestID string) (*PurgeManifest, error) {
	path := filepath.Join(projectRoot, "purge_manifests", manifestID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m PurgeManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
