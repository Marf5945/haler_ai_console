package main

import (
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
)

const (
	backgroundSourceUserInput     = "user_input"
	backgroundSourceClarification = "clarification"
	backgroundSourceUserProfile   = "user_profile"
	backgroundSourceEnvironment   = "environment"
	backgroundSourceDefault       = "default"
	backgroundSourceLLMGuess      = "llm_guess"

	confidenceUserInput     = 100
	confidenceClarification = 100
	confidenceUserProfile   = 85
	confidenceEnvironment   = 75
	confidenceDefault       = 60
	confidenceLLMGuess      = 0
)

type pendingToolQuestion struct {
	SessionID        string
	Action           string
	Target           string
	MissingContext   string
	Question         string
	OriginalUserText string
	CreatedAt        time.Time
}

type toolBackgroundAnswer struct {
	Type       string
	Question   string
	Answer     string
	Source     string
	Confidence int
}

type toolReadinessQuestion struct {
	MissingContext string
	Question       string
}

func (a *App) consumePendingToolAnswer(sessionID, userText, traceID string) (toolRoutingDecision, bool) {
	if a == nil {
		return toolRoutingDecision{}, false
	}
	answer := strings.TrimSpace(userText)
	if answer == "" {
		return toolRoutingDecision{}, false
	}
	a.toolReadinessMu.Lock()
	pending, ok := a.pendingToolQuestions[sessionID]
	if !ok {
		a.toolReadinessMu.Unlock()
		return toolRoutingDecision{}, false
	}
	delete(a.pendingToolQuestions, sessionID)
	a.toolBackgroundContexts[sessionID] = append(a.toolBackgroundContexts[sessionID], toolBackgroundAnswer{
		Type:       pending.MissingContext,
		Question:   pending.Question,
		Answer:     answer,
		Source:     backgroundSourceClarification,
		Confidence: confidenceClarification,
	})
	a.toolReadinessMu.Unlock()

	debugtrace.Record("tool_readiness.clarification", traceID, map[string]interface{}{
		"action":          pending.Action,
		"target":          pending.Target,
		"missing_context": pending.MissingContext,
		"confidence":      confidenceClarification,
	})
	return toolRoutingDecision{
		Kind:   toolRoutingDecisionAction,
		Action: pending.Action,
		Target: pending.Target,
		Next:   actionchain.StandbyNext,
		Raw:    mergePendingToolQuestionContext(pending.OriginalUserText, pending.Question, answer),
	}, true
}

func mergePendingToolQuestionContext(original, question, answer string) string {
	original = strings.TrimSpace(original)
	answer = strings.TrimSpace(answer)
	if original == "" {
		return answer
	}
	if answer == "" {
		return original
	}
	return original + "\n\n使用者補充:\n" + strings.TrimSpace(question) + "\n" + answer
}

func (a *App) maybeAskForToolReadiness(sessionID string, decision toolRoutingDecision, userText string, traceID string) (bool, skill_step.CLIResponse) {
	if a == nil || decision.Kind != toolRoutingDecisionAction || !isReadinessAction(decision.Action) {
		return false, skill_step.CLIResponse{}
	}
	question, needQuestion := a.assessToolReadiness(sessionID, decision, userText)
	if !needQuestion {
		return false, skill_step.CLIResponse{}
	}
	a.toolReadinessMu.Lock()
	if a.pendingToolQuestions == nil {
		a.pendingToolQuestions = make(map[string]pendingToolQuestion)
	}
	a.pendingToolQuestions[sessionID] = pendingToolQuestion{
		SessionID:        sessionID,
		Action:           decision.Action,
		Target:           decision.Target,
		MissingContext:   question.MissingContext,
		Question:         question.Question,
		OriginalUserText: userText,
		CreatedAt:        time.Now(),
	}
	a.toolReadinessMu.Unlock()

	debugtrace.Record("tool_readiness.question", traceID, map[string]interface{}{
		"action":          decision.Action,
		"target":          decision.Target,
		"next":            decision.Next,
		"missing_context": question.MissingContext,
		"question":        question.Question,
	})
	return true, skill_step.CLIResponse{
		Text:   question.Question,
		Action: decision.Action,
		Target: decision.Target,
		Next:   actionchain.QuestionNext,
	}
}

func (a *App) assessToolReadiness(sessionID string, decision toolRoutingDecision, userText string) (toolReadinessQuestion, bool) {
	action := strings.TrimSpace(decision.Action)
	target := strings.TrimSpace(decision.Target)
	if actionchain.IsQuestionNext(decision.Next) {
		return inferredQuestionForAction(action, target), true
	}
	// 使用者本輪明確打了地點（即使抽詞掉了）也不該再問。
	if action == "網路" && isContextSensitiveWebQuery(target) && !a.hasBackgroundContext(sessionID, "地點") && !containsLocationHint(target) && !containsLocationHint(userText) {
		return toolReadinessQuestion{MissingContext: "地點", Question: "你想查哪個地點？"}, true
	}
	return toolReadinessQuestion{}, false
}

func (a *App) hasBackgroundContext(sessionID, kind string) bool {
	a.toolReadinessMu.Lock()
	defer a.toolReadinessMu.Unlock()
	for _, item := range a.toolBackgroundContexts[sessionID] {
		if item.Type == kind && strings.TrimSpace(item.Answer) != "" && item.Confidence >= confidenceUserProfile {
			return true
		}
	}
	return false
}

func (a *App) targetWithBackground(sessionID, target string) string {
	background := a.formatToolBackgroundContext(sessionID)
	if background == "" {
		return strings.TrimSpace(target)
	}
	return strings.TrimSpace(target) + "\n\n" + background
}

func (a *App) formatToolBackgroundContext(sessionID string) string {
	if a == nil {
		return ""
	}
	a.toolReadinessMu.Lock()
	items := append([]toolBackgroundAnswer(nil), a.toolBackgroundContexts[sessionID]...)
	a.toolReadinessMu.Unlock()
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[已補充背景]\n")
	for _, item := range items {
		if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" {
			continue
		}
		b.WriteString(item.Question)
		b.WriteString("\n")
		b.WriteString(item.Answer)
		b.WriteString("\n\n")
	}
	b.WriteString("[/已補充背景]")
	return b.String()
}

func isReadinessAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "網路", "搜尋", "查詢", "讀取", "操作":
		return true
	default:
		return false
	}
}

func inferredQuestionForAction(action, target string) toolReadinessQuestion {
	if action == "網路" && isContextSensitiveWebQuery(target) {
		return toolReadinessQuestion{MissingContext: "地點", Question: "你想查哪個地點？"}
	}
	switch action {
	case "查詢", "搜尋":
		return toolReadinessQuestion{MissingContext: "查詢範圍", Question: "你想查哪一類儲存資料？"}
	case "讀取":
		return toolReadinessQuestion{MissingContext: "檔案範圍", Question: "你要讀取哪個檔案或資料夾？"}
	case "操作":
		return toolReadinessQuestion{MissingContext: "操作目標", Question: "你要執行哪個已保存操作？"}
	default:
		return toolReadinessQuestion{MissingContext: "背景資訊", Question: "請補充必要背景，讓我可以正確處理這個請求。"}
	}
}

func isContextSensitiveWebQuery(text string) bool {
	lower := strings.ToLower(text)
	if containsAny(lower, []string{"weather", "rain", "temperature", "forecast"}) {
		return true
	}
	return containsAny(text, []string{"天氣", "下雨", "降雨", "氣溫", "溫度", "預報"})
}

func containsLocationHint(text string) bool {
	lower := strings.ToLower(text)
	if containsAny(lower, []string{"taipei", "taichung", "tainan", "kaohsiung", "tokyo", "osaka", "seoul", "london", "new york"}) {
		return true
	}
	return containsAny(text, []string{
		"台北", "臺北", "新北", "桃園", "新竹", "苗栗", "台中", "臺中", "彰化", "南投",
		"雲林", "嘉義", "台南", "臺南", "高雄", "屏東", "宜蘭", "花蓮", "台東", "臺東",
		"澎湖", "金門", "馬祖", "東京", "大阪", "首爾", "倫敦", "紐約",
	})
}
