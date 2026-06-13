package dag

import "testing"

func TestGoalContract_IsZero(t *testing.T) {
	if !(GoalContract{}).IsZero() {
		t.Errorf("empty contract must be zero (legacy)")
	}
	if (GoalContract{GoalSummary: "x"}).IsZero() {
		t.Errorf("contract with goal summary must not be zero")
	}
}

func TestGoalContract_HashStableAndSensitive(t *testing.T) {
	a := GoalContract{GoalSummary: "整理研究", OutputType: "file", Scope: "proj/"}
	b := GoalContract{GoalSummary: "整理研究", OutputType: "file", Scope: "proj/"}
	if a.Hash() != b.Hash() {
		t.Errorf("same content must hash equal")
	}
	c := GoalContract{GoalSummary: "整理研究", OutputType: "file", Scope: "other/"}
	if a.Hash() == c.Hash() {
		t.Errorf("different scope must hash differently")
	}
}

func TestNewGoalContractFromPlan(t *testing.T) {
	gc := NewGoalContractFromPlan("  幫我整理資料  ", TaskPlan{Title: "t"})
	if gc.IsZero() {
		t.Fatalf("derived contract must not be zero")
	}
	if gc.GoalSummary != "幫我整理資料" {
		t.Errorf("goal summary should be trimmed, got %q", gc.GoalSummary)
	}
}
