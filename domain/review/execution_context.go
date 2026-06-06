// review/execution_context.go — Review Execution Context API（§4.6 Step 1/2 hash 計算層）。
// 提供 ComputeExecutionContext：後端即時計算所有相關 hash，
// 前端僅做顯示/除錯用，真正 snapshot 由 DualStepConfirmStep1 自行保存。
package review

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
)

// ──────────────────────────────────────────────
// ReviewExecutionContext：前端顯示 + 後端 snapshot 共用結構
// ──────────────────────────────────────────────

// ReviewExecutionContext 包含 Review Card 執行時的完整 hash 快照。
type ReviewExecutionContext struct {
	ReviewID         string `json:"review_id"`
	ScopeHash        string `json:"scope_hash"`
	RiskPolicyHash   string `json:"risk_policy_hash"`
	ToolRegistryHash string `json:"tool_registry_hash"`
	TargetHashSet    string `json:"target_hash_set"`
}

// ──────────────────────────────────────────────
// 計算方法
// ──────────────────────────────────────────────

// ComputeExecutionContext 即時計算指定 Review Card 的 execution context。
// projectRoot 為專案資料根目錄。
// 注意：computeFileHashForReview 定義於 service.go（同 package 共用），
// 避免 domain/review → orchestration/dag 的 import cycle。
func (s *Service) ComputeExecutionContext(cardID, projectRoot string) (*ReviewExecutionContext, error) {
	card, err := s.GetCard(cardID)
	if err != nil {
		return nil, err
	}

	// 從檔案計算 hash（邏輯與 dag/resume_guard.go 一致）
	riskPolicyHash := computeFileHashForReview(filepath.Join(projectRoot, "risk_policy", "policy.json"))
	toolRegistryHash := computeFileHashForReview(filepath.Join(projectRoot, "tool_registry", "registry.json"))

	// 從 card 目標計算 target hash
	targetHash := ComputeTargetHashSet([]TargetEntry{{
		Operation:     card.Operation,
		Target:        card.Target,
		AffectedScope: card.AcceptEffect,
	}})

	// 組合 scope hash = SHA-256(riskPolicy + toolRegistry + target)
	scopeHash := computeScopeHashFromParts(riskPolicyHash, toolRegistryHash, targetHash)

	return &ReviewExecutionContext{
		ReviewID:         cardID,
		ScopeHash:        scopeHash,
		RiskPolicyHash:   riskPolicyHash,
		ToolRegistryHash: toolRegistryHash,
		TargetHashSet:    targetHash,
	}, nil
}

// computeScopeHashFromParts 從三個子 hash 組合 scope hash。
func computeScopeHashFromParts(riskPolicy, toolRegistry, targetHash string) string {
	raw := riskPolicy + "|" + toolRegistry + "|" + targetHash
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}
