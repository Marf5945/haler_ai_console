// auth_domains.go — CLI adapter 授權 URL 白名單。
// SEC-05: 防止惡意 LLM 回傳釣魚 URL 讓 App 自動開啟瀏覽器。
// 已知 adapter 使用內建 allowlist 自動放行；未知 domain 需使用者確認。
package urlsafe

import (
	"net/url"
	"strings"
)

// KnownAuthDomains 已知 CLI adapter 的 OAuth/授權 domain 白名單。
// key = adapter ID（與 adapter_registry 一致），value = 允許的 domain 清單。
var KnownAuthDomains = map[string][]string{
	"gemini-cli": {
		"accounts.google.com",
		"aistudio.google.com",
		"myaccount.google.com",
		"oauth2.googleapis.com",
	},
	"claude-cli": {
		"console.anthropic.com",
		"claude.ai",
		"login.anthropic.com",
	},
	"codex-cli": {
		"github.com",
		"login.microsoftonline.com",
		"auth.openai.com",
		"platform.openai.com",
	},
	"copilot-cli": {
		"github.com",
		"login.microsoftonline.com",
	},
}

// ValidateAuthURL 驗證 auth_url 是否在該 adapter 的已知授權 domain 清單中。
// 回傳 trusted=true 代表可直接開啟瀏覽器；false 代表需使用者確認。
// hostname 回傳解析後的 host（供確認對話框顯示）。
func ValidateAuthURL(adapterID, rawURL string) (trusted bool, hostname string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false, ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false, ""
	}

	hostname = strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return false, ""
	}

	// scheme 必須是 https（auth URL 不應走 http）
	if strings.ToLower(parsed.Scheme) != "https" {
		return false, hostname
	}

	// 查 adapter 白名單
	domains, exists := KnownAuthDomains[strings.ToLower(adapterID)]
	if !exists {
		// 未知 adapter → 一律需確認
		return false, hostname
	}

	// suffix match：允許 sub.accounts.google.com 匹配 accounts.google.com
	for _, allowed := range domains {
		allowed = strings.ToLower(allowed)
		if hostname == allowed || strings.HasSuffix(hostname, "."+allowed) {
			return true, hostname
		}
	}

	return false, hostname
}
