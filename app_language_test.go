package main

import (
	"strings"
	"testing"

	"ui_console/shared/localsearch"
)

func TestFormatLocalSearchOutcomeLanguage(t *testing.T) {
	outcome := localsearch.SearchOutcome{Results: []localsearch.SearchResult{{Snippet: "Python variables"}}}
	req := localsearch.SearchRequest{Query: "Python"}

	en := (&App{}).formatLocalSearchOutcomeForLanguage(req, outcome, responseLanguageEN)
	if !strings.Contains(en, "Local search found 1 result") || strings.Contains(en, "本機搜尋") {
		t.Fatalf("English local search format should be English, got %q", en)
	}

	pt := (&App{}).formatLocalSearchOutcomeForLanguage(req, outcome, responseLanguagePT)
	if !strings.Contains(pt, "A pesquisa local encontrou 1 resultado") || strings.Contains(pt, "本機搜尋") {
		t.Fatalf("Portuguese local search format should be Portuguese, got %q", pt)
	}
}
