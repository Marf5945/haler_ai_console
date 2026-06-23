// injection.go 定義兩件事：
//  1. CLIAdapter 介面——所有 CLI 通訊層的統一合約
//  2. InjectionStore——記憶體層級的 session→Injection 映射，並發安全
//
// 設計說明：
//   - CLIAdapter 使用介面而非具體實作，讓 Claude CLI、Codex CLI、Gemini CLI
//     各自實作自己的 adapter，app.go 只依賴介面，方便測試與替換。
//   - InjectionStore 只存活在記憶體中，不持久化；
//     持久性審計記錄由 audit.go 的 AuditLog 負責（append-only JSONL）。
//   - 每個 sessionID 對應最多一個 active Injection，
//     Set 會直接覆蓋，Clear 用於 session 結束或使用者手動撤銷時。
package skill_step

import "sync"

// CLIMessageOptions 封裝發送訊息給 CLI 所需的所有資料。
//
// SkillInjection 可為 nil（代表本次訊息不附帶 skill 上下文）；
// 若不為 nil，adapter 實作應將 Injection 的資訊嵌入 prompt 或系統訊息中，
// 但絕對不得將 Injection.BlockedUse 裡列舉的操作呈現給 CLI 執行。
type CLIMessageOptions struct {
	AdapterID       string     // 目前選中的 CLI adapter ID（claude / codex / gemini...）
	CLIPath         string     // 已解析的 CLI 可執行檔完整路徑
	SessionID       string     // 目前的使用者 session ID
	UserText        string     // 使用者原始輸入文字
	Model           string     // 選填；單次呼叫指定 CLI model，空字串則用 adapter 設定
	SkillInjection  *Injection // 選填；nil 表示無 skill 上下文注入
	SystemPrompt    string     // 選填；本次訊息的 system/persona prompt
	ContinuityKey   string     // 選填；隔離對話連續性狀態的 key
	TraceID         string     // DEBUG_TRACE_REMOVE: debug-only correlation ID for UI -> CLI tracing
	ToolRoutingMode string     // 空字串=一般工具選擇；judge=第一輪只判斷是否需要工具
	SkipContinuity  bool       // true = 跳過 SentenceStore / Synthesize（閒聊用）
	IsCommand       bool       // true = Controller 已判定本輪走命令入口，需蓋 seal
}

// CLIResponse 是 CLIAdapter.SendMessage 的回傳值。
// Text 是 CLI 的回應文字；Error 是 CLI 層的錯誤描述（非 Go error，方便 JSON 傳輸）。
// 設計為值型別（非指標），確保 caller 不需要檢查 nil。
type CLIResponse struct {
	Text         string `json:"text"`                    // CLI 回應文字
	Error        string `json:"error,omitempty"`         // 錯誤描述，空字串表示無錯誤
	AuthRequired bool   `json:"auth_required,omitempty"` // CLI 需要瀏覽器 OAuth 授權
	AuthURL      string `json:"auth_url,omitempty"`      // OAuth 授權 URL（由 sidecar 從 CLI 輸出擷取）
	AdapterID    string `json:"adapter_id,omitempty"`    // 需要授權的 adapter ID
	Action       string `json:"action,omitempty"`        // 解析後的 action tag，供 Controller/UI 做內建副作用
	Target       string `json:"target,omitempty"`        // 解析後的內容/目標，不直接等於可執行權限
	Next         string `json:"next,omitempty"`          // 解析後的下一步：待命 / 澄清 / 確認 / 完成
	NeedsUser    bool   `json:"needs_user,omitempty"`    // 路由結果在等使用者（skill 待確認/提問）；chat_route 節點用來判斷是否暫停 DAG
}

// CLIAdapter 是所有 CLI 通訊層的統一介面。
//
// 安全限制（必須在所有實作中遵守）：
//   - 不得記錄原始 CLI 輸出到磁碟或日誌
//   - 不得在回應中暴露 token、API key 或認證快取
//   - 不得將 SkillInjection 的完整原始資料傳遞給 CLI（只傳摘要或 ID）
//   - 實作必須驗證 CLIMessageOptions.SkillInjection.BlockedUse，拒絕被禁止的操作
//
// 目前尚未有具體實作（#58 任務）；測試時可用 mock struct 實作此介面。
type CLIAdapter interface {
	SendMessage(options CLIMessageOptions) (CLIResponse, error)
}

// InjectionStore 是以 sessionID 為鍵的記憶體 Injection 倉庫。
// 每個 session 最多儲存一個 active Injection；
// 舊的注入被 Set 覆蓋時不會寫入日誌（日誌由 AuditLog 負責）。
//
// 並發安全：所有公開方法都持有 mu 鎖，可安全用於多 goroutine 環境。
type InjectionStore struct {
	mu      sync.Mutex            // 保護 current 的並發存取
	current map[string]*Injection // key = sessionID，value = 當前 active Injection
}

// NewInjectionStore 建立一個空的 InjectionStore。
// 在 app.go 的 NewApp() 中應與 AuditLog 一起初始化，確保兩者生命週期一致。
func NewInjectionStore() *InjectionStore {
	return &InjectionStore{
		current: make(map[string]*Injection),
	}
}

// Set 將 inj 設定為 sessionID 的 active Injection。
// 若該 session 已有注入，直接覆蓋（舊注入應在覆蓋前由呼叫端寫入 AuditLog）。
func (s *InjectionStore) Set(sessionID string, inj *Injection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current[sessionID] = inj
}

// Get 回傳 sessionID 對應的 active Injection。
// 若該 session 無注入（或已被 Clear），回傳 nil。
// 呼叫端應在使用前檢查 nil，不得直接解引用。
func (s *InjectionStore) Get(sessionID string) *Injection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current[sessionID]
}

// Clear 移除 sessionID 對應的 Injection。
// 適用於：session 結束、使用者手動撤銷注入、或注入逾時後的清理。
// 呼叫端應在 Clear 前透過 AuditLog.Record 記錄清除事件（含 ClearedAt 與 ClearReason）。
func (s *InjectionStore) Clear(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.current, sessionID)
}
