package main

import (
	"regexp"
	"testing"
)

// SEC-14 驗證：validSubID regex 拒絕路徑穿越字元。
// 注意：此測試驗證 regex 邏輯，不依賴 App 初始化。
func TestValidSubIDRegex(t *testing.T) {
	validSubID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	tests := []struct {
		name  string
		subID string
		valid bool
	}{
		// 合法 subID
		{"正常 system code", "sub-1716000000", true},
		{"純英數", "mysub123", true},
		{"含底線", "my_sub_v2", true},
		{"含連字號", "my-sub-v2", true},
		{"單字元", "a", true},

		// 非法 subID（路徑穿越）
		{"路徑穿越 ../", "../../etc/passwd", false},
		{"路徑穿越 ./", "./hidden", false},
		{"含斜線", "sub/evil", false},
		{"含反斜線", "sub\\evil", false},
		{"含空格", "sub evil", false},
		{"含冒號", "sub:evil", false},
		{"含 null byte", "sub\x00evil", false},
		{"空字串", "", false},
		{"只有 dots", "..", false},
		{"含點", "sub.evil", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validSubID.MatchString(tt.subID)
			if got != tt.valid {
				t.Errorf("validSubID.MatchString(%q) = %v, want %v", tt.subID, got, tt.valid)
			}
		})
	}
}
