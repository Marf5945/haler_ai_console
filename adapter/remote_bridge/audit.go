// remote_bridge/audit.go — append-only 稽核日誌（§12A.12）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ §12A.12 要求所有遠端橋接操作都必須寫入不可竄改的稽核日誌。  │
// │                                                             │
// │ 記錄範圍：dispatch / flush / remote_intent / review response│
// │  / continue / retry / ignored / rejected / failed delivery  │
// │  / identity mismatch / permission mismatch                  │
// │                                                             │
// │ 格式：JSONL（每行一筆 JSON），append-only 寫入。            │
// │ 路徑：{projectRoot}/remote_bridge/remote_bridge_audit.jsonl │
// │                                                             │
// │ 注意：§12A.12 明確規定「raw message content 不得預設記錄」  │
// │ → AuditEntry 只記 metadata（dispatch_id, channel, mode,     │
// │   risk_class, outcome 等），不記原始訊息內容。               │
// │                                                             │
// │ 介面：Append / ReadAll / ReadRecent                          │
// │ 被 service.go 各操作方法在成功/失敗後呼叫。                 │
// │                                                             │
// │ 重構：v4.0 遷移至 audit_log.AppendLog[AuditEntry] 共用抽象。│
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"path/filepath"
	"time"

	"ui_console/audit_log"
)

// ──────────────────────────────────────────────
// 稽核日誌服務
// ──────────────────────────────────────────────

const auditFileName = "remote_bridge_audit.jsonl"

// AuditLog append-only 遠端橋接稽核日誌。
// 內部委託 audit_log.AppendLog[AuditEntry]。
type AuditLog struct {
	inner *audit_log.AppendLog[AuditEntry]
}

// NewAuditLog 建立稽核日誌服務。
func NewAuditLog(projectRoot string) *AuditLog {
	p := filepath.Join(projectRoot, "remote_bridge", auditFileName)
	return &AuditLog{
		inner: audit_log.New[AuditEntry](
			p,
			audit_log.WithSkipCorruptLines[AuditEntry](),
			audit_log.WithBeforeAppend[AuditEntry](func(e *AuditEntry) error {
				if e.CreatedAt.IsZero() {
					e.CreatedAt = time.Now()
				}
				return nil
			}),
		),
	}
}

// Append 寫入一筆稽核記錄。
func (al *AuditLog) Append(entry AuditEntry) error {
	return al.inner.Append(entry)
}

// ReadAll 讀取所有稽核記錄（用於 UI 顯示或除錯）。
func (al *AuditLog) ReadAll() ([]AuditEntry, error) {
	return al.inner.ReadAll()
}

// ReadRecent 讀取最近 N 筆稽核記錄。
func (al *AuditLog) ReadRecent(n int) ([]AuditEntry, error) {
	return al.inner.ReadRecent(n)
}
