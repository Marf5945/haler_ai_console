// policy.go 實作 Skill Context Orchestration 的安全政策引擎 Policy。
// Policy 是一組純函式（無狀態），將「安全規則」集中在一個地方，
// 讓其他模組（app.go、Router、InjectionStore）只需呼叫 Policy 方法，
// 而不需要分散地在各處重複實作相同的條件判斷。
//
// 設計原則：
//   - 純函式設計：Policy 方法沒有副作用，輸入相同輸出一定相同，方便單元測試
//   - 明確性：每個方法名稱清楚說明「這個決策是什麼」，程式碼即文件
//   - 擴充性：若未來需要組織層級的 policy override，在此加入設定欄位即可
//
// 與 AmbiguityDetector 的關係：
//   - AmbiguityDetector 關注「多個候選如何取捨」（路由選擇問題）
//   - Policy 關注「選定的 skill 是否符合安全限制」（注入授權問題）
//   - 兩者共同確保路由與注入的雙重安全閘門
package skill_step

// Policy 是 Skill Context Orchestration 的安全政策引擎。
// 目前為無狀態設計（空結構），所有方法均為純函式。
// 若未來需要可配置的政策（如 RiskOverride、OrganizationPolicy），
// 可在此結構加入欄位，不影響呼叫端介面。
type Policy struct{}

// NewPolicy 建立一個 Policy 實例。
// 目前 Policy 不需要任何初始化，此函式保留是為了維持一致的工廠模式。
func NewPolicy() *Policy { return &Policy{} }

// CanAutoSelect 判斷一個候選 skill 是否可以在不需使用者確認的情況下自動注入。
//
// 自動注入必須同時滿足兩個條件：
//  1. score >= minScore：分數達到設定的最低自動選取門檻（通常 0.85）
//  2. risk == "low"：風險等級為低風險
//
// 任何一個條件不滿足，就不應自動注入——改由 AmbiguityDetector 判定後續流程
// （可能是 needs_cli_candidate 或 needs_user_review）。
//
// 呼叫端：app.go 的 BuildSkillContext 在決定是否直接呼叫 BuildInjection 前應先查詢此方法。
func (p *Policy) CanAutoSelect(score float64, risk string, minScore float64) bool {
	return score >= minScore && risk == "low"
}

// RequiresReview 判斷指定風險等級的 skill 是否需要使用者明確確認才能注入。
//
// 目前需要人工審查的風險等級：
//   - "high"：高風險，例如可能存取外部網路或寫入重要檔案的 skill
//   - "critical"：緊急風險，例如可執行系統指令或操作敏感資料的 skill
//
// 若未來新增 "extreme" 等更高等級，在此加入條件即可，其他程式碼不需改動。
//
// 設計注意：medium 風險不需要審查（回傳 false），但也不會被 CanAutoSelect 允許自動注入，
// medium 風險的 skill 通常走 needs_cli_candidate 流程，讓 CLI 決定。
func (p *Policy) RequiresReview(risk string) bool {
	return risk == "high" || risk == "critical"
}

// CLIOutputMayModifyManifest 永遠回傳 false。
//
// 這是系統層級的不可變規則：
//   - skill_manifest.json 與 .skill_rel.json 只能由使用者透過 Archive / Review 流程修改
//   - CLI 的輸出（無論來自 Claude、Codex 或 Gemini）永遠不得直接修改 manifest 或 relation
//   - 別名（routing.target_aliases）的修改也在此禁止範圍內
//
// 這個函式存在的意義是讓其他模組可以明確查詢這個規則，
// 而不是讓它以隱式的方式散落在程式碼各處。
// 若未來政策有變動，只需修改此函式，其他模組自動遵守。
func (p *Policy) CLIOutputMayModifyManifest() bool {
	return false // 不可變規則：CLI 輸出永遠不得修改 manifest
}
