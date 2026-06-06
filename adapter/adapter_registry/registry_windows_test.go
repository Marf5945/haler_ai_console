package adapter_registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExecutableFileAcceptsWindowsCommandLaunchers(t *testing.T) {
	dir := t.TempDir()
	for _, ext := range []string{".exe", ".cmd", ".bat", ".com", ".ps1"} {
		path := filepath.Join(dir, "claude"+ext)
		if err := os.WriteFile(path, []byte("@echo off\r\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", ext, err)
		}
		if !isExecutableFile(path) {
			t.Fatalf("expected %s to be executable on Windows", path)
		}
	}
}

func TestExecutableCandidatesAddsWindowsLauncherExtensions(t *testing.T) {
	base := filepath.Join("C:\\tools", "claude")
	got := executableCandidates(base)
	want := []string{
		base,
		base + ".exe",
		base + ".cmd",
		base + ".bat",
		base + ".com",
		base + ".ps1",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestUnwrapWindowsCommandLauncherReturnsRealExecutable(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "node_modules", "@anthropic-ai", "claude-code", "bin", "claude.exe")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("fake exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(dir, "claude.cmd")
	content := "@ECHO off\r\n\"%dp0%\\node_modules\\@anthropic-ai\\claude-code\\bin\\claude.exe\"   %*\r\n"
	if err := os.WriteFile(launcher, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := unwrapWindowsCommandLauncher(launcher); got != target {
		t.Fatalf("unwrapWindowsCommandLauncher() = %q, want %q", got, target)
	}
}
