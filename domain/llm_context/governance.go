// llm_context/governance.go — LLM Context 治理層（§11）。
// 控制哪些資料可進入 LLM context，確保禁止項目不外洩。
// 採雙層掃描架構：入口粗篩（正則快速過濾）+ 出口精掃（結構化驗證）。
package llm_context

import (
	"fmt"
	"strings"

	"ui_console/internal/securitytext"
)

// ──────────────────────────────────────────────
// 安全比對前處理（§3.4.1）
// ──────────────────────────────────────────────

// NormalizeForSecurityCheck 在所有安全相關字串比對前統一正規化空白。
// 合併連續空白（space/tab/CR/LF/VT/FF）為單一空格，去除首尾空白。
// 消除 Bearer\tTOKEN 等正規化繞過攻擊。
func NormalizeForSecurityCheck(s string) string {
	return securitytext.NormalizeForSecurityCheck(s)
}

// ──────────────────────────────────────────────
// Source Token / Warning Token 結構
// ──────────────────────────────────────────────

// SourceToken 是注入 LLM context 的來源標記。
// 格式：[SRC:hostname] [RANK:score] [AUTH_OK:bool]
type SourceToken struct {
	Hostname string `json:"hostname"`
	Rank     int    `json:"rank"`
	AuthOK   bool   `json:"auth_ok"` // 僅高影響任務才填入
}

// RenderSourceToken 將 SourceToken 渲染為 context 內嵌格式。
func RenderSourceToken(st SourceToken, includeAuthOK bool) string {
	base := fmt.Sprintf("[SRC:%s] [RANK:%d]", st.Hostname, st.Rank)
	if includeAuthOK {
		base += fmt.Sprintf(" [AUTH_OK:%v]", st.AuthOK)
	}
	return base
}

// ──────────────────────────────────────────────
// Context Payload 組合
// ──────────────────────────────────────────────

// ContextPayload 是送入 LLM 的結構化 context。
type ContextPayload struct {
	SourceTokens  []SourceToken  `json:"source_tokens"`
	WarningTokens []WarningToken `json:"warning_tokens"`
	ContentBlocks []ContentBlock `json:"content_blocks"`
	IsHighImpact  bool           `json:"is_high_impact"`
}

// ContentBlock 是 context 中的內容區塊。
type ContentBlock struct {
	Source  string `json:"source"`  // 來源標識
	Content string `json:"content"` // 實際內容（已通過入口掃描）
	Role    string `json:"role"`    // system / user / reference
}

// BuildContextPayload 組合 LLM context payload。
// 一般任務：SRC + RANK；高影響任務：SRC + RANK + AUTH_OK。
func BuildContextPayload(blocks []ContentBlock, sources []SourceToken, isHighImpact bool) (*ContextPayload, error) {
	payload := &ContextPayload{
		SourceTokens:  sources,
		ContentBlocks: blocks,
		IsHighImpact:  isHighImpact,
	}

	// 入口掃描：粗篩所有 content block
	for i, block := range payload.ContentBlocks {
		cleaned, removed := EntryFilter(block.Content)
		if len(removed) > 0 {
			payload.ContentBlocks[i].Content = cleaned
		}
	}

	// 組合 warning tokens
	payload.WarningTokens = CollectWarningTokens(sources, isHighImpact)

	// 出口掃描：精掃完整 payload
	if err := ExitValidate(payload); err != nil {
		return nil, fmt.Errorf("出口掃描失敗: %w", err)
	}

	return payload, nil
}

// ──────────────────────────────────────────────
// 出口驗證（結構化精掃）
// ──────────────────────────────────────────────

// ExitValidate 對組合完成的 payload 做結構化驗證。
// 確保沒有禁止項目殘留在最終 context 中。
func ExitValidate(payload *ContextPayload) error {
	for _, block := range payload.ContentBlocks {
		// 掃描禁止項目（精確比對 + 結構化檢查）
		for _, pattern := range forbiddenPatterns {
			if strings.Contains(strings.ToLower(block.Content), pattern) {
				return fmt.Errorf("出口掃描偵測到禁止項目: %s (來源: %s)", pattern, block.Source)
			}
		}

		// 結構化檢查：偵測疑似 credential 格式
		if containsCredentialPattern(block.Content) {
			return fmt.Errorf("出口掃描偵測到疑似 credential (來源: %s)", block.Source)
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// 禁止項目清單（§11.2）
// ──────────────────────────────────────────────

// forbiddenPatterns 是入口粗篩 + 出口精掃共用的禁止關鍵字。
// 全部小寫，比對時 toLower。
var forbiddenPatterns = []string{
	// avatar 相關
	"avatar_expression",
	"avatar_mood",
	"avatar_animation_state",
	// credential / 機密
	"auth_cache",
	"api_key=",
	"api_secret",
	"private_key",
	"credential_ref",
	// 敏感資料
	"full_screenshot",
	"readable_local_patch",
	"form_content",
	"payment_info",
	"credit_card",
	// token / cookie
	"session_token=",
	"cookie:",
	"bearer ",
}

// forbiddenStructural 是出口掃描專用的結構化禁止項目。
var forbiddenStructural = []string{
	"-----begin rsa private key-----",
	"-----begin openssh private key-----",
	"-----begin pgp private key-----",
	"sk-",     // OpenAI API key prefix
	"sk_live", // Stripe live key prefix
	"ghp_",    // GitHub personal access token
	"gho_",    // GitHub OAuth token
	"glpat-",  // GitLab personal access token
}

// containsCredentialPattern 偵測疑似 credential 的結構化模式。
func containsCredentialPattern(content string) bool {
	lower := strings.ToLower(content)
	for _, pattern := range forbiddenStructural {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// Payload 序列化輔助
// ──────────────────────────────────────────────

// RenderPayloadTokens 將 payload 的所有 token 渲染為字串。
// 用於注入 LLM prompt。
func RenderPayloadTokens(payload *ContextPayload) string {
	var parts []string
	for _, st := range payload.SourceTokens {
		parts = append(parts, RenderSourceToken(st, payload.IsHighImpact))
	}
	for _, wt := range payload.WarningTokens {
		parts = append(parts, RenderWarningToken(wt))
	}
	return strings.Join(parts, " ")
}
