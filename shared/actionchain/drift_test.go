package actionchain

import "testing"

func TestDetectDrift_InsufficientToNetwork(t *testing.T) {
	expected := []ActionChain{
		{Action: "輸入", Target: "天氣", Next: "輸出"},
		{Action: "輸出", Target: "請問地點", Next: "待命"},
	}
	actual := []ActionChain{
		{Action: "輸入", Target: "天氣", Next: "網路"},
	}
	drifts := DetectDrift(expected, actual)
	if len(drifts) != 1 {
		t.Fatalf("want 1 drift, got %d", len(drifts))
	}
	d := drifts[0]
	if d.Type != DriftInsufficientToNetwork || d.Decision != DriftDecisionAskUser {
		t.Errorf("want insufficient_to_network/ask_user, got %+v", d)
	}
	if d.MissingSlot != "地點" {
		t.Errorf("want missing slot 地點, got %q", d.MissingSlot)
	}
}

func TestDetectDrift_NoDrift(t *testing.T) {
	chain := []ActionChain{{Action: "輸入", Target: "天氣", Next: "輸出"}}
	if d := DetectDrift(chain, chain); len(d) != 0 {
		t.Errorf("identical chains should have no drift, got %+v", d)
	}
}

func TestDetectDrift_ActionMismatch(t *testing.T) {
	expected := []ActionChain{{Action: "查詢", Target: "x", Next: "輸出"}}
	actual := []ActionChain{{Action: "網路", Target: "x", Next: "輸出"}}
	d := DetectDrift(expected, actual)
	if len(d) != 1 || d[0].Type != DriftActionMismatch || d[0].Decision != DriftDecisionReview {
		t.Errorf("want action_mismatch/review, got %+v", d)
	}
}
