package main

// highlight_binding.go — 路線 B 重點標記 / 自動歸類群組的 Go 後台
//
// 分工：
//   前端只回傳「圈了哪段文字 + 哪個 slot」。顏色不帶語意；
//   群組聚合、混合評分、摘要、記憶權重全在這裡。
//
// 儲存：每個對話 / subagent 一份 highlights.json，落在該對話自己的根目錄
//   （conversationRootForAgent）。主對話 → projectRoot；sub → subagents/callable/<id>/。
//   因此複製 subagent 時色表、摘要、權重會「自動一起被打包」，且天然不跨對話。
//
// 評分：混合制（使用者選定）。先用 cheap lexical 預篩取前 N 組候選，
//   再讓本地 LLM 決勝；本地 LLM 不可用時退化成 lexical 排序；
//   非空群組 < 2 時回空陣列（前端沿用上一次顏色）。
//
// 摘要：RebuildGroupSummary 用本地 LLM 生成；不可用時退化成擷取式（前幾條 quote）。
//
// Wails 綁定：把本檔 App 方法加入 Bind 清單後，前端 wailsjs/go/main/App 會產生
//   SaveHighlight / ListHighlights / DeleteHighlight / SetHighlightWeight /
//   ScoreHighlightGroups / RebuildGroupSummary / ExportGroup。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ui_console/internal/urlsafe"
)

const highlightMaxGroups = 8 // 一色一組，固定 8 組上限

// Highlight 對齊前端 highlightCore.js 的 annotation 欄位。
type Highlight struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversationId"`
	MessageID      string  `json:"messageId"`
	GroupID        string  `json:"groupId"`
	ColorSlot      int     `json:"colorSlot"` // 0..7，純視覺
	StartOffset    int     `json:"startOffset"`
	EndOffset      int     `json:"endOffset"`
	Quote          string  `json:"quote"`
	Weight         float64 `json:"weight"`
	CreatedAt      string  `json:"createdAt"`

	// 雙 track 共用欄位（使用者標註與系統暗標同一資料模型）。
	Provenance string  `json:"provenance,omitempty"` // user_confirmed | user_text | llm_output
	System     bool    `json:"system,omitempty"`     // true = 系統暗標，使用者看不到
	Confidence float64 `json:"confidence,omitempty"` // 系統暗標信心 0..1
}

// HighlightGroup 是「使用者圈在一起」的一袋標註（後台維護摘要 / 關鍵字供評分）。
type HighlightGroup struct {
	GroupID   string   `json:"groupId"`
	ColorSlot int      `json:"colorSlot"`
	Label     string   `json:"label"`
	Summary   string   `json:"summary"`
	KeyTerms  []string `json:"keyTerms"`
	UpdatedAt string   `json:"updatedAt"`
}

type highlightStore struct {
	Highlights []Highlight      `json:"highlights"`
	Groups     []HighlightGroup `json:"groups"`
}

// HighlightScore 是回給前端的單組評分。
type HighlightScore struct {
	GroupID   string  `json:"groupId"`
	ColorSlot int     `json:"colorSlot"`
	Score     float64 `json:"score"`
}

var highlightMu sync.Mutex

// 各對話 / subagent 獨立檔案，隨 sub 打包。
func highlightFileForAgent(agentID string) (string, error) {
	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "highlights.json"), nil
}

// path-based 讀寫：使用者標註與系統暗標共用同一套程式、不同檔。
func loadHighlightStoreAt(path string) (*highlightStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &highlightStore{}, nil
		}
		return nil, err
	}
	st := &highlightStore{}
	if err := json.Unmarshal(data, st); err != nil {
		return &highlightStore{}, nil
	}
	return st, nil
}

func saveHighlightStoreAt(path string, st *highlightStore) error {
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return mkErr
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadHighlightStore(agentID string) (*highlightStore, error) {
	path, err := highlightFileForAgent(agentID)
	if err != nil {
		return nil, err
	}
	return loadHighlightStoreAt(path)
}

func saveHighlightStore(agentID string, st *highlightStore) error {
	path, err := highlightFileForAgent(agentID)
	if err != nil {
		return err
	}
	return saveHighlightStoreAt(path, st)
}

func defaultWeightForSlot(slot int) float64 {
	w := 0.5 + 0.06*float64(slot)
	if w > 1 {
		w = 1
	}
	return w
}

func (st *highlightStore) groupByID(id string) *HighlightGroup {
	for i := range st.Groups {
		if st.Groups[i].GroupID == id {
			return &st.Groups[i]
		}
	}
	return nil
}

func (st *highlightStore) highlightsOfGroup(id string) []Highlight {
	var out []Highlight
	for _, h := range st.Highlights {
		if h.GroupID == id {
			out = append(out, h)
		}
	}
	return out
}

// 非空群組（至少一條標註）。
func (st *highlightStore) nonEmptyGroups() []HighlightGroup {
	counts := map[string]int{}
	for _, h := range st.Highlights {
		counts[h.GroupID]++
	}
	var out []HighlightGroup
	for _, g := range st.Groups {
		if counts[g.GroupID] > 0 {
			out = append(out, g)
		}
	}
	return out
}

// ───────────────────────── CRUD ─────────────────────────

func (a *App) SaveHighlight(payload string) error {
	var h Highlight
	if err := json.Unmarshal([]byte(payload), &h); err != nil {
		return fmt.Errorf("highlight payload 解析失敗: %w", err)
	}
	if h.ID == "" || h.MessageID == "" {
		return fmt.Errorf("highlight 缺少 id 或 messageId")
	}
	if h.ColorSlot < 0 || h.ColorSlot >= highlightMaxGroups {
		return fmt.Errorf("colorSlot 超出範圍 0..%d", highlightMaxGroups-1)
	}
	if h.CreatedAt == "" {
		h.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if h.Weight == 0 {
		h.Weight = defaultWeightForSlot(h.ColorSlot)
	}
	// 使用者 track：標註即「使用者確認」，永不為系統暗標。
	if h.Provenance == "" {
		h.Provenance = "user_confirmed"
	}
	h.System = false

	highlightMu.Lock()
	defer highlightMu.Unlock()

	st, err := loadHighlightStore(h.ConversationID)
	if err != nil {
		return err
	}
	replaced := false
	for i := range st.Highlights {
		if st.Highlights[i].ID == h.ID {
			st.Highlights[i] = h
			replaced = true
			break
		}
	}
	if !replaced {
		st.Highlights = append(st.Highlights, h)
	}
	if h.GroupID != "" && st.groupByID(h.GroupID) == nil {
		st.Groups = append(st.Groups, HighlightGroup{
			GroupID:   h.GroupID,
			ColorSlot: h.ColorSlot,
			UpdatedAt: h.CreatedAt,
		})
	}
	return saveHighlightStore(h.ConversationID, st)
}

func (a *App) ListHighlights(agentID string) (string, error) {
	highlightMu.Lock()
	defer highlightMu.Unlock()
	st, err := loadHighlightStore(agentID)
	if err != nil {
		return "[]", err
	}
	// 防禦性過濾：使用者 track 永遠看不到系統暗標（System==true）。
	visible := make([]Highlight, 0, len(st.Highlights))
	for _, h := range st.Highlights {
		if !h.System {
			visible = append(visible, h)
		}
	}
	out, err := json.Marshal(visible)
	if err != nil {
		return "[]", err
	}
	return string(out), nil
}

func (a *App) DeleteHighlight(agentID, id string) error {
	highlightMu.Lock()
	defer highlightMu.Unlock()
	st, err := loadHighlightStore(agentID)
	if err != nil {
		return err
	}
	next := st.Highlights[:0]
	for _, h := range st.Highlights {
		if h.ID != id {
			next = append(next, h)
		}
	}
	st.Highlights = next
	return saveHighlightStore(agentID, st)
}

func (a *App) SetHighlightWeight(agentID, id string, weight float64) error {
	highlightMu.Lock()
	defer highlightMu.Unlock()
	st, err := loadHighlightStore(agentID)
	if err != nil {
		return err
	}
	for i := range st.Highlights {
		if st.Highlights[i].ID == id {
			st.Highlights[i].Weight = weight
			return saveHighlightStore(agentID, st)
		}
	}
	return fmt.Errorf("highlight 不存在: %s", id)
}

// ───────────────────────── 混合評分 ─────────────────────────

// ScoreHighlightGroups：反白新字詞時呼叫。回傳已排序（高→低）的 JSON。
// 非空群組 < 2 → 回 "[]"（前端沿用上一次顏色）。
func (a *App) ScoreHighlightGroups(agentID, candidate string) (string, error) {
	highlightMu.Lock()
	st, err := loadHighlightStore(agentID)
	highlightMu.Unlock()
	if err != nil {
		return "[]", err
	}
	groups := st.nonEmptyGroups()
	if len(groups) < 2 {
		return "[]", nil // 0~1 組：不評分
	}

	// 1) lexical 預篩（cheap，亦作為無模型時的 fallback）
	lex := lexicalScores(candidate, st, groups)
	sort.Slice(lex, func(i, j int) bool { return lex[i].Score > lex[j].Score })

	// 2) 取前 N 候選送本地 LLM 決勝
	const topN = 3
	cands := lex
	if len(cands) > topN {
		cands = cands[:topN]
	}
	if refined, ok := llmRescore(candidate, st, cands); ok {
		sort.Slice(refined, func(i, j int) bool { return refined[i].Score > refined[j].Score })
		out, _ := json.Marshal(refined)
		return string(out), nil
	}

	// 無模型：回 lexical 排序
	out, _ := json.Marshal(lex)
	return string(out), nil
}

// lexicalScores：候選字詞 vs 各組（摘要 + 關鍵字 + 既有 quote）的 token 重疊度。
// CJK 以字元 bigram、英數以 word 切，兼顧中英。
func lexicalScores(candidate string, st *highlightStore, groups []HighlightGroup) []HighlightScore {
	cand := tokenSet(candidate)
	var scores []HighlightScore
	for _, g := range groups {
		var sb strings.Builder
		sb.WriteString(g.Summary)
		sb.WriteString(" ")
		sb.WriteString(strings.Join(g.KeyTerms, " "))
		for _, h := range st.highlightsOfGroup(g.GroupID) {
			sb.WriteString(" ")
			sb.WriteString(h.Quote)
		}
		gset := tokenSet(sb.String())
		inter := 0
		for t := range cand {
			if gset[t] {
				inter++
			}
		}
		denom := len(cand)
		score := 0.0
		if denom > 0 {
			score = float64(inter) / float64(denom)
		}
		scores = append(scores, HighlightScore{GroupID: g.GroupID, ColorSlot: g.ColorSlot, Score: round2(score)})
	}
	return scores
}

func tokenSet(s string) map[string]bool {
	set := map[string]bool{}
	s = strings.ToLower(strings.TrimSpace(s))
	// 英數 word
	var word strings.Builder
	var cjk []rune
	flushWord := func() {
		if word.Len() > 0 {
			set[word.String()] = true
			word.Reset()
		}
	}
	flushCJK := func() {
		for i := 0; i+1 < len(cjk); i++ {
			set[string(cjk[i:i+2])] = true
		}
		if len(cjk) == 1 {
			set[string(cjk)] = true
		}
		cjk = cjk[:0]
	}
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			flushCJK()
			word.WriteRune(r)
		case r > 0x2E7F: // 粗略視為 CJK / 全形
			flushWord()
			cjk = append(cjk, r)
		default:
			flushWord()
			flushCJK()
		}
	}
	flushWord()
	flushCJK()
	return set
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

// ───────────────────────── 本地 LLM ─────────────────────────

var highlightLocalClient = urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "highlight_local_model", 3*time.Second)

func highlightLocalModel() (model, endpoint string) {
	model = strings.TrimSpace(os.Getenv("HIGHLIGHT_LOCAL_MODEL"))
	if model == "" {
		model = strings.TrimSpace(os.Getenv("STATUS_RAIL_LOCAL_MODEL"))
	}
	if model == "" {
		model = "llama3.2:1b"
	}
	endpoint = strings.TrimSpace(os.Getenv("HIGHLIGHT_LOCAL_ENDPOINT"))
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434/api/generate"
	}
	return model, endpoint
}

type hlOllamaReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}
type hlOllamaResp struct {
	Response string `json:"response"`
}

// callLocalModel：送 prompt 給本地模型，回純文字。逾時 / 不可用回 error。
func callLocalModel(prompt string, timeout time.Duration) (string, error) {
	model, endpoint := highlightLocalModel()
	if !strings.HasPrefix(endpoint, "http://127.0.0.1:") &&
		!strings.HasPrefix(endpoint, "http://localhost:") &&
		!strings.HasPrefix(endpoint, "http://[::1]:") {
		return "", fmt.Errorf("highlight local endpoint 必須為 loopback")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	body, _ := json.Marshal(hlOllamaReq{Model: model, Prompt: prompt, Stream: false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := highlightLocalClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("highlight local model unavailable: %d", resp.StatusCode)
	}
	var decoded hlOllamaResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&decoded); err != nil {
		return "", err
	}
	return strings.TrimSpace(decoded.Response), nil
}

// llmRescore：請本地模型對候選組打分（0..100），回正規化分數。
// 回 (scores, true) 成功；(nil, false) 表示模型不可用 → 由呼叫端走 lexical。
func llmRescore(candidate string, st *highlightStore, cands []HighlightScore) ([]HighlightScore, bool) {
	if len(cands) == 0 {
		return nil, false
	}
	var b strings.Builder
	b.WriteString("你是分類器。判斷詞語最適合歸入哪一個既有群組。\n")
	b.WriteString("只輸出 JSON 陣列，格式 [{\"groupId\":\"...\",\"score\":0-100}]，不要其他文字。\n\n")
	b.WriteString("待分類詞語：")
	b.WriteString(candidate)
	b.WriteString("\n\n候選群組：\n")
	for _, c := range cands {
		g := st.groupByID(c.GroupID)
		summary := ""
		var terms []string
		if g != nil {
			summary = g.Summary
			terms = g.KeyTerms
		}
		if summary == "" {
			// 沒摘要就用前幾條 quote 充當描述
			qs := st.highlightsOfGroup(c.GroupID)
			parts := make([]string, 0, 3)
			for i := 0; i < len(qs) && i < 3; i++ {
				parts = append(parts, qs[i].Quote)
			}
			summary = strings.Join(parts, "；")
		}
		fmt.Fprintf(&b, "- groupId=%s 摘要=%s 關鍵字=%s\n", c.GroupID, summary, strings.Join(terms, ","))
	}
	out, err := callLocalModel(b.String(), 3*time.Second)
	if err != nil || out == "" {
		return nil, false
	}
	jsonPart := extractJSONArray(out)
	if jsonPart == "" {
		return nil, false
	}
	var parsed []struct {
		GroupID string  `json:"groupId"`
		Score   float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &parsed); err != nil {
		return nil, false
	}
	byID := map[string]float64{}
	for _, p := range parsed {
		byID[p.GroupID] = p.Score / 100.0
	}
	result := make([]HighlightScore, 0, len(cands))
	for _, c := range cands {
		s := c.Score // 預設沿用 lexical
		if v, ok := byID[c.GroupID]; ok {
			s = round2(v)
		}
		result = append(result, HighlightScore{GroupID: c.GroupID, ColorSlot: c.ColorSlot, Score: s})
	}
	return result, true
}

// extractJSONArray：從模型回應裡擷取第一個 [...] 區塊。
func extractJSONArray(s string) string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}

// ───────────────────────── 摘要 ─────────────────────────

// RebuildGroupSummary：重算某組摘要 + 關鍵字。本地 LLM 優先，不可用退化成擷取式。
func (a *App) RebuildGroupSummary(agentID, groupID string) (string, error) {
	highlightMu.Lock()
	defer highlightMu.Unlock()

	st, err := loadHighlightStore(agentID)
	if err != nil {
		return "", err
	}
	g := st.groupByID(groupID)
	if g == nil {
		return "", fmt.Errorf("群組不存在: %s", groupID)
	}
	items := st.highlightsOfGroup(groupID)
	if len(items) == 0 {
		g.Summary = ""
		g.KeyTerms = nil
		g.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		_ = saveHighlightStore(agentID, st)
		return "", nil
	}
	quotes := make([]string, 0, len(items))
	for _, h := range items {
		quotes = append(quotes, strings.TrimSpace(h.Quote))
	}

	summary, keyTerms := summarizeQuotes(quotes)
	g.Summary = summary
	g.KeyTerms = keyTerms
	g.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveHighlightStore(agentID, st); err != nil {
		return "", err
	}
	return summary, nil
}

func summarizeQuotes(quotes []string) (string, []string) {
	joined := strings.Join(quotes, "；")
	// 本地 LLM 摘要
	prompt := "用一句話（30 字內）總結這些重點的共同主題，只輸出該句：\n" + joined
	if out, err := callLocalModel(prompt, 3*time.Second); err == nil {
		line := firstLine(out)
		if line != "" && !strings.ContainsAny(line, "{}[]<>") {
			return clip(line, 60), topTerms(quotes, 6)
		}
	}
	// 退化：擷取式（前幾條 quote 串接）
	extract := clip(joined, 60)
	return extract, topTerms(quotes, 6)
}

func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			return ln
		}
	}
	return strings.TrimSpace(s)
}

func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// topTerms：取出現頻率最高的 token 當關鍵字。
func topTerms(quotes []string, n int) []string {
	freq := map[string]int{}
	for _, q := range quotes {
		for t := range tokenSet(q) {
			freq[t]++
		}
	}
	type kv struct {
		k string
		v int
	}
	var arr []kv
	for k, v := range freq {
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

// ───────────────────────── 匯出 ─────────────────────────

// ExportGroup：把某群組標註串成 Markdown，寫成檔案並回傳路徑（可接原生拖曳）。
func (a *App) ExportGroup(agentID, groupID string) (string, error) {
	highlightMu.Lock()
	defer highlightMu.Unlock()

	st, err := loadHighlightStore(agentID)
	if err != nil {
		return "", err
	}
	items := st.highlightsOfGroup(groupID)
	if len(items) == 0 {
		return "", fmt.Errorf("群組沒有標註: %s", groupID)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })

	g := st.groupByID(groupID)
	label := groupID
	summary := ""
	if g != nil {
		if strings.TrimSpace(g.Label) != "" {
			label = g.Label
		}
		summary = g.Summary
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# 重點群組：%s\n\n", label)
	if summary != "" {
		fmt.Fprintf(&b, "> 摘要：%s\n", summary)
	}
	fmt.Fprintf(&b, "> 來源對話：%s ・ 共 %d 條重點\n\n", agentID, len(items))
	for i, h := range items {
		fmt.Fprintf(&b, "%d. %s\n", i+1, strings.TrimSpace(h.Quote))
		fmt.Fprintf(&b, "   <sub>來源訊息：%s ・ 權重：%.2f</sub>\n\n", h.MessageID, h.Weight)
	}

	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return "", err
	}
	outDir := filepath.Join(root, "highlights_exports")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("重點群組_%s.md", sanitizeHighlightName(label)))
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

func sanitizeHighlightName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "untitled"
	}
	repl := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_", "\n", " ", "\t", " ")
	out := repl.Replace(s)
	if len([]rune(out)) > 80 {
		out = string([]rune(out)[:80])
	}
	return out
}
