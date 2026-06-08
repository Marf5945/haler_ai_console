package cli_manager

import (
	"testing"

	"ui_console/data/conversation"
	"ui_console/shared/eventbus"
)

func TestSanitizeAdapterWorkspaceName(t *testing.T) {
	tests := map[string]string{
		"gemini-cli":     "gemini-cli",
		" Codex CLI ":    "codex-cli",
		"../../bad/path": "bad-path",
		"":               "default",
		"!!!":            "default",
	}

	for input, want := range tests {
		if got := sanitizeAdapterWorkspaceName(input); got != want {
			t.Fatalf("sanitizeAdapterWorkspaceName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSummarizationTriggerUsesContinuityCounter(t *testing.T) {
	adapter := NewSidecarCLIAdapter(nil, "")
	adapter.SetEventBus(eventbus.New(nil))
	state := adapter.getContinuity("chat-session")
	state.counter.Add(conversation.SummarizationThreshold)

	adapter.checkAndEmitSummarizationNeeded("chat-session", state)

	if !adapter.summaryTriggered {
		t.Fatal("expected session continuity counter to trigger summarization banner")
	}
	if adapter.summaryContinuityKey != "chat-session" {
		t.Fatalf("summaryContinuityKey = %q, want chat-session", adapter.summaryContinuityKey)
	}
}

func TestSummarizationTriggerBelowThreshold(t *testing.T) {
	adapter := NewSidecarCLIAdapter(nil, "")
	adapter.SetEventBus(eventbus.New(nil))
	state := adapter.getContinuity("chat-session")
	state.counter.Add(conversation.SummarizationThreshold - 1)

	adapter.checkAndEmitSummarizationNeeded("chat-session", state)

	if adapter.summaryTriggered {
		t.Fatal("summarization should not trigger below threshold")
	}
}
