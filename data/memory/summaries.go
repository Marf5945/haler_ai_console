// memory/summaries.go — summaries.md 寫入器（§29.3 摘要輸出，Rule 15 目標）。
// 摘要結果寫這裡，talk_full.md 不動（原始對話完整保留）。
package memory

import (
	"fmt"
	"os"
	"path/filepath"
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
