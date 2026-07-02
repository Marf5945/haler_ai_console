package main

// highlight_system_marks.go — 系統側暗標（輔助系統）
//
// 與使用者標註「同一條脊椎、不同檔」：沿用 highlightStore / Highlight /
// 評分 / 摘要 / 本地模型 / 儲存定位，只多三小塊：
//   1) 即時 EnqueueSystemMark：redact + 接地 + 入佇列（不碰大模型，熱路徑零卡）。
//   2) idle  OrganizeSystemMarks：模型抽重點字 + grounding + 去重 + 評分歸組（重，受 idle/電量 gating）。
//   3) 匯入 RevalidateImportedSystemMarks：逐字反驗證，丟掉對不上原文的暗標。
//
// 安全不變式：
//   - 暗標必為對話 redacted 後的逐字 substring（程式驗證，不靠模型自律）。
//   - PII 確定性過濾在抽取「之前」。
//   - llm_output / 外部來源暗標只能當參考，不得獲得命令權限（Provenance 標記）。
//
// 暗標檔 system_marks.json 與 highlights.json 並存於各對話 root；
// 使用者 ListHighlights 永遠過濾掉 System==true，看不到暗標。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"ui_console/shared/controlseal"
)

const systemMarksFilename = "system_marks.json"
const systemMarksPendingFilename = "system_marks_pending.json"
const systemMarkMaxPerMessage = 12 // 資源耗盡防護：每則訊息最多抽幾個暗標

// ───────────────────────── 路徑 ─────────────────────────

func systemMarksPath(agentID string) (string, error) {
	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, systemMarksFilename), nil
}

func systemMarksPendingPath(agentID string) (string, error) {
	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, systemMarksPendingFilename), nil
}

// ───────────────────────── 佇列 ─────────────────────────

// SystemMarkPending：即時層只存「已遮蔽的待整理文字」，idle 才抽取。
type SystemMarkPending struct {
	MessageID string `json:"messageId"`
	Text      string `json:"text"`   // redacted 後文字
	Source    string `json:"source"` // user_text | llm_output
	CreatedAt string `json:"createdAt"`
}

type systemPendingQueue struct {
	Items []SystemMarkPending `json:"items"`
}

func loadSystemPending(agentID string) (*systemPendingQueue, error) {
	path, err := systemMarksPendingPath(agentID)
	if err != nil {
		return nil, err
	}
	q := &systemPendingQueue{}
	raw, rerr := os.ReadFile(path)
	if rerr == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, q) // 壞檔回空，不擋使用者
	}
	return q, nil
}

func saveSystemPending(agentID string, q *systemPendingQueue) error {
	path, err := systemMarksPendingPath(agentID)
	if err != nil {
		return err
	}
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return mkErr
	}
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ───────────────────────── PII / 接地 ─────────────────────────

var (
	reEmail = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	rePhone = regexp.MustCompile(`(?:\+?\d[\d\-\s]{7,}\d)`)
	reLong  = regexp.MustCompile(`\d{9,}`) // 卡號 / 身分證 / 長數字串
)

// redactPII：確定性遮蔽，抽取前先做（不靠模型自律）。
// 同時過 controlseal 擋注入殘留。
func redactPII(text, source string) string {
	src := controlseal.SourceUserRaw
	if source == "llm_output" {
		src = controlseal.SourceToolOutput
	}
	t := controlseal.SanitizeForLLM(src, text).LLMText
	t = reEmail.ReplaceAllString(t, "[REDACTED_EMAIL]")
	t = rePhone.ReplaceAllString(t, "[REDACTED_PHONE]")
	t = reLong.ReplaceAllString(t, "[REDACTED_NUM]")
	return t
}

// grounded：暗標必須是 redacted 來源的逐字 substring，否則丟棄。
func grounded(term, redactedSource string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return false
	}
	if strings.Contains(term, "[REDACTED_") {
		return false // 不收任何含遮蔽標記的片段
	}
	return strings.Contains(redactedSource, term)
}

// ───────────────────────── 即時層（便宜，不碰大模型） ─────────────────────────

// EnqueueSystemMark：使用者輸入 / LLM 定稿時呼叫。
// payload: {"messageId","text","source"}。只 redact + 入佇列。
func (a *App) EnqueueSystemMark(agentID, payload string) error {
	var p struct {
		MessageID string `json:"messageId"`
		Text      string `json:"text"`
		Source    string `json:"source"`
	}
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("system mark payload 解析失敗: %w", err)
	}
	if strings.TrimSpace(p.Text) == "" || p.MessageID == "" {
		return nil // 空的不排
	}
	if p.Source != "llm_output" {
		p.Source = "user_text"
	}
	redacted := redactPII(p.Text, p.Source)

	highlightMu.Lock()
	defer highlightMu.Unlock()

	q, err := loadSystemPending(agentID)
	if err != nil {
		return err
	}
	q.Items = append(q.Items, SystemMarkPending{
		MessageID: p.MessageID,
		Text:      redacted,
		Source:    p.Source,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	return saveSystemPending(agentID, q)
}

// ───────────────────────── idle 層（重，受 gating） ─────────────────────────

// OrganizeSystemMarks：idle 觸發時批次處理佇列。
// 呼叫端（scheduler / 學習模式）負責 idle + 電量 gating，使用者一動就別呼叫。
// 回傳本次新增的暗標數。
func (a *App) OrganizeSystemMarks(agentID string) (int, error) {
	// 1) 取佇列快照 + 使用者群組（短暫持鎖）。
	highlightMu.Lock()
	q, err := loadSystemPending(agentID)
	if err != nil {
		highlightMu.Unlock()
		return 0, err
	}
	if len(q.Items) == 0 {
		highlightMu.Unlock()
		return 0, nil
	}
	snapshot := make([]SystemMarkPending, len(q.Items))
	copy(snapshot, q.Items)
	userStore, _ := loadHighlightStore(agentID)
	highlightMu.Unlock()

	// 2) 模型抽取（lock-free，讓位給前景使用者操作）。
	var candidates []Highlight
	for _, item := range snapshot {
		terms := extractKeyTerms(item.Text) // 模型優先，無模型退化
		seen := map[string]bool{}
		n := 0
		for _, term := range terms {
			if n >= systemMarkMaxPerMessage {
				break
			}
			if !grounded(term, item.Text) {
				continue // 接地：對不上原文就丟
			}
			if seen[term] {
				continue
			}
			seen[term] = true
			n++
			slot, conf := suggestSlotForTerm(term, userStore)
			start := strings.Index(item.Text, term)
			candidates = append(candidates, Highlight{
				ID:             newSystemMarkID(),
				ConversationID: agentID,
				MessageID:      item.MessageID,
				ColorSlot:      slot,
				StartOffset:    runeLen(item.Text[:max(start, 0)]),
				EndOffset:      runeLen(item.Text[:max(start, 0)]) + runeLen(term),
				Quote:          term,
				Weight:         conf,
				Confidence:     conf,
				System:         true,
				Provenance:     item.Source,
				CreatedAt:      time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	// 3) 落地（短暫持鎖）：去重 append + 只移除已處理的佇列項。
	highlightMu.Lock()
	defer highlightMu.Unlock()

	sysPath, err := systemMarksPath(agentID)
	if err != nil {
		return 0, err
	}
	sys, err := loadHighlightStoreAt(sysPath)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, h := range candidates {
		if systemMarkExists(sys, h.MessageID, h.Quote) {
			continue
		}
		sys.Highlights = append(sys.Highlights, h)
		added++
	}
	gcSystemMarks(sys) // 超上限時淘汰低信心 / 最舊
	if err := saveHighlightStoreAt(sysPath, sys); err != nil {
		return 0, err
	}

	// 移除已處理項（保留整理期間新進的佇列項）。
	processed := map[string]bool{}
	for _, it := range snapshot {
		processed[it.CreatedAt+"|"+it.MessageID+"|"+it.Text] = true
	}
	cur, _ := loadSystemPending(agentID)
	remain := cur.Items[:0]
	for _, it := range cur.Items {
		if !processed[it.CreatedAt+"|"+it.MessageID+"|"+it.Text] {
			remain = append(remain, it)
		}
	}
	cur.Items = remain
	if err := saveSystemPending(agentID, cur); err != nil {
		return added, err
	}
	return added, nil
}

// extractKeyTerms：本地模型抽重點字（回 JSON 字串陣列）；不可用時退化成高頻 token。
func extractKeyTerms(redactedText string) []string {
	prompt := "從下面文字挑出最多 8 個重點詞（名詞 / 關鍵概念），" +
		"必須是原文出現過的逐字片段，不要改寫、不要含個資。只輸出 JSON 字串陣列：\n" + redactedText
	if out, err := callLocalModel(prompt, 4*time.Second); err == nil && out != "" {
		if arr := parseStringArray(out); len(arr) > 0 {
			return arr
		}
	}
	// 退化：高頻 token（grounding 仍會再過濾）
	return topTerms([]string{redactedText}, 6)
}

func parseStringArray(s string) []string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(s[start:end+1]), &arr); err != nil {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, t := range arr {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// suggestSlotForTerm：用既有評分把暗標歸到最適合的使用者群組。
// 沒有可比的使用者群組時，回 slot 0、低信心（proto，等使用者真的開組再對應）。
func suggestSlotForTerm(term string, userStore *highlightStore) (int, float64) {
	if userStore == nil {
		return 0, 0.3
	}
	groups := userStore.nonEmptyGroups()
	if len(groups) == 0 {
		return 0, 0.3
	}
	scores := lexicalScores(term, userStore, groups)
	sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })
	best := scores[0]
	conf := best.Score
	if conf < 0.3 {
		conf = 0.3
	}
	return best.ColorSlot, round2(conf)
}

func systemMarkExists(sys *highlightStore, messageID, term string) bool {
	for _, h := range sys.Highlights {
		if h.MessageID == messageID && h.Quote == term {
			return true
		}
	}
	return false
}

// ───────────────────────── 匯入反驗證 ─────────────────────────

// RevalidateImportedSystemMarks：匯入 sub 後呼叫，傳入該對話完整文字。
// 丟掉所有「不是該文字逐字 substring」的暗標，防惡意 sub 夾帶偽造事實。
// 回傳剩餘暗標數。
func (a *App) RevalidateImportedSystemMarks(agentID, conversationText string) (int, error) {
	highlightMu.Lock()
	defer highlightMu.Unlock()

	sysPath, err := systemMarksPath(agentID)
	if err != nil {
		return 0, err
	}
	sys, err := loadHighlightStoreAt(sysPath)
	if err != nil {
		return 0, err
	}
	kept := sys.Highlights[:0]
	for _, h := range sys.Highlights {
		if h.System && strings.Contains(conversationText, h.Quote) {
			kept = append(kept, h)
		} else if !h.System {
			kept = append(kept, h) // 非暗標不在此處理
		}
	}
	sys.Highlights = kept
	if err := saveHighlightStoreAt(sysPath, sys); err != nil {
		return 0, err
	}
	return len(kept), nil
}

// ───────────────────────── 檢視 / 清空（信任 & 除錯） ─────────────────────────

// ListSystemMarks：預設隱藏，但提供入口供除錯 / 稽核。
func (a *App) ListSystemMarks(agentID string) (string, error) {
	highlightMu.Lock()
	defer highlightMu.Unlock()
	sysPath, err := systemMarksPath(agentID)
	if err != nil {
		return "[]", err
	}
	sys, err := loadHighlightStoreAt(sysPath)
	if err != nil {
		return "[]", err
	}
	out, _ := json.Marshal(sys.Highlights)
	return string(out), nil
}

// PurgeSystemMarks：清空某對話全部暗標 + 佇列（對話刪除時級聯呼叫）。
func (a *App) PurgeSystemMarks(agentID string) error {
	highlightMu.Lock()
	defer highlightMu.Unlock()
	sysPath, err := systemMarksPath(agentID)
	if err != nil {
		return err
	}
	if err := saveHighlightStoreAt(sysPath, &highlightStore{}); err != nil {
		return err
	}
	return saveSystemPending(agentID, &systemPendingQueue{})
}

// PurgeMessageMarks：刪除單則訊息時，級聯清掉該 messageId 的使用者標註、
// 系統暗標與待整理佇列。這不是「刪整個對話」；只處理同一個 messageId。
func (a *App) PurgeMessageMarks(agentID, messageID string) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil
	}

	highlightMu.Lock()
	defer highlightMu.Unlock()

	userStore, err := loadHighlightStore(agentID)
	if err != nil {
		return err
	}
	userStore.Highlights = filterHighlightsByMessageID(userStore.Highlights, messageID)
	if err := saveHighlightStore(agentID, userStore); err != nil {
		return err
	}

	sysPath, err := systemMarksPath(agentID)
	if err != nil {
		return err
	}
	sysStore, err := loadHighlightStoreAt(sysPath)
	if err != nil {
		return err
	}
	sysStore.Highlights = filterHighlightsByMessageID(sysStore.Highlights, messageID)
	if err := saveHighlightStoreAt(sysPath, sysStore); err != nil {
		return err
	}

	q, err := loadSystemPending(agentID)
	if err != nil {
		return err
	}
	remain := q.Items[:0]
	for _, item := range q.Items {
		if item.MessageID != messageID {
			remain = append(remain, item)
		}
	}
	q.Items = remain
	return saveSystemPending(agentID, q)
}

func filterHighlightsByMessageID(in []Highlight, messageID string) []Highlight {
	out := in[:0]
	for _, h := range in {
		if h.MessageID != messageID {
			out = append(out, h)
		}
	}
	return out
}

// ───────────────────────── 小工具 ─────────────────────────

var _sysMarkSeq int

func newSystemMarkID() string {
	_sysMarkSeq++
	return fmt.Sprintf("sys_%s_%d", time.Now().UTC().Format("20060102150405"), _sysMarkSeq)
}

func runeLen(s string) int { return len([]rune(s)) }

// ───────────────────────── GC 淘汰 ─────────────────────────

const systemMarkStoreCap = 500 // 每對話暗標上限

// gcSystemMarks：超過上限時，先淘汰最低信心、同信心淘汰最舊。
func gcSystemMarks(sys *highlightStore) {
	if len(sys.Highlights) <= systemMarkStoreCap {
		return
	}
	sort.Slice(sys.Highlights, func(i, j int) bool {
		if sys.Highlights[i].Confidence != sys.Highlights[j].Confidence {
			return sys.Highlights[i].Confidence > sys.Highlights[j].Confidence // 高信心保留
		}
		return sys.Highlights[i].CreatedAt > sys.Highlights[j].CreatedAt // 新的保留
	})
	sys.Highlights = sys.Highlights[:systemMarkStoreCap]
}

// ───────────────────────── acceptance 指標 ─────────────────────────

type highlightStats struct {
	SuggestedTotal int `json:"suggestedTotal"` // 有建議的次數
	Accepted       int `json:"accepted"`       // 使用者採用建議色的次數
	Overridden     int `json:"overridden"`     // 使用者覆寫的次數
}

func highlightStatsPath(agentID string) (string, error) {
	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "highlight_stats.json"), nil
}

// RecordSuggestionOutcome：commit 時回報「建議色 vs 實選色」。
// suggestedSlot < 0 表示當時沒有建議（不列入分母）。
func (a *App) RecordSuggestionOutcome(agentID string, suggestedSlot, chosenSlot int) error {
	if suggestedSlot < 0 {
		return nil
	}
	highlightMu.Lock()
	defer highlightMu.Unlock()

	path, err := highlightStatsPath(agentID)
	if err != nil {
		return err
	}
	st := &highlightStats{}
	if raw, rerr := os.ReadFile(path); rerr == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, st)
	}
	st.SuggestedTotal++
	if suggestedSlot == chosenSlot {
		st.Accepted++
	} else {
		st.Overridden++
	}
	data, _ := json.MarshalIndent(st, "", "  ")
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return mkErr
	}
	tmp := path + ".tmp"
	if werr := os.WriteFile(tmp, data, 0o600); werr != nil {
		return werr
	}
	return os.Rename(tmp, path)
}

// GetHighlightStats：回傳 acceptance 統計（供設定頁 / 除錯）。
func (a *App) GetHighlightStats(agentID string) (string, error) {
	highlightMu.Lock()
	defer highlightMu.Unlock()
	path, err := highlightStatsPath(agentID)
	if err != nil {
		return "{}", err
	}
	raw, rerr := os.ReadFile(path)
	if rerr != nil || len(raw) == 0 {
		return "{}", nil
	}
	return string(raw), nil
}

// PurgeConversationMarks：清空某對話的全部標註衍生資料（使用者標註 + 系統暗標 +
// 佇列 + 統計）。清空 / 刪除整個對話時呼叫；刪 subagent 因整個目錄被移除而自動涵蓋，
// 此方法主要給「清空主對話」(ClearMainTalk) 等保留資料夾的情境。
func (a *App) PurgeConversationMarks(agentID string) error {
	highlightMu.Lock()
	defer highlightMu.Unlock()

	var firstErr error
	keep := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	keep(saveHighlightStore(agentID, &highlightStore{})) // 使用者標註
	if sysPath, err := systemMarksPath(agentID); err == nil {
		keep(saveHighlightStoreAt(sysPath, &highlightStore{})) // 系統暗標
	} else {
		keep(err)
	}
	keep(saveSystemPending(agentID, &systemPendingQueue{})) // 佇列
	if statsPath, err := highlightStatsPath(agentID); err == nil {
		if rmErr := os.Remove(statsPath); rmErr != nil && !os.IsNotExist(rmErr) {
			keep(rmErr) // 統計（不存在不算錯）
		}
	} else {
		keep(err)
	}
	return firstErr
}
