package main

import (
	"strings"
	"testing"

	"ui_console/orchestration/dag"
)

// 判定解析：取最後一行合法控制行；垃圾輸出回 ok=false（fail-open 由 caller 處理）。
func TestParseLoopCriticVerdict(t *testing.T) {
	if pass, _, ok := parseLoopCriticVerdict("通過ㄌ目標已達成ㄌ待命"); !ok || !pass {
		t.Fatal("pass line")
	}
	pass, q, ok := parseLoopCriticVerdict("反問ㄌ宣稱已產出報表但紀錄沒有寫入動作ㄌ待命")
	if !ok || pass || !strings.Contains(q, "寫入動作") {
		t.Fatalf("question line: %v %q", pass, q)
	}
	// 模型先碎念再給控制行 → 取最後一行合法判定
	multi := "讓我檢查一下。\n目標說要產出檔案。\n反問ㄌ缺少實際產出檔案的動作ㄌ待命"
	if pass, q, ok = parseLoopCriticVerdict(multi); !ok || pass || q == "" {
		t.Fatal("multiline should take last valid line")
	}
	if _, _, ok = parseLoopCriticVerdict("我覺得大致沒問題"); ok {
		t.Fatal("garbage must be not-ok")
	}
	// 空缺口的反問不算合法判定
	if _, _, ok = parseLoopCriticVerdict("反問ㄌ ㄌ待命"); ok {
		t.Fatal("empty question must be not-ok")
	}
}

// critic prompt：含目標、紀錄、狀況與檢查清單，且問「缺什麼」。
func TestBuildLoopCriticPrompt(t *testing.T) {
	state := &dag.LoopState{Observations: []dag.ObservationRecord{
		{Kind: "tool", Action: "搜尋", Target: "設定", SanitizedText: "找到 config"},
	}}
	prompt := buildLoopCriticPrompt("整理設定並輸出報告", "完成宣告：已整理", state)
	for _, want := range []string{"整理設定並輸出報告", "找到 config", "完成宣告：已整理", "檢查清單", "通過ㄌ", "反問ㄌ"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

// flag 與上限設定化：env 覆寫生效、非法值回預設。
func TestTaskLoopConfigValues(t *testing.T) {
	t.Setenv("AI_CONSOLE_TASK_LOOP_MAX_ROUNDS", "12")
	t.Setenv("AI_CONSOLE_TASK_LOOP_BUDGET_KB", "16")
	if taskLoopMaxRoundsValue() != 12 || taskLoopBudgetValue() != 16*1024 {
		t.Fatalf("env override failed: %d %d", taskLoopMaxRoundsValue(), taskLoopBudgetValue())
	}
	t.Setenv("AI_CONSOLE_TASK_LOOP_MAX_ROUNDS", "0")    // 非法 → 預設
	t.Setenv("AI_CONSOLE_TASK_LOOP_BUDGET_KB", "99999") // 超界 → 預設
	if taskLoopMaxRoundsValue() != taskLoopMaxRounds || taskLoopBudgetValue() != taskLoopObservationBudget {
		t.Fatal("invalid values should fall back to defaults")
	}
	t.Setenv("AI_CONSOLE_TASK_LOOP_CRITIC", "1")
	if !taskLoopCriticEnabled() {
		t.Fatal("critic flag")
	}
}

// 觸發封裝：flag 關 / 額度用完 / 無觀察 → 一律不觸發（不耗呼叫）。
func TestMaybeRunLoopCriticGates(t *testing.T) {
	app := &App{}
	run := &dag.DAGRun{}
	node := dag.DAGNode{ID: "n1"}
	state := &dag.LoopState{Observations: []dag.ObservationRecord{{Kind: "tool"}}}
	t.Setenv("AI_CONSOLE_TASK_LOOP_CRITIC", "0")
	if app.maybeRunLoopCritic(run, node, "g", "c", "a", "s", state) {
		t.Fatal("flag off must not trigger")
	}
	t.Setenv("AI_CONSOLE_TASK_LOOP_CRITIC", "1")
	state.CriticRounds = loopCriticMaxRounds
	if app.maybeRunLoopCritic(run, node, "g", "c", "a", "s", state) {
		t.Fatal("budget exhausted must not trigger")
	}
	state.CriticRounds = 0
	state.Observations = nil
	if app.maybeRunLoopCritic(run, node, "g", "c", "a", "s", state) {
		t.Fatal("no observations must not trigger")
	}
}
