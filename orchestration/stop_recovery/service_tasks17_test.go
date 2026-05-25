// stop_recovery/service_tasks17_test.go — TASKS_1_7 驗收測試
package stop_recovery

import "testing"

// ── 驗收 5：critical_runtime_action + restart_sidecar 必須被拒絕 ──
func TestResolveCard_CriticalCannotRestart(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonCriticalRuntimeAction, "detected: login prompt")

	err := svc.ResolveCard(card.ID, ActionRestartSidecar)
	if err == nil {
		t.Error("critical_runtime_action + restart_sidecar should be rejected")
	}
}

// ── 驗收 5b：resume_guard_failed + restart_sidecar 必須被拒絕 ──
func TestResolveCard_ResumeGuardCannotRestart(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonResumeGuardFailed, "hash mismatch")

	err := svc.ResolveCard(card.ID, ActionRestartSidecar)
	if err == nil {
		t.Error("resume_guard_failed + restart_sidecar should be rejected")
	}
}

// ── 驗收 6：Stop Recovery card 必須包含 resume_conditions ──
func TestCreateCard_HasResumeConditions(t *testing.T) {
	svc := NewService()

	cases := []struct {
		reason StopReason
		signal string
	}{
		{ReasonSidecarCrash, "pipe broken"},
		{ReasonCriticalRuntimeAction, "login prompt"},
		{ReasonUserStop, "user pressed stop"},
		{ReasonResumeGuardFailed, "hash mismatch"},
	}

	for _, tc := range cases {
		card := svc.CreateCard(tc.reason, tc.signal)
		if card.ResumeConditions == nil || len(card.ResumeConditions) == 0 {
			t.Errorf("reason=%s should have resume_conditions", tc.reason)
		}
	}
}

// ── sidecar_crashed 可以 restart ──
func TestResolveCard_SidecarCrashCanRestart(t *testing.T) {
	svc := NewService()
	card := svc.CreateCard(ReasonSidecarCrash, "process exited")

	err := svc.ResolveCard(card.ID, ActionRestartSidecar)
	if err != nil {
		t.Errorf("sidecar_crashed should allow restart: %v", err)
	}
}
