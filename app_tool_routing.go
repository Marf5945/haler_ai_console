// app_tool_routing.go - split out of app.go (same package, codemod).
package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/data/conversation"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/localsearch"
	"ui_console/shared/package_import"
	"ui_console/shared/preference"
	"ui_console/shared/tools"
	"ui_console/shared/websearch"
)

func (a *App) ListTools() []tools.Tool {
	a.syncArchivedSkillsToToolbar()
	return a.toolsService.List()
}

func (a *App) ActivateTool(id string) tools.ActionResult {
	return a.toolsService.Activate(id)
}

// registerPackageTool 在套件確認安裝後，把 mcp / skill 類型登記到工具列。
// adapter / persona 走各自既有流程，不在此登記。
func (a *App) registerPackageTool(pi *package_import.PendingImport) {
	if a == nil || a.toolsService == nil || pi == nil {
		return
	}
	name := strings.TrimSpace(pi.Manifest.Name)
	if name == "" {
		return
	}
	kind := strings.TrimSpace(pi.Manifest.PackageType)
	if pi.Manifest.AddsMCPServer {
		kind = "mcp"
	}
	if kind != "mcp" && kind != "skill" {
		return
	}
	icon := "\u2726" // ✦
	detail := "已安裝 skill"
	if kind == "mcp" {
		icon = "\u2699" // ⚙
		detail = "已安裝 MCP server"
	}
	a.toolsService.AddTool(tools.Tool{
		ID:        kind + ":" + name,
		Icon:      icon,
		Title:     name,
		Detail:    detail,
		Kind:      kind,
		Target:    name,
		Enabled:   true,
		Available: true,
	})
}

// GetToolRegistryPatchProposals returns tool registry patch proposals generated
// by hook evidence analysis.
func (a *App) GetToolRegistryPatchProposals() (interface{}, error) {
	proposals, err := a.reinforcementService.GetRegistryProposals()
	return frontendDTO(proposals), err
}

// MarkToolUnavailable 標記工具為斷線狀態。
func (a *App) MarkToolUnavailable(toolID, reason string) {
	a.toolsService.MarkUnavailable(toolID, reason)
}

// MarkToolAvailable 標記工具恢復連線。
func (a *App) MarkToolAvailable(toolID string) {
	a.toolsService.MarkAvailable(toolID)
}

func (a *App) lookupToolRoutingContext(terms []string, userText string, traceID string) toolRoutingLookupContext {
	query := strings.Join(normalizeSearchTerms(terms, 16), " ")
	if strings.TrimSpace(query) == "" {
		query = compactReferenceQuery(userText)
	}
	ctx := toolRoutingLookupContext{
		Query: query,
		Terms: normalizeSearchTerms(append([]string{}, terms...), 16),
	}
	if a != nil {
		ctx.RecentReferences = a.recentReferenceFilesForRouting(6)
	}
	if a != nil && a.learningService != nil && strings.TrimSpace(query) != "" {
		if operations, err := a.learningService.SearchOperations(query, 5); err == nil {
			ctx.Operations = operations
		} else {
			debugtrace.Record("go.toolRouting.lookup.operations_error", traceID, map[string]interface{}{"error": err.Error()})
		}
		if len(ctx.Operations) == 0 {
			if recent, err := a.learningService.ListReplayCatalog(5); err == nil {
				ctx.RecentOperations = recent
			} else {
				debugtrace.Record("go.toolRouting.lookup.recent_operations_error", traceID, map[string]interface{}{"error": err.Error()})
			}
		}
	}
	if a != nil && strings.TrimSpace(query) != "" {
		searchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		service := localsearch.NewService(a.localSearchRoots(), a.localSearchItems(""))
		outcome, err := service.SearchWithContext(searchCtx, localsearch.SearchRequest{
			Query:    query,
			Scope:    []string{"all"},
			Limit:    6,
			AuxTerms: localsearch.AuxTermsFromText(userText),
		})
		if err == nil {
			ctx.LocalMatches = outcome.Results
		} else if !errors.Is(err, localsearch.ErrEmptyQuery) {
			debugtrace.Record("go.toolRouting.lookup.local_error", traceID, map[string]interface{}{"error": err.Error()})
		}
	}
	debugtrace.Record("go.toolRouting.lookup", traceID, map[string]interface{}{
		"query":          ctx.Query,
		"terms":          ctx.Terms,
		"operation_hits": len(ctx.Operations),
		"recent_ops":     len(ctx.RecentOperations),
		"local_hits":     len(ctx.LocalMatches),
		"loaded_files":   compactReferenceFilesForTrace(ctx.RecentReferences),
	})
	return ctx
}

func formatToolRoutingLookupContext(ctx toolRoutingLookupContext) string {
	var b strings.Builder
	b.WriteString("\n[lookup] ")
	fmt.Fprintf(&b, "query=%q\n", ctx.Query)
	if len(ctx.Terms) > 0 {
		fmt.Fprintf(&b, "terms=%q\n", strings.Join(ctx.Terms, " "))
	}
	b.WriteString("loaded_files=")
	if len(ctx.RecentReferences) == 0 {
		b.WriteString("none\n")
	} else {
		for i, ref := range ctx.RecentReferences {
			if i >= 6 {
				break
			}
			if i > 0 {
				b.WriteString("; ")
			}
			fmt.Fprintf(&b, "name=%q ext=%q mtime=%q", ref.Name, ref.Ext, ref.ModifiedAt.Format(time.RFC3339))
		}
		b.WriteString("\n")
	}
	b.WriteString("saved_operations=")
	if len(ctx.Operations) == 0 {
		b.WriteString("none\n")
	} else {
		for i, op := range ctx.Operations {
			if i >= 5 {
				break
			}
			if i > 0 {
				b.WriteString("; ")
			}
			risk := learningRiskLevel(op.Risk)
			fmt.Fprintf(&b, "tag=%q title=%q action=%q risk=%q score=%.2f", op.Tag, op.Title, op.OperationTag, risk, op.Score)
		}
		b.WriteString("\n")
	}
	b.WriteString("recent_operations_when_no_match=")
	if len(ctx.RecentOperations) == 0 {
		b.WriteString("none\n")
	} else {
		for i, op := range ctx.RecentOperations {
			if i >= 3 {
				break
			}
			if i > 0 {
				b.WriteString("; ")
			}
			risk := learningRiskLevel(op.Risk)
			fmt.Fprintf(&b, "tag=%q title=%q action=%q risk=%q steps=%d time=%q",
				op.Tag, op.Title, op.OperationTag, risk, op.StepCount, op.StoppedAt.Format(time.RFC3339))
		}
		b.WriteString("\n")
	}
	b.WriteString("local_matches=")
	if len(ctx.LocalMatches) == 0 {
		b.WriteString("none\n")
	} else {
		for i, item := range ctx.LocalMatches {
			if i >= 6 {
				break
			}
			if i > 0 {
				b.WriteString("; ")
			}
			fmt.Fprintf(&b, "source=%q title=%q file=%q score=%d",
				item.Source, item.Title, filepath.Base(item.Path), item.Score)
		}
		b.WriteString("\n")
	}
	b.WriteString("[/lookup]\n")
	return b.String()
}

func buildToolRoutingDecisionPrompt(systemPrompt string, userText string, lookupContext string, recentOpt ...[]conversation.Sentence) string {
	_ = systemPrompt
	var recent []conversation.Sentence
	if len(recentOpt) > 0 {
		recent = recentOpt[0]
	}
	routingRules := strings.Join([]string{
		fmt.Sprintf("本app預設本機搜尋；只有即時/今天/最新/現在/網路等變動資料才輸出 網路%s<搜尋關鍵字>%s%s。local_matches none 且使用者像在找本機資料時，輸出 %s%s<query>%s%s。", actionchain.Separator, actionchain.Separator, actionchain.StandbyNext, "\u641c\u5c0b", actionchain.Separator, actionchain.Separator, "\u6587\u4ef6"),
		"若 loaded_files 不是 none，使用者問剛剛/最近/拉進來/拖進來/已載入/引用的檔案時，不要問檔名或路徑；輸出 搜尋ㄌ引用文件ㄌ文件。",
		"勿呼叫任何工具；「工具」僅是分類標籤。",
		"格式只能是：閒聊ㄌ<回答> | 操作ㄌ<候選tag/名稱/關鍵詞>ㄌ待命 | 程式ㄌ<程式名稱>ㄌ輸出 | 流程ㄌ<skill名稱>ㄌ輸出 | 查詢ㄌ<關鍵詞>ㄌ操作 | 搜尋ㄌ<關鍵詞>ㄌ文件 | 網路ㄌ<搜尋關鍵字>ㄌ待命 | 提問ㄌ<問題>ㄌ待命 | 需要工具",
		"需要工具：需要其他工具，或候選不足但不像閒聊。",
		"網路路由：凡需網路搜尋才能判斷的變動資料，如網路、即時、今天、今日、最新、現在等關鍵字→網路。",
		"操作：明確重現/回放/照做/執行已保存操作且 saved_operations 明確→操作；只有 recent_operations 不算明確。",
		"判斷=製作獨立程式(產出 .go 等程式檔)→程式; 使用既有/已安裝 skill，或要既有 skill 處理資料/表格/CSV/XLSX/JSON並輸出→流程; 找操作候選→查詢; 找本機資料/文件/skill/記憶/對話/trace/專案→搜尋; 無法判斷本機或網路且缺必要資訊→提問; 明顯聊天→閒聊",
	}, " ")
	parts := []string{strings.TrimSpace(lookupContext), "rules=" + routingRules}
	if h := formatCompactRoutingHistory(recent, userText, 3); h != "" {
		parts = append(parts, h)
	}
	parts = append(parts, "Q="+compactPromptField(userText))
	return strings.Join(parts, " | ")
}

func parseToolRoutingDecision(text string) toolRoutingDecision {
	raw := strings.TrimSpace(text)
	decision := toolRoutingDecision{Kind: toolRoutingDecisionNeedTool, Raw: raw}
	if raw == "" {
		return decision
	}
	firstLine := strings.TrimSpace(strings.Split(raw, "\n")[0])
	if isNeedToolResponse(firstLine) {
		return decision
	}
	chain, err := actionchain.Parse(firstLine)
	if err == nil {
		if chain.Action == "聊天" {
			decision.Kind = toolRoutingDecisionChat
			decision.Text = chain.Target
			return decision
		}
		decision.Kind = toolRoutingDecisionAction
		decision.Action = chain.Action
		decision.Target = chain.Target
		decision.Next = chain.Next
		return decision
	}
	return decision
}

func normalizeToolRoutingDecision(decision toolRoutingDecision, userText string, lookup toolRoutingLookupContext) toolRoutingDecision {
	if isLoadedReferenceVisibilityQuestion(userText) && len(lookup.RecentReferences) > 0 {
		decision.Kind = toolRoutingDecisionAction
		decision.Action = "搜尋"
		decision.Target = "引用文件"
		decision.Next = "文件"
		return decision
	}
	if shouldRouteUserTextToWebSearch(userText) && shouldPromoteDecisionToWebSearch(decision) {
		target := firstNonEmpty(decision.Target, lookup.Query, compactReferenceQuery(userText), userText)
		if strings.TrimSpace(target) != "" {
			decision.Kind = toolRoutingDecisionAction
			decision.Action = "網路"
			decision.Target = strings.TrimSpace(target)
			decision.Next = actionchain.StandbyNext
			return decision
		}
	}
	if decision.Kind != toolRoutingDecisionAction {
		return decision
	}
	action := strings.TrimSpace(decision.Action)
	next := strings.TrimSpace(decision.Next)
	if (action != "查詢" && action != "搜尋" && action != "query" && action != "search") || (next != "操作" && next != "操做") {
		return decision
	}
	if !isLearningOperationExecutionRequest(userText) || len(lookup.Operations) == 0 {
		return decision
	}
	best := lookup.Operations[0]
	target := firstNonEmpty(best.Tag, best.Title, best.OperationTag, decision.Target, lookup.Query)
	decision.Action = "操作"
	decision.Target = target
	decision.Next = actionchain.StandbyNext
	return decision
}

func shouldRepairToolRoutingDecision(userText string, decision toolRoutingDecision) bool {
	if _, ok := inferGoProgramAuthoringRequest(userText); !ok {
		return false
	}
	if decision.Kind == toolRoutingDecisionAction && strings.TrimSpace(decision.Action) == "程式" {
		return false
	}
	if decision.Kind == toolRoutingDecisionChat {
		return true
	}
	if decision.Kind == toolRoutingDecisionNeedTool {
		return true
	}
	return false
}

func buildToolRoutingRepairPrompt(basePrompt, previousOutput, userText string) string {
	programName, _ := inferGoProgramAuthoringRequest(userText)
	if programName == "" {
		programName = "資料處理程式"
	}
	var b strings.Builder
	b.WriteString(basePrompt)
	b.WriteString("\n\n[系統修正]\n")
	b.WriteString("上一輪輸出不符合本 app 的工具路由語意，請重新輸出，仍然只能輸出允許格式之一。\n")
	b.WriteString("使用者是在要求建立/製作一個資料處理 skill；本 app 內建受控 Go 小程式製作器，不使用 Gemini CLI skill.yaml，不要說無法寫檔，不要嘗試 activate_skill/write_file/invoke_agent，不要產 Python。\n")
	b.WriteString("若判斷需要製作小程式，請輸出：程式ㄌ")
	b.WriteString(programName)
	b.WriteString("ㄌ輸出\n")
	b.WriteString("上一輪輸出:\n")
	b.WriteString(previousOutput)
	b.WriteString("\n[/系統修正]")
	return b.String()
}

func (a *App) responseFromToolRoutingDecision(decision toolRoutingDecision, sessionID, traceID string, userTextOpt ...string) (bool, skill_step.CLIResponse) {
	userText := strings.TrimSpace(decision.Raw)
	if len(userTextOpt) > 0 && strings.TrimSpace(userTextOpt[0]) != "" {
		userText = strings.TrimSpace(userTextOpt[0])
	}
	if isLoadedReferenceVisibilityQuestion(userText) {
		if refs := a.recentReferenceFilesForRouting(6); len(refs) > 0 {
			return true, skill_step.CLIResponse{
				Text:   formatRecentReferenceFilesAnswer(refs),
				Action: "搜尋",
				Target: "引用文件",
				Next:   "文件",
			}
		}
	}
	switch decision.Kind {
	case toolRoutingDecisionChat:
		return true, skill_step.CLIResponse{Text: strings.TrimSpace(decision.Text)}
	case toolRoutingDecisionAction:
		if strings.TrimSpace(decision.Target) == "" {
			return false, skill_step.CLIResponse{}
		}
		// judge 主動提問（模糊：本機還是網路 / 缺必要資訊）→ 直接把問題回給使用者。
		if strings.TrimSpace(decision.Action) == "提問" {
			return true, skill_step.CLIResponse{
				Text:   setQuestionFloatingCandidates(questionPayload(decision.Target, decision.Next), traceID),
				Action: decision.Action,
				Target: decision.Target,
				Next:   decision.Next,
			}
		}
		if resp, handled := a.maybeHandleResourceGate(strings.TrimSpace(decision.Action+" "+decision.Target), sessionID, traceID); handled {
			return true, *resp
		}
		if handled, resp := a.maybeAskForToolReadiness(sessionID, decision, userText, traceID); handled {
			return true, resp
		}
		if handled, flowResp := a.maybeHandleSkillFlow(decision, sessionID, traceID, userText); handled {
			return true, flowResp
		}
		if handled, resp := a.maybeHandleGoProgramAuthoring(decision, sessionID, traceID, userText); handled {
			return true, resp
		}
		next := strings.TrimSpace(decision.Next)
		if next == "操作" || next == "操做" || decision.Action == "操作" || decision.Action == "操做" {
			return true, skill_step.CLIResponse{
				Text:   decision.Target,
				Action: decision.Action,
				Target: decision.Target,
				Next:   decision.Next,
			}
		}
		target := decision.Target
		if decision.Action == "網路" {
			target = a.targetWithBackground(sessionID, target)
		}
		if req, ok := websearch.RequestFromAction(decision.Action, target); ok {
			webResp := a.executeWebSearch(req, traceID)
			webResp.Action = decision.Action
			webResp.Target = target
			webResp.Next = decision.Next
			return true, webResp
		}
		if req, ok := localsearch.RequestFromAction(decision.Action, decision.Target); ok {
			localResp := a.executeLocalSearch(req, sessionID, traceID)
			localResp.Action = decision.Action
			localResp.Target = decision.Target
			localResp.Next = decision.Next
			return true, localResp
		}
		// 展開：撈回 deep_memory 細節（v3.1.7）。
		if handled, memResp := a.maybeExpandMemory(decision.Action, decision.Target, traceID); handled {
			memResp.Next = decision.Next
			return true, memResp
		}
	}
	return false, skill_step.CLIResponse{}
}

func buildToolRoutingJudgePrompt(systemPrompt string, userText string) string {
	_ = systemPrompt
	return strings.Join([]string{
		"任務=工具粗判",
		"輸出只能是 需要工具 或 閒聊ㄌ<回答>",
		"需要搜尋本機文件/讀資料/開啟/寫入/匯出/排程/製作或使用小程式/已保存螢幕操作/DAG/自動流程/重現/回放/操作/點擊時，輸出需要工具",
		"不要猜工具名稱; 不要輸出無前綴自然語言",
		"Q=" + compactPromptField(userText),
	}, " | ")
}

func isNeedToolResponse(text string) bool {
	firstLine := strings.TrimSpace(strings.Split(strings.TrimSpace(text), "\n")[0])
	firstLine = strings.Trim(firstLine, "。.!！ \t\r\n")
	return firstLine == "需要工具"
}

// GetToolVisibility returns the rendered visibility state (rank only) for all installed tools.
// 工具可用性（Available / NeedsReauth）由 tools / adapter_registry 套件管理。
func (a *App) GetToolVisibility() interface{} {
	toolList := a.toolsService.List()
	toolIDs := make([]string, len(toolList))
	for i, t := range toolList {
		toolIDs[i] = t.ID
	}
	return frontendDTO(a.prefStore.BuildVisibilityList(toolIDs, ""))
}

// SetToolPreference records a user's explicit drag-position preference for a tool.
// scope: "sub" | "main" | "global" | "routing". dagRevision required when scope="main".
func (a *App) SetToolPreference(toolID, scope, dagRevision string, rank int) error {
	return a.prefStore.SetPreference(preference.ToolPreferenceEntry{
		ToolID:      toolID,
		Rank:        preference.PreferenceRank(rank),
		Scope:       preference.PreferenceScope(scope),
		DAGRevision: dagRevision,
	})
}
