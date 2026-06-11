// skill_execution_binding.go — TASK 31：Skill 執行接線（Wails bindings）。
// 把路由結果 + lifecycle + riskgrant 收斂成執行決策，auto/授權通過時建立
// injection，並沿用既有 SendCLIMessage/SendAPIMessage 管線實際送出。
package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/domain/review"
	"ui_console/domain/risk"
	"ui_console/orchestration/skill_eval"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/controlseal"
	"ui_console/shared/eventbus"
)

// SkillExecutionDecision 是回給前端的決策 DTO。
type SkillExecutionDecision struct {
	Decision     string                  `json:"decision"` // auto_execute|need_confirm|candidate|review|no_skill|cancelled
	ResolveID    string                  `json:"resolve_id"`
	SkillID      string                  `json:"skill_id,omitempty"`
	Status       string                  `json:"status"`
	Injected     bool                    `json:"injected"` // auto 時是否已建立 injection
	Executed     bool                    `json:"executed"` // 是否已送入 CLI/API 執行
	ActionTarget string                  `json:"action_target,omitempty"`
	Response     *skill_step.CLIResponse `json:"response,omitempty"`
	Message      string                  `json:"message,omitempty"` // 需確認/無 skill 時給前端的提示文字
}

type SkillDraftSaveResult struct {
	Manifest *skill_step.SkillManifest `json:"manifest"`
	Problems []string                  `json:"problems"`
}

// findManifestForExec 從歸檔取出指定 skill 的 manifest（含 lifecycle）。
func (a *App) findManifestForExec(skillID string) *skill_step.SkillManifest {
	if skillID == "" {
		return nil
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return nil
	}
	for i := range manifests {
		if manifests[i].SkillID == skillID {
			m := manifests[i] // 複製一份，避免外洩切片底層
			return &m
		}
	}
	return nil
}

// ExecuteResolvedSkill：resolve → DecideExecution →（auto 時）build injection。
func (a *App) ExecuteResolvedSkill(actionTarget, sessionID string) (*SkillExecutionDecision, error) {
	return a.resolveSkillExecution(actionTarget, sessionID)
}

func (a *App) resolveSkillExecution(actionTarget, sessionID string) (*SkillExecutionDecision, error) {
	result, err := a.resolveSkillForActionTarget(actionTarget, sessionID)
	if err != nil {
		return nil, err
	}
	manifest := a.findManifestForExec(result.SelectedSkillID)
	var lc *skill_step.Lifecycle
	if manifest != nil {
		lc = manifest.Lifecycle
	}
	hasGrant := manifest != nil && skill_eval.HasSkillGrant(a.skillGrants, manifest)

	decision := skill_eval.DecideExecution(result.Status, lc, hasGrant)
	out := &SkillExecutionDecision{
		Decision:     string(decision),
		ResolveID:    result.ResolveID,
		SkillID:      result.SelectedSkillID,
		Status:       string(result.Status),
		ActionTarget: actionTarget,
	}
	switch decision {
	case skill_eval.ExecAuto:
		if _, err := a.BuildSkillContext(result.ResolveID, sessionID); err != nil {
			return nil, err
		}
		out.Injected = true
	case skill_eval.ExecNeedConfirm:
		out.Message = "這個 skill 尚未完成執行權限確認。要允許它執行這次任務嗎？"
	case skill_eval.ExecNoSkill:
		out.Message = "目前沒有對應 skill，將以一般流程完成；完成後可詢問是否存成 skill。"
	}
	return out, nil
}

// ExecuteSkillMessage 是 TASK 31 的完整後端入口：
// userText → action target → resolve/decision → inject → SendCLI/API。
// no_skill 會走一般訊息流程；need_confirm/candidate/review 交回前端處理。
func (a *App) ExecuteSkillMessage(adapterID, sessionID, userText, traceID string) (*SkillExecutionDecision, error) {
	actionTarget, ok := a.inferSkillActionTarget(userText)
	if !ok {
		resp, err := a.sendSkillMessage(adapterID, sessionID, userText, traceID)
		if err != nil {
			return nil, err
		}
		return &SkillExecutionDecision{
			Decision: string(skill_eval.ExecNoSkill),
			Executed: true,
			Response: resp,
			Message:  "未偵測到可路由的 skill action，已走一般流程。",
		}, nil
	}
	invocationID := a.hookGeneInvocationID(traceID, "skill-execution")
	out, err := a.resolveSkillExecution(actionTarget, sessionID)
	if err != nil {
		return nil, err
	}
	geneSkillID := hookGeneSkillID(out.SkillID)
	if geneSkillID != "" {
		a.emitHookGeneDataEntered(geneSkillID, invocationID)
		a.emitHookGeneDataProcessed(geneSkillID, invocationID)
		// 保證所有 return 路徑都收尾（與 go_program / native replay 一致），避免缺 completed 造成 pending 殘留。
		defer a.emitHookGeneCompleted(geneSkillID, invocationID)
	}
	if blocked := a.createBlockedSkillDriftReview(actionTarget, out.SkillID, sessionID, userText); blocked != nil {
		out.Decision = string(skill_eval.ExecReview)
		out.Executed = false
		out.Message = "偵測到高風險 skill drift，已建立 review card，暫停本次執行。"
		if geneSkillID != "" {
			a.emitHookGenePaused(geneSkillID, invocationID)
		}
		return out, nil
	}
	switch skill_eval.ExecDecision(out.Decision) {
	case skill_eval.ExecAuto:
		resp, err := a.sendSkillMessage(adapterID, sessionID, userText, traceID)
		if err != nil {
			if geneSkillID != "" {
				a.emitHookGenePaused(geneSkillID, invocationID)
			}
			return nil, err
		}
		out.Executed = true
		out.Response = resp
		// skill 使用紀錄 producer：auto 路徑執行成功後記一筆到當前 agent 的
		// tool_history.jsonl，供「拉出 sub / 匯出 sub」自動關聯用過的 skill。
		// displayName 留空，由歸檔以 SkillID 反查補齊。
		a.recordSkillUsage(out.SkillID, "", sessionID, traceID)
		if geneSkillID != "" {
			a.emitHookGeneDataLeft(geneSkillID, invocationID, true)
		}
	case skill_eval.ExecNoSkill:
		resp, err := a.sendSkillMessage(adapterID, sessionID, userText, traceID)
		if err != nil {
			return nil, err
		}
		out.Executed = true
		out.Response = resp
	default:
		if geneSkillID != "" {
			a.emitHookGenePaused(geneSkillID, invocationID)
		}
	}
	return out, nil
}

// ConfirmSkillExecution 處理前端 [允許一次/總是允許/取消]。
// choice: "allow_once" | "always" | "cancel"
func (a *App) ConfirmSkillExecution(resolveID, sessionID, choice string) (*SkillExecutionDecision, error) {
	a.cacheMu.Lock()
	result, ok := a.resolveCache[resolveID]
	a.cacheMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("ConfirmSkillExecution: resolve %q 不存在或已過期", resolveID)
	}
	manifest := a.findManifestForExec(result.SelectedSkillID)
	if manifest == nil {
		return nil, fmt.Errorf("ConfirmSkillExecution: 找不到 skill %q", result.SelectedSkillID)
	}
	out := &SkillExecutionDecision{ResolveID: resolveID, SkillID: manifest.SkillID, Status: string(result.Status)}

	switch choice {
	case "allow_once":
		skill_eval.GrantOnce(a.skillGrants, manifest) // 帶 TTL，不改 manifest
	case "always":
		skill_eval.PromoteToEnabled(manifest) // lifecycle → enabled_skill
		if err := a.skillArchive.UpdateManifest(manifest); err != nil {
			return nil, err
		}
	case "cancel":
		out.Decision = "cancelled"
		return out, nil
	default:
		return nil, fmt.Errorf("ConfirmSkillExecution: 未知選項 %q", choice)
	}

	// 授權通過 → 建立 injection（與 auto 路徑相同）。
	if _, err := a.BuildSkillContext(resolveID, sessionID); err != nil {
		return nil, err
	}
	out.Decision = string(skill_eval.ExecAuto)
	out.Injected = true
	return out, nil
}

// ConfirmAndExecuteSkillExecution 接續 need_confirm：授權成功後立即送出原訊息。
func (a *App) ConfirmAndExecuteSkillExecution(resolveID, sessionID, choice, adapterID, userText, traceID string) (*SkillExecutionDecision, error) {
	out, err := a.ConfirmSkillExecution(resolveID, sessionID, choice)
	if err != nil || out == nil || out.Decision == "cancelled" {
		return out, err
	}
	invocationID := a.hookGeneInvocationID(traceID, "skill-confirm-execution")
	geneSkillID := hookGeneSkillID(out.SkillID)
	if geneSkillID != "" {
		a.emitHookGeneDataEntered(geneSkillID, invocationID)
		a.emitHookGeneDataProcessed(geneSkillID, invocationID)
		// defer 收尾，保證 sendSkillMessage 成功/失敗兩條路徑都結 gene。
		defer a.emitHookGeneCompleted(geneSkillID, invocationID)
	}
	resp, err := a.sendSkillMessage(adapterID, sessionID, userText, traceID)
	if err != nil {
		if geneSkillID != "" {
			a.emitHookGenePaused(geneSkillID, invocationID)
		}
		return nil, err
	}
	out.Executed = true
	out.Response = resp
	// skill 使用紀錄 producer：need_confirm → 使用者授權後執行成功，同樣要記。
	a.recordSkillUsage(out.SkillID, "", sessionID, traceID)
	if geneSkillID != "" {
		a.emitHookGeneDataLeft(geneSkillID, invocationID, true)
	}
	return out, nil
}

// BuildSkillDraft：無 skill 時，由 LLM 提供欄位建立 pending 草稿（先驗證）。
// 回傳草稿與問題清單；問題非空 → 草稿為 draft_candidate（不顯示為工具）。
// 注意：實際寫入磁碟（建立新 skill 目錄結構）建議沿用 ArchiveService 既有歸檔流程；
// 此處只負責產生並驗證草稿，持久化由呼叫端依 archive 佈局接線。
func (a *App) BuildSkillDraft(skillID, displayName string, actionTags, domainTags []string, chain *skill_step.ExpectedChain) (*skill_step.SkillManifest, []string) {
	return skill_eval.BuildPendingDraft(skillID, displayName, actionTags, domainTags, chain)
}

func (a *App) BuildAndSaveSkillDraft(skillID, displayName string, actionTags, domainTags []string, chain *skill_step.ExpectedChain) (*SkillDraftSaveResult, error) {
	draft, problems := skill_eval.BuildPendingDraft(skillID, displayName, actionTags, domainTags, chain)
	saved, err := a.skillArchive.SavePendingDraft(draft)
	if err != nil {
		return nil, err
	}
	return &SkillDraftSaveResult{Manifest: saved, Problems: problems}, nil
}

func (a *App) resolveSkillForActionTarget(actionTarget, sessionID string) (*skill_step.ResolveResult, error) {
	at, err := skill_step.ParseActionTarget(actionTarget)
	if err != nil {
		return nil, err
	}
	result, err := a.skillRouter.Resolve(at, sessionID)
	if err != nil {
		return nil, err
	}
	a.cacheMu.Lock()
	a.resolveCache[result.ResolveID] = result
	a.cacheMu.Unlock()
	return result, nil
}

func (a *App) sendSkillMessage(adapterID, sessionID, userText, traceID string) (*skill_step.CLIResponse, error) {
	if strings.TrimSpace(adapterID) == "" {
		adapterID = a.defaultSkillExecutionAdapterID()
	}
	if strings.TrimSpace(adapterID) == "" {
		return nil, fmt.Errorf("skill execution: no adapter available")
	}
	isAPI := a.isAPIOrLocalAdapter(adapterID)
	isLocal := strings.HasPrefix(adapterID, "local-")
	if a.adapterRegistry != nil {
		if adapter, err := a.adapterRegistry.GetStatus(adapterID); err == nil {
			isLocal = adapter.Kind == "local"
		}
	}
	route := "composer_three_stage_routing"
	if isLocal {
		route = "local_single_pass_guided_routing"
	}
	debugtrace.Record("go.skillExecution.sendMessage", traceID, map[string]interface{}{
		"adapter_id": adapterID,
		"session_id": sessionID,
		"text_len":   len([]rune(userText)),
		"route":      route,
		"is_api":     isAPI,
		"is_local":   isLocal,
	})
	if isAPI {
		return a.SendAPIMessage(adapterID, sessionID, userText, traceID)
	}
	return a.SendCLIMessage(adapterID, sessionID, userText, traceID)
}

func (a *App) defaultSkillExecutionAdapterID() string {
	if a == nil || a.adapterRegistry == nil {
		return ""
	}
	adapters := a.adapterRegistry.ListAvailable()
	for _, adapter := range adapters {
		if adapter.ID != "" && adapter.Status != "offline" {
			return adapter.ID
		}
	}
	if len(adapters) > 0 {
		return adapters[0].ID
	}
	return ""
}

func (a *App) inferSkillActionTarget(input string) (string, bool) {
	text := strings.TrimSpace(input)
	if text == "" {
		return "", false
	}
	// Natural language must enter composer routing first:
	// stage 1 keyword extraction -> stage 2 lookup-aware judge -> stage 3 tool menu.
	// Only explicit internal action-chain text should bypass that router.
	if !strings.Contains(text, actionchain.Separator) {
		return "", false
	}
	if chain, err := actionchain.Parse(text); err == nil {
		return chain.Action + "ㄌ" + chain.Target, true
	}
	if at, err := skill_step.ParseActionTarget(text); err == nil {
		return at.Action + "ㄌ" + at.Target, true
	}
	return "", false
}

func (a *App) createBlockedSkillDriftReview(actionTarget, skillID, sessionID, rawInput string) *review.Card {
	step, err := skill_eval.ParseStep(actionTarget)
	if err != nil {
		return nil
	}
	sanitized := controlseal.SanitizeForLLM(controlseal.SourceUserRaw, rawInput)
	drifts := skill_eval.EvaluateLowLevel(step, sanitized, "", "")
	var blocked []skill_eval.DriftEvent
	for _, drift := range drifts {
		if drift.Blocked {
			blocked = append(blocked, drift)
		}
	}
	if len(blocked) == 0 {
		return nil
	}
	_ = skill_eval.NewStore(appDataRoot(), "default").AppendEvent(skill_eval.EventRecord{
		SessionID: sessionID,
		SkillID:   skillID,
		Drifts:    blocked,
		Note:      "blocked drift routed to review card",
	})
	card := a.reviewService.AddCard(review.CardParams{
		RiskClass:      risk.HighNonDestructive,
		Operation:      "skill_drift_review",
		Target:         actionTarget,
		Reason:         "偵測到高風險 skill drift，需確認後才能繼續。",
		AcceptLabel:    "確認後重試",
		RejectLabel:    "取消",
		AcceptEffect:   "保留 review 紀錄，使用者可重新送出任務",
		RejectEffect:   "不執行本次 skill 呼叫",
		SourceType:     "drift",
		SourceID:       skillID,
		EngineerReason: blocked[0].Reason,
	})
	a.eventBus.Emit(eventbus.EventReviewCardAdded, card)
	return &card
}

// --- need_confirm 待確認狀態（修 skill 權限確認迴圈）---------------------------
// 舊流程：need_confirm 只回一句純文字「要允許嗎？」，使用者回「要」又被丟回三段式
// 路由 → 又判 need_confirm → 無限迴圈，永遠到不了 ConfirmAndExecuteSkillExecution。
// 新流程：need_confirm 時把 resolveID 記在 session，下一句肯定/否定改由
// maybeHandlePendingSkillConfirm 在 LLM 路由前直接接手。

type pendingSkillConfirm struct {
	ResolveID    string
	SkillID      string
	Target       string
	ActionChain  string
	AdapterID    string
	OriginalText string
	ExpiresAt    time.Time
}

var (
	pendingSkillConfirmMu sync.Mutex
	pendingSkillConfirms  = map[string]pendingSkillConfirm{} // sessionID → pending
)

func rememberPendingSkillConfirm(sessionID string, p pendingSkillConfirm) {
	p.ExpiresAt = time.Now().Add(10 * time.Minute)
	pendingSkillConfirmMu.Lock()
	pendingSkillConfirms[sessionID] = p
	pendingSkillConfirmMu.Unlock()
}

func clearPendingSkillConfirm(sessionID string) {
	pendingSkillConfirmMu.Lock()
	delete(pendingSkillConfirms, sessionID)
	pendingSkillConfirmMu.Unlock()
}

// skillConfirmPrompt 組 need_confirm 的提示文字：點名 skill，並列出目前已載入、
// 會被當輸入的引用檔；沒有任何引用檔時提醒先載入資料（修「沒跟我要資料」）。
func (a *App) skillConfirmPrompt(target string) string {
	var b strings.Builder
	b.WriteString("要用 skill「" + target + "」執行這次任務嗎？回覆「要」開始，或「取消」放棄。")
	refs := a.recentReferenceFilesForRouting(6)
	if len(refs) == 0 {
		b.WriteString("\n（目前沒有偵測到已載入的資料檔。若這個 skill 需要表格，請先拖入或引用檔案再確認。）")
		return b.String()
	}
	names := make([]string, 0, len(refs))
	for _, r := range refs {
		names = append(names, r.Name)
	}
	b.WriteString("\n將使用這些已載入檔案作為輸入：" + strings.Join(names, "、") + "。")
	return b.String()
}

// maybeHandlePendingSkillConfirm 在 LLM 路由前攔截「使用者正在回應上一輪 skill 權限確認」。
// 肯定（confirmRe）→ 直接 allow_once 並執行原任務；否定（isDeclineText）→ 取消清狀態；
// 其他 → 不攔截（回 false），讓正常路由處理，pending 保留至過期或下一句確認。
func (a *App) maybeHandlePendingSkillConfirm(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	pendingSkillConfirmMu.Lock()
	pending, has := pendingSkillConfirms[sessionID]
	if has && time.Now().After(pending.ExpiresAt) {
		delete(pendingSkillConfirms, sessionID)
		has = false
	}
	pendingSkillConfirmMu.Unlock()
	if !has {
		return nil, false
	}
	lower := strings.ToLower(strings.TrimSpace(userText))

	if isDeclineText(lower) {
		clearPendingSkillConfirm(sessionID)
		clearConfirmQuestion(sessionID)
		debugtrace.Record("go.skillConfirm.cancelled", traceID, map[string]interface{}{
			"target":     pending.Target,
			"resolve_id": pending.ResolveID,
		})
		return &skill_step.CLIResponse{Text: "好，已取消。需要時再說一次就行。"}, true
	}

	if confirmRe.MatchString(lower) {
		clearPendingSkillConfirm(sessionID)
		clearConfirmQuestion(sessionID)
		adapterID := pending.AdapterID
		if strings.TrimSpace(adapterID) == "" {
			adapterID = a.defaultSkillExecutionAdapterID()
		}
		debugtrace.Record("go.skillConfirm.granted", traceID, map[string]interface{}{
			"target":     pending.Target,
			"resolve_id": pending.ResolveID,
		})
		outputsSince := time.Now()
		out, err := a.ConfirmAndExecuteSkillExecution(pending.ResolveID, sessionID, "allow_once", adapterID, pending.OriginalText, traceID)
		if err != nil {
			return &skill_step.CLIResponse{
				Error:  err.Error(),
				Action: "流程",
				Target: pending.Target,
			}, true
		}
		a.harvestSkillOutputs(pending.Target, outputsSince)
		if out != nil && out.Response != nil {
			resp := *out.Response
			resp.Action = "流程"
			resp.Target = pending.Target
			if strings.TrimSpace(resp.Next) == "" {
				resp.Next = actionchain.NormalizeNext("輸出")
			}
			return &resp, true
		}
		msg := "已允許並執行 skill「" + pending.Target + "」。"
		if out != nil && strings.TrimSpace(out.Message) != "" {
			msg = strings.TrimSpace(out.Message)
		}
		return &skill_step.CLIResponse{Text: msg, Action: "流程", Target: pending.Target}, true
	}

	// 既非肯定也非否定：不攔截，交回正常路由；pending 保留待過期或下一句確認。
	return nil, false
}

// maybeHandleSkillFlow 處理「流程」路由：使用既有/已安裝 skill。
// 與「程式」（製作獨立 .go 程式）區隔：流程＝把工作交給既有 skill。
// 用 skill 自身 manifest 標籤組出可被 skillRouter 高分命中的 action-chain，
// 交給 ExecuteSkillMessage（做風險判定、必要時注入 SKILL.md 並送出）。
// decision.Target 不是既有 skill 時回 false，讓後續路由（程式/搜尋等）接手。
func (a *App) maybeHandleSkillFlow(decision toolRoutingDecision, sessionID, traceID, userText string) (bool, skill_step.CLIResponse) {
	if strings.TrimSpace(decision.Action) != "流程" {
		return false, skill_step.CLIResponse{}
	}
	actionChain := a.existingSkillActionChain(decision.Target)
	if actionChain == "" {
		debugtrace.Record("go.skillFlow.no_match", traceID, map[string]interface{}{
			"target": decision.Target,
		})
		return false, skill_step.CLIResponse{}
	}
	// 產出電料Bom：走專屬互動收集流程（手動逐項電料→review→輸出），
	// 不進通用閘 / need_confirm。
	if isDianliaoBomTarget(decision.Target) {
		return true, a.startDianliaoBomFlow(sessionID, traceID)
	}
	// 通用必填輸入閘（data-driven）：執行前先用該 skill 自宣告的 InputSchema.Required
	// 比對「使用者訊息 + 已載入檔案」，缺欄位直接回「提問」，不進 need_confirm。
	// 讀 manifest 推導，對所有 skill 生效，不為任何單一 skill 寫死。
	if required, ok := a.skillRequiredInputs(decision.Target); ok {
		refs := a.recentReferenceFilesForRouting(6)
		refNames := make([]string, 0, len(refs))
		for _, r := range refs {
			refNames = append(refNames, r.Name)
		}
		if missing := missingRequiredInputs(required, userText, refNames); len(missing) > 0 {
			question := skillMissingInputQuestion(decision.Target, missing)
			debugtrace.Record("go.skillFlow.missing_required_input", traceID, map[string]interface{}{
				"target":  decision.Target,
				"missing": missing,
			})
			return true, skill_step.CLIResponse{
				Text:   setQuestionFloatingCandidates(question, traceID),
				Action: "提問",
				Target: question,
				Next:   actionchain.StandbyNext,
			}
		}
	}
	adapterID := a.defaultSkillExecutionAdapterID()
	debugtrace.Record("go.skillFlow.execute", traceID, map[string]interface{}{
		"target":       decision.Target,
		"action_chain": actionChain,
	})
	// 通用產出收集：記錄執行前時間，auto 成功後掃 outputs 把新產出落位上架。
	outputsSince := time.Now()
	out, err := a.ExecuteSkillMessage(adapterID, sessionID, actionChain, traceID)
	if err != nil {
		return true, skill_step.CLIResponse{
			Error:  err.Error(),
			Action: decision.Action,
			Target: decision.Target,
			Next:   actionchain.NormalizeNext(decision.Next),
		}
	}
	// auto 執行成功 → 回 skill 的實際輸出。
	if out != nil && out.Response != nil {
		resp := *out.Response
		resp.Action = decision.Action
		resp.Target = decision.Target
		if strings.TrimSpace(resp.Next) == "" {
			resp.Next = actionchain.NormalizeNext(decision.Next)
		}
		// 把這個 skill 本回合在 outputs 新產生的檔案，依類型落位並上架到右側引用面板。
		a.harvestSkillOutputs(decision.Target, outputsSince)
		return true, resp
	}
	// need_confirm：把確認狀態記在 session，讓使用者下一句「要/取消」直接被
	// maybeHandlePendingSkillConfirm 接手（修 need_confirm 迴圈），不再回三段式路由。
	if out != nil && skill_eval.ExecDecision(out.Decision) == skill_eval.ExecNeedConfirm && strings.TrimSpace(out.ResolveID) != "" {
		rememberPendingSkillConfirm(sessionID, pendingSkillConfirm{
			ResolveID:    out.ResolveID,
			SkillID:      out.SkillID,
			Target:       decision.Target,
			ActionChain:  actionChain,
			AdapterID:    adapterID,
			OriginalText: actionChain,
		})
		msg := a.skillConfirmPrompt(decision.Target)
		rememberConfirmQuestion(sessionID, msg)
		debugtrace.Record("go.skillFlow.need_confirm", traceID, map[string]interface{}{
			"target":     decision.Target,
			"resolve_id": out.ResolveID,
		})
		return true, skill_step.CLIResponse{
			Text:      msg,
			Action:    decision.Action,
			Target:    decision.Target,
			Next:      actionchain.NormalizeNext(decision.Next),
			NeedsUser: true,
		}
	}
	// review / 其他沒有 Response：回 skill 的提示訊息。
	msg := "已路由到既有 skill「" + decision.Target + "」。"
	if out != nil && strings.TrimSpace(out.Message) != "" {
		msg = strings.TrimSpace(out.Message)
	}
	return true, skill_step.CLIResponse{
		Text:      msg,
		Action:    decision.Action,
		Target:    decision.Target,
		Next:      actionchain.NormalizeNext(decision.Next),
		NeedsUser: true, // need_confirm/review：在等使用者，chat_route 節點據此暫停 DAG
	}
}

// existingSkillActionChain 用既有 skill 的 manifest 標籤組出可被 skillRouter
// 高分命中的 action-chain（動作取 action_tag[0]、目標取 target_aliases[0]），
// 避免用分類動詞（如「流程」）導致動作維度 0 分、整體掉到 0.70 而無法自動命中。
// 找不到對應 skill 時回傳空字串。
func (a *App) existingSkillActionChain(name string) string {
	if a == nil || a.skillArchive == nil {
		return ""
	}
	want := normalizeGoProgramLookup(name)
	if want == "" {
		return ""
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return ""
	}
	for _, m := range manifests {
		if normalizeGoProgramLookup(m.DisplayName) != want && normalizeGoProgramLookup(m.SkillID) != want {
			continue
		}
		action := firstNonEmpty(firstListItem(m.Tags.ActionTag), firstListItem(m.Routing.ActionPatterns))
		target := firstNonEmpty(firstListItem(m.Routing.TargetAliases), firstListItem(m.Tags.DomainTag), firstNonEmpty(m.DisplayName, m.SkillID))
		if action == "" || target == "" {
			return ""
		}
		return action + actionchain.Separator + target
	}
	return ""
}

// firstListItem 回傳清單中第一個非空白字串，皆空回 ""。
func firstListItem(items []string) string {
	for _, s := range items {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
