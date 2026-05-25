// remote_bridge/preset.go — Platform Preset 定義（§12A.2.1）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ Preset 是設定模板，將平台特定的 URL/Header/Body 格式       │
// │ 封裝為可填入 generic sender 的組態。                        │
// │                                                             │
// │ 新增平台只需加一筆 preset 資料，不需改 sender 核心。       │
// │ 內建 preset：Telegram / Discord / LINE / Teams             │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"fmt"
	"strings"
)

// ──────────────────────────────────────────────
// Preset 結構
// ──────────────────────────────────────────────

// PlatformPreset 平台預設組態模板。
type PlatformPreset struct {
	PlatformID       string            `json:"platform_id"`        // "telegram" | "discord" | "line" | "slack" | "ntfy" | "custom"
	URLTemplate      string            `json:"url_template"`       // 含 {{token}}、{{chat_id}} 等佔位符
	Method           string            `json:"method"`             // 預設 "POST"
	HeadersTemplate  map[string]string `json:"headers_template"`   // header 模板
	BodyTemplate     string            `json:"body_template"`      // Go template 格式
	MaxLength        int               `json:"max_length"`         // 平台字數上限
	RequiredFields   []string          `json:"required_fields"`    // 使用者必填欄位
	AutoDetectFields []string          `json:"auto_detect_fields"` // 系統可自動偵測的欄位
}

// ──────────────────────────────────────────────
// 內建 Preset 定義
// ──────────────────────────────────────────────

var presets = map[string]PlatformPreset{
	"telegram": {
		PlatformID:       "telegram",
		URLTemplate:      "https://api.telegram.org/bot{{bot_token}}/sendMessage",
		Method:           "POST",
		HeadersTemplate:  map[string]string{"Content-Type": "application/json"},
		BodyTemplate:     `{"chat_id":"{{chat_id}}","text":"{{.Content}}","parse_mode":"HTML"}`,
		MaxLength:        4096,
		RequiredFields:   []string{"bot_token"},
		AutoDetectFields: []string{"chat_id"},
	},
	"discord": {
		PlatformID:       "discord",
		URLTemplate:      "https://discord.com/api/v10/channels/{{channel_id}}/messages",
		Method:           "POST",
		HeadersTemplate:  map[string]string{"Content-Type": "application/json", "Authorization": "Bot {{bot_token}}"},
		BodyTemplate:     `{"content":"{{.Content}}","allowed_mentions":{"parse":[]}}`,
		MaxLength:        2000,
		RequiredFields:   []string{"bot_token", "guild_id", "channel_id"},
		AutoDetectFields: []string{},
	},
	"line": {
		PlatformID:  "line",
		URLTemplate: "https://api.line.me/v2/bot/message/push",
		Method:      "POST",
		HeadersTemplate: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer {{channel_access_token}}",
		},
		BodyTemplate:     `{"to":"{{recipient_id}}","messages":[{"type":"text","text":"{{.Content}}"}]}`,
		MaxLength:        5000,
		RequiredFields:   []string{"channel_access_token", "recipient_id"},
		AutoDetectFields: []string{},
	},
	"teams": {
		PlatformID:       "teams",
		URLTemplate:      "{{webhook_url}}",
		Method:           "POST",
		HeadersTemplate:  map[string]string{"Content-Type": "application/json"},
		BodyTemplate:     `{"text":"{{.Content}}"}`,
		MaxLength:        28000,
		RequiredFields:   []string{"webhook_url"},
		AutoDetectFields: []string{},
	},
	"qq": {
		PlatformID:  "qq",
		URLTemplate: "https://api.sgroup.qq.com/channels/{{channel_id}}/messages",
		Method:      "POST",
		HeadersTemplate: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bot {{bot_token}}",
		},
		BodyTemplate:     `{"content":"{{.Content}}"}`,
		MaxLength:        2000,
		RequiredFields:   []string{"bot_app_id", "bot_token", "channel_id"},
		AutoDetectFields: []string{},
	},
}

// ──────────────────────────────────────────────
// 公開 API
// ──────────────────────────────────────────────

// GetPreset 取得指定平台的 preset。找不到回傳空 preset。
func GetPreset(platformID string) (PlatformPreset, bool) {
	p, ok := presets[platformID]
	return p, ok
}

// ListPresets 列出所有可用 preset ID。
func ListPresets() []string {
	ids := make([]string, 0, len(presets))
	for id := range presets {
		ids = append(ids, id)
	}
	return ids
}

// BuildWebhookRequest 將 preset 模板 + 使用者填入值 + 內容 → 產出 WebhookRequest。
func BuildWebhookRequest(preset PlatformPreset, fields map[string]string, content string) (WebhookRequest, error) {
	// 檢查必填欄位
	for _, f := range preset.RequiredFields {
		if _, ok := fields[f]; !ok {
			return WebhookRequest{}, fmt.Errorf("missing required field: %s", f)
		}
	}

	// 替換 URL 模板
	url := replaceTemplateFields(preset.URLTemplate, fields)

	// 替換 Headers 模板
	headers := make(map[string]string)
	for k, v := range preset.HeadersTemplate {
		headers[k] = replaceTemplateFields(v, fields)
	}

	// 替換 Body 模板（先換 fields 佔位符，再換 content）
	body := replaceTemplateFields(preset.BodyTemplate, fields)
	body = strings.ReplaceAll(body, "{{.Content}}", escapeJSON(content))

	return WebhookRequest{
		URL:            url,
		Method:         preset.Method,
		Headers:        headers,
		Body:           body,
		TimeoutSeconds: 10,
	}, nil
}

// ──────────────────────────────────────────────
// 內部 helper
// ──────────────────────────────────────────────

// replaceTemplateFields 將 {{field_name}} 替換為 fields map 中的值。
func replaceTemplateFields(template string, fields map[string]string) string {
	result := template
	for k, v := range fields {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

// escapeJSON 對字串做基本 JSON 轉義（避免內容破壞 JSON 結構）。
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
