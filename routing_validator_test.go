package main

import (
	"strings"
	"testing"
)

// 撞點 1/2 回歸釘子：背景框架標記必須在出境前被剝除，永不混進 query。
func TestSanitizeRoutingTargetStripsBackgroundFraming(t *testing.T) {
	in := "星座預報\n\n[已補充背景]\n你想查哪個地點？\n台北\n[/已補充背景]"
	got := sanitizeRoutingTarget(in)
	if strings.Contains(got, "已補充背景") || strings.Contains(got, "你想查哪個地點") {
		t.Fatalf("framing 未剝除: %q", got)
	}
	if got != "星座預報" {
		t.Fatalf("got %q, want 星座預報", got)
	}
}

// 硬閘：JSON / 工具呼叫語法不該成為 target（回空 → 上層提問）。
func TestSanitizeRoutingTargetRejectsStructured(t *testing.T) {
	if got := sanitizeRoutingTarget(`{"tool":"x"}`); got != "" {
		t.Fatalf("JSON payload 應被拒, got %q", got)
	}
	if got := sanitizeRoutingTarget("```go\nfmt.Println()\n```"); got != "" {
		t.Fatalf("code fence 應被拒, got %q", got)
	}
}

// 硬閘：未知 action → 退回 need_tool，不臆測工具。
func TestValidateRoutingDecisionRejectsUnknownAction(t *testing.T) {
	app := &App{}
	d := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "亂搞", Target: "x", Next: "待命"}
	out, _ := app.validateRoutingDecision(d, routingValidationContext{})
	if out.Kind != toolRoutingDecisionNeedTool {
		t.Fatalf("未知 action 應退回 need_tool, got %#v", out)
	}
}

// 硬閘：非法 next 收斂成該 action 的安全預設。
func TestValidateRoutingNextNormalized(t *testing.T) {
	app := &App{}
	d := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "網路", Target: "天氣", Next: "亂"}
	out, _ := app.validateRoutingDecision(d, routingValidationContext{})
	if out.Next != "待命" {
		t.Fatalf("非法 next 應預設 待命, got %q", out.Next)
	}
}

// 降級友善：候選/目錄 context 為空時，流程 SkillID 分支 pass-through（不擋）。
func TestValidateSkillIDWhitelistDegraded(t *testing.T) {
	app := &App{}
	d := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "流程", Target: "builtin.scheduler", Next: "輸出"}
	out, verdict := app.validateRoutingDecision(d, routingValidationContext{})
	if verdict != routingValidationPass || out.Action != "流程" {
		t.Fatalf("context 空時應 pass-through, got verdict=%v decision=%#v", verdict, out)
	}
}

// 安全背刺：context 已接上但 SkillID 既不在候選也不在目錄（幻覺）→ 降級提問，不執行。
func TestValidateSkillIDWhitelistRejectsHallucination(t *testing.T) {
	app := &App{}
	vctx := routingValidationContext{
		CandidateSkillIDs: map[string]struct{}{"builtin.scheduler": {}},
		ArchivedSkillIDs:  map[string]struct{}{"builtin.scheduler": {}},
		AlreadyRepicked:   true, // 直接走最終判定，不再 repick
	}
	d := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "流程", Target: "totally.madeup.skill", Next: "輸出"}
	out, _ := app.validateRoutingDecision(d, vctx)
	if out.Action != "提問" {
		t.Fatalf("幻覺 SkillID 應降級提問、不執行, got %#v", out)
	}
}
