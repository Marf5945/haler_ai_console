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
//   - Gemini/Claude/Codex 用輕量候選清單；Ollama 走 scanOllamaModels() 動態。
//   - 無候選清單時回 nil，前端負責不渲染 badge。
package main

import "strings"

// adapterModelPresets — CLI kind adapter 的 model 彈窗候選。
// 想加新 CLI 或換 model 名稱，改這一個 map 就好，不必動其他地方。
var adapterModelPresets = map[string][]string{
	"gemini-cli": {
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
//   - 已知 CLI adapter（gemini/claude/codex）→ adapterModelPresets
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
	if presets, ok := adapterModelPresets[id]; ok {
		// 回 copy，避免前端誤改 caller 共用的 slice。
		out := make([]string, len(presets))
		copy(out, presets)
		return out
	}
	return nil
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
