package urlsafe

import (
	"testing"
)

func TestValidateURL_CloudAPI(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https 公網 OK", "https://api.openai.com/v1", false},
		{"http 被拒", "http://api.openai.com/v1", true},
		{"localhost 被拒", "http://localhost:11434", true},
		{"127.0.0.1 被拒", "http://127.0.0.1:8080", true},
		{"::1 被拒", "http://[::1]:8080", true},
		{"private 10.x 被拒", "https://10.0.0.1:443", true},
		{"private 192.168 被拒", "https://192.168.1.100:443", true},
		{"private 172.16 被拒", "https://172.16.0.1:443", true},
		{"link-local 被拒", "https://169.254.169.254/metadata", true},
		{"空 URL 被拒", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, PolicyCloudAPI)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q, CloudAPI) error=%v, wantErr=%v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_LocalLLM(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https 公網 OK", "https://api.openai.com/v1", false},
		{"http localhost OK", "http://localhost:11434", false},
		{"http 127.0.0.1 OK", "http://127.0.0.1:11434", false},
		{"http [::1] OK", "http://[::1]:11434", false},
		{"LAN 192.168 被拒", "http://192.168.1.100:8080", true},
		{"LAN 10.x 被拒", "http://10.0.0.5:8080", true},
		{"link-local 被拒", "http://169.254.169.254/latest", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, PolicyLocalLLM)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q, LocalLLM) error=%v, wantErr=%v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_Webhook(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https 公網 OK", "https://hooks.slack.com/services/xxx", false},
		{"http 被拒", "http://hooks.slack.com/services/xxx", true},
		{"localhost 被拒", "http://127.0.0.1:3000/hook", true},
		{"private 被拒", "https://192.168.1.1:443/hook", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, PolicyWebhook)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q, Webhook) error=%v, wantErr=%v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_WebhookDev(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https 公網 OK", "https://hooks.slack.com/services/xxx", false},
		{"http localhost OK", "http://localhost:3000/hook", false},
		{"http 127.0.0.1 OK", "http://127.0.0.1:3000/hook", false},
		{"http private OK", "http://192.168.1.100:8080/hook", false},
		{"link-local 被拒", "http://169.254.169.254/metadata", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, PolicyWebhookDev)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q, WebhookDev) error=%v, wantErr=%v", tt.url, err, tt.wantErr)
			}
		})
	}
}
