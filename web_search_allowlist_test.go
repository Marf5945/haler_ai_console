package main

import "testing"

func TestWebSearchAllowlistEntryApplies(t *testing.T) {
	if !webSearchAllowlistEntryApplies([]string{"web_search"}, nil) {
		t.Fatal("web_search allowed_for should apply")
	}
	if !webSearchAllowlistEntryApplies([]string{"rag_ranking"}, nil) {
		t.Fatal("rag_ranking source allowlist should apply to web search results")
	}
	if webSearchAllowlistEntryApplies([]string{"web_search"}, []string{"web_search"}) {
		t.Fatal("not_allowed_for web_search should block the entry")
	}
	if webSearchAllowlistEntryApplies([]string{"adapter_permission"}, nil) {
		t.Fatal("unrelated allowed_for should not apply")
	}
}
