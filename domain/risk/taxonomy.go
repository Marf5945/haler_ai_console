// Package risk 定義 AI Console v3.6.1 的七級風險分類與確認策略。
// 所有風險判定的源頭在此，其他模組（review、controlled_trust、dag）引用本套件常數。
package risk

// ──────────────────────────────────────────────
// 風險等級定義（§4.1）
// ──────────────────────────────────────────────

// RiskClass 代表操作的風險等級，由低到高排列。
type RiskClass string

const (
	Low                       RiskClass = "low"
	Medium                    RiskClass = "medium"
	HighNonDestructive        RiskClass = "high_non_destructive"
	UserOwnedAssetDestructive RiskClass = "user_owned_asset_destructive"
	SubagentLifecycleRemoval  RiskClass = "subagent_lifecycle_removal"
	SecurityBoundaryRewrite   RiskClass = "security_boundary_rewrite"
	CriticalRuntimeAction     RiskClass = "critical_runtime_action"
)

// allClasses 用於排序比較，索引越大風險越高。
var classOrder = map[RiskClass]int{
	Low:                       0,
	Medium:                    1,
	HighNonDestructive:        2,
	UserOwnedAssetDestructive: 3,
	SubagentLifecycleRemoval:  4,
	SecurityBoundaryRewrite:   5,
	CriticalRuntimeAction:     6,
}

// ──────────────────────────────────────────────
// 風險不可降級規則（§4.2）
// ──────────────────────────────────────────────

// CanDowngrade 永遠回傳 false。
// Spec §4.2: final_risk 只能被提高，不能被 LLM、Hook、adapter 或任何外部來源降低。
func CanDowngrade(from, to RiskClass) bool {
	return false
}

// IsHigherOrEqual 判斷 a 的風險是否 >= b。
func IsHigherOrEqual(a, b RiskClass) bool {
	return classOrder[a] >= classOrder[b]
}

// Max 回傳兩者中較高的風險等級。
func Max(a, b RiskClass) RiskClass {
	if classOrder[a] >= classOrder[b] {
		return a
	}
	return b
}

// ──────────────────────────────────────────────
// 確認策略（§4.6 v3.6.1 雙步驟按鈕）
// ──────────────────────────────────────────────

// ConfirmationType 描述每個風險等級需要的確認方式。
type ConfirmationType string

const (
	ConfirmSilent          ConfirmationType = "silent"           // low：靜默或一般 UI
	ConfirmNormal          ConfirmationType = "normal"           // medium：一般 UI 或群組 review
	ConfirmReviewButton    ConfirmationType = "review_button"    // high_non_destructive：Review Card + 按鈕/長按
	ConfirmConsequenceMenu ConfirmationType = "consequence_menu" // user_owned_asset_destructive：明確後果選單
	ConfirmExportFirst     ConfirmationType = "export_first"     // subagent_lifecycle_removal：匯出優先
	ConfirmDualStep        ConfirmationType = "dual_step"        // security_boundary_rewrite：雙步驟按鈕
	ConfirmStopRecovery    ConfirmationType = "stop_recovery"    // critical_runtime_action：完全停下 + 彈窗
)

// ConfirmationFor 回傳指定風險等級所需的確認方式。
func ConfirmationFor(c RiskClass) ConfirmationType {
	switch c {
	case Low:
		return ConfirmSilent
	case Medium:
		return ConfirmNormal
	case HighNonDestructive:
		return ConfirmReviewButton
	case UserOwnedAssetDestructive:
		return ConfirmConsequenceMenu
	case SubagentLifecycleRemoval:
		return ConfirmExportFirst
	case SecurityBoundaryRewrite:
		return ConfirmDualStep
	case CriticalRuntimeAction:
		return ConfirmStopRecovery
	default:
		return ConfirmStopRecovery // 未知風險採最嚴格
	}
}

// RequiresDualStep 判斷是否需要雙步驟確認（僅 security_boundary_rewrite）。
func RequiresDualStep(c RiskClass) bool {
	return c == SecurityBoundaryRewrite
}

// RequiresExportFirst 判斷是否需要匯出優先流程（僅 subagent_lifecycle_removal）。
func RequiresExportFirst(c RiskClass) bool {
	return c == SubagentLifecycleRemoval
}

// RequiresFullStop 判斷是否需要完全停下（僅 critical_runtime_action）。
func RequiresFullStop(c RiskClass) bool {
	return c == CriticalRuntimeAction
}

// ──────────────────────────────────────────────
// 信任覆蓋範圍限制（§15.1–§15.2）
// ──────────────────────────────────────────────

// CanBeCoveredByContextualOverride 判斷此風險等級是否可被 Contextual Risk Override 覆蓋。
// 最高只到 high_non_destructive。
func CanBeCoveredByContextualOverride(c RiskClass) bool {
	return classOrder[c] <= classOrder[HighNonDestructive]
}

// CanBeCoveredByTrustedSession 判斷此風險等級是否可被 Trusted Session 覆蓋。
// 最高只到 medium。
func CanBeCoveredByTrustedSession(c RiskClass) bool {
	return classOrder[c] <= classOrder[Medium]
}

// CanBatchApprove 判斷此風險等級是否可以批次核准。
// security_boundary_rewrite 和 critical_runtime_action 永遠不可批次。
func CanBatchApprove(c RiskClass) bool {
	switch c {
	case SecurityBoundaryRewrite, CriticalRuntimeAction:
		return false
	default:
		return true
	}
}

// ──────────────────────────────────────────────
// 使用者友善標籤（§6.2）
// ──────────────────────────────────────────────

// UserLabel 回傳風險等級的使用者友善中文標籤。
// 這些標籤會顯示在 UI 上，禁止出現工程 token 名稱。
func UserLabel(c RiskClass) string {
	switch c {
	case Low:
		return "一般操作"
	case Medium:
		return "需要你確認"
	case HighNonDestructive:
		return "高風險 / 需要你確認"
	case UserOwnedAssetDestructive:
		return "破壞性操作 / 建議先備份"
	case SubagentLifecycleRemoval:
		return "移除子代理 / 建議先匯出"
	case SecurityBoundaryRewrite:
		return "安全設定變更 / 需要確認"
	case CriticalRuntimeAction:
		return "已中斷 / 需要你處理"
	default:
		return "未知操作"
	}
}
