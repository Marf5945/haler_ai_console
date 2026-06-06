package main

import (
	"strings"
	"testing"

	"ui_console/shared/localsearch"
)

func resetLocalSearchWebFallbackForTest() {
	pendingLocalSearchWebFallbackMu.Lock()
	pendingLocalSearchWebFallbacks = map[string]pendingLocalSearchWebFallback{}
	pendingLocalSearchWebFallbackMu.Unlock()
}

func TestLocalSearchNoResultsAsksForWebFallback(t *testing.T) {
	resetLocalSearchWebFallbackForTest()
	app := &App{}
	sessionID := "session-local-fallback"
	query := "unlikely-local-search-query-7f6b2d4e"

	resp := app.executeLocalSearch(localsearch.SearchRequest{Query: query}, sessionID, "trace-local-fallback")
	if !strings.Contains(resp.Text, query) || !strings.Contains(resp.Text, "網路搜尋") {
		t.Fatalf("expected web fallback question, got %q", resp.Text)
	}

	pendingLocalSearchWebFallbackMu.Lock()
	pending, ok := pendingLocalSearchWebFallbacks[sessionID]
	pendingLocalSearchWebFallbackMu.Unlock()
	if !ok {
		t.Fatal("expected pending web fallback")
	}
	if pending.Query != query {
		t.Fatalf("pending query = %q, want %q", pending.Query, query)
	}
}

func TestPendingLocalSearchWebFallbackCanBeCancelled(t *testing.T) {
	resetLocalSearchWebFallbackForTest()
	sessionID := "session-local-fallback-cancel"
	rememberLocalSearchWebFallback(sessionID, localsearch.SearchRequest{Query: "C# 教學"})

	app := &App{}
	resp, handled := app.maybeHandlePendingLocalSearchWebFallback("不用", sessionID, "trace-cancel")
	if !handled {
		t.Fatal("expected cancellation to be handled")
	}
	if !strings.Contains(resp.Text, "不改用網路搜尋") {
		t.Fatalf("unexpected cancellation response: %q", resp.Text)
	}

	pendingLocalSearchWebFallbackMu.Lock()
	_, ok := pendingLocalSearchWebFallbacks[sessionID]
	pendingLocalSearchWebFallbackMu.Unlock()
	if ok {
		t.Fatal("pending fallback should be cleared after cancellation")
	}
}
