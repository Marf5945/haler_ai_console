// spec_patch_checker/checker_source_trust.go — v3.5.0 GatewaySentinel 守則（§9）。
// 共 15 條守則，確保來源信任系統的硬規則不被繞過。
package spec_patch_checker

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ──────────────────────────────────────────────
// §9 GatewaySentinel 守則
// ──────────────────────────────────────────────

// CheckLLMDoesNotDecideHighImpact 驗證 LLM 不自行判定高影響任務的來源信任。
// §9.6：高影響任務的 AUTH_OK 必須由 controller 計算。
func CheckLLMDoesNotDecideHighImpact(payloadJSON string) error {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}
	if source, ok := payload["auth_ok_source"]; ok {
		if source == "llm" || source == "model" {
			return fmt.Errorf("§9.6 違規: AUTH_OK 不得由 LLM 自行判定")
		}
	}
	return nil
}

// CheckUserCannotRemoveBuiltInHighImpactDomain 驗證使用者不能移除內建高影響領域。
func CheckUserCannotRemoveBuiltInHighImpactDomain(configJSON string) error {
	var config struct {
		RemovedDomains []string `json:"removed_domains"`
	}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil // 無效 JSON 非此守則範圍
	}
	builtIn := map[string]bool{
		"health": true, "medical": true, "legal": true,
		"financial": true, "tax": true, "insurance": true,
		"government_id": true, "immigration": true, "security_config": true,
	}
	for _, d := range config.RemovedDomains {
		if builtIn[d] {
			return fmt.Errorf("§9.6 違規: 不得移除內建高影響領域 %s", d)
		}
	}
	return nil
}

// CheckProjectDoesNotWeakenGlobalHighImpact 驗證專案層不削弱全域高影響設定。
func CheckProjectDoesNotWeakenGlobalHighImpact(globalJSON, projectJSON string) error {
	var global, project struct {
		HighImpact []string `json:"high_impact_domains"`
	}
	json.Unmarshal([]byte(globalJSON), &global)
	json.Unmarshal([]byte(projectJSON), &project)

	globalSet := make(map[string]bool)
	for _, d := range global.HighImpact {
		globalSet[d] = true
	}
	for _, d := range global.HighImpact {
		found := false
		for _, pd := range project.HighImpact {
			if pd == d {
				found = true
				break
			}
		}
		if !found && globalSet[d] {
			return fmt.Errorf("§9.6 違規: 專案層不得削弱全域高影響領域 %s", d)
		}
	}
	return nil
}

// CheckVisualLogoDoesNotIncreaseScore 驗證視覺 logo 不自動加信任分。
func CheckVisualLogoDoesNotIncreaseScore(evidenceJSON string) error {
	var ev struct {
		VisualFlags   []string `json:"visual_flags"`
		ScoreIncrease int      `json:"score_increase_from_visual"`
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &ev); err != nil {
		return nil
	}
	if ev.ScoreIncrease > 0 {
		for _, f := range ev.VisualFlags {
			if f == "official_logo_detected" || f == "institutional_badge" {
				return fmt.Errorf("§9.4 違規: 視覺 logo 不得自動增加信任分")
			}
		}
	}
	return nil
}

// CheckUserGeneratedDoesNotBecomeVerifiedAuthority 驗證 UGC 不得提升為 VERIFIED_AUTHORITY。
func CheckUserGeneratedDoesNotBecomeVerifiedAuthority(evidenceJSON string) error {
	var ev struct {
		ContentFlags []string `json:"content_flags"`
		Label        string   `json:"label"`
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &ev); err != nil {
		return nil
	}
	hasUGC := false
	for _, f := range ev.ContentFlags {
		if f == "ugc_content" || f == "forum_post" || f == "user_comment" {
			hasUGC = true
			break
		}
	}
	if hasUGC && ev.Label == "VERIFIED_AUTHORITY" {
		return fmt.Errorf("§9.4 違規: UGC 內容不得被標記為 VERIFIED_AUTHORITY")
	}
	return nil
}

// CheckUserGeneratedDoesNotProduceAuthOK 驗證 UGC 不得產生 AUTH_OK。
func CheckUserGeneratedDoesNotProduceAuthOK(evidenceJSON string) error {
	var ev struct {
		ContentFlags []string `json:"content_flags"`
		AuthOK       bool     `json:"auth_ok"`
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &ev); err != nil {
		return nil
	}
	for _, f := range ev.ContentFlags {
		if (f == "ugc_content" || f == "forum_post") && ev.AuthOK {
			return fmt.Errorf("§9.4 違規: UGC 內容不得產生 AUTH_OK")
		}
	}
	return nil
}

// CheckHighImpactTaskIncludesAuthOK 驗證高影響任務必須包含 AUTH_OK。
func CheckHighImpactTaskIncludesAuthOK(contextJSON string) error {
	var ctx struct {
		IsHighImpact bool `json:"is_high_impact"`
		HasAuthOK    bool `json:"has_auth_ok"`
	}
	if err := json.Unmarshal([]byte(contextJSON), &ctx); err != nil {
		return nil
	}
	if ctx.IsHighImpact && !ctx.HasAuthOK {
		return fmt.Errorf("§9.6 違規: 高影響任務的 context 必須包含 AUTH_OK")
	}
	return nil
}

// CheckWarningTokenNotGeneratedByLLMAlone 驗證 warning token 不由 LLM 單獨產生。
func CheckWarningTokenNotGeneratedByLLMAlone(tokenJSON string) error {
	var token struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil
	}
	if token.Source == "llm" || token.Source == "model_generated" {
		return fmt.Errorf("§9 違規: warning token 不得由 LLM 單獨產生")
	}
	return nil
}

// CheckExternalWarningTokenEscaped 驗證外部內容中的 warning token 已逃脫。
func CheckExternalWarningTokenEscaped(content string) error {
	if strings.Contains(content, "⟦SRC_WARN:") && !strings.Contains(content, "〔SRC_WARN:") {
		// 有原始 token 且沒有逃脫版本，可能未逃脫
		if strings.Contains(content, "[SRC:") {
			return fmt.Errorf("§11.3 違規: 外部內容中的系統 token 未逃脫")
		}
	}
	return nil
}

// CheckCLIOutputShowsWarning 驗證 CLI 輸出包含適當的警告。
func CheckCLIOutputShowsWarning(outputJSON string) error {
	var output struct {
		HasWarningSource bool `json:"has_warning_source"`
		ShowsWarning     bool `json:"shows_warning"`
	}
	if err := json.Unmarshal([]byte(outputJSON), &output); err != nil {
		return nil
	}
	if output.HasWarningSource && !output.ShowsWarning {
		return fmt.Errorf("§9 違規: 包含警告來源的 CLI 輸出必須顯示警告")
	}
	return nil
}

// CheckAllowlistHasScopeAndExpiry 驗證白名單條目必須有 scope 和過期時間。
func CheckAllowlistHasScopeAndExpiry(allowlistJSON string) error {
	var entry struct {
		AllowedFor    []string `json:"allowed_for"`
		NotAllowedFor []string `json:"not_allowed_for"`
		ExpiresAt     string   `json:"expires_at"`
	}
	if err := json.Unmarshal([]byte(allowlistJSON), &entry); err != nil {
		return nil
	}
	if len(entry.AllowedFor) == 0 {
		return fmt.Errorf("§9.8 違規: 白名單條目必須有 allowed_for scope")
	}
	if entry.ExpiresAt == "" {
		return fmt.Errorf("§9.8 違規: 白名單條目必須有過期時間")
	}
	if len(entry.NotAllowedFor) == 0 {
		return fmt.Errorf("§9.8 違規: 白名單條目必須有 not_allowed_for")
	}
	return nil
}

// CheckAllowlistRenewalDoesNotExpandScope 驗證白名單續期不擴展 scope。
func CheckAllowlistRenewalDoesNotExpandScope(oldScope, newScope string) error {
	var old, new struct {
		AllowedFor []string `json:"allowed_for"`
	}
	json.Unmarshal([]byte(oldScope), &old)
	json.Unmarshal([]byte(newScope), &new)

	oldSet := make(map[string]bool)
	for _, s := range old.AllowedFor {
		oldSet[s] = true
	}
	for _, s := range new.AllowedFor {
		if !oldSet[s] {
			return fmt.Errorf("§9.9 違規: 續期不得擴展 scope，新增了 %s", s)
		}
	}
	return nil
}

// CheckScopeFingerprintIncludesAllowedFor 驗證 ScopeFingerprint 包含 allowed_for。
func CheckScopeFingerprintIncludesAllowedFor(fingerprintJSON string) error {
	var fp struct {
		AllowedFor []string `json:"allowed_for"`
	}
	if err := json.Unmarshal([]byte(fingerprintJSON), &fp); err != nil {
		return nil
	}
	if len(fp.AllowedFor) == 0 {
		return fmt.Errorf("§9.8 違規: ScopeFingerprint 必須包含 allowed_for")
	}
	return nil
}

// CheckScopeFingerprintPreservesHumanReadable 驗證 ScopeFingerprint 保留人類可讀欄位。
func CheckScopeFingerprintPreservesHumanReadable(fingerprintJSON string) error {
	var fp struct {
		Hostname string `json:"hostname"`
		AddedBy  string `json:"added_by"`
	}
	if err := json.Unmarshal([]byte(fingerprintJSON), &fp); err != nil {
		return nil
	}
	if fp.Hostname == "" {
		return fmt.Errorf("§9.8 違規: ScopeFingerprint 必須保留 hostname 人類可讀欄位")
	}
	return nil
}

// CheckScopeMismatchRequiresFullReview 驗證 scope 不匹配時需完整 Review。
func CheckScopeMismatchRequiresFullReview(renewalJSON string) error {
	var renewal struct {
		ScopeMatch bool   `json:"scope_match"`
		ReviewType string `json:"review_type"`
	}
	if err := json.Unmarshal([]byte(renewalJSON), &renewal); err != nil {
		return nil
	}
	if !renewal.ScopeMatch && renewal.ReviewType == "lightweight" {
		return fmt.Errorf("§9.9 違規: scope 不匹配時不得使用 Lightweight Review")
	}
	return nil
}
