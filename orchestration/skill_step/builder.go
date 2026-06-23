// builder.go 負責三件事：
//  1. 將已路由的 skill 組裝成「注入載體」（Injection）
//  2. 管理 session 層級的注入生命週期（見 injection.go）
//  3. 以 append-only JSONL 記錄所有注入事件的審計日誌（見 audit.go）
//
// 安全邊界：
//   - Injection 不包含 skill 的完整原始內容（如程式碼、範例文字）
//   - 只傳遞 ResourceRefs（資源 ID 清單），讓 CLI 按需自行查詢，不主動推送
//   - SummaryHash 讓審計日誌可以交叉比對，但不洩露原始資料
//   - CLIAdapter 實作必須保證不記錄原始 CLI 輸出、token 或認證快取
package skill_step

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Injection 是 skill 注入的核心載體，描述「把哪個 skill 以何種條件注入哪個 session」。
// 它會被 InjectionStore 以 sessionID 為鍵暫存在記憶體，
// 並由 AuditLog 以 SummaryHash 記錄至磁碟（不記錄原始內容）。
//
// 欄位設計原則：
//   - InjectionID：每次注入的唯一識別碼，使用 UnixNano 確保唯一性
//   - SkillID：對應 skill_manifest.json 的 skill_id，作為跨模組的橋接鍵
//   - Reason：人類可讀的注入理由，包含 sessionID 與分數，供 UI 顯示
//   - SummaryHash：skill_id + reason + risk 的 SHA-256，保護審計記錄完整性
//   - Risk：從 Candidate.Risk 繼承，決定後續的 Policy 行為
//   - AllowedUse：允許的使用方式，目前固定為 ["context_augmentation"]
//   - BlockedUse：明確禁止的使用方式，防止 CLI 誤用注入的 skill 資訊
//   - ResourceRefs：以 ID 清單形式傳遞資源，不傳遞實際內容
type Injection struct {
	InjectionID  string              `json:"injection_id"`  // 本次注入的唯一 ID
	SkillID      string              `json:"skill_id"`      // 對應的 skill ID
	Reason       string              `json:"reason"`        // 注入理由（人類可讀）
	SummaryHash  string              `json:"summary_hash"`  // 內容摘要的 SHA-256
	Risk         string              `json:"risk"`          // 風險等級，繼承自 Candidate
	AllowedUse   []string            `json:"allowed_use"`   // 允許的使用方式
	BlockedUse   []string            `json:"blocked_use"`   // 禁止的使用方式
	ResourceRefs map[string][]string `json:"resource_refs"` // 資源 ID 清單（非實際內容）
}

// BuildInjection 從已路由的 Candidate 與其對應的 SkillManifest 建立一個 Injection。
//
// SummaryHash 的計算方式：
//   - 輸入：skill_id + reason + risk（三個識別欄位，不含可變動的時間戳或計數器）
//   - 演算法：SHA-256
//   - 目的：讓 AuditLog 可以用 hash 交叉比對注入記錄，而不需要在 audit 中儲存原始內容
//
// BlockedUse 固定包含以下三項，確保 CLI 明確知道不可做的操作：
//   - manifest_modification：禁止修改 skill_manifest.json
//   - relation_modification：禁止修改 .skill_rel.json
//   - alias_modification：禁止修改 routing.target_aliases 或 tags
//
// ResourceRefs 從 manifest.Resources 直接對應，鍵名與子目錄名稱一致：
//   - "examples"：examples/ 下的資源 ID 清單
//   - "programs"：programs/ 下的資源 ID 清單
//   - "cli_md"：cli_md/ 下的資源 ID 清單
func BuildInjection(sessionID string, candidate Candidate, manifest SkillManifest) *Injection {
	// 組裝人類可讀的注入理由，包含 session ID 與評分，方便日後 audit 追蹤
	reason := fmt.Sprintf("resolved via router for session %s (score=%.2f)", sessionID, candidate.Score)

	// 計算 SummaryHash：使用三個核心識別欄位，避免儲存敏感原始資料
	h := sha256.New()
	h.Write([]byte(candidate.SkillID + reason + candidate.Risk))
	summaryHash := hex.EncodeToString(h.Sum(nil))

	// 建立資源參考：只傳 ID，不傳實際檔案內容或路徑
	// CLI 若需要具體資源，應自行透過 Console API 查詢
	refs := map[string][]string{
		"examples": manifest.Resources.Examples,
		"programs": manifest.Resources.Programs,
		"cli_md":   manifest.Resources.CLIMd,
	}

	return &Injection{
		InjectionID: fmt.Sprintf("inj-%d", time.Now().UnixNano()),
		SkillID:     candidate.SkillID,
		Reason:      reason,
		SummaryHash: summaryHash,
		Risk:        candidate.Risk,
		AllowedUse:  []string{"context_augmentation"}, // 目前唯一允許的使用場景
		// 明確列舉禁止操作，讓 CLI adapter 與使用者都清楚邊界
		BlockedUse: []string{
			"manifest_modification", // 禁止修改 skill_manifest.json
			"relation_modification", // 禁止修改 .skill_rel.json
			"alias_modification",    // 禁止修改 routing 別名或 tag
		},
		ResourceRefs: refs,
	}
}
