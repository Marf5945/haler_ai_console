package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportReferenceFileToDir_RejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "folder-ref")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "nested.txt"), []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}

	referenceDir := filepath.Join(dir, "references")
	_, err := importReferenceFileToDir(sourceDir, referenceDir)
	if err == nil || !strings.Contains(err.Error(), "folders are not supported") {
		t.Fatalf("expected folder rejection, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(referenceDir, filepath.Base(sourceDir))); !os.IsNotExist(statErr) {
		t.Fatalf("folder import should not leave an empty reference file, stat err=%v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(sourceDir, "nested.txt")); statErr != nil {
		t.Fatalf("source directory content should remain intact: %v", statErr)
	}
}

func TestImportReferenceFileToDir_CopiesFile(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(sourcePath, []byte("hello reference"), 0o600); err != nil {
		t.Fatal(err)
	}

	referenceDir := filepath.Join(dir, "references")
	ref, err := importReferenceFileToDir(sourcePath, referenceDir)
	if err != nil {
		t.Fatalf("expected import success, got %v", err)
	}
	if ref.Name != "note.txt" || ref.Source != "library" || ref.Status != "ready" {
		t.Fatalf("unexpected reference metadata: %+v", ref)
	}
	data, err := os.ReadFile(filepath.Join(referenceDir, "note.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello reference" {
		t.Fatalf("unexpected copied content: %q", string(data))
	}
}
