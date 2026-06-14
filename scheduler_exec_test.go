// scheduler_exec_test.go — Phase E 故障切換執行迴圈與分類的單元測試。
package main

import (
	"errors"
	"testing"

	"ui_console/adapter/adapter_registry"
)

func TestRunSchedulerWithFailover_SkipsRetriableThenSucceeds(t *testing.T) {
	cands := []failoverCandidate{
		{AdapterID: "cli-a", Kind: failoverCLI, Order: 0, Available: true},
		{AdapterID: "cli-b", Kind: failoverCLI, Order: 1, Available: true},
		{AdapterID: "api", Kind: failoverAPI, Order: 0, Available: true},
	}
	var tried []string
	got, err := runSchedulerWithFailover(cands, func(id string) schedulerAttemptResult {
		tried = append(tried, id)
		if id == "cli-a" {
			return schedulerAttemptResult{ok: false, retriable: true, err: errors.New("quota exhausted")}
		}
		return schedulerAttemptResult{ok: true}
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "cli-b" {
		t.Fatalf("got %q want cli-b (tried=%v)", got, tried)
	}
	if len(tried) != 2 {
		t.Fatalf("應在成功後停止，tried=%v", tried)
	}
}

func TestRunSchedulerWithFailover_NonRetriableStops(t *testing.T) {
	cands := []failoverCandidate{
		{AdapterID: "cli-a", Kind: failoverCLI, Order: 0, Available: true},
		{AdapterID: "cli-b", Kind: failoverCLI, Order: 1, Available: true},
	}
	var tried []string
	_, err := runSchedulerWithFailover(cands, func(id string) schedulerAttemptResult {
		tried = append(tried, id)
		return schedulerAttemptResult{ok: false, retriable: false, err: errors.New("skill logic error")}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(tried) != 1 {
		t.Fatalf("非可重試應在第一個就停止，tried=%v", tried)
	}
}

func TestRunSchedulerWithFailover_AllRetriableFail(t *testing.T) {
	cands := []failoverCandidate{
		{AdapterID: "cli-a", Kind: failoverCLI, Order: 0, Available: true},
		{AdapterID: "api", Kind: failoverAPI, Order: 0, Available: true},
	}
	if _, err := runSchedulerWithFailover(cands, func(id string) schedulerAttemptResult {
		return schedulerAttemptResult{ok: false, retriable: true, err: errors.New("timeout")}
	}); err == nil {
		t.Fatal("全部失敗時應回傳錯誤")
	}
}

func TestRunSchedulerWithFailover_NoCandidates(t *testing.T) {
	if _, err := runSchedulerWithFailover(nil, func(string) schedulerAttemptResult {
		return schedulerAttemptResult{ok: true}
	}); err == nil {
		t.Fatal("無候選時應回傳錯誤")
	}
}

func TestClassifySchedulerAdapterKind(t *testing.T) {
	cases := []struct {
		ad   adapter_registry.Adapter
		kind failoverKind
		keep bool
	}{
		{adapter_registry.Adapter{Kind: "api"}, failoverAPI, true},
		{adapter_registry.Adapter{Kind: "cli"}, failoverCLI, true},
		{adapter_registry.Adapter{Kind: "cli", Endpoint: "http://localhost:11434"}, failoverLocal, true},
		{adapter_registry.Adapter{Kind: "main"}, failoverCLI, false},
		{adapter_registry.Adapter{Kind: "sub"}, failoverCLI, false},
	}
	for i, c := range cases {
		k, keep := classifySchedulerAdapterKind(c.ad)
		if keep != c.keep {
			t.Errorf("case %d keep=%v want %v", i, keep, c.keep)
		}
		if keep && k != c.kind {
			t.Errorf("case %d kind=%v want %v", i, k, c.kind)
		}
	}
}

func TestIsSchedulerRetriableFailure(t *testing.T) {
	retriable := []string{"connection refused", "request timeout", "service unavailable", "no capacity available", "broken pipe"}
	for _, m := range retriable {
		if !isSchedulerRetriableFailure(m) {
			t.Errorf("%q 應可重試", m)
		}
	}
	if isSchedulerRetriableFailure("skill not found") {
		t.Error("skill not found 不應可重試")
	}
}
