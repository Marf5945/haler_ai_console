package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ui_console/builtin"
	"ui_console/data/storage"
	"ui_console/domain/execution_hook"
	"ui_console/domain/review"
	"ui_console/domain/risk"
	"ui_console/orchestration/dag"
	"ui_console/orchestration/delegation"
	"ui_console/orchestration/replan"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/controlseal"
	"ui_console/shared/eventbus"
)

const taskDependencyContextLimit = 48 * 1024

type taskReviewRef struct {
	RunID  string
	NodeID string
}

type taskNodeExecutor interface {
	ExecuteTaskNode(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error)
}

type taskExecutorFunc func(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error)

func (fn taskExecutorFunc) ExecuteTaskNode(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error) {
	return fn(run, node, adapterID, sessionID, traceID)
}

// StartTaskProgress creates the backend-owned task run behind the user-facing progress UI.
func (a *App) StartTaskProgress(userText, adapterID, modelID, sessionID string) (*dag.DAGRun, error) {
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return nil, fmt.Errorf("task progress: empty input")
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	now := time.Now().Format(time.RFC3339)
	runID := fmt.Sprintf("dag-%d", time.Now().UnixNano())
	planTraceID := fmt.Sprintf("task-plan-%s", runID)
	run := &dag.DAGRun{
		ID:            runID,
		Status:        "planning",
		Title:         userText,
		CreatedAt:     now,
		UpdatedAt:     now,
		ActiveTraceID: planTraceID,
		Planner: dag.PlannerMetadata{
			PlannerAdapterID: adapterID,
			PlannerModelID:   modelID,
		},
	}
	a.taskMu.Lock()
	if a.activeTaskRunID != "" {
		a.taskMu.Unlock()
		return nil, fmt.Errorf("已有任務正在執行")
	}
	a.activeTaskRunID = run.ID
	a.taskMu.Unlock()
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = dag.AppendRunIndex(projectRoot, dag.DAGRunSummary{
		RunID:     run.ID,
		Status:    run.Status,
		StartedAt: run.CreatedAt,
		NodeCount: 0,
	})
	a.emitTaskRun("task:progress_started", run)

	rawPlan, normalized, repairCount, err := a.planTaskProgress(userText, adapterID, modelID, sessionID, planTraceID)
	if err != nil {
		if latest, loadErr := dag.LoadFullRun(projectRoot, run.ID); loadErr == nil && latest.Status == "cancelled" {
			return latest, nil
		}
		message := taskPlanningErrorMessage(err)
		a.failPlanningTask(projectRoot, run, message)
		return nil, errors.New(message)
	}
	if latest, loadErr := dag.LoadFullRun(projectRoot, run.ID); loadErr == nil && latest.Status == "cancelled" {
		return latest, nil
	}
	rawStored, truncated := dag.TruncateRawPlan(rawPlan)
	now = time.Now().Format(time.RFC3339)
	run.Status = "running"
	run.Title = normalized.Plan.Title
	run.Nodes = dag.TaskPlanToNodes(normalized.Plan)
	// Bounded Replan：plan 階段建立並持久化目標契約，供日後 replan 同目標判定。
	if gc := dag.NewGoalContractFromPlan(userText, normalized.Plan); !gc.IsZero() {
		run.GoalContract = &gc
	}
	run.UpdatedAt = now
	run.ActiveTraceID = ""
	run.Planner = dag.PlannerMetadata{
		NormalizedPlan:        &normalized.Plan,
		RawModelPlan:          rawStored,
		RawModelPlanTruncated: truncated,
		RepairAttemptCount:    repairCount,
		PlannerAdapterID:      adapterID,
		PlannerModelID:        modelID,
		ValidationWarnings:    normalized.Warnings,
	}
	hook, err := a.hookService.StartRun(run.ID, "outline-from-dag")
	if err == nil {
		run.HookRunID = hook.ID
		run.OutlineID = "outline-from-dag"
	}
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, "", 0, len(run.Nodes), 0, "")

	a.emitTaskRun("task:progress_updated", run)
	go a.runTaskProgress(run.ID, adapterID, sessionID)
	return run, nil
}

func (a *App) failPlanningTask(projectRoot string, run *dag.DAGRun, message string) {
	if run == nil {
		return
	}
	now := time.Now().Format(time.RFC3339)
	run.Status = "failed"
	run.UpdatedAt = now
	run.InterruptReason = message
	run.ActiveTraceID = ""
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, now, 0, len(run.Nodes), 1, message)
	a.clearActiveTask(run.ID)
	a.emitTaskRun("task:progress_failed", run)
}

func taskPlanningErrorMessage(err error) string {
	var clarification taskPlannerClarificationError
	if errors.As(err, &clarification) {
		return clarification.Error()
	}
	raw := strings.TrimSpace(fmt.Sprint(err))
	raw = strings.TrimPrefix(raw, "planner unavailable: ")
	if raw == "" {
		return "任務規劃暫時無法完成，請稍後重試或切換模型。"
	}
	if containsAny(strings.ToLower(raw), []string{"容量不足", "限流", "resource_exhausted", "too many requests", "status 429", "no capacity available"}) {
		return "任務規劃暫時無法完成：" + raw
	}
	return "任務規劃失敗：" + raw
}

type taskPlannerClarificationError struct {
	Text string
}

func (e taskPlannerClarificationError) Error() string {
	text := strings.TrimSpace(e.Text)
	if text == "" {
		return "我需要更多資訊才能繼續，請補充你要處理的來源與輸出格式。"
	}
	return text
}

func addDebugTraceUserText(fields map[string]interface{}, traceID, text string) {
	if isTaskProgressTraceID(traceID) {
		fields["user_text_len"] = len([]rune(text))
		fields["user_text_preview"] = truncateRunes(text, 360)
		fields["user_text_compacted"] = len([]rune(text)) > len([]rune(fields["user_text_preview"].(string)))
		return
	}
	fields["user_text"] = text
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
	reviewIDs := []string{}
	for i := range run.Nodes {
		if strings.TrimSpace(run.Nodes[i].ReviewID) != "" {
			reviewIDs = append(reviewIDs, run.Nodes[i].ReviewID)
		}
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
	activeTraceID := run.ActiveTraceID
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, now, 0, len(run.Nodes), 0, run.InterruptReason)
	a.cancelTaskTrace(activeTraceID)
	a.resolveTaskReviewCards(projectRoot, reviewIDs)
	a.clearTaskReviewRefs(reviewIDs)
	a.clearActiveTask(run.ID)
	a.appendTaskSystemMessage("任務已中斷：" + run.InterruptReason)
	a.emitTaskRun("task:progress_cancelled", run)
	return run, nil
}

func (a *App) cancelTaskTrace(traceID string) {
	if strings.TrimSpace(traceID) == "" || a.sidecar == nil {
		return
	}
	// Best-effort: run state is already cancelled; this stops the active CLI child.
	_ = a.sidecar.CancelTrace(traceID)
}

func (a *App) resolveTaskReviewCards(projectRoot string, reviewIDs []string) {
	for _, reviewID := range reviewIDs {
		if strings.TrimSpace(reviewID) == "" || a.reviewService == nil {
			continue
		}
		_ = a.reviewService.Resolve(reviewID, projectRoot)
		a.eventBus.Emit(eventbus.EventReviewCardResolved, map[string]string{"review_id": reviewID})
	}
}

func (a *App) clearTaskReviewRefs(reviewIDs []string) {
	if len(reviewIDs) == 0 {
		return
	}
	a.taskMu.Lock()
	defer a.taskMu.Unlock()
	for _, reviewID := range reviewIDs {
		delete(a.taskReviewIndex, reviewID)
	}
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
	if run.Status != "waiting_review" {
		return nil, fmt.Errorf("task is not waiting for review: %s", run.Status)
	}
	now := time.Now().Format(time.RFC3339)
	found := false
	for i := range run.Nodes {
		if run.Nodes[i].ID == ref.NodeID {
			if run.Nodes[i].Status != dag.StatusWaitingReview || run.Nodes[i].ReviewID != reviewID {
				return nil, fmt.Errorf("task review is no longer pending: %s", reviewID)
			}
			run.Nodes[i].ApprovedBy = "local_user"
			run.Nodes[i].AppSessionID = a.globalSessionID
			run.Nodes[i].ApprovedAt = now
			run.Nodes[i].Status = dag.StatusReady
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("review node not found: %s", ref.NodeID)
	}
	run.Status = "running"
	run.UpdatedAt = now
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = a.reviewService.Resolve(reviewID, projectRoot)
	a.clearTaskReviewRefs([]string{reviewID})
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

func (a *App) planTaskProgress(userText, adapterID, modelID, sessionID, traceID string) (string, dag.NormalizeResult, int, error) {
	if intent, ok := a.classifyTaskSearchIntent(userText, adapterID, modelID, sessionID, traceID); ok {
		if intent.Question != "" {
			return "", dag.NormalizeResult{}, 0, taskPlannerClarificationError{Text: intent.Question}
		}
		if intent.Search {
			if raw, normalized, ok := buildDeterministicFileSearchPlanFromQuery(intent.Query); ok {
				return raw, normalized, 0, nil
			}
		}
	}
	if raw, normalized, ok := buildDeterministicFileSearchPlan(userText); ok {
		return raw, normalized, 0, nil
	}
	prompt := buildTaskPlanPrompt(userText)
	resp, err := a.callPlannerAdapter(adapterID, modelID, sessionID, prompt, traceID)
	if err != nil {
		return "", dag.NormalizeResult{}, 0, err
	}
	raw, normalized, normErr := normalizePlannerResponse(resp)
	if normErr == nil {
		if semErr := validateTaskPlanExecutionSemantics(userText, normalized.Plan); semErr == nil {
			return raw, normalized, 0, nil
		} else {
			normErr = semErr
		}
	}
	if isPlannerTransientFailure(resp) {
		return raw, dag.NormalizeResult{}, 0, fmt.Errorf("planner unavailable: %s", plannerTransientFailureMessage(resp))
	}
	if clarification := plannerClarificationText(resp); clarification != "" {
		return raw, dag.NormalizeResult{}, 0, taskPlannerClarificationError{Text: clarification}
	}
	repaired, repairedNormalized, err := a.repairTaskPlan(raw, normErr, adapterID, modelID, sessionID, traceID)
	if err != nil {
		return repaired, dag.NormalizeResult{}, 1, err
	}
	if semErr := validateTaskPlanExecutionSemantics(userText, repairedNormalized.Plan); semErr != nil {
		return repaired, dag.NormalizeResult{}, 1, fmt.Errorf("planner DAG repair failed: %w", semErr)
	}
	return repaired, repairedNormalized, 1, nil
}

type taskSearchIntent struct {
	Search   bool
	Query    string
	Question string
}

func (a *App) classifyTaskSearchIntent(userText, adapterID, modelID, sessionID, traceID string) (taskSearchIntent, bool) {
	prompt := buildTaskSearchIntentPrompt(userText)
	intentTraceID := strings.Replace(traceID, "task-plan-", "task-intent-", 1)
	resp, err := a.callPlannerAdapter(adapterID, modelID, sessionID, prompt, intentTraceID)
	if err != nil || resp == nil || isPlannerTransientFailure(resp) {
		return taskSearchIntent{}, false
	}
	if clarification := plannerClarificationText(resp); clarification != "" {
		return taskSearchIntent{Question: clarification}, true
	}
	return parseTaskSearchIntent(plannerResponseText(resp))
}

func buildTaskSearchIntentPrompt(userText string) string {
	return `判斷使用者是否要搜尋本機/已載入/引用的文件、資料、書、筆記或檔案。
只輸出一行: 是，搜尋關鍵詞
或: 否，
若資訊不足而無法判斷，輸出: 提問ㄌ要問使用者的必要問題ㄌ待命
不要回答是否找到，不要編搜尋結果，不要輸出 JSON。
例: 有看到甜點書嗎？ -> 是，甜點書
例: 幫我找測試用文件 -> 是，測試用文件
例: 幫我寫一封信 -> 否，
例: 幫我12-18寫到excel -> 否，
使用者輸入:` + sanitizeTaskPlannerUserText(userText)
}

func parseTaskSearchIntent(text string) (taskSearchIntent, bool) {
	text = strings.TrimSpace(dag.ExtractJSONPlan(text))
	if text == "" {
		return taskSearchIntent{}, false
	}
	var obj struct {
		Search bool   `json:"search"`
		Query  string `json:"query"`
	}
	if json.Unmarshal([]byte(text), &obj) == nil && obj.Query != "" {
		return taskSearchIntent{Search: obj.Search, Query: strings.TrimSpace(obj.Query)}, true
	}
	line := strings.TrimSpace(strings.Split(text, "\n")[0])
	lower := strings.ToLower(line)
	if strings.HasPrefix(line, "否") || strings.HasPrefix(lower, "no") || strings.HasPrefix(lower, "false") {
		return taskSearchIntent{Search: false}, true
	}
	if strings.HasPrefix(line, "是") || strings.HasPrefix(lower, "yes") || strings.HasPrefix(lower, "true") {
		query := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "是"), "yes"), "true"), " ，,:：-"))
		if query == "" {
			return taskSearchIntent{}, false
		}
		return taskSearchIntent{Search: true, Query: query}, true
	}
	return taskSearchIntent{}, false
}

func (a *App) repairTaskPlan(raw string, planErr error, adapterID, modelID, sessionID, traceID string) (string, dag.NormalizeResult, error) {
	repairPrompt := buildTaskPlanRepairPrompt(raw, planErr)
	resp, err := a.callPlannerAdapter(adapterID, modelID, sessionID, repairPrompt, strings.Replace(traceID, "task-plan-", "task-plan-repair-", 1))
	if err != nil {
		return raw, dag.NormalizeResult{}, err
	}
	if isPlannerTransientFailure(resp) {
		return plannerResponseText(resp), dag.NormalizeResult{}, fmt.Errorf("planner unavailable: %s", plannerTransientFailureMessage(resp))
	}
	if clarification := plannerClarificationText(resp); clarification != "" {
		return plannerResponseText(resp), dag.NormalizeResult{}, taskPlannerClarificationError{Text: clarification}
	}
	repaired, normalized, err := normalizePlannerResponse(resp)
	if err != nil {
		return repaired, dag.NormalizeResult{}, fmt.Errorf("planner JSON repair failed: %w", err)
	}
	return repaired, normalized, nil
}

func validateTaskPlanExecutionSemantics(userText string, plan dag.TaskPlan) error {
	if err := validateTaskPlanToolActions(userText, plan); err != nil {
		return err
	}
	if err := validateTaskPlanSearchStrategy(userText, plan); err != nil {
		return err
	}
	if err := validateTaskPlanDependencies(plan); err != nil {
		return err
	}
	if taskNeedsMultiStep(userText) && len(plan.Nodes) < 2 {
		return fmt.Errorf("task needs a multi-step DAG: create at least two nodes with dependencies")
	}
	if taskNeedsFinalAnswer(userText) && len(plan.Nodes) > 1 {
		last := plan.Nodes[len(plan.Nodes)-1]
		if last.ExecutorType != "cli_task" {
			return fmt.Errorf("final node should be cli_task so prior node result_summary becomes a user-readable answer")
		}
	}
	return nil
}

func validateTaskPlanSearchStrategy(userText string, plan dag.TaskPlan) error {
	if !taskNeedsFileSearch(userText) {
		return nil
	}
	hasUsefulSearch := false
	for _, node := range plan.Nodes {
		if node.ExecutorType != "tool_call" {
			continue
		}
		switch node.ActionCode {
		case "glob", "grep_search", "read_file":
			hasUsefulSearch = true
		}
	}
	if !hasUsefulSearch {
		return fmt.Errorf("file search task should use glob or grep_search/read_file before the final answer; list_directory alone is too shallow")
	}
	return nil
}

func validateTaskPlanToolActions(userText string, plan dag.TaskPlan) error {
	for _, node := range plan.Nodes {
		if node.ExecutorType != "tool_call" {
			continue
		}
		if isTaskProgressToolAction(node.ActionCode) {
			if strings.HasPrefix(node.ActionCode, "mock_") && !taskAllowsMockActions(userText) {
				return fmt.Errorf("node %s uses mock action_code %q for a non-test task; use real tool_manifest action or cli_task", node.ID, node.ActionCode)
			}
			continue
		}
		return fmt.Errorf("node %s unsupported tool_call action_code %q; use tool_manifest id or mock_*", node.ID, node.ActionCode)
	}
	return nil
}

func validateTaskPlanDependencies(plan dag.TaskPlan) error {
	if len(plan.Nodes) < 2 {
		return nil
	}
	for i := 1; i < len(plan.Nodes); i++ {
		n := plan.Nodes[i]
		// TASK 31 / Phase 0.3：放寬線性限制——允許明確標記的平行 root 無依賴，
		// 支援「天氣給 sub1、路徑給 sub2 同步跑」；其餘無依賴節點仍視為孤兒擋下。
		if len(n.Dependencies) == 0 && !n.ParallelRoot {
			return fmt.Errorf("node %s 無依賴且非 parallel_root（疑似孤兒節點）", n.ID)
		}
	}
	return nil
}

func isTaskProgressToolAction(actionCode string) bool {
	switch actionCode {
	case "mock_success", "mock_fail", "mock_waiting_review", "xlsx_write":
		return true
	}
	return isFSActionCode(actionCode)
}

func taskNeedsMultiStep(userText string) bool {
	text := strings.ToLower(strings.TrimSpace(userText))
	if text == "" || taskExplicitlyAsksSingleStep(text) {
		return false
	}
	if taskNeedsFileSearch(text) {
		return true
	}
	if strings.Contains(text, "找到後") || strings.Contains(text, "查到後") || strings.Contains(text, "讀完") {
		return true
	}
	acquire := containsAny(text, []string{"查詢", "搜尋", "搜索", "尋找", "找到", "找", "讀取", "閱讀", "掃描", "grep", "glob", "weather", "天氣", "文件", "檔案"})
	answer := containsAny(text, []string{"告訴", "回報", "回答", "摘要", "整理", "說明", "輸出", "列出", "總結", "summarize", "report"})
	sequence := containsAny(text, []string{"並", "然後", "再", "之後", "後再", "and then"})
	return (acquire && answer) || (acquire && sequence)
}

func taskNeedsFinalAnswer(userText string) bool {
	text := strings.ToLower(strings.TrimSpace(userText))
	if text == "" {
		return false
	}
	if taskNeedsFileSearch(text) {
		return true
	}
	return containsAny(text, []string{"告訴", "回報", "回答", "摘要", "整理", "說明", "輸出", "列出", "總結", "天氣", "summarize", "report"})
}

func taskNeedsFileSearch(userText string) bool {
	text := strings.ToLower(strings.TrimSpace(userText))
	if text == "" {
		return false
	}
	hasSearchVerb := containsAny(text, []string{"找", "尋找", "搜尋", "搜索", "查找", "查詢", "find", "search", "query"})
	hasDocTerm := containsAny(text, []string{"文件", "檔案", "檔", "文檔", "資料", "教材", "教學", "講義", "手冊", "reference", "document", "file", "pdf", "docx", ".md", ".txt"})
	return hasSearchVerb && hasDocTerm
}

func taskExplicitlyAsksSingleStep(text string) bool {
	return containsAny(text, []string{"只有一個步驟", "一個步驟", "單步", "單一節點", "一個 node", "one step", "single step"})
}

func taskAllowsMockActions(userText string) bool {
	text := strings.ToLower(strings.TrimSpace(userText))
	return containsAny(text, []string{"mock", "mock_", "模擬", "測試", "test"})
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func (a *App) callPlannerAdapter(adapterID, modelID, sessionID, prompt, traceID string) (*skill_step.CLIResponse, error) {
	if a.isAPIOrLocalAdapter(adapterID) {
		return a.SendAPIMessage(adapterID, sessionID, prompt, traceID)
	}
	return a.sendCLIMessage(adapterID, sessionID, prompt, traceID, modelID)
}

func isTaskProgressTraceID(traceID string) bool {
	return strings.HasPrefix(traceID, "task-plan-") || strings.HasPrefix(traceID, "task-plan-repair-") || strings.HasPrefix(traceID, "task-intent-") || strings.HasPrefix(traceID, "task-node-")
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

func buildDeterministicFileSearchPlan(userText string) (string, dag.NormalizeResult, bool) {
	if !taskNeedsFileSearch(userText) {
		return "", dag.NormalizeResult{}, false
	}
	return buildDeterministicFileSearchPlanFromQuery(userText)
}

func buildDeterministicFileSearchPlanFromQuery(query string) (string, dag.NormalizeResult, bool) {
	pattern := taskFileSearchPattern(query)
	if pattern == "" {
		return "", dag.NormalizeResult{}, false
	}
	plan := dag.TaskPlan{
		Title: "尋找本機文件",
		Nodes: []dag.TaskPlanNode{
			{
				ID:           "node_1",
				Title:        "搜尋本機文件內容",
				ExecutorType: "tool_call",
				ActionCode:   "grep_search",
				Action:       "搜尋本機文件內容",
				Target:       pattern,
				RiskClass:    "low",
				Dependencies: []string{},
			},
			{
				ID:           "node_2",
				Title:        "回覆搜尋結果",
				ExecutorType: "cli_task",
				ActionCode:   "answer",
				Action:       "根據搜尋結果回答是否有看到相關文件，並簡短列出可辨識的文件名稱或摘要。",
				Target:       "node_1.result_summary",
				RiskClass:    "low",
				Dependencies: []string{"node_1"},
			},
		},
	}
	raw, normalized, err := encodeAndNormalizeCanonicalPlan(plan)
	if err != nil {
		return "", dag.NormalizeResult{}, false
	}
	return raw, normalized, true
}

func taskFileSearchPattern(userText string) string {
	cleaned := stripReferenceSearchFiller(userText)
	cleaned = strings.NewReplacer(
		"有看到嗎", "", "看到嗎", "", "有看見嗎", "", "看見嗎", "",
		"有找到嗎", "", "找到嗎", "", "嗎", "",
	).Replace(cleaned)
	primary := compactReferenceQuery(cleaned)
	secondary := compactReferenceQuery(stripReferenceDocNouns(primary))
	candidates := []string{primary, secondary}
	for _, token := range []string{"測試用", "測試", "教學", "教程", "甜點", "書", "book", "test", "spec", "tutorial", "guide"} {
		if strings.Contains(strings.ToLower(userText), strings.ToLower(token)) {
			candidates = append(candidates, token)
		}
	}
	seen := map[string]bool{}
	var parts []string
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		parts = append(parts, regexp.QuoteMeta(candidate))
		if len(parts) >= 5 {
			break
		}
	}
	return strings.Join(parts, "|")
}

func buildTaskPlanPrompt(userText string) string {
	sanitizedTask := sanitizeTaskPlannerUserText(userText)
	return `輸出單一 JSON，禁止 Markdown。資訊不足改輸出: 提問ㄌ要問使用者的必要問題ㄌ待命
	格式: {"title":"短標題","nodes":[{"id":"node_1","type":"tool|answer","tool":"grep_search|glob|read_file|list_directory|xlsx_write","params":{"pattern":"","path":"","file_name":"輸出.xlsx","cells":[{"cell":"A1","value":"值","style":"標題"}],"rows":[["A欄"],["值"]],"format":"粗體置中黃底","styles":{"標題":{"bold":true,"fill_color":"FFF2CC","align":"center"}},"col_widths":{"A":16}},"depends_on":[]}]}
	規則: tool=grep_search/glob 用 pattern；read_file/list_directory 用 path；xlsx_write 是 app 內建 Excel 產生器，不需 shell，需 file_name 且 cells 或 rows；缺儲存格/資料/格式就提問；type=answer 依賴前步；找文件用 grep_search/glob；不允許 action-chain；不要輸出 executor_type/action_code/risk_class。
	使用者任務:` + sanitizedTask
}

func buildTaskPlanRepairPrompt(raw string, planErr error) string {
	errText := ""
	if planErr != nil {
		errText = strings.TrimSpace(planErr.Error())
	}
	return `修正計畫，只輸出合法 JSON，勿改任務意圖。若原始輸出是在補問必要資訊，改輸出: 提問ㄌ要問使用者的必要問題ㄌ待命
	格式: {"title":"短標題","nodes":[{"id":"node_1","type":"tool|answer","tool":"grep_search|glob|read_file|list_directory|xlsx_write","params":{"pattern":"關鍵字或glob","path":"相對路徑","file_name":"輸出.xlsx","cells":[{"cell":"A1","value":"文字或數字","style":"標題"}],"rows":[["A欄","B欄"],["值1","值2"]],"format":"格式需求文字","styles":{"標題":{"bold":true}},"col_widths":{"A":16}},"depends_on":[]}]}
	規則: tool=grep_search/glob 只填 params.pattern；read_file/list_directory 只填 params.path；xlsx_write 只填 file_name/cells/rows/format/styles/col_widths，至少 cells 或 rows 擇一；回答或整理結果用 type=answer 且 depends_on 前一步；禁止 Markdown。
	錯誤:` + errText + `
	原始輸出:
	` + truncateRunes(raw, 4096)
}

func sanitizeTaskPlannerUserText(userText string) string {
	return controlseal.SanitizeForLLM(controlseal.SourceUserRaw, userText).LLMText
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

func normalizePlannerResponse(resp *skill_step.CLIResponse) (string, dag.NormalizeResult, error) {
	candidates := plannerResponseCandidates(resp)
	var lastErr error
	for _, candidate := range candidates {
		normalized, err := dag.NormalizePlan(candidate)
		if err == nil {
			return candidate, normalized, nil
		}
		repairedCandidate, repairedNormalized, repairErr := normalizePlannerResponseLocally(candidate)
		if repairErr == nil {
			return repairedCandidate, repairedNormalized, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return plannerResponseText(resp), dag.NormalizeResult{}, lastErr
	}
	return "", dag.NormalizeResult{}, fmt.Errorf("planner response is empty")
}

func normalizePlannerResponseLocally(raw string) (string, dag.NormalizeResult, error) {
	var plan struct {
		Title string                   `json:"title"`
		Nodes []map[string]interface{} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(dag.ExtractJSONPlan(raw)), &plan); err != nil {
		return "", dag.NormalizeResult{}, err
	}
	if strings.TrimSpace(plan.Title) == "" || len(plan.Nodes) == 0 {
		return "", dag.NormalizeResult{}, fmt.Errorf("local planner normalize: empty plan")
	}
	if looksLikeSimplePlannerPlan(plan.Nodes) {
		return canonicalizeSimplePlannerPlan(plan.Title, plan.Nodes)
	}
	return canonicalizeLegacyPlannerPlan(plan.Title, plan.Nodes)
}

func looksLikeSimplePlannerPlan(nodes []map[string]interface{}) bool {
	for _, node := range nodes {
		if _, ok := node["type"]; ok {
			return true
		}
		if _, ok := node["tool"]; ok {
			return true
		}
		if _, ok := node["params"]; ok {
			return true
		}
		if _, ok := node["depends_on"]; ok {
			return true
		}
	}
	return false
}

func canonicalizeSimplePlannerPlan(title string, nodes []map[string]interface{}) (string, dag.NormalizeResult, error) {
	out := dag.TaskPlan{Title: title, Nodes: make([]dag.TaskPlanNode, 0, len(nodes))}
	for i, rawNode := range nodes {
		if err := rejectUnknownPlannerNodeKeys(rawNode, map[string]bool{
			"id": true, "title": true, "type": true, "tool": true, "params": true, "depends_on": true,
		}); err != nil {
			return "", dag.NormalizeResult{}, err
		}
		nodeType := strings.TrimSpace(plannerNodeString(rawNode, "type"))
		if nodeType == "" {
			nodeType = "tool"
		}
		id := plannerNodeString(rawNode, "id")
		if id == "" {
			id = fmt.Sprintf("node_%d", i+1)
		}
		deps := plannerNodeStringSlice(rawNode, "depends_on")
		switch nodeType {
		case "tool":
			tool := plannerNodeString(rawNode, "tool")
			action, target, err := canonicalToolActionAndTarget(tool, rawNode["params"])
			if err != nil {
				return "", dag.NormalizeResult{}, fmt.Errorf("node %s: %w", id, err)
			}
			riskClass := "low"
			if tool == "xlsx_write" {
				riskClass = "medium"
			}
			out.Nodes = append(out.Nodes, dag.TaskPlanNode{
				ID:           id,
				Title:        defaultPlannerNodeTitle(rawNode, action),
				ExecutorType: "tool_call",
				ActionCode:   tool,
				Action:       action,
				Target:       target,
				RiskClass:    riskClass,
				Dependencies: deps,
			})
		case "answer":
			out.Nodes = append(out.Nodes, dag.TaskPlanNode{
				ID:           id,
				Title:        defaultPlannerNodeTitle(rawNode, "整理並回覆結果"),
				ExecutorType: "cli_task",
				ActionCode:   "answer",
				Action:       "根據前置步驟結果整理成使用者可讀答案",
				Target:       taskPlannerDefaultTarget(deps),
				RiskClass:    "low",
				Dependencies: deps,
			})
		default:
			return "", dag.NormalizeResult{}, fmt.Errorf("node %s unsupported type %q", id, nodeType)
		}
	}
	return encodeAndNormalizeCanonicalPlan(out)
}

func canonicalizeLegacyPlannerPlan(title string, nodes []map[string]interface{}) (string, dag.NormalizeResult, error) {
	out := dag.TaskPlan{Title: title, Nodes: make([]dag.TaskPlanNode, 0, len(nodes))}
	for i, rawNode := range nodes {
		node := dag.TaskPlanNode{
			ID:           plannerNodeString(rawNode, "id"),
			Title:        plannerNodeString(rawNode, "title"),
			ExecutorType: plannerNodeString(rawNode, "executor_type"),
			ActionCode:   plannerNodeString(rawNode, "action_code"),
			Action:       plannerNodeString(rawNode, "action"),
			Target:       plannerNodeString(rawNode, "target"),
			RiskClass:    plannerNodeString(rawNode, "risk_class"),
			Dependencies: plannerNodeStringSlice(rawNode, "dependencies"),
		}
		if node.ID == "" {
			node.ID = fmt.Sprintf("node_%d", i+1)
		}
		if node.ExecutorType == "tool_call" {
			var err error
			node, err = canonicalizeLegacyToolTaskNode(node, rawNode)
			if err != nil {
				return "", dag.NormalizeResult{}, err
			}
		}
		if strings.TrimSpace(node.Target) == "" && (node.ExecutorType == "cli_task" || node.ExecutorType == "subagent_call") {
			node.Target = taskPlannerDefaultTarget(node.Dependencies)
		}
		out.Nodes = append(out.Nodes, node)
	}
	return encodeAndNormalizeCanonicalPlan(out)
}

func encodeAndNormalizeCanonicalPlan(out dag.TaskPlan) (string, dag.NormalizeResult, error) {
	encoded, err := json.Marshal(out)
	if err != nil {
		return "", dag.NormalizeResult{}, err
	}
	normalized, err := dag.NormalizePlan(string(encoded))
	if err != nil {
		return "", dag.NormalizeResult{}, err
	}
	return string(encoded), normalized, nil
}

func canonicalizeLegacyToolTaskNode(node dag.TaskPlanNode, raw map[string]interface{}) (dag.TaskPlanNode, error) {
	switch node.ActionCode {
	case "grep_search":
		if pattern, err := plannerActionPattern(raw["action"], true); err != nil {
			return node, err
		} else if pattern != "" && (node.Target == "" || node.Target == ".") {
			node.Target = pattern
		}
		if strings.TrimSpace(node.Action) == "" || strings.HasPrefix(strings.TrimSpace(node.Action), "{") {
			node.Action = "搜尋本地文件內容"
		}
	case "glob":
		if pattern, err := plannerActionPattern(raw["action"], true); err != nil {
			return node, err
		} else if pattern != "" && node.Target == "" {
			node.Target = pattern
		}
		if strings.TrimSpace(node.Action) == "" || strings.HasPrefix(strings.TrimSpace(node.Action), "{") {
			node.Action = "搜尋本地檔名"
		}
	}
	return node, nil
}

func canonicalToolActionAndTarget(tool string, params interface{}) (string, string, error) {
	switch tool {
	case "grep_search":
		pattern, err := plannerParamString(params, "pattern", []string{"pattern"})
		return "搜尋本地文件內容", pattern, err
	case "glob":
		pattern, err := plannerParamString(params, "pattern", []string{"pattern"})
		return "搜尋本地檔名", pattern, err
	case "read_file":
		path, err := plannerParamString(params, "path", []string{"path"})
		return "讀取本地文件", path, err
	case "list_directory":
		path, err := plannerParamString(params, "path", []string{"path"})
		return "列出本地目錄", path, err
	case "xlsx_write":
		target, err := plannerXlsxTarget(params)
		return "寫入 Excel 檔案", target, err
	default:
		return "", "", fmt.Errorf("unsupported tool %q", tool)
	}
}

func plannerXlsxTarget(params interface{}) (string, error) {
	m, ok := params.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("params must be an object")
	}
	allowed := map[string]bool{
		"file_name":  true,
		"cells":      true,
		"rows":       true,
		"format":     true,
		"sheet":      true,
		"styles":     true,
		"col_widths": true,
	}
	for k := range m {
		if !allowed[k] {
			return "", fmt.Errorf("unsupported params.%s", k)
		}
	}
	fileName := strings.TrimSpace(fmt.Sprint(m["file_name"]))
	if fileName == "" || fileName == "<nil>" {
		return "", fmt.Errorf("missing params.file_name")
	}
	if _, hasCells := m["cells"]; !hasCells {
		if _, hasRows := m["rows"]; !hasRows {
			return "", fmt.Errorf("xlsx_write requires params.cells or params.rows")
		}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("xlsx_write params marshal: %w", err)
	}
	return string(data), nil
}

func plannerParamString(params interface{}, key string, allowed []string) (string, error) {
	m, ok := params.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("params must be an object")
	}
	allowedSet := map[string]bool{}
	for _, k := range allowed {
		allowedSet[k] = true
	}
	for k := range m {
		if !allowedSet[k] {
			return "", fmt.Errorf("unsupported params.%s", k)
		}
	}
	value := strings.TrimSpace(fmt.Sprint(m[key]))
	if value == "" || value == "<nil>" {
		return "", fmt.Errorf("missing params.%s", key)
	}
	return value, nil
}

func rejectUnknownPlannerNodeKeys(raw map[string]interface{}, allowed map[string]bool) error {
	for key := range raw {
		if !allowed[key] {
			return fmt.Errorf("unsupported node field %q", key)
		}
	}
	return nil
}

func defaultPlannerNodeTitle(raw map[string]interface{}, fallback string) string {
	title := plannerNodeString(raw, "title")
	if title != "" {
		return title
	}
	return fallback
}

func plannerNodeString(raw map[string]interface{}, key string) string {
	value, ok := raw[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
}

func plannerNodeStringSlice(raw map[string]interface{}, key string) []string {
	value, ok := raw[key]
	if !ok || value == nil {
		return nil
	}
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func plannerActionPattern(action interface{}, allowIncludePattern bool) (string, error) {
	switch v := action.(type) {
	case map[string]interface{}:
		allowed := []string{"pattern"}
		if allowIncludePattern {
			allowed = append(allowed, "include_pattern")
		}
		return plannerParamString(v, "pattern", allowed)
	case string:
		var parsed map[string]interface{}
		if json.Unmarshal([]byte(v), &parsed) == nil {
			return plannerActionPattern(parsed, allowIncludePattern)
		}
	}
	return "", nil
}

func taskPlannerDefaultTarget(dependencies []string) string {
	if len(dependencies) == 0 {
		return "使用者任務"
	}
	return strings.Join(dependencies, ",") + ".result_summary"
}

func plannerResponseCandidates(resp *skill_step.CLIResponse) []string {
	if resp == nil {
		return nil
	}
	fields := []string{resp.Text, resp.Target, resp.Next, resp.Error}
	candidates := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" || seen[field] {
			continue
		}
		seen[field] = true
		candidates = append(candidates, field)
	}
	return candidates
}

func isPlannerTransientFailure(resp *skill_step.CLIResponse) bool {
	if resp == nil {
		return false
	}
	combined := strings.Join([]string{resp.Text, resp.Target, resp.Next, resp.Error}, "\n")
	combined = strings.ToLower(combined)
	return containsAny(combined, []string{
		"model_capacity_exhausted",
		"no capacity available for model",
		"resource_exhausted",
		"ratelimitexceeded",
		"too many requests",
		"status 429",
		"伺服器容量不足",
		"容量不足",
		"限流",
	})
}

func plannerTransientFailureMessage(resp *skill_step.CLIResponse) string {
	text := strings.TrimSpace(plannerResponseText(resp))
	if text == "" {
		return "模型服務暫時不可用，請稍後重試或切換模型。"
	}
	if len([]rune(text)) > 240 {
		text = truncateRunes(text, 240) + "..."
	}
	return text
}

func plannerClarificationText(resp *skill_step.CLIResponse) string {
	if question := plannerActionQuestionText(resp); question != "" {
		return question
	}
	text := strings.TrimSpace(plannerResponseText(resp))
	if text == "" || !looksLikePlannerClarification(text) {
		return ""
	}
	if len([]rune(text)) > 1600 {
		return truncateRunes(text, 1600) + "..."
	}
	return text
}

func plannerActionQuestionText(resp *skill_step.CLIResponse) string {
	for _, candidate := range plannerResponseCandidates(resp) {
		if question := actionChainQuestionText(candidate); question != "" {
			return question
		}
	}
	return ""
}

func actionChainQuestionText(text string) string {
	chain, err := actionchain.Parse(strings.TrimSpace(text))
	if err != nil {
		return ""
	}
	switch chain.Action {
	case "提問", "澄清":
	default:
		return ""
	}
	question := strings.TrimSpace(chain.Target)
	if question == "" {
		return ""
	}
	if len([]rune(question)) > 1600 {
		return truncateRunes(question, 1600) + "..."
	}
	return question
}

func looksLikePlannerClarification(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	hasQuestion := strings.Contains(trimmed, "?") || strings.Contains(trimmed, "？")
	hasAsk := containsAny(trimmed, []string{
		"請提供", "請告訴", "請確認", "我需要知道", "需要知道", "以下幾點",
		"哪個檔案", "哪個文件", "什麼格式", "什麼內容", "更多資訊",
	}) || containsAny(lower, []string{
		"please provide", "please clarify", "need to know", "which file",
		"what format", "more information", "additional information",
	})
	hasBlocker := containsAny(trimmed, []string{
		"無法", "不能", "資訊不足", "資料不足", "不清楚", "太模糊", "缺少",
		"沒有更多資訊", "沒有足夠資訊",
	}) || containsAny(lower, []string{
		"cannot", "can't", "unable", "ambiguous", "lack", "not enough information",
	})
	return (hasAsk && hasBlocker) || (hasQuestion && hasAsk)
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
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, "", 0, len(run.Nodes), 0, "")
	a.eventBus.Emit(eventbus.EventReviewCardAdded, card)
	a.emitTaskRun("task:progress_waiting_review", run)
}

func (a *App) executeTaskNode(projectRoot string, run *dag.DAGRun, idx int, adapterID, sessionID string) {
	node := &run.Nodes[idx]
	now := time.Now().Format(time.RFC3339)
	node.Status = dag.StatusRunning
	node.StartedAt = now
	traceID := fmt.Sprintf("task-node-%s", node.ID)
	run.ActiveNodeID = node.ID
	run.ActiveTraceID = traceID
	run.Status = "running"
	run.UpdatedAt = now
	_ = dag.SaveFullRunLocked(projectRoot, run)
	a.emitTaskRun("task:progress_updated", run)

	result, err := a.executeTaskNodeAction(run, *node, adapterID, sessionID, traceID)
	if latest, loadErr := dag.LoadFullRun(projectRoot, run.ID); loadErr == nil {
		if latest.Status == "cancelled" || latest.Status == "interrupted" {
			return
		}
	}
	end := time.Now().Format(time.RFC3339)
	node.CompletedAt = end
	node.ResultSummary = result
	node.TraceHash = fmt.Sprintf("trace-%d", time.Now().UnixNano())
	node.OutputRef = run.HookRunID
	run.ActiveTraceID = ""
	if err != nil {
		// Bounded Replan：先轉結構化 FailureCategory，再嘗試 low-risk 自動換路（flag 預設關）。
		node.FailureCategory = string(replan.ClassifyFailure(node.Action, result, err))
		if a.tryReplanOnFailure(projectRoot, run, node, adapterID, sessionID, result, err) {
			return // 已 silent re-route；runTaskProgress 會 reload 接上新 tail
		}
		node.Status = dag.StatusFailed
		node.Error = err.Error()
		run.Status = "failed"
	} else {
		node.Status = dag.StatusSucceeded
		if replanEnabled() {
			// Bounded Replan 軟失敗：節點「成功」但結果是失敗訊號（找不到/不存在…）→ 嘗試換路。
			if cat := replan.ClassifyResult(result); cat != "" {
				node.FailureCategory = string(cat)
				if a.tryReplanOnFailure(projectRoot, run, node, adapterID, sessionID, result, fmt.Errorf("soft failure: %s", cat)) {
					return // 已 silent re-route；runTaskProgress 會 reload 接上新 tail
				}
			} else {
				// 真正成功且有摘要 → 連續無進展計數歸零。
				replanCounterFor(run.ID).RecordProgress(*node)
			}
		}
	}
	run.UpdatedAt = end
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = a.recordTaskTrace(run, node)
	if err != nil {
		_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, end, 0, len(run.Nodes), 1, err.Error())
		a.appendTaskResultMessage(run, true)
		a.clearActiveTask(run.ID)
		a.emitTaskRun("task:progress_failed", run)
	} else {
		a.emitTaskRun("task:progress_updated", run)
	}
}

func (a *App) executeTaskNodeAction(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error) {
	// Executor dispatch is intentionally small so structured tool APIs can plug in later.
	executors := map[string]taskNodeExecutor{
		"tool_call":     taskExecutorFunc(a.executeToolTaskNode),
		"cli_task":      taskExecutorFunc(a.executeAdapterTaskNode),
		"subagent_call": taskExecutorFunc(a.executeSubagentTaskNode),
	}
	executor, ok := executors[node.ExecutorType]
	if !ok {
		return "", fmt.Errorf("unsupported_executor: %s", node.ExecutorType)
	}
	return executor.ExecuteTaskNode(run, node, adapterID, sessionID, traceID)
}

func (a *App) executeToolTaskNode(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error) {
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
	case "xlsx_write":
		return a.executeXlsxWriteTaskNode(node)
	default:
		// Phase A（2026-05-27）：dispatch read-only FS action 給 task_progress_fs.go。
		// 路徑邊界與大小限制由 dispatchFSAction 內部處理；任何違規回傳 error。
		if isFSActionCode(node.ActionCode) {
			result, err := a.dispatchFSAction(node.ActionCode, node.Target)
			if err != nil {
				return "", err
			}
			return result, nil
		}
		// 未知 action_code 才走 legacy tool registry
		result := a.ActivateTool(node.Target)
		if !result.OK {
			return result.Message, errors.New(result.Message)
		}
		return result.Message, nil
	}
}

type xlsxWriteTarget struct {
	FileName  string                       `json:"file_name"`
	Sheet     string                       `json:"sheet,omitempty"`
	Cells     []xlsxCellTarget             `json:"cells,omitempty"`
	Rows      [][]interface{}              `json:"rows,omitempty"`
	Format    string                       `json:"format,omitempty"`
	Styles    map[string]builtin.XlsxStyle `json:"styles,omitempty"`
	ColWidths map[string]float64           `json:"col_widths,omitempty"`
}

type xlsxCellTarget struct {
	Cell  string      `json:"cell"`
	Value interface{} `json:"value"`
	Style string      `json:"style,omitempty"`
}

func (a *App) executeXlsxWriteTaskNode(node dag.DAGNode) (string, error) {
	var target xlsxWriteTarget
	decoder := json.NewDecoder(strings.NewReader(node.Target))
	decoder.UseNumber()
	if err := decoder.Decode(&target); err != nil {
		return "", fmt.Errorf("xlsx_write: target must be JSON params: %w", err)
	}
	target.FileName = sanitizeTaskOutputFileName(target.FileName, ".xlsx")
	if target.FileName == "" {
		return "", fmt.Errorf("xlsx_write: file_name is required")
	}
	grid, err := buildXlsxGrid(target)
	if err != nil {
		return "", err
	}
	spec, err := buildStyledXlsxSpec(target)
	if err != nil {
		return "", err
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	outputDir := filepath.Join(projectRoot, "outputs")
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return "", fmt.Errorf("xlsx_write: mkdir outputs: %w", err)
	}
	outputPath := filepath.Join(outputDir, target.FileName)
	if err := builtin.GenerateStyledXlsx(spec, outputPath); err != nil {
		return "", fmt.Errorf("xlsx_write: generate xlsx: %w", err)
	}
	result := map[string]interface{}{
		"tool":        "xlsx_write",
		"file_name":   target.FileName,
		"path":        outputPath,
		"rows":        len(grid),
		"columns":     maxGridColumns(grid),
		"format":      strings.TrimSpace(target.Format),
		"styles":      len(spec.Styles),
		"format_note": "已套用 xlsx_write 支援的基本樣式：粗體、字色、底色、對齊與欄寬。",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data), nil
}

func buildStyledXlsxSpec(target xlsxWriteTarget) (builtin.XlsxSpec, error) {
	styles := map[string]builtin.XlsxStyle{}
	for name, style := range target.Styles {
		if strings.TrimSpace(name) != "" {
			styles[name] = style
		}
	}

	globalStyle := ""
	if inferred, ok := inferXlsxStyleFromFormat(target.Format); ok {
		globalStyle = "format"
		styles[globalStyle] = inferred
	}

	spec := builtin.XlsxSpec{
		SheetName: target.Sheet,
		Styles:    styles,
		ColWidths: target.ColWidths,
	}
	if len(target.Rows) > 0 {
		spec.Rows = make([][]builtin.XlsxCell, 0, len(target.Rows))
		for _, row := range target.Rows {
			out := make([]builtin.XlsxCell, 0, len(row))
			for _, value := range row {
				out = append(out, builtin.XlsxCell{Value: xlsxCellValueString(value), Style: globalStyle})
			}
			spec.Rows = append(spec.Rows, out)
		}
	}
	if len(target.Cells) > 0 {
		spec.Cells = map[string]builtin.XlsxCell{}
		for _, cell := range target.Cells {
			style := strings.TrimSpace(cell.Style)
			if style == "" {
				style = globalStyle
			}
			spec.Cells[cell.Cell] = builtin.XlsxCell{Value: xlsxCellValueString(cell.Value), Style: style}
		}
	}
	if len(spec.Rows) == 0 && len(spec.Cells) == 0 {
		return builtin.XlsxSpec{}, fmt.Errorf("xlsx_write: rows or cells are required")
	}
	return spec, nil
}

func inferXlsxStyleFromFormat(format string) (builtin.XlsxStyle, bool) {
	text := strings.ToLower(strings.TrimSpace(format))
	if text == "" {
		return builtin.XlsxStyle{}, false
	}
	style := builtin.XlsxStyle{}
	if strings.Contains(text, "粗體") || strings.Contains(text, "bold") {
		style.Bold = true
	}
	if strings.Contains(text, "置中") || strings.Contains(text, "居中") || strings.Contains(text, "center") {
		style.Align = "center"
	} else if strings.Contains(text, "靠右") || strings.Contains(text, "right") {
		style.Align = "right"
	}
	if color := firstHexColor(text); color != "" {
		if strings.Contains(text, "字") || strings.Contains(text, "font") {
			style.FontColor = color
		} else {
			style.FillColor = color
		}
	}
	if strings.Contains(text, "黃底") || strings.Contains(text, "黃色底") || strings.Contains(text, "yellow") {
		style.FillColor = "FFF2CC"
	}
	if strings.Contains(text, "藍底") || strings.Contains(text, "blue fill") {
		style.FillColor = "D9EAF7"
	}
	if strings.Contains(text, "紅字") || strings.Contains(text, "red font") {
		style.FontColor = "C00000"
	}
	if strings.Contains(text, "藍字") || strings.Contains(text, "blue font") {
		style.FontColor = "1F4E79"
	}
	if style == (builtin.XlsxStyle{}) {
		return builtin.XlsxStyle{}, false
	}
	return style, true
}

func firstHexColor(text string) string {
	match := regexp.MustCompile(`#?[0-9a-fA-F]{6}`).FindString(text)
	if match == "" {
		return ""
	}
	return strings.TrimPrefix(match, "#")
}

func buildXlsxGrid(target xlsxWriteTarget) ([][]string, error) {
	var grid [][]string
	for _, row := range target.Rows {
		out := make([]string, 0, len(row))
		for _, value := range row {
			out = append(out, xlsxCellValueString(value))
		}
		grid = append(grid, out)
	}
	for _, cell := range target.Cells {
		row, col, err := parseA1CellRef(cell.Cell)
		if err != nil {
			return nil, err
		}
		for len(grid) <= row {
			grid = append(grid, nil)
		}
		for len(grid[row]) <= col {
			grid[row] = append(grid[row], "")
		}
		grid[row][col] = xlsxCellValueString(cell.Value)
	}
	if len(grid) == 0 {
		return nil, fmt.Errorf("xlsx_write: rows or cells are required")
	}
	return grid, nil
}

var xlsxA1CellPattern = regexp.MustCompile(`^([A-Za-z]+)([1-9][0-9]*)$`)

func parseA1CellRef(ref string) (int, int, error) {
	ref = strings.TrimSpace(ref)
	match := xlsxA1CellPattern.FindStringSubmatch(ref)
	if match == nil {
		return 0, 0, fmt.Errorf("xlsx_write: invalid cell reference %q", ref)
	}
	row, err := strconv.Atoi(match[2])
	if err != nil || row <= 0 {
		return 0, 0, fmt.Errorf("xlsx_write: invalid row in cell %q", ref)
	}
	col := 0
	for _, r := range strings.ToUpper(match[1]) {
		col = col*26 + int(r-'A'+1)
	}
	if col <= 0 {
		return 0, 0, fmt.Errorf("xlsx_write: invalid column in cell %q", ref)
	}
	return row - 1, col - 1, nil
}

func xlsxCellValueString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func gridToTSV(grid [][]string) string {
	lines := make([]string, 0, len(grid))
	for _, row := range grid {
		lines = append(lines, strings.Join(row, "\t"))
	}
	return strings.Join(lines, "\n")
}

func maxGridColumns(grid [][]string) int {
	max := 0
	for _, row := range grid {
		if len(row) > max {
			max = len(row)
		}
	}
	return max
}

func sanitizeTaskOutputFileName(name, ext string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	)
	name = strings.TrimSpace(replacer.Replace(name))
	if name == "" {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(name), ext) {
		name = strings.TrimSuffix(name, filepath.Ext(name)) + ext
	}
	return name
}

func (a *App) executeAdapterTaskNode(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error) {
	prompt := buildTaskNodePrompt(run, node)
	resp, err := a.callPlannerAdapter(adapterID, run.Planner.PlannerModelID, sessionID, prompt, traceID)
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}
	// SEC-C 補洞（2026-05-28）：CLI 回的內容會成為下一步的 result_summary，
	// 進入下游 prompt 前必須 sanitize，防止 prompt injection 跨節點傳遞。
	return controlseal.SanitizeForLLM(controlseal.SourceCLIOutput, strings.TrimSpace(resp.Text)).LLMText, nil
}

func (a *App) executeSubagentTaskNode(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	subID, err := resolveTaskSubagentID(projectRoot, node.Target)
	if err != nil {
		return "", err
	}
	prompt := buildTaskNodePrompt(run, node)
	_ = a.AppendTalkEntryForAgent(subID, "user", prompt)
	subSessionID := strings.TrimSpace(sessionID)
	if subSessionID == "" {
		subSessionID = run.ID
	}
	subSessionID = "subagent:" + subID + ":" + subSessionID
	resp, err := a.callPlannerAdapter(adapterID, run.Planner.PlannerModelID, subSessionID, prompt, traceID)
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}
	result := controlseal.SanitizeForLLM(controlseal.SourceCLIOutput, strings.TrimSpace(resp.Text)).LLMText
	_ = a.AppendTalkEntryForAgent(subID, "assistant", result)
	_ = appendSubagentDelegationRecord(projectRoot, delegation.ActionRecord{
		Type:      "sub_delegated",
		ToolID:    subID,
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if result == "" {
		return "subagent " + subID + " completed", nil
	}
	return "subagent " + subID + " completed:\n" + result, nil
}

func resolveTaskSubagentID(projectRoot, target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("subagent_call requires node.target to be a subagent id or label")
	}
	for _, sub := range listSubagentTabs(projectRoot) {
		if target == sub.id || target == sub.label {
			return sub.id, nil
		}
	}
	return "", fmt.Errorf("subagent not found for task target: %s", target)
}

func appendSubagentDelegationRecord(projectRoot string, record delegation.ActionRecord) error {
	if record.Timestamp == "" {
		record.Timestamp = time.Now().Format(time.RFC3339)
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	path := filepath.Join(projectRoot, "subagents", "delegation_actions.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func buildTaskNodePrompt(run *dag.DAGRun, node dag.DAGNode) string {
	base := strings.TrimSpace(node.Action + "\n" + node.Target)
	depContext := buildTaskDependencyContext(run, node)
	var assembled string
	if depContext == "" {
		assembled = base
	} else {
		assembled = strings.TrimSpace(base + "\n\n前置步驟結果（供本步驟使用）：\n" + depContext)
	}
	// cli_task 走對話口吻；摘要/回覆類步驟要直接收尾，不硬問下一步。
	if node.ExecutorType == "cli_task" {
		instruction := buildCLITaskNodeInstruction()
		// Bounded Replan（flag 開時）：引導「找不到」短答，配合 ClassifyResult 長度守門。
		if replanEnabled() {
			instruction += " " + replanShortFailureHint()
		}
		assembled = instruction + "\n\n" + assembled
	}
	// SEC-C 補洞（2026-05-28）：node.Action/Target 來自 LLM 規劃、depContext 是
	// 上游節點 result_summary——兩者都可能含 injection。整段拼好後過一次 sanitizer，
	// 比每個欄位逐一處理更乾淨（不會破壞中文上下文）。
	return controlseal.SanitizeForLLM(controlseal.SourceCLIOutput, assembled).LLMText
}

func buildCLITaskNodeInstruction() string {
	return "請用對話口吻完成這個步驟，根據前置步驟結果給出使用者可讀答案。若這一步是摘要、整理、回覆、回答、告訴使用者或列出結果，請直接給最終內容，不要主動追問下一步；只有資料不足或任務明確要求互動時才提問。不要表格、不要列檔案路徑、不要行號。"
}

func buildTaskDependencyContext(run *dag.DAGRun, node dag.DAGNode) string {
	if run == nil || len(node.Dependencies) == 0 {
		return ""
	}
	byID := make(map[string]dag.DAGNode, len(run.Nodes))
	for _, n := range run.Nodes {
		byID[n.ID] = n
	}
	var b strings.Builder
	truncated := false
	for _, depID := range node.Dependencies {
		dep, ok := byID[depID]
		if !ok {
			continue
		}
		summary := strings.TrimSpace(dep.ResultSummary)
		if summary == "" && dep.Error == "" {
			continue
		}
		chunk := fmt.Sprintf("- %s %s [%s]\naction_code: %s\ntarget: %s\nresult:\n%s",
			dep.ID, dep.Title, dep.Status, dep.ActionCode, dep.Target, summary)
		if dep.Error != "" {
			chunk += "\nerror:\n" + dep.Error
		}
		if b.Len() > 0 {
			chunk = "\n\n" + chunk
		}
		if b.Len()+len(chunk) > taskDependencyContextLimit {
			remaining := taskDependencyContextLimit - b.Len()
			if remaining > 0 {
				b.WriteString(truncateRunes(chunk, remaining))
			}
			truncated = true
			break
		}
		b.WriteString(chunk)
	}
	if truncated {
		b.WriteString("\n\n[dependency results truncated]")
	}
	return strings.TrimSpace(b.String())
}

func truncateRunes(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		if maxBytes <= 0 {
			return ""
		}
		return s
	}
	var b strings.Builder
	for _, r := range s {
		next := string(r)
		if b.Len()+len(next) > maxBytes {
			break
		}
		b.WriteString(next)
	}
	return b.String()
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
	_ = dag.SaveFullRunLocked(projectRoot, run)
	_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, now, 0, len(run.Nodes), 0, "")
	a.appendTaskResultMessage(run, hasFailed)
	a.clearActiveTask(run.ID)
	a.emitTaskRun("task:progress_completed", run)
}

func (a *App) appendTaskResultMessage(run *dag.DAGRun, failed bool) {
	message := taskResultMessage(run, failed)
	if message == "" {
		return
	}
	a.appendTaskSystemMessage(message)
}

func taskResultMessage(run *dag.DAGRun, failed bool) string {
	if run == nil {
		return ""
	}
	for i := len(run.Nodes) - 1; i >= 0; i-- {
		node := run.Nodes[i]
		if node.Status != dag.StatusSucceeded {
			continue
		}
		if node.ExecutorType != "cli_task" && node.ExecutorType != "subagent_call" {
			continue
		}
		result := strings.TrimSpace(node.ResultSummary)
		if result == "" {
			continue
		}
		if failed {
			return "任務後續步驟失敗，但已取得以下結果：\n\n" + result
		}
		return result
	}
	return ""
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
	case "review_cancel":
		return "你取消高風險步驟"
	case "app_restart":
		return "App 重新啟動"
	case "app_close":
		return "你關閉 App"
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
		if run.Status != "planning" && run.Status != "running" && run.Status != "waiting_review" {
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
		_ = dag.SaveFullRunLocked(projectRoot, run)
		_ = dag.UpdateRunIndex(projectRoot, run.ID, run.Status, now, 0, len(run.Nodes), 0, run.InterruptReason)
		_ = reason
	}
}
