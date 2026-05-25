package risk

import "testing"

// 測試 final_risk 永遠不可降級（§4.2）
func TestCanDowngradeAlwaysFalse(t *testing.T) {
	pairs := [][2]RiskClass{
		{CriticalRuntimeAction, Low},
		{SecurityBoundaryRewrite, Medium},
		{HighNonDestructive, Low},
		{Medium, Low},
	}
	for _, p := range pairs {
		if CanDowngrade(p[0], p[1]) {
			t.Errorf("CanDowngrade(%s, %s) should be false", p[0], p[1])
		}
	}
}

// 測試風險排序正確
func TestIsHigherOrEqual(t *testing.T) {
	if !IsHigherOrEqual(CriticalRuntimeAction, Low) {
		t.Error("critical should be >= low")
	}
	if IsHigherOrEqual(Low, CriticalRuntimeAction) {
		t.Error("low should not be >= critical")
	}
	if !IsHigherOrEqual(Medium, Medium) {
		t.Error("medium should be >= medium")
	}
}

// 測試雙步驟確認僅限 security_boundary_rewrite（§4.6 v3.6.1）
func TestRequiresDualStep(t *testing.T) {
	if !RequiresDualStep(SecurityBoundaryRewrite) {
		t.Error("security_boundary_rewrite must require dual-step")
	}
	others := []RiskClass{Low, Medium, HighNonDestructive, UserOwnedAssetDestructive, SubagentLifecycleRemoval, CriticalRuntimeAction}
	for _, c := range others {
		if RequiresDualStep(c) {
			t.Errorf("%s should not require dual-step", c)
		}
	}
}

// 測試確認方式矩陣（§5.3）
func TestConfirmationMatrix(t *testing.T) {
	cases := map[RiskClass]ConfirmationType{
		Low:                       ConfirmSilent,
		Medium:                    ConfirmNormal,
		HighNonDestructive:        ConfirmReviewButton,
		UserOwnedAssetDestructive: ConfirmConsequenceMenu,
		SubagentLifecycleRemoval:  ConfirmExportFirst,
		SecurityBoundaryRewrite:   ConfirmDualStep,
		CriticalRuntimeAction:     ConfirmStopRecovery,
	}
	for rc, want := range cases {
		got := ConfirmationFor(rc)
		if got != want {
			t.Errorf("ConfirmationFor(%s) = %s, want %s", rc, got, want)
		}
	}
}

// 測試信任覆蓋範圍（§15.1–§15.2）
func TestTrustCoverage(t *testing.T) {
	// Contextual Override 最高到 high_non_destructive
	if !CanBeCoveredByContextualOverride(HighNonDestructive) {
		t.Error("high_non_destructive should be coverable by override")
	}
	if CanBeCoveredByContextualOverride(UserOwnedAssetDestructive) {
		t.Error("user_owned_asset_destructive should NOT be coverable by override")
	}

	// Trusted Session 最高到 medium
	if !CanBeCoveredByTrustedSession(Medium) {
		t.Error("medium should be coverable by trusted session")
	}
	if CanBeCoveredByTrustedSession(HighNonDestructive) {
		t.Error("high_non_destructive should NOT be coverable by trusted session")
	}
}

// 測試批次核准限制
func TestCanBatchApprove(t *testing.T) {
	if CanBatchApprove(SecurityBoundaryRewrite) {
		t.Error("security_boundary_rewrite must not allow batch approve")
	}
	if CanBatchApprove(CriticalRuntimeAction) {
		t.Error("critical_runtime_action must not allow batch approve")
	}
	if !CanBatchApprove(Medium) {
		t.Error("medium should allow batch approve")
	}
}

// 測試使用者標籤不含工程 token（§6.2）
func TestUserLabelNoEngineeringToken(t *testing.T) {
	forbidden := []string{"pink_cut_black", "forgiveness_green", "defeat_blue", "passive_white"}
	for _, c := range []RiskClass{Low, Medium, HighNonDestructive, UserOwnedAssetDestructive, SubagentLifecycleRemoval, SecurityBoundaryRewrite, CriticalRuntimeAction} {
		label := UserLabel(c)
		for _, f := range forbidden {
			if label == f {
				t.Errorf("UserLabel(%s) contains forbidden token %q", c, f)
			}
		}
		if label == "" {
			t.Errorf("UserLabel(%s) is empty", c)
		}
	}
}
