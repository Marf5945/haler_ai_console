package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"ui_console/shared/executil"
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
	"claude-cli": {"sonnet", "opus", "haiku", "fable"},
	"codex-cli":  {"gpt-5.5", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex", "gpt-5.3-codex-spark", "gpt-5.2"},
}

var codexLegacyModelAliases = []string{
	"gpt-5.3-codex-spark",
}

var codexDebugModelCache = struct {
	sync.Mutex
	entries map[string]codexCLIModelScanResult
}{
	entries: map[string]codexCLIModelScanResult{},
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
	case id == "claude-cli":
		version, cliPath := a.adapterCLIExecutableVersion(id)
		if options := a.scanClaudeCLIModelOptions(id); len(options) > 0 {
			options = appendUniqueStrings(options, adapterModelPresets[id]...)
			return adapterModelCatalog{
				Options:    options,
				Source:     "bundle-scan",
				CLIVersion: version,
				Note:       cliSourceNote(cliPath, "Read from the installed Claude CLI bundle."),
			}
		}
		return adapterModelCatalog{
			Options:    copyStringSlice(adapterModelPresets[id]),
			Source:     "preset-fallback",
			CLIVersion: version,
			Note:       cliSourceNote(cliPath, "Claude CLI bundle could not be inspected, so this is the app fallback list."),
		}
	case id == "codex-cli":
		version, cliPath := a.adapterCLIExecutableVersion(id)
		if options, hidden, ok := scanCodexCLIModelOptionsFromDebugCommand(cliPath, version); ok && len(options) > 0 {
			if aliases, aliasHidden, aliasOK := scanCodexCLIAdditionalModelAliases(cliPath, options); aliasOK && len(aliases) > 0 {
				options = appendUniqueStrings(options, aliases...)
				hidden = appendUniqueStrings(hidden, aliasHidden...)
			}
			note := "Read from `codex debug models` of the installed Codex CLI."
			if len(hidden) > 0 {
				note = "Read from `codex debug models` of the installed Codex CLI. Hidden internal models: " + strings.Join(hidden, ", ")
			}
			return adapterModelCatalog{
				Options:    options,
				Source:     "debug-models",
				CLIVersion: version,
				Note:       cliSourceNote(cliPath, note),
			}
		}
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

func (a *App) scanClaudeCLIModelOptions(adapterID string) []string {
	if a == nil || a.adapterRegistry == nil {
		return nil
	}
	cliPath, err := a.adapterRegistry.ResolveExecutable(adapterID)
	if err != nil || strings.TrimSpace(cliPath) == "" {
		return nil
	}
	return scanClaudeCLIModelOptionsFromExecutable(cliPath)
}

func scanClaudeCLIModelOptionsFromExecutable(cliPath string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, root := range claudeCLIBundleRoots(cliPath) {
		for _, name := range []string{"sdk-tools.d.ts", "sdk.d.ts", "index.d.ts", "cli.js", "index.js"} {
			path := filepath.Join(root, name)
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			for _, model := range parseClaudeModelDefinitions(string(raw)) {
				if seen[model] {
					continue
				}
				seen[model] = true
				out = append(out, model)
			}
		}
	}
	sortClaudeModels(out)
	return out
}

func claudeCLIBundleRoots(cliPath string) []string {
	cliPath = strings.TrimSpace(cliPath)
	if cliPath == "" {
		return nil
	}
	base := filepath.Dir(cliPath)
	dirs := []string{
		filepath.Join(base, "node_modules", "@anthropic-ai", "claude-code"),
		filepath.Join(base, "..", "node_modules", "@anthropic-ai", "claude-code"),
		filepath.Join(base, "..", "..", "node_modules", "@anthropic-ai", "claude-code"),
		filepath.Join(base, "..", "lib", "node_modules", "@anthropic-ai", "claude-code"),
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
var claudeModelUnionRE = regexp.MustCompile(`model\??:\s*((?:"[A-Za-z0-9._-]+"\s*\|\s*)+"[A-Za-z0-9._-]+")`)
var quotedClaudeModelRE = regexp.MustCompile(`"([A-Za-z0-9._-]+)"`)
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

func parseClaudeModelDefinitions(raw string) []string {
	matches := claudeModelUnionRE.FindAllStringSubmatch(raw, -1)
	out := []string{}
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		for _, quoted := range quotedClaudeModelRE.FindAllStringSubmatch(match[1], -1) {
			if len(quoted) < 2 {
				continue
			}
			model := normalizeClaudeModelID(quoted[1])
			if model == "" || seen[model] {
				continue
			}
			seen[model] = true
			out = append(out, model)
		}
	}
	sortClaudeModels(out)
	return out
}

func normalizeClaudeModelID(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	switch model {
	case "sonnet", "opus", "haiku", "fable":
		return model
	default:
		if strings.HasPrefix(model, "claude-") {
			return strings.TrimPrefix(model, "claude-")
		}
	}
	return ""
}

func sortGeminiModels(models []string) {
	sort.SliceStable(models, func(i, j int) bool {
		return geminiModelRank(models[i]) < geminiModelRank(models[j])
	})
}

func sortClaudeModels(models []string) {
	sort.SliceStable(models, func(i, j int) bool {
		return claudeModelRank(models[i]) < claudeModelRank(models[j])
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

func claudeModelRank(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "sonnet":
		return "0"
	case "opus":
		return "1"
	case "haiku":
		return "2"
	case "fable":
		return "3"
	default:
		return "9|" + strings.ToLower(strings.TrimSpace(model))
	}
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func appendUniqueStrings(base []string, values ...string) []string {
	if len(values) == 0 {
		return base
	}
	seen := make(map[string]bool, len(base)+len(values))
	out := make([]string, 0, len(base)+len(values))
	for _, value := range base {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
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
	cmd := executil.CommandContext(ctx, cliPath, "--version")
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

type codexCLIModelScanResult struct {
	Options []string
	Hidden  []string
}

type codexDebugModelsResponse struct {
	Models []codexDebugModelInfo `json:"models"`
}

type codexDebugModelInfo struct {
	Slug       string `json:"slug"`
	Visibility string `json:"visibility"`
	Priority   int    `json:"priority"`
}

func scanCodexCLIModelOptionsFromDebugCommand(cliPath, cliVersion string) ([]string, []string, bool) {
	cliPath = strings.TrimSpace(cliPath)
	if cliPath == "" {
		return nil, nil, false
	}
	cacheKey := codexDebugModelCacheKey(cliPath, cliVersion)
	if cached, ok := loadCodexDebugModelCache(cacheKey); ok {
		return cached.Options, cached.Hidden, true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	cmd := executil.CommandContext(ctx, cliPath, "debug", "models")
	cmd.Dir = codexProbeWorkdir()
	output, err := cmd.CombinedOutput()
	if len(output) == 0 || err != nil {
		return nil, nil, false
	}
	options, hidden, ok := parseCodexDebugModelsOutput(output)
	if !ok || len(options) == 0 {
		return nil, nil, false
	}
	storeCodexDebugModelCache(cacheKey, codexCLIModelScanResult{Options: options, Hidden: hidden})
	return options, hidden, true
}

func loadCodexDebugModelCache(key string) (codexCLIModelScanResult, bool) {
	codexDebugModelCache.Lock()
	defer codexDebugModelCache.Unlock()
	entry, ok := codexDebugModelCache.entries[key]
	if !ok {
		return codexCLIModelScanResult{}, false
	}
	return codexCLIModelScanResult{
		Options: copyStringSlice(entry.Options),
		Hidden:  copyStringSlice(entry.Hidden),
	}, true
}

func storeCodexDebugModelCache(key string, entry codexCLIModelScanResult) {
	codexDebugModelCache.Lock()
	defer codexDebugModelCache.Unlock()
	codexDebugModelCache.entries[key] = codexCLIModelScanResult{
		Options: copyStringSlice(entry.Options),
		Hidden:  copyStringSlice(entry.Hidden),
	}
}

func codexDebugModelCacheKey(cliPath, cliVersion string) string {
	if cliVersion == "" {
		return cliPath
	}
	return cliPath + "|" + cliVersion
}

func parseCodexDebugModelsOutput(raw []byte) ([]string, []string, bool) {
	payload := bytes.TrimSpace(raw)
	start := bytes.IndexByte(payload, '{')
	end := bytes.LastIndexByte(payload, '}')
	if start < 0 || end <= start {
		return nil, nil, false
	}
	payload = payload[start : end+1]
	var decoded codexDebugModelsResponse
	if err := json.Unmarshal(payload, &decoded); err != nil || len(decoded.Models) == 0 {
		return nil, nil, false
	}
	models := make([]codexDebugModelInfo, len(decoded.Models))
	copy(models, decoded.Models)
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].Priority == models[j].Priority {
			return strings.ToLower(models[i].Slug) < strings.ToLower(models[j].Slug)
		}
		return models[i].Priority < models[j].Priority
	})
	options := []string{}
	hidden := []string{}
	seenOptions := map[string]bool{}
	seenHidden := map[string]bool{}
	for _, model := range models {
		slug := strings.TrimSpace(model.Slug)
		switch strings.ToLower(strings.TrimSpace(model.Visibility)) {
		case "list":
			if slug == "" || seenOptions[slug] {
				continue
			}
			seenOptions[slug] = true
			options = append(options, slug)
		case "hide":
			if slug == "" || seenHidden[slug] {
				continue
			}
			seenHidden[slug] = true
			hidden = append(hidden, slug)
		}
	}
	if len(options) == 0 {
		return nil, nil, false
	}
	return options, hidden, true
}

func scanCodexCLIAdditionalModelAliases(cliPath string, listed []string) ([]string, []string, bool) {
	if strings.TrimSpace(cliPath) == "" || len(codexLegacyModelAliases) == 0 {
		return nil, nil, false
	}
	seen := map[string]bool{}
	for _, model := range listed {
		model = strings.TrimSpace(model)
		if model != "" {
			seen[model] = true
		}
	}
	candidates := []string{}
	for _, model := range codexLegacyModelAliases {
		model = strings.TrimSpace(model)
		if model == "" || seen[model] {
			continue
		}
		candidates = append(candidates, model)
	}
	if len(candidates) == 0 {
		return nil, nil, false
	}
	return scanCodexCLIModelOptionsFromExecutable(cliPath, candidates)
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
	cmd := executil.CommandContext(ctx, cliPath, "exec", "--skip-git-repo-check", "--model", model, "probe")
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
