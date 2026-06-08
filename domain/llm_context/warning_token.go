// llm_context/warning_token.go — Warning Token 定義與渲染（§11.1）。
// Warning token 由 controller 產生，LLM 不得自行生成。
// UI 模式替換為標籤，CLI 模式替換為文字警語。
package llm_context

import (
	"fmt"
)

// ──────────────────────────────────────────────
// Warning Token 型別
// ──────────────────────────────────────────────

// WarningTokenType 定義警告 token 的類型。
type WarningTokenType string

const (
	WarnUnverifiedSource WarningTokenType = "UNVERIFIED_SOURCE"
	WarnLowTrustSource   WarningTokenType = "LOW_TRUST_SOURCE"
	WarnPendingReview    WarningTokenType = "PENDING_REVIEW"
	WarnHighImpactNoAuth WarningTokenType = "HIGH_IMPACT_NO_AUTH"
)

// WarningToken 是注入 LLM context 的警告標記。
// 格式：⟦SRC_WARN:type:detail⟧
type WarningToken struct {
	Type   WarningTokenType `json:"type"`
	Detail string           `json:"detail"` // 補充資訊（如 hostname）
}

// ──────────────────────────────────────────────
// Token 渲染
// ──────────────────────────────────────────────

// RenderWarningToken 將 WarningToken 渲染為 context 內嵌格式。
func RenderWarningToken(wt WarningToken) string {
	return fmt.Sprintf("⟦SRC_WARN:%s:%s⟧", wt.Type, wt.Detail)
}

// RenderForUI 將 WarningToken 替換為 UI 標籤（中文）。
func RenderForUI(wt WarningToken) string {
	switch wt.Type {
	case WarnUnverifiedSource:
		return fmt.Sprintf("⚠ 未驗證來源：%s", wt.Detail)
	case WarnLowTrustSource:
		return fmt.Sprintf("🔴 低信任來源：%s", wt.Detail)
	case WarnPendingReview:
		return fmt.Sprintf("🟡 待審查來源：%s", wt.Detail)
	case WarnHighImpactNoAuth:
		return fmt.Sprintf("🔴 高影響任務缺少授權驗證：%s", wt.Detail)
	default:
		return fmt.Sprintf("⚠ 警告：%s", wt.Detail)
	}
}

// RenderForCLI 將 WarningToken 替換為 CLI 文字警語。
func RenderForCLI(wt WarningToken) string {
	switch wt.Type {
	case WarnUnverifiedSource:
		return fmt.Sprintf("[WARNING] Unverified source: %s", wt.Detail)
	case WarnLowTrustSource:
		return fmt.Sprintf("[WARNING] Low-trust source: %s", wt.Detail)
	case WarnPendingReview:
		return fmt.Sprintf("[WARNING] Pending source review: %s", wt.Detail)
	case WarnHighImpactNoAuth:
		return fmt.Sprintf("[WARNING] High-impact task without auth verification: %s", wt.Detail)
	default:
		return fmt.Sprintf("[WARNING] %s", wt.Detail)
	}
}

// ──────────────────────────────────────────────
// Warning Token 收集
// ──────────────────────────────────────────────

// CollectWarningTokens 根據 source 狀態收集需要的 warning tokens。
// 此函式由 controller 呼叫，LLM 不得自行產生 warning token。
func CollectWarningTokens(sources []SourceToken, isHighImpact bool) []WarningToken {
	var tokens []WarningToken

	for _, src := range sources {
		// 排名極低 → 低信任警告
		if src.Rank < 20 {
			tokens = append(tokens, WarningToken{
				Type:   WarnLowTrustSource,
				Detail: src.Hostname,
			})
			continue
		}

		// 排名中低 → 未驗證警告
		if src.Rank < 50 {
			tokens = append(tokens, WarningToken{
				Type:   WarnUnverifiedSource,
				Detail: src.Hostname,
			})
		}

		// 高影響任務但 AUTH_OK 為 false
		if isHighImpact && !src.AuthOK {
			tokens = append(tokens, WarningToken{
				Type:   WarnHighImpactNoAuth,
				Detail: src.Hostname,
			})
		}
	}

	return tokens
}
