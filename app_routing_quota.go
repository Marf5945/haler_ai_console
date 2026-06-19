package main

import (
	"strings"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_step"
)

// normalizeFastPathText 將短句正規化：去頭尾空白與標點、轉小寫，方便做精確比對。
func normalizeFastPathText(text string) string {
	t := strings.TrimSpace(text)
	t = strings.Trim(t, " \t\r\n。.！!？?，,、~～()（）「」『』\"'`")
	return strings.ToLower(strings.TrimSpace(t))
}

// offlineChatReply 是「窄白名單」：只對極短、語意明確、且有安全定型回覆的寒暄／確認句
// 直接回覆，完全不呼叫任何模型。用於：(1) routing 前的 fast path 早退；(2) 模型配額用盡
// 時的保底回覆。刻意只放可安全定型回答的句子；需要上下文才能答的（如「第幾次」）不放進來，
// 維持走正常路由，避免亂答。
func offlineChatReply(userText string) (string, bool) {
	return localizedOfflineChatReply(userText, responseLanguageZH)
}

// isRoutingQuotaHit 判斷一次 routing 階段（keyword/judge/repair）的模型呼叫是否命中
// 配額／限流。回應文字、回應 error 欄位、Go error 任一命中都算。
func isRoutingQuotaHit(respText, respError string, err error) bool {
	if err != nil && isQuotaExhaustedError(err.Error()) {
		return true
	}
	return isQuotaExhaustedError(respError) || isQuotaExhaustedError(respText)
}

// routeAfterRoutingQuotaHit 在 routing 階段模型呼叫命中配額／限流時，建立一條不再呼叫
// 該（已耗盡）模型的確定性路由，避免空等 CLI backoff 或對耗盡模型再打一次。
// 終點順序：窄白名單定型回覆 → 本機候選保底搜尋 → 提示切換模型。
// lookup 可為 nil（keyword 階段就命中、judge 尚未跑、還沒建 lookup），此時用本機抽詞補建，
// 全程不呼叫任何模型。
func (a *App) routeAfterRoutingQuotaHit(adapterID, sessionID, userText, traceID string, lookup *toolRoutingLookupContext) (*skill_step.CLIResponse, bool) {
	// 1. 窄白名單定型回覆：零模型呼叫，配額耗盡也能答。
	if reply, ok := a.offlineChatReply(userText); ok {
		debugtrace.Record("go.toolRouting.quota_fast_fail.offline_chat", traceID, map[string]interface{}{
			"adapter_id": adapterID,
		})
		return &skill_step.CLIResponse{Text: reply}, true
	}
	// 2. 本機候選保底：keyword 階段命中時 lookup 為 nil，用本機抽詞補建（不打模型）。
	lk := lookup
	if lk == nil {
		terms := parseSearchTerms("", userText)
		built := a.lookupToolRoutingContext(terms, userText, traceID)
		lk = &built
	}
	if fb, ok := fallbackDecisionFromLookup(*lk); ok {
		debugtrace.Record("go.toolRouting.quota_fast_fail.local_fallback", traceID, map[string]interface{}{
			"adapter_id": adapterID,
			"action":     fb.Action,
			"target":     fb.Target,
			"local_hits": len(lk.LocalMatches),
		})
		if handled, resp := a.responseFromToolRoutingDecision(fb, sessionID, traceID, nil, userText); handled {
			return &resp, true
		}
	}
	// 3. 無法本機保底：回提示切換模型，不亂答。
	debugtrace.Record("go.toolRouting.quota_fast_fail.switch_model", traceID, map[string]interface{}{
		"adapter_id": adapterID,
	})
	return &skill_step.CLIResponse{Text: quotaSwitchModelMessage(adapterID)}, true
}
