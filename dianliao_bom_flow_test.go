package main

import (
	"strings"
	"testing"
	"time"

	"ui_console/builtin"
)

func TestParseBOMItemInput(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantPart string
		wantQty  string
		wantNote string
		wantOK   bool
	}{
		{"料號+數量", "ABC-123 5", "ABC-123", "5", "", true},
		{"逗號分隔", "ABC-123，5", "ABC-123", "5", "", true},
		{"頓號分隔", "ABC-123、10", "ABC-123", "10", "", true},
		{"料號+數量+註解", "XYZ-9 3 備品", "XYZ-9", "3", "備品", true},
		{"只有料號(數量留空)", "PN-001", "PN-001", "", "", true},
		{"小數數量", "PN-7 2.5", "PN-7", "2.5", "", true},
		{"空字串", "   ", "", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			it, ok := parseBOMItemInput(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if it.PartNo != tc.wantPart || it.Qty != tc.wantQty || it.Note != tc.wantNote {
				t.Fatalf("got {%q %q %q} want {%q %q %q}",
					it.PartNo, it.Qty, it.Note, tc.wantPart, tc.wantQty, tc.wantNote)
			}
		})
	}
}

func TestDianliaoControlWords(t *testing.T) {
	for _, w := range []string{"不要", "輸出", "好了", "ok", "結束"} {
		if !isDianliaoDoneText(w) {
			t.Errorf("isDianliaoDoneText(%q) = false, want true", w)
		}
	}
	for _, w := range []string{"繼續", "補", "要", "下一項"} {
		if !isDianliaoMoreText(w) {
			t.Errorf("isDianliaoMoreText(%q) = false, want true", w)
		}
	}
	for _, w := range []string{"取消", "算了", "cancel"} {
		if !isDianliaoCancelText(w) {
			t.Errorf("isDianliaoCancelText(%q) = false, want true", w)
		}
	}
	if isDianliaoDoneText("ABC-123 5") || isDianliaoMoreText("ABC-123 5") {
		t.Error("item text misclassified as control word")
	}
}

func TestNormalizeMachineInput(t *testing.T) {
	cases := map[string]string{
		"SLM003":      "SLM003",
		"機台 SLM003":   "SLM003",
		"機台:SLM003":   "SLM003",
		"機台是SLM003":   "SLM003",
		"機台名稱是all001": "all001",
		"  All-1111 ": "All-1111",
	}
	for in, want := range cases {
		if got := normalizeMachineInput(in); got != want {
			t.Errorf("normalizeMachineInput(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildDianliaoRequest(t *testing.T) {
	now := time.Date(2026, 6, 11, 9, 30, 0, 0, time.UTC)
	it, ok := parseBOMItemInput("ABC-1 2")
	if !ok {
		t.Fatal("parse failed")
	}
	req := buildDianliaoRequest("SLM003", []builtin.BOMItem{it}, now)
	if req.Machine != "SLM003" {
		t.Errorf("machine = %q, want SLM003", req.Machine)
	}
	if req.Date != "2026/06/11" {
		t.Errorf("date = %q, want 2026/06/11", req.Date)
	}
	if len(req.Sheets) != 1 || req.Sheets[0].Name != "SLM003" {
		t.Fatalf("sheets = %+v, want 1 sheet named SLM003", req.Sheets)
	}
	if len(req.Sheets[0].Items) != 1 || req.Sheets[0].Items[0].PartNo != "ABC-1" {
		t.Fatalf("items = %+v, want 1 item ABC-1", req.Sheets[0].Items)
	}
}

func TestDianliaoOutputFileName(t *testing.T) {
	now := time.Date(2026, 6, 11, 9, 30, 5, 0, time.UTC)
	got := dianliaoOutputFileName("SL/M:003", now)
	if !strings.HasPrefix(got, "電料BOM_SL-M-003_") || !strings.HasSuffix(got, ".xlsx") {
		t.Fatalf("unexpected file name: %q", got)
	}
	if strings.ContainsAny(got[:len(got)-5], `/\:*?"<>|`) {
		t.Fatalf("file name not sanitized: %q", got)
	}
}

func TestSplitItemInput(t *testing.T) {
	cases := []struct {
		in        string
		wantQuery string
		wantQty   string
	}{
		{"murr 10m", "murr 10m", ""},
		{"murr 10m 5", "murr 10m", "5"},
		{"ABC-123 5", "ABC-123", "5"},
		{"IO link 元件", "IO link 元件", ""},
		{"接頭，3", "接頭", "3"},
	}
	for _, tc := range cases {
		q, n := splitItemInput(tc.in)
		if q != tc.wantQuery || n != tc.wantQty {
			t.Errorf("splitItemInput(%q) = (%q,%q), want (%q,%q)", tc.in, q, n, tc.wantQuery, tc.wantQty)
		}
	}
}

func TestParseQtyInput(t *testing.T) {
	cases := map[string]struct {
		want string
		ok   bool
	}{
		"5":     {"5", true},
		"5個":    {"5", true},
		"數量 12": {"12", true},
		"2.5":   {"2.5", true},
		"沒有":    {"", false},
	}
	for in, exp := range cases {
		got, ok := parseQtyInput(in)
		if got != exp.want || ok != exp.ok {
			t.Errorf("parseQtyInput(%q) = (%q,%v), want (%q,%v)", in, got, ok, exp.want, exp.ok)
		}
	}
}

func TestParseModelPickNumbers(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want []int
	}{
		{"2,5", 8, []int{2, 5}},
		{" 1 ", 8, []int{1}},
		{"0", 8, nil},
		{"3,3,1", 8, []int{3, 1}},
		{"9", 8, nil}, // 超出範圍丟棄
		{"挑 2 和 4", 8, []int{2, 4}},
		{"", 8, nil},
	}
	for _, tc := range cases {
		got := parseModelPickNumbers(tc.in, tc.max)
		if len(got) != len(tc.want) {
			t.Fatalf("parseModelPickNumbers(%q) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("parseModelPickNumbers(%q) = %v, want %v", tc.in, got, tc.want)
			}
		}
	}
}

func TestMatchDianliaoPick(t *testing.T) {
	cands := []builtin.DianliaoRecord{
		{Name: "Murr.M12 公母接線", PartNoQuanta: "Q-001", Spec: "10m"},
		{Name: "端子台", PartNoQuanta: "Q-002"},
	}
	if r, ok := matchDianliaoPick("1", cands); !ok || r.PartNoQuanta != "Q-001" {
		t.Errorf("by index failed: %+v %v", r, ok)
	}
	if r, ok := matchDianliaoPick("q-002", cands); !ok || r.PartNoQuanta != "Q-002" {
		t.Errorf("by partno failed: %+v %v", r, ok)
	}
	if r, ok := matchDianliaoPick("端子台", cands); !ok || r.PartNoQuanta != "Q-002" {
		t.Errorf("by name failed: %+v %v", r, ok)
	}
	if _, ok := matchDianliaoPick("不存在", cands); ok {
		t.Error("should not match")
	}
}
