package visual_learning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingRuntimeFiles(t *testing.T) {
	dir := t.TempDir()
	required := []string{"onnxruntime.dll", "DirectML.dll"}

	if err := os.WriteFile(filepath.Join(dir, "onnxruntime.dll"), []byte("stub"), 0644); err != nil {
		t.Fatal(err)
	}

	missing := missingRuntimeFiles(dir, required)
	if len(missing) != 1 || missing[0] != "DirectML.dll" {
		t.Fatalf("missingRuntimeFiles() = %#v, want DirectML.dll only", missing)
	}
}

func TestDirectMLRuntimeLockConstants(t *testing.T) {
	if DirectMLRuntimeName != "onnxruntime-directml" {
		t.Fatalf("DirectMLRuntimeName = %q", DirectMLRuntimeName)
	}
	if DirectMLRuntimeVersion == "" {
		t.Fatal("DirectMLRuntimeVersion is empty")
	}
	if DirectMLRuntimeRID != "win-x64" {
		t.Fatalf("DirectMLRuntimeRID = %q", DirectMLRuntimeRID)
	}
}

func TestVerifyDirectMLRuntimeHashesAcceptsPinnedDLLs(t *testing.T) {
	dir := t.TempDir()
	files := []string{"onnxruntime.dll", "DirectML.dll"}
	pins := map[string]string{}
	for _, file := range files {
		data := []byte("runtime-" + file)
		if err := os.WriteFile(filepath.Join(dir, file), data, 0644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		pins[file] = "sha256:" + hex.EncodeToString(sum[:])
	}

	manifest, err := json.Marshal(DirectMLRuntimeHashManifest{
		Runtime: DirectMLRuntimeName,
		Version: DirectMLRuntimeVersion,
		RID:     DirectMLRuntimeRID,
		Files:   pins,
	})
	if err != nil {
		t.Fatal(err)
	}

	if errs := verifyDirectMLRuntimeHashesWithManifest(dir, files, manifest); len(errs) != 0 {
		t.Fatalf("verifyDirectMLRuntimeHashesWithManifest() = %#v, want no errors", errs)
	}
}

func TestVerifyDirectMLRuntimeHashesRejectsUnpinnedDLLs(t *testing.T) {
	dir := t.TempDir()
	files := []string{"onnxruntime.dll", "DirectML.dll"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("runtime"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	manifest, err := json.Marshal(DirectMLRuntimeHashManifest{
		Runtime: DirectMLRuntimeName,
		Version: DirectMLRuntimeVersion,
		RID:     DirectMLRuntimeRID,
		Files:   map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}

	errs := verifyDirectMLRuntimeHashesWithManifest(dir, files, manifest)
	if len(errs) == 0 {
		t.Fatal("verifyDirectMLRuntimeHashesWithManifest() returned no errors for unpinned DLLs")
	}
	if !strings.Contains(strings.Join(errs, "\n"), "not pinned") {
		t.Fatalf("errors = %#v, want not pinned", errs)
	}
}

func TestVerifyDirectMLRuntimeHashesRejectsMismatchedDLL(t *testing.T) {
	dir := t.TempDir()
	files := []string{"onnxruntime.dll"}
	if err := os.WriteFile(filepath.Join(dir, "onnxruntime.dll"), []byte("actual"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(DirectMLRuntimeHashManifest{
		Runtime: DirectMLRuntimeName,
		Version: DirectMLRuntimeVersion,
		RID:     DirectMLRuntimeRID,
		Files: map[string]string{
			"onnxruntime.dll": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	errs := verifyDirectMLRuntimeHashesWithManifest(dir, files, manifest)
	if len(errs) == 0 || !strings.Contains(strings.Join(errs, "\n"), "hash mismatch") {
		t.Fatalf("errors = %#v, want hash mismatch", errs)
	}
}
