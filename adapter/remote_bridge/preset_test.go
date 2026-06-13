// remote_bridge/preset_test.go — Platform Preset 測試。
package remote_bridge

import (
	"strings"
	"testing"
)

// TestBuildWebhookRequest_Telegram 驗證 Telegram preset 正確替換模板。
func TestBuildWebhookRequest_Telegram(t *testing.T) {
	preset, ok := GetPreset("telegram")
	if !ok {
		t.Fatal("telegram preset not found")
	}

	fields := map[string]string{
		"bot_token": "123456:ABC-DEF",
		"chat_id":   "99887766",
	}

	req, err := BuildWebhookRequest(preset, fields, "Hello World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedURL := "https://api.telegram.org/bot123456:ABC-DEF/sendMessage"
	if req.URL != expectedURL {
		t.Errorf("URL mismatch:\n  got:  %s\n  want: %s", req.URL, expectedURL)
	}
	if !strings.Contains(req.Body, `"chat_id":"99887766"`) {
		t.Errorf("body missing chat_id: %s", req.Body)
	}
	if !strings.Contains(req.Body, "Hello World") {
		t.Errorf("body missing content: %s", req.Body)
	}
	if req.Method != "POST" {
		t.Errorf("expected POST, got %s", req.Method)
	}
}

// TestBuildWebhookRequest_Discord 驗證 Discord preset 正確替換模板。
func TestBuildWebhookRequest_Discord(t *testing.T) {
	preset, ok := GetPreset("discord")
	if !ok {
		t.Fatal("discord preset not found")
	}

	fields := map[string]string{
		"bot_token":  "bot-secret",
		"guild_id":   "111",
		"channel_id": "222",
	}

	req, err := BuildWebhookRequest(preset, fields, "Test message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.URL != "https://discord.com/api/v10/channels/222/messages" {
		t.Errorf("URL mismatch: %s", req.URL)
	}
	if req.Headers["Authorization"] != "Bot bot-secret" {
		t.Errorf("Authorization header mismatch: %v", req.Headers)
	}
	if !strings.Contains(req.Body, "Test message") {
		t.Errorf("body missing content: %s", req.Body)
	}
	if !strings.Contains(req.Body, `"allowed_mentions":{"parse":[]}`) {
		t.Errorf("body should disable Discord mentions: %s", req.Body)
	}
}

// TestBuildWebhookRequest_LINE 驗證 LINE preset 正確替換模板。
func TestBuildWebhookRequest_LINE(t *testing.T) {
	preset, ok := GetPreset("line")
	if !ok {
		t.Fatal("line preset not found")
	}

	fields := map[string]string{
		"channel_access_token": "mytoken123",
		"recipient_id":         "U1234567890",
	}

	req, err := BuildWebhookRequest(preset, fields, "LINE msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.URL != "https://api.line.me/v2/bot/message/push" {
		t.Errorf("URL mismatch: %s", req.URL)
	}
	if !strings.Contains(req.Headers["Authorization"], "Bearer mytoken123") {
		t.Errorf("Authorization header mismatch: %v", req.Headers)
	}
	if !strings.Contains(req.Body, "U1234567890") {
		t.Errorf("body missing recipient_id: %s", req.Body)
	}
}

// TestBuildWebhookRequest_Teams 驗證 Teams preset 與 Discord 一樣可直接套 webhook URL。
func TestBuildWebhookRequest_Teams(t *testing.T) {
	preset, ok := GetPreset("teams")
	if !ok {
		t.Fatal("teams preset not found")
	}

	fields := map[string]string{
		"webhook_url": "https://prod-12.westus.logic.azure.com/workflows/abc/triggers/manual/paths/invoke",
	}

	req, err := BuildWebhookRequest(preset, fields, "Teams msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URL != fields["webhook_url"] {
		t.Errorf("URL mismatch: %s", req.URL)
	}
	if !strings.Contains(req.Body, `"text":"Teams msg"`) {
		t.Errorf("body missing content: %s", req.Body)
	}
}

// TestBuildWebhookRequest_QQ 驗證 QQ Guild Bot preset 會套用 channel_id 與 Bot token。
func TestBuildWebhookRequest_QQ(t *testing.T) {
	preset, ok := GetPreset("qq")
	if !ok {
		t.Fatal("qq preset not found")
	}

	fields := map[string]string{
		"bot_app_id": "1024",
		"bot_token":  "qq-token",
		"channel_id": "guild-channel-1",
	}

	req, err := BuildWebhookRequest(preset, fields, "QQ msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URL != "https://api.sgroup.qq.com/channels/guild-channel-1/messages" {
		t.Errorf("URL mismatch: %s", req.URL)
	}
	if req.Headers["Authorization"] != "Bot qq-token" {
		t.Errorf("Authorization header mismatch: %v", req.Headers)
	}
	if !strings.Contains(req.Body, `"content":"QQ msg"`) {
		t.Errorf("body missing content: %s", req.Body)
	}
}

// TestBuildWebhookRequest_MissingField 驗證缺少必填欄位時回傳錯誤。
func TestBuildWebhookRequest_MissingField(t *testing.T) {
	preset, _ := GetPreset("telegram")

	fields := map[string]string{} // 缺少 bot_token

	_, err := BuildWebhookRequest(preset, fields, "content")
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "bot_token") {
		t.Errorf("error should mention missing field: %v", err)
	}
}

// TestGetPreset_Unknown 驗證未知平台回傳 false。
func TestGetPreset_Unknown(t *testing.T) {
	_, ok := GetPreset("unknown_platform")
	if ok {
		t.Error("expected false for unknown platform")
	}
}
