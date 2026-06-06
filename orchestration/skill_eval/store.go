// store.go — skill_eval 的 append-only JSONL 持久化（TASK 31 / Phase 1.3）。
// 路徑刻意放在 localsearch 掃描範圍外（不變式 1/2）：
//   <dataRoot>/data/projects/<project>/skill_eval/events.jsonl
// shouldSkipDir 已將 "skill_eval" 列入跳過，確保事件不會被當語料餵回 LLM。
package skill_eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	SchemaEventV1  = "skill_eval.event.v1"  // 每筆事件的 schema 標記（避免重演 DAG 雙格式同檔名）
	SchemaReportV1 = "skill_eval.report.v1" // 月報記錄 schema（Phase 3）
)

// EventRecord 是一筆評量事件（drift / score / candidate）。
type EventRecord struct {
	Schema    string       `json:"schema"`              // 固定 SchemaEventV1
	Timestamp time.Time    `json:"timestamp"`           // UTC
	SessionID string       `json:"session_id,omitempty"`
	SkillID   string       `json:"skill_id,omitempty"`
	Drifts    []DriftEvent `json:"drifts,omitempty"`    // 本次評量產生的 drift
	Score     float64      `json:"score,omitempty"`     // 相似度/評分（Phase 1.5）
	Note      string       `json:"note,omitempty"`
}

// Store 負責把事件寫進獨立 JSONL 目錄。
type Store struct {
	mu  sync.Mutex
	dir string // <dataRoot>/data/projects/<project>/skill_eval
}

// NewStore 建立 store；dir 不存在時惰性建立（0700）。
func NewStore(dataRoot, project string) *Store {
	if project == "" {
		project = "default"
	}
	return &Store{dir: filepath.Join(dataRoot, "data", "projects", project, "skill_eval")}
}

// Dir 回傳事件目錄（供測試與排除規則驗證使用）。
func (s *Store) Dir() string { return s.dir }

// AppendEvent 以 O_APPEND 寫入一行 JSON（JSONL），比照 skill_step/audit.go 模式。
func (s *Store) AppendEvent(rec EventRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec.Schema = SchemaEventV1 // 強制標記，呼叫端不需手動帶
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now().UTC()
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("skill_eval: mkdir store: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(s.dir, "events.jsonl"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("skill_eval: open store: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("skill_eval: marshal event: %w", err)
	}
	if _, err := w.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("skill_eval: write event: %w", err)
	}
	return w.Flush()
}
