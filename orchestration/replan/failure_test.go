package replan

import (
	"errors"
	"testing"
)

func TestClassifyFailure(t *testing.T) {
	cases := []struct {
		name   string
		action string
		output string
		err    error
		want   FailureCategory
	}{
		{"path_guard 系統目錄", "read_file", "", errors.New("path_guard: 拒絕系統目錄內的檔案: /etc/x"), FailureSensitivePath},
		{"路徑穿越", "read_file", "", errors.New("path_guard: 拒絕含 .. 的路徑: ../x"), FailureSensitivePath},
		{"檔案不存在", "read_file", "", errors.New("path_guard: 檔案不存在: stat /a: no such file"), FailurePathNotFound},
		{"not found 英文", "glob", "", errors.New("pattern not found"), FailurePathNotFound},
		{"截斷", "read_file", "", errors.New("size_guard: read /big: too large"), FailureTruncated},
		{"模糊多重", "grep_search", "multiple match candidates", nil, FailureAmbiguous},
		{"無命中-空輸出", "grep_search", "   ", nil, FailureNoResults},
		{"無命中-明確訊號", "grep_search", "no results found", nil, FailureNoResults},
		{"未知 error 預設 ambiguous", "read_file", "", errors.New("weird transient glitch"), FailureAmbiguous},
	}
	for _, c := range cases {
		if got := ClassifyFailure(c.action, c.output, c.err); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestIsReplanTrigger(t *testing.T) {
	if IsReplanTrigger("") {
		t.Errorf("empty category must not trigger")
	}
	if !IsReplanTrigger(FailureNoResults) {
		t.Errorf("no_results must trigger")
	}
}

func TestClassifyResult(t *testing.T) {
	cases := []struct {
		name string
		text string
		want FailureCategory
	}{
		{"真成功(有實質內容)", "已整理出三個重點：A、B、C。", ""},
		{"空字串", "   ", ""},
		{"沒有找到相關文件", "沒有找到相關文件。", FailureNoResults},
		{"查無結果", "查無符合的資料", FailureNoResults},
		{"no results 英文", "Sorry, no results were found.", FailureNoResults},
		{"檔案不存在", "該檔案不存在", FailurePathNotFound},
		{"cannot be accessed", "The file cannot be accessed.", FailurePathNotFound},
		{"outside workspace → sensitive", "cannot be accessed because it is outside the allowed workspace directories", FailureSensitivePath},
		{"短答找不到 → no_results", "找不到", FailureNoResults},
		{"長篇含找不到 → 不判定(長度守門)", "很抱歉，經過一輪詳細搜尋之後我並沒有找到任何明確相關的資料，建議您換個關鍵字、補充更多背景，或縮小範圍再讓我幫您查一次。", ""},
		{"長篇越權 → 仍判 sensitive(安全不受長度守門)", "I'm sorry but the requested file cannot be accessed because it is located outside the allowed workspace directories configured for this session.", FailureSensitivePath},
	}
	for _, c := range cases {
		if got := ClassifyResult(c.text); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}
