package main

import (
	"strings"
	"testing"
)

func TestExpandReplyStrategyPrompt(t *testing.T) {
	got := expandReplyStrategyPrompt("concise")
	if !strings.Contains(got, "節省模式") || !strings.Contains(got, "極短") {
		t.Fatalf("concise strategy was not expanded: %q", got)
	}

	got = expandReplyStrategyPrompt("反問模式")
	if !strings.Contains(got, "1到3句反問") {
		t.Fatalf("Chinese strategy label was not expanded: %q", got)
	}

	custom := "先確認再執行"
	if got := expandReplyStrategyPrompt(custom); got != custom {
		t.Fatalf("custom strategy should be preserved, got %q", got)
	}
}
