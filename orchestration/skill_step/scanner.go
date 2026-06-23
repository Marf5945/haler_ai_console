package skill_step

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ResourceClassification 描述掃描時對單一檔案或目錄的分類結果。
// 分類決定了歸檔時這個資源會被放進哪個子目錄（examples/、programs/、cli_md/）。
type ResourceClassification string

const (
	// ClassExample 表示使用範例或操作紀錄，歸入 examples/。
	ClassExample ResourceClassification = "example"
	// ClassProgram 表示可執行的程式或腳本，歸入 programs/。
	ClassProgram ResourceClassification = "program"
	// ClassCLIMd 表示給 CLI 閱讀的 Markdown 摘要，歸入 cli_md/。
	ClassCLIMd ResourceClassification = "cli_md"
	// ClassUnknown 表示無法自動分類，會在 Review Card 中標記 needs_review
	// 讓使用者手動確認，而不是直接拒絕或猜測。
	ClassUnknown ResourceClassification = "needs_review"
)

// ScannedResource 代表掃描時找到的一個檔案或子目錄。
// NeedsReview 為 true 時，這個資源不會被自動歸檔，
// 會留在 Review Card 中等待使用者決定如何處理。
type ScannedResource struct {
	Path           string                 // 來源的完整絕對路徑
	Name           string                 // 檔案或目錄名稱
	Classification ResourceClassification // 自動分類結果
	NeedsReview    bool                   // true 表示分類不確定，需要人工確認
}

// ScanPreview 是「掃描但尚未歸檔」的暫存結果。
// UI 會把這份預覽展示給使用者確認，確認後才呼叫 ConfirmArchive 真正寫入磁碟。
// PreviewID 是暫存快取的鍵值，由 ArchiveService 負責管理生命週期。
type ScanPreview struct {
	PreviewID   string            // 用於後續 ConfirmArchive 的暫存 ID
	SourcePath  string            // 被掃描的原始資料夾路徑
	SkillID     string            // 從資料夾名稱衍生的 skill ID（已 sanitize）
	DisplayName string            // 原始資料夾名稱，用於 UI 顯示
	Resources   []ScannedResource // 掃描到的所有資源
	HasManifest bool              // true 表示已有 skill_manifest.json，是 Console-native skill
	ScannedAt   time.Time         // 掃描時間戳記
	Frontmatter *SkillFrontmatter // 若來源含 SKILL.md frontmatter，解析後的中繼資料（否則為 nil）
	Description string            // 內嵌 manifest 的描述（拖回安裝時保留）
	Embedded    *SkillManifest    // 來源資料夾已含的 skill_manifest.json；拖回安裝時用它保留原始身分
}

// nonAlnum 是用來 sanitize skill ID 的正規表示式：
// 把所有非英文小寫字母與數字的字元替換為連字號。
var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// sanitizeSkillID 將任意資料夾名稱轉換成合法的 skill ID。
// 規則：全部轉小寫 → 非字母數字替換為「-」→ 移除首尾連字號。
// 若結果為空字串（例如全部是特殊字元），回傳預設值 "skill"。
func sanitizeSkillID(name string) string {
	lower := strings.ToLower(name)
	id := nonAlnum.ReplaceAllString(lower, "-")
	id = strings.Trim(id, "-")
	if id == "" {
		id = "skill"
	}
	return id
}

// ScanSkillFolder 掃描 sourcePath 目錄，對每個檔案或子目錄進行分類，
// 並回傳一份 ScanPreview 供使用者確認後再正式歸檔。
//
// 分類邏輯：
//   - skill_manifest.json → 不列為資源，但設定 HasManifest = true
//   - 目錄名為 examples / programs / cli_md → 對應的 ClassExample / ClassProgram / ClassCLIMd
//   - 其他目錄 → ClassUnknown（NeedsReview）
//   - SKILL.md / README.md / 任何 .md 檔 → ClassCLIMd
//   - .go / .py / .sh / .js 檔 → ClassProgram
//   - 其他檔案 → ClassUnknown（NeedsReview）
//
// 注意：這個函式只做「讀取與分類」，不會寫入任何檔案。
func ScanSkillFolder(sourcePath string) (*ScanPreview, error) {
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("skill_step: scan folder: %w", err)
	}

	folderName := filepath.Base(sourcePath)
	skillID := sanitizeSkillID(folderName)

	preview := &ScanPreview{
		PreviewID:   fmt.Sprintf("preview-%d", time.Now().UnixNano()),
		SourcePath:  sourcePath,
		SkillID:     skillID,
		DisplayName: folderName,
		ScannedAt:   time.Now(),
	}

	for _, e := range entries {
		name := e.Name()
		fullPath := filepath.Join(sourcePath, name)

		// skill_manifest.json 本身不是一個「資源」，只是標記這個 skill 是否已有 manifest
		if name == "skill_manifest.json" {
			preview.HasManifest = true
			continue
		}

		var res ScannedResource
		res.Path = fullPath
		res.Name = name

		if e.IsDir() {
			// 依目錄名稱判斷資源類別
			switch strings.ToLower(name) {
			case "examples":
				res.Classification = ClassExample
			case "programs":
				res.Classification = ClassProgram
			case "cli_md":
				res.Classification = ClassCLIMd
			default:
				// 未知目錄：標記 NeedsReview，讓使用者在 Review Card 中決定
				res.Classification = ClassUnknown
				res.NeedsReview = true
			}
		} else {
			// 依副檔名與特定檔名判斷資源類別
			ext := strings.ToLower(filepath.Ext(name))
			baseLower := strings.ToLower(name)
			switch {
			case baseLower == "skill.md" || baseLower == "readme.md":
				// 這兩個特定檔名優先歸類為 CLI 可讀的摘要文件
				res.Classification = ClassCLIMd
				// 外部 CLI skill 的 SKILL.md：解析開頭 frontmatter，把宣告的 name/description 帶進預覽。
				// 解析失敗就維持原行為（純當作 cli_md 文件）。
				if baseLower == "skill.md" {
					if data, rerr := os.ReadFile(fullPath); rerr == nil {
						if parsed, ok := ParseSkillFrontmatter(data); ok {
							fmCopy := parsed
							preview.Frontmatter = &fmCopy
							if name := strings.TrimSpace(parsed.Name); name != "" {
								preview.DisplayName = name
								preview.SkillID = sanitizeSkillID(name)
							}
						}
					}
				}
			case ext == ".md":
				// 其他 Markdown 也歸入 cli_md
				res.Classification = ClassCLIMd
			case ext == ".go" || ext == ".py" || ext == ".sh" || ext == ".js":
				// 可執行程式或腳本歸入 programs
				res.Classification = ClassProgram
			default:
				// 無法識別的格式：標記 NeedsReview
				res.Classification = ClassUnknown
				res.NeedsReview = true
			}
		}

		preview.Resources = append(preview.Resources, res)
	}

	// 來源資料夾本身就帶 skill_manifest.json（典型情境：把先前匯出的 skill 拖回安裝）：
	// 以內嵌 manifest 的 skill_id / display_name / description 為準，避免改用資料夾名稱，
	// 否則拖回來就會變成名字壞掉、標籤遺失的孤兒。SKILL.md frontmatter（外部 CLI skill）
	// 仍維持既有優先序，不被覆蓋。
	if preview.HasManifest && preview.Frontmatter == nil {
		if m, err := LoadManifest(filepath.Join(sourcePath, "skill_manifest.json")); err == nil && m != nil {
			if id := strings.TrimSpace(m.SkillID); id != "" {
				preview.SkillID = id
			}
			if name := strings.TrimSpace(m.DisplayName); name != "" {
				preview.DisplayName = name
			}
			preview.Description = strings.TrimSpace(m.Description)
			preview.Embedded = m
		}
	}

	return preview, nil
}
