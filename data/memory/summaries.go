// memory/summaries.go — summaries.md 寫入器（§29.3 摘要輸出，Rule 15 目標）。
// 摘要結果寫這裡，talk_full.md 不動（原始對話完整保留）。
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileSummaries 是摘要輸出檔（Rule 15：摘要不可寫 talk_full.md）。
const FileSummaries = "summaries.md"

// AppendSummary 將一段摘要追加到 summaries.md。
// Rule 8：寫入磁碟前執行 write redaction。回傳被遮蔽的記錄供稽核。
func (p *Pipeline) AppendSummary(tag, content string) ([]RedactionRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Rule 8：寫入前 redaction。
	cleaned, records := RedactBeforeWrite(content)

	ts := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n## %s — %s\n%s\n", tag, ts, cleaned)

	path := filepath.Join(p.rootDir, FileSummaries)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("寫入 summaries 失敗: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return nil, fmt.Errorf("寫入 summaries 失敗: %w", err)
	}
	return records, nil
}

// ReadRecentSummaries 讀 summaries.md，回傳最後 limit 段摘要內容（去標頭）。
// 供 routing_memory_context 注入 judge 用；找不到檔/空檔回 nil。
// 注意 summaries.md 於 AppendSummary 寫入時已遮蔽，這裡讀回再經一次注入端遮蔽=雙閘。
func (p *Pipeline) ReadRecentSummaries(limit int) []string {
	if p == nil || limit <= 0 {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(p.rootDir, FileSummaries))
	if err != nil {
		return nil
	}
	var entries []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			if t := strings.TrimSpace(strings.Join(cur, "\n")); t != "" {
				entries = append(entries, t)
			}
			cur = nil
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			continue
		}
		cur = append(cur, line)
	}
	flush()
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries
}
