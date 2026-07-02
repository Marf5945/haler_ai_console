// document_sections.go — 文檔段落標籤快取（一次建圖、之後零依賴檢索）。
//
// 目標：引用文檔第一次匯入時，把全文切成「段落」並替每段打標籤 + 算詞頻，
// 存成 sidecar（<docID>.sections.json）。之後查詢時用純 Go BM25 只挑出相關
// 段落，連同一份壓縮目錄餵給模型，不再餵全文 → 省 token、省時間。
//
// 設計（呼應暗標系統 highlight_system_marks.go 的三原則）：
//  1. 規則先行、熱路徑便宜：切分 + 規則分類全是 stdlib，零模型、零外部服務。
//  2. idle 才補強：規則判不出的段落標 LabelSource="pending"，交 BackfillPendingLabels
//     在 idle 時用 app 注入的本地模型補強（builtin 不直接依賴 LLM / UI）。
//  3. 增量重建：沿用 vector 索引同套策略——schema/sectioner 版本或 content hash
//     不符才重建（SectionIndexNeedsRebuild）。
//
// 零第三方相依：只用 stdlib（math / regexp / sort / strings / json / os …）與
// 本套件既有 helper（tokenizeForVector / sha256Hex / snippet / runeLen）。
package builtin

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// 版本標記：改切分演算法時 bump SectionerVersion，所有舊 sidecar 自動視為過期。
const (
	DocSectionSchema = "doc_sections.v1"
	SectionerVersion = "heading-v1"
)

// ───────────────────────── 封閉標籤集 ─────────────────────────

// SectionLabel 段落類型（封閉集；集合外的輸出一律落 LabelOther）。
type SectionLabel string

const (
	LabelDefinition SectionLabel = "定義"
	LabelData       SectionLabel = "數據"
	LabelConclusion SectionLabel = "結論"
	LabelSteps      SectionLabel = "步驟"
	LabelCode       SectionLabel = "程式碼"
	LabelExample    SectionLabel = "範例"
	LabelQuote      SectionLabel = "引述"
	LabelFAQ        SectionLabel = "問答"
	LabelOther      SectionLabel = "其他"
)

// AllSectionLabels 供模型補強時驗收輸出（在集合內才採用）。
var AllSectionLabels = []SectionLabel{
	LabelDefinition, LabelData, LabelConclusion, LabelSteps,
	LabelCode, LabelExample, LabelQuote, LabelFAQ, LabelOther,
}

// IsValidSectionLabel 回報字串是否為合法標籤。
func IsValidSectionLabel(s string) bool {
	for _, l := range AllSectionLabels {
		if string(l) == s {
			return true
		}
	}
	return false
}

// ───────────────────────── 資料結構 ─────────────────────────

// DocSection 一個段落（標題層級切出）及其標籤 / 詞頻。
// StartRune / EndRune 是對「原始 content」的 rune offset（與標註系統同慣例）。
type DocSection struct {
	ID          string         `json:"id"`
	Level       int            `json:"level"` // 0 = 前言 / 無標題
	Heading     string         `json:"heading"`
	HeadingPath []string       `json:"heading_path"` // 麵包屑（含本層）
	StartRune   int            `json:"start_rune"`
	EndRune     int            `json:"end_rune"`
	CharCount   int            `json:"char_count"`
	TokenLen    int            `json:"token_len"`
	Label       SectionLabel   `json:"label"`
	LabelSource string         `json:"label_source"` // rule | model | pending
	Keywords    []string       `json:"keywords"`
	TermFreq    map[string]int `json:"term_freq"`
}

// DocSectionIndex 一份文檔的段落標籤快取（sidecar 落地用）。
type DocSectionIndex struct {
	Schema           string         `json:"schema"`
	SectionerVersion string         `json:"sectioner_version"`
	DocID            string         `json:"doc_id"`
	ContentHash      string         `json:"content_hash"`
	BuiltAt          time.Time      `json:"built_at"`
	AvgTokenLen      float64        `json:"avg_token_len"`
	DocFreq          map[string]int `json:"doc_freq"` // df：含該 term 的段落數（BM25 用）
	Sections         []DocSection   `json:"sections"`
}

// ───────────────────────── 切分（標題層級） ─────────────────────────

var headingRe = regexp.MustCompile(`^(#{1,6})[ \t]+(\S.*?)[ \t]*$`)

// SplitSectionsByHeading 依 markdown 標題切段；無任何標題則退化成空行段落切。
// 每段帶 rune offset 與麵包屑（HeadingPath）。
func SplitSectionsByHeading(content string) []DocSection {
	type hpos struct {
		rune  int
		level int
		title string
	}
	var heads []hpos
	offset := 0
	for _, raw := range strings.Split(content, "\n") {
		ln := strings.TrimRight(raw, "\r")
		if m := headingRe.FindStringSubmatch(ln); m != nil {
			heads = append(heads, hpos{rune: offset, level: len(m[1]), title: m[2]})
		}
		offset += len([]rune(raw)) + 1 // +1 補回 split 吃掉的 '\n'
	}
	if len(heads) == 0 {
		return splitByBlankLine(content)
	}

	runes := []rune(content)
	total := len(runes)
	var sections []DocSection

	// 前言（第一個標題之前若有非空白內容）。
	if strings.TrimSpace(string(runes[:heads[0].rune])) != "" {
		sections = append(sections, DocSection{Level: 0, StartRune: 0, EndRune: heads[0].rune})
	}

	var stack []hpos // 麵包屑
	for i, h := range heads {
		end := total
		if i+1 < len(heads) {
			end = heads[i+1].rune
		}
		for len(stack) > 0 && stack[len(stack)-1].level >= h.level {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, h)
		path := make([]string, 0, len(stack))
		for _, s := range stack {
			path = append(path, s.title)
		}
		sections = append(sections, DocSection{
			Level: h.level, Heading: h.title, HeadingPath: path,
			StartRune: h.rune, EndRune: end,
		})
	}
	return sections
}

// splitByBlankLine 無標題文檔的退化切法：連續非空白行為一段，空行為界。
func splitByBlankLine(content string) []DocSection {
	runes := []rune(content)
	var sections []DocSection
	add := func(s, e int) {
		if strings.TrimSpace(string(runes[s:e])) != "" {
			sections = append(sections, DocSection{Level: 0, StartRune: s, EndRune: e})
		}
	}
	offset := 0
	blockStart := -1
	for _, raw := range strings.Split(content, "\n") {
		blank := strings.TrimSpace(strings.TrimRight(raw, "\r")) == ""
		if blank {
			if blockStart >= 0 {
				add(blockStart, offset)
				blockStart = -1
			}
		} else if blockStart < 0 {
			blockStart = offset
		}
		offset += len([]rune(raw)) + 1
	}
	if blockStart >= 0 {
		add(blockStart, len(runes))
	}
	if len(sections) == 0 && strings.TrimSpace(content) != "" {
		sections = append(sections, DocSection{Level: 0, StartRune: 0, EndRune: len(runes)})
	}
	return sections
}

// ───────────────────────── 建索引 ─────────────────────────

// BuildDocSectionIndex 切段 + 詞頻 + 規則分類，組成可落地的 sidecar。
// 不呼叫模型——pending 段落留給 BackfillPendingLabels 在 idle 補強。
func BuildDocSectionIndex(docID, content string) *DocSectionIndex {
	runes := []rune(content)
	secs := SplitSectionsByHeading(content)
	df := map[string]int{}
	totalTok := 0

	for i := range secs {
		s := &secs[i]
		s.ID = fmt.Sprintf("s%03d", i)
		if s.EndRune > len(runes) {
			s.EndRune = len(runes)
		}
		text := string(runes[s.StartRune:s.EndRune])
		s.CharCount = runeLen(strings.TrimSpace(text))

		toks := tokenizeForVector(text)
		s.TokenLen = len(toks)
		totalTok += len(toks)
		tf := map[string]int{}
		for _, t := range toks {
			tf[t]++
		}
		s.TermFreq = tf
		for t := range tf {
			df[t]++
		}
		s.Keywords = topNTerms(tf, 8)

		if label, ok := classifySectionRule(s.HeadingPath, s.Heading, text); ok {
			s.Label = label
			s.LabelSource = "rule"
		} else {
			s.Label = LabelOther
			s.LabelSource = "pending" // idle 交模型補強
		}
	}

	avg := 0.0
	if len(secs) > 0 {
		avg = float64(totalTok) / float64(len(secs))
	}
	return &DocSectionIndex{
		Schema:           DocSectionSchema,
		SectionerVersion: SectionerVersion,
		DocID:            docID,
		ContentHash:      sha256Hex(content),
		BuiltAt:          time.Now(),
		AvgTokenLen:      avg,
		DocFreq:          df,
		Sections:         secs,
	}
}

// topNTerms 取詞頻最高的 n 個 term（跳過單字元雜訊，偏好雙字 bigram / 英數詞）。
func topNTerms(tf map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	arr := make([]kv, 0, len(tf))
	for k, v := range tf {
		if len([]rune(k)) < 2 {
			continue
		}
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].v != arr[j].v {
			return arr[i].v > arr[j].v
		}
		return arr[i].k < arr[j].k
	})
	out := make([]string, 0, n)
	for i := 0; i < len(arr) && i < n; i++ {
		out = append(out, arr[i].k)
	}
	return out
}

// ───────────────────────── 規則分類 ─────────────────────────

var reOrderedStep = regexp.MustCompile(`(?m)^\s*(\d+[.)、]|[-*+]\s)`)

// classifySectionRule 規則先行分類。ok=false 表示判不出（→ pending 交模型）。
// 標題關鍵字最優先，其次內文結構特徵（程式碼圍欄 / 引述 / 條列 / 數字密度 / 表格）。
func classifySectionRule(path []string, heading, text string) (SectionLabel, bool) {
	hay := strings.ToLower(strings.Join(append(append([]string{}, path...), heading), " "))
	lower := strings.ToLower(text)
	has := func(keys ...string) bool {
		for _, k := range keys {
			kk := strings.ToLower(k)
			if strings.Contains(hay, kk) || strings.Contains(lower, kk) {
				return true
			}
		}
		return false
	}

	switch {
	case strings.Contains(hay, "faq") || strings.Contains(hay, "問答") || strings.Contains(hay, "常見問題"):
		return LabelFAQ, true
	case has("結論", "總結", "小結", "conclusion", "summary"):
		return LabelConclusion, true
	case has("定義", "什麼是", "何謂", "definition", "概念"):
		return LabelDefinition, true
	case has("步驟", "流程", "操作步驟", "step", "how to", "教學", "安裝"):
		return LabelSteps, true
	case has("範例", "例如", "舉例", "example", "案例"):
		return LabelExample, true
	}

	if strings.Contains(text, "```") {
		return LabelCode, true
	}
	if strongQuote(text) {
		return LabelQuote, true
	}
	if countMatches(reOrderedStep, text) >= 3 {
		return LabelSteps, true
	}
	if digitRatio(text) >= 0.18 || tableLike(text) {
		return LabelData, true
	}
	return LabelOther, false
}

func strongQuote(body string) bool {
	q, n := 0, 0
	for _, ln := range strings.Split(body, "\n") {
		t := strings.TrimSpace(strings.TrimRight(ln, "\r"))
		if t == "" {
			continue
		}
		n++
		if strings.HasPrefix(t, ">") {
			q++
		}
	}
	return n > 0 && float64(q)/float64(n) >= 0.5
}

func countMatches(re *regexp.Regexp, s string) int {
	return len(re.FindAllStringIndex(s, -1))
}

func digitRatio(s string) float64 {
	var d, tot int
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			continue
		}
		tot++
		if r >= '0' && r <= '9' {
			d++
		}
	}
	if tot == 0 {
		return 0
	}
	return float64(d) / float64(tot)
}

func tableLike(body string) bool {
	pipe := 0
	for _, ln := range strings.Split(body, "\n") {
		if strings.Count(ln, "|") >= 2 {
			pipe++
		}
	}
	return pipe >= 2
}

// ───────────────────────── BM25 檢索（零依賴） ─────────────────────────

const (
	bm25K1 = 1.5
	bm25B  = 0.75
)

// SectionHit 一筆命中段落 + 分數。
type SectionHit struct {
	Section *DocSection
	Score   float64
}

// Rank 用 BM25 對所有段落打分，回傳分數 > 0 者由高到低；k>0 時截前 k 筆（k<=0 全回）。
// 單篇段落數少，直接線性掃；跨多文檔擴充時再加倒排索引。
func (idx *DocSectionIndex) Rank(query string, k int) []SectionHit {
	if idx == nil || len(idx.Sections) == 0 {
		return nil
	}
	qToks := dedupeTokens(tokenizeForVector(query))
	N := float64(len(idx.Sections))
	avg := idx.AvgTokenLen
	if avg <= 0 {
		avg = 1
	}
	hits := make([]SectionHit, 0, len(idx.Sections))
	for i := range idx.Sections {
		s := &idx.Sections[i]
		dl := float64(s.TokenLen)
		var score float64
		for _, t := range qToks {
			tf := float64(s.TermFreq[t])
			if tf == 0 {
				continue
			}
			df := float64(idx.DocFreq[t])
			idf := math.Log(1 + (N-df+0.5)/(df+0.5))
			score += idf * (tf * (bm25K1 + 1)) / (tf + bm25K1*(1-bm25B+bm25B*dl/avg))
		}
		if score > 0 {
			hits = append(hits, SectionHit{Section: s, Score: score})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Section.ID < hits[j].Section.ID
	})
	if k > 0 && len(hits) > k {
		hits = hits[:k]
	}
	return hits
}

func dedupeTokens(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, t := range in {
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// ───────────────────────── 餵模型用：目錄 + 命中段落 ─────────────────────────

// SectionExcerpt 要餵給模型的單一段落內容。
type SectionExcerpt struct {
	ID      string       `json:"id"`
	Heading string       `json:"heading"`
	Label   SectionLabel `json:"label"`
	Text    string       `json:"text"`
	Score   float64      `json:"score"`
}

// TOC 產生極小目錄：每段一行（縮排表層級 + 標籤 + 首行摘要），第一次餵一次給
// 模型建立全局意識，之後只補命中段落即可。
func (idx *DocSectionIndex) TOC(content string) string {
	runes := []rune(content)
	var b strings.Builder
	for i := range idx.Sections {
		s := &idx.Sections[i]
		indent := strings.Repeat("  ", max(s.Level-1, 0))
		head := s.Heading
		if head == "" {
			head = "（前言）"
		}
		end := s.EndRune
		if end > len(runes) {
			end = len(runes)
		}
		first := snippet(string(runes[s.StartRune:end]))
		if runeLen(first) > 60 {
			first = string([]rune(first)[:60]) + "…"
		}
		fmt.Fprintf(&b, "%s- [%s] %s（%s）：%s\n", indent, s.ID, head, s.Label, first)
	}
	return b.String()
}

// ContextOptions 控制 BuildContext 的取段與輸出。
type ContextOptions struct {
	K               int                 // 取前 K 段（<=0 不限段數，僅受 MaxTotalRunes 約束）
	MaxSectionRunes int                 // 單段上限；超過取「命中附近視窗」（<=0 不截）
	MaxTotalRunes   int                 // 所有 picks 文字總量上限（<=0 不限）
	PreferLabel     SectionLabel        // 軟加權目標標籤（空字串略過）
	PreferBoost     float64             // 軟加權乘係數，建議 1.2；<=0 視為 1（不加權）
	Sanitize        func(string) string // egress 清洗（toc 與每段 Text）；nil 表不清洗
}

// BuildContext 回傳「壓縮目錄 + 命中段落」，取代餵全文。
// 設計鎖定：
//   - BM25 為主判斷；PreferLabel 只做「軟加權」（乘係數）不硬過濾，高分的非偏好段落仍能贏。
//   - 長段落以「命中附近視窗」截斷（MaxSectionRunes），不整段丟。
//   - picks 文字總量受 MaxTotalRunes 約束。
//   - toc 與每段 Text 回傳前都過 opts.Sanitize（注入式，builtin 不依賴 controlseal）。
func (idx *DocSectionIndex) BuildContext(content, query string, opts ContextOptions) (toc string, picks []SectionExcerpt) {
	if idx == nil {
		return "", nil
	}
	clean := opts.Sanitize
	if clean == nil {
		clean = func(s string) string { return s } // 預設不清洗；正式呼叫應注入 controlseal
	}
	toc = clean(idx.TOC(content))

	hits := idx.Rank(query, 0) // 先全算
	boost := opts.PreferBoost
	if boost <= 0 {
		boost = 1
	}
	if opts.PreferLabel != "" && boost != 1 {
		for i := range hits {
			if hits[i].Section.Label == opts.PreferLabel {
				hits[i].Score *= boost
			}
		}
		sort.SliceStable(hits, func(i, j int) bool {
			if hits[i].Score != hits[j].Score {
				return hits[i].Score > hits[j].Score
			}
			return hits[i].Section.ID < hits[j].Section.ID
		})
	}

	qToks := dedupeTokens(tokenizeForVector(query))
	runes := []rune(content)
	total := 0
	for i := 0; i < len(hits); i++ {
		if opts.K > 0 && len(picks) >= opts.K {
			break
		}
		if opts.MaxTotalRunes > 0 && total >= opts.MaxTotalRunes {
			break
		}
		s := hits[i].Section
		end := s.EndRune
		if end > len(runes) {
			end = len(runes)
		}
		text := strings.TrimSpace(string(runes[s.StartRune:end]))
		if opts.MaxSectionRunes > 0 && runeLen(text) > opts.MaxSectionRunes {
			text = hitWindow(text, qToks, opts.MaxSectionRunes)
		}
		if opts.MaxTotalRunes > 0 {
			remain := opts.MaxTotalRunes - total
			if remain <= 0 {
				break
			}
			if runeLen(text) > remain {
				text = string([]rune(text)[:remain]) + "…"
			}
		}
		text = clean(text)
		picks = append(picks, SectionExcerpt{
			ID:      s.ID,
			Heading: s.Heading,
			Label:   s.Label,
			Text:    text,
			Score:   hits[i].Score,
		})
		total += runeLen(text)
	}
	return toc, picks
}

// hitWindow 在 text 內找最早命中的 query 詞位置，取其前後合計 maxRunes 的視窗（頭尾補 …）。
// 找不到命中時退化為取段首 maxRunes。
func hitWindow(text string, qToks []string, maxRunes int) string {
	tr := []rune(text)
	if len(tr) <= maxRunes {
		return text
	}
	lower := strings.ToLower(text)
	hit := -1
	for _, t := range qToks {
		if t == "" {
			continue
		}
		if bidx := strings.Index(lower, t); bidx >= 0 {
			r := runeLen(lower[:bidx])
			if hit < 0 || r < hit {
				hit = r
			}
		}
	}
	if hit < 0 {
		return string(tr[:maxRunes]) + "…"
	}
	start := hit - maxRunes/2
	if start < 0 {
		start = 0
	}
	end := start + maxRunes
	if end > len(tr) {
		end = len(tr)
		start = end - maxRunes
		if start < 0 {
			start = 0
		}
	}
	out := string(tr[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(tr) {
		out = out + "…"
	}
	return out
}

// ───────────────────────── idle 模型補強 ─────────────────────────

// SectionClassifier 由 app 層注入（內部呼叫本地模型），builtin 不直接依賴 LLM / UI。
// 回 ok=false 代表模型也無法判定，維持 pending。
type SectionClassifier func(headingPath []string, text string) (SectionLabel, bool)

// BackfillPendingLabels idle 時批次補強 pending 段落；回補強成功數。
// 模型輸出須落在封閉集（IsValidSectionLabel）才採用，否則維持 pending。
func (idx *DocSectionIndex) BackfillPendingLabels(content string, classify SectionClassifier) int {
	if idx == nil || classify == nil {
		return 0
	}
	runes := []rune(content)
	n := 0
	for i := range idx.Sections {
		s := &idx.Sections[i]
		if s.LabelSource != "pending" {
			continue
		}
		end := s.EndRune
		if end > len(runes) {
			end = len(runes)
		}
		label, ok := classify(s.HeadingPath, string(runes[s.StartRune:end]))
		if !ok || !IsValidSectionLabel(string(label)) {
			continue
		}
		s.Label = label
		s.LabelSource = "model"
		n++
	}
	return n
}

// ───────────────────────── 落地 / 增量重建 ─────────────────────────

// DocSectionIndexPath sidecar 路徑：<dir>/<docID>.sections.json。
func DocSectionIndexPath(dir, docID string) string {
	return filepath.Join(dir, docID+".sections.json")
}

// SaveDocSectionIndex 原子寫入 sidecar。
func SaveDocSectionIndex(dir string, idx *DocSectionIndex) error {
	if idx == nil {
		return fmt.Errorf("document_sections: nil index")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	path := DocSectionIndexPath(dir, idx.DocID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadDocSectionIndex 讀 sidecar；不存在 / 壞檔回 ok=false（呼叫端重建）。
func LoadDocSectionIndex(dir, docID string) (*DocSectionIndex, bool) {
	raw, err := os.ReadFile(DocSectionIndexPath(dir, docID))
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	var idx DocSectionIndex
	if json.Unmarshal(raw, &idx) != nil {
		return nil, false
	}
	return &idx, true
}

// SectionIndexNeedsRebuild schema / sectioner 版本或內容雜湊不符就重建
// （沿用 vector 索引同套增量策略）。
func SectionIndexNeedsRebuild(existing *DocSectionIndex, content string) bool {
	if existing == nil {
		return true
	}
	if existing.Schema != DocSectionSchema || existing.SectionerVersion != SectionerVersion {
		return true
	}
	if existing.ContentHash == "" || existing.ContentHash != sha256Hex(content) {
		return true
	}
	return false
}

// EnsureDocSectionIndex 有效就讀，否則重建並存。這是「第一次建快取、之後沿用」的入口。
func EnsureDocSectionIndex(dir, docID, content string) (*DocSectionIndex, error) {
	if idx, ok := LoadDocSectionIndex(dir, docID); ok && !SectionIndexNeedsRebuild(idx, content) {
		return idx, nil
	}
	idx := BuildDocSectionIndex(docID, content)
	if err := SaveDocSectionIndex(dir, idx); err != nil {
		return idx, err
	}
	return idx, nil
}
