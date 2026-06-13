// remote_bridge/webhook_sender_test.go — Generic Webhook Sender 測試。
// SEC-04: 新增 SSRF 封鎖測試。
package remote_bridge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSendWebhook_Success 驗證正常 200 回應（DevMode=true 因 httptest 為 localhost）。
func TestSendWebhook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	req := WebhookRequest{
		URL:            server.URL,
		Method:         "POST",
		Headers:        map[string]string{"X-Custom": "test"},
		Body:           `{"text":"hello"}`,
		TimeoutSeconds: 5,
		DevMode:        true, // httptest 為 http://127.0.0.1，需 DevMode
	}

	resp, err := SendWebhook(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.ResponseBody != `{"ok":true}` {
		t.Errorf("unexpected body: %s", resp.ResponseBody)
	}
	if resp.Error != "" {
		t.Errorf("unexpected error field: %s", resp.Error)
	}
}

// TestSendWebhook_Timeout 驗證超時處理。
func TestSendWebhook_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req := WebhookRequest{
		URL:            server.URL,
		Method:         "POST",
		Body:           `{"text":"hello"}`,
		TimeoutSeconds: 1,
		DevMode:        true,
	}

	resp, err := SendWebhook(req)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if resp.Error == "" {
		t.Error("expected error message in response")
	}
}

// TestSendWebhook_Non2xx 驗證非 2xx 回應。
func TestSendWebhook_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	req := WebhookRequest{
		URL:            server.URL,
		Method:         "POST",
		Body:           `{"text":"hello"}`,
		TimeoutSeconds: 5,
		DevMode:        true,
	}

	resp, err := SendWebhook(req)
	if err != nil {
		t.Fatalf("non-2xx should not return error: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// --- SEC-04: SSRF 防護測試 ---

// TestSendWebhook_SSRF_BlockLocalhost 正式模式封鎖 localhost。
func TestSendWebhook_SSRF_BlockLocalhost(t *testing.T) {
	req := WebhookRequest{
		URL:     "http://127.0.0.1:3000/hook",
		Method:  "POST",
		Body:    `{"text":"test"}`,
		DevMode: false,
	}
	_, err := SendWebhook(req)
	if err == nil {
		t.Fatal("expected SSRF block for localhost, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

// TestSendWebhook_SSRF_BlockPrivateIP 正式模式封鎖 private IP。
func TestSendWebhook_SSRF_BlockPrivateIP(t *testing.T) {
	req := WebhookRequest{
		URL:     "https://192.168.1.100:443/hook",
		Method:  "POST",
		Body:    `{"text":"test"}`,
		DevMode: false,
	}
	_, err := SendWebhook(req)
	if err == nil {
		t.Fatal("expected SSRF block for private IP, got nil")
	}
}

// TestSendWebhook_SSRF_BlockMetadata 封鎖雲端 metadata endpoint。
func TestSendWebhook_SSRF_BlockMetadata(t *testing.T) {
	req := WebhookRequest{
		URL:     "http://169.254.169.254/latest/meta-data/",
		Method:  "GET",
		DevMode: false,
	}
	_, err := SendWebhook(req)
	if err == nil {
		t.Fatal("expected SSRF block for link-local, got nil")
	}
}

// TestSendWebhook_SSRF_BlockHTTP 正式模式封鎖 http（非 https）。
func TestSendWebhook_SSRF_BlockHTTP(t *testing.T) {
	req := WebhookRequest{
		URL:     "http://hooks.slack.com/services/xxx",
		Method:  "POST",
		Body:    `{"text":"test"}`,
		DevMode: false,
	}
	_, err := SendWebhook(req)
	if err == nil {
		t.Fatal("expected block for http in production mode, got nil")
	}
}

// TestSendWebhook_DevMode_AllowLocalhost 開發模式允許 localhost。
func TestSendWebhook_DevMode_AllowLocalhost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	req := WebhookRequest{
		URL:     server.URL,
		Method:  "POST",
		Body:    `{"text":"test"}`,
		DevMode: true,
	}
	resp, err := SendWebhook(req)
	if err != nil {
		t.Fatalf("DevMode should allow localhost: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
