// memory/validation.go — 記憶項目驗證（§18.6）。
// 驗證寫入 main_memory / deep_memory 的 YAML 項目格式正確。
// 拒絕疑似 prompt injection 自動成為 confirmed_rule。
package memory

import (
	"fmt"
	"strings"
)

// ──────────────────────────────────────────────
// 驗證狀態
// ──────────────────────────────────────────────

// ValidationResult 驗證結果。
type ValidationResult struct {
	Valid  bool   `json:"valid"`
	Status string `json:"status"` // ok, pending_review, rejected
	Reason string `json:"reason"` // 拒絕/待審原因
}

// ──────────────────────────────────────────────
// YAML 記憶項目驗證
// ──────────────────────────────────────────────

// ValidateMemoryItem 驗證記憶項目。
// 檢查格式、禁止模式、可疑內容。
func ValidateMemoryItem(content string) ValidationResult {
	// 空內容
	if strings.TrimSpace(content) == "" {
		return ValidationResult{Valid: false, Status: "rejected", Reason: "空白內容"}
	}

	// 檢查禁止模式（疑似 prompt injection）
	for _, pattern := range injectionPatterns {
		if strings.Contains(strings.ToLower(content), pattern) {
			return ValidationResult{
				Valid:  false,
				Status: "rejected",
				Reason: fmt.Sprintf("偵測到疑似 prompt injection 模式: %s", pattern),
			}
		}
	}

	// 檢查不應自動成為 confirmed_rule 的內容
	for _, pattern := range suspiciousRulePatterns {
		if strings.Contains(strings.ToLower(content), pattern) {
			return ValidationResult{
				Valid:  true,
				Status: "pending_review",
				Reason: fmt.Sprintf("內容需人工審查: %s", pattern),
			}
		}
	}

	// 檢查過長內容（單項不超過 2KB）
	if len(content) > 2048 {
		return ValidationResult{
			Valid:  true,
			Status: "pending_review",
			Reason: "內容過長，需人工審查",
		}
	}

	return ValidationResult{Valid: true, Status: "ok", Reason: ""}
}

// ──────────────────────────────────────────────
// 禁止模式清單
// ──────────────────────────────────────────────

// injectionPatterns 疑似 prompt injection 的模式。
// 偵測到時直接拒絕。
var injectionPatterns = []string{
	"ignore previous instructions",
	"ignore all previous",
	"disregard your instructions",
	"you are now",
	"act as root",
	"sudo mode",
	"override safety",
	"bypass security",
	"忽略之前的指令",
	"忽略所有規則",
	"你現在是",
	"覆蓋安全規則",
}

// suspiciousRulePatterns 可疑內容，不應自動成為 confirmed_rule。
// 偵測到時標記為 pending_review。
var suspiciousRulePatterns = []string{
	"white-hat",
	"白帽",
	"security test",
	"滲透測試",
	"cli self-description",
	"confirmed_rule", // 不允許內容自稱為 confirmed_rule
	"system prompt",
	"系統提示詞",
}
