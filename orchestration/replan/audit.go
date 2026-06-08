package replan

import (
	"path/filepath"
	"regexp"
	"time"

	"ui_console/audit_log"
)

// ReplanAuditEntry 是每次 replan（含 silent）落下的一筆稽核。
// silent 對使用者可低調，對 log 必須完整可重建——但只記 safe summary + hash，
// 不寫 raw 工具輸出 / 敏感路徑 / credential-like text。
type ReplanAuditEntry struct {
	Timestamp             string          `json:"timestamp"`
	RunID                 string          `json:"run_id"`
	TriggerReason         string          `json:"trigger_reason"` // 已過 SafeSummary
	Failure               FailureCategory `json:"failure"`
	Decision              Decision        `json:"decision"`
	OldTailHash           string          `json:"old_tail_hash"`
	NewTailHash           string          `json:"new_tail_hash"`
	ClassifiedMaxRisk     string          `json:"classified_max_risk"`
	Silent                bool            `json:"silent"`
	ScopeReplan           bool            `json:"scope_replan"`
	ConsecutiveNoProgress int             `json:"consecutive_no_progress"`
	RunTotal              int             `json:"run_total"`
	GoalHash              string          `json:"goal_hash"`
}

var (
	// 類絕對路徑（含使用者目錄）→ 收斂成 <path>，避免洩漏本機結構。
	rePath = regexp.MustCompile(`(?:[A-Za-z]:\\|/)[^\s"']{3,}`)
	// 類 token / credential（長英數字串）→ 收斂成 <redacted>。
	reToken = regexp.MustCompile(`[A-Za-z0-9_\-]{32,}`)
)

// SafeSummary 把任意說明字串清洗成可入 log 的安全摘要：
// 先遮蔽路徑與 token，再截斷長度。
func SafeSummary(s string) string {
	s = rePath.ReplaceAllString(s, "<path>")
	s = reToken.ReplaceAllString(s, "<redacted>")
	const max = 200
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

// NewAuditLog 在 projectRoot/audit_log/replan.jsonl 開一個 append-only 稽核日誌。
// 直接複用既有 audit_log.AppendLog[T]，零新增依賴。
func NewAuditLog(projectRoot string) *audit_log.AppendLog[ReplanAuditEntry] {
	path := filepath.Join(projectRoot, "audit_log", "replan.jsonl")
	return audit_log.New[ReplanAuditEntry](
		path,
		audit_log.WithSkipCorruptLines[ReplanAuditEntry](),
	)
}

// AppendAuditEntry 寫入一筆稽核；未設時間戳則補上，並確保 TriggerReason 已清洗。
func AppendAuditEntry(log *audit_log.AppendLog[ReplanAuditEntry], e ReplanAuditEntry) error {
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	e.TriggerReason = SafeSummary(e.TriggerReason)
	return log.Append(e)
}
