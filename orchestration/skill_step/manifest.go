// Package skill_step 統一管理 skill domain 的三段職責：
// 歸檔（archive）、路由（router）、上下文注入（context）。
//
// 外部 CLI（Claude / Codex / Gemini）不會直接讀取完整 skill 套件，
// 所有 skill 均由 AI Console 集中管理，並透過路由注入最小摘要。
package skill_step

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SourceType 描述 skill 被匯入系統的方式。
// 這個欄位讓維護人員知道某個 skill 是從哪個渠道進來的，
// 方便後續審查或重新匯入時定位來源。
type SourceType string

const (
	// SourceDragImport 表示使用者以拖曳方式將 skill 資料夾丟入 Console。
	SourceDragImport SourceType = "drag_import"
	// SourceFolderScan 表示使用者透過「掃描資料夾」流程匯入。
	SourceFolderScan SourceType = "folder_scan"
	// SourceExternalCLIArchive 表示從外部 CLI 的 skill 目錄（如 SKILL.md 資料夾）歸檔而來。
	SourceExternalCLIArchive SourceType = "external_cli_skill_archive"
	// SourceBuiltin 表示系統內建能力，啟動時由 Go 程式碼直接註冊，不寫入磁碟。
	SourceBuiltin SourceType = "builtin"
)

// SkillSource 記錄 skill 的來源資訊。
// OriginalPathHash 儲存的是原始路徑的 SHA-256 雜湊，
// 而非明文路徑——這樣可以避免在 manifest 中暴露使用者的本機路徑。
type SkillSource struct {
	SourceType       SourceType `json:"source_type"`
	OriginalPathHash string     `json:"original_path_hash"` // 原始路徑的 SHA-256，非明文
}

// SkillTags 用於路由的匹配與篩選。
// 路由時會比對 ActionTag 與 DomainTag；
// RiskTag 決定這個 skill 啟動後需要哪一級的確認流程。
type SkillTags struct {
	PurposeTag []string `json:"purpose_tag"` // 用途分類，例如 ["lookup", "transform"]
	ActionTag  []string `json:"action_tag"`  // 對應的動作關鍵字，例如 ["查詢"]
	DomainTag  []string `json:"domain_tag"`  // 領域關鍵字，例如 ["weather", "天氣"]
	RiskTag    []string `json:"risk_tag"`    // 風險等級：low / medium / high / critical
}

// SkillPermissions 宣告這個 skill 執行時允許接觸的系統資源。
// Console 會在注入前檢查這些欄位，並在 Review Card 中顯示給使用者確認。
// 欄位值範例：network="none|localhost|external"；filesystem="none|workspace_read|workspace_write"；execution="none|controlled"
type SkillPermissions struct {
	Network    string `json:"network"`    // 網路存取範圍
	Filesystem string `json:"filesystem"` // 檔案系統存取範圍
	Execution  string `json:"execution"`  // 程式執行權限
}

// SkillResources 列出這個 skill 下所有資源的 ID 清單。
// 注意：這裡存的是資源 ID（resource_id），不是檔案名稱或路徑。
// 實際路徑須透過各資源目錄內的 .skill_rel.json 來解析，
// 這樣即使資料夾改名，只要 ID 不變，關聯就不會斷。
type SkillResources struct {
	Examples []string `json:"examples"` // examples/ 底下的資源 ID
	Programs []string `json:"programs"` // programs/ 底下的資源 ID
	CLIMd    []string `json:"cli_md"`   // cli_md/ 底下的資源 ID
}

// SkillRouting 控制路由的自動解析行為。
// ActionPatterns 與 TargetAliases 是路由匹配的主要依據。
// MinimumAutoScore 是自動選取的最低門檻——低於此分數的候選
// 會進入「詢問 CLI」或「請使用者確認」流程，不會自動注入。
type SkillRouting struct {
	ActionPatterns   []string `json:"action_patterns"`    // 完整動作模式，例如 ["查詢ㄌ天氣"]
	TargetAliases    []string `json:"target_aliases"`     // 目標別名，例如 ["天氣", "氣象", "weather"]
	MinimumAutoScore float64  `json:"minimum_auto_score"` // 自動選取門檻，預設 0.82
}

// Manifest schema 版本：新寫入用 v2，loader 同時接受 v1/v2（TASK 31 / Phase 1.2）。
const (
	SchemaManifestV1 = "skill_manifest.v1"
	SchemaManifestV2 = "skill_manifest.v2"
)

// SkillManifest 是一個已歸檔 skill 的完整機器可讀描述符。
// 每個 skill 資料夾下必有一個 skill_manifest.json，
// 這是 Console 用來識別、路由、審查 skill 的唯一來源。
// SchemaVersion 固定為 "skill_manifest.v1"。
type SkillManifest struct {
	SchemaVersion  string           `json:"schema_version"`  // 固定 "skill_manifest.v1"
	SkillID        string           `json:"skill_id"`        // 唯一識別碼，例如 "weather.lookup"
	DisplayName    string           `json:"display_name"`    // 使用者介面顯示名稱
	Version        string           `json:"version"`         // 語意版本，例如 "1.0.0"
	DescriptionDoc string           `json:"description_doc"` // 人類可讀說明文件的檔名，通常為 "README.md"
	Source         SkillSource      `json:"source"`
	Tags           SkillTags        `json:"tags"`
	Permissions    SkillPermissions `json:"permissions"`
	Resources      SkillResources   `json:"resources"`
	Routing        SkillRouting     `json:"routing"`
	// TASK 31：以下兩欄皆 optional，舊 v1 manifest 缺欄位 = 零值，向後相容。
	Lifecycle     *Lifecycle     `json:"lifecycle,omitempty"`      // 可見性/可執行性，nil 時由 EnsureLifecycle 補預設
	ExpectedChain *ExpectedChain `json:"expected_chain,omitempty"` // drift 比對基準，nil 時只跑低階 drift
	Hash          string         `json:"hash"`                     // 全欄位 canonical SHA-256（見 canonicalHash）
	HashMismatch  bool           `json:"-"`                        // runtime-only：load 時 hash 不符（不持久化）
}

// LoadManifest 從指定路徑讀取並解析 skill_manifest.json。
// 若檔案不存在或 JSON 格式錯誤，會回傳帶 context 的錯誤，方便上層定位問題。
func LoadManifest(path string) (*SkillManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("skill_step: load manifest: %w", err)
	}
	var m SkillManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("skill_step: parse manifest: %w", err)
	}
	// 接受 v1/v2：缺 lifecycle 補安全預設（不變式：builtin/舊 skill 不可消失）。
	EnsureLifecycle(&m)
	// hash 不符採「警告不硬拒」：避免未來匯入舊 v1（舊算法）skill 被鎖死。
	if m.Hash != "" && m.Hash != canonicalHash(&m) {
		m.HashMismatch = true
	}
	return &m, nil
}

// SaveManifest 將 manifest 寫入 dir/skill_manifest.json。
// 寫入前會自動計算並填入 Hash 欄位，不需要呼叫方手動設定。
// dir 不存在時會自動建立（權限 0700，限制其他使用者讀取）。
func SaveManifest(dir string, m *SkillManifest) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("skill_step: mkdir manifest dir: %w", err)
	}
	// 寫入前升 schema 版本、補 lifecycle、用全欄位 canonical hash（TASK 31）。
	if m.SchemaVersion == "" || m.SchemaVersion == SchemaManifestV1 {
		m.SchemaVersion = SchemaManifestV2
	}
	EnsureLifecycle(m)
	m.Hash = canonicalHash(m)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("skill_step: marshal manifest: %w", err)
	}
	path := filepath.Join(dir, "skill_manifest.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("skill_step: write manifest: %w", err)
	}
	return nil
}

// canonicalHash 清空 Hash 欄位後序列化「整份 manifest」再 SHA-256。
// 這樣 expected_chain / lifecycle / permissions 任何竄改都會反映在 hash，
// 杜絕「改了 drift 基準卻不改 hash」的信任漏洞（TASK 31 / Phase 1.2）。
func canonicalHash(m *SkillManifest) string {
	c := *m
	c.Hash = ""                 // 先清空，避免把舊 hash 算進新 hash
	data, _ := json.Marshal(&c) // struct 欄位順序固定 → 結果具確定性
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
