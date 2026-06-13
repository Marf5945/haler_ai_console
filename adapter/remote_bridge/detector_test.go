package remote_bridge

import "testing"

func TestDetectChannel(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    ChannelType
		matched bool
	}{
		{"telegram bot", "https://api.telegram.org/bot123456:ABC-DEF/sendMessage", ChannelTelegram, true},
		{"telegram bot no method", "https://api.telegram.org/bot123456:ABC-DEF", ChannelTelegram, true},
		{"discord webhook", "https://discord.com/api/webhooks/123/abc-token", ChannelDiscord, true},
		{"discord old domain", "https://discordapp.com/api/webhooks/123/abc-token", ChannelDiscord, true},
		{"line notify", "https://notify-api.line.me/api/notify", ChannelLINE, true},
		{"line messaging api", "https://api.line.me/v2/bot/message/push", ChannelLINE, true},
		{"line platform site", "https://www.line.me/tw/", ChannelLINE, true},
		{"telegram platform site", "https://telegram.org/", ChannelTelegram, true},
		{"discord platform site", "https://discord.com/", ChannelDiscord, true},
		{"teams platform site", "https://teams.microsoft.com/", ChannelTeams, true},
		{"teams workflow webhook", "https://prod-12.westus.logic.azure.com/workflows/abc/triggers/manual/paths/invoke", ChannelTeams, true},
		{"microsoft generic site", "https://www.microsoft.com/", "", false},
		{"qq platform site", "https://im.qq.com/index/", ChannelQQ, true},
		{"qq guild bot platform", "https://bot.q.qq.com/wiki/", ChannelQQ, true},
		{"qq bot api", "https://api.sgroup.qq.com/channels/123/messages", ChannelQQ, true},
		{"unknown url", "https://example.com/webhook", "", false},
		{"empty", "", "", false},
		{"http not https", "http://api.telegram.org/bot123/send", ChannelTelegram, true}, // detect still works, validation rejects
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectChannel(tt.url)
			if result.Matched != tt.matched {
				t.Errorf("Matched = %v, want %v", result.Matched, tt.matched)
			}
			if result.Matched && result.Channel != tt.want {
				t.Errorf("Channel = %v, want %v", result.Channel, tt.want)
			}
		})
	}
}

func TestExtractBotToken(t *testing.T) {
	token, err := ExtractBotToken("https://api.telegram.org/bot123456:ABC-DEF/sendMessage")
	if err != nil {
		t.Fatal(err)
	}
	if token != "123456:ABC-DEF" {
		t.Errorf("got %q, want %q", token, "123456:ABC-DEF")
	}
}

func TestExtractDiscordWebhookParts(t *testing.T) {
	id, tok, err := ExtractDiscordWebhookParts("https://discord.com/api/webhooks/99887766/my-secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if id != "99887766" {
		t.Errorf("id = %q, want %q", id, "99887766")
	}
	if tok != "my-secret-token" {
		t.Errorf("token = %q, want %q", tok, "my-secret-token")
	}
}

func TestValidateURLFormat(t *testing.T) {
	if err := ValidateURLFormat("https://api.telegram.org/bot123/sendMessage"); err != nil {
		t.Errorf("valid URL should pass: %v", err)
	}
	if err := ValidateURLFormat("http://api.telegram.org/bot123"); err == nil {
		t.Error("HTTP should be rejected")
	}
	if err := ValidateURLFormat(""); err == nil {
		t.Error("empty should be rejected")
	}
}
