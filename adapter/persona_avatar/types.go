// Package persona_avatar 實作 AI Console v3.6.1 的人格圖像系統（§10）。
// 三層 provider：Image API > Static Image > Built-in Pixel。
// 圖像屬於 UI 呈現層，不得影響風險、信任、路由、記憶等系統決策。
package persona_avatar

// ──────────────────────────────────────────────
// Avatar Provider（§10.2 優先順序）
// ──────────────────────────────────────────────

// AvatarProvider 描述頭像來源類型。
type AvatarProvider string

const (
	ProviderImageAPI    AvatarProvider = "user_image_api"
	ProviderStaticImage AvatarProvider = "static_image"
	ProviderPixel       AvatarProvider = "built_in_pixel"
)

// ──────────────────────────────────────────────
// 狀態觸發器（§10.3 單向映射）
// ──────────────────────────────────────────────

// AvatarStateTrigger 描述 avatar 表情狀態。
// 系統狀態 → avatar 表情（單向），avatar 不得回寫任何系統狀態。
type AvatarStateTrigger string

const (
	StateIdle       AvatarStateTrigger = "idle"
	StateThinking   AvatarStateTrigger = "thinking"
	StateWorking    AvatarStateTrigger = "working"
	StateHappy      AvatarStateTrigger = "happy"
	StateWarning    AvatarStateTrigger = "warning"
	StateBlocked    AvatarStateTrigger = "blocked"
	StateSleepy     AvatarStateTrigger = "sleepy"
	StateSad        AvatarStateTrigger = "sad"
	StateSpeechless AvatarStateTrigger = "speechless"
)

// AllStateTriggers 回傳所有支援的狀態觸發器。
var AllStateTriggers = []AvatarStateTrigger{
	StateIdle, StateThinking, StateWorking, StateHappy,
	StateWarning, StateBlocked, StateSleepy, StateSad, StateSpeechless,
}

// ──────────────────────────────────────────────
// Avatar 設定（§10.12 avatar_config.json）
// ──────────────────────────────────────────────

// AvatarConfig 對應 data/personas/[id]/avatar/avatar_config.json。
type AvatarConfig struct {
	AvatarProvider    AvatarProvider `json:"avatar_provider"`
	PersonaID         string         `json:"persona_id"`
	PixelPack         string         `json:"pixel_pack,omitempty"`
	StaticAvatarPath  string         `json:"static_avatar_path,omitempty"`
	OriginalImagePath string         `json:"original_image_path,omitempty"`
	Crop              *CropRect      `json:"crop,omitempty"`
	OutputSize        int            `json:"output_size"`
	UpdatedAt         string         `json:"updated_at"`

	// Image API 設定（僅 provider=user_image_api 時使用）
	StylePresetID string `json:"style_preset_id,omitempty"`
	CredentialRef string `json:"credential_ref,omitempty"`
	APIEndpoint   string `json:"api_endpoint,omitempty"`
}

// CropRect 描述裁切區域。
type CropRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ──────────────────────────────────────────────
// Style Preset（§10.4 Image API 省 token 設計）
// ──────────────────────────────────────────────

// StylePreset 定義圖像生成 API 的風格模板。
// prompt 由 controller 靜態組合，不經 LLM。
type StylePreset struct {
	StylePresetID  string                        `json:"style_preset_id"`
	Name           string                        `json:"name"`
	PromptTemplate string                        `json:"prompt_template"` // 含 {state_prompt} 佔位符
	StatePrompts   map[AvatarStateTrigger]string `json:"state_prompts"`
}
