// conversation_root_test.go — SEC-W03 agentID 路徑驗證
//
// 第二刀（2026-05-24）為 conversationRootForAgent 加上 validAgentID regex，
// 比舊版 `strings.ContainsAny(id, "/\\")` 嚴格——拒絕 ".." 等路徑穿越字元。
//
// 注意：conversationRootForAgent 內部有 white-list 早退處理空字串 / "main" /
// "主haㄌer" 三個非 regex 友善的合法 ID；本測試只驗 regex 行為本身，
// 不觸碰檔案系統，與 sub_export_sec_test.go (SEC-14) 同風格。
//
// 執行：go test -run TestValidAgentIDRegex -v
package main

import "testing"

func TestValidAgentIDRegex(t *testing.T) {
	cases := []struct {
		name  string
		id    string
		valid bool
	}{
		// ── 合法 sub-agent id 形式（必須通過）──
		{"timestamp sub", "sub-1716000000", true},
		{"snake case", "my_sub_agent", true},
		{"hyphen mixed", "my-sub-1", true},
		{"single letter", "x", true},
		{"all digits", "12345", true},
		{"main string also passes regex (whitelist superset)", "main", true},

		// ── 路徑穿越 / 控制字元（核心 SEC-W03 攻擊面，必須 reject）──
		{"dotdot", "..", false},
		{"single dot", ".", false},
		{"forward slash with dotdot", "../etc", false},
		{"backslash with dotdot", `..\etc`, false},
		{"absolute path", "/etc/passwd", false},
		{"four dots traversal", "....", false}, // Windows 8.3 short-name 變種
		{"null byte", "abc\x00def", false},
		{"newline injection", "abc\ndef", false},
		{"whitespace inside", "a b", false},
		{"trailing slash", "abc/", false},

		// ── 非 ASCII（regex 擋掉是預期；中文 ID 走 white-list 不走 regex）──
		{"chinese", "你好", false},
		{"emoji", "🦀", false},

		// ── 空字串（white-list 早退處理，regex 仍應拒絕）──
		{"empty", "", false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := validAgentID.MatchString(tt.id)
			if got != tt.valid {
				t.Errorf("validAgentID.MatchString(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}
