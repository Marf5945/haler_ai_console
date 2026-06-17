// conversation_root_test.go — SEC-W03 agentID 路徑驗證。
//
// conversationRootForAgent 內部有 white-list 早退處理空字串 / "main" /
// "主haㄌer" 三個合法 ID；本測試只驗單一目錄名安全規則本身，
// 不觸碰檔案系統，與 sub_export_sec_test.go (SEC-14) 同風格。
//
// 執行：go test -run TestSafeConversationAgentID -v
package main

import "testing"

func TestSafeConversationAgentID(t *testing.T) {
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
		{"imported chinese display system code", "排程：H-I高風險確認smoke_SUB_KGR2PC3MTLCQ", true},
		{"display name with spaces", "排程 測試_SUB_KGR2PC3MTLCQ", true},

		// ── 路徑穿越 / 控制字元（核心 SEC-W03 攻擊面，必須 reject）──
		{"dotdot", "..", false},
		{"single dot", ".", false},
		{"forward slash with dotdot", "../etc", false},
		{"backslash with dotdot", `..\etc`, false},
		{"absolute path", "/etc/passwd", false},
		{"four dots traversal", "....", false}, // Windows 8.3 short-name 變種
		{"ascii colon", "C:tmp", false},
		{"null byte", "abc\x00def", false},
		{"newline injection", "abc\ndef", false},
		{"trailing slash", "abc/", false},

		// ── 空字串（white-list 早退處理，regex 仍應拒絕）──
		{"empty", "", false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := isSafeConversationAgentID(tt.id)
			if got != tt.valid {
				t.Errorf("isSafeConversationAgentID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}
