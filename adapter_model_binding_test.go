package main

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestListAdapterModelOptionsIncludesGemini35Flash(t *testing.T) {
	app := &App{}
	options := app.ListAdapterModelOptions("gemini-cli")
	if len(options) == 0 || options[0] != "gemini-3.5-flash" {
		t.Fatalf("expected gemini-3.5-flash first, got %#v", options)
	}
}

func TestParseGeminiModelDefinitionsKeepsVisibleConcreteModels(t *testing.T) {
	raw := `
modelDefinitions: {
  "gemini-3.5-flash": {
    tier: "flash",
    family: "gemini-3",
    isPreview: false,
    isVisible: true,
    features: { thinking: false, multimodalToolUse: true }
  },
  "gemini-3.5-pro": {
    tier: "pro",
    family: "gemini-3",
    isPreview: false,
    isVisible: true,
    features: { thinking: true }
  },
  "gemini-3.5-pro-customtools": {
    tier: "pro",
    isVisible: true
  },
  "gemini-3.5-internal": {
    tier: "flash",
    isVisible: false
  }
}`
	got := parseGeminiModelDefinitions(raw)
	want := []string{"gemini-3.5-flash", "gemini-3.5-pro"}
	if len(got) != len(want) {
		t.Fatalf("models = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("models = %#v, want %#v", got, want)
		}
	}
}

func TestScanGeminiCLIModelOptionsFromExecutableReadsBundle(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "node_modules", ".bin")
	bundleDir := filepath.Join(root, "node_modules", "@google", "gemini-cli", "bundle")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bundleDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cliPath := filepath.Join(binDir, "gemini")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "chunk-test.js"), []byte(`
"gemini-3.5-pro": { tier: "pro", isVisible: true, features: { thinking: true } },
"gemini-3.5-flash": { tier: "flash", isVisible: true, features: { thinking: false } },
`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := scanGeminiCLIModelOptionsFromExecutable(cliPath)
	want := []string{"gemini-3.5-flash", "gemini-3.5-pro"}
	if len(got) != len(want) {
		t.Fatalf("models = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("models = %#v, want %#v", got, want)
		}
	}
}
