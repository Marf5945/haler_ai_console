package skill_step

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SkillRelation 是資源層級的關聯描述檔，儲存在每個資源子目錄下的 .skill_rel.json。
// 它扮演「橋樑」角色：讓 examples/、programs/、cli_md/ 三大資源類別
// 能夠互相知道彼此的 ID，而不需要 hard-code 路徑或顯示名稱。
//
// 重要規則：
//   - 程式碼必須透過 resource_id 與 .skill_rel.json 來解析資源，
//     不得直接使用檔案名稱或路徑——這樣即使資料夾改名，關聯仍然有效。
//   - Export 時，.skill_rel.json 不會被複製到輸出目錄，
//     只有根目錄的 skill_manifest.json + README.md 會被匯出（詳見 export.go）。
//
// SchemaVersion 固定為 "skill_relation.v1"。
type SkillRelation struct {
	SchemaVersion   string   `json:"schema_version"`   // 固定 "skill_relation.v1"
	SkillID         string   `json:"skill_id"`         // 所屬 skill 的 ID
	ResourceID      string   `json:"resource_id"`      // 本資源的唯一 ID
	ResourceType    string   `json:"resource_type"`    // "example" | "program" | "cli_md"
	DescriptionDoc  string   `json:"description_doc"`  // 指向說明文件的相對路徑，例如 "../../README.md"
	Tags            []string `json:"tags"`             // 繼承自 manifest 的標籤，用於搜尋
	RelatedExamples []string `json:"related_examples"` // 此資源依賴的 example ID
	RelatedPrograms []string `json:"related_programs"` // 此資源依賴的 program ID
	RelatedCLIMd    []string `json:"related_cli_md"`   // 此資源依賴的 cli_md ID
	Notes           string   `json:"notes"`            // 給維護人員的自由文字備註
	Hash            string   `json:"hash"`             // 關聯內容的 SHA-256
}

// LoadRelation 從指定路徑讀取並解析 .skill_rel.json。
// 通常 path 形如 data/skills/<skill_id>/examples/<res_id>/.skill_rel.json。
func LoadRelation(path string) (*SkillRelation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("skill_step: load relation: %w", err)
	}
	var r SkillRelation
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("skill_step: parse relation: %w", err)
	}
	return &r, nil
}

// SaveRelation 將 relation 寫入 dir/.skill_rel.json。
// 寫入前自動計算並填入 Hash 欄位。
// dir 不存在時自動建立（0700）。
func SaveRelation(dir string, r *SkillRelation) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("skill_step: mkdir relation dir: %w", err)
	}
	// 寫入前更新 hash，保持內容與 hash 同步
	r.Hash = hashRelation(r)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("skill_step: marshal relation: %w", err)
	}
	// 檔名固定為 .skill_rel.json，不得改變（archive 與 export 邏輯依賴此名稱）
	path := filepath.Join(dir, ".skill_rel.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("skill_step: write relation: %w", err)
	}
	return nil
}

// hashRelation 計算 skill_id + resource_id + resource_type 的 SHA-256 摘要。
// 只用識別欄位，讓 hash 穩定且不受可選欄位（如 Notes）影響。
func hashRelation(r *SkillRelation) string {
	h := sha256.New()
	h.Write([]byte(r.SkillID + r.ResourceID + r.ResourceType))
	return hex.EncodeToString(h.Sum(nil))
}
