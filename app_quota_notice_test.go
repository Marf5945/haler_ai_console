package main

import (
	"errors"
	"strings"
	"testing"

	"ui_console/orchestration/skill_step"
)

func TestQuotaSwitchModelNotice(t *testing.T) {
	// 這次 trace 的實際字串。
	_, hit := quotaSwitchModelNotice("gemini-cli", nil,
		errors.New("Attempt 10 failed: You have exhausted your capacity on this model.. Max attempts reached"))
	if !hit {
		t.Fatal("should detect 'exhausted your capacity'")
	}

	// resp.Error 路徑 + 429。
	notice, hit := quotaSwitchModelNotice("claude-cli",
		&skill_step.CLIResponse{Error: "RetryableQuotaError: status 429"}, nil)
	if !hit {
		t.Fatal("should detect status 429 in resp.Error")
	}
	if !strings.Contains(notice, "claude-cli") || !strings.Contains(notice, "切換") {
		t.Fatalf("notice should name adapter and tell user to switch: %q", notice)
	}

	// 一般成功回應不可誤判。
	if _, hit := quotaSwitchModelNotice("gemini-cli",
		&skill_step.CLIResponse{Text: "本機搜尋「電料表」有找到 2 筆"}, nil); hit {
		t.Fatal("normal result must not be flagged as quota error")
	}
}
