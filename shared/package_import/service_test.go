package package_import

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareInstallAcceptsSinglePersonaJSON(t *testing.T) {
	srcDir := t.TempDir()
	sourcePath := filepath.Join(srcDir, "可愛星_persona-1778256736096.json")
	payload := `{
  "schema": "ai-console.persona.v1",
  "id": "persona-1778256736096",
  "name": "可愛星",
  "icon": "＋",
  "avatarUrl": "",
  "identity": "你好",
  "replyStrategy": "",
  "roleStrength": "50%",
  "personality": "",
  "scenario": "",
  "description": "真的很可愛\n很可愛\n很可愛\n很可愛"
}`
	if err := os.WriteFile(sourcePath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write source persona: %v", err)
	}

	service := NewService(t.TempDir())
	pending, err := service.PrepareInstall(sourcePath, PackageManifest{
		Name:        "可愛星",
		Version:     "0.0.1",
		PackageType: "persona",
		SourcePath:  sourcePath,
		RiskTag:     "unknown",
	})
	if err != nil {
		t.Fatalf("PrepareInstall failed: %v", err)
	}
	if pending == nil || pending.Persona.ID != "persona-1778256736096" || pending.Persona.Name != "可愛星" {
		t.Fatalf("unexpected pending persona: %#v", pending)
	}
	if pending.Status != StatusQuarantined {
		t.Fatalf("persona import should be quarantined, got %q", pending.Status)
	}
	if _, err := os.Stat(filepath.Join(pending.QuarantinePath, "payload", filepath.Base(sourcePath))); err != nil {
		t.Fatalf("persona json was not copied into quarantine: %v", err)
	}
}
