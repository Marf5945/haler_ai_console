// project_lifecycle/purge.go — Project Lifecycle Purge（§7）。
// 清除專案暫存資料，釋放 USB 空間。
// 策略：分類自動（安全類自動清、禁止類跳過、邊界類產生 Review Card）。
// 觸發時機：使用者歸檔/刪除專案 + 啟動時掃描 30 天過期暫存。
package project_lifecycle

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// 清除分類定義
// ──────────────────────────────────────────────

// PurgeCategory 資料類別的清除策略。
type PurgeCategory string

const (
	CategoryAutoSafe  PurgeCategory = "auto_safe"       // 可安全自動清除
	CategoryForbidden PurgeCategory = "forbidden"       // 禁止清除
	CategoryBoundary  PurgeCategory = "boundary_review" // 需 Review Card 確認
)

// PurgeTrigger 清除觸發原因。
type PurgeTrigger string

const (
	TriggerUserArchives PurgeTrigger = "user_archives_project"
	TriggerUserDeletes  PurgeTrigger = "user_deletes_project"
	TriggerStartupScan  PurgeTrigger = "startup_expiry_scan"
)

// ──────────────────────────────────────────────
// 可清除 / 禁止清除路徑定義（§7.3）
// ──────────────────────────────────────────────

// autoSafeDirs 可安全自動清除的目錄（相對於 projectRoot）。
var autoSafeDirs = []string{
	"runtime/temp_sessions",
	"runtime/action_results",
	"runtime/crash_recovery",
}

// forbiddenDirs 禁止清除的目錄/檔案。
var forbiddenDirs = []string{
	"review/review_decision_log.jsonl",
	"review/security_change_log.jsonl",
	"controlled_trust/controlled_trust_log.jsonl",
	"memory/main_memory.md",
	"memory/deep_memory.md",
	"memory/deep_memory_THREAT.md",
	"memory/memory_manifest.json",
	"source_trust/source_trust_log.jsonl",
	"source_trust/project_source_allowlist.json",
}

// boundaryDirs 邊界類，需使用者確認。
var boundaryDirs = []string{
	"controlled_trust/draft_sandbox_runs",
	"visual_learning/processed_temp",
	"dag_runs",
}

// ──────────────────────────────────────────────
// Purge Service
// ──────────────────────────────────────────────

// Service 管理專案生命週期清除。
type Service struct {
	projectRoot string
}

// NewService 建立 purge service。
func NewService(projectRoot string) *Service {
	return &Service{projectRoot: projectRoot}
}

// ──────────────────────────────────────────────
// 完整 Purge（使用者歸檔/刪除時觸發）
// ──────────────────────────────────────────────

// PurgeResult 清除操作結果。
type PurgeResult struct {
	Manifest    *PurgeManifest `json:"manifest"`
	AutoCleaned []string       `json:"auto_cleaned"` // 自動清除的路徑
	Skipped     []string       `json:"skipped"`      // 禁止清除而跳過的路徑
	NeedReview  []string       `json:"need_review"`  // 需 Review Card 確認的路徑
}

// BackupEntry 記錄單一備份項目。
type BackupEntry struct {
	SourcePath string `json:"source_path"`
	BackupPath string `json:"backup_path"`
	Size       int64  `json:"size"`
	Action     string `json:"action"`
}

// PurgeBackupManifest 記錄清除前的最小備份。
type PurgeBackupManifest struct {
	BackupID  string        `json:"backup_id"`
	ProjectID string        `json:"project_id"`
	Timestamp string        `json:"timestamp"`
	Root      string        `json:"root"`
	Entries   []BackupEntry `json:"entries"`
}

// Purge 執行完整清除。
// 自動清除安全類、跳過禁止類、回傳邊界類供呼叫端產生 Review Card。
func (s *Service) Purge(projectID string, trigger PurgeTrigger) (*PurgeResult, error) {
	result := &PurgeResult{}
	var cleaned []PurgeEntry

	// 1. 自動清除安全類
	for _, dir := range autoSafeDirs {
		fullPath := filepath.Join(s.projectRoot, dir)
		removed, size := removeDir(fullPath)
		if removed {
			result.AutoCleaned = append(result.AutoCleaned, dir)
			cleaned = append(cleaned, PurgeEntry{
				Path:     dir,
				Category: string(CategoryAutoSafe),
				Size:     size,
				Action:   "removed",
			})
		}
	}

	// 2. 標記禁止類
	for _, path := range forbiddenDirs {
		fullPath := filepath.Join(s.projectRoot, path)
		if _, err := os.Stat(fullPath); err == nil {
			result.Skipped = append(result.Skipped, path)
			cleaned = append(cleaned, PurgeEntry{
				Path:     path,
				Category: string(CategoryForbidden),
				Action:   "preserved",
			})
		}
	}

	// 3. 標記邊界類（由呼叫端決定是否產生 Review Card）
	for _, dir := range boundaryDirs {
		fullPath := filepath.Join(s.projectRoot, dir)
		if _, err := os.Stat(fullPath); err == nil {
			result.NeedReview = append(result.NeedReview, dir)
			cleaned = append(cleaned, PurgeEntry{
				Path:     dir,
				Category: string(CategoryBoundary),
				Action:   "pending_review",
			})
		}
	}

	// 4. 生成 PurgeManifest
	manifest := NewPurgeManifest(projectID, trigger, cleaned)
	if err := manifest.Save(s.projectRoot); err != nil {
		return nil, fmt.Errorf("寫入 PurgeManifest 失敗: %w", err)
	}
	result.Manifest = manifest

	return result, nil
}

// BackupAutoSafeDirs 只備份即將被專案清除移除的安全暫存目錄。
func (s *Service) BackupAutoSafeDirs(projectID string) (*PurgeBackupManifest, error) {
	backupID := fmt.Sprintf("purge-backup-%d", time.Now().UnixNano())
	backupRoot := filepath.Join(s.projectRoot, "purge_backups", backupID)
	manifest := &PurgeBackupManifest{
		BackupID:  backupID,
		ProjectID: projectID,
		Timestamp: time.Now().Format(time.RFC3339),
		Root:      backupRoot,
	}

	for _, rel := range autoSafeDirs {
		src := filepath.Join(s.projectRoot, rel)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(backupRoot, rel)
		size, err := copyPath(src, dst)
		if err != nil {
			return nil, fmt.Errorf("備份 %s 失敗: %w", rel, err)
		}
		manifest.Entries = append(manifest.Entries, BackupEntry{
			SourcePath: rel,
			BackupPath: filepath.Join("purge_backups", backupID, rel),
			Size:       size,
			Action:     "backed_up",
		})
	}

	if err := saveBackupManifest(manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

// PurgeBoundaryDir 清除使用者已確認的邊界類目錄。
// 僅接受 boundaryDirs 中的路徑，防止誤刪。
func (s *Service) PurgeBoundaryDir(dir string) error {
	allowed := false
	for _, bd := range boundaryDirs {
		if bd == dir {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("不允許清除此路徑: %s", dir)
	}

	fullPath := filepath.Join(s.projectRoot, dir)
	return os.RemoveAll(fullPath)
}

// ──────────────────────────────────────────────
// 啟動時過期掃描（30 天自動清理）
// ──────────────────────────────────────────────

// ScanExpiredResult 掃描結果。
type ScanExpiredResult struct {
	Cleaned   []string `json:"cleaned"`    // 已清除的過期項目
	TotalSize int64    `json:"total_size"` // 釋放的總大小（bytes）
}

// ScanAndCleanExpired 掃描並清除超過 maxAge 天的暫存資料。
// 僅清除 autoSafeDirs 中的過期檔案，不觸碰禁止或邊界類。
func (s *Service) ScanAndCleanExpired(maxAgeDays int) (*ScanExpiredResult, error) {
	if maxAgeDays <= 0 {
		maxAgeDays = 30
	}

	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	result := &ScanExpiredResult{}

	for _, dir := range autoSafeDirs {
		fullPath := filepath.Join(s.projectRoot, dir)
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			continue // 目錄不存在就跳過
		}

		for _, entry := range entries {
			entryPath := filepath.Join(fullPath, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// 檢查修改時間是否超過 cutoff
			if info.ModTime().Before(cutoff) {
				size := info.Size()
				if entry.IsDir() {
					size = dirSize(entryPath)
				}

				if err := os.RemoveAll(entryPath); err == nil {
					result.Cleaned = append(result.Cleaned, filepath.Join(dir, entry.Name()))
					result.TotalSize += size
				}
			}
		}
	}

	return result, nil
}

// ──────────────────────────────────────────────
// 分類查詢
// ──────────────────────────────────────────────

// ClassifyPath 判斷指定路徑的清除分類。
func ClassifyPath(relativePath string) PurgeCategory {
	for _, dir := range autoSafeDirs {
		if strings.HasPrefix(relativePath, dir) {
			return CategoryAutoSafe
		}
	}
	for _, path := range forbiddenDirs {
		if strings.HasPrefix(relativePath, path) {
			return CategoryForbidden
		}
	}
	for _, dir := range boundaryDirs {
		if strings.HasPrefix(relativePath, dir) {
			return CategoryBoundary
		}
	}
	return CategoryForbidden // 未知路徑預設禁止清除
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

// removeDir 移除目錄，回傳是否成功移除 + 原始大小。
func removeDir(path string) (bool, int64) {
	info, err := os.Stat(path)
	if err != nil {
		return false, 0
	}
	size := int64(0)
	if info.IsDir() {
		size = dirSize(path)
	} else {
		size = info.Size()
	}

	if err := os.RemoveAll(path); err != nil {
		return false, 0
	}
	return true, size
}

func copyPath(src, dst string) (int64, error) {
	info, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return copyFile(src, dst, info.Mode())
	}

	var total int64
	err = filepath.Walk(src, func(path string, walkInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if walkInfo.IsDir() {
			return os.MkdirAll(target, walkInfo.Mode())
		}
		size, err := copyFile(path, target, walkInfo.Mode())
		total += size
		return err
	})
	return total, err
}

func copyFile(src, dst string, mode os.FileMode) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	return io.Copy(out, in)
}

func saveBackupManifest(m *PurgeBackupManifest) error {
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.Root, "backup_manifest.json"), data, 0o600)
}

// dirSize 計算目錄總大小。
func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}
