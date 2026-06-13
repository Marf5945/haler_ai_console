// app_search.go - split out of app.go (same package, codemod).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/data/conversation"
	"ui_console/data/storage"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/localsearch"
	"ui_console/shared/websearch"
)

func (a *App) maybeHandleLocalSearch(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	if isLearningOperationCatalogText(userText) {
		return nil, false
	}
	if target, ok := parseMemoryExpandRequest(userText); ok {
		if handled, resp := a.maybeExpandMemory(memoryExpandActionLabel, target, traceID); handled {
			return &resp, true
		}
	}
	if resp, ok := a.maybeHandlePendingLocalSearchWebFallback(userText, sessionID, traceID); ok {
		return resp, true
	}
	if resp, ok := a.maybeHandleWebSearch(userText, sessionID, traceID); ok {
		return resp, true
	}
	// 「某個 skill 怎麼用」→ 直接讀該 skill 的用法欄位（Description/Tags/權限）回用法卡，
	// 不交給 LLM 猜、也不受配額影響。
	if resp, ok := a.maybeHandleSkillUsage(userText, sessionID, traceID); ok {
		return resp, true
	}
	// 「列出你有的 skill / 你有哪些技能」是 skill scope 的列舉，直接回傳本機已安裝
	// skill 清單；其餘（含 image/video/document 各 scope）一律走一般本機搜尋。
	if isListSkillsRequest(userText) {
		a.pushActionStatus("技能", "列出已安裝技能")
		resp := skill_step.CLIResponse{Text: a.formatInstalledSkills()}
		debugtrace.Record("go.list_skills.direct", traceID, map[string]interface{}{"text": resp.Text})
		return &resp, true
	}
	req, ok := localsearch.ParseUserQuery(userText)
	if !ok {
		return nil, false
	}
	// 護欄：原句若含「用法/做法」這類問法詞，帶進去當輔助搜尋詞（不卡門檻、低加分）。
	req.AuxTerms = localsearch.AuxTermsFromText(userText)
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "搜尋", Target: req.Query, Next: actionchain.StandbyNext}
	if handled, resp := a.maybeAskForToolReadiness(sessionID, decision, userText, traceID); handled {
		return &resp, true
	}
	resp := a.executeLocalSearch(req, sessionID, traceID)
	return &resp, true
}

func parseMemoryExpandRequest(userText string) (string, bool) {
	text := strings.TrimSpace(userText)
	if text == "" {
		return "", false
	}
	for _, prefix := range []string{"展開", "查回", "展開記憶", "展開摘要"} {
		if strings.HasPrefix(text, prefix) {
			target := strings.TrimSpace(strings.TrimPrefix(text, prefix))
			target = strings.TrimSpace(strings.TrimSuffix(target, "的細節"))
			target = strings.TrimSpace(strings.TrimSuffix(target, "細節"))
			target = strings.Trim(target, "：: -")
			return target, target != ""
		}
	}
	return "", false
}

func (a *App) maybeHandlePendingLocalSearchWebFallback(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	text := strings.ToLower(strings.TrimSpace(userText))
	if text == "" {
		return nil, false
	}
	pendingLocalSearchWebFallbackMu.Lock()
	pending, has := pendingLocalSearchWebFallbacks[sessionID]
	if has && time.Now().After(pending.ExpiresAt) {
		delete(pendingLocalSearchWebFallbacks, sessionID)
		has = false
	}
	pendingLocalSearchWebFallbackMu.Unlock()
	if !has {
		return nil, false
	}
	if text == "不要" || text == "不用" || text == "no" || text == "n" {
		pendingLocalSearchWebFallbackMu.Lock()
		delete(pendingLocalSearchWebFallbacks, sessionID)
		pendingLocalSearchWebFallbackMu.Unlock()
		return &skill_step.CLIResponse{Text: "好，我先不改用網路搜尋。"}, true
	}
	if !confirmRe.MatchString(text) {
		return nil, false
	}
	pendingLocalSearchWebFallbackMu.Lock()
	delete(pendingLocalSearchWebFallbacks, sessionID)
	pendingLocalSearchWebFallbackMu.Unlock()
	req := websearch.SearchRequest{Query: pending.Query, Limit: pending.Limit}
	resp := a.executeWebSearch(req, traceID)
	return &resp, true
}

func rememberLocalSearchWebFallback(sessionID string, req localsearch.SearchRequest) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(req.Query) == "" {
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = localsearch.DefaultLimit
	}
	pendingLocalSearchWebFallbackMu.Lock()
	pendingLocalSearchWebFallbacks[sessionID] = pendingLocalSearchWebFallback{
		Query:     strings.TrimSpace(req.Query),
		Limit:     limit,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	pendingLocalSearchWebFallbackMu.Unlock()
}

func (a *App) executeLocalSearch(req localsearch.SearchRequest, sessionID, traceID string) skill_step.CLIResponse {
	a.pushActionStatus("搜尋", req.Query) // status rail：正在搜尋本機資料「…」…
	debugtrace.Record("local_search.enter", traceID, map[string]interface{}{
		"query": req.Query,
		"scope": req.Scope,
		"limit": req.Limit,
	})
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	// Bound local scans so a large managed data folder cannot freeze chat.
	ctx, cancel := context.WithTimeout(baseCtx, 2*time.Second)
	defer cancel()

	service := localsearch.NewService(a.localSearchRoots(), a.localSearchItems(traceID))
	outcome, err := service.SearchWithContext(ctx, req)
	if err != nil {
		if errors.Is(err, localsearch.ErrEmptyQuery) {
			return skill_step.CLIResponse{Text: localsearch.EmptyQueryMessage()}
		}
		debugtrace.Record("local_search.error", traceID, map[string]interface{}{
			"error": err.Error(),
		})
		return skill_step.CLIResponse{Error: err.Error()}
	}
	debugtrace.Record("local_search.results", traceID, map[string]interface{}{
		"count":         len(outcome.Results),
		"incomplete":    outcome.Incomplete,
		"reason":        outcome.Reason,
		"files_scanned": outcome.FilesScanned,
		"bytes_scanned": outcome.BytesScanned,
	})
	if len(outcome.Results) == 0 {
		rememberLocalSearchWebFallback(sessionID, req)
		return skill_step.CLIResponse{Text: fmt.Sprintf("本機資料裡找不到「%s」。要改用網路搜尋嗎？", req.Query)}
	}
	return skill_step.CLIResponse{Text: localsearch.FormatSearchOutcome(req, outcome)}
}

func (a *App) localSearchRoots() []localsearch.Root {
	root := appDataRoot()
	projectRoot := storage.ProjectRoot(root, "default")
	return []localsearch.Root{
		{Path: filepath.Join(projectRoot, "memory"), Source: "memory"},
		{Path: filepath.Join(projectRoot, "runtime"), Source: "trace"},
		{Path: filepath.Join(root, "documents"), Source: "document"},
		{Path: filepath.Join(root, "data", "documents"), Source: "document"},
		{Path: filepath.Join(root, "data", "references", "files"), Source: "document"},
		{Path: filepath.Join(root, "data", "images"), Source: "image"},
		{Path: filepath.Join(root, "data", "references", "images"), Source: "image"},
		{Path: filepath.Join(root, "data", "videos"), Source: "video"}, // 影片獨立資料夾：agent 可發現
		// 注意：data/skills 不再做原始檔案掃描——skill 改由 localSearchItems 以
		// 「名稱＋功能摘要」的形式提供，避免搜尋結果回傳 hash／main.go 等內部檔。
		{Path: filepath.Join(root, "debug"), Source: "trace"},
	}
}

func (a *App) localSearchItems(excludeTraceID string) []localsearch.Item {
	var items []localsearch.Item
	if a.toolsService != nil {
		for _, tool := range a.toolsService.List() {
			items = append(items, localsearch.Item{
				Source: "tool",
				Title:  "工具: " + tool.Title,
				Path:   tool.ID,
				Content: strings.Join([]string{
					tool.ID,
					tool.Title,
					tool.Detail,
					tool.Kind,
					tool.Target,
					strings.Join(tool.ActionTags, " "),
				}, "\n"),
			})
		}
	}
	if a.skillRouter != nil {
		if tags, err := a.skillRouter.ActionTags(); err == nil {
			items = append(items, localsearch.Item{
				Source:  "skill",
				Title:   "技能 action tags",
				Content: strings.Join(tags, " "),
			})
		}
	}
	// 每個已歸檔 skill 都提供一筆「名稱＋功能摘要」的搜尋項目，讓使用者搜尋時
	// 看到的是技能名稱與用途，而不是 skill_id 雜湊或內部檔案內容。
	if a.skillArchive != nil {
		if manifests, err := a.skillArchive.ListArchived(); err == nil {
			for i := range manifests {
				m := manifests[i]
				if m.SkillID == "" {
					continue
				}
				items = append(items, localsearch.Item{
					Source:  "skill",
					Title:   firstNonEmpty(m.DisplayName, m.SkillID),
					Path:    "skill:" + m.SkillID,
					Content: skillSearchContent(m),
				})
			}
		}
	}
	referenceDir := filepath.Join(appDataRoot(), "data", "references", "files")
	if entries, err := os.ReadDir(referenceDir); err == nil {
		type recentReference struct {
			name string
			path string
			info os.FileInfo
		}
		var refs []recentReference
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			path := filepath.Join(referenceDir, entry.Name())
			refs = append(refs, recentReference{name: entry.Name(), path: path, info: info})
		}
		sort.SliceStable(refs, func(i, j int) bool {
			return refs[i].info.ModTime().After(refs[j].info.ModTime())
		})
		for i, ref := range refs {
			if i >= 12 {
				break
			}
			ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(ref.name)), ".")
			items = append(items, localsearch.Item{
				Source: "document",
				Title:  "最近引用文件: " + ref.name,
				Path:   ref.path,
				// Content 只放真正要顯示給使用者的內容。
				Content: strings.Join([]string{
					ref.name,
					"副檔名 " + ext,
					"修改時間 " + ref.info.ModTime().Format(time.RFC3339),
				}, "\n"),
				// 別名／同義詞只用於比對，不會出現在 Snippet。
				Keywords: "引用文件 已載入 檔案 本機資料 最近 最新 剛剛 剛才 拉進來 拉進來的 拖進來 拖進來的 匯入 加入",
			})
		}
	}
	for _, event := range debugtrace.EventsSnapshot() {
		if strings.TrimSpace(excludeTraceID) != "" && event.TraceID == excludeTraceID {
			continue
		}
		payload, _ := json.Marshal(event.Data)
		items = append(items, localsearch.Item{
			Source: "trace",
			Title:  fmt.Sprintf("trace #%d %s", event.ID, event.Node),
			Path:   event.TraceID,
			Content: strings.Join([]string{
				event.Time,
				event.TraceID,
				event.Node,
				string(payload),
			}, "\n"),
		})
	}
	return items
}

func webSearchRoutingPrompt() string {
	return "\n\n[web_search_routing]\n" +
		"網路路由：凡需網路搜尋才能判斷的變動資料，如網路、即時、今天、今日、最新、現在等關鍵字，輸出：網路ㄌ<搜尋關鍵字>ㄌ" + actionchain.StandbyNext + "。\n" +
		"[/web_search_routing]"
}

func buildSearchTermExtractionPrompt(systemPrompt string, userText string, recent []conversation.Sentence) string {
	_ = systemPrompt
	parts := []string{
		"任務=抽搜尋關鍵詞",
		"輸出=只用空格分隔詞; 不要句子/JSON/Markdown",
		"規則=保留使用者提到的動作、物件、App、時間指代、文件、skill、操作、對話; 若提到剛剛/最近/上一個/錄製/回放/重現/點擊/開啟/關閉要保留; 問法詞不是關鍵詞，不要輸出：怎麼用、要怎麼用、怎樣用、如何用、怎麼做、如何做、怎麼操作、如何操作; 可用H補缺主詞或地點; 不回答問題; 不判斷工具",
	}
	if h := formatCompactRoutingHistory(recent, userText, 3); h != "" {
		parts = append(parts, h)
	}
	parts = append(parts, "Q="+compactPromptField(userText))
	return strings.Join(parts, " | ")
}

func parseSearchTerms(text string, userText string) []string {
	raw := strings.TrimSpace(text)
	replacer := strings.NewReplacer(
		"、", " ", "，", " ", ",", " ", "\n", " ", "\t", " ",
		"：", " ", ":", " ", ";", " ", "；", " ",
		"[", " ", "]", " ", "{", " ", "}", " ", "\"", " ",
		"「", " ", "」", " ", "『", " ", "』", " ",
	)
	terms := strings.Fields(replacer.Replace(raw))
	if len(terms) == 0 {
		terms = strings.Fields(compactReferenceQuery(userText))
	}
	return normalizeSearchTerms(terms, 16)
}

func normalizeSearchTerms(values []string, limit int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "。.!！?？,，、:：;；\"'`[]{}()（）")
		if value == "" {
			continue
		}
		lower := strings.ToLower(value)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func shouldPromoteDecisionToWebSearch(decision toolRoutingDecision) bool {
	if decision.Kind == toolRoutingDecisionNeedTool {
		return true
	}
	if decision.Kind != toolRoutingDecisionAction {
		return false
	}
	action := strings.ToLower(strings.TrimSpace(decision.Action))
	switch action {
	case "搜尋", "查找", "本機搜尋", "search", "find":
		return true
	default:
		return false
	}
}

func shouldRouteUserTextToWebSearch(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || isExplicitLocalSearchRequest(trimmed) {
		return false
	}
	lower := strings.ToLower(trimmed)
	if containsAny(lower, []string{
		"web", "internet", "online", "latest", "current", "today", "news", "weather",
		"stock price", "exchange rate", "horoscope", "realtime", "real-time",
	}) {
		return true
	}
	return containsAny(trimmed, []string{
		"網路", "上網", "線上", "即時", "最新", "今天", "今日", "現在", "新聞", "天氣",
		"股價", "匯率", "星座運勢", "運勢", "目前", "最近",
	})
}

func isExplicitLocalSearchRequest(text string) bool {
	lower := strings.ToLower(text)
	if containsAny(lower, []string{"local", "file", "workspace", "project", "trace", "log"}) {
		return true
	}
	return containsAny(text, []string{
		"本機", "檔案", "文件", "專案", "工作區", "記憶", "紀錄", "對話紀錄", "trace", "日誌",
	})
}
