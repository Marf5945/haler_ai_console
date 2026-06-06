package main

import "testing"

func TestNormalizeAdapterModelChoiceUpgradesUnsupportedCodexAliases(t *testing.T) {
	cases := map[string]string{
		"gpt-5":       "gpt-5.5",
		"gpt-5-codex": "gpt-5.5",
		"gpt-5.5":     "gpt-5.5",
		"o4":          "o4",
	}
	for input, want := range cases {
		if got := normalizeAdapterModelChoice("codex-cli", input); got != want {
			t.Fatalf("normalizeAdapterModelChoice(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestListAdapterModelOptionsUsesCurrentCodexModelIDs(t *testing.T) {
	app := &App{}
	options := app.ListAdapterModelOptions("codex-cli")
	if len(options) == 0 || options[0] != "gpt-5.5" {
		t.Fatalf("expected gpt-5.5 first, got %#v", options)
	}
	for _, option := range options {
		if option == "gpt-5" || option == "gpt-5-codex" {
			t.Fatalf("unsupported codex model should not be listed: %#v", options)
		}
	}
}
