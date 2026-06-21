package main

// tool_readiness.go
//
// 收尾版 Step 2 + Step 3（兩者共用同一套澄清狀態機，故一起改）。
//
// Step 2——二次 judge + 不再拼背景進 query：
//   - 廢除關鍵字驅動的「缺地點就問」臆測（移除 isContextSensitiveWebQuery /
//     containsLocationHint）。澄清改由 judge 自己決定（輸出 提問ㄌ問題ㄌ待命，
//     在 responseFromToolRoutingDecision 的提問分支交給 handleJudgeClarification）。
//   - consumePendingToolAnswer 不再沿用舊 decision，而是合成「乾淨 re-judge 文字」
//     （根問題 + 累積答案，不含問句鷹架），回給上層重跑完整 routing（第二次 judge）。
//   - targetWithBackground 整個移除：背景只進 judge/composer context，不進出境 query。
//
// Step 3——澄清迴圈煞車：
//   - clarificationRoot 跨輪保存 根問題 / Count / AskedSlots / 累積答案 / TTL。
//   - 2 分鐘 TTL（沿用 web_search_egress 模式）；過期不再 merge，當新問題。
//   - 同一原始問題最多 maxClarificationRounds 次；同一 canonical slot 重問即停。
//   - 消費補答後「立即刪除 pending」反劫持：下一則非補答訊息不會被誤併進這題。
//   - 耗盡分支採產品取捨：不猜測、不執行，回自然語言說明缺什麼（避免做錯）。

import (
	"fmt"
	"hash/fnv"
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

	// Step 3：澄清迴圈煞車參數。
	toolClarificationTTL   = 2 * time.Minute // 與 web_search_egress 的 pending TTL 一致
	maxClarificationRounds = 2               // 同一原始問題最多澄清次數
)

// pendingToolQuestion 是「目前尚未被回答的那一題」。跨輪的 count/根問題在
// clarificationRoot；這裡只放本題、且消費後即刪（反劫持）。
type pendingToolQuestion struct {
	SessionID      string
	Action         string
	Target         string
	MissingContext string // 原始自由字串（trace/debug 用）
	CanonicalSlot  string // 正規化後的 slot（去重比對用）
	Question       string
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

// clarificationRoot 是一整串澄清的「根」，跨輪穩定，用來：
//   - 套用 Count / AskedSlots 上限（Step 3 煞車）
//   - 累積乾淨答案以合成 re-judge 文字（Step 2 第二次 judge）
//   - 供 composer 取背景（formatToolBackgroundContext）
//
// 自帶 TTL：過期即視同清空，count 不會卡死、背景不會殘留汙染後續對話。
type clarificationRoot struct {
	RootText   string
	Hash       string
	Count      int
	AskedSlots map[string]bool
	Answers    []clarAnswer
	ExpiresAt  time.Time
}

type clarAnswer struct {
	Question string
	Answer   string
}

// toolBackgroundAnswer 保留型別以相容既有 App 欄位宣告（toolBackgroundContexts）。
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

// consumePendingToolAnswer 接住「上一輪問了澄清」後的補答。
// 回 (rejudgeText, true)：本輪以 rejudgeText 重跑完整 routing（第二次 judge）。
// 回 ("", false)：沒有有效 pending（或已過 TTL，當新問題處理）。
func (a *App) consumePendingToolAnswer(sessionID, userText, traceID string) (string, bool) {
	if a == nil {
		return "", false
	}
	answer := strings.TrimSpace(userText)
	if answer == "" {
		return "", false
	}
	a.toolReadinessMu.Lock()
	pending, ok := a.pendingToolQuestions[sessionID]
	if !ok {
		a.toolReadinessMu.Unlock()
		return "", false
	}
	// Step 3：TTL。補答太晚 → 丟棄，當新問題，不 merge 陳舊背景。
	if !pending.ExpiresAt.IsZero() && time.Now().After(pending.ExpiresAt) {
		delete(a.pendingToolQuestions, sessionID)
		delete(a.clarRoots, sessionID)
		a.toolReadinessMu.Unlock()
		debugtrace.Record("tool_readiness.pending_expired", traceID, map[string]interface{}{
			"canonical_slot": pending.CanonicalSlot,
		})
		return "", false
	}
	// Step 2 反劫持：消費後立即刪 pending，下一則非補答訊息不會被誤併進這題。
	delete(a.pendingToolQuestions, sessionID)
	root := a.clarRoots[sessionID]
	if root == nil {
		root = &clarificationRoot{
			RootText:   strings.TrimSpace(pending.Target),
			Hash:       shortHash(pending.Target),
			AskedSlots: map[string]bool{},
			ExpiresAt:  time.Now().Add(toolClarificationTTL),
		}
		a.clarRoots[sessionID] = root
	}
	root.Answers = append(root.Answers, clarAnswer{Question: pending.Question, Answer: answer})
	root.ExpiresAt = time.Now().Add(toolClarificationTTL)
	clarCount := root.Count
	rejudge := cleanRejudgeText(root.RootText, root.Answers)
	a.toolReadinessMu.Unlock()

	debugtrace.Record("tool_readiness.clarification", traceID, map[string]interface{}{
		"canonical_slot": pending.CanonicalSlot,
		"clar_count":     clarCount,
		"rejudge_text":   rejudge,
	})
	return rejudge, true
}

// cleanRejudgeText 合成第二次 judge 的輸入：根問題 + 累積答案，不含任何問句鷹架，
// 因此第二次 judge 能直接產生乾淨 target（撞點 2/3 的根因解法）。
func cleanRejudgeText(root string, answers []clarAnswer) string {
	root = strings.TrimSpace(root)
	clean := make([]string, 0, len(answers))
	for _, qa := range answers {
		if t := strings.TrimSpace(qa.Answer); t != "" {
			clean = append(clean, t)
		}
	}
	if len(clean) == 0 {
		return root
	}
	if root == "" {
		return strings.Join(clean, "；")
	}
	return root + "（補充：" + strings.Join(clean, "；") + "）"
}

// storeClarification 記一筆待澄清並套用 Step 3 煞車。
// 回 (true, "")：可以問；(false, reason)：達上限/重複 slot，上層走耗盡分支。
func (a *App) storeClarification(sessionID, action, target, rootText, question string) (bool, string) {
	slot := canonicalMissingSlot(question)
	now := time.Now()
	a.toolReadinessMu.Lock()
	defer a.toolReadinessMu.Unlock()
	if a.pendingToolQuestions == nil {
		a.pendingToolQuestions = make(map[string]pendingToolQuestion)
	}
	if a.clarRoots == nil {
		a.clarRoots = make(map[string]*clarificationRoot)
	}
	root := a.clarRoots[sessionID]
	if root == nil || now.After(root.ExpiresAt) {
		root = &clarificationRoot{
			RootText:   strings.TrimSpace(rootText),
			Hash:       shortHash(rootText),
			AskedSlots: map[string]bool{},
		}
	}
	if root.Count >= maxClarificationRounds {
		delete(a.clarRoots, sessionID)
		delete(a.pendingToolQuestions, sessionID)
		return false, "max_rounds"
	}
	if root.AskedSlots[slot] {
		delete(a.clarRoots, sessionID)
		delete(a.pendingToolQuestions, sessionID)
		return false, "same_slot"
	}
	root.Count++
	root.AskedSlots[slot] = true
	root.ExpiresAt = now.Add(toolClarificationTTL)
	a.clarRoots[sessionID] = root
	a.pendingToolQuestions[sessionID] = pendingToolQuestion{
		SessionID:      sessionID,
		Action:         action,
		Target:         target,
		MissingContext: question,
		CanonicalSlot:  slot,
		Question:       question,
		CreatedAt:      now,
		ExpiresAt:      now.Add(toolClarificationTTL),
	}
	return true, ""
}

// clearClarification 清掉本 session 的待澄清與澄清根（耗盡 / TTL / 解析完成時呼叫）。
func (a *App) clearClarification(sessionID string) {
	if a == nil {
		return
	}
	a.toolReadinessMu.Lock()
	delete(a.pendingToolQuestions, sessionID)
	delete(a.clarRoots, sessionID)
	a.toolReadinessMu.Unlock()
}

// handleJudgeClarification 處理 judge 主動輸出的 提問ㄌ問題ㄌ待命。
// 套 Step 3 煞車；達上限就走耗盡分支（不執行、自然語言說明）。
func (a *App) handleJudgeClarification(sessionID, rootText, question, traceID string) skill_step.CLIResponse {
	question = strings.TrimSpace(question)
	ask, reason := a.storeClarification(sessionID, "提問", question, rootText, question)
	if !ask {
		debugtrace.Record("tool_readiness.clarification_exhausted", traceID, map[string]interface{}{
			"reason":   reason,
			"question": question,
		})
		a.clearClarification(sessionID)
		return skill_step.CLIResponse{Text: clarificationExhaustedMessage(question)}
	}
	debugtrace.Record("tool_readiness.question", traceID, map[string]interface{}{
		"action":         "提問",
		"question":       question,
		"canonical_slot": canonicalMissingSlot(question),
	})
	return skill_step.CLIResponse{
		Text:   setQuestionFloatingCandidates(question, traceID),
		Action: "提問",
		Target: question,
		Next:   actionchain.StandbyNext,
	}
}

// clarificationExhaustedMessage：超過上限後的回覆。產品取捨＝寧可不做也不要做錯，
// 因此不猜測、不執行，明確說出還缺什麼，把決定權交回使用者。
func clarificationExhaustedMessage(question string) string {
	q := strings.TrimSpace(question)
	if q == "" {
		return "我還缺一些必要資訊才能繼續，可以再說明一下嗎？"
	}
	return "我問了幾次仍缺必要資訊，先不貿然執行以免做錯。還需要：" + q
}

// maybeAskForToolReadiness 處理「工具動作但 judge 標記 next=提問」的次要澄清路徑。
// 已移除關鍵字臆測；只在 judge 明確 QuestionNext 時才問，並走同一套煞車。
func (a *App) maybeAskForToolReadiness(sessionID string, decision toolRoutingDecision, userText string, traceID string) (bool, skill_step.CLIResponse) {
	if a == nil || decision.Kind != toolRoutingDecisionAction || !isReadinessAction(decision.Action) {
		return false, skill_step.CLIResponse{}
	}
	question, need := a.assessToolReadiness(decision, userText)
	if !need {
		return false, skill_step.CLIResponse{}
	}
	ask, reason := a.storeClarification(sessionID, decision.Action, decision.Target, userText, question.Question)
	if !ask {
		debugtrace.Record("tool_readiness.clarification_exhausted", traceID, map[string]interface{}{
			"reason":   reason,
			"question": question.Question,
		})
		a.clearClarification(sessionID)
		return true, skill_step.CLIResponse{Text: clarificationExhaustedMessage(question.Question)}
	}
	debugtrace.Record("tool_readiness.question", traceID, map[string]interface{}{
		"action":          decision.Action,
		"target":          decision.Target,
		"next":            decision.Next,
		"missing_context": question.MissingContext,
		"canonical_slot":  canonicalMissingSlot(question.Question),
		"question":        question.Question,
	})
	return true, skill_step.CLIResponse{
		Text:   question.Question,
		Action: decision.Action,
		Target: decision.Target,
		Next:   actionchain.QuestionNext,
	}
}

// ---------------------------------------------------------------------------
// 出境前『資料充分性』驗證（egress sufficiency pass）
// ---------------------------------------------------------------------------
// 設計：不枚舉主題、不查關鍵字表。對「即將送出網路的 query」做一次無狀態的
// 自問自答——讓模型逐查詢判斷「答案是否取決於使用者尚未提供的關鍵 slot」。
//   足夠 → OK；不足 → 提問ㄌ<一句具體問題>ㄌ待命。
// 不足時呼叫端把 Next 翻成 提問，交既有 handleJudgeClarification（自帶 TTL +
// 次數煞車），所以天氣缺地點、星座缺星座、股價缺標的全包，新主題零修改。
// 失效保護：judge 為 nil 或呼叫出錯一律放行（fail-open），不因模型抖動擋住搜尋。

// querySufficiencyCheckPrompt 組「充分性檢查」的無狀態提示。輸入只有使用者原句
// 與即將出境的 query，不挾帶對話狀態，符合『模型無狀態、每次重新判斷』。
func querySufficiencyCheckPrompt(userText, query string) string {
	sep := actionchain.Separator
	return strings.Join([]string{
		"你是送出網路查詢前的『資料充分性』把關。只判斷一件事：這個查詢要得到對使用者有用的答案，是否取決於使用者『尚未提供』的關鍵資訊（例如地點、日期或時間、對象身分、標的、單位等）。",
		"足夠回答 → 只輸出：OK",
		"關鍵資訊不足 → 只輸出：提問" + sep + "<一句具體、好回答的問題>" + sep + "待命",
		"規則：只問會改變結果的關鍵資訊；能直接查就輸出 OK，不要為了問而問；不要解釋、不要輸出多行。",
		"使用者原句：" + strings.TrimSpace(userText),
		"即將送出的查詢：" + strings.TrimSpace(query),
	}, "\n")
}

// assessQuerySufficiency 跑出境充分性驗證。回 (問句, true) 代表資料不足、需反問；
// 回 ("", false) 代表足夠或無法判斷（放行）。judge 是無狀態的一次性模型呼叫。
func (a *App) assessQuerySufficiency(decision toolRoutingDecision, userText string, judge searchRerouteJudge, traceID string) (string, bool) {
	if a == nil || judge == nil {
		return "", false
	}
	query := strings.TrimSpace(decision.Target)
	if query == "" {
		return "", false
	}
	out, err := judge(querySufficiencyCheckPrompt(userText, query))
	if err != nil {
		// fail-open：模型出錯不擋搜尋，只記錄供 monitor 觀察。
		debugtrace.Record("tool_readiness.sufficiency_error", traceID, map[string]interface{}{
			"query": query,
			"error": err.Error(),
		})
		return "", false
	}
	parsed := parseToolRoutingDecision(out)
	question := strings.TrimSpace(parsed.Target)
	insufficient := parsed.Action == "提問" && question != ""
	debugtrace.Record("tool_readiness.sufficiency", traceID, map[string]interface{}{
		"query":        query,
		"raw":          strings.TrimSpace(out),
		"insufficient": insufficient,
		"question":     question,
	})
	if insufficient {
		return question, true
	}
	return "", false
}

// assessToolReadiness 只在 judge 明確標記此工具動作 next=提問 時才產生問句；
// 不再用關鍵字表臆測缺漏（移除了「預報→地點」這類誤判的根源）。
func (a *App) assessToolReadiness(decision toolRoutingDecision, userText string) (toolReadinessQuestion, bool) {
	_ = userText
	if actionchain.IsQuestionNext(decision.Next) {
		return inferredQuestionForAction(decision.Action, decision.Target), true
	}
	return toolReadinessQuestion{}, false
}

// formatToolBackgroundContext 給 composer 用的背景（僅 judge/composer，不進出境 query）。
// 從 clarificationRoot 讀，過期即不提供，避免殘留背景汙染後續對話。
func (a *App) formatToolBackgroundContext(sessionID string) string {
	if a == nil {
		return ""
	}
	a.toolReadinessMu.Lock()
	root := a.clarRoots[sessionID]
	var answers []clarAnswer
	if root != nil && !time.Now().After(root.ExpiresAt) {
		answers = append(answers, root.Answers...)
	}
	a.toolReadinessMu.Unlock()
	if len(answers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[已補充背景]\n")
	for _, qa := range answers {
		if strings.TrimSpace(qa.Question) == "" || strings.TrimSpace(qa.Answer) == "" {
			continue
		}
		b.WriteString(qa.Question)
		b.WriteString("\n")
		b.WriteString(qa.Answer)
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

// inferredQuestionForAction：已移除 網路→地點 的關鍵字預設。網路缺資訊時走通用問句，
// 真正「該不該問、問什麼」交由 judge 決定。
func inferredQuestionForAction(action, target string) toolReadinessQuestion {
	_ = target
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

func shortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(s)))
	return fmt.Sprintf("%08x", h.Sum32())
}
