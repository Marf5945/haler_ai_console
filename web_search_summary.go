package main

import (
	"fmt"
	"strings"

	"ui_console/adapter/debugtrace"
	"ui_console/shared/controlseal"
	"ui_console/shared/websearch"
)

// webSearchSummaryMaxRunes 限制送進模型的單筆 snippet 長度，避免 prompt 過長。
const webSearchSummaryMaxRunes = 500

// summarizeWebSearchOutcome 把搜尋結果交給目前對話的 CLI/API 模型整理成一段
// 連貫摘要，內文以 [n] 標號標示出處，並在文末附上「來源：」清單。
//
// 回傳 (摘要文字, true) 代表成功；任何失敗（無結果、無可用模型、模型呼叫失敗、
// 回應為空）都回傳 ("", false)，由呼叫端回退成原始清單（FormatSearchOutcome）。
//
// SEC：搜尋結果屬外部不可信內容，標題與 snippet 一律經 controlseal.SourceToolOutput
// 消毒後才拼進 prompt，避免搜尋結果夾帶的指令注入。URL 本身不進模型 prompt，
// 只用於文末來源清單（由程式端決定，確保連結正確）。
func (a *App) summarizeWebSearchOutcome(req websearch.SearchRequest, outcome websearch.SearchOutcome, traceID string) (string, bool) {
	if a == nil || len(outcome.Results) == 0 {
		return "", false
	}
	query := strings.TrimSpace(firstNonEmptyText(outcome.Query, req.Query))

	// 組裝給模型的來源區塊（已消毒、已標號）。
	var sources strings.Builder
	for i, r := range outcome.Results {
		title := controlseal.SanitizeForLLM(controlseal.SourceToolOutput, cleanWebText(r.Title)).LLMText
		snippet := controlseal.SanitizeForLLM(controlseal.SourceToolOutput, cleanWebText(r.Snippet)).LLMText
		snippet = truncateWebRunes(snippet, webSearchSummaryMaxRunes)
		fmt.Fprintf(&sources, "[%d] 標題：%s\n", i+1, title)
		if snippet != "" {
			fmt.Fprintf(&sources, "    內容：%s\n", snippet)
		}
	}

	sys := "你是網路搜尋結果整理助手。只根據下方提供的搜尋結果回答，不得自行編造或補充未出現的資訊。" +
		"請用繁體中文寫成一段連貫、好讀的摘要，直接回答使用者的查詢。" +
		"每當引用某筆結果的內容時，在句末加上對應的來源標號，例如 [1] 或 [1][2]。" +
		"只輸出摘要本文，不要輸出來源清單、前言、結語或任何動作格式。"
	prompt := fmt.Sprintf("系統規則：\n%s\n\n使用者查詢：%s\n\n搜尋結果：\n%s\n請依上述規則輸出摘要。", sys, query, sources.String())

	// source 留空 → callRawModel 內部走 defaultSkillExecutionAdapterID 取目前可用 adapter。
	summary, err := a.callRawModel("", "web-search-summary", prompt, traceID)
	if err != nil {
		debugtrace.Record("web_search.summary.error", traceID, map[string]interface{}{
			"error": err.Error(),
		})
		return "", false
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		debugtrace.Record("web_search.summary.empty", traceID, nil)
		return "", false
	}

	var b strings.Builder
	b.WriteString(summary)
	b.WriteString("\n\n來源：")
	for i, r := range outcome.Results {
		title := cleanWebText(r.Title)
		if title == "" {
			title = "(無標題)"
		}
		fmt.Fprintf(&b, "\n[%d] %s", i+1, title)
		if url := strings.TrimSpace(r.URL); url != "" {
			fmt.Fprintf(&b, " - %s", url)
		}
	}
	debugtrace.Record("web_search.summary.ok", traceID, map[string]interface{}{
		"sources": len(outcome.Results),
	})
	return b.String(), true
}

// firstNonEmptyText 回傳第一個去空白後非空的字串。
func firstNonEmptyText(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// cleanWebText 收斂空白並去頭尾，給摘要與來源清單共用。
func cleanWebText(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// truncateWebRunes 以 rune 為單位截斷，避免切斷多位元組字元。
func truncateWebRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
