package main

import (
	"strings"
	"testing"

	"ui_console/shared/localsearch"
	"ui_console/shared/settings"
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

func TestResponseLanguageRecognizesJapaneseAndKorean(t *testing.T) {
	cases := []struct {
		label string
		want  string
		rule  string
		field string
	}{
		{label: "日本語", want: responseLanguageJA, rule: "回答內容請使用日本語", field: "語言=日本語"},
		{label: "한국어", want: responseLanguageKO, rule: "回答內容請使用한국어", field: "語言=한국어"},
	}
	for _, c := range cases {
		root := t.TempDir()
		ui := settings.NewUISettingsService(root)
		if _, err := ui.ApplyStyleDiff(`{"panel_language":"繁中","role_language":"` + c.label + `"}`); err != nil {
			t.Fatalf("ApplyStyleDiff(%q): %v", c.label, err)
		}
		app := &App{uiSettingsService: ui}
		if got := app.responseLanguage(); got != c.want {
			t.Fatalf("responseLanguage(%q) = %q, want %q", c.label, got, c.want)
		}
		if got := app.routingReplyLanguageRule(); got != c.rule {
			t.Fatalf("routingReplyLanguageRule(%q) = %q, want %q", c.label, got, c.rule)
		}
		if got := app.replyLanguageField(); !strings.Contains(got, c.field) {
			t.Fatalf("replyLanguageField(%q) = %q, want containing %q", c.label, got, c.field)
		}
	}
}
