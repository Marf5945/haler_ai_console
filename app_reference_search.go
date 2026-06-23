package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/localsearch"
)

// isReferenceListingQuestion 只認「列出有哪些引用檔」這種清單意圖（要回檔名列表），
// 與「搜尋引用檔內容」明確區分。比 isLoadedReferenceVisibilityQuestion 更窄：必須是
// 問「有哪些／列出／載入了什麼檔」，且不帶內容查詢詞。
func isReferenceListingQuestion(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	mentionsFile := containsAny(trimmed, []string{"檔案", "文件", "引用", "引用文件", "拉進來", "拖進來", "已載入", "匯入", "檔"}) ||
		containsAny(lower, []string{"file", "files", "upload", "uploaded", "reference", "ficheiro", "ficheiros", "referencia", "referência"})
	if !mentionsFile {
		return false
	}
	// 內容查詢詞出現時，優先當成「查內容」，不算清單。
	if containsAny(trimmed, []string{"內容", "裡面", "裡的", "寫了什麼", "寫什麼", "提到", "搜尋", "查", "找", "關於"}) ||
		containsAny(lower, []string{"content", "inside", "about", "search", "mention"}) {
		return false
	}
	// 清單意圖詞：問「有哪些 / 列出 / 看到哪些 / 載入了什麼」。
	listing := containsAny(trimmed, []string{"有哪些", "列出", "哪幾個", "幾個檔", "看到哪些", "有什麼檔", "什麼檔", "清單", "列表", "有看到", "看得到", "有沒有", "看到"}) ||
		containsAny(lower, []string{"list", "which file", "which files", "what file", "what files", "what reference file", "what reference files", "do i have", "have loaded", "how many file", "que ficheiro", "que ficheiros", "quais ficheiros", "ficheiros tenho", "tenho"})
	return listing
}

// isReferenceSearchSentinel 判斷 routing target 是否為「引用文件」sentinel
// （代表「對已載入引用檔做事」，而非字面要搜尋『引用文件』這幾個字）。
func isReferenceSearchSentinel(target string) bool {
	return strings.TrimSpace(target) == "引用文件"
}

// namedReferenceFile 在 userText 明講某個已載入引用檔名時，回傳該檔完整檔名，否則回 ""。
// 比對：userText 是否包含引用檔名（含或不含副檔名）；取最長命中（最精確）。
func namedReferenceFile(userText string, refs []routingReferenceFile) string {
	lowerText := strings.ToLower(userText)
	best := ""
	for _, ref := range refs {
		name := strings.TrimSpace(ref.Name)
		if name == "" {
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if len([]rune(stem)) < 2 { // 太短檔名不做模糊命中，避免誤判
			continue
		}
		if strings.Contains(lowerText, strings.ToLower(name)) || strings.Contains(lowerText, strings.ToLower(stem)) {
			if len([]rune(name)) > len([]rune(best)) {
				best = name
			}
		}
	}
	return best
}

// executeReferenceContentSearch 搜尋「已載入引用檔的實際內容」（而非列檔）。
// 限定 document scope（涵蓋 data/references/files，TXT/MD 內容即在此）。
// query 取用順序：明講檔名 → 檔名 stem（保證命中該檔並精確過濾）；否則 → llmQuery
// （本輪 keyword 抽詞的 LLM 斷詞結果，空白分隔，中文才能逐詞比對）→ 退回
// compactReferenceQuery。仍查不到時退回列檔，不誤回「找不到」。
func (a *App) executeReferenceContentSearch(userText, llmQuery, sessionID, traceID string) skill_step.CLIResponse {
	refs := a.recentReferenceFilesForRouting(12)
	named := namedReferenceFile(userText, refs)

	// 無檔名時優先用 LLM 斷詞 query（localsearch 對中文需逐詞比對），沒有才退回粗正規化。
	query := strings.TrimSpace(llmQuery)
	if query == "" {
		query = strings.TrimSpace(compactReferenceQuery(userText))
	}
	if named != "" {
		// 指定檔：用檔名 stem 當 query，保證以 path/title contains 命中該檔。
		query = strings.TrimSpace(strings.TrimSuffix(named, filepath.Ext(named)))
	}
	if query == "" {
		return a.referenceListingResponse(refs)
	}

	req := localsearch.SearchRequest{
		Query:    query,
		Scope:    []string{"document"},
		Limit:    8,
		AuxTerms: localsearch.AuxTermsFromText(userText),
	}
	a.pushActionStatus("搜尋", req.Query)
	defer a.clearActionStatus() // 完成/錯誤都收回 idle，避免頂部停在「正在搜尋…」
	debugtrace.Record("reference_content_search.enter", traceID, map[string]interface{}{
		"query":      req.Query,
		"named_file": named,
	})

	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, 2*time.Second)
	defer cancel()

	service := localsearch.NewService(a.localSearchRoots(), a.localSearchItems(traceID))
	outcome, err := service.SearchWithContext(ctx, req)
	if err != nil {
		if errors.Is(err, localsearch.ErrEmptyQuery) {
			return skill_step.CLIResponse{Text: localsearch.EmptyQueryMessage()}
		}
		debugtrace.Record("reference_content_search.error", traceID, map[string]interface{}{"error": err.Error()})
		return skill_step.CLIResponse{Error: err.Error()}
	}

	// 排除只帶 metadata 的「最近引用文件: …」合成項，避免內容查詢退化成列檔。
	outcome.Results = filterOutReferenceListingItems(outcome.Results)

	if named != "" {
		// 使用者明講檔名 → 精確過濾到該檔。
		outcome.Results = filterResultsByFileName(outcome.Results, named)
		debugtrace.Record("reference_content_search.named_filter", traceID, map[string]interface{}{
			"named_file": named,
			"hits":       len(outcome.Results),
		})
		if len(outcome.Results) == 0 {
			return skill_step.CLIResponse{Text: localizedReferenceNotFound(named, req.Query, a.responseLanguage())}
		}
		return skill_step.CLIResponse{Text: a.formatLocalSearchOutcome(req, outcome)}
	}

	if len(outcome.Results) == 0 {
		// 未指定檔名又查不到內容 → 退回列檔，不誤回「找不到」。
		return a.referenceListingResponse(refs)
	}
	return skill_step.CLIResponse{Text: a.formatLocalSearchOutcome(req, outcome)}
}

// referenceListingResponse 回覆「已載入引用檔清單」（沒有引用檔時誠實說明）。
func (a *App) referenceListingResponse(refs []routingReferenceFile) skill_step.CLIResponse {
	if len(refs) > 0 {
		return skill_step.CLIResponse{
			Text:   a.formatRecentReferenceFilesAnswer(refs),
			Action: "搜尋",
			Target: "引用文件",
			Next:   "文件",
		}
	}
	return skill_step.CLIResponse{Text: localizedNoReferenceFiles(a.responseLanguage())}
}

// filterOutReferenceListingItems 濾掉 localSearchItems 注入的「最近引用文件: …」合成項，
// 那些只帶檔名／副檔名／時間，不是檔案內容。
func filterOutReferenceListingItems(results []localsearch.SearchResult) []localsearch.SearchResult {
	out := make([]localsearch.SearchResult, 0, len(results))
	for _, r := range results {
		if strings.HasPrefix(strings.TrimSpace(r.Title), "最近引用文件:") {
			continue
		}
		out = append(out, r)
	}
	return out
}

// filterResultsByFileName 精確過濾到指定檔（base 檔名相等，或標題含該檔名）。
func filterResultsByFileName(results []localsearch.SearchResult, fileName string) []localsearch.SearchResult {
	target := strings.ToLower(strings.TrimSpace(fileName))
	if target == "" {
		return results
	}
	out := make([]localsearch.SearchResult, 0, len(results))
	for _, r := range results {
		base := strings.ToLower(filepath.Base(strings.TrimSpace(r.Path)))
		if base == target || strings.Contains(strings.ToLower(r.Title), target) {
			out = append(out, r)
		}
	}
	return out
}
