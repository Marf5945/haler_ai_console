package main

import (
	"testing"

	"ui_console/adapter/visual_learning"
	"ui_console/shared/actionchain"
)

func chainText(action, target, next string) string {
	return action + actionchain.Separator + target + actionchain.Separator + next
}

func TestParseToolRoutingDecisionChatRequiresPrefix(t *testing.T) {
	decision := parseToolRoutingDecision("\u9592\u804a" + actionchain.Separator + "\u9019\u662f\u4e00\u822c\u56de\u7b54")
	if decision.Kind != toolRoutingDecisionChat {
		t.Fatalf("Kind = %q, want %q", decision.Kind, toolRoutingDecisionChat)
	}
	if decision.Text != "\u9019\u662f\u4e00\u822c\u56de\u7b54" {
		t.Fatalf("Text = %q", decision.Text)
	}

	plain := parseToolRoutingDecision("\u9019\u662f\u6c92\u6709\u524d\u7db4\u7684\u81ea\u7136\u8a9e\u8a00")
	if plain.Kind != toolRoutingDecisionNeedTool {
		t.Fatalf("plain natural language should fall back to need_tool, got %q", plain.Kind)
	}
}

func TestParseToolRoutingDecisionOperationAction(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u67e5\u8a62", "Chrome ChatGPT \u95dc\u9589\u5206\u9801", "\u64cd\u4f5c"))
	if decision.Kind != toolRoutingDecisionAction {
		t.Fatalf("Kind = %q, want action", decision.Kind)
	}
	if decision.Action != "\u67e5\u8a62" || decision.Target != "Chrome ChatGPT \u95dc\u9589\u5206\u9801" || decision.Next != "\u64cd\u4f5c" {
		t.Fatalf("unexpected action chain: %#v", decision)
	}
}

func TestParseSearchTermsSpaceSeparated(t *testing.T) {
	terms := parseSearchTerms("\u91cd\u73fe Chrome \u700f\u89bd\u5668 \u9ede\u64ca ChatGPT \u9801\u9762\u5167\u5bb9 \u95dc\u9589 \u5206\u9801 \u9304\u88fd \u52d5\u4f5c", "\u5e6b\u6211\u91cd\u73fe Chrome")
	if len(terms) < 8 {
		t.Fatalf("expected parsed terms, got %#v", terms)
	}
	want := map[string]bool{
		"\u91cd\u73fe":       true,
		"Chrome":             true,
		"\u700f\u89bd\u5668": true,
		"ChatGPT":            true,
		"\u95dc\u9589":       true,
		"\u9304\u88fd":       true,
		"\u52d5\u4f5c":       true,
	}
	for _, term := range terms {
		delete(want, term)
	}
	if len(want) != 0 {
		t.Fatalf("missing terms: %#v from %#v", want, terms)
	}
}

func TestInferSkillActionTargetDoesNotStealNaturalLanguage(t *testing.T) {
	app := &App{}
	if got, ok := app.inferSkillActionTarget("\u5e6b\u6211\u67e5\u8a62\u9304\u88fd\u7684\u64cd\u4f5c"); ok {
		t.Fatalf("natural language should enter three-stage routing first, got %q", got)
	}
	if got, ok := app.inferSkillActionTarget(chainText("\u67e5\u8a62", "\u5df2\u9304\u88fd\u7684\u95dc\u9589\u5206\u9801", "\u64cd\u4f5c")); !ok || got != "\u67e5\u8a62"+actionchain.Separator+"\u5df2\u9304\u88fd\u7684\u95dc\u9589\u5206\u9801" {
		t.Fatalf("explicit action-chain should still route, got %q ok=%v", got, ok)
	}
}

func TestNormalizeToolRoutingDecisionPromotesReplayQueryToOperation(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u67e5\u8a62", "Chrome ChatGPT \u95dc\u9589\u5206\u9801", "\u64cd\u4f5c"))
	lookup := toolRoutingLookupContext{
		Query: "Chrome ChatGPT \u95dc\u9589\u5206\u9801",
		Operations: []visual_learning.OperationSearchResult{
			{Tag: "demo-03677400", Title: "\u5728 Chrome \u4e2d\u64cd\u4f5c ChatGPT \u9801\u9762\u4e26\u95dc\u9589\u5206\u9801"},
		},
	}
	normalized := normalizeToolRoutingDecision(decision, "\u5e6b\u6211\u91cd\u73fe Chrome \u700f\u89bd\u5668\u4e2d\u9ede\u64ca ChatGPT \u9801\u9762\u5167\u5bb9\u5f8c\u95dc\u9589\u5206\u9801", lookup)
	if normalized.Action != "\u64cd\u4f5c" || normalized.Target != "demo-03677400" || normalized.Next != actionchain.StandbyNext {
		t.Fatalf("expected replay query to promote to operation, got %#v", normalized)
	}

	queryOnly := normalizeToolRoutingDecision(decision, "\u5e6b\u6211\u67e5\u8a62\u9304\u88fd\u7684\u64cd\u4f5c", lookup)
	if queryOnly.Action != "\u67e5\u8a62" || queryOnly.Next != "\u64cd\u4f5c" {
		t.Fatalf("query-only request should remain a query, got %#v", queryOnly)
	}
}

func TestNormalizeToolRoutingDecisionPromotesCurrentQuestionToWeb(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u641c\u5c0b", "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2", actionchain.StandbyNext))
	lookup := toolRoutingLookupContext{Query: "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2"}

	normalized := normalizeToolRoutingDecision(decision, "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2\u5982\u4f55", lookup)
	if normalized.Action != "\u7db2\u8def" || normalized.Target != "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2" || normalized.Next != actionchain.StandbyNext {
		t.Fatalf("expected current question to route to web search, got %#v", normalized)
	}
}

func TestNormalizeToolRoutingDecisionKeepsExplicitLocalSearchLocal(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u641c\u5c0b", "API key", actionchain.StandbyNext))
	lookup := toolRoutingLookupContext{Query: "API key"}

	normalized := normalizeToolRoutingDecision(decision, "\u5e6b\u6211\u672c\u6a5f\u641c\u5c0b API key", lookup)
	if normalized.Action != "\u641c\u5c0b" {
		t.Fatalf("explicit local search should remain local, got %#v", normalized)
	}
}
