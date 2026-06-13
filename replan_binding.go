package main

// replan_binding.go — Bounded Replan 4b：main 端真 Proposer adapter 與活線觸發。
//
// 安全開關：整條活線觸發由 replanEnabled()（環境變數）控制，預設「關」。
// 關閉時 tryReplanOnFailure 直接回 false，行為與接線前完全一致。

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ui_console/orchestration/dag"
	"ui_console/orchestration/replan"
	"ui_console/shared/actionchain"
	"ui_console/shared/controlseal"
)

// replanShortFailureHint 引導 LLM 在「找不到」時短答，讓 ClassifyResult 的長度守門可靠。
// 只在 flag 開時加入節點 prompt，維持關閉時零行為改變。
func replanShortFailureHint() string {
	return "若沒有找到相關資料、檔案不存在或查無結果，請只簡短回覆「找不到」或「沒有找到相關資料」，不要長篇道歉或說明。"
}

// replanEnabled 回報是否啟用活線 Bounded Replan（預設關）。
func replanEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("UICONSOLE_BOUNDED_REPLAN"))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// 每個 run 一個連續無進展計數器（process 內，跨 run 獨立）。
var (
	replanCountersMu sync.Mutex
	replanCounters   = map[string]*replan.Counter{}
)

func replanCounterFor(runID string) *replan.Counter {
	replanCountersMu.Lock()
	defer replanCountersMu.Unlock()
	c, ok := replanCounters[runID]
	if !ok {
		c = replan.NewCounter()
		replanCounters[runID] = c
	}
	return c
}

// plannerProposer 是 replan.Proposer 的真實實作：把 ProposerContext 組成 prompt、
// 沿用既有 callPlannerAdapter 取得結構化 JSON，再 parse 成 ReplanProposal。
// 不引入任何 agent framework。
type plannerProposer struct {
	app       *App
	adapterID string
	modelID   string
	sessionID string
}

// Propose 在 goroutine 內呼叫 planner，並以 ctx 實作真正的逾時 fail-safe。
func (p plannerProposer) Propose(ctx context.Context, pc replan.ProposerContext) (replan.ReplanProposal, error) {
	type outcome struct {
		prop replan.ReplanProposal
		err  error
	}
	ch := make(chan outcome, 1)
	go func() {
		traceID := fmt.Sprintf("replan-%d", time.Now().UnixNano())
		prop, perr, terr := p.callAndParse(buildReplanPrompt(pc), traceID)
		if terr != nil {
			ch <- outcome{err: terr}
			return
		}
		// 格式錯一次 → 用更嚴格的修復提示重試一次；仍錯交給 Gate review（不放寬安全）。
		if perr != nil {
			if prop2, perr2, terr2 := p.callAndParse(buildReplanRepairPrompt(pc), traceID+"-repair"); terr2 == nil {
				prop, perr = prop2, perr2
			}
		}
		ch <- outcome{prop: prop, err: perr}
	}()
	select {
	case <-ctx.Done():
		return replan.ReplanProposal{}, ctx.Err() // 逾時 → Coordinator fail-safe 進 review
	case r := <-ch:
		return r.prop, r.err
	}
}

// callAndParse 呼叫 planner 後解析。回傳 (提案, 解析錯, 傳輸錯)；
// 傳輸錯=adapter 失敗，解析錯=格式不對（可由呼叫端決定是否重試）。
func (p plannerProposer) callAndParse(prompt, traceID string) (replan.ReplanProposal, error, error) {
	resp, err := p.app.callPlannerAdapter(p.adapterID, p.modelID, p.sessionID, prompt, traceID)
	if err != nil {
		return replan.ReplanProposal{}, nil, err
	}
	if resp == nil {
		return replan.ReplanProposal{}, fmt.Errorf("planner 回應為空"), nil
	}
	prop, perr := parseReplanProposal(resp.Text)
	return prop, perr, nil
}

// buildReplanPrompt 組出 replan 提案 prompt。只給同目標換路所需脈絡，
// 明確限制只能用 eligible（read-only）動作，並要求嚴格 JSON 輸出。
func buildReplanPrompt(pc replan.ProposerContext) string {
	seal := controlseal.CurrentSeal()
	var b strings.Builder
	b.WriteString("原步驟沒有找到資料。請在不改變原目標下，提出一個新的本機查找步驟。\n")
	b.WriteString("目標：" + pc.Contract.GoalSummary + "\n")
	if len(pc.CompletedSummary) > 0 {
		b.WriteString("已完成：" + strings.Join(pc.CompletedSummary, "；") + "\n")
	}
	b.WriteString("只輸出一行，格式固定：" + seal + "動作ㄌ目標ㄌ輸出\n")
	b.WriteString("動作只能是：搜尋、讀取、列出（搜尋=找本機資料；讀取=看檔案；列出=列出目錄）\n")
	b.WriteString("命令最前面一定要加上這個前綴：" + seal + "（沒有前綴的內容不會被當成命令）\n")
	b.WriteString("不要解釋、不要 JSON。想不到新方式就只回：找不到\n")
	b.WriteString("範例：" + seal + "搜尋ㄌapp 設定檔ㄌ輸出\n")
	return b.String()
}

// buildReplanRepairPrompt：第一次格式錯後的修復提示，比原 prompt 更嚴格、只准一行。
func buildReplanRepairPrompt(pc replan.ProposerContext) string {
	seal := controlseal.CurrentSeal()
	var b strings.Builder
	b.WriteString("剛才的格式不對。請嚴格只輸出一行，不要任何其他字。\n")
	b.WriteString("格式：" + seal + "動作ㄌ目標ㄌ輸出\n")
	b.WriteString("動作只能三選一：搜尋、讀取、列出\n")
	b.WriteString("目標：" + pc.Contract.GoalSummary + "\n")
	b.WriteString("想不到就只回：" + actionchain.NoProposalToken + "\n")
	b.WriteString("範例：" + seal + "搜尋ㄌ設定檔ㄌ輸出\n")
	return b.String()
}

// parseReplanProposal 解析模型回應：逐行驗印（controlseal）→ actionchain.ParseSingleLine。
// 沒有當前印的行視為內容（含被檢索回來的注入）而忽略，只有帶印命令才成案。
func parseReplanProposal(text string) (replan.ReplanProposal, error) {
	seal := controlseal.CurrentSeal()
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// 「找不到」可無印：模型放棄。
		if line == actionchain.NoProposalToken {
			return replan.ReplanProposal{Intent: replan.IntentSameGoalPath, Reason: "找不到"}, nil
		}
		// 只有帶當前印的行才是命令；沒印的行是內容（含被檢索回來的注入）→ 忽略。
		if !strings.HasPrefix(line, seal) {
			continue
		}
		r := actionchain.ParseSingleLine(strings.TrimPrefix(line, seal))
		if r.NoProposal {
			return replan.ReplanProposal{Intent: replan.IntentSameGoalPath, Reason: "找不到"}, nil
		}
		if r.Err != nil {
			return replan.ReplanProposal{}, fmt.Errorf("replan 提案格式錯誤: %s", r.Err.Reason)
		}
		return replan.ReplanProposal{
			Intent:       replan.IntentSameGoalPath,
			ProposedTail: []replan.ProposedNode{{Action: r.Chain.Action, Target: r.Chain.Target}},
		}, nil
	}
	// 沒有任何帶印命令不能成案；回解析錯讓 Propose 走一次 repair prompt。
	return replan.ReplanProposal{}, fmt.Errorf("replan 提案格式錯誤: 沒有有效帶印命令")
}

// tryReplanOnFailure 是活線觸發入口（flag 關時即 no-op）。
// 回傳 true 代表已 silent re-route（呼叫端不應再標 failed，runTaskProgress 會 reload 續跑）。
func (a *App) tryReplanOnFailure(projectRoot string, run *dag.DAGRun, node *dag.DAGNode, adapterID, sessionID, result string, execErr error) bool {
	if !replanEnabled() {
		return false
	}
	failure := replan.FailureCategory(node.FailureCategory)
	if !replan.IsReplanTrigger(failure) {
		return false
	}
	if run.GoalContract == nil {
		return false // legacy run 無契約 → 不自動 replan，走原失敗路徑
	}
	triggerRisk := replan.TriggerRiskFor(node.Action, node.Target)

	co := replan.NewCoordinator(
		plannerProposer{app: a, adapterID: adapterID, modelID: run.Planner.PlannerModelID, sessionID: sessionID},
		replanCounterFor(run.ID),
		replan.NewAuditLog(projectRoot),
		0,
	)
	failedID := node.ID // Attempt 會重配 run.Nodes，先記 id

	res := co.Attempt(run, failure, triggerRisk)
	if !res.Applied {
		return false // review / stop / fail-safe → 交回現有失敗/審核機制
	}

	// silent re-route：把失敗節點標 skipped（已繞過）。Attempt 已重配 slice，依 id 重找。
	for i := range run.Nodes {
		if run.Nodes[i].ID == failedID {
			run.Nodes[i].Status = dag.StatusSkipped
			run.Nodes[i].Error = execErr.Error()
			break
		}
	}
	run.Status = "running"
	// status rail 活動摘要：走 event、不進 talk 主歷史，不暴露 raw 輸出。
	run.ReplanActivity = replan.ActivitySummary(failure, res.Stage, replanCounterFor(run.ID).ConsecutiveNoProgress)
	run.UpdatedAt = time.Now().Format(time.RFC3339)
	_ = dag.SaveFullRunLocked(projectRoot, run)
	a.emitTaskRun("task:progress_replanned", run)
	return true
}
