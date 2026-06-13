// web_chain.go — Web Chain 執行記錄與升格判準（TASK 31 衍生 / Web Chain Phase 3）。
// 與 report.go 同風格：只提供可單元測試的「純判準」與聚合，不自行抓資料。
// 為了算「drift 歸零率」需要分母（成功跑了幾次），故 web chain 每次收尾都記一筆
// WebChainRun（含執行簽章與 drift 數），寫在 skill_eval 目錄下獨立檔，
// 不動既有 EventRecord schema。升格只產出「候選訊號」交使用者確認，不自動建 skill。
package skill_eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	SchemaWebChainRunV1 = "skill_eval.web_chain_run.v1"

	// 升格門檻：同簽章累積 >= N 次乾淨跑（drift=0）且完全沒出現過 drift → skill 候選。
	WebChainPromoteCleanRuns = 3
)

// WebChainRun 是一次複合鏈執行的精簡記錄（升格分母 + 月報來源）。
type WebChainRun struct {
	Schema     string    `json:"schema"`
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"session_id,omitempty"`
	Signature  string    `json:"signature"`   // 執行動作序列簽章，如 "網路>網路"
	Steps      int       `json:"steps"`       // 實際執行步數
	DriftCount int       `json:"drift_count"` // 本次與期望鏈不符的步數；0 = 乾淨
}

// WebChainSignature 把執行過的動作序列轉成穩定簽章（升格/聚合的 key）。
// target 含即時關鍵字，刻意不入簽章——同「類」複合問題才能歸併。
func WebChainSignature(actions []string) string {
	parts := make([]string, 0, len(actions))
	for _, a := range actions {
		if a = strings.TrimSpace(a); a != "" {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, ">")
}

// WebChainSigStats 是單一簽章的聚合統計。
type WebChainSigStats struct {
	Signature   string `json:"signature"`
	TotalRuns   int    `json:"total_runs"`
	CleanRuns   int    `json:"clean_runs"`   // drift_count==0
	DriftRuns   int    `json:"drift_runs"`   // drift_count>0
	TotalDrifts int    `json:"total_drifts"` // 各次 drift 數總和
}

// IsSkillCandidate：累積夠多乾淨跑且從未發生 drift → 可升格 skill 候選。
// 「從未 drift」是刻意嚴格：偏離過代表這類問題拆法還不穩，不該固化成 skill。
func (s WebChainSigStats) IsSkillCandidate() bool {
	return s.DriftRuns == 0 && s.CleanRuns >= WebChainPromoteCleanRuns
}

// SummarizeWebChainRuns 聚合一批執行記錄，依簽章彙整（簽章排序，輸出穩定）。
func SummarizeWebChainRuns(runs []WebChainRun) []WebChainSigStats {
	bySig := map[string]*WebChainSigStats{}
	for _, r := range runs {
		st, ok := bySig[r.Signature]
		if !ok {
			st = &WebChainSigStats{Signature: r.Signature}
			bySig[r.Signature] = st
		}
		st.TotalRuns++
		st.TotalDrifts += r.DriftCount
		if r.DriftCount == 0 {
			st.CleanRuns++
		} else {
			st.DriftRuns++
		}
	}
	out := make([]WebChainSigStats, 0, len(bySig))
	for _, st := range bySig {
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Signature < out[j].Signature })
	return out
}

// WebChainSkillCandidates 從聚合結果挑出達升格門檻的簽章。
func WebChainSkillCandidates(stats []WebChainSigStats) []WebChainSigStats {
	var out []WebChainSigStats
	for _, s := range stats {
		if s.IsSkillCandidate() {
			out = append(out, s)
		}
	}
	return out
}

// ── 持久化（與 EventRecord 分檔，互不污染）──

func (s *Store) webChainRunPath() string { return filepath.Join(s.dir, "web_chain_runs.jsonl") }

// AppendWebChainRun 以 O_APPEND 寫入一筆執行記錄（JSONL）。
func (s *Store) AppendWebChainRun(rec WebChainRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec.Schema = SchemaWebChainRunV1
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now().UTC()
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("skill_eval: mkdir web_chain: %w", err)
	}
	f, err := os.OpenFile(s.webChainRunPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("skill_eval: open web_chain: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("skill_eval: marshal web_chain run: %w", err)
	}
	w := bufio.NewWriter(f)
	if _, err := w.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("skill_eval: write web_chain run: %w", err)
	}
	return w.Flush()
}

// LoadWebChainRuns 讀回所有執行記錄（聚合/月報用）；檔案不存在回空。
func (s *Store) LoadWebChainRuns() ([]WebChainRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.webChainRunPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("skill_eval: open web_chain: %w", err)
	}
	defer f.Close()

	var runs []WebChainRun
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r WebChainRun
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue // 壞行跳過，不讓單行毀掉整份聚合
		}
		runs = append(runs, r)
	}
	return runs, sc.Err()
}
