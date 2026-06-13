package external_link

import "testing"

func TestDetectLLMProviderURLWhitelist(t *testing.T) {
	cases := []struct {
		name     string
		rawURL   string
		provider string
	}{
		{"deepseek keys", "https://platform.deepseek.com/api_keys", "deepseek"},
		{"claude docs", "https://platform.claude.com/docs/en/api/admin/api_keys/retrieve", "anthropic"},
		{"openai docs", "https://developers.openai.com/api/docs", "openai"},
		{"gemini api", "https://ai.google.dev/gemini-api", "gemini"},
		{"xai api", "https://x.ai/api", "xai"},
		{"openrouter", "https://openrouter.ai/docs/api-keys", "openrouter"},
		{"qwen", "https://docs.qwencloud.com/api-reference/preparation/api-key", "qwen"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			match, ok := DetectLLMProviderURL(tt.rawURL)
			if !ok {
				t.Fatalf("expected match for %s", tt.rawURL)
			}
			if match.Provider.ID != tt.provider {
				t.Fatalf("provider=%s want %s", match.Provider.ID, tt.provider)
			}
		})
	}
}

func TestPreviewClassifiesLLMProviderBeforeExternalService(t *testing.T) {
	svc := NewService(t.TempDir(), false)
	preview := svc.Preview("https://www.deepseek.com/en/")
	if !preview.Valid {
		t.Fatalf("preview invalid: %s", preview.Reason)
	}
	if preview.LinkType != LinkLLMProvider {
		t.Fatalf("link_type=%s want %s", preview.LinkType, LinkLLMProvider)
	}
	if preview.ProviderID != "deepseek" || preview.BaseURL == "" {
		t.Fatalf("unexpected provider metadata: %+v", preview)
	}
}

func TestDetectGenericAPIWithLLMHint(t *testing.T) {
	match, ok := DetectLLMProviderURL("https://api.example-ai.com/v1/chat/completions")
	if !ok {
		t.Fatal("expected generic api+llm hint match")
	}
	if match.Provider.ID != "generic-api" {
		t.Fatalf("provider=%s want generic-api", match.Provider.ID)
	}
}
