// source_trust/allowlist.go — Project Source Allowlist（§9.8–§9.9）。
// 使用者可將來源加入專案白名單（需 Review Card），
// 續期時若 ScopeFingerprint 一致可用 Lightweight Review Card。
package source_trust

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// Allowlist Entry 結構（§9.8）
// ──────────────────────────────────────────────

// AllowlistEntry 代表一筆白名單記錄。
type AllowlistEntry struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"project_id"`
	CanonicalHostname string    `json:"canonical_hostname"`
	URLPattern        string    `json:"url_pattern"`
	URLPatternHash    string    `json:"url_pattern_hash"`
	ContentType       []string  `json:"content_type"`
	SourcePurpose     []string  `json:"source_purpose"`
	AllowedFor        []string  `json:"allowed_for"`
	NotAllowedFor     []string  `json:"not_allowed_for"`
	Expiry            time.Time `json:"expiry"`
	CreatedAt         time.Time `json:"created_at"`
	RenewedAt         *time.Time `json:"renewed_at,omitempty"`

	// ScopeFingerprint 用於續期比對
	ScopeFingerprint string `json:"scope_fingerprint"`
}

// ──────────────────────────────────────────────
// Not-allowed-for 必要清單（§9.8 硬規則）
// ──────────────────────────────────────────────

// RequiredNotAllowedFor 是所有白名單必須包含的 not_allowed_for 項目。
var RequiredNotAllowedFor = []string{
	"confirmed_rule",
	"user_intent",
	"security_policy",
	"risk_policy",
	"tool_registry",
	"adapter_permission",
}

// ──────────────────────────────────────────────
// ScopeFingerprint（§9.9）
// ──────────────────────────────────────────────

// ScopeFingerprintInput 是計算 fingerprint 的輸入欄位。
// 注意：expiry 不包含在內（因為續期會改變 expiry）。
type ScopeFingerprintInput struct {
	CanonicalHostname string   `json:"canonical_hostname"`
	URLPatternHash    string   `json:"url_pattern_hash"`
	ContentType       []string `json:"content_type"`
	SourcePurpose     []string `json:"source_purpose"`
	AllowedFor        []string `json:"allowed_for"`
	NotAllowedFor     []string `json:"not_allowed_for"`
}

// ComputeFingerprint 計算 ScopeFingerprint。
// 將所有欄位串接後 SHA-256 雜湊。
func ComputeFingerprint(input ScopeFingerprintInput) string {
	// 排序所有 slice 確保一致性
	sort.Strings(input.ContentType)
	sort.Strings(input.SourcePurpose)
	sort.Strings(input.AllowedFor)
	sort.Strings(input.NotAllowedFor)

	parts := []string{
		input.CanonicalHostname,
		input.URLPatternHash,
		strings.Join(input.ContentType, ","),
		strings.Join(input.SourcePurpose, ","),
		strings.Join(input.AllowedFor, ","),
		strings.Join(input.NotAllowedFor, ","),
	}

	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("sha256:%x", hash)
}

// ComputeURLPatternHash 計算 URL pattern 的雜湊值。
func ComputeURLPatternHash(pattern string) string {
	hash := sha256.Sum256([]byte(pattern))
	return fmt.Sprintf("sha256:%x", hash)
}

// ──────────────────────────────────────────────
// Allowlist Entry 建構與驗證
// ──────────────────────────────────────────────

// NewAllowlistEntry 建立新的白名單記錄。
// 自動填入 not_allowed_for 必要項目、計算 fingerprint。
// 預設有效期 30 天。
func NewAllowlistEntry(projectID, hostname, urlPattern string, contentType, sourcePurpose, allowedFor []string) AllowlistEntry {
	now := time.Now()
	patternHash := ComputeURLPatternHash(urlPattern)

	// 確保 not_allowed_for 包含所有必要項目
	notAllowed := ensureRequiredNotAllowed(nil)

	entry := AllowlistEntry{
		ID:                fmt.Sprintf("al_%d", now.UnixNano()),
		ProjectID:         projectID,
		CanonicalHostname: hostname,
		URLPattern:        urlPattern,
		URLPatternHash:    patternHash,
		ContentType:       contentType,
		SourcePurpose:     sourcePurpose,
		AllowedFor:        allowedFor,
		NotAllowedFor:     notAllowed,
		Expiry:            now.Add(30 * 24 * time.Hour), // 預設 30 天
		CreatedAt:         now,
	}

	// 計算 fingerprint
	entry.ScopeFingerprint = ComputeFingerprint(ScopeFingerprintInput{
		CanonicalHostname: entry.CanonicalHostname,
		URLPatternHash:    entry.URLPatternHash,
		ContentType:       entry.ContentType,
		SourcePurpose:     entry.SourcePurpose,
		AllowedFor:        entry.AllowedFor,
		NotAllowedFor:     entry.NotAllowedFor,
	})

	return entry
}

// ensureRequiredNotAllowed 確保 not_allowed_for 包含所有必要項目。
func ensureRequiredNotAllowed(existing []string) []string {
	set := make(map[string]bool)
	for _, e := range existing {
		set[e] = true
	}
	for _, req := range RequiredNotAllowedFor {
		set[req] = true
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// ──────────────────────────────────────────────
// 續期與到期檢查
// ──────────────────────────────────────────────

// IsExpired 檢查白名單是否已過期。
func (e *AllowlistEntry) IsExpired() bool {
	return time.Now().After(e.Expiry)
}

// IsExpiringSoon 檢查是否在 3 天內到期。
func (e *AllowlistEntry) IsExpiringSoon() bool {
	return !e.IsExpired() && time.Until(e.Expiry) <= 3*24*time.Hour
}

// ScopeMatches 判斷當前 entry 的 scope 是否與指定 fingerprint 一致。
// 一致時可用 Lightweight Review Card 續期。
func (e *AllowlistEntry) ScopeMatches(currentFingerprint string) bool {
	return e.ScopeFingerprint == currentFingerprint
}

// Renew 續期（延長 30 天）。
func (e *AllowlistEntry) Renew() {
	now := time.Now()
	e.Expiry = now.Add(30 * 24 * time.Hour)
	e.RenewedAt = &now
}
