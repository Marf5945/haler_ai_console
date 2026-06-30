// Package imagegen 定義「外接產圖軟體」的統一介面與提示詞組裝。
//
// 介面 ImageGenAdapter 讓後端可插拔：本輪實作本機 ComfyUI（comfyui.go），
// 之後要接雲端 API（DALL·E / 即夢 / Gemini）只要再寫一個實作即可，
// app 層與相冊邏輯都不需改動。
//
// 設計原則（對齊 repo）：純函式優先、可注入 http.Client（egress 由呼叫端用
// urlsafe.NewSafeClient(PolicyLocalLLM,...) 控制，本套件不自行決定網路政策）。
package imagegen

import (
	"context"
	"strings"
)

// Request 是一次產圖請求。零值欄位由 adapter 套用合理預設。
type Request struct {
	Positive string // 正向提示詞
	Negative string // 負向提示詞
	Width    int
	Height   int
	Seed     int64 // <0 表示讓 adapter 隨機
	Steps    int
	CFG      float64
	Sampler  string
	Model    string // checkpoint 名稱；空字串用 adapter 預設
}

// Result 是產圖結果。PNG 為原始位元組（不落地由呼叫端決定）。
type Result struct {
	PNG   []byte
	Seed  int64
	Model string
}

// Adapter 是所有產圖後端的統一介面。
type Adapter interface {
	// Name 回傳後端識別字（例 "comfyui"）。
	Name() string
	// Health 確認後端可用（例 ComfyUI /system_stats 可達）。
	Health(ctx context.Context) error
	// Generate 執行一次 txt2img，回傳 PNG。
	Generate(ctx context.Context, req Request) (Result, error)
}

// SceneInput 是「把對話場景轉成提示詞」的輸入。
type SceneInput struct {
	PersonaIdentity    string // 人格的外觀/身分描述（settings.Persona.Identity）
	PersonaPersonality string // 人格個性（可選，補氣氛）
	Scene              string // LLM 提議的場景描述（紀念照的 target）
	StylePreset        string // 全域畫風前綴，可空
}

// 預設畫風與負向詞：偏向乾淨的角色立繪/合照風，可被設定覆蓋。
const (
	DefaultStylePreset = "masterpiece, best quality, highly detailed, soft lighting, anime style portrait"
	DefaultNegative    = "lowres, bad anatomy, bad hands, text, error, missing fingers, extra digit, " +
		"fewer digits, cropped, worst quality, low quality, jpeg artifacts, signature, watermark, blurry"
)

// BuildScenePrompt 把人格外觀 + 場景描述組成正向提示詞。
// 順序：畫風前綴 → 人格外觀 → 場景 → 個性氛圍，去除空段並以逗號相連。
func BuildScenePrompt(in SceneInput) string {
	parts := []string{
		firstNonEmpty(in.StylePreset, DefaultStylePreset),
		strings.TrimSpace(in.PersonaIdentity),
		strings.TrimSpace(in.Scene),
		strings.TrimSpace(in.PersonaPersonality),
	}
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, ", ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
