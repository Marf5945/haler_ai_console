package main

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func TestImageExtForMIME(t *testing.T) {
	cases := map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/jpg":  ".jpg",
		"image/webp": ".webp",
		"image/gif":  ".gif",
		"image/bmp":  ".bmp",
		"weird/type": ".img",
	}
	for mime, want := range cases {
		if got := imageExtForMIME(mime); got != want {
			t.Errorf("imageExtForMIME(%q)=%q want %q", mime, got, want)
		}
	}
}

func TestCliAdapterSupportsImagePath(t *testing.T) {
	for _, id := range []string{"claude-cli", "gemini-cli", "CLAUDE-CLI"} {
		if !cliAdapterSupportsImagePath(id) {
			t.Errorf("%s should support path image", id)
		}
	}
	for _, id := range []string{"codex-cli", "ollama-cli", "", "aider"} {
		if cliAdapterSupportsImagePath(id) {
			t.Errorf("%s should NOT support path image", id)
		}
	}
}

func TestCliImagePromptSuffix(t *testing.T) {
	paths := []string{"/tmp/a/img-01.png", "/tmp/a/img-02.jpg"}

	claude, ok := cliImagePromptSuffix("claude-cli", paths)
	if !ok || !strings.Contains(claude, "/tmp/a/img-01.png") || !strings.Contains(claude, "/tmp/a/img-02.jpg") {
		t.Errorf("claude suffix wrong: %q (ok=%v)", claude, ok)
	}
	if strings.Contains(claude, "@/tmp") {
		t.Errorf("claude should not use @ syntax: %q", claude)
	}

	gem, ok := cliImagePromptSuffix("gemini-cli", paths)
	if !ok || !strings.Contains(gem, "@/tmp/a/img-01.png") || !strings.Contains(gem, "@/tmp/a/img-02.jpg") {
		t.Errorf("gemini suffix wrong: %q (ok=%v)", gem, ok)
	}

	if _, ok := cliImagePromptSuffix("codex-cli", paths); ok {
		t.Error("codex should not be supported")
	}
	if _, ok := cliImagePromptSuffix("claude-cli", nil); ok {
		t.Error("empty paths should return ok=false")
	}
}

func TestStageCLIImagesIfSupported_DirectAndCleanup(t *testing.T) {
	imgs := []composerImage{
		{MIME: "image/png", DataB64: b64("fake-png-bytes")},
		{MIME: "image/jpeg", DataB64: b64("fake-jpg-bytes")},
	}
	suffix, cleanup, ok := stageCLIImagesIfSupported("claude-cli", "sess-123", imgs)
	if !ok {
		t.Fatal("expected ok for claude-cli")
	}
	// 後綴內應有兩個落地路徑，且檔案真的存在。
	lines := []string{}
	for _, ln := range strings.Split(suffix, "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, os.TempDir()) || strings.Contains(ln, cliVisionTempRoot) {
			lines = append(lines, ln)
		}
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 staged paths in suffix, got %d: %q", len(lines), suffix)
	}
	for _, p := range lines {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("staged file missing: %s (%v)", p, err)
		}
	}
	// cleanup 後檔案應消失。
	cleanup()
	for _, p := range lines {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("file should be removed after cleanup: %s", p)
		}
	}
}

func TestStageCLIImagesIfSupported_UnsupportedAdapter(t *testing.T) {
	imgs := []composerImage{{MIME: "image/png", DataB64: b64("x")}}
	_, cleanup, ok := stageCLIImagesIfSupported("codex-cli", "s", imgs)
	if ok {
		t.Error("codex-cli should not stage (returns ok=false)")
	}
	if cleanup == nil {
		t.Error("cleanup must be non-nil even when ok=false")
	}
	cleanup() // 應為 no-op，不 panic
}

func TestStageCLIImagesIfSupported_BadBase64(t *testing.T) {
	imgs := []composerImage{{MIME: "image/png", DataB64: "!!!not-base64!!!"}}
	_, _, ok := stageCLIImagesIfSupported("claude-cli", "s", imgs)
	if ok {
		t.Error("invalid base64 should fail staging")
	}
}

func TestSanitizeSessionDir(t *testing.T) {
	if got := sanitizeSessionDir("a/b/../c"); strings.ContainsAny(got, "/.") {
		t.Errorf("path separators should be sanitized: %q", got)
	}
	if got := sanitizeSessionDir(""); got != "sess" {
		t.Errorf("empty -> %q want sess", got)
	}
	long := strings.Repeat("x", 200)
	if got := sanitizeSessionDir(long); len(got) > 64 {
		t.Errorf("should cap length, got %d", len(got))
	}
}
