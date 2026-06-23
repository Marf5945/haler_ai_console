// source_trust/classifier.go — GatewaySentinel 核心分類器（§9.1–§9.4）。
// 接收 URL + content snippet + visual flags，產出 SourceTrustEvidence。
package source_trust

import (
	"net/url"
	"strings"
)

// ──────────────────────────────────────────────
// 內容指紋偵測關鍵字（§9.4 中英文）
// ──────────────────────────────────────────────

var ugcFingerprints = []string{
	"disclaimer", "user-generated", "comment", "post by",
	"forum", "discussion", "community", "wiki edit",
	"網友", "論壇", "留言", "評論", "免責聲明", "使用者產生內容",
}

// ──────────────────────────────────────────────
// 主分類函式
// ──────────────────────────────────────────────

// Classify 是 GatewaySentinel 的核心入口。
// 接收來源 URL、內容片段、視覺旗標，回傳完整的信任證據。
// 此函式由 controller 呼叫，LLM 不得自行判定信任。
func Classify(sourceURL string, contentSnippet string, visualFlags []VisualFlag) SourceTrustEvidence {
	evidence := SourceTrustEvidence{
		SourceURL:       sourceURL,
		AllowlistStatus: "not_listed",
		RankingScore:    50, // 預設中位分數
	}

	// 步驟 a: 解析正規化主機名稱
	evidence.CanonicalHostname = resolveCanonicalHostname(sourceURL)

	// 步驟 b: 網域分類
	evidence.DomainClass = classifyDomain(evidence.CanonicalHostname, sourceURL)

	// 步驟 c: 內容指紋掃描
	evidence.ContentFlags = scanContentFingerprints(contentSnippet)

	// 步驟 d: 視覺證據評估
	evidence.VisualFlags = evaluateVisualEvidence(visualFlags)

	// 步驟 e: 綜合決定 label
	evidence.SourceTrustLabel = computeLabel(evidence)

	// 產生警告 token（供 LLM context 最小化使用）
	evidence.WarningTokens = buildWarningTokens(evidence)

	return evidence
}

// ──────────────────────────────────────────────
// 步驟 a: 正規化主機名稱
// ──────────────────────────────────────────────

func resolveCanonicalHostname(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	// 補上 scheme（讓 url.Parse 能正確解析）
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	// 去除 www. 前綴
	host = strings.TrimPrefix(host, "www.")
	return host
}

// ──────────────────────────────────────────────
// 步驟 b: 網域分類
// ──────────────────────────────────────────────

func classifyDomain(hostname, rawURL string) DomainClass {
	if hostname == "" {
		return DomainUnknown
	}

	// 可疑網域偵測（IP 位址、過短、含可疑字元）
	if isSuspiciousDomain(hostname) {
		return DomainSuspicious
	}

	// UGC 子路徑偵測（即使是 .edu/.gov 也可能有 UGC 區塊）
	lowURL := strings.ToLower(rawURL)
	if containsUGCPath(lowURL) {
		return DomainUserGeneratedSubpath
	}

	// .gov 網域（弱正面信號 §9.4）
	if strings.HasSuffix(hostname, ".gov") || strings.HasSuffix(hostname, ".gov.tw") {
		return DomainOfficialGov
	}

	// .edu 網域（弱正面信號 §9.4）
	if strings.HasSuffix(hostname, ".edu") || strings.HasSuffix(hostname, ".edu.tw") {
		return DomainOfficialEdu
	}

	// 機構子網域偵測
	if isInstitutionalSubdomain(hostname) {
		return DomainInstitutionalSubdomain
	}

	return DomainUnknown
}

// isSuspiciousDomain 偵測可疑網域（IP、過短等）。
func isSuspiciousDomain(hostname string) bool {
	// 純 IP 位址
	parts := strings.Split(hostname, ".")
	if len(parts) == 4 {
		allDigits := true
		for _, p := range parts {
			for _, c := range p {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
		}
		if allDigits {
			return true
		}
	}
	// 過短（如 a.tk）
	if len(hostname) < 4 {
		return true
	}
	return false
}

// containsUGCPath 偵測 URL 路徑中是否包含 UGC 區段。
func containsUGCPath(rawURL string) bool {
	ugcPaths := []string{
		"/forum", "/discussion", "/comments", "/community",
		"/wiki/", "/talk:", "/user:", "/blog/",
		"/questions/", "/answers/",
	}
	low := strings.ToLower(rawURL)
	for _, p := range ugcPaths {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// isInstitutionalSubdomain 偵測機構子網域模式。
func isInstitutionalSubdomain(hostname string) bool {
	institutional := []string{
		".org", ".ac.", ".go.", ".gob.",
		".museum", ".int", ".mil",
	}
	for _, suffix := range institutional {
		if strings.Contains(hostname, suffix) {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// 步驟 c: 內容指紋掃描
// ──────────────────────────────────────────────

func scanContentFingerprints(snippet string) []ContentFlag {
	if snippet == "" {
		return nil
	}
	low := strings.ToLower(snippet)
	var flags []ContentFlag
	seen := make(map[ContentFlag]bool)

	for _, fp := range ugcFingerprints {
		if strings.Contains(low, fp) {
			flag := fingerprintToFlag(fp)
			if !seen[flag] {
				flags = append(flags, flag)
				seen[flag] = true
			}
		}
	}
	return flags
}

// fingerprintToFlag 將關鍵字映射到 ContentFlag。
func fingerprintToFlag(fp string) ContentFlag {
	switch fp {
	case "disclaimer", "免責聲明":
		return ContentDisclaimer
	case "user-generated", "使用者產生內容":
		return ContentUserGenerated
	case "comment", "留言", "評論":
		return ContentComment
	case "forum", "論壇":
		return ContentForum
	case "discussion":
		return ContentDiscussion
	case "community":
		return ContentCommunity
	case "wiki edit":
		return ContentWikiEdit
	case "post by":
		return ContentComment
	case "網友":
		return ContentUserGenerated
	default:
		return ContentUserGenerated
	}
}

// ──────────────────────────────────────────────
// 步驟 d: 視覺證據評估（§9.3）
// ──────────────────────────────────────────────

func evaluateVisualEvidence(flags []VisualFlag) []VisualFlag {
	// 視覺證據只能當作 evidence，不能自動提升信任分數
	// 直接回傳，不做額外處理（禁止 logo → trust score increase）
	return flags
}

// ──────────────────────────────────────────────
// 步驟 e: 綜合決定 label
// ──────────────────────────────────────────────

func computeLabel(e SourceTrustEvidence) SourceTrustLabel {
	hasUGCContent := len(e.ContentFlags) > 0
	hasVisualClaim := len(e.VisualFlags) > 0

	// 規則 1: 有 UGC 內容指紋 → USER_GENERATED
	// .edu/.gov 不得覆蓋 UGC 降級（§9.4 硬規則）
	if hasUGCContent {
		return LabelUserGenerated
	}

	// 規則 2: UGC 子路徑 → USER_GENERATED
	if e.DomainClass == DomainUserGeneratedSubpath {
		return LabelUserGenerated
	}

	// 規則 3: 可疑網域 → LOW_TRUST
	if e.DomainClass == DomainSuspicious {
		return LabelLowTrust
	}

	// 規則 4: 有官方視覺標記但網域未驗證 → PENDING_SOURCE_REVIEW（§9.3）
	if hasVisualClaim && e.DomainClass == DomainUnknown {
		return LabelPendingSourceReview
	}

	// 規則 5: .gov/.edu 網域（弱正面信號）
	if e.DomainClass == DomainOfficialGov || e.DomainClass == DomainOfficialEdu {
		return LabelInstitutionalButUnverified
	}

	// 規則 6: 機構子網域
	if e.DomainClass == DomainInstitutionalSubdomain {
		return LabelInstitutionalButUnverified
	}

	// 規則 7: 有視覺標記 + 機構/gov/edu → INSTITUTIONAL（不自動升 VERIFIED）
	if hasVisualClaim && (e.DomainClass == DomainOfficialGov || e.DomainClass == DomainOfficialEdu || e.DomainClass == DomainInstitutionalSubdomain) {
		return LabelInstitutionalButUnverified
	}

	// 預設：UNVERIFIED
	return LabelUnverified
}

// ──────────────────────────────────────────────
// 警告 token（供 LLM context 最小化使用）
// ──────────────────────────────────────────────

func buildWarningTokens(e SourceTrustEvidence) []string {
	var tokens []string

	switch e.SourceTrustLabel {
	case LabelLowTrust:
		tokens = append(tokens, "⚠ low_trust_source")
	case LabelUserGenerated:
		tokens = append(tokens, "📝 user_generated_content")
	case LabelPendingSourceReview:
		tokens = append(tokens, "🔍 pending_source_review")
	case LabelUnverified:
		tokens = append(tokens, "❓ unverified_source")
	}

	if len(e.ContentFlags) > 0 {
		tokens = append(tokens, "📌 ugc_content_detected")
	}

	return tokens
}
