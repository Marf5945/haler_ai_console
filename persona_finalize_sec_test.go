// persona_finalize_sec_test.go — SEC-W02 第三刀 table test
//
// 目的：驗證 FinalizeNativePersonaExport 的 "cancel" 分支在 6 條 inline 檢查下，
// 對被誘導的非 persona 路徑會 reject、對合法 persona 匯出檔會成功刪除。
//
// 不抽 helper，table-driven 直接呼叫 binding method。
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ui_console/shared/settings"
)

// makeValidPersonaExport 在 tempDir 寫一個合法的 persona 匯出 JSON。
func makeValidPersonaExport(t *testing.T, dir, personaID string) string {
	t.Helper()
	path := filepath.Join(dir, "persona_"+personaID+".json")
	payload := map[string]interface{}{
		"schema":      "ai-console.persona.v1",
		"id":          personaID,
		"name":        "test persona",
		"description": "for SEC-W02 test",
	}
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFinalizeNativePersonaExportCancel_RejectsDangerousPaths(t *testing.T) {
	dir := t.TempDir()
	validPath := makeValidPersonaExport(t, dir, "persona-001")

	// 在 dir 下另準備：目錄、錯副檔名、錯 schema、錯 personaID 的檔案
	subdir := filepath.Join(dir, "looks_like_export_dir")
	if err := os.MkdirAll(subdir, 0o700); err != nil {
		t.Fatal(err)
	}

	wrongExt := filepath.Join(dir, "persona_001.txt")
	_ = os.WriteFile(wrongExt, []byte(`{"schema":"ai-console.persona.v1","id":"persona-001"}`), 0o600)

	wrongSchema := filepath.Join(dir, "wrong_schema.json")
	_ = os.WriteFile(wrongSchema, []byte(`{"schema":"foo.bar.v1","id":"persona-001"}`), 0o600)

	wrongID := filepath.Join(dir, "wrong_id.json")
	_ = os.WriteFile(wrongID, []byte(`{"schema":"ai-console.persona.v1","id":"persona-EVIL"}`), 0o600)

	notJSON := filepath.Join(dir, "not_json.json")
	_ = os.WriteFile(notJSON, []byte(`<html>not json</html>`), 0o600)

	// settingsService 必須非 nil — FinalizeNativePersonaExport line 101
	// 在進 switch 前無條件呼叫 a.settingsService.State()。用 settings.NewService
	// 既有 constructor 傳一個 tempDir，不抽 mock / 不改生產函式。
	app := &App{settingsService: settings.NewService(t.TempDir())}

	cases := []struct {
		name           string
		landedPath     string
		tempExportPath string
		personaID      string
		wantErr        bool
		wantContains   string
	}{
		{"valid path → should remove",
			validPath, validPath, "persona-001", false, ""},
		{"directory rejected",
			subdir, validPath, "persona-001", true, "is a directory"},
		{"wrong extension rejected",
			wrongExt, validPath, "persona-001", true, "extension must be .json"},
		{"basename mismatch rejected",
			wrongSchema, validPath, "persona-001", true, "basename mismatch"},
		{"non-existent path",
			filepath.Join(dir, "nonexistent.json"), "", "persona-001", true, "stat landed"},
		{"system path /etc/passwd",
			"/etc/passwd", "", "persona-001", true, ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.FinalizeNativePersonaExport("cancel", tt.personaID, tt.tempExportPath, tt.landedPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.wantContains != "" && !contains(err.Error(), tt.wantContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantContains)
				}
			} else {
				if err != nil {
					t.Errorf("expected success, got %v", err)
				}
				if _, statErr := os.Stat(tt.landedPath); !os.IsNotExist(statErr) {
					t.Errorf("expected landed path to be removed; still exists or other error: %v", statErr)
				}
			}
		})
	}

	// 額外 case：basename 匹配但 schema 錯 — 用同名同副檔名但內容錯的方式構造
	t.Run("schema mismatch rejected", func(t *testing.T) {
		landed := filepath.Join(dir, "matching_basename.json")
		_ = os.WriteFile(landed, []byte(`{"schema":"foo.bar.v1","id":"persona-001"}`), 0o600)
		_, err := app.FinalizeNativePersonaExport("cancel", "persona-001", landed, landed)
		if err == nil || !contains(err.Error(), "schema mismatch") {
			t.Errorf("expected schema mismatch error, got %v", err)
		}
	})

	t.Run("personaID mismatch rejected", func(t *testing.T) {
		landed := filepath.Join(dir, "matching_basename_2.json")
		_ = os.WriteFile(landed, []byte(`{"schema":"ai-console.persona.v1","id":"persona-EVIL"}`), 0o600)
		_, err := app.FinalizeNativePersonaExport("cancel", "persona-001", landed, landed)
		if err == nil || !contains(err.Error(), "personaID mismatch") {
			t.Errorf("expected personaID mismatch error, got %v", err)
		}
	})

	t.Run("malformed JSON rejected", func(t *testing.T) {
		landed := filepath.Join(dir, "malformed.json")
		_ = os.WriteFile(landed, []byte(`<html>not json</html>`), 0o600)
		_, err := app.FinalizeNativePersonaExport("cancel", "persona-001", landed, landed)
		if err == nil || !contains(err.Error(), "parse landed JSON") {
			t.Errorf("expected parse error, got %v", err)
		}
	})
}

// contains 是輕量 substring 檢查（避免 import strings 影響可讀性）。
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
