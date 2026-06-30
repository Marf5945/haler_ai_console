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

// 註：原本的「Image API 請求 + 憑證」stub 已移除。頭像產圖統一改走
// app 層的 GenerateAvatarViaImageGen（與紀念照共用同一個 imagegen 引擎），
// 不再有獨立的 APIEndpoint / credential 接口。上方的 style preset 與
// ComposePrompt 仍保留，供未來模板選擇之用。
