package remote_bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisterChannelWithMode_DeveloperConfigStoredAsSecret(t *testing.T) {
	dir := t.TempDir()
	secrets := &mockSecretStore{data: make(map[string]string)}
	svc := NewService(dir, secrets)

	cfg := &WebhookRequest{
		URL:            "https://example.com/hook?token=secret-token",
		Method:         "POST",
		Headers:        map[string]string{"Authorization": "Bearer secret-token"},
		Body:           `{"text":"{{.Content}}","token":"secret-token"}`,
		TimeoutSeconds: 7,
	}

	binding, err := svc.RegisterChannelWithMode("developer", "", nil, cfg)
	if err != nil {
		t.Fatalf("RegisterChannelWithMode: %v", err)
	}

	bindingsRaw, err := os.ReadFile(filepath.Join(dir, "remote_bridge", bindingsFile))
	if err != nil {
		t.Fatalf("read bindings: %v", err)
	}
	if strings.Contains(string(bindingsRaw), "secret-token") || strings.Contains(string(bindingsRaw), "example.com") {
		t.Fatalf("developer webhook secret material leaked into bindings: %s", bindingsRaw)
	}
	if !secrets.Has("remote_bridge:" + binding.ID + ":custom_config") {
		t.Fatal("expected full developer config in SecretStore")
	}

	req := NewDispatcher(svc).buildWebhookRequest(&binding, "hello", 0, 1)
	if req.URL != cfg.URL {
		t.Fatalf("URL = %q, want %q", req.URL, cfg.URL)
	}
	if req.Headers["Authorization"] != cfg.Headers["Authorization"] {
		t.Fatalf("Authorization header = %q, want %q", req.Headers["Authorization"], cfg.Headers["Authorization"])
	}
	if !strings.Contains(req.Body, `"text":"hello"`) || !strings.Contains(req.Body, "secret-token") {
		t.Fatalf("custom body template was not rendered from secret config: %s", req.Body)
	}
}
