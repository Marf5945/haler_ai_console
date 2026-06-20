package adapter_registry

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// mkexec 在指定路徑建立一個具執行權限的檔案（測試用假 CLI）。
func mkexec(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec-bit / symlink semantics differ on Windows")
	}
}

// 貼到「上層」資料夾，仍能往下找到 node_modules/.bin 內的 CLI。
func TestResolveExecutablePath_DownFromParent(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	want := filepath.Join(root, "proj", "node_modules", ".bin", "gemini")
	mkexec(t, want)

	got, err := resolveExecutablePath(root, "gemini")
	if err != nil {
		t.Fatalf("resolveExecutablePath(%q) error: %v", root, err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// 貼到「下層」過深的資料夾，仍能往上找到 bin 內的 CLI。
func TestResolveExecutablePath_UpFromChild(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	want := filepath.Join(root, "proj", "bin", "gemini")
	mkexec(t, want)
	// 使用者貼到比執行檔更深一層的資料夾。
	deep := filepath.Join(root, "proj", "node_modules")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}

	got, err := resolveExecutablePath(deep, "gemini")
	if err != nil {
		t.Fatalf("resolveExecutablePath(%q) error: %v", deep, err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// 執行檔被改名、但底層套件仍是已知 CLI（symlink 指向 gemini-cli 套件）。
func TestResolveExecutablePath_RenamedKnownCLI(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	pkgEntry := filepath.Join(root, "proj", "node_modules", "@google", "gemini-cli", "index.js")
	mkexec(t, pkgEntry)

	binDir := filepath.Join(root, "proj", "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	link := filepath.Join(binDir, "mycli") // 改名後的執行檔
	if err := os.Symlink(pkgEntry, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// 名稱留空、且 .bin 下沒有叫 gemini 的檔，必須靠「改名後備」才能找到。
	got, err := resolveExecutablePath(filepath.Join(root, "proj"), "")
	if err != nil {
		t.Fatalf("resolveExecutablePath error: %v", err)
	}
	if got != link {
		t.Fatalf("got %q, want %q", got, link)
	}
}

// bin 內只有單一可執行檔（完全自訂名稱）時，也不靠資料夾亂猜。
func TestResolveExecutablePath_UniqueExecutableFallbackRejected(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	mkexec(t, filepath.Join(root, "proj", "bin", "totally-custom-name"))

	if _, err := resolveExecutablePath(filepath.Join(root, "proj"), ""); err == nil {
		t.Fatalf("expected error for unknown executable in folder, got nil")
	}
}

// 兩個以上候選執行檔時不亂猜，回傳錯誤交由使用者明確指定。
func TestResolveExecutablePath_AmbiguousReturnsError(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	mkexec(t, filepath.Join(root, "proj", "bin", "alpha"))
	mkexec(t, filepath.Join(root, "proj", "bin", "beta"))

	if _, err := resolveExecutablePath(filepath.Join(root, "proj"), ""); err == nil {
		t.Fatalf("expected error for ambiguous executables, got nil")
	}
}

// 直接指向可執行檔：原樣採用。
func TestResolveExecutablePath_DirectExecutable(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	want := filepath.Join(root, "renamed-binary")
	mkexec(t, want)

	got, err := resolveExecutablePath(want, "")
	if err != nil {
		t.Fatalf("resolveExecutablePath error: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveExecutablePath_RejectsHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("home directory unavailable")
	}
	if _, err := resolveExecutablePath(home, ""); err == nil {
		t.Fatalf("expected broad home directory to be rejected")
	}
}

func TestResolveCustomCLIRejectsKnownUnsupportedChatCLIs(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	for _, name := range []string{"ollama", "yc", "hf", "huggingface-cli", "groq"} {
		path := filepath.Join(root, name)
		mkexec(t, path)
		got, err := ResolveCustomCLI("", path)
		if err == nil {
			t.Fatalf("ResolveCustomCLI(%q) expected unsupported error", name)
		}
		if got.Supported {
			t.Fatalf("ResolveCustomCLI(%q) Supported = true, want false", name)
		}
		if got.Reason == "" {
			t.Fatalf("ResolveCustomCLI(%q) reason is empty", name)
		}
	}
}
