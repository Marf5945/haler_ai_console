package urlsafe

import "testing"

func TestValidateAuthURL(t *testing.T) {
	tests := []struct {
		name      string
		adapter   string
		url       string
		wantTrust bool
		wantHost  string
	}{
		// Gemini — 已知 domain
		{"gemini google accounts", "gemini-cli",
			"https://accounts.google.com/o/oauth2/auth?client_id=xxx", true, "accounts.google.com"},
		{"gemini aistudio", "gemini-cli",
			"https://aistudio.google.com/apikey", true, "aistudio.google.com"},
		// Gemini — 未知 domain
		{"gemini evil domain", "gemini-cli",
			"https://evil.com/phish?redirect=google.com", false, "evil.com"},
		// Claude — 已知 domain
		{"claude console", "claude-cli",
			"https://console.anthropic.com/login", true, "console.anthropic.com"},
		// Codex — 已知 domain
		{"codex github", "codex-cli",
			"https://github.com/login/oauth/authorize", true, "github.com"},
		// 未知 adapter — 一律不信任
		{"unknown adapter", "unknown-cli",
			"https://example.com/auth", false, "example.com"},
		// http 不信任（即使 domain 正確）
		{"http not trusted", "gemini-cli",
			"http://accounts.google.com/o/oauth2/auth", false, "accounts.google.com"},
		// 空 URL
		{"empty url", "gemini-cli", "", false, ""},
		// subdomain match
		{"subdomain match", "gemini-cli",
			"https://sub.accounts.google.com/auth", true, "sub.accounts.google.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trusted, host := ValidateAuthURL(tt.adapter, tt.url)
			if trusted != tt.wantTrust {
				t.Errorf("ValidateAuthURL(%q, %q) trusted=%v, want %v",
					tt.adapter, tt.url, trusted, tt.wantTrust)
			}
			if host != tt.wantHost {
				t.Errorf("ValidateAuthURL(%q, %q) host=%q, want %q",
					tt.adapter, tt.url, host, tt.wantHost)
			}
		})
	}
}
