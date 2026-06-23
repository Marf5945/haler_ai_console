// remote_bridge/detector.go — URL 模式偵測器。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 使用者在「引用連結」輸入框貼上一串 URL 後，本檔案負責        │
// │ 根據 hostname + path pattern 判斷屬於哪種通訊平台。          │
// │                                                             │
// │ 支援的 URL 模式：                                           │
// │  • Telegram : api.telegram.org/bot<TOKEN>/…                 │
// │  • Discord  : discord.com/api/webhooks/<ID>/<TOKEN>         │
// │  • LINE     : notify-api.line.me/… 或 api.line.me/v2/bot/… │
// │  • Teams    : teams.microsoft.com / Teams webhook URLs      │
// │  • QQ        : QQ / QQ Guild platform URLs                   │
// │                                                             │
// │ 呼叫鏈：                                                    │
// │  前端 onPreview → Wails DetectRemoteBridgeChannel            │
// │    → service.DetectChannelFromURL → DetectChannel (本檔案)   │
// │                                                             │
// │ 本檔案只做「辨識」，不做連線測試（見 connection.go）。       │
// │ ExtractBotToken / ExtractDiscordWebhookParts 供              │
// │ connection.go 拆解 URL 取出 token 用。                      │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"fmt"
	"net/url"
	"strings"
)

// ──────────────────────────────────────────────
// URL 模式偵測
// ──────────────────────────────────────────────

// DetectResult 偵測結果。
type DetectResult struct {
	Channel   ChannelType `json:"channel"`
	Matched   bool        `json:"matched"`
	URLType   string      `json:"url_type"`   // "bot_api" / "webhook" / "notify"
	HintLabel string      `json:"hint_label"` // 給 UI 顯示的提示，例如 "Telegram Bot API"
}

// DetectChannel 從 URL 字串偵測通道類型。
// 支援的 URL 模式：
//   - Telegram: https://api.telegram.org/bot<TOKEN>/...
//   - Discord:  https://discord.com/api/webhooks/<ID>/<TOKEN>
//     https://discordapp.com/api/webhooks/<ID>/<TOKEN>
//   - LINE:     https://notify-api.line.me/api/notify
//     https://api.line.me/v2/bot/message/...
//   - Platform sites open the setup flow instead of registering the page URL.
func DetectChannel(rawURL string) DetectResult {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return DetectResult{}
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return DetectResult{}
	}

	host := strings.ToLower(parsed.Hostname())
	path := strings.ToLower(parsed.Path)

	// Telegram Bot API
	if host == "api.telegram.org" && strings.HasPrefix(path, "/bot") {
		return DetectResult{
			Channel:   ChannelTelegram,
			Matched:   true,
			URLType:   "bot_api",
			HintLabel: "Telegram Bot API",
		}
	}
	if host == "telegram.org" || strings.HasSuffix(host, ".telegram.org") || host == "t.me" {
		return DetectResult{
			Channel:   ChannelTelegram,
			Matched:   true,
			URLType:   "platform_site",
			HintLabel: "Telegram",
		}
	}

	// Discord Webhook
	if (host == "discord.com" || host == "discordapp.com") && strings.Contains(path, "/api/webhooks/") {
		return DetectResult{
			Channel:   ChannelDiscord,
			Matched:   true,
			URLType:   "webhook",
			HintLabel: "Discord Webhook",
		}
	}
	if host == "discord.com" || host == "discordapp.com" || host == "discord.gg" {
		return DetectResult{
			Channel:   ChannelDiscord,
			Matched:   true,
			URLType:   "platform_site",
			HintLabel: "Discord",
		}
	}

	// LINE Notify
	if host == "notify-api.line.me" {
		return DetectResult{
			Channel:   ChannelLINE,
			Matched:   true,
			URLType:   "notify",
			HintLabel: "LINE Notify",
		}
	}

	// LINE Messaging API
	if host == "api.line.me" && strings.Contains(path, "/bot/message") {
		return DetectResult{
			Channel:   ChannelLINE,
			Matched:   true,
			URLType:   "bot_api",
			HintLabel: "LINE Messaging API",
		}
	}
	if host == "line.me" || strings.HasSuffix(host, ".line.me") || host == "line.biz" || strings.HasSuffix(host, ".line.biz") {
		return DetectResult{
			Channel:   ChannelLINE,
			Matched:   true,
			URLType:   "platform_site",
			HintLabel: "LINE",
		}
	}

	if isTeamsWebhookHost(host) {
		return DetectResult{
			Channel:   ChannelTeams,
			Matched:   true,
			URLType:   "webhook",
			HintLabel: "Teams Webhook",
		}
	}
	if isTeamsPlatformHost(host) {
		return DetectResult{
			Channel:   ChannelTeams,
			Matched:   true,
			URLType:   "platform_site",
			HintLabel: "Microsoft Teams",
		}
	}

	if isQQBotAPIHost(host) {
		return DetectResult{
			Channel:   ChannelQQ,
			Matched:   true,
			URLType:   "bot_api",
			HintLabel: "QQ Bot / QQ Guild Bot",
		}
	}
	if isQQPlatformHost(host) {
		return DetectResult{
			Channel:   ChannelQQ,
			Matched:   true,
			URLType:   "platform_site",
			HintLabel: "QQ Bot / QQ Guild Bot",
		}
	}

	return DetectResult{}
}

func isTeamsPlatformHost(host string) bool {
	return host == "teams.microsoft.com" ||
		strings.HasSuffix(host, ".teams.microsoft.com") ||
		host == "teams.live.com"
}

func isTeamsWebhookHost(host string) bool {
	return host == "outlook.office.com" ||
		strings.HasSuffix(host, ".webhook.office.com") ||
		strings.Contains(host, ".logic.azure.com")
}

func isQQPlatformHost(host string) bool {
	return host == "im.qq.com" ||
		host == "bot.q.qq.com" ||
		strings.HasSuffix(host, ".bot.q.qq.com") ||
		host == "q.qq.com" ||
		strings.HasSuffix(host, ".q.qq.com")
}

func isQQBotAPIHost(host string) bool {
	return host == "api.sgroup.qq.com" ||
		strings.HasSuffix(host, ".api.sgroup.qq.com")
}

// ExtractBotToken 從 Telegram Bot API URL 解析 token。
// 格式: https://api.telegram.org/bot<TOKEN>/sendMessage
func ExtractBotToken(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	path := parsed.Path
	if !strings.HasPrefix(path, "/bot") {
		return "", fmt.Errorf("not a Telegram bot URL")
	}

	// /bot<TOKEN>/sendMessage → <TOKEN>
	after := strings.TrimPrefix(path, "/bot")
	if idx := strings.Index(after, "/"); idx > 0 {
		return after[:idx], nil
	}
	if after != "" {
		return after, nil
	}
	return "", fmt.Errorf("token not found in URL path")
}

// ExtractDiscordWebhookParts 從 Discord Webhook URL 解析 ID 和 Token。
// 格式: https://discord.com/api/webhooks/<ID>/<TOKEN>
func ExtractDiscordWebhookParts(rawURL string) (webhookID, webhookToken string, err error) {
	parsed, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", "", fmt.Errorf("invalid URL: %w", parseErr)
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	// api / webhooks / <ID> / <TOKEN>
	webhookIdx := -1
	for i, p := range parts {
		if p == "webhooks" {
			webhookIdx = i
			break
		}
	}
	if webhookIdx < 0 || webhookIdx+2 >= len(parts) {
		return "", "", fmt.Errorf("cannot parse webhook ID/token from path")
	}
	return parts[webhookIdx+1], parts[webhookIdx+2], nil
}

// ValidateURLFormat 基本格式驗證（不做連線測試）。
func ValidateURLFormat(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("URL is empty")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are accepted")
	}
	if parsed.Host == "" {
		return fmt.Errorf("missing host in URL")
	}
	return nil
}
