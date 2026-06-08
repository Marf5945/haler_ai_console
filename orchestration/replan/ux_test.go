package replan

import (
	"strings"
	"testing"
)

func TestActivitySummary(t *testing.T) {
	// silent_notice：不帶計數。
	s1 := ActivitySummary(FailureNoResults, StageSilentNotice, 1)
	if strings.Contains(s1, "/") {
		t.Errorf("silent_notice should not show count: %q", s1)
	}
	if s1 == "" {
		t.Errorf("summary should not be empty")
	}
	// adjusting：帶 (n/5)。
	s2 := ActivitySummary(FailureNoResults, StageAdjusting, 3)
	if !strings.Contains(s2, "3/5") {
		t.Errorf("adjusting should show 3/5: %q", s2)
	}
	// 不暴露 raw：只是固定文案，不含 path/token（基本檢查）。
	if strings.Contains(s2, "/Users") {
		t.Errorf("must not leak paths")
	}
}
