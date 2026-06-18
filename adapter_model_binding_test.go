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
	foundGPT54 := false
	foundSpark := false
	for _, option := range options {
		if option == "gpt-5" || option == "gpt-5-codex" {
			t.Fatalf("unsupported codex model should not be listed: %#v", options)
		}
		if option == "o4" || option == "o4-mini" {
			t.Fatalf("stale codex fallback should not be listed: %#v", options)
		}
		if option == "gpt-5.4" {
			foundGPT54 = true
		}
		if option == "gpt-5.3-codex-spark" {
			foundSpark = true
		}
	}
	if !foundGPT54 {
		t.Fatalf("expected gpt-5.4 in codex options, got %#v", options)
	}
	if !foundSpark {
		t.Fatalf("expected gpt-5.3-codex-spark in codex options, got %#v", options)
	}
}

func TestListAdapterModelOptionsIncludesGemini35Flash(t *testing.T) {
	app := &App{}
	options := app.ListAdapterModelOptions("gemini-cli")
	if len(options) == 0 || options[0] != "gemini-3.5-flash" {
		t.Fatalf("expected gemini-3.5-flash first, got %#v", options)
	}
}

func TestParseCLIExecutableVersion(t *testing.T) {
	got := parseCLIExecutableVersion([]byte("codex-cli 0.121.0\n"))
	if got != "0.121.0" {
		t.Fatalf("parseCLIExecutableVersion() = %q, want %q", got, "0.121.0")
	}
}

func TestClassifyCodexCLIModelProbeOutputRejectsUnknownModel(t *testing.T) {
	raw := `2026-06-18T01:27:07Z WARN codex_models_manager::model_info: Unknown model o4-mini is used. This will use fallback model metadata.
model: o4-mini`
	if got := classifyCodexCLIModelProbeOutput(raw, "o4-mini"); got != codexModelUnsupported {
		t.Fatalf("classifyCodexCLIModelProbeOutput() = %v, want %v", got, codexModelUnsupported)
	}
}

func TestClassifyCodexCLIModelProbeOutputKeepsKnownModelOnAuthFailure(t *testing.T) {
	raw := `OpenAI Codex v0.121.0
model: gpt-5.5
ERROR codex_api::endpoint::responses_websocket: failed to connect to websocket: HTTP error: 401 Unauthorized`
	if got := classifyCodexCLIModelProbeOutput(raw, "gpt-5.5"); got != codexModelSupported {
		t.Fatalf("classifyCodexCLIModelProbeOutput() = %v, want %v", got, codexModelSupported)
	}
}

func TestParseCodexDebugModelsOutputKeepsVisibleModelsInPriorityOrder(t *testing.T) {
	raw := []byte(`warning before json
{"models":[
  {"slug":"codex-auto-review","visibility":"hide","priority":29},
  {"slug":"gpt-5.4-mini","visibility":"list","priority":4},
  {"slug":"gpt-5.5","visibility":"list","priority":0},
  {"slug":"gpt-5.4","visibility":"list","priority":2}
]}`)
	got, hidden, ok := parseCodexDebugModelsOutput(raw)
	if !ok {
		t.Fatalf("parseCodexDebugModelsOutput() reported failure")
	}
	want := []string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini"}
	if len(got) != len(want) {
		t.Fatalf("models = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("models = %#v, want %#v", got, want)
		}
	}
	if len(hidden) != 1 || hidden[0] != "codex-auto-review" {
		t.Fatalf("hidden = %#v, want %#v", hidden, []string{"codex-auto-review"})
	}
}

func TestAppendUniqueStringsKeepsExistingOrder(t *testing.T) {
	got := appendUniqueStrings([]string{"gpt-5.5", "gpt-5.4"}, "gpt-5.4", "gpt-5.3-codex-spark")
	want := []string{"gpt-5.5", "gpt-5.4", "gpt-5.3-codex-spark"}
	if len(got) != len(want) {
		t.Fatalf("values = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("values = %#v, want %#v", got, want)
		}
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
