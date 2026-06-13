package skill_step

import "testing"

// V-31-04：builtin lifecycle 預設——低風險唯讀 auto_execute=true；workspace_write/medium=false。
func TestBuiltinLifecycleDefaults(t *testing.T) {
	write := builtinDocWrite() // workspace_write + medium
	EnsureLifecycle(write)
	if write.Lifecycle == nil || write.Lifecycle.AutoExecute {
		t.Fatalf("write builtin 不應自動執行: %+v", write.Lifecycle)
	}
	if !write.Lifecycle.VisibleInToolbar {
		t.Fatal("write builtin 仍應出現在工具列")
	}
	search := builtinLocalSearch() // workspace_read + low
	EnsureLifecycle(search)
	if search.Lifecycle == nil || !search.Lifecycle.AutoExecute {
		t.Fatalf("low/read builtin 應可自動執行: %+v", search.Lifecycle)
	}
}

// V-31-06：canonicalHash 涵蓋 expected_chain 與 lifecycle 的變更。
func TestCanonicalHashCoversNewFields(t *testing.T) {
	base := &SkillManifest{
		SkillID: "weather.lookup", DisplayName: "天氣", Version: "1.0.0",
		Lifecycle:     &Lifecycle{Status: LifecycleEnabled, AutoExecute: false},
		ExpectedChain: &ExpectedChain{Schema: "skill_chain.v1", MaxSteps: 16},
	}
	h0 := canonicalHash(base)

	// 改 expected_chain → hash 必須變
	withStep := *base
	ec := *base.ExpectedChain
	ec.Steps = []ExpectedStep{{Action: "查詢", Target: "天氣", Code: "ㄔ", Requirement: "RE"}}
	withStep.ExpectedChain = &ec
	if canonicalHash(&withStep) == h0 {
		t.Fatal("改 expected_chain 後 hash 應改變（舊算法不會）")
	}

	// 改 lifecycle.auto_execute → hash 必須變
	flipped := *base
	lc := *base.Lifecycle
	lc.AutoExecute = true
	flipped.Lifecycle = &lc
	if canonicalHash(&flipped) == h0 {
		t.Fatal("改 lifecycle.auto_execute 後 hash 應改變")
	}
}
