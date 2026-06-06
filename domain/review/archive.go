// Package review — archive.go
//
// v3.6.1 升級：ArchivedCard 新增 RiskClass 欄位，
// security_boundary_rewrite 額外寫入 security_change_log.jsonl（§5.4）。
//
// review_archive.json 是 append-only 語意的 JSON 陣列，
// 每張卡片包含 status（resolved / rejected）與 reject_reason。
package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ui_console/domain/risk"
)

// ArchivedCard 是寫入 review_archive.json 的持久化卡片。
type ArchivedCard struct {
	ID             string         `json:"id"`
	RiskClass      risk.RiskClass `json:"risk_class"`         // v3.6.1 新增
	Level          Level          `json:"level"`               // 向下相容
	Status         string         `json:"status"`              // "resolved" | "rejected"
	SourceType     string         `json:"source_type"`
	SourceID       string         `json:"source_id"`
	PlainReason    string         `json:"plain_reason"`
	EngineerReason string         `json:"engineer_reason"`
	RejectReason   string         `json:"reject_reason,omitempty"`
	CreatedAt      string         `json:"created_at"`
	ArchivedAt     string         `json:"archived_at"`
}

// ArchiveService 管理 review_archive.json 的讀寫。
type ArchiveService struct {
	mu          sync.Mutex
	archivePath string
}

// NewArchiveService 建立 archive 服務。dataRoot 是應用程式的資料根目錄。
func NewArchiveService(dataRoot string) *ArchiveService {
	return &ArchiveService{
		archivePath: filepath.Join(dataRoot, "review", "review_archive.json"),
	}
}

// AppendRejected 將一張 rejected 狀態的卡片寫入 review_archive.json。
// 用於 Package 安裝被拒絕或 zip-slip/symlink 攔截時留下審計軌跡。
func (a *ArchiveService) AppendRejected(sourceType, sourceID, plainReason, engineerReason, rejectReason string) (*ArchivedCard, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	card := ArchivedCard{
		ID:             generateID("arc"),
		Level:          LevelBackground,
		Status:         "rejected",
		SourceType:     sourceType,
		SourceID:       sourceID,
		PlainReason:    plainReason,
		EngineerReason: engineerReason,
		RejectReason:   rejectReason,
		CreatedAt:      time.Now().Format(time.RFC3339),
		ArchivedAt:     time.Now().Format(time.RFC3339),
	}

	return &card, a.appendLocked(card)
}

// ListArchived 讀取 review_archive.json 中所有已封存的卡片。
func (a *ArchiveService) ListArchived() ([]ArchivedCard, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := os.ReadFile(a.archivePath)
	if os.IsNotExist(err) {
		return []ArchivedCard{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cards []ArchivedCard
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// appendLocked 將卡片追加到 review_archive.json（JSON 陣列格式）。
func (a *ArchiveService) appendLocked(card ArchivedCard) error {
	if err := os.MkdirAll(filepath.Dir(a.archivePath), 0o700); err != nil {
		return err
	}

	var archive []ArchivedCard
	if data, err := os.ReadFile(a.archivePath); err == nil {
		_ = json.Unmarshal(data, &archive)
	}
	archive = append(archive, card)

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.archivePath, data, 0o600)
}
