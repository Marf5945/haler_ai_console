package main

import (
	"strings"
	"testing"

	"ui_console/shared/localsearch"
)

func TestVagueSearchIntentTriggersClarification(t *testing.T) {
	app := newToolReadinessTestApp()
	resp, handled := app.maybeHandleSearchIntentClarification("我要查東西", "s1", "trace-test")
	if !handled || resp == nil {
		t.Fatal("expected vague search intent clarification")
	}
	if resp.Text != "你想搜尋哪一類？" {
		t.Fatalf("question = %q", resp.Text)
	}
	pending, ok := app.pendingToolQuestions["s1"]
	if !ok {
		t.Fatal("expected pending clarification to be stored")
	}
	if pending.Question != buildSearchIntentQuestion() {
		t.Fatalf("pending question = %q", pending.Question)
	}
}

func TestSearchIntentQuestionOffersSearchChoices(t *testing.T) {
	question, candidates := floatingCandidatesFromQuestionTarget(buildSearchIntentQuestion())
	if question != "你想搜尋哪一類？" {
		t.Fatalf("question = %q", question)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d", len(candidates))
	}
	if candidates[0].Draft != "input:本機搜尋" {
		t.Fatalf("first candidate = %+v", candidates[0])
	}
	if candidates[1].Draft != "input:搜尋 git 紀錄" {
		t.Fatalf("second candidate = %+v", candidates[1])
	}
	if candidates[2].Draft != "input:網路搜尋" {
		t.Fatalf("third candidate = %+v", candidates[2])
	}
}

func TestExplicitSearchTopicDoesNotTriggerGenericClarification(t *testing.T) {
	if !isVagueSearchIntent("我要查東西") {
		t.Fatal("expected vague intent to be detected")
	}
	if isVagueSearchIntent("我要查台中天氣") {
		t.Fatal("explicit search topic should not be treated as vague")
	}
	if isVagueSearchIntent("網路查詢天氣 台中") {
		t.Fatal("explicit web command should not be treated as vague")
	}
}

func TestVagueSearchIntentNormalizesNeedToolToQuestion(t *testing.T) {
	decision := toolRoutingDecision{Kind: toolRoutingDecisionNeedTool, Raw: "plain text"}
	normalized := normalizeToolRoutingDecision(decision, "我要查東西", toolRoutingLookupContext{})
	if normalized.Kind != toolRoutingDecisionAction || normalized.Action != "提問" {
		t.Fatalf("expected vague search intent to ask, got %#v", normalized)
	}
	if normalized.Target != buildSearchIntentQuestion() {
		t.Fatalf("question payload = %q", normalized.Target)
	}
}

func TestSearchIntentNaturalLanguageTriggersRepair(t *testing.T) {
	decision := toolRoutingDecision{Kind: toolRoutingDecisionChat, Text: "What can I help with?"}
	if !shouldRepairToolRoutingDecision("網路查詢天氣 台中", decision) {
		t.Fatal("search intent chat output should be repaired")
	}
	prompt := buildToolRoutingRepairPrompt("BASE", "bad output", "網路查詢天氣 台中")
	for _, want := range []string{"BASE", "網路", "搜尋", "提問"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("repair prompt missing %q: %s", want, prompt)
		}
	}
}

func TestLocalSearchReroutePromptUsesTopThreeOnly(t *testing.T) {
	outcome := localsearch.SearchOutcome{Results: []localsearch.SearchResult{
		{Title: "one", Source: "trace", Score: 12},
		{Title: "two", Source: "tool", Score: 20},
		{Title: "three", Source: "memory", Score: 30},
		{Title: "four", Source: "document", Score: 40},
	}}
	if !shouldRejudgeLocalSearchToWeb(outcome) {
		t.Fatal("low-quality local results should trigger reroute judge")
	}
	prompt := buildLocalSearchWebReroutePrompt("網路查詢天氣 台中", localsearch.SearchRequest{Query: "天氣 台中"}, outcome)
	if !strings.Contains(prompt, "local_hits=4") || !strings.Contains(prompt, "title=\"three\"") {
		t.Fatalf("prompt missing local summary: %s", prompt)
	}
	if strings.Contains(prompt, "title=\"four\"") {
		t.Fatalf("prompt should only include top 3 results: %s", prompt)
	}
}
