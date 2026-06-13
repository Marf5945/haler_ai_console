// adapter_model_binding.go — UI 雙擊 adapter 卡片開啟 model 選單的 Wails binding。
//
// 流程：
//
//	UI 雙擊 → ListAdapterModelOptions 拿候選清單 → 彈窗點選 → SetAdapterModelChoice 寫入
//	settings.adapterModelChoices → 下次 sendMessage 時 sidecarAdapter.modelProvider
//	查詢這份 map → 把 model 塞進 sidecar IPC → commandFor 加 --model 旗標。
//
// 設計原則：
//   - 不抽 helper struct；只是 3 個 thin forwarding method。
//   - Gemini 優先掃 CLI bundle；Claude/Codex 用輕量候選清單；Ollama 走 scanOllamaModels() 動態。
//   - 無候選清單時回 nil，前端負責不渲染 badge。
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// adapterModelPresets — CLI kind adapter 的 model 彈窗候選。
// 想加新 CLI 或換 model 名稱，改這一個 map 就好，不必動其他地方。
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

// GetAdapterModelChoices 回傳目前所有 adapter 的 model 偏好。
// 前端開機時呼叫一次塞進 state，之後跟著 SetAdapterModelChoice 同步即可。
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

// SetAdapterModelChoice 寫入單一 adapter 的 model 偏好。
// model 為空字串時等同清除（fallback 走 CLI 自身預設或 commandFor 內的 hardcoded）。
func (a *App) SetAdapterModelChoice(adapterID, model string) error {
	if a.settingsService == nil {
		return nil
	}
	a.settingsService.SaveAdapterModelChoice(strings.TrimSpace(adapterID), normalizeAdapterModelChoice(adapterID, model))
	return nil
}

// ListAdapterModelOptions 回傳指定 adapter 的雙擊循環候選清單。
// 規則：
//   - "local-ollama-*" → 即時掃描 `ollama list`
//   - gemini-cli → 從 Gemini CLI bundle 的 modelDefinitions 掃描；失敗才用 adapterModelPresets
//   - 其他已知 CLI adapter（claude/codex）→ adapterModelPresets
//   - 其他（API / 未知）→ nil（前端不渲染 badge、不開啟雙擊）
func (a *App) ListAdapterModelOptions(adapterID string) []string {
	id := strings.ToLower(strings.TrimSpace(adapterID))
	if strings.HasPrefix(id, "local-ollama") {
		out := []string{}
		for _, m := range scanOllamaModels() {
			if m.ID != "" {
				out = append(out, m.ID)
			}
		}
		return out
	}
	if id == "gemini-cli" {
		if out := a.scanGeminiCLIModelOptions(id); len(out) > 0 {
			return out
		}
	}
	if presets, ok := adapterModelPresets[id]; ok {
		// 回 copy，避免前端誤改 caller 共用的 slice。
		out := make([]string, len(presets))
		copy(out, presets)
		return out
	}
	return nil
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
