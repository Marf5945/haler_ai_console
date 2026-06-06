// audit.go 實作 append-only JSONL 審計日誌 AuditLog。
// 每次 skill 注入（或撤銷）都應呼叫 AuditLog.Record，
// 確保所有注入事件有完整的可審計記錄。
//
// 儲存格式：
//   - 每筆記錄以一行 JSON 儲存，尾接換行符（JSONL 格式）
//   - 檔案路徑：<dataRoot>/data/skill_step/audit.jsonl
//   - 檔案以 O_APPEND 模式開啟，不覆寫，保證 append-only 語意
//
// 安全限制（必須嚴格遵守）：
//   - InjectionAudit 結構不得包含：原始 CLI 輸出、token、API key、認證快取
//   - 不得儲存使用者本機的完整絕對路徑（避免路徑資訊洩漏）
//   - SummaryHash 取代原始內容，確保可交叉比對但不洩露資料
//
// 與 trust_log.go 的關係：
//   - 兩者都是 append-only JSONL，架構相同
//   - trust_log 記錄 AI 行為信任分數；audit 記錄 skill 注入事件
//   - 未來可考慮共用底層的 JSONL writer utility，但目前各自獨立避免耦合
package skill_step

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InjectionAudit 是一筆 skill 注入的審計記錄。
//
// 欄位設計原則：
//   - 時間戳記（Timestamp）：UTC 時間，使用 time.Time 以便 JSON 自動格式化為 RFC 3339
//   - SessionID：關聯到使用者 session，不儲存使用者個人資料
//   - ActionTarget：使用者輸入的「動作目標」原始字串，用於 audit 追蹤
//   - SkillID：被注入的 skill 識別碼
//   - ResolveStatus：路由判定結果（auto_selected / needs_cli_candidate / ...）
//   - Reason：注入理由（來自 Injection.Reason）
//   - SummaryHash：Injection.SummaryHash，不儲存原始內容
//   - Risk：風險等級
//   - ClearedAt：若此注入已被撤銷，記錄撤銷時間；nil 表示仍 active
//   - ClearReason：撤銷理由，如 "session_ended"、"user_cancelled"
type InjectionAudit struct {
	Timestamp     time.Time  `json:"timestamp"`              // 注入時間（UTC）
	SessionID     string     `json:"session_id"`             // 所屬 session
	ActionTarget  string     `json:"action_target"`          // 原始動作目標字串
	SkillID       string     `json:"skill_id"`               // 注入的 skill ID
	ResolveStatus string     `json:"resolve_status"`         // 路由判定結果
	Reason        string     `json:"reason"`                 // 注入理由
	SummaryHash   string     `json:"summary_hash"`           // 內容摘要 SHA-256
	Risk          string     `json:"risk"`                   // 風險等級
	ClearedAt     *time.Time `json:"cleared_at,omitempty"`   // 撤銷時間（nil = 仍 active）
	ClearReason   string     `json:"clear_reason,omitempty"` // 撤銷理由
}

// AuditLog 是 append-only JSONL 審計日誌的主體。
// 每個 App 實例應只建立一個 AuditLog，與 App struct 共用生命週期。
// 並發安全：Record 與 Recent 都持有 mu 鎖。
type AuditLog struct {
	mu   sync.Mutex // 保護檔案的並發讀寫
	path string     // audit.jsonl 的完整路徑
}

// NewAuditLog 建立一個 AuditLog，日誌路徑設定為
// <dataRoot>/data/skill_step/audit.jsonl。
// 日誌目錄會在第一次 Record 時惰性建立，不在這裡預先建立。
func NewAuditLog(dataRoot string) *AuditLog {
	logPath := filepath.Join(dataRoot, "data", "skill_step", "audit.jsonl")
	return &AuditLog{path: logPath}
}

// Record 將 entry 以 JSON 格式 append 到 JSONL 日誌檔。
//
// 執行步驟：
//  1. 以 O_CREATE|O_APPEND|O_WRONLY 開啟日誌檔（目錄不存在則先建立）
//  2. 序列化 entry 為 JSON，尾接 '\n'
//  3. 寫入檔案後關閉（每次 Record 都重新開關檔，確保 crash-safe）
//
// 並發安全：持有 mu 鎖，確保多個 goroutine 同時呼叫不會交錯寫入。
// 檔案權限：0o600（只有擁有者可讀寫，防止其他使用者存取審計記錄）。
func (a *AuditLog) Record(entry InjectionAudit) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 確保日誌目錄存在（惰性建立）
	if err := os.MkdirAll(filepath.Dir(a.path), 0o700); err != nil {
		return fmt.Errorf("skill_step: audit mkdir: %w", err)
	}

	// 以 append 模式開啟，確保不覆寫歷史記錄
	f, err := os.OpenFile(a.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("skill_step: audit open: %w", err)
	}
	defer f.Close()

	// 序列化為 JSON，再加換行符（JSONL 格式）
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("skill_step: audit marshal: %w", err)
	}
	line = append(line, '\n')

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("skill_step: audit write: %w", err)
	}
	return nil
}

// Recent 讀取日誌中屬於 sessionID 的最近 limit 筆記錄。
//
// 實作說明：
//   - 讀取整個 JSONL 檔案，逐行解析後以 sessionID 過濾
//   - 適合短期日誌（數百筆內）；若日誌成長到數萬筆，應考慮加入索引
//   - limit <= 0 或結果筆數不足 limit 時，回傳所有符合的記錄
//   - 損毀的 JSON 行靜默跳過，不中止讀取
//   - 若日誌檔不存在（首次使用），回傳空清單而非錯誤
//
// 並發安全：持有 mu 鎖，與 Record 互斥，防止讀取時檔案被同時寫入。
func (a *AuditLog) Recent(sessionID string, limit int) ([]InjectionAudit, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	f, err := os.Open(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []InjectionAudit{}, nil // 日誌尚未建立，回傳空清單
		}
		return nil, fmt.Errorf("skill_step: audit read: %w", err)
	}
	defer f.Close()

	// 逐行解析 JSONL，只收集屬於指定 sessionID 的記錄
	var all []InjectionAudit
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // 跳過空行
		}
		var entry InjectionAudit
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // 損毀行靜默跳過，不中止整個讀取
		}
		if entry.SessionID == sessionID {
			all = append(all, entry)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("skill_step: audit scan: %w", err)
	}

	// 若結果不超過 limit，直接回傳全部
	if limit <= 0 || len(all) <= limit {
		return all, nil
	}
	// 回傳最後 limit 筆（最新的 limit 筆）
	return all[len(all)-limit:], nil
}
