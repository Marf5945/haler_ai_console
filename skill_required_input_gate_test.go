package main

import "testing"

func TestMissingRequiredInputs(t *testing.T) {
	cases := []struct {
		name     string
		required []string
		userText string
		refs     []string
		want     []string
	}{
		{
			name:     "缺機台時回報缺（trace 重現案例）",
			required: []string{"機台", "電料BOM", "編碼紀錄"},
			userText: "幫我使用skill 產出電料Bom，資料庫為電料編碼紀錄，範例為電料BOM",
			refs:     []string{"電料BOM-260327M1.xlsx", "電料編碼紀錄_260428_M1.xlsx"},
			want:     []string{"機台"},
		},
		{
			name:     "機台已於訊息提供時不缺",
			required: []string{"機台", "電料BOM", "編碼紀錄"},
			userText: "用產出電料Bom，機台 M1",
			refs:     []string{"電料BOM-260327M1.xlsx", "電料編碼紀錄_260428_M1.xlsx"},
			want:     nil,
		},
		{
			name:     "缺檔案型輸入時回報缺（保持原順序）",
			required: []string{"機台", "電料BOM", "編碼紀錄"},
			userText: "機台 M1",
			refs:     nil,
			want:     []string{"電料BOM", "編碼紀錄"},
		},
		{
			name:     "預設佔位 input 不攔截",
			required: []string{"input"},
			userText: "隨便跑一下",
			refs:     nil,
			want:     nil,
		},
		{
			name:     "純值型 temperature 已提供時不缺",
			required: []string{"temperature"},
			userText: "現在 temperature 15 度",
			refs:     nil,
			want:     nil,
		},
		{
			name:     "去重與空白欄位處理",
			required: []string{"機台", " 機台 ", ""},
			userText: "沒有給",
			refs:     nil,
			want:     []string{"機台"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := missingRequiredInputs(tc.required, tc.userText, tc.refs)
			if !equalGateStringSlices(got, tc.want) {
				t.Fatalf("missingRequiredInputs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSkillMissingInputQuestion(t *testing.T) {
	got := skillMissingInputQuestion("產出電料Bom", []string{"機台"})
	if got == "" {
		t.Fatal("question should not be empty")
	}
	for _, sub := range []string{"產出電料Bom", "機台"} {
		if !containsGate(got, sub) {
			t.Fatalf("question %q should contain %q", got, sub)
		}
	}
}

func equalGateStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsGate(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOfGate(s, sub) >= 0)
}

func indexOfGate(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
