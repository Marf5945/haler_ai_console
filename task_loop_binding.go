// task_loop_binding.go — chat_route 節點內 tool loop（ReAct）v3.1.6 M1。
// 核心原則：模型只提議下一步；Go 持有結構（LoopState sidecar）、裁決風險、決定何時停。
// flag 預設關（AI_CONSOLE_TASK_LOOP），關閉時 chat_route 行為與 v3.1.5 完全相同。
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ui_console/data/storage"
	"ui_console/orchestration/dag"
	"ui_console/shared/actionchain"
	"ui_console/shared/controlseal"
)

// ──────────────────────────────────────────────
// Flag 與雙上限（輪數 + sanitized observation 預算，先到先停）
// ──────────────────────────────────────────────

const (
	taskLoopMaxRounds          = 8
	taskLoopObservationBudget  = 8 * 1024 // 累計 sanitized bytes
	taskLoopPerObservationCap  = 2 * 1024 // 單筆 compact 上限（head+tail）
	taskLoopInputArgsMaxBytes  = 32 * 1024
	taskLoopSignatureHintAt    = 2 // 同簽名第 2 次出現 → 注入 system 提示改路
	taskLoopSignatureStopAt    = 3 // 第 3 次 → 轉提問交人
	taskLoopEmitTargetMaxBytes = 120
)

// taskLoopEnabled 回報節點內 loop feature flag。預設關 → 行為與舊版完全相同。
func taskLoopEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AI_CONSOLE_TASK_LOOP"))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// errTaskLoopNeedsUser：loop 內低風險澄清/補參數 → 只暫停本節點（waiting_user），
// run 保持 running，其他 branch 照跑。與 errChatRouteNeedsUser（整 DAG 暫停）區分。
var errTaskLoopNeedsUser = errors.New("task_loop_needs_user")

// ──────────────────────────────────────────────
// Loop 主體
// ──────────────────────────────────────────────

// executeChatRouteLoop 在單一 chat_route 節點內跑多輪「模型提議 → 路由執行 → 觀察回饋」。
// 路由真相仍只有一套：每輪走 routeUserIntentOnce，本函式不另立路由器。
func (a *App) executeChatRouteLoop(run *dag.DAGRun, node dag.DAGNode, adapterID, sessionID, traceID string) (string, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	state := dag.LoadLoopState(projectRoot, run.ID, node.ID)
	state.TraceID = traceID

	for {
		// 雙上限：先到先停，轉提問交人，不無限燒 token。
		// 使用者補充過資訊會追加 ExtraRounds，避免回覆後立刻又撞上限。
		if state.Iteration >= taskLoopMaxRounds+state.ExtraRounds || state.SanitizedBytes() >= taskLoopObservationBudget {
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			return "已達本步驟的嘗試上限，請補充指示或確認目前結果是否足夠。", errTaskLoopNeedsUser
		}
		state.Iteration++

		prompt := buildTaskLoopPrompt(node.Target, state)
		roundTrace := fmt.Sprintf("%s-r%d", traceID, state.Iteration)
		resp, err := a.routeUserIntentOnce(adapterID, sessionID, prompt, roundTrace)
		if err != nil {
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			return "", err
		}
		if resp == nil {
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			return "", fmt.Errorf("chat_route loop: 無回應")
		}
		if resp.Error != "" {
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			return "", errors.New(resp.Error)
		}

		action := strings.TrimSpace(resp.Action)
		text := strings.TrimSpace(resp.Text)
		a.emitTaskLoopRound(run.ID, node.ID, state.Iteration, action, resp.Target)

		// 授權類等待（skill 待確認等）→ 沿用既有整 DAG review 暫停，高風險不走 inline。
		// 已核准（ApprovedAt 非空）就不再暫停，直接收結果——與單發版語意一致。
		if resp.NeedsUser && action != "提問" {
			if strings.TrimSpace(node.ApprovedAt) == "" {
				_ = dag.SaveLoopStateLocked(projectRoot, state)
				return text, errChatRouteNeedsUser
			}
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			return controlseal.SanitizeForLLM(controlseal.SourceCLIOutput, text).LLMText, nil
		}

		// 澄清類提問 → inline waiting_user（低風險：缺資訊/缺參數）。
		if action == "提問" {
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			question := text
			if question == "" {
				question = "這一步需要你補充資訊後才會繼續。"
			}
			return question, errTaskLoopNeedsUser
		}

		// 終局：模型給出最終回答（輸出/聊天/無 action 純文字）→ 節點完成。
		if action == "" || action == "輸出" || action == "聊天" {
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			if text == "" {
				text = "（此步驟無文字輸出）"
			}
			// 結果會進下游節點 prompt，與單發版相同先 sanitize 防跨節點注入。
			return controlseal.SanitizeForLLM(controlseal.SourceCLIOutput, text).LLMText, nil
		}

		// 輸入：只有 PendingInput 存在才是回填參數；否則依規則當一般文字觀察。
		if action == actionchain.InputAction {
			if state.PendingInput != nil {
				obs, ierr := a.applyLoopPendingInput(state, text)
				if ierr != nil {
					// 參數不合法 → 轉提問請補（觀察已記錄缺什麼）。
					_ = dag.SaveLoopStateLocked(projectRoot, state)
					return ierr.Error(), errTaskLoopNeedsUser
				}
				appendLoopObservation(state, obs)
				_ = dag.SaveLoopStateLocked(projectRoot, state)
				continue
			}
			appendLoopObservation(state, newLoopObservation("system", action, "", "（無待補參數的工具，輸入內容視為一般文字）"))
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			continue
		}

		// PendingInput 建立端：模型提出寫 Excel → 設 pending，參數一律走 schema gate。
		// target 已是 JSON 就當場驗證執行；不是 JSON 才停下要參數（inline waiting_user）。
		if state.PendingInput == nil && wantsXlsxWrite(action, resp.Target) {
			state.PendingInput = &dag.PendingInput{Tool: "xlsx_write", SchemaID: "xlsx_write.v1", MissingFields: []string{"file_name", "cells 或 rows"}}
			if strings.HasPrefix(strings.TrimSpace(resp.Target), "{") {
				obs, ierr := a.applyLoopPendingInput(state, resp.Target)
				if ierr == nil {
					appendLoopObservation(state, obs)
					_ = dag.SaveLoopStateLocked(projectRoot, state)
					continue
				}
				_ = dag.SaveLoopStateLocked(projectRoot, state)
				return ierr.Error() + "。" + xlsxInputFormatHint, errTaskLoopNeedsUser
			}
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			return "要產出 Excel，" + xlsxInputFormatHint, errTaskLoopNeedsUser
		}

		// 工具輪：router 已執行工具，回應文字就是 observation。
		count := state.RecordSignature(loopSignature(action, resp.Target))
		switch {
		case count >= taskLoopSignatureStopAt:
			// 原地打轉 → 停下交人，不靠模型自己醒。
			_ = dag.SaveLoopStateLocked(projectRoot, state)
			return fmt.Sprintf("已重複嘗試「%s %s」仍無進展，請補充指示。", action, truncateRunes(resp.Target, 60)), errTaskLoopNeedsUser
		case count == taskLoopSignatureHintAt:
			appendLoopObservation(state, newLoopObservation("system", action, resp.Target, "你已重複過這個動作，請換方法或直接輸出結論。"))
		}
		appendLoopObservation(state, newLoopObservation("tool", action, resp.Target, text))
		state.TrimToBudget(taskLoopObservationBudget)
		_ = dag.SaveLoopStateLocked(projectRoot, state)
	}
}

// buildTaskLoopPrompt 組每輪 prompt：原始步驟目標 + 已累積觀察（只放 sanitized）+ 控制規則。
func buildTaskLoopPrompt(goal string, state *dag.LoopState) string {
	var b strings.Builder
	b.WriteString("你正在執行任務步驟：")
	b.WriteString(goal)
	b.WriteString("\n")
	if len(state.Observations) > 0 {
		b.WriteString("目前已完成的觀察（依序）：\n")
		for i, o := range state.Observations {
			label := o.Action
			if o.Kind == "user_input" {
				label = "使用者補充"
			} else if o.Kind == "system" {
				label = "系統提示"
			}
			fmt.Fprintf(&b, "%d. [%s %s] %s\n", i+1, label, truncateRunes(o.Target, 60), o.SanitizedText)
		}
	}
	b.WriteString("規則：每次只做一步。需要工具就用 動作ㄌ目標ㄌ待命；資訊已足夠時用 輸出ㄌ最終結果ㄌ待命 結束；缺必要資訊才用 提問ㄌ問題ㄌ待命。不要重複已完成的動作。")
	return b.String()
}

// newLoopObservation 依「compact → sanitize → store」順序產生一筆觀察。
// 順序不可反：先 sanitize 再截斷會把 escape marker 切壞（同 task_progress_fs.go 教訓）。
func newLoopObservation(kind, action, target, rawText string) dag.ObservationRecord {
	compact, truncated := compactObservationText(rawText)
	source := controlseal.SourceCLIOutput
	if kind == "user_input" {
		source = controlseal.SourceUserRaw
	}
	return dag.ObservationRecord{
		Kind:          kind,
		Action:        action,
		Target:        truncateRunes(target, 200),
		CompactText:   compact,
		SanitizedText: controlseal.SanitizeForLLM(source, compact).LLMText,
		Hash:          dag.HashObservation(action, target, rawText),
		Truncated:     truncated,
	}
}

func appendLoopObservation(state *dag.LoopState, obs dag.ObservationRecord) {
	state.Observations = append(state.Observations, obs)
}

// compactObservationText 是 deterministic 壓縮：超限取 head+tail，不叫 LLM。
func compactObservationText(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= taskLoopPerObservationCap {
		return trimmed, false
	}
	half := taskLoopPerObservationCap / 2
	head := truncateRunes(trimmed, half)
	tail := trimmed[len(trimmed)-half:]
	// tail 起點可能落在 multi-byte rune 中間，先丟掉 continuation bytes。
	for len(tail) > 0 && (tail[0]&0xC0) == 0x80 {
		tail = tail[1:]
	}
	return head + "\n…（中段省略）…\n" + tail, true
}

const xlsxInputFormatHint = `請提供參數：輸入ㄌ{"file_name":"out.xlsx","rows":[["A1","B1"]]}ㄌ待命（或直接貼 JSON）`

// wantsXlsxWrite 判斷模型這一步是否在要求產出 Excel（PendingInput 的建立條件）。
// 保守判斷：寫入/匯出 + target 帶 xlsx/excel/試算表 線索，避免攔到一般檔案寫入。
func wantsXlsxWrite(action, target string) bool {
	switch action {
	case "寫入", "匯出":
	default:
		return false
	}
	lower := strings.ToLower(target)
	return strings.Contains(lower, ".xlsx") || strings.Contains(lower, "excel") || strings.Contains(target, "試算表")
}

// loopSignature 分級簽名：read-only 用 Action+Target 二元組；
// write/side-effect 類加 target hash，防同一路徑重複寫。
func loopSignature(action, target string) string {
	normalized := strings.TrimSpace(strings.ToLower(target))
	switch action {
	case "寫入", "匯出", "儲存", "匯入", "git", "排程", "操作":
		return action + "|" + dag.HashObservation(action, normalized, "")
	default:
		return action + "|" + truncateRunes(normalized, 120)
	}
}

// emitTaskLoopRound 每輪一個事件，前端節點卡片顯示「第 N 輪：動作 目標」。
func (a *App) emitTaskLoopRound(runID, nodeID string, iteration int, action, target string) {
	if a.eventBus == nil {
		return
	}
	a.eventBus.Emit("task:loop_round", map[string]interface{}{
		"run_id":    runID,
		"node_id":   nodeID,
		"iteration": iteration,
		"action":    action,
		"target":    truncateRunes(target, taskLoopEmitTargetMaxBytes),
	})
}

// ──────────────────────────────────────────────
// inline waiting_user：暫停本節點、run 保持 running
// ──────────────────────────────────────────────

// pauseTaskLoopForUser 只把節點轉 waiting_user，不開 review card、不暫停整個 run；
// 其他 ready branch 由 runTaskProgress 繼續排程。
func (a *App) pauseTaskLoopForUser(projectRoot string, run *dag.DAGRun, node *dag.DAGNode, question string) {
	if strings.TrimSpace(question) == "" {
		question = "這一步需要你補充資訊後才會繼續。"
	}
	node.Status = dag.StatusWaitingUser
	node.ResultSummary = question
	run.UpdatedAt = time.Now().Format(time.RFC3339)
	_ = dag.SaveFullRunLocked(projectRoot, run)
	a.emitTaskRun("task:progress_updated", run)
}

// SubmitTaskLoopInput 是 inline 提問的回覆入口（Wails binding）。
// 使用者輸入與模型輸出走同一條 schema gate，不開後門。
func (a *App) SubmitTaskLoopInput(runID, nodeID, text string) (*dag.DAGRun, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("輸入不可為空")
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	run, err := dag.LoadFullRun(projectRoot, runID)
	if err != nil {
		return nil, err
	}
	var node *dag.DAGNode
	for i := range run.Nodes {
		if run.Nodes[i].ID == nodeID {
			node = &run.Nodes[i]
			break
		}
	}
	if node == nil {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}
	if node.Status != dag.StatusWaitingUser {
		return nil, fmt.Errorf("node is not waiting for input: %s", node.Status)
	}

	state := dag.LoadLoopState(projectRoot, runID, nodeID)
	if state.PendingInput != nil {
		// 待補參數模式：回覆必須是 輸入ㄌ{json}ㄌ待命 或裸 JSON，過 schema gate 才執行。
		obs, ierr := a.applyLoopPendingInput(state, text)
		if ierr != nil {
			return nil, ierr
		}
		appendLoopObservation(state, obs)
	} else {
		appendLoopObservation(state, newLoopObservation("user_input", "提問回覆", "", text))
	}
	state.TrimToBudget(taskLoopObservationBudget)
	if state.Iteration >= taskLoopMaxRounds+state.ExtraRounds {
		state.ExtraRounds += 2 // 補充資訊换 2 輪額度，不無限展延
	}
	if err := dag.SaveLoopStateLocked(projectRoot, state); err != nil {
		return nil, err
	}

	// 節點回 Ready，loop 會帶著已存 observations 從下一輪續跑。
	now := time.Now().Format(time.RFC3339)
	node.Status = dag.StatusReady
	node.ResultSummary = ""
	run.Status = "running"
	run.UpdatedAt = now
	_ = dag.SaveFullRunLocked(projectRoot, run)
	a.emitTaskRun("task:progress_updated", run)
	go a.runTaskProgress(run.ID, run.Planner.PlannerAdapterID, a.globalSessionID)
	return run, nil
}

// ──────────────────────────────────────────────
// PendingInput schema gate（M1 已接 xlsx_write；其他工具依 SchemaID 擴充）
// ──────────────────────────────────────────────

// applyLoopPendingInput：parse → schema 驗證 → 合併 PartialArgs → 執行工具 → 回觀察。
// 任一步失敗回 error（內容說明缺什麼），由呼叫端轉提問。
func (a *App) applyLoopPendingInput(state *dag.LoopState, text string) (dag.ObservationRecord, error) {
	pending := state.PendingInput
	payload := strings.TrimSpace(text)
	if actionchain.IsInputLine(payload) {
		chain, err := actionchain.ParseInputLine(payload)
		if err != nil {
			return dag.ObservationRecord{}, fmt.Errorf("輸入格式不對，請用 輸入ㄌ{JSON}ㄌ待命")
		}
		payload = chain.Target
	}
	if len(payload) > taskLoopInputArgsMaxBytes {
		return dag.ObservationRecord{}, fmt.Errorf("參數太大（上限 %d bytes）", taskLoopInputArgsMaxBytes)
	}
	merged, err := mergePendingArgs(pending.PartialArgs, []byte(payload))
	if err != nil {
		return dag.ObservationRecord{}, fmt.Errorf("參數 JSON 不合法：%v", err)
	}

	switch pending.SchemaID {
	case "xlsx_write.v1":
		if err := validateXlsxWriteArgs(merged); err != nil {
			pending.PartialArgs = merged // 留下已給的部分，下次只補缺的
			return dag.ObservationRecord{}, err
		}
		result, err := a.executeXlsxWriteTaskNode(dag.DAGNode{ExecutorType: "tool_call", ActionCode: "xlsx_write", Target: string(merged)})
		if err != nil {
			return dag.ObservationRecord{}, fmt.Errorf("xlsx_write 執行失敗：%v", err)
		}
		state.PendingInput = nil
		return newLoopObservation("tool", "寫入", pending.Tool, result), nil
	default:
		return dag.ObservationRecord{}, fmt.Errorf("未知的 pending schema: %s", pending.SchemaID)
	}
}

// mergePendingArgs 淺層合併兩包 JSON object（新值覆蓋舊值）。
func mergePendingArgs(base, patch json.RawMessage) (json.RawMessage, error) {
	merged := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(base)) > 0 {
		if err := json.Unmarshal(base, &merged); err != nil {
			return nil, err
		}
	}
	patchMap := map[string]json.RawMessage{}
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, err
	}
	for k, v := range patchMap {
		merged[k] = v
	}
	return json.Marshal(merged)
}

// validateXlsxWriteArgs：DisallowUnknownFields + 欄位白名單 + 數量/長度上限。
func validateXlsxWriteArgs(raw json.RawMessage) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var target xlsxWriteTarget
	if err := dec.Decode(&target); err != nil {
		return fmt.Errorf("欄位不合法（只接受 xlsx_write 已知欄位）：%v", err)
	}
	var missing []string
	if strings.TrimSpace(target.FileName) == "" {
		missing = append(missing, "file_name")
	}
	if len(target.Cells) == 0 && len(target.Rows) == 0 {
		missing = append(missing, "cells 或 rows")
	}
	if len(missing) > 0 {
		return fmt.Errorf("還缺欄位：%s", strings.Join(missing, "、"))
	}
	if len(target.Rows) > 2000 || len(target.Cells) > 2000 {
		return fmt.Errorf("rows/cells 數量超過上限 2000")
	}
	if len(target.Styles) > 100 {
		return fmt.Errorf("styles 數量超過上限 100")
	}
	return nil
}
