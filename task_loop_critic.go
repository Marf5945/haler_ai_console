// task_loop_critic.go — M2 觸發式 Critic（v3.1.7）。
// 只在兩個觸發點審：actor 的完成宣告、seen_signatures 打轉。
// 原則沿用 replan Critic：「只收緊、不放寬」——critic 失效時 fail-open 不擋路，
// 但會留 system observation 供稽核。每節點最多 1 次反問（CriticRounds 持久化）。
// flag：AI_CONSOLE_TASK_LOOP_CRITIC，預設關。
package main

import (
	"fmt"
	"os"
	"strings"

	"ui_console/orchestration/dag"
	"ui_console/shared/actionchain"
)

const (
	loopCriticPassAction     = "通過"
	loopCriticQuestionAction = "反問"
	loopCriticMaxRounds      = 1   // 每節點最多 1 次反問
	loopCriticObsDigestLimit = 200 // 給 critic 看的每筆觀察截斷
)

// taskLoopCriticEnabled：M2 critic feature flag，預設關。
func taskLoopCriticEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AI_CONSOLE_TASK_LOOP_CRITIC"))) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// buildLoopCriticPrompt：問「缺什麼」不問「好不好」，給檢查清單，強制單行控制行。
func buildLoopCriticPrompt(goal, claim string, state *dag.LoopState) string {
	var b strings.Builder
	b.WriteString("你是任務審查者，只負責收緊、不放寬。\n")
	b.WriteString("任務目標：" + goal + "\n")
	if len(state.Observations) > 0 {
		b.WriteString("執行紀錄（依序）：\n")
		for i, o := range state.Observations {
			fmt.Fprintf(&b, "%d. [%s %s] %s\n", i+1, o.Action, truncateRunes(o.Target, 40), truncateRunes(o.SanitizedText, loopCriticObsDigestLimit))
		}
	}
	b.WriteString("目前狀況：" + claim + "\n")
	b.WriteString("檢查清單：1) 目標真的達成了嗎？2) 有沒有宣稱產出但紀錄裡沒有對應動作？3) 有沒有漏掉目標要求的步驟？\n")
	b.WriteString("只輸出一行：都通過 → 通過ㄌ簡短理由ㄌ待命；有具體缺口 → 反問ㄌ缺口一句話ㄌ待命")
	return b.String()
}

// parseLoopCriticVerdict 逐行掃描取最後一行合法判定；解析不到回 ok=false（caller fail-open）。
func parseLoopCriticVerdict(text string) (pass bool, question string, ok bool) {
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		chain, err := actionchain.Parse(strings.TrimSpace(line))
		if err != nil {
			continue
		}
		switch chain.Action {
		case loopCriticPassAction:
			pass, question, ok = true, "", true
		case loopCriticQuestionAction:
			if strings.TrimSpace(chain.Target) != "" {
				pass, question, ok = false, strings.TrimSpace(chain.Target), true
			}
		}
	}
	return pass, question, ok
}

// runLoopCritic 呼叫 planner 介面審一次（task-node- 前綴 trace → 不經工具路由）。
// 回傳 (是否通過, 反問內容)。任何失效（錯誤/不可解析）→ fail-open 視為通過，
// 並在 LoopState 留 system observation 供稽核——critic 只能收緊，不能因故障擋路。
func (a *App) runLoopCritic(run *dag.DAGRun, node dag.DAGNode, goal, claim, adapterID, sessionID string, state *dag.LoopState) (bool, string) {
	traceID := fmt.Sprintf("task-node-critic-%s-c%d", node.ID, state.CriticRounds+1)
	resp, err := a.callPlannerAdapter(adapterID, run.Planner.PlannerModelID, sessionID, buildLoopCriticPrompt(goal, claim, state), traceID)
	if err != nil || resp == nil || resp.Error != "" {
		appendLoopObservation(state, newLoopObservation("system", "審查", "", "critic 呼叫失敗，視為通過（fail-open）"))
		return true, ""
	}
	pass, question, ok := parseLoopCriticVerdict(resp.Text)
	if !ok {
		appendLoopObservation(state, newLoopObservation("system", "審查", "", "critic 輸出無法判讀，視為通過（fail-open）"))
		return true, ""
	}
	return pass, question
}

// maybeRunLoopCritic 封裝觸發條件：flag 開、額度未用完、且有觀察可審。
// 回傳 true 表示 critic 已反問（caller 應 continue 進修正輪）。
func (a *App) maybeRunLoopCritic(run *dag.DAGRun, node dag.DAGNode, goal, claim, adapterID, sessionID string, state *dag.LoopState) bool {
	if !taskLoopCriticEnabled() || state.CriticRounds >= loopCriticMaxRounds || len(state.Observations) == 0 {
		return false
	}
	pass, question := a.runLoopCritic(run, node, goal, claim, adapterID, sessionID, state)
	if pass {
		return false
	}
	state.CriticRounds++
	a.emitTaskLoopRound(run.ID, node.ID, state.Iteration, "審查", truncateRunes(question, 60))
	appendLoopObservation(state, newLoopObservation("critic", loopCriticQuestionAction, "", question))
	return true
}
