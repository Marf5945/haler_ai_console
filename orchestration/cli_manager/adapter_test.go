package cli_manager

import "testing"

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
