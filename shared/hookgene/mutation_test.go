package hookgene

import (
	"math/rand"
	"testing"
)

func TestCanTransition(t *testing.T) {
	allow := [][2]CandidateState{
		{StateStaged, StatePendingReview},
		{StateStaged, StateDormant},
		{StatePendingReview, StateDormant},
		{StateDormant, StatePendingReview},
	}
	for _, c := range allow {
		if !CanTransition(c[0], c[1]) {
			t.Fatalf("expected allowed: %s → %s", c[0], c[1])
		}
	}
	// 任何 → ACTIVE 一律禁止（MVP 不自動啟用，§3.1.5.18.6）。
	deny := [][2]CandidateState{
		{StateStaged, StateActive},
		{StatePendingReview, StateActive},
		{StateDormant, StateActive},
	}
	for _, c := range deny {
		if CanTransition(c[0], c[1]) {
			t.Fatalf("expected DENIED: %s → %s", c[0], c[1])
		}
	}
}

func TestMutateGeneKeepsLengthAndAlphabet(t *testing.T) {
	src := []HookCode{HookInput, HookList, HookList, HookOutput, HookStandby}
	c := CopyForMutation("skill-x", src)
	if c.State != StateStaged {
		t.Fatalf("new candidate state = %s, want STAGED", c.State)
	}
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		MutateGene(&c, r)
	}
	if len(c.Gene) != len(src) {
		t.Fatalf("gene length changed: %d", len(c.Gene))
	}
	valid := map[HookCode]bool{HookList: true, HookInput: true, HookOutput: true, HookStandby: true}
	for _, h := range c.Gene {
		if !valid[h] {
			t.Fatalf("invalid hook after mutation: %q", string(h))
		}
	}
	// 原始來源不可被修改。
	if src[0] != HookInput {
		t.Fatal("mutation must not modify the original source gene")
	}
}
