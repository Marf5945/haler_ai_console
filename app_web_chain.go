// app_web_chain.go — 複合意圖 Web Chain（2.5.5.11）。
// judge 對複合網路問題輸出多行 chain plan：第一行立即執行，其餘行是期望鏈
// （cache/預備訊號 + 事後比對基準）。每步結果經 SanitizeForLLM 回灌，模型
// 當輪產出實際下一行；實際與期望不符記 drift，現實優先。
// 三段式與封閉 next 詞彙不變：續跑只認 next=網路，其餘一律視同待命終止。
package main

import (
	"fmt"
	"os"
	"strings"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_eval"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/controlseal"
	"ui_console/shared/websearch"
)

const (
	webChainMaxFollowupRounds   = 2    // 第一步之外最多續跑輪數
	webChainObservationMaxRunes = 4000 // 單輪回灌觀察上限（rune）
)

// webChainEnabled 預設開；AI_CONSOLE_WEB_CHAIN=0/false/off 可整體關閉。
func webChainEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AI_CONSOLE_WEB_CHAIN"))) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// webChainShouldContinue 只有「網路ㄌ…ㄌ網路」才續跑。
func webChainShouldContinue(action, next string) bool {
	if actionchain.NormalizeAction(action) != "網路" {
		return false
	}
	return actionchain.NormalizeNext(next) == "網路"
}

// webChainExpectedFromRaw 從 judge 原始多行輸出取期望鏈（含第一步）；單行回 nil。
func webChainExpectedFromRaw(raw string) []actionchain.ActionChain {
	if !strings.Contains(strings.TrimSpace(raw), "\n") {
		return nil
	}
	res := actionchain.ParseChainLines(raw)
	if len(res.Steps) < 2 {
		return nil
	}
	return res.Steps
}

// webChainDriftReason 比對第 round 輪（1-based，對 expected[round]）的實際 action。
// target 多為占位（<依結果>），依 OP 步慣例不比對。回空字串=無 drift。
func webChainDriftReason(expected []actionchain.ActionChain, round int, actualAction string) string {
	if len(expected) == 0 {
		return ""
	}
	if round >= len(expected) {
		return fmt.Sprintf("期望鏈僅 %d 步，實際出現第 %d 步", len(expected), round+1)
	}
	want := actionchain.NormalizeAction(expected[round].Action)
	got := actionchain.NormalizeAction(actualAction)
	if want != got {
		return fmt.Sprintf("action 不符：期望 %s 實際 %s", want, got)
	}
	return ""
}

// webChainTruncateRunes 截斷觀察文字，防整篇網頁塞爆回灌 prompt。
func webChainTruncateRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…(截斷)"
}

// buildWebChainFollowupPrompt 組回灌 prompt：原始問題 + 清洗後觀察 + 封閉格式。
func buildWebChainFollowupPrompt(userText, prevTarget, sanitizedObservation string) string {
	return strings.Join([]string{
		"任務=複合問題下一步",
		"上一步已執行 網路" + actionchain.Separator + compactPromptField(prevTarget),
		"[觀察]" + sanitizedObservation + "[/觀察]",
		"格式只能是：網路ㄌ<下一步搜尋關鍵字>ㄌ<待命|網路> | 閒聊ㄌ<回答>",
		"規則：依觀察組出下一步搜尋關鍵字；觀察已足夠回答就輸出 閒聊ㄌ<回答>；不可新增未確認的地點、時間、數字或來源。",
		"Q=" + compactPromptField(userText),
	}, " | ")
}

// webChainStepCall 把「丟 prompt 拿文字」抽象化，CLI 與 API 路徑共用同一條迴圈。
type webChainStepCall func(prompt string) (string, error)

// webChainCLIStepCall 包 cliAdapter（CLI 路徑用）。
func (a *App) webChainCLIStepCall(adapterID, cliPath, sessionID, modelOverride, traceID string) webChainStepCall {
	round := 0
	return func(prompt string) (string, error) {
		round++
		resp, err := a.cliAdapter.SendMessage(skill_step.CLIMessageOptions{
			AdapterID:      adapterID,
			CLIPath:        cliPath,
			SessionID:      sessionID,
			UserText:       prompt,
			Model:          strings.TrimSpace(modelOverride),
			ContinuityKey:  conversationContinuityKey("web-chain", sessionID),
			TraceID:        fmt.Sprintf("%s-wc%d", traceID, round),
			SkipContinuity: true,
		})
		if err != nil {
			return "", err
		}
		if resp.Error != "" {
			return "", fmt.Errorf("%s", resp.Error)
		}
		if resp.AuthRequired {
			return "", fmt.Errorf("auth required")
		}
		return resp.Text, nil
	}
}

// maybeContinueWebChain 在主 judge 路徑的 web search 回應後續跑複合鏈。
// 終止：next=待命、非網路動作、解析失敗、輪數/觀察雙上限、模型錯誤。
// 各步搜尋結果原文累加呈現（忠實呈現，不再製）。judgeRaw 取 decision.Raw（含多行期望鏈）。
func (a *App) maybeContinueWebChain(stepCall webChainStepCall, sessionID, traceID, userText, judgeRaw string, first skill_step.CLIResponse) skill_step.CLIResponse {
	if !webChainEnabled() || !webChainShouldContinue(first.Action, first.Next) {
		return first
	}
	expected := webChainExpectedFromRaw(judgeRaw)
	if len(expected) > 0 {
		// 期望鏈即 cache 訊號：先記 trace 供預備與事後比對（Tavily 無狀態，暫無可熱連線）。
		planLines := make([]string, 0, len(expected))
		for _, s := range expected {
			planLines = append(planLines, s.Action+actionchain.Separator+s.Target+actionchain.Separator+s.Next)
		}
		debugtrace.Record("go.webChain.plan", traceID, map[string]interface{}{
			"steps": len(expected),
			"plan":  strings.Join(planLines, "\n"),
		})
	}

	var combined strings.Builder
	combined.WriteString(first.Text)
	resp := first

	// Phase 2：累積各輪 drift。Phase 3：記錄執行簽章（第一步已是網路）。
	// 單一 defer 收尾：drift 落 events.jsonl，run 落 web_chain_runs.jsonl（升格分母 + 月報）。
	var drifts []skill_eval.DriftEvent
	executedActions := []string{actionchain.NormalizeAction(first.Action)}
	defer func() {
		recordWebChainDrifts(sessionID, drifts)
		recordWebChainRun(sessionID, executedActions, len(drifts))
	}()

	for round := 1; round <= webChainMaxFollowupRounds; round++ {
		sanitized := controlseal.SanitizeForLLM(controlseal.SourceToolOutput, resp.Text).LLMText
		sanitized = webChainTruncateRunes(sanitized, webChainObservationMaxRunes)
		stepText, err := stepCall(buildWebChainFollowupPrompt(userText, resp.Target, sanitized))
		if err != nil {
			debugtrace.Record("go.webChain.step_error", traceID, map[string]interface{}{
				"round": round, "err": err.Error(),
			})
			break
		}
		parsed := actionchain.ParseSingleLine(stepText)
		if parsed.Err != nil || parsed.NoProposal {
			debugtrace.Record("go.webChain.step_unparsed", traceID, map[string]interface{}{
				"round": round, "text": stepText,
			})
			break
		}
		action := actionchain.NormalizeAction(parsed.Chain.Action)

		// 期望 vs 實際：不符記 drift，現實優先續走。trace 即時可見，store 收尾沉澱。
		if reason := webChainDriftReason(expected, round, action); reason != "" {
			debugtrace.Record("go.webChain.drift", traceID, map[string]interface{}{
				"round": round, "reason": reason, "actual": action, "target": parsed.Chain.Target,
			})
			var expAction, expTarget, expNext string
			if round < len(expected) {
				expAction, expTarget, expNext = expected[round].Action, expected[round].Target, expected[round].Next
			}
			drifts = append(drifts, buildWebChainDriftEvent(
				expAction, expTarget, expNext,
				action, parsed.Chain.Target, parsed.Chain.Next, reason))
		}

		// 模型判斷觀察已足夠 → 收尾回答（聊天=閒聊 normalize 後）。
		if action == "聊天" {
			combined.WriteString("\n\n")
			combined.WriteString(strings.TrimSpace(parsed.Chain.Target))
			resp.Text = combined.String()
			resp.Next = actionchain.StandbyNext
			debugtrace.Record("go.webChain.finish", traceID, map[string]interface{}{"rounds": round, "end": "answer"})
			return resp
		}
		// 封閉詞彙：非網路動作不執行，視同待命終止（防 namespace 污染）。
		if action != "網路" {
			debugtrace.Record("go.webChain.finish", traceID, map[string]interface{}{"rounds": round, "end": "unknown_action:" + action})
			break
		}

		target := a.targetWithBackground(sessionID, strings.TrimSpace(parsed.Chain.Target))
		req, ok := websearch.RequestFromAction("網路", target)
		if !ok {
			break
		}
		webResp := a.executeWebSearch(req, fmt.Sprintf("%s-wc%d", traceID, round))
		combined.WriteString(fmt.Sprintf("\n\n— 第%d步：網路%s%s —\n", round+1, actionchain.Separator, strings.TrimSpace(parsed.Chain.Target)))
		combined.WriteString(webResp.Text)
		executedActions = append(executedActions, "網路") // 計入執行簽章
		resp = webResp
		resp.Action = "網路"
		resp.Target = target
		resp.Next = actionchain.NormalizeNext(parsed.Chain.Next)
		if actionchain.IsStandbyNext(parsed.Chain.Next) {
			debugtrace.Record("go.webChain.finish", traceID, map[string]interface{}{"rounds": round, "end": "standby"})
			break
		}
	}

	resp.Text = combined.String()
	resp.Next = actionchain.StandbyNext
	return resp
}
