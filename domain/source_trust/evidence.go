// source_trust/evidence.go — 來源信任證據結構（§9.1）。
// GatewaySentinel 分類後產出 SourceTrustEvidence，供 RAG / UI / Review Card 使用。
package source_trust

// SourceTrustEvidence 是 GatewaySentinel 分類後的完整證據記錄。
// controller 根據此證據決定 UI 標籤、是否 blocking、是否產生 Review Card。
type SourceTrustEvidence struct {
	// 來源 URL（原始輸入）
	SourceURL string `json:"source_url"`

	// 正規化後的主機名稱
	CanonicalHostname string `json:"canonical_hostname"`

	// 網域分類
	DomainClass DomainClass `json:"domain_class"`

	// 最終信任標籤
	SourceTrustLabel SourceTrustLabel `json:"source_trust_label"`

	// 視覺證據旗標（來自前端 DOM/截圖分析）
	VisualFlags []VisualFlag `json:"visual_flags,omitempty"`

	// 內容指紋旗標（來自 content snippet 掃描）
	ContentFlags []ContentFlag `json:"content_flags,omitempty"`

	// Allowlist 狀態
	AllowlistStatus string `json:"allowlist_status"` // "active" | "expired" | "not_listed"

	// 排序分數（可被 UGC quality signals 調整，但不改變 label）
	RankingScore int `json:"ranking_score"`

	// AUTH_OK：僅當 VERIFIED_AUTHORITY + allowlist 未過期 + 無 UGC/disclaimer
	AuthOK bool `json:"auth_ok"`

	// 是否需要 Review Card
	ReviewRequired bool `json:"review_required"`

	// Review 原因（若 ReviewRequired 為 true）
	ReviewReason string `json:"review_reason,omitempty"`

	// 是否為高影響領域
	IsHighImpact bool `json:"is_high_impact"`

	// 警告 token（供 LLM context 使用的最小化信號）
	WarningTokens []string `json:"warning_tokens,omitempty"`
}
