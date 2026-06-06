// review/lightweight.go — Lightweight Review Card 子型別（§5.2）。
// 僅在 ScopeFingerprint 一致 + 風險 ≤ high_non_destructive 時允許使用。
package review

import (
	"ui_console/domain/risk"
)

// ──────────────────────────────────────────────
// Lightweight Review Card（§5.2.1）
// ──────────────────────────────────────────────

// LightweightCard 是簡化版 Review Card，用於低摩擦的續期或確認場景。
// 例如：來源信任 allowlist 續期。
type LightweightCard struct {
	// §5.2.1 必要欄位（僅 5 個）
	ReviewID    string `json:"review_id"`
	Operation   string `json:"operation"`
	Target      string `json:"target"`
	AcceptLabel string `json:"accept_label"`
	RejectLabel string `json:"reject_label"`

	// §5.2.2 選填欄位（放在「展開工程細節」區塊）
	Details map[string]interface{} `json:"details,omitempty"`

	// 狀態追蹤
	CreatedAt  string `json:"created_at"`
	Resolved   bool   `json:"resolved"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

// LightweightParams 建立 LightweightCard 時的參數。
type LightweightParams struct {
	Operation   string
	Target      string
	AcceptLabel string
	RejectLabel string
	Details     map[string]interface{}
}

// NewLightweightCard 建立 Lightweight Review Card。
func NewLightweightCard(params LightweightParams) LightweightCard {
	return LightweightCard{
		ReviewID:    generateID("rev_light"),
		Operation:   params.Operation,
		Target:      params.Target,
		AcceptLabel: params.AcceptLabel,
		RejectLabel: params.RejectLabel,
		Details:     params.Details,
		CreatedAt:   nowRFC3339(),
		Resolved:    false,
	}
}

// ──────────────────────────────────────────────
// 使用範圍限制（§5.2.3）
// ──────────────────────────────────────────────

// IsLightweightAllowed 判斷是否允許使用 Lightweight Review Card。
// 條件（全部必須滿足）：
//   - scopeMatch: ScopeFingerprint 一致（無 scope 擴展）
//   - riskClass: 風險等級不超過 high_non_destructive
//
// 若任一條件不滿足，系統必須升級為 Standard Review Card。
func IsLightweightAllowed(scopeMatch bool, riskClass risk.RiskClass) bool {
	if !scopeMatch {
		return false
	}
	// 風險等級 ≤ high_non_destructive
	return risk.IsHigherOrEqual(risk.HighNonDestructive, riskClass)
}
