package skill_step

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfirmArchiveRejectsSymlinkResource(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	if err := os.MkdirAll(filepath.Join(src, "programs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("# test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(src, "programs", "outside.txt")
	if err := os.Symlink(filepath.Join(root, "missing.txt"), link); err != nil {
		t.Skipf("symlink unavailable on this platform: %v", err)
	}

	svc := NewArchiveService(filepath.Join(root, "appdata"))
	preview, err := svc.ScanFolder(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmArchive(preview); err == nil || !strings.Contains(err.Error(), "symlink is not allowed") {
		t.Fatalf("ConfirmArchive should reject symlink resources, got %v", err)
	}
}

func TestConfirmArchiveNormalizesEmbeddedAutoExecute(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	embedded := &SkillManifest{
		SchemaVersion: SchemaManifestV2,
		SkillID:       "dangerous.imported",
		DisplayName:   "Dangerous Imported",
		Version:       "1.0.0",
		Tags:          SkillTags{RiskTag: []string{"medium"}},
		Permissions:   SkillPermissions{Network: "external", Filesystem: "workspace_write", Execution: "controlled"},
		Lifecycle: &Lifecycle{
			Status:           LifecycleEnabled,
			VisibleInToolbar: true,
			RouteAsCandidate: true,
			AutoExecute:      true,
			UserConfirmed:    true,
		},
	}
	if err := SaveManifest(src, embedded); err != nil {
		t.Fatal(err)
	}

	svc := NewArchiveService(filepath.Join(root, "appdata"))
	preview, err := svc.ScanFolder(src)
	if err != nil {
		t.Fatal(err)
	}
	archived, err := svc.ConfirmArchive(preview)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Lifecycle == nil {
		t.Fatal("expected lifecycle")
	}
	if archived.Lifecycle.AutoExecute {
		t.Fatalf("embedded auto_execute should be recalculated from permissions, got %+v", archived.Lifecycle)
	}
	if archived.Lifecycle.Status != LifecycleEnabled || !archived.Lifecycle.UserConfirmed {
		t.Fatalf("normalization should keep lifecycle identity fields, got %+v", archived.Lifecycle)
	}
}
