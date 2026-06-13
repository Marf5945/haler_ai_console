// adapter.go 實作 CLIAdapter 介面，透過 SidecarManager IPC 與 Node 通訊。
//
// #I-803: SkillInjection 只在記憶體中傳遞，不持久化。
//   - SendMessage 將 SkillInjection 轉為 JSON payload 傳給 Node
//   - Node 端應在命令執行完畢後將 injection 變數設為 null
//   - Go 端不在任何磁碟檔案中儲存 injection payload
//
// v4.0: Conversation Continuity — 在送出 CLI 請求前由 Go Controller 合成 prompt
//   - 句子編號追蹤（SentenceStore）
//   - 歷史摘要 + 原始句子 + 當前輸入 → 合成完整 prompt
//   - 意圖分類 + action tag 路由
package cli_manager

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/data/conversation"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/controlseal"
	"ui_console/shared/eventbus"
)

// rpcTimeout 是單次 IPC 呼叫的最長等待時間。
const rpcTimeout = 90 * time.Second
const actionChainMaxRetries = 2

// sendMessageParams 是 sendMessage RPC 的參數結構。
// 注意：SkillInjection 為指標，nil 時 JSON 欄位為 null，
// Node 端應檢查此欄位，null 代表無 skill 注入。
type sendMessageParams struct {
	AdapterID      string                `json:"adapter_id"`
	CLIPath        string                `json:"cli_path"`
	WorkspaceDir   string                `json:"workspace_dir"`
	SessionID      string                `json:"session_id"`
	UserText       string                `json:"user_text"`
	Model          string                `json:"model,omitempty"` // 使用者於 UI 雙擊選定的 CLI model（空字串=用 CLI 預設）
	SkillInjection *skill_step.Injection `json:"skill_injection"` // 可為 null
	TraceID        string                `json:"trace_id"`        // DEBUG_TRACE_REMOVE
}

// SidecarCLIAdapter 透過 SidecarManager 實作 skill_step.CLIAdapter 介面。
// 它是 Go Controller 與 Node Sidecar 之間的橋梁：
//   - 將 SkillInjection 嵌入 IPC 請求（記憶體傳遞，不碰磁碟）
//   - Node 回傳結果後直接映射為 CLIResponse
//   - v4.0: 合成 prompt（歷史 + 摘要 + 當前輸入）後再送出
type SidecarCLIAdapter struct {
	manager       *SidecarManager
	workspaceRoot string

	// v4.0: 對話連續性元件
	sentences    *conversation.SentenceStore // 句子管理（記憶體優先）
	counter      *conversation.CharCounter   // 字數統計
	summaries    []conversation.Summary      // 當前有效摘要
	systemPrompt string                      // 固定 system prompt（不摘要）
	actionTags   []string                    // 當前可用動作 tag
	sealManager  *controlseal.Manager        // Controller-only command seal
	sealSettings controlseal.Settings        // project-level seal rotation setting

	continuityMu sync.Mutex
	continuities map[string]*continuityState

	// §29.3 摘要觸發 — eventbus + cooldown
	eventBus             *eventbus.Bus
	summaryCooldownChars int  // cooldown 結束時的字數門檻
	summaryTriggered     bool // 是否已觸發過（避免重複）
	summaryDismissedAt   int  // 使用者點「稍後」時的字數
	summaryContinuityKey string

	// modelProvider：由 app 注入；給定 adapter_id 回傳使用者選的 model。
	// 為 nil 或回傳空字串時走 CLI 自己的預設。
	modelProvider func(adapterID string) string
}

type continuityState struct {
	sentences       *conversation.SentenceStore
	counter         *conversation.CharCounter
	summaries       []conversation.Summary
	successfulTurns int
}

// NewSidecarCLIAdapter 建立一個 SidecarCLIAdapter。
func NewSidecarCLIAdapter(manager *SidecarManager, workspaceRoot string) *SidecarCLIAdapter {
	sentences := conversation.NewSentenceStore()
	counter := conversation.NewCharCounter()
	return &SidecarCLIAdapter{
		manager:       manager,
		workspaceRoot: workspaceRoot,
		sentences:     sentences,
		counter:       counter,
		sealManager:   controlseal.NewManager(controlseal.DefaultSettings()),
		sealSettings:  controlseal.DefaultSettings(),
		continuities: map[string]*continuityState{
			"default": {sentences: sentences, counter: counter},
		},
	}
}

// SetSystemPrompt 設定固定 system prompt（v4.0：不摘要）
func (a *SidecarCLIAdapter) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
}

// SetModelProvider 注入 model 選擇器（由 app.go 從 settings.Service 包成 closure 傳入）。
func (a *SidecarCLIAdapter) SetModelProvider(fn func(adapterID string) string) {
	a.modelProvider = fn
}

// SetActionTags 設定當前可用動作 tag（來自 skill/mcp/app/sub）
func (a *SidecarCLIAdapter) SetActionTags(tags []string) {
	a.actionTags = tags
}

// SetControlSealSettings updates the project-level rotation interval.
func (a *SidecarCLIAdapter) SetControlSealSettings(settings controlseal.Settings) {
	a.sealSettings = settings.Normalize()
}

// AddSummary 加入一段摘要
func (a *SidecarCLIAdapter) AddSummary(s conversation.Summary) {
	a.addSummaryToContinuity("default", s)
}

func (a *SidecarCLIAdapter) addSummaryToContinuity(key string, s conversation.Summary) {
	a.summaries = append(a.summaries, s)
	a.getContinuity(key).summaries = append(a.getContinuity(key).summaries, s)
}

// GetSentenceStore 取得句子管理器（供外部存取）
func (a *SidecarCLIAdapter) GetSentenceStore() *conversation.SentenceStore {
	return a.sentences
}

// GetCharCounter 取得字數計算器
func (a *SidecarCLIAdapter) GetCharCounter() *conversation.CharCounter {
	return a.counter
}

// NeedsSummarization 檢查是否需要觸發摘要整理
func (a *SidecarCLIAdapter) NeedsSummarization() bool {
	return a.getContinuity("default").counter.NeedsSummarization()
}

// SetEventBus 注入 eventbus（供摘要觸發事件使用）。
func (a *SidecarCLIAdapter) SetEventBus(bus *eventbus.Bus) {
	a.eventBus = bus
}

// DismissSummarization 使用者點「稍後」— 啟動 5000 字 cooldown。
func (a *SidecarCLIAdapter) DismissSummarization() {
	a.summaryDismissedAt = a.getContinuity("default").counter.Count()
	a.summaryCooldownChars = a.summaryDismissedAt + 5000
	a.summaryTriggered = false
}

// checkAndEmitSummarizationNeeded 檢查摘要觸發條件，emit 事件。
// 規則：≥10000 字 + 尚未觸發 + 不在 cooldown 中。
func (a *SidecarCLIAdapter) checkAndEmitSummarizationNeeded(continuityKey string, state *continuityState) {
	if a.eventBus == nil {
		return
	}
	if state == nil || state.counter == nil {
		return
	}
	// 對話字數累積在 session continuity；不能看全域 default counter。
	count := state.counter.Count()

	// 門檻未達
	if count < conversation.SummarizationThreshold {
		return
	}

	// 已觸發過且未被 dismiss
	if a.summaryTriggered {
		return
	}

	// Cooldown 中（使用者曾點稍後）
	if a.summaryCooldownChars > 0 && count < a.summaryCooldownChars {
		return
	}

	// 觸發
	a.summaryTriggered = true
	if strings.TrimSpace(continuityKey) == "" {
		continuityKey = "default"
	}
	a.summaryContinuityKey = continuityKey
	a.eventBus.Emit("summarization:needed", map[string]interface{}{
		"total_chars":    count,
		"threshold":      conversation.SummarizationThreshold,
		"continuity_key": continuityKey,
	})
}

// callModelOnce 送一次 prompt 給模型，不走對話歷史合成、不做 action-chain 重試。
// 專供內部用途（摘要）使用，不污染 SentenceStore / 連續性。
func (a *SidecarCLIAdapter) callModelOnce(adapterID, cliPath, model, systemPrompt, userText, traceID string) (string, error) {
	if a.manager.State() != StateRunning {
		return "", fmt.Errorf("cli_manager: sidecar not running")
	}
	workspaceDir, err := a.ensureWorkspaceDir(adapterID, "summarize")
	if err != nil {
		return "", err
	}
	chosenModel := strings.TrimSpace(model)
	if chosenModel == "" && a.modelProvider != nil {
		chosenModel = strings.TrimSpace(a.modelProvider(adapterID))
	}
	prompt := userText
	if strings.TrimSpace(systemPrompt) != "" {
		prompt = systemPrompt + "\n\n" + userText
	}
	params := sendMessageParams{
		AdapterID:    adapterID,
		CLIPath:      cliPath,
		WorkspaceDir: workspaceDir,
		SessionID:    "summarize",
		UserText:     prompt,
		Model:        chosenModel,
		TraceID:      traceID,
	}
	resp, err := a.manager.Call("sendMessage", params, rpcTimeout)
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("cli rpc error: %s", resp.Error)
	}
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return strings.TrimSpace(string(resp.Result)), nil
	}
	return strings.TrimSpace(result.Text), nil
}

// summarizeLLM 實作 conversation.SummarizationLLM：透過 callModelOnce 呼叫模型壓縮對話。
type summarizeLLM struct {
	adapter   *SidecarCLIAdapter
	adapterID string
	cliPath   string
	model     string
}

func (l summarizeLLM) Summarize(content string, maxOutputChars int) (string, error) {
	sys := "你是對話摘要器。只輸出摘要文字，不要任何前綴、解釋或動作格式。"
	prompt := fmt.Sprintf("請把以下對話壓縮成不超過 %d 字的重點摘要：\n\n%s", maxOutputChars, content)
	traceID := fmt.Sprintf("summarize-%d", time.Now().UnixNano())
	return l.adapter.callModelOnce(l.adapterID, l.cliPath, l.model, sys, prompt, traceID)
}

// RunSummarizationNow 立即執行一次摘要（針對 default 連續性——即觸發門檻所看的那桶）：
// 取未摘要句子 → 模型壓縮 → AddSummary（之後這些句子會被當已摘要過濾）→ 扣字數 → 解除已觸發旗標。
// 摘要失敗回 error，呼叫端據此「不靜默送舊 context」（Rule 11）。寫 summaries.md 由 App 層執行。
func (a *SidecarCLIAdapter) RunSummarizationNow(adapterID, cliPath, model string) (conversation.Summary, error) {
	continuityKey := a.summaryContinuityKey
	if strings.TrimSpace(continuityKey) == "" {
		continuityKey = "default"
	}
	state := a.getContinuity(continuityKey)
	_, rawSentences := a.promptHistorySnapshot(state)
	if len(rawSentences) == 0 {
		return conversation.Summary{}, fmt.Errorf("沒有可摘要的句子")
	}
	var content strings.Builder
	ids := make([]string, 0, len(rawSentences))
	for _, sent := range rawSentences {
		content.WriteString(fmt.Sprintf("[%s] %s: %s\n", sent.ID, sent.Role, sent.Content))
		ids = append(ids, sent.ID)
	}
	origChars := len([]rune(content.String()))
	target := origChars * 6 / 10 // 壓縮目標 ~60%
	if target < 200 {
		target = 200
	}
	llm := summarizeLLM{adapter: a, adapterID: adapterID, cliPath: cliPath, model: model}
	summaryText, err := llm.Summarize(content.String(), target)
	if err != nil {
		return conversation.Summary{}, fmt.Errorf("摘要失敗: %w", err)
	}
	summaryText = strings.TrimSpace(summaryText)
	if summaryText == "" {
		return conversation.Summary{}, fmt.Errorf("摘要結果為空")
	}
	now := time.Now()
	sum := conversation.Summary{
		Tag:             fmt.Sprintf("S-%d", now.Unix()%100000),
		Content:         summaryText,
		SentenceIDs:     ids,
		Timestamp:       now,
		Valid:           true,
		OriginalContent: content.String(), // 給 caller 落 deep_memory，防摘要喪失
	}
	a.addSummaryToContinuity(continuityKey, sum)
	state.counter.Subtract(origChars)
	a.summaryTriggered = false
	return sum, nil
}

func (a *SidecarCLIAdapter) getContinuity(key string) *continuityState {
	if key == "" {
		key = "default"
	}
	a.continuityMu.Lock()
	defer a.continuityMu.Unlock()
	if a.continuities == nil {
		a.continuities = make(map[string]*continuityState)
	}
	if state, ok := a.continuities[key]; ok {
		return state
	}
	state := &continuityState{
		sentences: conversation.NewSentenceStore(),
		counter:   conversation.NewCharCounter(),
	}
	a.continuities[key] = state
	return state
}

func (a *SidecarCLIAdapter) promptHistorySnapshot(state *continuityState) ([]conversation.Summary, []conversation.Sentence) {
	if state == nil {
		return nil, nil
	}
	var validSummaries []conversation.Summary
	for _, s := range state.summaries {
		if s.Valid && state.sentences.CheckSummaryIntegrity(s.SentenceIDs) {
			validSummaries = append(validSummaries, s)
		}
	}
	summarizedIDs := make(map[string]bool)
	for _, s := range validSummaries {
		for _, id := range s.SentenceIDs {
			summarizedIDs[id] = true
		}
	}
	var rawSentences []conversation.Sentence
	for _, sent := range state.sentences.GetAll() {
		if !summarizedIDs[sent.ID] {
			rawSentences = append(rawSentences, sent)
		}
	}
	return validSummaries, rawSentences
}

func isNeedToolJudgeResponse(text string) bool {
	firstLine := strings.TrimSpace(strings.Split(strings.TrimSpace(text), "\n")[0])
	firstLine = strings.Trim(firstLine, "。.!！ \t\r\n")
	return firstLine == "需要工具"
}

// SendMessage 實作 skill_step.CLIAdapter 介面。
// v4.0: 在送出前，先合成包含歷史的完整 prompt。
// 透過 IPC 將合成後的 prompt 與 SkillInjection 傳給 Node。
// SkillInjection 只存在於此次呼叫的記憶體中，不會被寫入磁碟。
func (a *SidecarCLIAdapter) SendMessage(opts skill_step.CLIMessageOptions) (skill_step.CLIResponse, error) {
	if a.manager.State() != StateRunning {
		return skill_step.CLIResponse{
			Error: "sidecar not running",
		}, fmt.Errorf("cli_manager: sidecar not running")
	}

	// continuityKey 將下方主聊天與上方互動拆成不同歷史桶。
	isToolJudge := opts.ToolRoutingMode == "judge"
	isControlMessage := opts.SkipContinuity || isToolJudge || isAdapterControlMessage(opts.UserText)
	inputSentenceID := ""
	continuityKey := opts.ContinuityKey
	if continuityKey == "" {
		continuityKey = opts.SessionID
	}
	if strings.TrimSpace(continuityKey) == "" {
		continuityKey = "default"
	}
	state := a.getContinuity(continuityKey)
	systemPrompt := opts.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = a.systemPrompt
	}

	synthesizedPrompt := opts.UserText
	rawSentenceCount := 0
	validSummaryCount := 0
	windowElided := 0
	if isToolJudge {
		validSummaries, rawSentences := a.promptHistorySnapshot(state)
		// 滑動視窗(熱點 1):tool judge 只需近期脈絡,套用同一視窗。
		windowed := conversation.ApplyPromptWindow(rawSentences, conversation.DefaultPromptWindowConfig())
		rawSentences = windowed.Sentences
		windowElided = windowed.Elided
		rawSentenceCount = len(rawSentences)
		validSummaryCount = len(validSummaries)
		judgePrompt := systemPrompt + "\n\n工具判斷規則：只能輸出兩種格式之一：\n1. 需要工具\n2. 閒聊ㄌ<回答>\n若需要搜尋本機文件、讀取資料、開啟/寫入/匯出、排程、已保存螢幕操作、DAG/自動流程或任何系統工具，只輸出「需要工具」。\nReplay / 重現 / 操作 / 開啟 / 關閉 / 點擊 / 已保存示範相關請求必須輸出「需要工具」。\n不要輸出動作清單，不要猜工具名稱，不要直接輸出無前綴自然語言。"
		synthesizedPrompt = conversation.Synthesize(conversation.SynthesisConfig{
			SystemPrompt: judgePrompt,
			Summaries:    validSummaries,
			RawSentences: rawSentences,
			CurrentInput: opts.UserText,
			SanitizeLLM:  true,
		})
	}

	// Adapter control messages are health probes or internal commands, not user
	// conversation. They must not be inserted into the continuity store, or the
	// next real prompt will appear to repeat old/internal messages.
	if !isControlMessage {
		// ── v4.0: 記錄輸入句子 ──
		inputSentence := state.sentences.AddInput(opts.UserText)
		inputSentenceID = inputSentence.ID
		state.counter.Add(len([]rune(opts.UserText)))

		// ── v4.0: 合成完整 prompt ──
		validSummaries, rawSentences := a.promptHistorySnapshot(state)
		filteredRawSentences := rawSentences[:0]
		for _, sent := range rawSentences {
			if sent.ID != inputSentence.ID {
				filteredRawSentences = append(filteredRawSentences, sent)
			}
		}
		// 滑動視窗(熱點 1):讀取時套用,SentenceStore 不動,
		// 出問題設 conversation.PromptWindowEnabled = false 即退回全量。
		windowed := conversation.ApplyPromptWindow(filteredRawSentences, conversation.DefaultPromptWindowConfig())
		rawSentences = windowed.Sentences
		windowElided = windowed.Elided
		rawSentenceCount = len(rawSentences)
		validSummaryCount = len(validSummaries)

		// 組裝合成 prompt
		if a.sealManager == nil {
			a.sealManager = controlseal.NewManager(controlseal.DefaultSettings())
		}
		currentSeal := a.sealManager.CurrentSeal()
		synthesizedPrompt = conversation.Synthesize(conversation.SynthesisConfig{
			SystemPrompt: systemPrompt,
			ActionTags:   append([]string(nil), a.actionTags...),
			Summaries:    validSummaries,
			RawSentences: rawSentences,
			CurrentInput: opts.UserText,
			CommandSeal:  currentSeal,
			IsCommand:    opts.IsCommand,
			SanitizeLLM:  true,
		})
	}

	log.Printf("[CONV] 合成 prompt: key=%s sentences=%d summaries=%d window_elided=%d chars=%d input_id=%s control=%v system_prompt_len=%d",
		continuityKey, rawSentenceCount, validSummaryCount, windowElided, state.counter.Count(), inputSentenceID, isControlMessage, len([]rune(systemPrompt)))

	// 組裝 IPC 參數——使用合成後的 prompt 取代原始 userText
	workspaceDir, err := a.ensureWorkspaceDir(opts.AdapterID, continuityKey)
	if err != nil {
		return skill_step.CLIResponse{Error: err.Error()}, err
	}
	// 單次呼叫的 model 優先；沒有 override 才讀 UI 雙擊保存的 adapter 設定。
	chosenModel := strings.TrimSpace(opts.Model)
	if chosenModel == "" && a.modelProvider != nil {
		chosenModel = strings.TrimSpace(a.modelProvider(opts.AdapterID))
	}
	params := sendMessageParams{
		AdapterID:      opts.AdapterID,
		CLIPath:        opts.CLIPath,
		WorkspaceDir:   workspaceDir,
		SessionID:      opts.SessionID,
		UserText:       synthesizedPrompt, // v4.0: 合成 prompt 取代原始文字
		Model:          chosenModel,
		SkillInjection: opts.SkillInjection, // nil = 無注入
		TraceID:        opts.TraceID,
	}
	displayText := ""
	rawText := ""
	var parsedChain actionchain.ActionChain
	hasParsedChain := false
	for attempt := 0; attempt <= actionChainMaxRetries; attempt++ {
		if attempt == 0 {
			params.UserText = synthesizedPrompt
		} else {
			// Retry is intentionally short: ask the model to repair only protocol shape.
			params.UserText = synthesizedPrompt + "\n\n" + actionchain.RetryPrompt()
		}
		// DEBUG_TRACE_REMOVE: Captures the exact JSON payload before it goes to Node.
		paramsTrace := map[string]interface{}{
			"adapter_id":          params.AdapterID,
			"cli_path":            params.CLIPath,
			"workspace_dir":       params.WorkspaceDir,
			"session_id":          params.SessionID,
			"continuity_key":      continuityKey,
			"retry_attempt":       attempt,
			"model":               params.Model,
			"system_prompt_len":   len([]rune(systemPrompt)),
			"has_skill_injection": params.SkillInjection != nil,
		}
		addTraceUserText(paramsTrace, opts.TraceID, params.UserText)
		debugtrace.Record("go.SidecarCLIAdapter.params", opts.TraceID, paramsTrace)

		resp, err := a.manager.Call("sendMessage", params, rpcTimeout)
		if err != nil {
			// DEBUG_TRACE_REMOVE: IPC call error before a CLI response is available.
			debugtrace.Record("go.SidecarCLIAdapter.call.error", opts.TraceID, map[string]interface{}{
				"error":         err.Error(),
				"retry_attempt": attempt,
			})
			return skill_step.CLIResponse{Error: err.Error()}, err
		}
		if resp.Error != "" {
			// DEBUG_TRACE_REMOVE: Node returned an RPC error.
			debugtrace.Record("go.SidecarCLIAdapter.rpc.error", opts.TraceID, map[string]interface{}{
				"error":         resp.Error,
				"retry_attempt": attempt,
			})
			return skill_step.CLIResponse{Error: resp.Error}, fmt.Errorf("cli rpc error: %s", resp.Error)
		}
		// DEBUG_TRACE_REMOVE: Raw Node JSON result before Go maps it into CLIResponse.
		debugtrace.Record("go.SidecarCLIAdapter.rawResult", opts.TraceID, map[string]interface{}{
			"result":        string(resp.Result),
			"retry_attempt": attempt,
		})
		log.Printf("[CLI_MONITOR] Go IPCResponse.Result trace=%s retry=%d rpc_error=%q result_len=%d result=%s", opts.TraceID, attempt, resp.Error, len(resp.Result), string(resp.Result))

		// 解析 Node 回傳的結果。
		// Sidecar 可能回傳兩種格式：
		//   1. 正常回應：{text: "..."}
		//   2. 授權請求：{auth_required: true, adapter_id: "...", auth_url: "...", message: "..."}
		var result struct {
			Text         string `json:"text"`
			AuthRequired bool   `json:"auth_required"`
			AdapterID    string `json:"adapter_id"`
			AuthURL      string `json:"auth_url"`
			Message      string `json:"message"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			log.Printf("[CLI_MONITOR] Go CLIResponse.Text trace=%s source=unmarshal_fallback text_len=%d text=%s", opts.TraceID, len(resp.Result), string(resp.Result))
			return skill_step.CLIResponse{
				Text: string(resp.Result), // fallback：直接使用原始 JSON
			}, nil
		}

		// 如果 sidecar 回報 CLI 需要授權，將授權資訊透傳給上層（app.go），
		// 由 app.go 負責開瀏覽器和通知前端。
		if result.AuthRequired {
			log.Printf("[CLI_MONITOR] Go CLIResponse.Text trace=%s source=auth_required text_len=%d text=%s", opts.TraceID, len(result.Message), result.Message)
			return skill_step.CLIResponse{
				AuthRequired: true,
				AuthURL:      result.AuthURL,
				AdapterID:    result.AdapterID,
				Text:         result.Message,
			}, nil
		}

		rawText = result.Text
		displayText = result.Text
		log.Printf("[CLI_MONITOR] Go CLIResponse.Text trace=%s source=normal retry=%d text_len=%d text=%s", opts.TraceID, attempt, len(result.Text), result.Text)

		if isControlMessage {
			if isToolJudge && !isNeedToolJudgeResponse(result.Text) {
				state.sentences.AddInput(opts.UserText)
				state.sentences.AddOutput(result.Text)
				state.counter.Add(len([]rune(opts.UserText)) + len([]rune(result.Text)))
				a.checkAndEmitSummarizationNeeded(continuityKey, state)
			}
			return skill_step.CLIResponse{Text: result.Text}, nil
		}

		// TASK 28: ㄌ has structure only in direct LLM output.
		chain, parseErr := actionchain.Parse(result.Text)
		if parseErr != nil {
			log.Printf("[ACTION_CHAIN] structure error trace=%s retry=%d err=%v retry_prompt=%q", opts.TraceID, attempt, parseErr, actionchain.RetryPrompt())
			if attempt < actionChainMaxRetries {
				continue
			}
			break
		}
		parsedChain = chain
		hasParsedChain = true

		registry := actionchain.NewStaticRegistry(a.actionTags...)
		validation := actionchain.ValidateActionTag(chain.Action, registry)
		if validation.Code == actionchain.ValidationUnknown {
			log.Printf("[ACTION_CHAIN] unknown tag trace=%s tag=%q review_card_needed=true", opts.TraceID, chain.Action)
		}
		if decision := actionchain.ResolveBuiltIn(chain); decision.Handled {
			// Built-in tags render in the Controller; same-name tools do not auto-run.
			displayText = decision.DisplayText
			debugtrace.Record("go.SidecarCLIAdapter.actionChain.builtin", opts.TraceID, map[string]interface{}{
				"action":       decision.Chain.Action,
				"target":       decision.Chain.Target,
				"next":         decision.Chain.Next,
				"terminal":     decision.Terminal,
				"display_text": decision.DisplayText,
			})
		} else if validation.Code == actionchain.ValidationOK {
			// Non-builtin valid action: extract display text and emit routing event.
			// v3.1.8：loop 來源不出工具卡，同 app.go API 路徑的收斂。
			displayText = chain.Target
			if a.eventBus != nil && !eventbus.IsTaskLoopTrace(opts.TraceID) {
				a.eventBus.Emit(eventbus.EventSchedulerActionRequested, map[string]string{
					"action":     chain.Action,
					"target":     chain.Target,
					"next":       chain.Next,
					"trace_id":   opts.TraceID,
					"session_id": opts.SessionID,
					"raw":        chain.Raw,
				})
			}
			debugtrace.Record("go.SidecarCLIAdapter.actionChain.routed", opts.TraceID, map[string]interface{}{
				"action":       chain.Action,
				"target":       chain.Target,
				"next":         chain.Next,
				"display_text": displayText,
			})
		}
		state.successfulTurns++
		a.sealManager.RotateIfNeeded(state.successfulTurns, a.sealSettings)
		break
	}

	// ── v4.0: 記錄輸出句子 + 更新字數 ──
	state.sentences.AddOutput(displayText)
	state.counter.Add(len([]rune(displayText)))

	// §29.3: 檢查摘要觸發條件
	a.checkAndEmitSummarizationNeeded(continuityKey, state)

	// ── v4.0: 意圖分類（事後標記）──
	if len(a.actionTags) > 0 {
		intent := conversation.ClassifyIntent(rawText, a.actionTags)
		if intent.IsAction {
			log.Printf("[CONV] 偵測到動作 tag: %s", intent.ActionTag)
			// 動作 tag 會由上層（app.go）處理路由
		}
	}

	resp := skill_step.CLIResponse{Text: displayText}
	if hasParsedChain {
		resp.Action = parsedChain.Action
		resp.Target = parsedChain.Target
		resp.Next = parsedChain.Next
	}
	return resp, nil
}

func isAdapterControlMessage(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	var msg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(trimmed), &msg); err != nil {
		return false
	}
	switch msg.Type {
	case "ping", "healthcheck", "health_check":
		return true
	default:
		return false
	}
}

func addTraceUserText(fields map[string]interface{}, traceID, text string) {
	if isTaskProgressTraceID(traceID) {
		preview := truncateTraceRunes(text, 360)
		fields["user_text_len"] = len([]rune(text))
		fields["user_text_preview"] = preview
		fields["user_text_compacted"] = len([]rune(text)) > len([]rune(preview))
		return
	}
	fields["user_text"] = text
}

func isTaskProgressTraceID(traceID string) bool {
	return strings.HasPrefix(traceID, "task-plan-") || strings.HasPrefix(traceID, "task-plan-repair-") || strings.HasPrefix(traceID, "task-node-")
}

func truncateTraceRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

func (a *SidecarCLIAdapter) ensureWorkspaceDir(adapterID, continuityKey string) (string, error) {
	if a.workspaceRoot == "" {
		return "", nil
	}
	dir := filepath.Join(a.workspaceRoot, sanitizeAdapterWorkspaceName(adapterID))
	// 依對話線分目錄，避免 adapter 本地檔案互相覆寫。
	continuityName := sanitizeAdapterWorkspaceName(continuityKey)
	if continuityName != "" && continuityName != "default" {
		dir = filepath.Join(dir, continuityName)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cli_manager: create CLI workspace: %w", err)
	}
	return dir, nil
}

func sanitizeAdapterWorkspaceName(adapterID string) string {
	name := strings.ToLower(strings.TrimSpace(adapterID))
	if name == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	cleaned := strings.Trim(b.String(), "-.")
	if cleaned == "" {
		return "default"
	}
	return cleaned
}
