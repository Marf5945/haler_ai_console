package replan

import (
	"context"
	"errors"
	"testing"

	"ui_console/domain/risk"
)

type fakeCritic struct {
	concern bool
	err     error
	called  bool
}

func (f *fakeCritic) Review(_ context.Context, _ ProposerContext, _ ReplanProposal) (CriticVerdict, error) {
	f.called = true
	return CriticVerdict{Concern: f.concern, Note: "test"}, f.err
}

func silentProposal() ReplanProposal {
	return ReplanProposal{Intent: IntentSameGoalPath, Confidence: 0.8,
		ProposedTail: []ProposedNode{{Action: "grep_search", Target: "proj/data"}}}
}

func TestCoordinator_CriticBorderlineDowngrades(t *testing.T) {
	run := coordRun()
	c := NewCounter()
	c.ConsecutiveNoProgress = 3 // borderline
	fc := &fakeCritic{concern: true}
	co := NewCoordinator(fakeProposer{proposal: silentProposal()}, c, nil, 0)
	co.Critic = fc

	res := co.Attempt(run, FailureNoResults, risk.Low)
	if !fc.called {
		t.Fatalf("critic should be consulted on borderline")
	}
	if res.Decision != DecisionReview || res.Applied {
		t.Fatalf("critic concern must downgrade silent->review, got %s applied=%v", res.Decision, res.Applied)
	}
	if run.Revision != 0 {
		t.Errorf("downgraded attempt must not mutate run")
	}
}

func TestCoordinator_CriticSkippedWhenNotBorderline(t *testing.T) {
	run := coordRun()
	fc := &fakeCritic{concern: true} // 即使會反對，也不該被呼叫
	co := NewCoordinator(fakeProposer{proposal: silentProposal()}, NewCounter(), nil, 0)
	co.Critic = fc

	res := co.Attempt(run, FailureNoResults, risk.Low)
	if fc.called {
		t.Errorf("critic must NOT be consulted when not borderline")
	}
	if res.Decision != DecisionSilent || !res.Applied {
		t.Fatalf("non-borderline silent should proceed, got %s applied=%v", res.Decision, res.Applied)
	}
}

func TestCoordinator_CriticErrorDowngrades(t *testing.T) {
	run := coordRun()
	c := NewCounter()
	c.ConsecutiveNoProgress = 3
	fc := &fakeCritic{err: errors.New("critic timeout")}
	co := NewCoordinator(fakeProposer{proposal: silentProposal()}, c, nil, 0)
	co.Critic = fc
	if res := co.Attempt(run, FailureNoResults, risk.Low); res.Decision != DecisionReview {
		t.Fatalf("critic error must fail-safe to review, got %s", res.Decision)
	}
}
