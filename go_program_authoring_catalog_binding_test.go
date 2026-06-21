package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeGoProgramPreviewPathRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	if _, err := safeGoProgramPreviewPath(root, "main.go"); err != nil {
		t.Fatalf("safe path rejected valid file: %v", err)
	}
	bad := []string{"", ".", "../main.go", filepath.Join("..", "main.go"), filepath.Join(root, "main.go")}
	for _, rel := range bad {
		if _, err := safeGoProgramPreviewPath(root, rel); err == nil {
			t.Fatalf("safe path accepted %q", rel)
		}
	}
}

func TestReadGoProgramPreviewFileReadsGoSourceOnlyInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := readGoProgramPreviewFile(root, "main.go")
	if got.Error != "" {
		t.Fatalf("read preview returned error: %s", got.Error)
	}
	if got.Path != "main.go" || got.Language != "go" || got.Content == "" {
		t.Fatalf("unexpected preview: %#v", got)
	}
	escaped := readGoProgramPreviewFile(root, "../main.go")
	if escaped.Error == "" {
		t.Fatalf("escaped preview path should be rejected")
	}
}
