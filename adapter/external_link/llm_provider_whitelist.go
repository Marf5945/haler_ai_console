package external_link

import (
	"net/url"
	"strings"
)

// LLMProviderWhitelistEntry is the maintainable allowlist used by the
// Reference Link preview flow. Add domains here when a new model API provider
// should open the API adapter setup path instead of becoming a plain link.
type LLMProviderWhitelistEntry struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	BaseURL   string   `json:"base_url,omitempty"`
	APIKeyURL string   `json:"api_key_url,omitempty"`
	DocsURL   string   `json:"docs_url,omitempty"`
	Domains   []string `json:"domains"`
}

type LLMProviderMatch struct {
	Provider  LLMProviderWhitelistEntry `json:"provider"`
	MatchedBy string                    `json:"matched_by"`
}

var llmProviderWhitelist = []LLMProviderWhitelistEntry{
	{ID: "deepseek", Name: "DeepSeek", BaseURL: "https://api.deepseek.com", APIKeyURL: "https://platform.deepseek.com/api_keys", DocsURL: "https://api-docs.deepseek.com/", Domains: []string{"deepseek.com", "platform.deepseek.com", "api-docs.deepseek.com", "api.deepseek.com"}},
	{ID: "openai", Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIKeyURL: "https://platform.openai.com/api-keys", DocsURL: "https://platform.openai.com/docs", Domains: []string{"openai.com", "platform.openai.com", "developers.openai.com", "api.openai.com"}},
	{ID: "anthropic", Name: "Claude / Anthropic", BaseURL: "https://api.anthropic.com", APIKeyURL: "https://console.anthropic.com/settings/keys", DocsURL: "https://docs.anthropic.com/", Domains: []string{"anthropic.com", "claude.com", "platform.claude.com", "docs.anthropic.com", "api.anthropic.com"}},
	{ID: "gemini", Name: "Google Gemini", BaseURL: "https://generativelanguage.googleapis.com", APIKeyURL: "https://aistudio.google.com/api-keys", DocsURL: "https://ai.google.dev/gemini-api", Domains: []string{"ai.google.dev", "aistudio.google.com", "generativelanguage.googleapis.com", "googleapis.com"}},
	{ID: "xai", Name: "xAI", BaseURL: "https://api.x.ai/v1", APIKeyURL: "https://console.x.ai/", DocsURL: "https://docs.x.ai/", Domains: []string{"x.ai", "docs.x.ai", "console.x.ai", "api.x.ai"}},
	{ID: "openrouter", Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", APIKeyURL: "https://openrouter.ai/keys", DocsURL: "https://openrouter.ai/docs", Domains: []string{"openrouter.ai"}},
	{ID: "mistral", Name: "Mistral AI", BaseURL: "https://api.mistral.ai/v1", APIKeyURL: "https://console.mistral.ai/api-keys", DocsURL: "https://docs.mistral.ai/", Domains: []string{"mistral.ai", "docs.mistral.ai", "console.mistral.ai", "api.mistral.ai"}},
	{ID: "groq", Name: "Groq", BaseURL: "https://api.groq.com/openai/v1", APIKeyURL: "https://console.groq.com/keys", DocsURL: "https://console.groq.com/docs", Domains: []string{"groq.com", "console.groq.com", "api.groq.com"}},
	{ID: "together", Name: "Together AI", BaseURL: "https://api.together.xyz/v1", APIKeyURL: "https://api.together.ai/settings/api-keys", DocsURL: "https://docs.together.ai/", Domains: []string{"together.ai", "docs.together.ai", "api.together.ai", "api.together.xyz"}},
	{ID: "perplexity", Name: "Perplexity", BaseURL: "https://api.perplexity.ai", APIKeyURL: "https://www.perplexity.ai/settings/api", DocsURL: "https://docs.perplexity.ai/", Domains: []string{"perplexity.ai", "docs.perplexity.ai", "api.perplexity.ai"}},
	{ID: "cohere", Name: "Cohere", BaseURL: "https://api.cohere.com/v2", APIKeyURL: "https://dashboard.cohere.com/api-keys", DocsURL: "https://docs.cohere.com/", Domains: []string{"cohere.com", "docs.cohere.com", "dashboard.cohere.com", "api.cohere.com"}},
	{ID: "fireworks", Name: "Fireworks AI", BaseURL: "https://api.fireworks.ai/inference/v1", APIKeyURL: "https://fireworks.ai/account/api-keys", DocsURL: "https://fireworks.ai/docs", Domains: []string{"fireworks.ai", "docs.fireworks.ai", "api.fireworks.ai"}},
	{ID: "huggingface", Name: "Hugging Face", BaseURL: "https://router.huggingface.co/v1", APIKeyURL: "https://huggingface.co/settings/tokens", DocsURL: "https://huggingface.co/docs/inference-providers/", Domains: []string{"huggingface.co", "hf.co", "router.huggingface.co", "api-inference.huggingface.co"}},
	{ID: "replicate", Name: "Replicate", BaseURL: "https://api.replicate.com/v1", APIKeyURL: "https://replicate.com/account/api-tokens", DocsURL: "https://replicate.com/docs", Domains: []string{"replicate.com", "api.replicate.com"}},
	{ID: "cerebras", Name: "Cerebras", BaseURL: "https://api.cerebras.ai/v1", APIKeyURL: "https://cloud.cerebras.ai/platform", DocsURL: "https://inference-docs.cerebras.ai/", Domains: []string{"cerebras.ai", "cloud.cerebras.ai", "inference-docs.cerebras.ai", "api.cerebras.ai"}},
	{ID: "cloudflare-workers-ai", Name: "Cloudflare Workers AI", DocsURL: "https://developers.cloudflare.com/workers-ai/", Domains: []string{"cloudflare.com", "developers.cloudflare.com", "workers-ai"}},
	{ID: "nvidia-nim", Name: "NVIDIA NIM", BaseURL: "https://integrate.api.nvidia.com/v1", APIKeyURL: "https://org.ngc.nvidia.com/setup/personal-keys", DocsURL: "https://docs.api.nvidia.com/nim/", Domains: []string{"nvidia.com", "docs.api.nvidia.com", "integrate.api.nvidia.com", "ngc.nvidia.com"}},
	{ID: "qwen", Name: "Qwen / DashScope", BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", DocsURL: "https://docs.qwencloud.com/", Domains: []string{"qwencloud.com", "docs.qwencloud.com", "dashscope", "dashscope.aliyuncs.com", "dashscope-intl.aliyuncs.com"}},
	{ID: "kimi", Name: "Kimi / Moonshot", BaseURL: "https://api.moonshot.ai/v1", APIKeyURL: "https://platform.moonshot.ai/console/api-keys", DocsURL: "https://platform.moonshot.ai/docs/", Domains: []string{"moonshot.ai", "platform.moonshot.ai", "api.moonshot.ai", "kimi.ai", "platform.kimi.ai", "kimi.com"}},
	{ID: "zhipu", Name: "Zhipu / GLM / BigModel", BaseURL: "https://open.bigmodel.cn/api/paas/v4", APIKeyURL: "https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys", DocsURL: "https://docs.bigmodel.cn/", Domains: []string{"bigmodel.cn", "open.bigmodel.cn", "docs.bigmodel.cn", "zhipuai.cn", "z.ai"}},
	{ID: "baidu-qianfan", Name: "Baidu Qianfan", DocsURL: "https://cloud.baidu.com/doc/WENXINWORKSHOP/", Domains: []string{"baidu.com", "cloud.baidu.com", "console.bce.baidu.com", "qianfan"}},
	{ID: "tencent-hunyuan", Name: "Tencent Hunyuan", DocsURL: "https://intl.cloud.tencent.com/document/product/1290", Domains: []string{"tencentcloud.com", "cloud.tencent.com", "hunyuan"}},
	{ID: "azure-openai", Name: "Azure OpenAI", DocsURL: "https://learn.microsoft.com/azure/ai-services/openai/", Domains: []string{"openai.azure.com", "azure.com", "learn.microsoft.com"}},
	{ID: "amazon-bedrock", Name: "Amazon Bedrock", DocsURL: "https://docs.aws.amazon.com/bedrock/", Domains: []string{"aws.amazon.com", "docs.aws.amazon.com", "bedrock"}},
}

func ListLLMProviderWhitelist() []LLMProviderWhitelistEntry {
	result := make([]LLMProviderWhitelistEntry, len(llmProviderWhitelist))
	copy(result, llmProviderWhitelist)
	return result
}

func DetectLLMProviderURL(rawURL string) (LLMProviderMatch, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return LLMProviderMatch{}, false
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	full := strings.ToLower(strings.TrimSpace(rawURL))
	for _, provider := range llmProviderWhitelist {
		for _, domain := range provider.Domains {
			needle := strings.ToLower(strings.TrimSpace(domain))
			if needle == "" {
				continue
			}
			if hostMatchesDomain(host, needle) || strings.Contains(full, needle) {
				return LLMProviderMatch{Provider: provider, MatchedBy: needle}, true
			}
		}
	}
	if strings.Contains(full, "api") && hasLLMHint(full) {
		return LLMProviderMatch{
			Provider: LLMProviderWhitelistEntry{
				ID:      "generic-api",
				Name:    "未知 LLM/API 端點",
				Domains: []string{host},
			},
			MatchedBy: "api+llm_hint",
		}, true
	}
	return LLMProviderMatch{}, false
}

func hostMatchesDomain(host string, domain string) bool {
	domain = strings.TrimPrefix(domain, "www.")
	if host == domain {
		return true
	}
	return strings.HasSuffix(host, "."+domain)
}

func hasLLMHint(text string) bool {
	for _, hint := range []string{"ai", "llm", "model", "chat", "completion", "inference", "token", "key"} {
		if strings.Contains(text, hint) {
			return true
		}
	}
	return false
}
