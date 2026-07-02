package main

// app_dag_clarify.go
//
// DAG 模糊地帶 × 反問狀態機接線。
//
// 前端閘門（shouldCreateDagRun）判不進任務、又不像純閒聊時，不再自行猜測，
// 改走既有澄清狀態機（tool_readiness.go 的 storeClarification：count / slot /
// TTL 煞車全數共用）。Action=提問 由系統組出，模型不參與。
//
// 收口兩條：
//  1. 浮動候選「執行任務」draft 帶 /dag 前綴 → 前端顯式通道直接進 StartTaskProgress
//     （StartTaskProgress 端會 clearClarification 收掉 pending）。
//  2. 使用者手打肯定句 → SendCLIMessage / SendAPIMessage 在 consumePendingToolAnswer
//     之前先過 consumeDagIntentAffirmation，命中即回 Action=任務 + Target=原句，
//     由前端映射回任務入口。未命中則不動 pending，交給一般 re-judge 併答流程。

import (
	"regexp"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
)

// dagIntentClarifyAction 標記 pending 種類，供補答攔截辨識。
const dagIntentClarifyAction = "任務確認"

// 肯定／否定判定：否定與問句訊號優先（「好像不用」「要怎麼做？」都不算肯定）。
var (
	dagIntentNegativePattern = regexp.MustCompile(`(不用|不要|不必|先不|算了|免了|只是問|問問|先討論|討論就好|聊聊|再說|暫時不|怎麼|如何|為什麼|嗎|？|\?)`)
	dagIntentAffirmPattern   = regexp.MustCompile(`^(對|好|是|嗯|要|行|可以|沒錯|執行|做吧|去做|直接做|開始|進行|來吧|ok|OK|Ok)`)
)

// dagIntentQuestion 組出帶浮動候選的反問。候選 draft 走前端既有通道：
// 「執行任務」送 /dag 原句（顯式進任務）；「先討論」送答句，由澄清併答回一般對話。
func dagIntentQuestion(userText string) string {
	// '#' 是候選分隔符、換行會破壞 draft，先收斂。
	text := strings.Join(strings.Fields(userText), " ")
	text = strings.ReplaceAll(text, "#", "＃")
	return "任務確認：這句話是要我直接執行成任務，還是先討論就好？" +
		"#執行任務=/dag " + text +
		"#先討論=先討論就好"
}

// ClarifyDagIntent 由前端在 DAG 模糊地帶呼叫（Wails binding）。
// 回 Action=提問：本輪只反問，前端顯示問題並刷新浮動候選。
// 回空 Action：煞車已擋（同 slot 問過 / 達上限），前端應照一般對話送出。
func (a *App) ClarifyDagIntent(sessionID, userText, traceID string) skill_step.CLIResponse {
	text := strings.TrimSpace(userText)
	if a == nil || text == "" {
		return skill_step.CLIResponse{}
	}
	question := dagIntentQuestion(text)
	ask, reason := a.storeClarification(sessionID, dagIntentClarifyAction, text, text, question)
	if !ask {
		debugtrace.Record("dag_intent.clarification_skipped", traceID, map[string]interface{}{
			"reason": reason,
		})
		return skill_step.CLIResponse{}
	}
	debugtrace.Record("dag_intent.question", traceID, map[string]interface{}{
		"root_text": text,
	})
	return skill_step.CLIResponse{
		Text:   setQuestionFloatingCandidates(question, traceID),
		Action: "提問",
		Target: question,
		Next:   actionchain.StandbyNext,
	}
}

// consumeDagIntentAffirmation 攔截「任務確認」pending 的肯定補答。
// 命中 → 清 pending＋root，回 Action=任務 + Target=原句（前端映射 StartTaskProgress）。
// 未命中 → 完全不動 pending，交給 consumePendingToolAnswer 走一般 re-judge。
func (a *App) consumeDagIntentAffirmation(sessionID, userText, traceID string) (*skill_step.CLIResponse, bool) {
	if a == nil {
		return nil, false
	}
	answer := strings.TrimSpace(userText)
	if answer == "" {
		return nil, false
	}
	a.toolReadinessMu.Lock()
	pending, ok := a.pendingToolQuestions[sessionID]
	if !ok || pending.Action != dagIntentClarifyAction {
		a.toolReadinessMu.Unlock()
		return nil, false
	}
	// TTL 交給 consumePendingToolAnswer 的既有分支清理，這裡只放行未過期的。
	if !pending.ExpiresAt.IsZero() && time.Now().After(pending.ExpiresAt) {
		a.toolReadinessMu.Unlock()
		return nil, false
	}
	if dagIntentNegativePattern.MatchString(answer) || !dagIntentAffirmPattern.MatchString(answer) {
		a.toolReadinessMu.Unlock()
		return nil, false
	}
	rootText := strings.TrimSpace(pending.Target)
	delete(a.pendingToolQuestions, sessionID)
	delete(a.clarRoots, sessionID)
	a.toolReadinessMu.Unlock()
	debugtrace.Record("dag_intent.affirmed", traceID, map[string]interface{}{
		"root_text": rootText,
	})
	return &skill_step.CLIResponse{
		Text:   "好，這就把它排成任務。",
		Action: "任務",
		Target: rootText,
		Next:   actionchain.StandbyNext,
	}, true
}
