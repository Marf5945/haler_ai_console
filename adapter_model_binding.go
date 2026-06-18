package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var adapterModelPresets = map[string][]string{
	"gemini-cli": {
		"gemini-3.5-flash",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.5-pro",
		"gemini-3.1-flash-lite",
		"gemini-3.1-pro-preview",
	},
	"claude-cli": {"sonnet", "opus", "haiku"},
	"codex-cli":  {"gpt-5.5", "o4", "o4-mini"},
}

type adapterModelCatalog struct {
	Options    []string
	Source     string
	CLIVersion string
	Note       string
}

func (a *App) GetAdapterModelChoices() map[string]string {
	if a.settingsService == nil {
		return map[string]string{}
	}
	choices := a.settingsService.AdapterModelChoices()
	for adapterID, model := range choices {
		normalized := normalizeAdapterModelChoice(adapterID, model)
		if normalized != model {
			a.settingsService.SaveAdapterModelChoice(adapterID, normalized)
			choices[adapterID] = normalized
		}
	}
	return choices
}

func (a *App) SetAdapterModelChoice(adapterID, model string) error {
	if a.settingsService == nil {
		return nil
	}
	a.settingsService.SaveAdapterModelChoice(strings.TrimSpace(adapterID), normalizeAdapterModelChoice(adapterID, model))
	return nil
}

func (a *App) ListAdapterModelOptions(adapterID string) []string {
	return a.describeAdapterModelCatalog(adapterID).Options
}

func (a *App) describeAdapterModelCatalog(adapterID string) adapterModelCatalog {
	id := strings.ToLower(strings.TrimSpace(adapterID))
	switch {
	case strings.HasPrefix(id, "local-ollama"):
		options := []string{}
		for _, model := range scanOllamaModels() {
			if model.ID != "" {
				options = append(options, model.ID)
			}
		}
		return adapterModelCatalog{
			Options: options,
			Source:  "runtime-scan",
			Note:    "Scanned from local Ollama models.",
		}
	case id == "gemini-cli":
		version, cliPath := a.adapterCLIExecutableVersion(id)
		if options := a.scanGeminiCLIModelOptions(id); len(options) > 0 {
			return adapterModelCatalog{
				Options:    options,
				Source:     "bundle-scan",
				CLIVersion: version,
				Note:       cliSourceNote(cliPath, "Read from the installed Gemini CLI bundle."),
			}
		}
		return adapterModelCatalog{
			Options:    copyStringSlice(adapterModelPresets[id]),
			Source:     "preset-fallback",
			CLIVersion: version,
			Note:       cliSourceNote(cliPath, "Gemini CLI bundle could not be inspected, so this is the app fallback list."),
		}
	case id == "codex-cli":
		version, cliPath := a.adapterCLIExecutableVersion(id)
		if options, hidden, ok := scanCodexCLIModelOptionsFromExecutable(cliPath, adapterModelPresets[id]); ok && len(options) > 0 {
			note := "Filtered against the installed Codex CLI."
			if len(hidden) > 0 {
				note = "Filtered against the installed Codex CLI. Hidden unsupported candidates: " + strings.Join(hidden, ", ")
			}
			return adapterModelCatalog{
				Options:    options,
				Source:     "live-probe",
				CLIVersion: version,
				Note:       cliSourceNote(cliPath, note),
			}
		}
		return adapterModelCatalog{
			Options:    copyStringSlice(adapterModelPresets[id]),
			Source:     "preset-fallback",
			CLIVersion: version,
			Note:       cliSourceNote(cliPath, "Codex CLI could not be probed, so this is the app fallback list."),
		}
	default:
		if presets, ok := adapterModelPresets[id]; ok {
			version, cliPath := a.adapterCLIExecutableVersion(id)
			return adapterModelCatalog{
				Options:    copyStringSlice(presets),
				Source:     "preset",
				CLIVersion: version,
				Note:       cliSourceNote(cliPath, "This adapter uses the app preset list because the CLI does not expose a model catalog."),
			}
		}
	}
	return adapterModelCatalog{}
}

func (a *App) adapterCLIExecutableVersion(adapterID string) (string, string) {
	if a == nil || a.adapterRegistry == nil {
		return "", ""
	}
	cliPath, err := a.adapterRegistry.ResolveExecutable(adapterID)
	if err != nil || strings.TrimSpace(cliPath) == "" {
		return "", ""
	}
	return scanCLIExecutableVersion(cliPath), cliPath
}

func (a *App) scanGeminiCLIModelOptions(adapterID string) []string {
	if a == nil || a.adapterRegistry == nil {
		return nil
	}
	cliPath, err := a.adapterRegistry.ResolveExecutable(adapterID)
	if err != nil || strings.TrimSpace(cliPath) == "" {
		return nil
	}
	return scanGeminiCLIModelOptionsFromExecutable(cliPath)
}

func scanGeminiCLIModelOptionsFromExecutable(cliPath string) []string {
	bundleRoots := geminiCLIBundleRoots(cliPath)
	seen := map[string]bool{}
	out := []string{}
	for _, root := range bundleRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
				continue
			}
			path := filepath.Join(root, entry.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			for _, model := range parseGeminiModelDefinitions(string(raw)) {
				if seen[model] {
					continue
				}
				seen[model] = true
				out = append(out, model)
			}
		}
	}
	sortGeminiModels(out)
	return out
}

func geminiCLIBundleRoots(cliPath string) []string {
	cliPath = strings.TrimSpace(cliPath)
	if cliPath == "" {
		return nil
	}
	dirs := []string{
		filepath.Join(filepath.Dir(cliPath), "..", "@google", "gemini-cli", "bundle"),
		filepath.Join(filepath.Dir(cliPath), "..", "..", "@google", "gemini-cli", "bundle"),
		filepath.Join(filepath.Dir(cliPath), "node_modules", "@google", "gemini-cli", "bundle"),
	}
	out := make([]string, 0, len(dirs))
	seen := map[string]bool{}
	for _, dir := range dirs {
		clean := filepath.Clean(dir)
		if clean == "." || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out
}

var geminiModelDefinitionRE = regexp.MustCompile(`"((?:auto-)?gemini-[0-9][A-Za-z0-9._-]*)"\s*:\s*\{(?s:[^{}]|\{[^{}]*\})*?isVisible:\s*true`)
var cliVersionRE = regexp.MustCompile(`\b(?:v)?(\d+\.\d+\.\d+(?:[-+][A-Za-z0-9._-]+)?)\b`)

func parseGeminiModelDefinitions(raw string) []string {
	matches := geminiModelDefinitionRE.FindAllStringSubmatch(raw, -1)
	out := []string{}
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		model := normalizeGeminiModelID(match[1])
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		out = append(out, model)
	}
	sortGeminiModels(out)
	return out
}

func normalizeGeminiModelID(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "models/")
	if model == "" || strings.Contains(model, "customtools") {
		return ""
	}
	return model
}

func sortGeminiModels(models []string) {
	sort.SliceStable(models, func(i, j int) bool {
		return geminiModelRank(models[i]) < geminiModelRank(models[j])
	})
}

func geminiModelRank(model string) string {
	m := strings.ToLower(model)
	tier := "2"
	switch {
	case strings.Contains(m, "flash"):
		tier = "0"
	case strings.Contains(m, "pro"):
		tier = "1"
	case strings.Contains(m, "auto"):
		tier = "9"
	}
	return tier + "|" + m
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cliSourceNote(cliPath, note string) string {
	note = strings.TrimSpace(note)
	if strings.TrimSpace(cliPath) == "" {
		return note
	}
	if note == "" {
		return filepath.Base(cliPath)
	}
	return note + " (" + filepath.Base(cliPath) + ")"
}

func scanCLIExecutableVersion(cliPath string) string {
	cliPath = strings.TrimSpace(cliPath)
	if cliPath == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliPath, "--version")
	cmd.Dir = codexProbeWorkdir()
	output, err := cmd.CombinedOutput()
	if len(output) == 0 && err != nil {
		return ""
	}
	return parseCLIExecutableVersion(output)
}

func parseCLIExecutableVersion(output []byte) string {
	match := cliVersionRE.FindSubmatch(output)
	if len(match) >= 2 {
		return string(match[1])
	}
	return strings.TrimSpace(string(output))
}

func scanCodexCLIModelOptionsFromExecutable(cliPath string, candidates []string) ([]string, []string, bool) {
	cliPath = strings.TrimSpace(cliPath)
	if cliPath == "" || len(candidates) == 0 {
		return nil, nil, false
	}
	supported := make([]string, 0, len(candidates))
	hidden := []string{}
	probed := false
	for _, candidate := range candidates {
		status, ok := probeCodexCLIModelSupport(cliPath, candidate)
		if !ok {
			continue
		}
		probed = true
		if status == codexModelUnsupported {
			hidden = append(hidden, candidate)
			continue
		}
		supported = append(supported, candidate)
	}
	return supported, hidden, probed
}

type codexModelSupport int

const (
	codexModelUnknown codexModelSupport = iota
	codexModelSupported
	codexModelUnsupported
)

func probeCodexCLIModelSupport(cliPath, model string) (codexModelSupport, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliPath, "exec", "--skip-git-repo-check", "--model", model, "probe")
	cmd.Dir = codexProbeWorkdir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	raw := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	if raw == "" && err == nil {
		return codexModelUnknown, false
	}
	if raw == "" {
		return codexModelUnknown, false
	}
	return classifyCodexCLIModelProbeOutput(raw, model), true
}

func classifyCodexCLIModelProbeOutput(raw, model string) codexModelSupport {
	joined := strings.ToLower(strings.TrimSpace(raw))
	modelLower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case joined == "":
		return codexModelUnknown
	case strings.Contains(joined, "unknown model "+modelLower):
		return codexModelUnsupported
	case strings.Contains(joined, "the '"+modelLower+"' model is not supported"):
		return codexModelUnsupported
	case strings.Contains(joined, "model: "+modelLower):
		return codexModelSupported
	case strings.Contains(joined, "missing bearer or basic authentication"):
		return codexModelSupported
	case strings.Contains(joined, "401 unauthorized"):
		return codexModelSupported
	case strings.Contains(joined, "reconnecting"):
		return codexModelSupported
	default:
		return codexModelUnknown
	}
}

func codexProbeWorkdir() string {
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		dir := filepath.Join(root, "Temp", "haler-ai-codex-probe")
		_ = os.MkdirAll(dir, 0o700)
		return dir
	}
	dir := filepath.Join(os.TempDir(), "haler-ai-codex-probe")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func normalizeAdapterModelChoice(adapterID, model string) string {
	id := strings.ToLower(strings.TrimSpace(adapterID))
	m := strings.TrimSpace(model)
	if id == "codex-cli" {
		switch strings.ToLower(m) {
		case "gpt-5", "gpt-5-codex":
			return "gpt-5.5"
		}
	}
	return m
}
