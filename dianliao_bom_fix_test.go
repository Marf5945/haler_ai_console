package main

import "testing"

func TestHasDianliaoFixIntent(t *testing.T) {
	yes := []string{
		"幫我修正 人機 為 4項",
		"第2項改成5",
		"80160001 打錯，應該是10",
		"人機數量錯了",
		"把人機改為4個",
	}
	for _, s := range yes {
		if !hasDianliaoFixIntent(s) {
			t.Errorf("hasDianliaoFixIntent(%q) = false, want true", s)
		}
	}
	no := []string{
		"murr 10m",
		"80160001 21",
		"人機 21",
		"不要",
	}
	for _, s := range no {
		if hasDianliaoFixIntent(s) {
			t.Errorf("hasDianliaoFixIntent(%q) = true, want false", s)
		}
	}
}

func TestParseDianliaoFix(t *testing.T) {
	cases := []struct {
		in     string
		target string
		idx    int
		qty    string
	}{
		{"幫我修正 人機 為 4項", "人機", 0, "4"},
		{"第2項改成5", "", 2, "5"},
		{"第 3 個 改成 12", "", 3, "12"},
		{"第2項 5", "", 2, "5"},
		{"80160001 改成 10", "80160001", 0, "10"},
		{"把人機改為4個", "人機", 0, "4"},
		{"人機數量錯了，應該是7", "人機", 0, "7"},
		{"人機改4", "人機", 0, "4"},
		// 料號 8 碼不能被當數量；qty 留空之後補問。
		{"80160001 打錯了", "80160001", 0, ""},
		{"人機數量錯了", "人機", 0, ""},
	}
	for _, c := range cases {
		target, idx, qty := parseDianliaoFix(c.in)
		if target != c.target || idx != c.idx || qty != c.qty {
			t.Errorf("parseDianliaoFix(%q) = (%q,%d,%q), want (%q,%d,%q)",
				c.in, target, idx, qty, c.target, c.idx, c.qty)
		}
	}
}

func TestStripDianliaoFixFiller(t *testing.T) {
	if got := stripDianliaoFixFiller("幫我修正 人機 "); got != "人機" {
		t.Errorf("got %q", got)
	}
	if got := stripDianliaoFixFiller("  "); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestContainsIntSlice(t *testing.T) {
	if !containsIntSlice([]int{1, 2}, 2) || containsIntSlice([]int{1, 2}, 3) {
		t.Error("containsIntSlice mismatch")
	}
}
