package hookgene

import (
	"errors"
	"testing"
)

func TestDenyCoreLayerMutationBlocksCoreActions(t *testing.T) {
	blocked := []CoreAction{
		ActionActiveSkill, ActionCoreMemoryWrite, ActionPersonaCoreUpdate,
		ActionRiskPolicyChange, ActionSecurityBoundary, ActionActiveSubagentEnable,
	}
	phases := []GuardPhase{PhaseGenerate, PhaseReview, PhaseExecute}
	for _, a := range blocked {
		for _, p := range phases {
			err := DenyCoreLayerMutation(p, a)
			if err == nil {
				t.Fatalf("expected denial for %s at %s", a, p)
			}
			if !errors.Is(err, ErrCoreLayerDenied) {
				t.Fatalf("error should wrap ErrCoreLayerDenied: %v", err)
			}
		}
	}
}

func TestDenyCoreLayerMutationAllowsNonCore(t *testing.T) {
	if err := DenyCoreLayerMutation(PhaseGenerate, CoreAction("read_only_probe")); err != nil {
		t.Fatalf("non-core action should be allowed, got %v", err)
	}
}
