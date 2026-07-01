// Package inspectormemory holds the short-lived chat memory for the top
// "打開互動" popover (a.k.a. the CLI inspector lane).
//
// 設計原則：
//   - 純記憶體、絕不落盤。這段歷史只服務於上方互動彈窗，跟下方主聊天的
//     SentenceStore 是分開的兩套，互不影響。
//   - 只保留最近 HistoryLimit 句（user/assistant 各算一句）。
//   - 生命週期刻意很短：使用者關閉彈窗（Cancel）或切換人格時都會被清空；
//     關閉整個系統時因為本來就不落盤，自然也不會留下任何東西。這是特意
//     設計成「易忘」的短期記憶，跟紀念照那種永久記憶（album 套件）是兩種
//     不同的保留策略，不要混在一起改。
package inspectormemory

import "sync"

// HistoryLimit is the maximum number of history lines retained. Once
// exceeded, the oldest lines are dropped first.
const HistoryLimit = 30

// Service is a small in-memory ring buffer of "role: text" lines.
// Zero value is not usable; use New().
type Service struct {
	mu      sync.Mutex
	history []string // 格式："user: xxx" / "assistant: xxx"
}

// New creates an empty inspector memory service.
func New() *Service {
	return &Service{}
}

// Append records one turn and trims the buffer down to HistoryLimit.
func (s *Service) Append(role, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, role+": "+text)
	if len(s.history) > HistoryLimit {
		s.history = s.history[len(s.history)-HistoryLimit:]
	}
}

// Snapshot returns a copy of the current history, oldest first.
func (s *Service) Snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.history))
	copy(out, s.history)
	return out
}

// Clear discards all retained history. Called when the popover closes or
// the active persona changes.
func (s *Service) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = nil
}
