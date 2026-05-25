package risk

import "testing"

// 測試 critical_runtime_action 關鍵字偵測（§4.7）
func TestClassifyCriticalKeywords(t *testing.T) {
	criticalOps := []string{
		"login", "payment", "authorization",
		"permission_grant", "credential_request",
		"token_request", "api_key_request",
		"external_share", "account_change",
		"security_challenge", "captcha",
		"irreversible_third_party_confirmation",
	}
	for _, op := range criticalOps {
		got := ClassifyOperation(op, nil)
		if got != CriticalRuntimeAction {
			t.Errorf("ClassifyOperation(%q) = %s, want critical_runtime_action", op, got)
		}
	}
}

// 測試 security_boundary_rewrite 關鍵字偵測（§4.6）
func TestClassifySecurityBoundary(t *testing.T) {
	secOps := []string{
		"rewrite_security_settings",
		"modify_risk_policy",
		"allow_llm_lower_risk",
		"modify_safe_export_filters",
	}
	for _, op := range secOps {
		got := ClassifyOperation(op, nil)
		if got != SecurityBoundaryRewrite {
			t.Errorf("ClassifyOperation(%q) = %s, want security_boundary_rewrite", op, got)
		}
	}
}

// 測試 subagent_lifecycle_removal 偵測（§4.5）
func TestClassifySubagentRemoval(t *testing.T) {
	ops := []string{
		"delete_subagent",
		"remove_callable_subagent",
		"archive_subagent_lineage",
	}
	for _, op := range ops {
		got := ClassifyOperation(op, nil)
		if got != SubagentLifecycleRemoval {
			t.Errorf("ClassifyOperation(%q) = %s, want subagent_lifecycle_removal", op, got)
		}
	}
}

// 測試 user_owned_asset_destructive 偵測（§4.4）
func TestClassifyDestructive(t *testing.T) {
	ops := []string{
		"delete_project",
		"clear_memory",
		"overwrite_unrecoverable",
		"permanent_delete",
	}
	for _, op := range ops {
		got := ClassifyOperation(op, nil)
		if got != UserOwnedAssetDestructive {
			t.Errorf("ClassifyOperation(%q) = %s, want user_owned_asset_destructive", op, got)
		}
	}
}

// 測試未知操作預設回傳 Medium
func TestClassifyDefaultMedium(t *testing.T) {
	got := ClassifyOperation("read_file", []string{"config.json"})
	if got != Medium {
		t.Errorf("ClassifyOperation(read_file) = %s, want medium", got)
	}
}

// 測試目標字串也會觸發分類
func TestClassifyTargetTrigger(t *testing.T) {
	// 操作本身無風險，但目標包含 critical 關鍵字
	got := ClassifyOperation("process", []string{"payment_gateway"})
	if got != CriticalRuntimeAction {
		t.Errorf("target 'payment_gateway' should trigger critical, got %s", got)
	}
}

// 測試大小寫不敏感
func TestClassifyCaseInsensitive(t *testing.T) {
	got := ClassifyOperation("LOGIN", nil)
	if got != CriticalRuntimeAction {
		t.Errorf("ClassifyOperation(LOGIN) = %s, want critical_runtime_action", got)
	}

	got = ClassifyOperation("Delete_Project", nil)
	if got != UserOwnedAssetDestructive {
		t.Errorf("ClassifyOperation(Delete_Project) = %s, want user_owned_asset_destructive", got)
	}
}

// 測試優先順序：critical > security_boundary
func TestClassifyPriorityOrder(t *testing.T) {
	// 同時包含 critical 和 security 關鍵字時，critical 優先
	got := ClassifyOperation("login_rewrite_security_settings", nil)
	if got != CriticalRuntimeAction {
		t.Errorf("should be critical when both critical and security keywords match, got %s", got)
	}
}

// 測試 high_non_destructive 九項條件（§4.3）
func TestClassifyWithConditionsAllSatisfied(t *testing.T) {
	cond := &HighNonDestructiveConditions{
		NoDelete:                      true,
		NoOverwriteWithoutLocalUndo:   true,
		NoExternalShare:               true,
		NoPermissionChange:            true,
		NoAuthChange:                  true,
		NoPayment:                     true,
		NoAccountChange:               true,
		ReversibleWithLocalUndo:       true,
		TargetSetFullyKnownBeforeExec: true,
	}
	got := ClassifyWithConditions("rename_file", []string{"a.txt"}, cond)
	if got != HighNonDestructive {
		t.Errorf("all conditions satisfied should be high_non_destructive, got %s", got)
	}
}

// 測試九項條件未全部滿足時維持 Medium
func TestClassifyWithConditionsPartial(t *testing.T) {
	cond := &HighNonDestructiveConditions{
		NoDelete:                      true,
		NoOverwriteWithoutLocalUndo:   true,
		NoExternalShare:               false, // 不滿足
		NoPermissionChange:            true,
		NoAuthChange:                  true,
		NoPayment:                     true,
		NoAccountChange:               true,
		ReversibleWithLocalUndo:       true,
		TargetSetFullyKnownBeforeExec: true,
	}
	got := ClassifyWithConditions("rename_file", []string{"a.txt"}, cond)
	if got != Medium {
		t.Errorf("partial conditions should stay medium, got %s", got)
	}
}

// 測試條件滿足但關鍵字更高時不降級
func TestClassifyWithConditionsNeverDowngrade(t *testing.T) {
	cond := &HighNonDestructiveConditions{
		NoDelete:                      true,
		NoOverwriteWithoutLocalUndo:   true,
		NoExternalShare:               true,
		NoPermissionChange:            true,
		NoAuthChange:                  true,
		NoPayment:                     true,
		NoAccountChange:               true,
		ReversibleWithLocalUndo:       true,
		TargetSetFullyKnownBeforeExec: true,
	}
	// delete_project 是 destructive，即使條件全滿也不能降級
	got := ClassifyWithConditions("delete_project", nil, cond)
	if got != UserOwnedAssetDestructive {
		t.Errorf("keyword match should override conditions, got %s", got)
	}
}

// 測試 ClassifyToAtLeast 保底機制
func TestClassifyToAtLeast(t *testing.T) {
	// read_file 本身是 Medium，但保底設為 HighNonDestructive
	got := ClassifyToAtLeast("read_file", nil, HighNonDestructive)
	if got != HighNonDestructive {
		t.Errorf("floor should raise result to high_non_destructive, got %s", got)
	}

	// delete_project 本身是 destructive，保底設為 Medium 時不降級
	got = ClassifyToAtLeast("delete_project", nil, Medium)
	if got != UserOwnedAssetDestructive {
		t.Errorf("result above floor should stay, got %s", got)
	}
}

// 測試 HighNonDestructiveConditions.AllSatisfied
func TestAllSatisfied(t *testing.T) {
	full := HighNonDestructiveConditions{
		NoDelete: true, NoOverwriteWithoutLocalUndo: true,
		NoExternalShare: true, NoPermissionChange: true,
		NoAuthChange: true, NoPayment: true, NoAccountChange: true,
		ReversibleWithLocalUndo: true, TargetSetFullyKnownBeforeExec: true,
	}
	if !full.AllSatisfied() {
		t.Error("all true should be satisfied")
	}

	empty := HighNonDestructiveConditions{}
	if empty.AllSatisfied() {
		t.Error("all false should not be satisfied")
	}
}
