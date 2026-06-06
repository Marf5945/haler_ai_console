package review

import (
	"testing"

	"ui_console/domain/risk"
)

// 測試新增 Standard Card
func TestAddCard(t *testing.T) {
	svc := NewService()
	card := svc.AddCard(CardParams{
		RiskClass:   risk.UserOwnedAssetDestructive,
		Operation:   "delete_project",
		Target:      "project:demo",
		Reason:      "刪除使用者專案資產",
		AcceptLabel: "刪除",
		RejectLabel: "取消",
		AcceptEffect: "專案資料夾將被移除",
		RejectEffect: "不做任何變更",
	})
	if card.ID == "" {
		t.Error("card ID should not be empty")
	}
	if card.RiskClass != risk.UserOwnedAssetDestructive {
		t.Error("risk class mismatch")
	}
	if card.RequiresDualStep {
		t.Error("destructive should not require dual step")
	}
}

// 測試 security_boundary_rewrite 自動設定雙步驟
func TestCardDualStepAutoSet(t *testing.T) {
	svc := NewService()
	card := svc.AddCard(CardParams{
		RiskClass:   risk.SecurityBoundaryRewrite,
		Operation:   "modify_risk_policy",
		Target:      "risk_policy:main",
		Reason:      "修改風險策略",
		AcceptLabel: "我了解，繼續",
		RejectLabel: "取消",
		AcceptEffect: "風險策略將被修改",
		RejectEffect: "不做任何變更",
	})
	if !card.RequiresDualStep {
		t.Error("security_boundary_rewrite should require dual step")
	}
	if card.CooldownSeconds != 3 {
		t.Errorf("cooldown should be 3, got %d", card.CooldownSeconds)
	}
}

// 測試雙步驟確認流程
func TestDualStepFlow(t *testing.T) {
	svc := NewService()
	card := svc.AddCard(CardParams{
		RiskClass:   risk.SecurityBoundaryRewrite,
		Operation:   "modify_risk_policy",
		Target:      "risk_policy:main",
		Reason:      "test",
		AcceptLabel: "確認",
		RejectLabel: "取消",
	})

	// 未完成 Step 1 時直接 Resolve 應失敗
	err := svc.Resolve(card.ID)
	if err == nil {
		t.Error("resolve without step 1 should fail")
	}

	// 完成 Step 1
	err = svc.DualStepConfirmStep1(card.ID, "scope_abc", "risk_abc", "tool_abc")
	if err != nil {
		t.Fatalf("step 1 failed: %v", err)
	}

	// Step 2 太快應被冷卻時間擋住（3 秒）
	err = svc.Resolve(card.ID)
	if err == nil {
		t.Error("resolve within cooldown should fail")
	}
}

// 測試雙步驟失效
func TestDualStepInvalidation(t *testing.T) {
	svc := NewService()
	card := svc.AddCard(CardParams{
		RiskClass:   risk.SecurityBoundaryRewrite,
		Operation:   "modify_risk_policy",
		Target:      "risk_policy:main",
		Reason:      "test",
		AcceptLabel: "確認",
		RejectLabel: "取消",
	})

	svc.DualStepConfirmStep1(card.ID, "scope_abc", "risk_abc", "tool_abc")

	// 模擬 hash 變更 → 失效
	svc.InvalidateDualStep(card.ID)

	// 失效後 Resolve 應失敗
	err := svc.Resolve(card.ID)
	if err == nil {
		t.Error("resolve with invalidated dual step should fail")
	}
}

// 測試 Lightweight Card 使用限制（§5.2.3）
func TestLightweightAllowed(t *testing.T) {
	svc := NewService()

	// scopeMatch=true, risk=medium → 允許
	_, err := svc.AddLightweightCard(LightweightParams{
		Operation:   "renew_source_allowlist",
		Target:      "source:example.edu",
		AcceptLabel: "續期",
		RejectLabel: "保持到期",
	}, true, risk.Medium)
	if err != nil {
		t.Errorf("should allow lightweight for medium: %v", err)
	}

	// scopeMatch=true, risk=high_non_destructive → 允許
	_, err = svc.AddLightweightCard(LightweightParams{
		Operation:   "renew",
		Target:      "source:test",
		AcceptLabel: "續期",
		RejectLabel: "不續期",
	}, true, risk.HighNonDestructive)
	if err != nil {
		t.Errorf("should allow lightweight for high_non_destructive: %v", err)
	}

	// scopeMatch=false → 不允許
	_, err = svc.AddLightweightCard(LightweightParams{
		Operation:   "grant",
		Target:      "source:new",
		AcceptLabel: "允許",
		RejectLabel: "拒絕",
	}, false, risk.Medium)
	if err == nil {
		t.Error("should reject lightweight when scope mismatch")
	}

	// risk=destructive → 不允許
	_, err = svc.AddLightweightCard(LightweightParams{
		Operation:   "renew",
		Target:      "source:test",
		AcceptLabel: "續期",
		RejectLabel: "不續期",
	}, true, risk.UserOwnedAssetDestructive)
	if err == nil {
		t.Error("should reject lightweight for destructive risk")
	}
}

// 測試 HasBlocking 使用新的風險等級
func TestHasBlockingWithRiskClass(t *testing.T) {
	svc := NewService()

	// 無卡片時不 blocking
	if svc.HasBlocking() {
		t.Error("empty service should not have blocking")
	}

	// 新增 medium 卡片 → 不 blocking
	svc.AddCard(CardParams{
		RiskClass:   risk.Medium,
		Operation:   "test",
		Target:      "test",
		Reason:      "test",
		AcceptLabel: "ok",
		RejectLabel: "cancel",
	})
	if svc.HasBlocking() {
		t.Error("medium should not be blocking")
	}

	// 新增 high_non_destructive 卡片 → blocking
	svc.AddCard(CardParams{
		RiskClass:   risk.HighNonDestructive,
		Operation:   "high_op",
		Target:      "target",
		Reason:      "high",
		AcceptLabel: "確認",
		RejectLabel: "取消",
	})
	if !svc.HasBlocking() {
		t.Error("high_non_destructive should be blocking")
	}
}

// 測試舊版相容介面
func TestAddLegacyCard(t *testing.T) {
	svc := NewService()
	card := svc.AddLegacyCard(LevelBlocking, "package_import", "pkg_001", "安裝套件需要確認", "quarantine check")

	if card.RiskClass != risk.HighNonDestructive {
		t.Errorf("legacy blocking should map to high_non_destructive, got %s", card.RiskClass)
	}
	if card.LegacyLevel != LevelBlocking {
		t.Errorf("legacy level should be preserved, got %s", card.LegacyLevel)
	}
}

// 測試 InvalidateAll 含 Lightweight
func TestInvalidateAllIncludesLightweight(t *testing.T) {
	svc := NewService()
	svc.AddCard(CardParams{
		RiskClass: risk.Medium, Operation: "a", Target: "a",
		Reason: "a", AcceptLabel: "ok", RejectLabel: "cancel",
	})
	svc.AddLightweightCard(LightweightParams{
		Operation: "b", Target: "b", AcceptLabel: "ok", RejectLabel: "cancel",
	}, true, risk.Low)

	count := svc.InvalidateAll()
	if count != 2 {
		t.Errorf("should invalidate 2 cards, got %d", count)
	}
}

// 測試 IsLightweightAllowed 邊界情境
func TestIsLightweightAllowedEdge(t *testing.T) {
	// low + scope match → 允許
	if !IsLightweightAllowed(true, risk.Low) {
		t.Error("low with scope match should be allowed")
	}

	// critical + scope match → 不允許
	if IsLightweightAllowed(true, risk.CriticalRuntimeAction) {
		t.Error("critical should never allow lightweight")
	}

	// security_boundary_rewrite + scope match → 不允許
	if IsLightweightAllowed(true, risk.SecurityBoundaryRewrite) {
		t.Error("security_boundary_rewrite should never allow lightweight")
	}
}
