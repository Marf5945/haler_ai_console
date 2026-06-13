// conversation/sentence.go — 句子管理核心。
// 管理句子 ID（[I-XXX] / [O-XXX]），解析 talk_full.md，追蹤句子異動。
package conversation

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// Sentence 代表一句對話單元。
type Sentence struct {
	ID        string    // [I-001] / [O-001] / [tool-action: ...]
	Role      string    // user / assistant / tool-action
	Content   string    // 句子內文
	Timestamp time.Time // 建立時間
}

// SentenceStore 以有序切片持有全部句子，mutex 保護並發存取。
type SentenceStore struct {
	mu           sync.Mutex
	sentences    []Sentence
	nextInputID  int // 下一個 [I-XXX] 序號
	nextOutputID int // 下一個 [O-XXX] 序號
}

// ──────────────────────────────────────────────
// 建構
// ──────────────────────────────────────────────

// NewSentenceStore 建立空白句子庫，ID 從 001 開始。
func NewSentenceStore() *SentenceStore {
	return &SentenceStore{
		nextInputID:  1,
		nextOutputID: 1,
	}
}

// ──────────────────────────────────────────────
// 新增
// ──────────────────────────────────────────────

// AddInput 新增使用者輸入句，指派 [I-XXX] ID 並遞增計數器。
func (s *SentenceStore) AddInput(content string) Sentence {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("[I-%03d]", s.nextInputID)
	s.nextInputID++

	sent := Sentence{
		ID:        id,
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	}
	s.sentences = append(s.sentences, sent)
	return sent
}

// AddOutput 新增 AI 回覆句，指派 [O-XXX] ID 並遞增計數器。
func (s *SentenceStore) AddOutput(content string) Sentence {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("[O-%03d]", s.nextOutputID)
	s.nextOutputID++

	sent := Sentence{
		ID:        id,
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	}
	s.sentences = append(s.sentences, sent)
	return sent
}

// AddToolAction 新增工具執行記錄，ID 格式為 [tool-action: I-XXX toolName → result]。
func (s *SentenceStore) AddToolAction(inputID string, toolName string, result string) Sentence {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 去除 inputID 外層方括號以組合完整標籤
	rawID := strings.Trim(inputID, "[]")
	id := fmt.Sprintf("[tool-action: %s %s → %s]", rawID, toolName, result)

	sent := Sentence{
		ID:        id,
		Role:      "tool-action",
		Content:   fmt.Sprintf("%s → %s", toolName, result),
		Timestamp: time.Now(),
	}
	s.sentences = append(s.sentences, sent)
	return sent
}

// ──────────────────────────────────────────────
// 刪除 / 移動
// ──────────────────────────────────────────────

// Delete 依 ID 刪除句子，回傳是否成功找到並刪除。
func (s *SentenceStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sent := range s.sentences {
		if sent.ID == id {
			// 切除該元素，保持原始順序
			s.sentences = append(s.sentences[:i], s.sentences[i+1:]...)
			return true
		}
	}
	return false
}

// Move 將指定 ID 的句子移動到 newPosition（0-based），回傳是否成功。
func (s *SentenceStore) Move(id string, newPosition int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 找到來源索引
	srcIdx := -1
	for i, sent := range s.sentences {
		if sent.ID == id {
			srcIdx = i
			break
		}
	}
	if srcIdx < 0 {
		return false
	}

	// 邊界夾緊
	dst := newPosition
	if dst < 0 {
		dst = 0
	}
	if dst >= len(s.sentences) {
		dst = len(s.sentences) - 1
	}
	if dst == srcIdx {
		return true
	}

	// 取出後插入目標位置
	sent := s.sentences[srcIdx]
	s.sentences = append(s.sentences[:srcIdx], s.sentences[srcIdx+1:]...)

	// 移除元素後，若 dst 在 srcIdx 之後需向前調整一格
	if dst > srcIdx {
		dst--
	}
	// 邊界再次夾緊（移除後長度縮短一）
	if dst > len(s.sentences) {
		dst = len(s.sentences)
	}

	// 在 dst 位置插入元素：複製尾段後重組
	tail := make([]Sentence, len(s.sentences[dst:]))
	copy(tail, s.sentences[dst:])
	s.sentences = append(s.sentences[:dst], sent)
	s.sentences = append(s.sentences, tail...)
	return true
}

// ──────────────────────────────────────────────
// 查詢
// ──────────────────────────────────────────────

// GetAll 回傳所有句子的副本（執行緒安全）。
func (s *SentenceStore) GetAll() []Sentence {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Sentence, len(s.sentences))
	copy(result, s.sentences)
	return result
}

// GetByIDs 依 ID 清單回傳對應句子（保持原始順序）。
func (s *SentenceStore) GetByIDs(ids []string) []Sentence {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 建立 ID 查找集合加速比對
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	var result []Sentence
	for _, sent := range s.sentences {
		if _, ok := idSet[sent.ID]; ok {
			result = append(result, sent)
		}
	}
	return result
}

// CheckSummaryIntegrity 確認摘要群組的所有 ID 在句子庫中仍連續存在。
// 若任一 ID 缺失或順序不連續，回傳 false。
func (s *SentenceStore) CheckSummaryIntegrity(summaryIDs []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(summaryIDs) == 0 {
		return true
	}

	// 收集全域索引
	idIndex := make(map[string]int, len(s.sentences))
	for i, sent := range s.sentences {
		idIndex[sent.ID] = i
	}

	// 確認每個 ID 都存在
	positions := make([]int, 0, len(summaryIDs))
	for _, id := range summaryIDs {
		pos, ok := idIndex[id]
		if !ok {
			return false // 有 ID 已被刪除
		}
		positions = append(positions, pos)
	}

	// 確認連續性：相鄰 position 差值必須為 1
	for i := 1; i < len(positions); i++ {
		if positions[i]-positions[i-1] != 1 {
			return false
		}
	}
	return true
}

// CharCount 計算所有 user / assistant 句子的字元總數（排除 tool-action）。
func (s *SentenceStore) CharCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := 0
	for _, sent := range s.sentences {
		if sent.Role == "user" || sent.Role == "assistant" {
			total += len([]rune(sent.Content))
		}
	}
	return total
}

// ──────────────────────────────────────────────
// 格式化 / 解析
// ──────────────────────────────────────────────

// FormatForTalkFull 將句子序列格式化為 talk_full.md 內容字串。
func FormatForTalkFull(sentences []Sentence) string {
	var sb strings.Builder
	for _, sent := range sentences {
		// 時間戳記行
		ts := sent.Timestamp.Format("2006-01-02 15:04:05")
		sb.WriteString(fmt.Sprintf("\n## %s [%s] %s\n", sent.ID, ts, sent.Role))
		sb.WriteString(sent.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

// ParseTalkFull 將 talk_full.md 的文字內容解析回 []Sentence。
// 用於程序重啟後的記憶恢復。格式假設為 FormatForTalkFull 產生。
func ParseTalkFull(content string) []Sentence {
	var sentences []Sentence
	lines := strings.Split(content, "\n")

	var current *Sentence
	var bodyLines []string

	for _, line := range lines {
		// 偵測段落標題行：## [I-001] [2006-01-02 15:04:05] user
		if strings.HasPrefix(line, "## ") {
			// 儲存上一個句子
			if current != nil {
				current.Content = strings.TrimSpace(strings.Join(bodyLines, "\n"))
				sentences = append(sentences, *current)
			}

			// 解析新標題
			parts := strings.SplitN(strings.TrimPrefix(line, "## "), " ", 3)
			if len(parts) < 3 {
				current = nil
				bodyLines = nil
				continue
			}

			id := parts[0]
			// parts[1] 是時間戳 [2006-01-02，parts[2] 是 15:04:05] role
			// 合並時間部分與 role
			rest := parts[1] + " " + parts[2]
			tsEnd := strings.Index(rest, "]")
			role := ""
			var ts time.Time
			if tsEnd > 0 {
				// rest[:tsEnd+1] = "[2006-01-02 15:04:05]"，去除首尾括號
				tsStr := strings.Trim(rest[:tsEnd+1], "[]")
				ts, _ = time.Parse("2006-01-02 15:04:05", tsStr)
				role = strings.TrimSpace(rest[tsEnd+1:])
			}

			current = &Sentence{
				ID:        id,
				Role:      role,
				Timestamp: ts,
			}
			bodyLines = nil
			continue
		}

		// 收集段落內文
		if current != nil {
			bodyLines = append(bodyLines, line)
		}
	}

	// 儲存最後一個句子
	if current != nil {
		current.Content = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		sentences = append(sentences, *current)
	}

	return sentences
}
