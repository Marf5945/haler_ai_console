package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ui_console/data/storage"
	"ui_console/domain/execution_hook"
	"ui_console/domain/review"
	"ui_console/domain/risk"
	"ui_console/orchestration/dag"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/eventbus"
)

type taskReviewRef struct {
	RunID  string
	NodeID string
}

type taskNodeExecutor interface {
	ExecuteTaskNode(node dag.DAGNode, adapterID, sessionID string) (string, error)
}

type taskExecutorFunc func(node dag.DAGNode, adapterID, sessionID string) (string, error)

func (fn taskExecutorFunc) ExecuteTaskNode(node dag.DAGNode, adapterID, sessionID string) (string, error) {
	return fn(node, adapterID, sessionID)
}

// StartTaskProgress creates the backend-owned task run behind the user-facing progress UI.
func (a *App) StartTaskProgress(userText, adapterID, modelID, sessionID string) (*dag.DAGRun, error) {
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return nil, fmt.Errorf("task progress: empty input")
	}
	a.taskMu.Lock()
	if a.activeTaskRunID != "" {
		a.taskMu.Unlock()
		return nil, fmt.Errorf("已有任務正在執行")
	}
	a.taskMu.Unlock()

	rawPlan, normalized, repairCount, err := a.planTaskProgress(userText, adapterID, modelID, sessionID)
	if err != nil {
		return nil, err
	}
	rawStored, truncated := dag.TruncateRawPlan(rawPlan)
	now := time.Now().Format(time.RFC3339)
	run := &dag.DAGRun{
		ID:        fmt.Sprintf("dag-%d", time.Now().UnixNano()),
		Status:    "running",
		Title:     normalized.Plan.Title,
		Nodes:     dag.TaskPlanToNodes(normalized.Plan),
		CreatedAt: now,
		UpdatedAt: now,
		Planner: dag.PlannerMetadata{
			NormalizedPlan:        &normalized.Plan,
			RawModelPlan:          rawStored,
			RawModelPlanTruncated: truncated,
			RepairAttemptCount:    repairCount,
			PlannerAdapterID:      adapterID,
			PlannerModelID:        modelID,
			ValidationWarnings:    normalized.Warnings,
		},
	}
	hook, err := a.hookService.StartRun(run.ID, "outline-from-dag")
	if err == nil {
		run.HookRunID = hook.ID
		run.OutlineID = "outline-from-dag"
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	_ = dag.SaveFullRun(projectRoot, run)
	_ = dag.AppendRunIndex(projectRoot, dag.DAGRunSummary{
		RunID:     run.ID,
		Status:    run.Status,
		StartedAt: run.CreatedAt,
		NodeCount: len(run.Nodes),
	})

	a.taskMu.Lock()
	a.activeTaskRunID = run.ID
	a.taskMu.Unlock()
	a.emitTaskRun("task:progress_started", run)
	go a.runTaskProgress(run.ID, adapterID, sessionID)
	return run, nil
}

func (a *App) CancelActiveTaskProgress(reason string) (*dag.DAGRun, error) {
	a.taskMu.Lock()
	runID := a.activeTaskRunID
	a.taskMu.Unlock()
	if runID == "" {
		return nil, nil
	}
	return a.CancelTaskProgress(runID, reason)
}

func (a *App) CancelTaskProgress(runID, reason string) (*dag.DAGRun, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	run, err := dag.LoadFullRun(projectRoot, runID)
	if err != nil {
		return nil, err
	}
	now := time.Now().Format(time.RFC3339)
	for i := range run.Nodes {
		switch run.Nodes[i].Status {
		case dag.StatusRunning, dag.StatusWaitingReview, dag.StatusReady:
			run.Nodes[i].Status = dag.StatusCancelled
			run.Nodes[i].CompletedAt = now
		case dag.StatusPlanned:
			run.Nodes[i].Status = dag.StatusSkipped
		}
	}
	run.Status = "cancelled"
	run.UpdatedAt = now
	run.InterruptReason = taskInterruptReason(reason)
	_ = dag.SaveFullRun(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, now, 0, 0, run.InterruptReason)
	a.clearActiveTask(run.ID)
	a.appendTaskSystemMessage("任務已中斷：" + run.InterruptReason)
	a.emitTaskRun("task:progress_cancelled", run)
	return run, nil
}

func (a *App) ApproveTaskStep(reviewID string) (*dag.DAGRun, error) {
	a.taskMu.Lock()
	ref, ok := a.taskReviewIndex[reviewID]
	a.taskMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("review not linked to task: %s", reviewID)
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	run, err := dag.LoadFullRun(projectRoot, ref.RunID)
	if err != nil {
		return nil, err
	}
	now := time.Now().Format(time.RFC3339)
	for i := range run.Nodes {
		if run.Nodes[i].ID == ref.NodeID {
			run.Nodes[i].ApprovedBy = "local_user"
			run.Nodes[i].AppSessionID = a.globalSessionID
			run.Nodes[i].ApprovedAt = now
			run.Nodes[i].Status = dag.StatusReady
			break
		}
	}
	run.Status = "running"
	run.UpdatedAt = now
	_ = dag.SaveFullRun(projectRoot, run)
	_ = a.reviewService.Resolve(reviewID, projectRoot)
	a.eventBus.Emit(eventbus.EventReviewCardResolved, map[string]string{"review_id": reviewID})
	a.emitTaskRun("task:progress_updated", run)
	go a.runTaskProgress(run.ID, run.Planner.PlannerAdapterID, a.globalSessionID)
	return run, nil
}

func (a *App) GetDAGRunDebug(runID string) (interface{}, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	run, err := dag.LoadFullRun(projectRoot, runID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"notice": "開發工具，穩定後移除",
		"run":    run,
	}, nil
}

func (a *App) planTaskProgress(userText, adapterID, modelID, sessionID string) (string, dag.NormalizeResult, int, error) {
	prompt := buildTaskPlanPrompt(userText)
	traceID := fmt.Sprintf("task-plan-%d", time.Now().UnixNano())
	resp, err := a.callPlannerAdapter(adapterID, sessionID, prompt, traceID)
	if err != nil {
		return "", dag.NormalizeResult{}, 0, err
	}
	raw := plannerResponseText(resp)
	normalized, normErr := dag.NormalizePlan(raw)
	if normErr == nil {
		return raw, normalized, 0, nil
	}
	repairPrompt := prompt + "\n\n請只修正 JSON schema，不要改任務意圖。\n錯誤：" + normErr.Error() + "\n原始輸出：\n" + raw
	resp, err = a.callPlannerAdapter(adapterID, sessionID, repairPrompt, traceID+"-repair")
	if err != nil {
		return raw, dag.NormalizeResult{}, 1, err
	}
	repaired := plannerResponseText(resp)
	normalized, err = dag.NormalizePlan(repaired)
	if err != nil {
		return repaired, dag.NormalizeResult{}, 1, fmt.Errorf("planner JSON repair failed: %w", err)
	}
	return repaired, normalized, 1, nil
}

func (a *App) callPlannerAdapter(adapterID, sessionID, prompt, traceID string) (*skill_step.CLIResponse, error) {
	if a.isAPIOrLocalAdapter(adapterID) {
		return a.SendAPIMessage(adapterID, sessionID, prompt, traceID)
	}
	return a.SendCLIMessage(adapterID, sessionID, prompt, traceID)
}

func (a *App) isAPIOrLocalAdapter(adapterID string) bool {
	if strings.HasPrefix(adapterID, "llm-api-") || strings.HasPrefix(adapterID, "local-") {
		return true
	}
	if a.adapterRegistry != nil {
		if adapter, err := a.adapterRegistry.GetStatus(adapterID); err == nil {
			return adapter.Kind == "api" || adapter.Kind == "local"
		}
	}
	return false
}

func buildTaskPlanPrompt(userText string) string {
	return `請將使用者任務規劃成嚴格 JSON，不要輸出 Markdown。
schema:
{"title":"短標題","nodes":[{"id":"node_1","title":"中文步驟名","executor_type":"cli_task|tool_call|subagent_call","action_code":"snake_case","action":"使用者可讀動作","target":"目標","risk_class":"low|medium|high_non_destructive|user_owned_asset_destructive|subagent_lifecycle_removal|security_boundary_rewrite|critical_runtime_action","dependencies":[]}]}
規則：低依賴、單一路徑、第一版請偏向 sequential dependencies。使用者任務：` + userText
}

func plannerResponseText(resp *skill_step.CLIResponse) string {
	if resp == nil {
		return ""
	}
	if strings.TrimSpace(resp.Text) != "" {
		return resp.Text
	}
	if strings.TrimSpace(resp.Target) != "" {
		return resp.Target
	}
	return resp.Error
}

func (a *App) runTaskProgress(runID, adapterID, sessionID string) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	for {
		run, err := dag.LoadFullRun(projectRoot, runID)
		if err != nil || run.Status == "cancelled" || run.Status == "interrupted" {
			return
		}
		idx := nextReadyNodeIndex(run)
		if idx < 0 {
			a.finishTaskIfDone(projectRoot, run)
			return
		}
		node := &run.Nodes[idx]
		if requiresTaskReview(node) && node.ApprovedAt == "" {
			a.pauseForTaskReview(projectRoot, run, node)
			return
		}
		a.executeTaskNode(projectRoot, run, idx, adapterID, sessionID)
	}
}

func nextReadyNodeIndex(run *dag.DAGRun) int {
	done := map[string]bool{}
	for _, node := range run.Nodes {
		if node.Status == dag.StatusSucceeded || node.Status == dag.StatusSkipped {
			done[node.ID] = true
		}
	}
	for i, node := range run.Nodes {
		if node.Status != dag.StatusPlanned && node.Status != dag.StatusReady {
			continue
		}
		ok := true
		for _, dep := range node.Dependencies {
			if !done[dep] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func requiresTaskReview(node *dag.DAGNode) bool {
	if node.ActionCode == "mock_waiting_review" {
		return true
	}
	return risk.IsHigherOrEqual(risk.RiskClass(node.RiskClass), risk.HighNonDestructive)
}

func (a *App) pauseForTaskReview(projectRoot string, run *dag.DAGRun, node *dag.DAGNode) {
	card := a.reviewService.AddCard(review.CardParams{
		RiskClass:    risk.RiskClass(node.RiskClass),
		Operation:    node.Action,
		Target:       node.Target,
		Reason:       "這一步需要你確認後才會執行。",
		AcceptLabel:  "確認執行",
		RejectLabel:  "取消任務",
		AcceptEffect: "繼續目前任務步驟",
		RejectEffect: "取消整個任務",
		SourceType:   "task_progress",
		SourceID:     run.ID,
	})
	now := time.Now().Format(time.RFC3339)
	node.Status = dag.StatusWaitingReview
	node.ReviewID = card.ID
	run.Status = "waiting_review"
	run.ActiveNodeID = node.ID
	run.UpdatedAt = now
	a.taskMu.Lock()
	a.taskReviewIndex[card.ID] = taskReviewRef{RunID: run.ID, NodeID: node.ID}
	a.taskMu.Unlock()
	_ = dag.SaveFullRun(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, "", 0, 0, "")
	a.eventBus.Emit(eventbus.EventReviewCardAdded, card)
	a.emitTaskRun("task:progress_waiting_review", run)
}

func (a *App) executeTaskNode(projectRoot string, run *dag.DAGRun, idx int, adapterID, sessionID string) {
	node := &run.Nodes[idx]
	now := time.Now().Format(time.RFC3339)
	node.Status = dag.StatusRunning
	node.StartedAt = now
	run.ActiveNodeID = node.ID
	run.Status = "running"
	run.UpdatedAt = now
	_ = dag.SaveFullRun(projectRoot, run)
	a.emitTaskRun("task:progress_updated", run)

	result, err := a.executeTaskNodeAction(*node, adapterID, sessionID)
	end := time.Now().Format(time.RFC3339)
	node.CompletedAt = end
	node.ResultSummary = result
	node.TraceHash = fmt.Sprintf("trace-%d", time.Now().UnixNano())
	node.OutputRef = run.HookRunID
	if err != nil {
		node.Status = dag.StatusFailed
		node.Error = err.Error()
		run.Status = "failed"
	} else {
		node.Status = dag.StatusSucceeded
	}
	run.UpdatedAt = end
	_ = dag.SaveFullRun(projectRoot, run)
	_ = a.recordTaskTrace(run, node)
	if err != nil {
		_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, end, 0, 1, err.Error())
		a.clearActiveTask(run.ID)
		a.emitTaskRun("task:progress_failed", run)
	} else {
		a.emitTaskRun("task:progress_updated", run)
	}
}

func (a *App) executeTaskNodeAction(node dag.DAGNode, adapterID, sessionID string) (string, error) {
	// Executor dispatch is intentionally small so structured tool APIs can plug in later.
	executors := map[string]taskNodeExecutor{
		"tool_call":     taskExecutorFunc(a.executeToolTaskNode),
		"cli_task":      taskExecutorFunc(a.executeAdapterTaskNode),
		"subagent_call": taskExecutorFunc(a.executeAdapterTaskNode),
	}
	executor, ok := executors[node.ExecutorType]
	if !ok {
		return "", fmt.Errorf("unsupported_executor: %s", node.ExecutorType)
	}
	return executor.ExecuteTaskNode(node, adapterID, sessionID)
}

func (a *App) executeToolTaskNode(node dag.DAGNode, adapterID, sessionID string) (string, error) {
	switch node.ExecutorType {
	case "tool_call":
	default:
		return "", fmt.Errorf("unsupported_executor: %s", node.ExecutorType)
	}
	switch node.ActionCode {
	case "mock_success":
		return "mock_success completed", nil
	case "mock_fail":
		return "", fmt.Errorf("mock_fail")
	case "mock_waiting_review":
		return "mock_waiting_review approved", nil
	default:
		result := a.ActivateTool(node.Target)
		if !result.OK {
			return result.Message, errors.New(result.Message)
		}
		return result.Message, nil
	}
}

func (a *App) executeAdapterTaskNode(node dag.DAGNode, adapterID, sessionID string) (string, error) {
	prompt := strings.TrimSpace(node.Action + "\n" + node.Target)
	resp, err := a.callPlannerAdapter(adapterID, sessionID, prompt, fmt.Sprintf("task-node-%s", node.ID))
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}
	return strings.TrimSpace(resp.Text), nil
}

func (a *App) finishTaskIfDone(projectRoot string, run *dag.DAGRun) {
	hasFailed := false
	allTerminal := true
	for _, node := range run.Nodes {
		if node.Status == dag.StatusFailed {
			hasFailed = true
		}
		if !dag.IsTerminal(node.Status) {
			allTerminal = false
		}
	}
	if !allTerminal {
		return
	}
	now := time.Now().Format(time.RFC3339)
	if hasFailed {
		run.Status = "failed"
	} else {
		run.Status = "completed"
	}
	run.ActiveNodeID = ""
	run.UpdatedAt = now
	_ = dag.SaveFullRun(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, now, 0, 0, "")
	a.clearActiveTask(run.ID)
	a.emitTaskRun("task:progress_completed", run)
}

func (a *App) recordTaskTrace(run *dag.DAGRun, node *dag.DAGNode) error {
	if run.HookRunID == "" {
		return nil
	}
	status := executionStatus(node.Status)
	payload := execution_hook.StepTrace{
		StepID:        node.ID,
		OutlineStepID: run.OutlineID + "-" + node.ID,
		Action:        node.Action,
		Target:        node.Target,
		ToolUsed:      node.ExecutorType,
		ResultStatus:  status,
		RiskLevel:     execution_hook.RiskLevel(node.RiskClass),
	}
	if node.StartedAt != "" {
		payload.StartedAt, _ = time.Parse(time.RFC3339, node.StartedAt)
	}
	if node.CompletedAt != "" {
		payload.EndedAt, _ = time.Parse(time.RFC3339, node.CompletedAt)
	}
	return a.hookService.RecordTrace(run.HookRunID, payload)
}

func executionStatus(status dag.NodeStatus) execution_hook.StepResultStatus {
	switch status {
	case dag.StatusSucceeded:
		return execution_hook.StepResultOK
	case dag.StatusSkipped, dag.StatusCancelled:
		return execution_hook.StepResultSkipped
	default:
		return execution_hook.StepResultFailed
	}
}

func (a *App) emitTaskRun(eventName string, run *dag.DAGRun) {
	if a.eventBus != nil {
		a.eventBus.Emit(eventName, run)
	}
}

func (a *App) clearActiveTask(runID string) {
	a.taskMu.Lock()
	defer a.taskMu.Unlock()
	if a.activeTaskRunID == runID {
		a.activeTaskRunID = ""
	}
}

func (a *App) appendTaskSystemMessage(text string) {
	_ = a.AppendTalkEntryForAgent("main", "assistant", text)
	if a.eventBus != nil {
		a.eventBus.Emit("task:system_message", map[string]string{"text": text})
	}
}

func taskInterruptReason(reason string) string {
	switch reason {
	case "user_stop":
		return "你按下停止"
	case "app_restart":
		return "App 重新啟動"
	default:
		if strings.TrimSpace(reason) == "" {
			return "任務被中斷"
		}
		return reason
	}
}

func (a *App) interruptStaleTaskRuns(reason string) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	for _, run := range dag.ListFullRuns(projectRoot) {
		if run.Status != "running" && run.Status != "waiting_review" {
			continue
		}
		now := time.Now().Format(time.RFC3339)
		run.Status = "interrupted"
		run.InterruptReason = taskInterruptReason("app_restart")
		for i := range run.Nodes {
			if run.Nodes[i].Status == dag.StatusRunning || run.Nodes[i].Status == dag.StatusWaitingReview || run.Nodes[i].Status == dag.StatusReady {
				run.Nodes[i].Status = dag.StatusCancelled
				run.Nodes[i].CompletedAt = now
			}
			if run.Nodes[i].Status == dag.StatusPlanned {
				run.Nodes[i].Status = dag.StatusSkipped
			}
		}
		run.UpdatedAt = now
		_ = dag.SaveFullRun(projectRoot, run)
		_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, now, 0, 0, run.InterruptReason)
		_ = reason
	}
}
