package replan

import (
	"testing"

	"ui_console/orchestration/dag"
)

func TestCounter_ProgressResetsOnlyOnSucceededWithSummary(t *testing.T) {
	c := NewCounter()
	c.RecordReplan("sig-a")
	c.RecordReplan("sig-b")
	if c.ConsecutiveNoProgress != 2 {
		t.Fatalf("want 2, got %d", c.ConsecutiveNoProgress)
	}
	// skipped 不歸零
	c.RecordProgress(dag.DAGNode{Status: dag.StatusSkipped, ResultSummary: "x"})
	if c.ConsecutiveNoProgress != 2 {
		t.Errorf("skipped must not reset")
	}
	// succeeded 但空 summary 不歸零
	c.RecordProgress(dag.DAGNode{Status: dag.StatusSucceeded, ResultSummary: ""})
	if c.ConsecutiveNoProgress != 2 {
		t.Errorf("succeeded with empty summary must not reset")
	}
	// succeeded + 非空 summary 才歸零
	c.RecordProgress(dag.DAGNode{Status: dag.StatusSucceeded, ResultSummary: "done"})
	if c.ConsecutiveNoProgress != 0 {
		t.Errorf("succeeded+summary must reset to 0, got %d", c.ConsecutiveNoProgress)
	}
}

func TestCounter_OscillationAccelerates(t *testing.T) {
	c := NewCounter()
	c.RecordReplan("sig-x") // 首次：+1
	c.RecordReplan("sig-x") // 雷同：+OscillationPenalty
	want := 1 + OscillationPenalty
	if c.ConsecutiveNoProgress != want {
		t.Fatalf("oscillation want %d, got %d", want, c.ConsecutiveNoProgress)
	}
	if !c.IsOscillating("sig-x") {
		t.Errorf("sig-x should be detected as oscillating")
	}
}

func TestCounter_ShouldStop(t *testing.T) {
	c := NewCounter()
	for i := 0; i < MaxConsecutiveNoProgress; i++ {
		c.RecordReplan("sig-" + string(rune('a'+i)))
	}
	if !c.ShouldStop() {
		t.Errorf("should stop at consecutive max")
	}

	c2 := NewCounter()
	c2.RunTotal = MaxRunTotal
	if !c2.ShouldStop() {
		t.Errorf("should stop at run total cap")
	}
}

func TestStageFor(t *testing.T) {
	cases := map[int]Stage{1: StageSilentNotice, 2: StageSilentNotice, 3: StageAdjusting, 4: StageAdjusting, 5: StageStop}
	for n, want := range cases {
		if got := StageFor(n); got != want {
			t.Errorf("StageFor(%d)=%s want %s", n, got, want)
		}
	}
}
