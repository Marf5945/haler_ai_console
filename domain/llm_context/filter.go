// llm_context/filter.go — 雙層掃描過濾器（§11.2 / §3.4.1）。
// 入口粗篩：正規化 + 正則快速過濾明顯禁止項目。
// 出口精掃：結構化驗證（在 governance.go ExitValidate 中）。
package llm_context

import (
	"regexp"
	"strings"
)

// ──────────────────────────────────────────────
// 入口粗篩（Entry Filter）
// ──────────────────────────────────────────────

// EntryFilter 在資料進入 context 前做粗篩。
// 先正規化空白（§3.4.1），再移除禁止項目，回傳清理後內容 + 被移除清單。
func EntryFilter(content string) (cleaned string, removed []string) {
	// §3.4.1 逐行正規化——消除 tab/多空格繞過，但保留換行結構（禁止關鍵字是逐行比對）
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = NormalizeForSecurityCheck(line)
	}
	cleaned = strings.Join(lines, "\n")

	// 1. 移除疑似 API key / token
	for _, re := range entryRegexps {
		matches := re.FindAllString(cleaned, -1)
		for _, m := range matches {
			removed = append(removed, truncateForLog(m))
		}
		cleaned = re.ReplaceAllString(cleaned, "[REDACTED]")
	}

	// 2. 移除禁止關鍵字所在行
	lines = strings.Split(cleaned, "\n")
	var filteredLines []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		blocked := false
		for _, pattern := range forbiddenPatterns {
			if strings.Contains(lower, pattern) {
				removed = append(removed, truncateForLog(line))
				blocked = true
				break
			}
		}
		if !blocked {
			filteredLines = append(filteredLines, line)
		}
	}
	cleaned = strings.Join(filteredLines, "\n")
	return cleaned, removed
}

// ──────────────────────────────────────────────
// 外部 token 逃脫（§11.3）
// ──────────────────────────────────────────────

// EscapeExternalTokens 將外部內容中的系統 token 逃脫，防止偽造。
func EscapeExternalTokens(content string) string {
	result := content
	result = tokenEscapeRe.ReplaceAllStringFunc(result, func(match string) string {
		return strings.ReplaceAll(strings.ReplaceAll(match, "[", "［"), "]", "］")
	})
	result = warnEscapeRe.ReplaceAllStringFunc(result, func(match string) string {
		return strings.ReplaceAll(strings.ReplaceAll(match, "⟦", "〔"), "⟧", "〕")
	})
	return result
}

// ──────────────────────────────────────────────
// 入口掃描正則（§3.4.2 Layer 1 同步）
// ──────────────────────────────────────────────

var entryRegexps = []*regexp.Regexp{
	// 各供應商 API key——與 redaction.go builtinRules 同步
	regexp.MustCompile(`(?i)\bsk-ant-[A-Za-z0-9\-]{20,}`),            // Anthropic（先比對長前綴）
	regexp.MustCompile(`(?i)\bsk-or-v1-[A-Za-z0-9]{20,}`),            // OpenRouter
	regexp.MustCompile(`(?i)\b(sk|pk)_(test|live)_[A-Za-z0-9]{24,}`), // Stripe
	regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9]{20,}`),                  // OpenAI（放在 Anthropic/OpenRouter 之後）
	regexp.MustCompile(`\br8_[A-Za-z0-9]{20,}`),                      // Replicate
	regexp.MustCompile(`\bgh[ps]_[A-Za-z0-9]{36,}`),                  // GitHub PAT
	regexp.MustCompile(`\bgho_[A-Za-z0-9]{36,}`),                     // GitHub OAuth
	regexp.MustCompile(`\b(ghu|ghs|ghr)_[A-Za-z0-9]{36,}`),           // GitHub App
	regexp.MustCompile(`\bglpat-[A-Za-z0-9]{20,}`),                   // GitLab
	regexp.MustCompile(`\bAIza[A-Za-z0-9\-_]{35}`),                   // Google AI
	regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`),                       // AWS
	regexp.MustCompile(`\bhf_[A-Za-z0-9]{34,}`),                      // HuggingFace
	// Bearer token
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`),
	// PEM 私鑰
	regexp.MustCompile(`(?s)-----BEGIN\s+([\w\s]*)?PRIVATE\s+KEY-----.*?-----END\s+([\w\s]*)?PRIVATE\s+KEY-----`),
	// key=value 通用
	regexp.MustCompile(`(?i)(password|secret|token|api_key|apikey|credential|auth_token|access_token|secret_key)\s*[=:]\s*\S+`),
}

// token 逃脫正則
var tokenEscapeRe = regexp.MustCompile(`\[(SRC|RANK|AUTH_OK):[^\]]*\]`)
var warnEscapeRe = regexp.MustCompile(`⟦SRC_WARN:[^⟧]*⟧`)

// ──────────────────────────────────────────────
// 輔助
// ──────────────────────────────────────────────

// truncateForLog 截斷用於移除紀錄。
func truncateForLog(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
