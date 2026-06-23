// persona_avatar/image_api.go — Image Generation API Provider（§10.4–§10.5）。
// 使用靜態 style preset + compact state trigger 組合 prompt。
// controller 靜態組合，不經 LLM。credential_ref 指向加密檔案。
package persona_avatar

import (
	"fmt"
	"strings"
)

// ──────────────────────────────────────────────
// 內建 Style Preset 範例
// ──────────────────────────────────────────────

// BuiltInPresets 提供預設的風格模板。
var BuiltInPresets = []StylePreset{
	{
		StylePresetID:  "cyberpunk_helper",
		Name:           "Cyberpunk Helper",
		PromptTemplate: "A small cyberpunk assistant avatar, clean UI portrait, expressive face, {state_prompt}, square icon, no text.",
		StatePrompts: map[AvatarStateTrigger]string{
			StateIdle:     "neutral and calm",
			StateThinking: "looking thoughtful",
			StateWorking:  "focused with subtle terminal glow",
			StateHappy:    "smiling warmly",
			StateWarning:  "slightly serious expression",
			StateBlocked:  "confused but polite",
			StateSleepy:   "drowsy with half-closed eyes",
		},
	},
	{
		StylePresetID:  "cute_animal",
		Name:           "Cute Animal",
		PromptTemplate: "A cute cartoon animal assistant, soft colors, round shapes, {state_prompt}, UI avatar, no text.",
		StatePrompts: map[AvatarStateTrigger]string{
			StateIdle:     "sitting calmly",
			StateThinking: "tilting head curiously",
			StateWorking:  "busily typing",
			StateHappy:    "jumping with joy",
			StateWarning:  "ears folded back cautiously",
			StateBlocked:  "paws up in confusion",
			StateSleepy:   "curled up napping",
		},
	},
	{
		StylePresetID:  "minimal_geometric",
		Name:           "Minimal Geometric",
		PromptTemplate: "A minimal geometric avatar, flat design, single accent color, {state_prompt}, clean background, no text.",
		StatePrompts: map[AvatarStateTrigger]string{
			StateIdle:     "balanced symmetry",
			StateThinking: "shifting angles",
			StateWorking:  "dynamic motion lines",
			StateHappy:    "warm golden accent",
			StateWarning:  "angular sharp edges",
			StateBlocked:  "fragmented pieces",
			StateSleepy:   "soft faded opacity",
		},
	},
}

// ──────────────────────────────────────────────
// Prompt 組合（§10.4 靜態模板）
// ──────────────────────────────────────────────

// ComposePrompt 用靜態 style preset 組合圖像生成 prompt。
// 禁止輸入：raw conversation, talk_full.md, private files, secrets 等。
// 僅接受：persona_id, style_preset_id, current_state_trigger。
func ComposePrompt(preset StylePreset, stateTrigger AvatarStateTrigger) (string, error) {
	statePrompt, ok := preset.StatePrompts[stateTrigger]
	if !ok {
		// fallback 到 idle
		statePrompt = preset.StatePrompts[StateIdle]
		if statePrompt == "" {
			return "", fmt.Errorf("style preset %s 缺少 idle 狀態 prompt", preset.StylePresetID)
		}
	}

	prompt := strings.Replace(preset.PromptTemplate, "{state_prompt}", statePrompt, 1)
	return prompt, nil
}

// GetPresetByID 根據 ID 取得 style preset。
func GetPresetByID(presetID string) (*StylePreset, error) {
	for _, p := range BuiltInPresets {
		if p.StylePresetID == presetID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("style preset not found: %s", presetID)
}

// ListPresets 列出所有可用的 style presets。
func ListPresets() []StylePreset {
	result := make([]StylePreset, len(BuiltInPresets))
	copy(result, BuiltInPresets)
	return result
}

// ──────────────────────────────────────────────
// API 呼叫 stub（實際 HTTP 呼叫由呼叫端自行實作）
// ──────────────────────────────────────────────

// GenerateAvatarRequest 是呼叫圖像生成 API 的請求結構。
// 實際 HTTP 呼叫由呼叫端負責，此結構提供所需參數。
type GenerateAvatarRequest struct {
	Prompt      string `json:"prompt"`
	APIEndpoint string `json:"api_endpoint"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// PrepareGenerateRequest 組合圖像生成 API 請求。
// 回傳 request 結構，呼叫端需自行處理 HTTP + credential。
func (s *Service) PrepareGenerateRequest(personaID string, stateTrigger AvatarStateTrigger) (*GenerateAvatarRequest, error) {
	config := s.GetCurrentAvatar(personaID)

	if config.AvatarProvider != ProviderImageAPI {
		return nil, fmt.Errorf("persona %s 未設定為 Image API provider", personaID)
	}

	preset, err := GetPresetByID(config.StylePresetID)
	if err != nil {
		return nil, err
	}

	prompt, err := ComposePrompt(*preset, stateTrigger)
	if err != nil {
		return nil, err
	}

	size := config.OutputSize
	if size <= 0 {
		size = 256
	}

	return &GenerateAvatarRequest{
		Prompt:      prompt,
		APIEndpoint: config.APIEndpoint,
		Width:       size,
		Height:      size,
	}, nil
}
