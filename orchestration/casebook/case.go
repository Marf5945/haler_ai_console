// case.go — 案例庫的核心型別（v0 骨架）。
//
// 設計目標：把一次任務執行的「預期 → 實際 → 判定 → 標籤 → 摘要」收斂成
// 可分群、可回放的軌跡。沿用 repo 既有 jsonl + atomic 落盤慣例，零外部依賴。
//
// 兩層落盤：
//   - 節點級 → casebook.jsonl（CaseRecord，看得出哪一步、為何成敗）
//   - run 級 → 既有 dag_runs/index.json（DAGRunSummary，沿用不動）
package casebook

import "strings"

// Verdict 是「實際 vs 預期」的判定結果。分層判定：predicate 命中 → pass/fail；
// predicate 驗不了 → 退回模型自評，模型給不出明確結論時記 partial。
type Verdict string

const (
	VerdictPass    Verdict = "pass"    // 達成預期（predicate 通過，或模型明確判通過）
	VerdictFail    Verdict = "fail"    // 未達成（predicate 不過，或執行報錯）
	VerdictPartial Verdict = "partial" // 部分達成 / 無法判定（fail-open，不擋路但留痕）
)

// 受控核心 tag 字典（固定 enum）。規則優先指派，保證分群乾淨、可統計。
// 模型可在此之外補 free-text tag，但核心統計只認這份清單。
const (
	TagSuccess     = "success"      // 節點達成預期
	TagTimeout     = "timeout"      // 逾時 / 輪數預算耗盡
	TagAPIError    = "api_error"    // 外部呼叫 4xx/5xx/連線失敗
	TagWrongOutput = "wrong_output" // 有產出但 predicate 不過（格式/數量/內容不符）
	TagMissingStep = "missing_step" // 宣稱完成但紀錄裡缺對應動作
	TagBlocked     = "blocked"      // 需人工審核 / 風險閘擋下
	TagCancelled   = "cancelled"    // 使用者或系統取消
	TagUnknown     = "unknown"      // 規則命中不了、模型也沒結論
)

// CoreTags 是固定核心字典；GroupByTag 統計時用它界定「群」。
var CoreTags = []string{
	TagSuccess, TagTimeout, TagAPIError, TagWrongOutput,
	TagMissingStep, TagBlocked, TagCancelled, TagUnknown,
}

// IsCoreTag 回報 tag 是否屬於受控核心字典。
func IsCoreTag(tag string) bool {
	for _, t := range CoreTags {
		if t == tag {
			return true
		}
	}
	return false
}

// CaseRecord 是案例庫的一筆軌跡（節點級）。刻意精簡：足以回答
// 「這一步預期什麼、實際如何、成敗、屬於哪一群」。
type CaseRecord struct {
	RunID   string `json:"run_id"`
	NodeID  string `json:"node_id"`
	Goal    string `json:"goal"`               // run 的目標摘要（來自 GoalContract）
	Title   string `json:"title,omitempty"`    // 節點標題
	GroupID string `json:"group_id,omitempty"` // 同一原始任務的批次分組鍵（預設=RunID）

	Expected  string `json:"expected,omitempty"`  // plan 階段 planner 給的一句話預期
	Predicate string `json:"predicate,omitempty"` // 可機器驗的完成條件（見 predicate.go）
	Actual    string `json:"actual,omitempty"`    // sanitized 後的實際觀察摘要

	Verdict Verdict  `json:"verdict"`        // pass / fail / partial
	Tags    []string `json:"tags,omitempty"` // 核心 tag 在前，模型補的 free-text 在後
	Summary string   `json:"summary,omitempty"`

	CreatedAt string `json:"created_at"`
}

// CoreTag 回傳此筆紀錄的主群鍵：tags 中第一個落在核心字典的值；
// 都不是核心則回 TagUnknown。GroupByTag 用它歸群。
func (r CaseRecord) CoreTag() string {
	for _, t := range r.Tags {
		if IsCoreTag(t) {
			return t
		}
	}
	return TagUnknown
}

// IsSuccess 是否屬於成功案例群。
func (r CaseRecord) IsSuccess() bool {
	return r.Verdict == VerdictPass
}

// 大欄位落盤上限（rune）。確保單行 jsonl 遠小於 LoadRecent 的 scanner 上限，
// 避免一筆超長紀錄讓掃描中斷、後面的有效案例讀不到。
const (
	maxActualRunes   = 2000
	maxSummaryRunes  = 1000
	maxTextRunes     = 500 // goal / title / expected / predicate
	maxTagCount      = 16
	maxTagValueRunes = 64
)

// truncateRunes 以 rune 為單位截斷，避免切斷多位元組字元。
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// clampFields 在落盤前夾住所有大欄位與 tag 數量／長度（防爆行）。
func (r CaseRecord) clampFields() CaseRecord {
	r.Goal = truncateRunes(r.Goal, maxTextRunes)
	r.Title = truncateRunes(r.Title, maxTextRunes)
	r.Expected = truncateRunes(r.Expected, maxTextRunes)
	r.Predicate = truncateRunes(r.Predicate, maxTextRunes)
	r.Actual = truncateRunes(r.Actual, maxActualRunes)
	r.Summary = truncateRunes(r.Summary, maxSummaryRunes)
	if len(r.Tags) > maxTagCount {
		r.Tags = r.Tags[:maxTagCount]
	}
	for i, t := range r.Tags {
		r.Tags[i] = truncateRunes(t, maxTagValueRunes)
	}
	return r
}

// normalizeTags 去空白、去重、核心 tag 排前面（穩定分群）。
func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	var core, extra []string
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		if IsCoreTag(t) {
			core = append(core, t)
		} else {
			extra = append(extra, t)
		}
	}
	return append(core, extra...)
}
