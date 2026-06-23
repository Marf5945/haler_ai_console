// Package risk — classifier.go 提供操作風險分類邏輯。
// 根據 Spec §4.3 / §4.4 / §4.5 / §4.6 / §4.7 的規則，
// 將操作名稱與目標清單映射到七級風險等級。
package risk

import "strings"

// ──────────────────────────────────────────────
// critical_runtime_action 關鍵字清單（§4.7）
// ──────────────────────────────────────────────

// criticalKeywords 列出所有觸發 critical_runtime_action 的關鍵字。
// 若操作名稱或目標中包含任何一個關鍵字，即判定為最高等級。
var criticalKeywords = []string{
	"login",
	"payment",
	"authorization",
	"permission_grant",
	"credential_request",
	"token_request",
	"api_key_request",
	"external_share",
	"account_change",
	"security_challenge",
	"captcha",
	"irreversible_third_party_confirmation",
}

// ──────────────────────────────────────────────
// security_boundary_rewrite 關鍵字清單（§4.6）
// ──────────────────────────────────────────────

// securityBoundaryKeywords 列出所有觸發 security_boundary_rewrite 的關鍵字。
var securityBoundaryKeywords = []string{
	"rewrite_security_settings",
	"modify_risk_policy",
	"disable_destructive_confirmation",
	"weaken_destructive_confirmation",
	"modify_spec_patch_checker",
	"allow_relation_code_execution",
	"allow_privilege_elevation",
	"allow_llm_lower_risk",
	"allow_adapter_read_tokens",
	"allow_adapter_read_auth_cache",
	"allow_adapter_read_cookies",
	"allow_adapter_read_credentials",
	"allow_critical_tool_auto_tags",
	"modify_safe_export_filters",
	"modify_trust_gate_rules",
	"modify_review_gate_rules",
	"modify_risk_gate_rules",
	"allow_critical_via_trusted_session",
	"allow_destructive_via_trusted_session",
}

// ──────────────────────────────────────────────
// subagent_lifecycle_removal 關鍵字清單（§4.5）
// ──────────────────────────────────────────────

var subagentRemovalKeywords = []string{
	"delete_subagent",
	"remove_subagent",
	"remove_callable_subagent",
	"archive_subagent_lineage",
	"remove_subagent_from_preference",
	"remove_subagent_from_callable_pool",
}

// ──────────────────────────────────────────────
// user_owned_asset_destructive 關鍵字清單（§4.4）
// ──────────────────────────────────────────────

var destructiveKeywords = []string{
	"delete_project",
	"clear_memory",
	"delete_artifact",
	"overwrite_unrecoverable",
	"delete_cache",
	"delete_records",
	"purge_project",
	"permanent_delete",
}

// ──────────────────────────────────────────────
// high_non_destructive 九項必要條件（§4.3）
// ──────────────────────────────────────────────

// HighNonDestructiveConditions 描述一個操作在判定為 high_non_destructive 時
// 必須全部滿足的九項條件。呼叫端負責填入實際狀態。
type HighNonDestructiveConditions struct {
	NoDelete                      bool // 不涉及刪除
	NoOverwriteWithoutLocalUndo   bool // 不涉及無法本地復原的覆寫
	NoExternalShare               bool // 不涉及外部分享
	NoPermissionChange            bool // 不涉及權限變更
	NoAuthChange                  bool // 不涉及認證變更
	NoPayment                     bool // 不涉及付款
	NoAccountChange               bool // 不涉及帳號變更
	ReversibleWithLocalUndo       bool // 可以透過本地操作復原
	TargetSetFullyKnownBeforeExec bool // 目標集合在執行前完全已知
}

// AllSatisfied 判斷九項條件是否全部滿足。
func (c *HighNonDestructiveConditions) AllSatisfied() bool {
	return c.NoDelete &&
		c.NoOverwriteWithoutLocalUndo &&
		c.NoExternalShare &&
		c.NoPermissionChange &&
		c.NoAuthChange &&
		c.NoPayment &&
		c.NoAccountChange &&
		c.ReversibleWithLocalUndo &&
		c.TargetSetFullyKnownBeforeExec
}

// ──────────────────────────────────────────────
// 核心分類函式
// ──────────────────────────────────────────────

// ClassifyOperation 根據操作名稱與目標清單，回傳對應的風險等級。
// 採用最高優先原則：critical > security_boundary > subagent_removal > destructive > high > medium > low。
// 呼叫端若需精確判斷 high_non_destructive，應另外呼叫 ClassifyWithConditions。
func ClassifyOperation(operation string, targets []string) RiskClass {
	op := strings.ToLower(operation)

	// 合併操作名稱與目標，用於關鍵字掃描
	allTexts := append([]string{op}, lowercaseAll(targets)...)

	// §4.7: critical_runtime_action — 最高優先
	if matchesAny(allTexts, criticalKeywords) {
		return CriticalRuntimeAction
	}

	// §4.6: security_boundary_rewrite
	if matchesAny(allTexts, securityBoundaryKeywords) {
		return SecurityBoundaryRewrite
	}

	// §4.5: subagent_lifecycle_removal
	if matchesAny(allTexts, subagentRemovalKeywords) {
		return SubagentLifecycleRemoval
	}

	// §4.4: user_owned_asset_destructive
	if matchesAny(allTexts, destructiveKeywords) {
		return UserOwnedAssetDestructive
	}

	// §4.3: 若無法用關鍵字判定，預設回傳 Medium，
	// 讓呼叫端決定是否升級為 high_non_destructive 或降級為 low。
	return Medium
}

// ClassifyWithConditions 與 ClassifyOperation 類似，但額外接受
// high_non_destructive 九項條件。若操作被初步判定為 Medium 且
// 條件不完全滿足，仍回傳 Medium；若全部滿足且初判為 Medium，
// 則提升為 HighNonDestructive。
//
// 注意：即使條件全部滿足，若操作包含更高等級的關鍵字，
// 仍會回傳該更高等級（不可降級原則 §4.2）。
func ClassifyWithConditions(operation string, targets []string, cond *HighNonDestructiveConditions) RiskClass {
	base := ClassifyOperation(operation, targets)

	// 只有在 base 為 Medium 時才考慮提升為 HighNonDestructive
	if base == Medium && cond != nil && cond.AllSatisfied() {
		return HighNonDestructive
	}

	return base
}

// ClassifyToAtLeast 確保分類結果至少為指定等級（§4.2 不可降級原則）。
// 常用於 Hook 或 adapter 需要「只能提高」的場景。
func ClassifyToAtLeast(operation string, targets []string, floor RiskClass) RiskClass {
	result := ClassifyOperation(operation, targets)
	return Max(result, floor)
}

// ──────────────────────────────────────────────
// 內部輔助函式
// ──────────────────────────────────────────────

// matchesAny 檢查 texts 中是否有任何一個包含 keywords 中的任何一個關鍵字。
func matchesAny(texts []string, keywords []string) bool {
	for _, t := range texts {
		for _, kw := range keywords {
			if strings.Contains(t, kw) {
				return true
			}
		}
	}
	return false
}

// lowercaseAll 將字串切片中的所有元素轉為小寫。
func lowercaseAll(ss []string) []string {
	result := make([]string, len(ss))
	for i, s := range ss {
		result[i] = strings.ToLower(s)
	}
	return result
}
