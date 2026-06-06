// skill_execution_binding.go — TASK 31：Skill 執行接線（Wails bindings）。
// 把路由結果 + lifecycle + riskgrant 收斂成執行決策，auto/授權通過時建立
// injection，並沿用既有 SendCLIMessage/SendAPIMessage 管線實際送出。
package main

import (
	"fmt"
	"strings"

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
	out, err := a.resolveSkillExecution(actionTarget, sessionID)
	if err != nil {
		return nil, err
	}
	if blocked := a.createBlockedSkillDriftReview(actionTarget, out.SkillID, sessionID, userText); blocked != nil {
		out.Decision = string(skill_eval.ExecReview)
		out.Executed = false
		out.Message = "偵測到高風險 skill drift，已建立 review card，暫停本次執行。"
		return out, nil
	}
	switch skill_eval.ExecDecision(out.Decision) {
	case skill_eval.ExecAuto:
		resp, err := a.sendSkillMessage(adapterID, sessionID, userText, traceID)
		if err != nil {
			return nil, err
		}
		out.Executed = true
		out.Response = resp
	case skill_eval.ExecNoSkill:
		resp, err := a.sendSkillMessage(adapterID, sessionID, userText, traceID)
		if err != nil {
			return nil, err
		}
		out.Executed = true
		out.Response = resp
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
	resp, err := a.sendSkillMessage(adapterID, sessionID, userText, traceID)
	if err != nil {
		return nil, err
	}
	out.Executed = true
	out.Response = resp
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
