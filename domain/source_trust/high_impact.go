// source_trust/high_impact.go — 高影響領域設定（§9.6）。
// 9 個 built-in 領域不可移除，使用者可新增全域/專案層級。
// 專案設定不可削弱全域設定。
package source_trust

import "fmt"

// ──────────────────────────────────────────────
// Built-in 高影響領域（§9.6 不可移除）
// ──────────────────────────────────────────────

// BuiltInHighImpactDomains 是系統內建的 9 個高影響領域。
// 使用者不可移除，僅可新增。
var BuiltInHighImpactDomains = []string{
	"legal",
	"medical",
	"financial",
	"security_setting",
	"software_install",
	"risk_policy",
	"permission_change",
	"automation_execution",
	"long_term_memory_write",
}

// builtInSet 用於快速查找
var builtInSet map[string]bool

func init() {
	builtInSet = make(map[string]bool, len(BuiltInHighImpactDomains))
	for _, d := range BuiltInHighImpactDomains {
		builtInSet[d] = true
	}
}

// ──────────────────────────────────────────────
// 有效高影響領域計算
// ──────────────────────────────────────────────

// EffectiveHighImpactDomains 計算最終的高影響領域清單。
// 公式：built_in + global_user + project_user（§9.6）。
func EffectiveHighImpactDomains(globalUser, projectUser []string) []string {
	seen := make(map[string]bool)
	var result []string

	// built-in 永遠包含
	for _, d := range BuiltInHighImpactDomains {
		if !seen[d] {
			result = append(result, d)
			seen[d] = true
		}
	}
	// 全域使用者自定
	for _, d := range globalUser {
		if !seen[d] {
			result = append(result, d)
			seen[d] = true
		}
	}
	// 專案使用者自定
	for _, d := range projectUser {
		if !seen[d] {
			result = append(result, d)
			seen[d] = true
		}
	}
	return result
}

// IsHighImpact 判斷指定領域是否為高影響。
func IsHighImpact(domain string, globalUser, projectUser []string) bool {
	if builtInSet[domain] {
		return true
	}
	for _, d := range globalUser {
		if d == domain {
			return true
		}
	}
	for _, d := range projectUser {
		if d == domain {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// 驗證規則
// ──────────────────────────────────────────────

// ValidateUserRemoval 驗證使用者是否可移除某個高影響領域。
// Built-in 領域不可移除。
func ValidateUserRemoval(domain string) error {
	if builtInSet[domain] {
		return fmt.Errorf("不可移除內建高影響領域: %s", domain)
	}
	return nil
}

// ValidateProjectWeaken 驗證專案設定是否削弱全域設定。
// 專案不可移除全域或 built-in 已有的領域。
func ValidateProjectWeaken(globalDomains, projectDomains []string) error {
	globalSet := make(map[string]bool)
	for _, d := range BuiltInHighImpactDomains {
		globalSet[d] = true
	}
	for _, d := range globalDomains {
		globalSet[d] = true
	}

	// 專案不可聲明排除全域已有的領域
	// 此函式在 projectDomains 作為「完整取代」清單時檢查
	// 若 projectDomains 為空，代表使用全域設定（合法）
	if len(projectDomains) == 0 {
		return nil
	}

	projectSet := make(map[string]bool)
	for _, d := range projectDomains {
		projectSet[d] = true
	}

	for d := range globalSet {
		if !projectSet[d] {
			return fmt.Errorf("專案設定不可削弱全域設定: 缺少 %s", d)
		}
	}
	return nil
}
