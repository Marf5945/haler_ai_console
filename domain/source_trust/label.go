// Package source_trust 實作 GatewaySentinel 來源信任分類系統（§9）。
// 所有外部來源在進入 RAG / LLM context / citation 前，都必須經過本模組分類。
package source_trust

// ──────────────────────────────────────────────
// 來源信任標籤（§9.2）
// ──────────────────────────────────────────────

// SourceTrustLabel 描述一個外部來源的信任等級。
type SourceTrustLabel string

const (
	LabelVerifiedAuthority          SourceTrustLabel = "VERIFIED_AUTHORITY"
	LabelInstitutionalButUnverified SourceTrustLabel = "INSTITUTIONAL_BUT_UNVERIFIED"
	LabelUnverified                 SourceTrustLabel = "UNVERIFIED"
	LabelLowTrust                   SourceTrustLabel = "LOW_TRUST"
	LabelUserGenerated              SourceTrustLabel = "USER_GENERATED"
	LabelPendingSourceReview        SourceTrustLabel = "PENDING_SOURCE_REVIEW"
)

// UserFriendlyLabel 回傳來源信任標籤的使用者友善中文（§6.5 UI 標籤）。
func UserFriendlyLabel(l SourceTrustLabel) string {
	switch l {
	case LabelVerifiedAuthority:
		return "已驗證"
	case LabelInstitutionalButUnverified:
		return "機構來源（未驗證）"
	case LabelUnverified:
		return "未驗證"
	case LabelLowTrust:
		return "低信任"
	case LabelUserGenerated:
		return "社群來源"
	case LabelPendingSourceReview:
		return "待審查"
	default:
		return "未知"
	}
}

// ──────────────────────────────────────────────
// 網域分類（§9.4）
// ──────────────────────────────────────────────

// DomainClass 描述網域的分類。
type DomainClass string

const (
	DomainOfficialGov          DomainClass = "official_gov_domain"
	DomainOfficialEdu          DomainClass = "official_edu_domain"
	DomainInstitutionalSubdomain DomainClass = "institutional_subdomain"
	DomainUserGeneratedSubpath DomainClass = "user_generated_subpath"
	DomainUnknown              DomainClass = "unknown_domain"
	DomainSuspicious           DomainClass = "suspicious_domain"
)

// ──────────────────────────────────────────────
// 視覺證據旗標（§9.3）
// ──────────────────────────────────────────────

// VisualFlag 描述頁面上偵測到的視覺證據。
type VisualFlag string

const (
	VisualOfficialLogo       VisualFlag = "official_logo_detected"
	VisualInstitutionalBadge VisualFlag = "institutional_badge"
	VisualGovernmentSeal     VisualFlag = "government_seal"
	VisualVerifiedCheckmark  VisualFlag = "verified_checkmark"
)

// ──────────────────────────────────────────────
// 內容指紋旗標（§9.4）
// ──────────────────────────────────────────────

// ContentFlag 描述內容中偵測到的信任降級信號。
type ContentFlag string

const (
	ContentDisclaimer    ContentFlag = "disclaimer"
	ContentUserGenerated ContentFlag = "user_generated"
	ContentComment       ContentFlag = "comment"
	ContentForum         ContentFlag = "forum"
	ContentDiscussion    ContentFlag = "discussion"
	ContentCommunity     ContentFlag = "community"
	ContentWikiEdit      ContentFlag = "wiki_edit"
)
