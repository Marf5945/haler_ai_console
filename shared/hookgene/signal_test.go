package hookgene

import "testing"

func TestHookForBoundary(t *testing.T) {
	// data_left 跨邊界 → ㄔ；未跨邊界 → ㄅ。
	out, ok := HookFor(Signal{Type: SignalDataLeft, CrossedBoundary: true})
	if !ok || out != HookOutput {
		t.Fatalf("crossed boundary should be ㄔ, got %q ok=%v", string(out), ok)
	}
	internal, ok := HookFor(Signal{Type: SignalDataLeft, CrossedBoundary: false})
	if !ok || internal != HookList {
		t.Fatalf("internal transfer should be ㄅ, got %q", string(internal))
	}
	if _, ok := HookFor(Signal{Type: SignalCompleted}); ok {
		t.Fatal("completed should not yield a hook")
	}
}
