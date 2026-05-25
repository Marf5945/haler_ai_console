package urlsafe

import "testing"

// SEC-03 驗證：ValidateLLMBaseURL 依 providerID 分策略。
func TestValidateLLMBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		providerID  string
		baseURL     string
		wantConfirm bool
		wantErr     bool
	}{
		// --- ollama/lmstudio: 允許 localhost ---
		{"ollama localhost OK", "ollama", "http://localhost:11434/v1", false, false},
		{"ollama 127.0.0.1 OK", "ollama", "http://127.0.0.1:11434/v1", false, false},
		{"lmstudio localhost OK", "lmstudio", "http://localhost:1234/v1", false, false},
		{"ollama https 公網 OK", "ollama", "https://my-ollama.example.com/v1", false, false},
		{"ollama LAN 被拒", "ollama", "http://192.168.1.100:11434", false, true},
		{"ollama link-local 被拒", "ollama", "http://169.254.169.254/v1", false, true},

		// --- 一般 provider: 僅 https，private IP → needConfirm ---
		{"openai https OK", "openai", "https://api.openai.com/v1", false, false},
		{"deepseek https OK", "deepseek", "https://api.deepseek.com", false, false},
		{"generic https OK", "generic-api", "https://my-api.example.com/v1", false, false},

		// private IP → needConfirm（不報錯，讓前端確認）
		{"generic private 192.168 需確認", "generic-api", "https://192.168.1.100:443/v1", true, false},
		{"generic localhost 需確認", "generic-api", "http://localhost:8080/v1", true, false},
		{"generic 127.0.0.1 需確認", "generic-api", "http://127.0.0.1:8080/v1", true, false},

		// scheme 錯誤 → 直接拒絕
		{"generic http 公網被拒", "generic-api", "http://api.example.com/v1", false, true},
		{"generic ftp 被拒", "generic-api", "ftp://api.example.com/v1", false, true},

		// 空 URL → 通過（由後續自動填入處理）
		{"空 URL 通過", "openai", "", false, false},

		// providerID 大小寫不敏感
		{"Ollama 大寫 OK", "Ollama", "http://localhost:11434/v1", false, false},
		{"LMSTUDIO 大寫 OK", "LMSTUDIO", "http://localhost:1234/v1", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needConfirm, _, err := ValidateLLMBaseURL(tt.providerID, tt.baseURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLLMBaseURL(%q, %q) error=%v, wantErr=%v",
					tt.providerID, tt.baseURL, err, tt.wantErr)
			}
			if needConfirm != tt.wantConfirm {
				t.Errorf("ValidateLLMBaseURL(%q, %q) needConfirm=%v, want %v",
					tt.providerID, tt.baseURL, needConfirm, tt.wantConfirm)
			}
		})
	}
}
