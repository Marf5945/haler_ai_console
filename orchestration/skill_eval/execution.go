// execution.go — Skill 執行決策與 candidate 流程的「純邏輯」（TASK 31 / Phase 2 後端）。
// 此檔不碰 app.go / CLI 執行路徑，只提供可單元測試的決策；實際 inject/execute 與
// 前端確認彈窗由 app 層接線（見 TASKS_1_9.md TASK 31 Phase 2 整合說明）。
package skill_eval

import (
	"ui_console/orchestration/skill_step"
	"ui_console/shared/riskgrant"
)

// ExecDecision 是依 ResolveStatus + lifecycle + 授權狀態算出的執行決策。
type ExecDecision string

const (
	ExecAuto        ExecDecision = "auto_execute"  // 直接 build injection → execute
	ExecNeedConfirm ExecDecision = "need_confirm"  // 第一次執行確認（允許一次/總是允許/取消）
	ExecCandidate   ExecDecision = "candidate"     // 多候選，顯示候選卡讓使用者選
	ExecReview      ExecDecision = "review"        // 高風險/分數不足，顯示 review card
	ExecNoSkill     ExecDecision = "no_skill"      // 無對應 skill：走一般流程，事後問是否存成 skill
)

// DecideExecution 把路由結果與 lifecycle 收斂成單一執行決策。
//   - hasGrant：riskgrant 是否已有「允許一次」有效授權（見 HasSkillGrant）。
func DecideExecution(status skill_step.ResolveStatus, lc *skill_step.Lifecycle, hasGrant bool) ExecDecision {
	switch status {
	case skill_step.StatusRejected:
		return ExecNoSkill
	case skill_step.StatusNeedsReview:
		return ExecReview
	case skill_step.StatusNeedsCLI:
		return ExecCandidate
	case skill_step.StatusAutoSelected:
		if lc == nil {
			return ExecNeedConfirm // 保守：無 lifecycle 資訊先要求確認
		}
		if lc.Status == skill_step.LifecycleDisabled {
			return ExecNoSkill
		}
		// enabled + auto_execute → 免確認；pending 或 auto=false → 視授權決定。
		if lc.AutoExecute && lc.Status == skill_step.LifecycleEnabled {
			return ExecAuto
		}
		if hasGrant {
			return ExecAuto // 「允許一次」仍在有效期內
		}
		return ExecNeedConfirm
	}
	return ExecNeedConfirm
}

// skillGrantTarget 用 session 粒度（不綁精確 target），讓「允許一次」涵蓋整段對話對該 skill 的呼叫。
const skillGrantTarget = "*"

func firstTag(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return tags[0]
}

// GrantOnce 對應「允許一次」：寫入帶 TTL 的 riskgrant，不改 manifest。
func GrantOnce(store *riskgrant.Store, m *skill_step.SkillManifest) {
	if store == nil || m == nil {
		return
	}
	store.Grant(m.SkillID, firstTag(m.Tags.ActionTag), skillGrantTarget, firstTag(m.Tags.RiskTag))
}

// HasSkillGrant 回報該 skill 是否已有有效的「允許一次」授權。
func HasSkillGrant(store *riskgrant.Store, m *skill_step.SkillManifest) bool {
	if store == nil || m == nil {
		return false
	}
	return store.HasValid(m.SkillID, firstTag(m.Tags.ActionTag), skillGrantTarget, firstTag(m.Tags.RiskTag))
}

// PromoteToEnabled 對應「總是允許」：升級 lifecycle 為 enabled_skill。
// auto_execute 仍依風險/權限決定（沿用 DefaultLifecycle 的判準），不無條件打開。
func PromoteToEnabled(m *skill_step.SkillManifest) {
	if m == nil {
		return
	}
	def := skill_step.DefaultLifecycle(m) // 取得依風險算出的 auto_execute
	if m.Lifecycle == nil {
		m.Lifecycle = &def
	}
	m.Lifecycle.Status = skill_step.LifecycleEnabled
	m.Lifecycle.UserConfirmed = true
	m.Lifecycle.AutoExecute = def.AutoExecute
}

// BuildPendingDraft 由 LLM 提供的欄位建立 skill 草稿（TASK 31 / Phase 2）。
// 先跑 ValidateDraft：通過 → pending_skill（可見/可候選/不自動執行）；
// 未過 → draft_candidate（不顯示為工具），並回傳問題清單供修正。
func BuildPendingDraft(skillID, displayName string, actionTags, domainTags []string, chain *skill_step.ExpectedChain) (*skill_step.SkillManifest, []string) {
	problems := ValidateDraft(chain)
	status := skill_step.LifecyclePending
	if len(problems) > 0 {
		status = skill_step.LifecycleDraftCandidate
	}
	visible := status == skill_step.LifecyclePending
	return &skill_step.SkillManifest{
		SchemaVersion: skill_step.SchemaManifestV2,
		SkillID:       skillID,
		DisplayName:   displayName,
		Version:       "0.1.0",
		Tags: skill_step.SkillTags{
			ActionTag: actionTags, DomainTag: domainTags, RiskTag: []string{"medium"},
		},
		// 草稿預設零權限，第一次執行時才由使用者授權（最小權限原則）。
		Permissions:   skill_step.SkillPermissions{Network: "none", Filesystem: "none", Execution: "none"},
		ExpectedChain: chain,
		Lifecycle: &skill_step.Lifecycle{
			Status: status, VisibleInToolbar: visible,
			RouteAsCandidate: visible, AutoExecute: false, UserConfirmed: false,
		},
	}, problems
}
