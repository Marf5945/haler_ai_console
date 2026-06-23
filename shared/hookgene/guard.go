package hookgene

import (
	"errors"
	"fmt"
)

// CoreAction 是 core layer 越權變更類型（§3.1.5.11 / §3.1.5.18.6）。
type CoreAction string

const (
	ActionActiveSkill          CoreAction = "active_skill"
	ActionCoreMemoryWrite      CoreAction = "core_memory_write"
	ActionPersonaCoreUpdate    CoreAction = "persona_core_update"
	ActionRiskPolicyChange     CoreAction = "risk_policy_change"
	ActionSecurityBoundary     CoreAction = "security_boundary_rewrite"
	ActionActiveSubagentEnable CoreAction = "active_subagent_enable"
)

// ErrCoreLayerDenied 是 guard 阻擋時回傳的 sentinel error（可用 errors.Is 比對）。
var ErrCoreLayerDenied = errors.New("hookgene: core layer mutation denied")

// deniedCoreActions：學習/突變流程一律不得碰的 core 動作。
var deniedCoreActions = map[CoreAction]bool{
	ActionActiveSkill:          true,
	ActionCoreMemoryWrite:      true,
	ActionPersonaCoreUpdate:    true,
	ActionRiskPolicyChange:     true,
	ActionSecurityBoundary:     true,
	ActionActiveSubagentEnable: true,
}

// GuardPhase 標示 guard 被呼叫的時機（§3.1.5.18.6：生成 / review / 執行前共三次）。
type GuardPhase string

const (
	PhaseGenerate GuardPhase = "candidate_generate"
	PhaseReview   GuardPhase = "candidate_review"
	PhaseExecute  GuardPhase = "candidate_execute"
)

// DenyCoreLayerMutation 阻擋學習/突變流程對 core layer 的越權變更。
// 維護：必須在 candidate「生成」「review」「評估或執行前」三個時機各呼叫一次，
// 不要只在啟用時擋（否則 STAGED 裡可能已躺著危險 candidate）。
func DenyCoreLayerMutation(phase GuardPhase, action CoreAction) error {
	if deniedCoreActions[action] {
		return fmt.Errorf("%w: phase=%s action=%s", ErrCoreLayerDenied, phase, action)
	}
	return nil
}
