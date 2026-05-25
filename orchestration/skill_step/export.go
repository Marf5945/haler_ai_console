package skill_step

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportManifest 記錄一次 skill 匯出的詳細資訊：
// 包含哪些檔案被匯出（IncludedFiles），以及哪些 .skill_rel.json 被刻意排除（RemovedRelFiles）。
//
// 安全說明：
// 匯出的套件只包含 skill_manifest.json 與 README.md。
// 所有 programs/、examples/ 內容、可讀文字片段、表單資料、帳號、
// email、token、API key 及 .skill_rel.json 均不得匯出。
// 這樣設計是為了防止敏感資料透過 skill 匯出流出。
type ExportManifest struct {
	ExportedAt      time.Time `json:"exported_at"`       // 匯出時間
	SkillID         string    `json:"skill_id"`          // 匯出的 skill ID
	IncludedFiles   []string  `json:"included_files"`    // 實際複製到目標目錄的檔案清單
	RemovedRelFiles []string  `json:"removed_rel_files"` // 刻意排除的 .skill_rel.json 相對路徑
	ExportHash      string    `json:"export_hash"`       // skill_id + exported_at 的 SHA-256
}

// ExportSkill 將 skillDir 中的 skill 安全匯出到 destDir。
// 只複製 skill_manifest.json 與 README.md；
// .skill_rel.json 會被列入 RemovedRelFiles 但不複製，
// 最後在 destDir 寫入一份 export_manifest.json 作為本次匯出的記錄。
//
// 呼叫端責任：
//   - 確保 skillDir 是合法的已歸檔 skill 目錄（含 skill_manifest.json）
//   - 確保 destDir 已由使用者選擇，且路徑通過邊界檢查
//
// 這個函式不會：
//   - 複製 programs/ 或 examples/ 的實際內容
//   - 複製任何可執行指令、原始文字片段、截圖或敏感資訊
func ExportSkill(skillDir string, destDir string) (*ExportManifest, error) {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return nil, fmt.Errorf("skill_step: export mkdir: %w", err)
	}

	em := &ExportManifest{
		ExportedAt: time.Now(),
	}

	// 讀取來源 manifest 以取得 SkillID（skill 識別依靠 manifest 而非目錄名）
	manifestSrc := filepath.Join(skillDir, "skill_manifest.json")
	m, err := LoadManifest(manifestSrc)
	if err != nil {
		return nil, fmt.Errorf("skill_step: export load manifest: %w", err)
	}
	em.SkillID = m.SkillID

	// 複製 skill_manifest.json——這是匯出的核心，接收端用它來識別 skill
	manifestDst := filepath.Join(destDir, "skill_manifest.json")
	if err := copyFile(manifestSrc, manifestDst); err != nil {
		return nil, fmt.Errorf("skill_step: export copy manifest: %w", err)
	}
	em.IncludedFiles = append(em.IncludedFiles, "skill_manifest.json")

	// 若有 README.md 則一起複製——作為人類可讀說明
	readmeSrc := filepath.Join(skillDir, "README.md")
	if _, err := os.Stat(readmeSrc); err == nil {
		readmeDst := filepath.Join(destDir, "README.md")
		if err := copyFile(readmeSrc, readmeDst); err != nil {
			return nil, fmt.Errorf("skill_step: export copy README: %w", err)
		}
		em.IncludedFiles = append(em.IncludedFiles, "README.md")
	}

	// 走訪整個 skillDir，找出所有 .skill_rel.json 記錄到 RemovedRelFiles。
	// 這些檔案「刻意不複製」——讓接收端知道關聯資訊需要在自己的 Console 重建。
	walkErr := filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略不可讀的條目，不中止整個 walk
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".skill_rel.json") {
			rel, relErr := filepath.Rel(skillDir, path)
			if relErr == nil {
				em.RemovedRelFiles = append(em.RemovedRelFiles, rel)
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("skill_step: export walk: %w", walkErr)
	}

	// 計算 export hash：以 skill_id + RFC3339 時間戳 雜湊，讓每次匯出都有唯一指紋
	h := sha256.New()
	h.Write([]byte(em.SkillID + em.ExportedAt.UTC().Format(time.RFC3339Nano)))
	em.ExportHash = hex.EncodeToString(h.Sum(nil))

	// 在目標目錄寫入 export_manifest.json，作為本次匯出的完整記錄
	data, err := json.MarshalIndent(em, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("skill_step: export marshal: %w", err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "export_manifest.json"), data, 0o600); err != nil {
		return nil, fmt.Errorf("skill_step: export write manifest: %w", err)
	}

	return em, nil
}
