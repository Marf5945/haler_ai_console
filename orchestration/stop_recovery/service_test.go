package stop_recovery

import (
	"testing"
)

// 測試建立 sidecar crash 恢復卡片
func TestCreateCardSidecarCrash(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonSidecarCrash, "exit code 1")

	if card.ID == "" {
		t.Error("card ID should not be empty")
	}
	if card.StopReason != ReasonSidecarCrash {
		t.Errorf("reason should be sidecar_crashed, got %s", card.StopReason)
	}
	if len(card.SafeNextActions) < 2 {
		t.Error("should have at least 2 actions")
	}
	if card.UserMessage == "" {
		t.Error("user message should not be empty")
	}
}

// 測試建立 critical_runtime_action 恢復卡片
func TestCreateCardCriticalRuntime(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonCriticalRuntimeAction, "payment_confirmation")

	if card.StopReason != ReasonCriticalRuntimeAction {
		t.Error("wrong reason")
	}
	// 應包含 dry-run 選項
	hasDryRun := false
	for _, a := range card.SafeNextActions {
		if a.Action == ActionDryRunCurrentStep {
			hasDryRun = true
		}
	}
	if !hasDryRun {
		t.Error("critical runtime should offer dry-run option")
	}
}

// 測試解決恢復卡片
func TestResolveCard(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonUserStop, "user pressed stop")

	err := svc.ResolveCard(card.ID, ActionHandleManuallyThenResume)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	resolved, _ := svc.GetCard(card.ID)
	if !resolved.Resolved {
		t.Error("card should be resolved")
	}
	if resolved.ResolvedAction != string(ActionHandleManuallyThenResume) {
		t.Error("resolved action mismatch")
	}
}

// 測試禁止動作被拒絕
func TestResolveCardForbiddenAction(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonSidecarCrash, "crash")

	err := svc.ResolveCard(card.ID, RecoveryAction("auto_login"))
	if err == nil {
		t.Error("forbidden action should be rejected")
	}
}

// 測試重複解決被拒絕
func TestResolveCardAlreadyResolved(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonUserStop, "stop")
	svc.ResolveCard(card.ID, ActionDiscardSandbox)

	err := svc.ResolveCard(card.ID, ActionRestartSidecar)
	if err == nil {
		t.Error("already resolved card should reject")
	}
}

// 測試 ListOpen
func TestListOpen(t *testing.T) {
	svc := NewService()
	svc.CreateCard(ReasonSidecarCrash, "crash1")
	card2 := svc.CreateCard(ReasonUserStop, "stop")
	svc.ResolveCard(card2.ID, ActionDiscardSandbox)

	open := svc.ListOpen()
	if len(open) != 1 {
		t.Errorf("expected 1 open card, got %d", len(open))
	}
}

// 測試 HasOpen
func TestHasOpen(t *testing.T) {
	svc := NewService()
	if svc.HasOpen() {
		t.Error("should have no open cards initially")
	}

	svc.CreateCard(ReasonSidecarCrash, "crash")
	if !svc.HasOpen() {
		t.Error("should have open card")
	}
}

// 測試 IsForbiddenAction
func TestIsForbiddenAction(t *testing.T) {
	forbidden := []string{
		"auto_login", "auto_confirm_payment", "auto_solve_captcha",
		"auto_grant_permission", "ignore_and_continue_for_critical",
	}
	for _, a := range forbidden {
		if !IsForbiddenAction(a) {
			t.Errorf("%s should be forbidden", a)
		}
	}

	allowed := []string{"restart_sidecar_runner", "discard_sandbox"}
	for _, a := range allowed {
		if IsForbiddenAction(a) {
			t.Errorf("%s should not be forbidden", a)
		}
	}
}

// 測試 resume guard failed 卡片
func TestCreateCardResumeGuardFailed(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonResumeGuardFailed, "memory hash changed")

	if card.StopReason != ReasonResumeGuardFailed {
		t.Error("wrong reason")
	}
	if len(card.SafeNextActions) < 2 {
		t.Error("should have at least 2 actions")
	}
}
