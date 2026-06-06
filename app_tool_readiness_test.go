package main

import (
	"strings"
	"testing"

	"ui_console/shared/actionchain"
)

func newToolReadinessTestApp() *App {
	return &App{
		pendingToolQuestions:   make(map[string]pendingToolQuestion),
		toolBackgroundContexts: make(map[string][]toolBackgroundAnswer),
	}
}

func TestActionChainQuestionNext(t *testing.T) {
	if !actionchain.IsQuestionNext("提問") {
		t.Fatal("提問 should be a next-state")
	}
	if actionchain.IsQuestionNext("待命") {
		t.Fatal("待命 should not be a question next-state")
	}
}

func TestToolReadinessAsksForWeatherLocation(t *testing.T) {
	app := newToolReadinessTestApp()
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "網路", Target: "今天會下雨嗎", Next: actionchain.StandbyNext}

	handled, resp := app.maybeAskForToolReadiness("s1", decision, "trace-test")
	if !handled {
		t.Fatal("expected readiness gate to ask for missing location")
	}
	if resp.Next != actionchain.QuestionNext || !strings.Contains(resp.Text, "地點") {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if _, ok := app.pendingToolQuestions["s1"]; !ok {
		t.Fatal("pending question was not recorded")
	}
}

func TestToolReadinessAcceptsLocationHint(t *testing.T) {
	app := newToolReadinessTestApp()
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "網路", Target: "台北今天會下雨嗎", Next: actionchain.StandbyNext}

	handled, _ := app.maybeAskForToolReadiness("s1", decision, "trace-test")
	if handled {
		t.Fatal("location hint in target should be enough for first pass")
	}
}

func TestToolReadinessConsumesClarificationAndRerunsAction(t *testing.T) {
	app := newToolReadinessTestApp()
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "網路", Target: "今天會下雨嗎", Next: actionchain.StandbyNext}
	if handled, _ := app.maybeAskForToolReadiness("s1", decision, "trace-test"); !handled {
		t.Fatal("expected initial question")
	}

	rerun, ok := app.consumePendingToolAnswer("s1", "台北", "trace-test")
	if !ok {
		t.Fatal("expected clarification to be consumed")
	}
	if rerun.Action != "網路" || rerun.Target != "今天會下雨嗎" || rerun.Next != actionchain.StandbyNext {
		t.Fatalf("unexpected rerun decision: %#v", rerun)
	}
	ctx := app.formatToolBackgroundContext("s1")
	if !strings.Contains(ctx, "台北") || !strings.Contains(ctx, "[已補充背景]") {
		t.Fatalf("background context missing clarification: %q", ctx)
	}
}

func TestToolReadinessQuestionNextDoesNotRequireQuestionAction(t *testing.T) {
	app := newToolReadinessTestApp()
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "讀取", Target: "那個檔案", Next: actionchain.QuestionNext}

	handled, resp := app.maybeAskForToolReadiness("s1", decision, "trace-test")
	if !handled {
		t.Fatal("next=提問 should ask without using 提問 as action")
	}
	if resp.Action != "讀取" || resp.Next != actionchain.QuestionNext {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestQueryActionIsNotPromotedToWebSearch(t *testing.T) {
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "查詢", Target: "今天會下雨嗎", Next: actionchain.StandbyNext}
	normalized := normalizeToolRoutingDecision(decision, "今天會下雨嗎", toolRoutingLookupContext{Query: "今天會下雨嗎"})
	if normalized.Action != "查詢" {
		t.Fatalf("查詢 should stay stored-data query, got %#v", normalized)
	}
}
