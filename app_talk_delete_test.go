package main

import "testing"

// TestTalkBlockDisplay 守住刪除比對用的顯示字串還原，必須與 parseTalkMessages 一致，
// 否則前端傳來的訊息字串會比對不到、刪不掉檔案條目。
func TestTalkBlockDisplay(t *testing.T) {
	cases := []struct {
		name  string
		block string
		want  string
		ok    bool
	}{
		{"assistant", "2026-06-07 10:00:00] assistant\n你好，今天天氣晴。", "Ai:你好，今天天氣晴。", true},
		{"user", "2026-06-07 10:01:00] user\n幫我查台中天氣", "幫我查台中天氣", true},
		{"other", "2026-06-07 10:02:00] system\n已重啟", "[system] 已重啟", true},
		{"no-body", "2026-06-07 10:03:00] user", "", false},
		{"empty-body", "2026-06-07 10:04:00] user\n   ", "", false},
	}
	for _, c := range cases {
		got, ok := talkBlockDisplay(c.block)
		if ok != c.ok || got != c.want {
			t.Errorf("%s: got (%q,%v), want (%q,%v)", c.name, got, ok, c.want, c.ok)
		}
	}
}
