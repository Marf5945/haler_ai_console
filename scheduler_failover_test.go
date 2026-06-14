// scheduler_failover_test.go — Phase C 故障切換順序單元測試。
package main

import "testing"

func TestOrderSchedulerFailover_FullCascade(t *testing.T) {
	cands := []failoverCandidate{
		{AdapterID: "api-openai", Kind: failoverAPI, Order: 0, Available: true},
		{AdapterID: "cli-b", Kind: failoverCLI, Order: 2, Available: true},
		{AdapterID: "ollama", Kind: failoverLocal, Order: 5, Available: true},
		{AdapterID: "cli-a", Kind: failoverCLI, Order: 1, Available: true},
		{AdapterID: "cli-offline", Kind: failoverCLI, Order: 0, Available: false},
	}
	got := orderSchedulerFailover(cands)
	want := []string{"cli-a", "cli-b", "ollama", "api-openai"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i].AdapterID != id {
			t.Errorf("pos %d = %q, want %q", i, got[i].AdapterID, id)
		}
	}
}

func TestOrderSchedulerFailover_APIAlwaysLast(t *testing.T) {
	cands := []failoverCandidate{
		{AdapterID: "api", Kind: failoverAPI, Order: -100, Available: true},
		{AdapterID: "loc", Kind: failoverLocal, Order: 999, Available: true},
	}
	got := orderSchedulerFailover(cands)
	if len(got) == 0 || got[len(got)-1].AdapterID != "api" {
		t.Fatalf("API 必須墊底，got %v", got)
	}
}

func TestOrderSchedulerFailover_ExcludesUnavailable(t *testing.T) {
	cands := []failoverCandidate{{AdapterID: "x", Kind: failoverCLI, Available: false}}
	if got := orderSchedulerFailover(cands); len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}
