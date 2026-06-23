// source_trust/review_behavior.go — PENDING_SOURCE_REVIEW 行為決策（§9.5）。
// 根據信任標籤、高影響旗標、目標用途，決定是否 blocking。
package source_trust

// ──────────────────────────────────────────────
// Blocking 場景清單（§9.5）
// ──────────────────────────────────────────────

// blockingUsages 列出所有需要 blocking review 的目標用途。
var blockingUsages = map[string]bool{
	"legal":                  true, // 法律
	"medical":                true, // 醫療
	"financial":              true, // 財務
	"security_setting":       true, // 安全設定
	"software_install":       true, // 程式安裝
	"risk_policy":            true, // 風險政策
	"permission_change":      true, // 權限設定
	"security_rule_modify":   true, // 安全規則修改
	"automation_execution":   true, // 自動化操作執行依據
	"long_term_memory_write": true, // 寫入長期知識庫
	"allowlist_add":          true, // 加入 project source allowlist
}

// ShouldBlock 判斷 PENDING_SOURCE_REVIEW 來源是否應阻擋操作。
// 非 PENDING 標籤永遠不 block（由其他機制處理）。
//
// Blocking 條件（任一即 block）：
//   - 目標用途在 blockingUsages 中
//   - 來源處於高影響領域
//   - 標籤為 LOW_TRUST
func ShouldBlock(label SourceTrustLabel, isHighImpact bool, targetUsage string) bool {
	// LOW_TRUST 永遠 blocking
	if label == LabelLowTrust {
		return true
	}

	// 僅 PENDING_SOURCE_REVIEW 需要判斷 blocking
	if label != LabelPendingSourceReview {
		return false
	}

	// 高影響領域 → blocking
	if isHighImpact {
		return true
	}

	// 特定用途 → blocking
	if blockingUsages[targetUsage] {
		return true
	}

	// 其他場景 → non-blocking（顯示標籤 + tooltip）
	return false
}

// EnrichReviewRequired 根據 blocking 判定更新 evidence。
func EnrichReviewRequired(e *SourceTrustEvidence, targetUsage string, globalUser, projectUser []string) {
	isHigh := IsHighImpact(targetUsage, globalUser, projectUser)
	e.IsHighImpact = isHigh

	if ShouldBlock(e.SourceTrustLabel, isHigh, targetUsage) {
		e.ReviewRequired = true
		if e.SourceTrustLabel == LabelLowTrust {
			e.ReviewReason = "low_trust_source_blocking"
		} else if isHigh {
			e.ReviewReason = "pending_review_high_impact_domain"
		} else {
			e.ReviewReason = "pending_review_sensitive_usage"
		}
	}
}
