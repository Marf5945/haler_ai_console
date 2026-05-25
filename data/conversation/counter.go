// conversation/counter.go — 字數統計與摘要觸發判斷。
// 追蹤對話字元數，判斷何時需要啟動摘要壓縮流程。
package conversation

import "sync"

// ──────────────────────────────────────────────
// 常數定義
// ──────────────────────────────────────────────

const (
	// SummarizationThreshold 達到此字元數時觸發摘要（預設 10000 字）
	SummarizationThreshold = 10000
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// CharCounter 執行緒安全的字元計數器。
type CharCounter struct {
	mu           sync.Mutex
	currentCount int
}

// ──────────────────────────────────────────────
// 建構
// ──────────────────────────────────────────────

// NewCharCounter 建立歸零的字元計數器。
func NewCharCounter() *CharCounter {
	return &CharCounter{}
}

// ──────────────────────────────────────────────
// 操作
// ──────────────────────────────────────────────

// Add 增加 n 個字元到計數器（新增句子時呼叫）。
func (c *CharCounter) Add(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentCount += n
}

// Subtract 從計數器減去 n 個字元（摘要後壓縮字數時呼叫）。
func (c *CharCounter) Subtract(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentCount -= n
	// 防止負值
	if c.currentCount < 0 {
		c.currentCount = 0
	}
}

// Reset 將計數器歸零。
func (c *CharCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentCount = 0
}

// Count 回傳目前字元數（執行緒安全）。
func (c *CharCounter) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentCount
}

// NeedsSummarization 判斷是否已達摘要觸發門檻。
func (c *CharCounter) NeedsSummarization() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentCount >= SummarizationThreshold
}

// ──────────────────────────────────────────────
// 純函式：統計句子字元數
// ──────────────────────────────────────────────

// CountConversationChars 計算句子清單中 user 與 assistant 的字元總數。
// 排除 tool-action 的 metadata，只計算有效對話內容。
func CountConversationChars(sentences []Sentence) int {
	total := 0
	for _, sent := range sentences {
		if sent.Role == "user" || sent.Role == "assistant" {
			// 以 rune 計算，正確處理中文字元
			total += len([]rune(sent.Content))
		}
	}
	return total
}
