// scheduler_autorun.go — 排程自動執行協調（3.1.10 Phase F）。
//
// 到點成功觸發後（由 scheduler 的 SchedulerJobRunner hook 呼叫）：
//   - 流程改變偵測（flow_hash 不符 → 清除重建，F-5）。
//   - 同日去重（last_output_date；流程改變時允許同日重建）。
//   - 高風險不自動跑、付費雲端 API 需確認（F-4）。
//   - 確保可見 sub，注入「當日日期」在 sub 內跑一輪 CLI（CLI→本地 failover）。
//   - 輸出累積進原 sub（帶日期）；第一次用真實流程做成可審核 skill 草稿（F-2/F-3）。
//   - 完成發 remote_bridge 通知（未綁定則 no-op）。
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/adapter/remote_bridge"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/scheduler"
)

// errSchedulerPaidConfirm：只剩付費雲端 API 可用時，依規格不自動跑、需使用者確認。
var errSchedulerPaidConfirm = errors.New("scheduler: 需確認才會使用付費雲端 API")

// schedulerJobRunner 實作 scheduler.SchedulerJobRunner。
type schedulerJobRunner struct {
	app *App
}

func (r schedulerJobRunner) RunJob(ctx context.Context, job *scheduler.Job) {
	if r.app == nil || job == nil {
		return
	}
	if err := r.app.runScheduledJobOrchestration(ctx, *job); err != nil {
		debugtrace.Record("go.scheduler.autorun.error", "", map[string]interface{}{
			"job_id": job.ID,
			"error":  err.Error(),
		})
	}
}

// runScheduledJobOrchestration 是 F 的 runner 入口（未確認）。
func (a *App) runScheduledJobOrchestration(ctx context.Context, job scheduler.Job) error {
	return a.runScheduledJobOrchestrationOpt(ctx, job, false)
}

// runScheduledJobOrchestrationOpt：confirmed=true 時繞過高風險閘門並允許付費 API
// （使用者在 app 內按「確認執行」後走此路徑）。
func (a *App) runScheduledJobOrchestrationOpt(ctx context.Context, job scheduler.Job, confirmed bool) error {
	if a == nil || a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	dateKey := time.Now().Format("20060102")

	// F-5：流程改變偵測——已有 skill 但 FlowHash 不符 → 視為改了流程，清除重建。
	flowChanged := schedulerJobFlowChanged(job)
	if flowChanged {
		_ = a.schedulerService.ResetJobSkill(job.ID)
		job.SkillID = ""
		job.SourceSubID = ""
		job.AutoRunSkill = false
		job.FlowHash = ""
	}
	if schedulerShouldSkipSameDay(job, dateKey, flowChanged) {
		debugtrace.Record("go.scheduler.autorun.skip_same_day", "", map[string]interface{}{"job_id": job.ID, "date": dateKey})
		return nil
	}
	spec := schedulerBootstrapSpecFromJob(&job)

	// F-4：高風險（非 auto 等級）未確認時不自動跑 → 跳「app 內確認」卡。
	if !confirmed && !scheduler.ShouldAutoExecute(job.RiskClass) {
		a.emitSchedulerConfirmNeeded(job, spec, confirmReasonHighRisk)
		return nil
	}

	// 確保可見 sub。
	sub, err := a.ensureSchedulerBootstrapSub(&job, spec)
	if err != nil {
		return err
	}

	// 組當日 prompt 並在 sub 內跑（CLI→本地 failover；只剩付費 API 則需確認）。
	prompt := schedulerRunPrompt(spec, dateKey)
	_ = a.AppendTalkEntryForAgent(sub.ID, "user", fmt.Sprintf("[排程 %s] %s", dateKey, spec.Action))
	stopHeartbeat := a.startSchedulerHeartbeat(ctx, spec.Title)
	output, runErr := a.runSchedulerTurnInSub(sub.ID, prompt, confirmed)
	stopHeartbeat()
	if errors.Is(runErr, errSchedulerPaidConfirm) {
		a.emitSchedulerConfirmNeeded(job, spec, confirmReasonPaidAPI)
		return nil
	}
	if runErr != nil {
		_ = a.AppendTalkEntryForAgent(sub.ID, "assistant", "排程執行失敗："+runErr.Error())
		a.dispatchSchedulerNotice(remote_bridge.NotifySystemWarning, "排程失敗："+spec.Title, runErr.Error())
		return runErr
	}

	// 輸出累積進原 sub（帶日期）。
	dated := fmt.Sprintf("%s %s\n%s", dateKey, spec.Title, strings.TrimSpace(output))
	_ = a.AppendTalkEntryForAgent(sub.ID, "assistant", dated)

	// 首次：用這次真實流程做成可審核 skill 草稿，並記錄到 job（不改 ActionType）。
	if strings.TrimSpace(job.SkillID) == "" {
		if skillID, serr := a.sealSchedulerSkillFromRun(&job, sub, spec); serr == nil {
			_ = a.schedulerService.RecordJobSkill(job.ID, skillID, sub.ID)
			_ = a.AppendTalkEntryForAgent(sub.ID, "assistant", "已把此流程存成可審核 skill："+skillID+"。之後到點會在此 sub 自動更新資料。")
		} else {
			debugtrace.Record("go.scheduler.autorun.seal_failed", "", map[string]interface{}{"job_id": job.ID, "error": serr.Error()})
		}
	}

	_ = a.schedulerService.MarkJobOutputDate(job.ID, dateKey)

	// 完成通知（remote_bridge，未綁定則 no-op）。
	summary := strings.TrimSpace(output)
	if r := []rune(summary); len(r) > 200 {
		summary = string(r[:200]) + "…"
	}
	a.dispatchSchedulerNotice(remote_bridge.NotifyResultSummary, "排程完成："+spec.Title, summary)
	debugtrace.Record("go.scheduler.autorun.done", "", map[string]interface{}{"job_id": job.ID, "date": dateKey, "sub": sub.ID})
	return nil
}

func schedulerJobFlowChanged(job scheduler.Job) bool {
	return strings.TrimSpace(job.SkillID) != "" &&
		strings.TrimSpace(job.FlowHash) != "" &&
		job.FlowHash != scheduler.ExpectedFlowHash(job)
}

func schedulerShouldSkipSameDay(job scheduler.Job, dateKey string, flowChanged bool) bool {
	return job.LastOutputDate == dateKey && !flowChanged
}

// schedulerRunPrompt 組出「注入當日日期」的執行 prompt。
func schedulerRunPrompt(spec schedulerBootstrapSpec, dateKey string) string {
	return strings.Join([]string{
		"你是排程自動執行器。今天日期：" + dateKey + "。",
		"請執行以下任務，輸出當日最新內容（不要解釋流程、不要詢問）：",
		strings.TrimSpace(spec.Action),
		"補充說明：" + strings.TrimSpace(spec.Summary),
	}, "\n")
}

// runSchedulerTurnInSub 依 failover 順序在 sub session 內跑一輪 CLI，回傳輸出文字。
// 付費 API（failoverAPI）不自動使用——若僅剩 API 可用，回 errSchedulerPaidConfirm。
func (a *App) runSchedulerTurnInSub(sessionID, prompt string, allowPaidAPI bool) (string, error) {
	ordered := orderSchedulerFailover(a.schedulerFailoverCandidates())
	if len(ordered) == 0 {
		return "", fmt.Errorf("scheduler: 沒有可用的執行後端")
	}
	triedNonAPI := false
	sawAPI := false
	var lastErr error
	for _, c := range ordered {
		if c.Kind == failoverAPI && !allowPaidAPI {
			sawAPI = true
			continue // 付費 API 預設不自動用（F-4）；使用者確認後 allowPaidAPI=true 才放行
		}
		triedNonAPI = true
		traceID := fmt.Sprintf("scheduler-run-%d", time.Now().UnixNano())
		var resp *skill_step.CLIResponse
		var err error
		if c.Kind == failoverLocal || c.Kind == failoverAPI {
			resp, err = a.SendAPIMessage(c.AdapterID, sessionID, prompt, traceID)
		} else {
			if a.cliAdapter == nil {
				return "", fmt.Errorf("scheduler: CLI adapter 尚未初始化")
			}
			if serr := a.ensureSidecarRunning(); serr != nil {
				return "", serr
			}
			cliPath := ""
			if a.adapterRegistry != nil {
				if p, rerr := a.adapterRegistry.ResolveExecutable(c.AdapterID); rerr == nil {
					cliPath = p
				}
			}
			cliResp, cliErr := a.cliAdapter.SendMessage(skill_step.CLIMessageOptions{
				AdapterID:     c.AdapterID,
				CLIPath:       cliPath,
				SessionID:     sessionID,
				UserText:      prompt,
				ContinuityKey: conversationContinuityKey("scheduler-run", sessionID),
				TraceID:       traceID,
			})
			resp, err = &cliResp, cliErr
		}
		if err != nil {
			lastErr = err
			if isSchedulerRetriableFailure(err.Error()) {
				continue
			}
			return "", err
		}
		if resp == nil {
			lastErr = fmt.Errorf("scheduler: 後端 %s 無回應", c.AdapterID)
			continue
		}
		if resp.Error != "" {
			lastErr = errors.New(resp.Error)
			if isSchedulerRetriableFailure(resp.Error) {
				continue
			}
			return "", lastErr
		}
		if strings.TrimSpace(resp.Text) == "" {
			lastErr = fmt.Errorf("scheduler: 後端 %s 回傳空內容", c.AdapterID)
			continue
		}
		return resp.Text, nil
	}
	if !triedNonAPI && sawAPI {
		return "", errSchedulerPaidConfirm
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("scheduler: 所有非付費後端皆無法執行")
	}
	return "", lastErr
}

// sealSchedulerSkillFromRun 用第一次成功的流程做成可審核 skill 草稿（pending）。
func (a *App) sealSchedulerSkillFromRun(job *scheduler.Job, sub *CreatedSubagent, spec schedulerBootstrapSpec) (string, error) {
	if a.skillArchive == nil {
		return "", fmt.Errorf("skill archive 尚未初始化")
	}
	skillID := schedulerSkillID(job)
	chain := &skill_step.ExpectedChain{
		Schema:   "skill_chain.v1",
		MaxSteps: 8,
		Steps: []skill_step.ExpectedStep{{
			Action:      "查詢",
			Target:      spec.Action,
			Next:        actionchain.StandbyNext,
			Code:        "ㄔ",
			Requirement: "OP",
		}},
	}
	draft, _ := a.BuildSkillDraft(skillID, spec.Title, []string{"查詢"}, []string{spec.Action, "排程"}, chain)
	draft.Description = spec.Summary
	if draft.Routing.ActionPatterns == nil {
		draft.Routing.ActionPatterns = []string{schedulerActionTarget(spec)}
	}
	if draft.Routing.TargetAliases == nil {
		draft.Routing.TargetAliases = []string{spec.Action, spec.Title}
	}
	if _, err := a.skillArchive.SavePendingDraft(draft); err != nil {
		return "", err
	}
	return skillID, nil
}

// --- H/I：app 內「點一下確認」動線 ---

type schedulerConfirmReason string

const (
	confirmReasonHighRisk schedulerConfirmReason = "high_risk"
	confirmReasonPaidAPI  schedulerConfirmReason = "paid_api"
)

func schedulerConfirmReasonText(r schedulerConfirmReason) string {
	switch r {
	case confirmReasonPaidAPI:
		return "只剩付費雲端 API 可用，需你確認後才會用 API 執行（可能產生費用）。"
	default:
		return "此排程風險等級較高，需你確認後才會執行。"
	}
}

// emitSchedulerConfirmNeeded 跳「app 內確認」卡，同時發 remote 通知（若有綁定）。
func (a *App) emitSchedulerConfirmNeeded(job scheduler.Job, spec schedulerBootstrapSpec, reason schedulerConfirmReason) {
	if a.eventBus != nil {
		a.eventBus.Emit("scheduler:confirm_needed", map[string]interface{}{
			"job_id":  job.ID,
			"title":   spec.Title,
			"summary": spec.Summary,
			"reason":  string(reason),
		})
	}
	a.dispatchSchedulerNotice(remote_bridge.NotifySystemWarning, "排程需確認："+spec.Title, schedulerConfirmReasonText(reason))
	debugtrace.Record("go.scheduler.autorun.confirm_needed", "", map[string]interface{}{"job_id": job.ID, "reason": string(reason)})
}
