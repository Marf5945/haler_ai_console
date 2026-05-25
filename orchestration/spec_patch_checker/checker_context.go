// spec_patch_checker/checker_context.go — v3.5.0 通用守則（跨章節）。
// 共 4 條守則，涵蓋 Lightweight Review、LLM Context Governance、Global Asset 保護。
package spec_patch_checker

import (
	"encoding/json"
	"fmt"
)

// ──────────────────────────────────────────────
// 跨章節通用守則
// ──────────────────────────────────────────────

// CheckLightweightCardNotUsedForScopeExpansion 驗證 Lightweight Review Card 不用於 scope 擴展。
// §9.9 + §21：Lightweight Card 僅限同 scope 續期，不得藉此擴展權限。
func CheckLightweightCardNotUsedForScopeExpansion(cardJSON string) error {
	var card struct {
		CardType       string   `json:"card_type"`
		OldScope       []string `json:"old_scope"`
		NewScope       []string `json:"new_scope"`
	}
	if err := json.Unmarshal([]byte(cardJSON), &card); err != nil {
		return nil
	}
	if card.CardType != "lightweight" {
		return nil
	}

	// 建立舊 scope 集合
	oldSet := make(map[string]bool)
	for _, s := range card.OldScope {
		oldSet[s] = true
	}
	// 檢查新 scope 是否超出舊 scope
	for _, s := range card.NewScope {
		if !oldSet[s] {
			return fmt.Errorf("§9.9 違規: Lightweight Card 不得擴展 scope，新增了 %s", s)
		}
	}
	return nil
}

// CheckLightweightCardNotUsedOnScopeMismatch 驗證 scope 不匹配時不使用 Lightweight Card。
// §9.9：scope fingerprint 不同時必須走完整 Review。
func CheckLightweightCardNotUsedOnScopeMismatch(reviewJSON string) error {
	var review struct {
		ScopeMatch bool   `json:"scope_match"`
		CardType   string `json:"card_type"`
	}
	if err := json.Unmarshal([]byte(reviewJSON), &review); err != nil {
		return nil
	}
	if !review.ScopeMatch && review.CardType == "lightweight" {
		return fmt.Errorf("§9.9 違規: scope 不匹配時不得使用 Lightweight Card")
	}
	return nil
}

// CheckLLMContextGovernanceNotBypassed 驗證 LLM Context Governance 未被繞過。
// §11：所有傳給 LLM 的 payload 必須經過 EntryFilter + ExitValidate。
func CheckLLMContextGovernanceNotBypassed(payloadJSON string) error {
	var payload struct {
		PassedEntryFilter  bool `json:"passed_entry_filter"`
		PassedExitValidate bool `json:"passed_exit_validate"`
		SentToLLM          bool `json:"sent_to_llm"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil
	}
	if payload.SentToLLM {
		if !payload.PassedEntryFilter {
			return fmt.Errorf("§11 違規: 傳給 LLM 的 payload 未經過 EntryFilter")
		}
		if !payload.PassedExitValidate {
			return fmt.Errorf("§11 違規: 傳給 LLM 的 payload 未經過 ExitValidate")
		}
	}
	return nil
}

// CheckGlobalAssetNotPurgedByProjectPurge 驗證全域資源不被專案 purge 刪除。
// §7.5：專案 purge 不得影響全域設定、記憶、工具定義。
func CheckGlobalAssetNotPurgedByProjectPurge(purgeJSON string) error {
	var purge struct {
		Scope       string   `json:"scope"`   // "project" or "global"
		TargetPaths []string `json:"target_paths"`
	}
	if err := json.Unmarshal([]byte(purgeJSON), &purge); err != nil {
		return nil
	}
	if purge.Scope != "project" {
		return nil
	}

	// 全域保護路徑前綴
	globalPrefixes := []string{
		"global/", "shared_config/", "tool_definitions/",
		"global_memory/", "system/",
	}
	for _, path := range purge.TargetPaths {
		for _, prefix := range globalPrefixes {
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				return fmt.Errorf("§7.5 違規: 專案 purge 不得刪除全域資源 %s", path)
			}
		}
	}
	return nil
}
