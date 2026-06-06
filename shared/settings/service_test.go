package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceSavesPersona(t *testing.T) {
	root := t.TempDir()
	service := NewService(root)

	state := service.SavePersona(Persona{
		ID:            "persona-a",
		Name:          "人格 A",
		Icon:          "♙",
		Identity:      "可靠助手",
		ReplyStrategy: "先確認再執行",
	})
	if state.ActivePersonaID != "persona-a" {
		t.Fatalf("active persona was not updated: %s", state.ActivePersonaID)
	}
	if state.Personas[0].Identity != "可靠助手" {
		t.Fatalf("persona was not saved: %#v", state.Personas[0])
	}
	if state.Personas[0].Name != reservedPersonaName {
		t.Fatalf("reserved persona name should stay locked, got %s", state.Personas[0].Name)
	}

	reloaded := NewService(root).State()
	if reloaded.Personas[0].ReplyStrategy != "先確認再執行" {
		t.Fatalf("persona was not persisted: %#v", reloaded.Personas[0])
	}
}

// Panel 設定已移至 UISettingsService，測試見 ui_settings_test.go。
func TestUISettingsServiceSavesPanel(t *testing.T) {
	root := t.TempDir()
	svc := NewUISettingsService(root)

	diffJSON := `{"panel_language":"英文","role_language":"繁中","font_preset":"等寬","font_scale":"110%","panel_style":"緊湊"}`
	result, err := svc.ApplyStyleDiff(diffJSON)
	if err != nil {
		t.Fatalf("ApplyStyleDiff failed: %v", err)
	}
	if result.PanelLanguage != "英文" || result.PanelStyle != "緊湊" {
		t.Fatalf("panel settings were not applied: %#v", result)
	}

	reloaded := NewUISettingsService(root).Get()
	if reloaded.FontPreset != "等寬" {
		t.Fatalf("panel settings were not persisted: %#v", reloaded)
	}
}

func TestServiceDefaultsReserveFourthSlotForAddPersona(t *testing.T) {
	service := NewService(t.TempDir())
	state := service.State()
	if len(state.Personas) != 3 {
		t.Fatalf("default personas should leave the fourth UI slot for add/install, got %d", len(state.Personas))
	}
	if state.Personas[2].ID != "persona-c" {
		t.Fatalf("unexpected third default persona: %#v", state.Personas[2])
	}
	if state.Personas[1].Name != "厭世大叔" || state.Personas[2].Name != "秘書小妹" {
		t.Fatalf("built-in persona defaults were not applied: %#v", state.Personas)
	}
}

func TestServiceDoesNotSaveMoreThanSixteenPersonas(t *testing.T) {
	service := NewService(t.TempDir())
	var state State
	for index := 0; index < 20; index++ {
		state = service.SavePersona(Persona{
			ID:   "persona-extra-" + string(rune('a'+index)),
			Name: "新增人格",
		})
	}
	if len(state.Personas) != MaxPersonas {
		t.Fatalf("personas should be capped at %d, got %d", MaxPersonas, len(state.Personas))
	}
}

func TestServiceReordersPersonas(t *testing.T) {
	root := t.TempDir()
	service := NewService(root)

	state := service.ReorderPersonas([]string{"persona-c", "persona-a", "persona-b"})
	if got := state.Personas[0].ID; got != "persona-c" {
		t.Fatalf("first persona should be persona-c, got %s", got)
	}
	if got := state.Personas[1].ID; got != "persona-a" {
		t.Fatalf("second persona should be persona-a, got %s", got)
	}
	if got := state.ActivePersonaID; got != "persona-c" {
		t.Fatalf("active persona should follow first persona after reorder, got %s", got)
	}

	reloaded := NewService(root).State()
	if got := reloaded.Personas[0].ID; got != "persona-c" {
		t.Fatalf("reordered personas were not persisted, got first %s", got)
	}
	if got := reloaded.ActivePersonaID; got != "persona-c" {
		t.Fatalf("active persona reorder was not persisted, got %s", got)
	}
}

func TestServiceExportsAndRemovesPersona(t *testing.T) {
	root := t.TempDir()
	service := NewService(root)
	service.SavePersona(Persona{ID: "persona-extra", Name: "外部角色", Identity: "可移動人格"})

	dest := t.TempDir()
	exportPath, err := service.ExportPersona("persona-extra", dest)
	if err != nil {
		t.Fatalf("ExportPersona failed: %v", err)
	}
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("export file was not written: %v", err)
	}
	if !strings.Contains(string(data), `"schema": "ai-console.persona.v1"`) {
		t.Fatalf("export payload missing schema: %s", string(data))
	}

	assetDir := storagePersonaRootForTest(root, "persona-extra")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}
	state, err := service.RemovePersona("persona-extra")
	if err != nil {
		t.Fatalf("RemovePersona failed: %v", err)
	}
	for _, persona := range state.Personas {
		if persona.ID == "persona-extra" {
			t.Fatalf("persona was not removed: %#v", state.Personas)
		}
	}
	if _, err := os.Stat(assetDir); !os.IsNotExist(err) {
		t.Fatalf("persona asset dir should be removed, err=%v", err)
	}
}

func storagePersonaRootForTest(root, personaID string) string {
	return filepath.Join(root, "data", "personas", personaID)
}

func TestRemoveLegacyDefaultPersonaD(t *testing.T) {
	personas := removeLegacyDefaultPersonaD([]Persona{
		{ID: "persona-a", Name: reservedPersonaName, Icon: "♙"},
		{ID: "persona-d", Name: "人格 D", Icon: "◇"},
		{ID: "persona-d-custom", Name: "人格 D", Icon: "◇", Identity: "保留"},
	})
	if len(personas) != 2 {
		t.Fatalf("legacy default persona-d should be removed, got %#v", personas)
	}
	if personas[1].ID != "persona-d-custom" {
		t.Fatalf("custom persona should be retained: %#v", personas)
	}
}
