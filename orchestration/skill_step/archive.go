package skill_step

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ArchiveService 管理 data/skills/ 目錄下所有已歸檔的 skill。
// 它是 skill 生命週期的核心服務：
//   - 掃描外部資料夾（ScanFolder）→ 產生 ScanPreview 供使用者確認
//   - 確認後歸檔（ConfirmArchive）→ 在 data/skills/<skill_id>/ 建立標準結構
//   - 列出所有已歸檔的 skill（ListArchived）→ 供 Router 查詢
//
// 重要安全規則：
//   - skill 不得安裝到 ~/.claude、~/.codex、~/.gemini 等外部 CLI 目錄
//   - 外部 CLI skill 必須先 quarantine → 分析 → normalize，不得直接使用
//   - ConfirmArchive 必須等使用者明確確認後才能呼叫
type ArchiveService struct {
	mu       sync.Mutex              // 保護 previews 與磁碟寫入的並發安全
	dataRoot string                  // app data 根目錄，例如 ~/.config/ai-console
	previews map[string]*ScanPreview // 暫存的掃描預覽，key 為 PreviewID
}

// NewArchiveService 建立一個 ArchiveService，根目錄設定為 dataRoot。
// data/skills/ 目錄會在第一次使用時惰性建立，不在這裡預先建立。
func NewArchiveService(dataRoot string) *ArchiveService {
	return &ArchiveService{
		dataRoot: dataRoot,
		previews: make(map[string]*ScanPreview),
	}
}

// skillsDir 回傳指定 skillID 的歸檔目錄路徑。
// 結構為：dataRoot/data/skills/<skill_id>/
func (svc *ArchiveService) skillsDir(skillID string) string {
	return filepath.Join(svc.dataRoot, "data", "skills", skillID)
}

// ScanFolder 掃描 sourcePath 目錄，將結果暫存為 ScanPreview 並回傳。
// Preview 只存在記憶體中，不會寫入磁碟。
// 呼叫 ConfirmArchive 並傳入 PreviewID 才會正式歸檔。
func (svc *ArchiveService) ScanFolder(sourcePath string) (*ScanPreview, error) {
	preview, err := ScanSkillFolder(sourcePath)
	if err != nil {
		return nil, err
	}
	// 將 preview 存入快取，供後續 ConfirmArchive 查詢
	svc.mu.Lock()
	svc.previews[preview.PreviewID] = preview
	svc.mu.Unlock()
	return preview, nil
}

// GetPreview 依 previewID 取得已快取的 ScanPreview。
// 若 previewID 不存在（例如已過期或從未掃描），回傳 false。
func (svc *ArchiveService) GetPreview(previewID string) (*ScanPreview, bool) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	p, ok := svc.previews[previewID]
	return p, ok
}

// ConfirmArchive 將 preview 所描述的 skill 正式寫入 data/skills/<skill_id>/。
// 這是唯一可以將 skill 寫入本機知識庫的函式，必須在使用者明確確認後才能呼叫。
//
// 歸檔流程：
//  1. 建立 data/skills/<skill_id>/ 目錄
//  2. 複製 README.md（若不存在則寫入最小版本）
//  3. 依分類將各資源複製到對應子目錄，並在每個子目錄寫入 .skill_rel.json
//  4. 建立並儲存 skill_manifest.json
//
// NeedsReview 的資源會被跳過，不會寫入歸檔（它們停留在 Review Card 等候）。
func (svc *ArchiveService) ConfirmArchive(preview *ScanPreview) (*SkillManifest, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	skillDir := svc.skillsDir(preview.SkillID)
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		return nil, fmt.Errorf("skill_step: create skill dir: %w", err)
	}

	// 複製 README.md；若來源沒有，寫入一個最小版本，確保 skill 目錄完整
	readmeSrc := filepath.Join(preview.SourcePath, "README.md")
	readmeDst := filepath.Join(skillDir, "README.md")
	if _, err := os.Stat(readmeSrc); err == nil {
		if err := copyFile(readmeSrc, readmeDst); err != nil {
			return nil, fmt.Errorf("skill_step: copy README: %w", err)
		}
	} else {
		minimal := fmt.Sprintf("# %s\n\nArchived by AI Console Skill Context Orchestration.\n", preview.DisplayName)
		if err := os.WriteFile(readmeDst, []byte(minimal), 0o600); err != nil {
			return nil, fmt.Errorf("skill_step: write README: %w", err)
		}
	}

	// 建立 manifest 骨架，稍後補入 Resources 欄位
	manifest := &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       preview.SkillID,
		DisplayName:   preview.DisplayName,
		Version:       "1.0.0",
		Source: SkillSource{
			SourceType:       SourceFolderScan,
			OriginalPathHash: hashPathString(preview.SourcePath), // 儲存路徑雜湊，不儲存明文路徑
		},
		Routing: SkillRouting{
			MinimumAutoScore: 0.85, // 預設自動選取門檻
		},
	}

	var exampleIDs, programIDs, cliMdIDs []string

	for _, res := range preview.Resources {
		// 跳過需要人工確認的資源，它們不應被自動歸檔
		if res.NeedsReview {
			continue
		}

		// 使用 UnixNano 確保每個資源取得唯一 ID
		resourceID := fmt.Sprintf("res-%d", time.Now().UnixNano())
		var subDir string
		var resType string

		// 依分類決定子目錄與資源類型字串
		switch res.Classification {
		case ClassExample:
			subDir = filepath.Join(skillDir, "examples", resourceID)
			resType = "example"
			exampleIDs = append(exampleIDs, resourceID)
		case ClassProgram:
			subDir = filepath.Join(skillDir, "programs", resourceID)
			resType = "program"
			programIDs = append(programIDs, resourceID)
		case ClassCLIMd:
			subDir = filepath.Join(skillDir, "cli_md", resourceID)
			resType = "cli_md"
			cliMdIDs = append(cliMdIDs, resourceID)
		default:
			continue // 理論上不會到這裡，NeedsReview 已在上面過濾
		}

		if err := os.MkdirAll(subDir, 0o700); err != nil {
			return nil, fmt.Errorf("skill_step: create resource dir: %w", err)
		}

		// 將資源檔案或目錄複製到子目錄
		dstPath := filepath.Join(subDir, res.Name)
		info, statErr := os.Stat(res.Path)
		if statErr == nil {
			if info.IsDir() {
				if err := copyDir(res.Path, dstPath); err != nil {
					return nil, fmt.Errorf("skill_step: copy resource dir: %w", err)
				}
			} else {
				if err := copyFile(res.Path, dstPath); err != nil {
					return nil, fmt.Errorf("skill_step: copy resource file: %w", err)
				}
			}
		}

		// 在資源子目錄寫入 .skill_rel.json，讓日後可以透過 ID 找到這個資源
		rel := &SkillRelation{
			SchemaVersion: "skill_relation.v1",
			SkillID:       preview.SkillID,
			ResourceID:    resourceID,
			ResourceType:  resType,
		}
		if err := SaveRelation(subDir, rel); err != nil {
			return nil, err
		}
	}

	// 將所有資源 ID 填回 manifest
	manifest.Resources = SkillResources{
		Examples: exampleIDs,
		Programs: programIDs,
		CLIMd:    cliMdIDs,
	}

	// 寫入 skill_manifest.json（SaveManifest 會自動計算 Hash）
	if err := SaveManifest(skillDir, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// ListArchived 讀取 data/skills/ 下所有子目錄的 skill_manifest.json，
// 回傳所有已成功歸檔的 skill 清單。
// 若某個子目錄的 manifest 損毀或不存在，該目錄會被靜默跳過，不影響其他 skill。
func (svc *ArchiveService) ListArchived() ([]SkillManifest, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	skillsRoot := filepath.Join(svc.dataRoot, "data", "skills")
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			// 尚未歸檔過任何 skill，回傳空清單而非錯誤
			return []SkillManifest{}, nil
		}
		return nil, fmt.Errorf("skill_step: list archived: %w", err)
	}

	var manifests []SkillManifest
	for _, e := range entries {
		if !e.IsDir() {
			continue // 只處理子目錄，忽略散落的檔案
		}
		mPath := filepath.Join(skillsRoot, e.Name(), "skill_manifest.json")
		m, err := LoadManifest(mPath)
		if err != nil {
			continue // manifest 損毀或不存在時靜默跳過
		}
		manifests = append(manifests, *m)
	}
	return manifests, nil
}

// ---------------------------------------------------------------------------
// 內部輔助函式
// ---------------------------------------------------------------------------

// hashPathString 計算任意字串的 SHA-256 摘要。
// 用於將原始路徑雜湊後存入 manifest，
// 避免在歸檔記錄中暴露使用者的本機路徑資訊。
func hashPathString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// copyFile 將 src 的內容逐位元組複製到 dst。
// dst 的父目錄若不存在會自動建立（0700）。
// 目標檔案權限設定為 0600，防止其他使用者讀取。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir 遞迴複製 src 目錄到 dst。
// 子目錄會以 0700 建立，檔案以 copyFile 複製（0600）。
// 若中途有檔案無法讀取，Walk 會中止並回傳錯誤。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		return copyFile(path, target)
	})
}
