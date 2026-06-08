package actionchain

import (
	"strings"
	"testing"
)

func TestHumanStatus(t *testing.T) {
	cases := map[string][2]string{ // action -> [want-substring, must-not-contain]
		"搜尋": {"搜尋本機資料", "ㄌ"},
		"網路": {"用網路搜尋", "ㄌ"},
		"讀取": {"查閱資料", "ㄌ"},
		"列出": {"列出項目", "ㄌ"},
	}
	for action, want := range cases {
		got := HumanStatus(action, "甜點食譜", PhaseRunning)
		if !strings.Contains(got, want[0]) || !strings.Contains(got, "甜點食譜") {
			t.Errorf("%s: got %q want substr %q", action, got, want[0])
		}
		if strings.Contains(got, "ㄌ") {
			t.Errorf("%s: must not leak wire format: %q", action, got)
		}
	}
}

func TestHumanStatus_UnknownFailSafe(t *testing.T) {
	got := HumanStatus("刪除", "磁碟", PhaseRunning)
	if !strings.Contains(got, "處理") {
		t.Errorf("unknown action should fall back to 處理, got %q", got)
	}
	if strings.Contains(got, "刪除") || strings.Contains(got, "ㄌ") {
		t.Errorf("must not leak raw action/wire format: %q", got)
	}
}

func TestHumanStatus_PendingVsRunning(t *testing.T) {
	if !strings.HasPrefix(HumanStatus("網路", "x", PhasePending), "準備") {
		t.Errorf("pending should start with 準備")
	}
	r := HumanStatus("網路", "x", PhaseRunning)
	if !strings.HasPrefix(r, "正在") || !strings.HasSuffix(r, "…") {
		t.Errorf("running should be 正在…, got %q", r)
	}
}

func TestTruncateForDisplay(t *testing.T) {
	long := strings.Repeat("字", statusDisplayMaxRunes+20)
	out := HumanStatus("查詢", long, PhaseRunning)
	if !strings.Contains(out, "…") {
		t.Errorf("long target should be truncated with …")
	}
}
