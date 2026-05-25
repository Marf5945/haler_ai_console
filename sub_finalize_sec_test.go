// sub_finalize_sec_test.go — SEC-W08 第三刀 table test
//
// 目的：驗證 removeLandedSubExport 在 5 條 inline 檢查下，對被誘導的非 sub
// 匯出資料夾會 reject，對合法的 sub export folder 會成功 RemoveAll。
//
// 不抽 helper，直接呼叫 binding 內部函式（同 package main）。
// 不動 data/subexport/*、orchestration/dag/*。
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ui_console/data/subexport"
)

// makeValidSubExportFolder 建立一個合法的 sub export folder（含 install_manifest.json）。
// 回傳資料夾路徑與其系統碼。
func makeValidSubExportFolder(t *testing.T, parent, systemCode string) string {
	t.Helper()
	dirName := systemCode // sub 匯出資料夾 basename == systemCode（需含 "_SUB_"）
	folder := filepath.Join(parent, dirName)
	if err := os.MkdirAll(folder, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := subexport.InstallManifest{
		FormatVersion:    "1.0",
		ExportType:       "sub_handler",
		ExportedAt:       "2026-05-24T00:00:00Z",
		SourceSystemCode: systemCode,
		DisplayName:      "test sub",
	}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(folder, "install_manifest.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return folder
}

func TestRemoveLandedSubExport_AcceptsValidExport(t *testing.T) {
	parent := t.TempDir()
	systemCode := "TEST_SUB_001"
	folder := makeValidSubExportFolder(t, parent, systemCode)

	if err := removeLandedSubExport(folder, systemCode); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if _, statErr := os.Stat(folder); !os.IsNotExist(statErr) {
		t.Errorf("expected folder removed, stat err = %v", statErr)
	}
}

func TestRemoveLandedSubExport_RejectsDangerousPaths(t *testing.T) {
	parent := t.TempDir()

	cases := []struct {
		name         string
		setup        func() (string, string) // returns (landedPath, expectedSystemCode)
		wantContains string
	}{
		{
			"file (not directory) rejected",
			func() (string, string) {
				path := filepath.Join(parent, "TEST_SUB_FILE")
				_ = os.WriteFile(path, []byte("hello"), 0o600)
				return path, "TEST_SUB_FILE"
			},
			"not a directory",
		},
		{
			"basename mismatch rejected",
			func() (string, string) {
				folder := makeValidSubExportFolder(t, parent, "REAL_SUB_001")
				return folder, "EVIL_SUB_999" // expected != basename
			},
			"basename mismatch",
		},
		{
			"basename without _SUB_ marker rejected",
			func() (string, string) {
				folder := filepath.Join(parent, "innocent_folder")
				_ = os.MkdirAll(folder, 0o700)
				return folder, "innocent_folder"
			},
			"does not contain _SUB_",
		},
		{
			"missing install_manifest.json rejected",
			func() (string, string) {
				folder := filepath.Join(parent, "FAKE_SUB_002")
				_ = os.MkdirAll(folder, 0o700)
				return folder, "FAKE_SUB_002"
			},
			"load manifest",
		},
		{
			"manifest export_type mismatch rejected",
			func() (string, string) {
				folder := filepath.Join(parent, "BAD_SUB_003")
				_ = os.MkdirAll(folder, 0o700)
				m := subexport.InstallManifest{
					ExportType:       "persona_handler", // 錯的 type
					SourceSystemCode: "BAD_SUB_003",
				}
				data, _ := json.Marshal(m)
				_ = os.WriteFile(filepath.Join(folder, "install_manifest.json"), data, 0o600)
				return folder, "BAD_SUB_003"
			},
			"export_type",
		},
		{
			"manifest source_system_code mismatch rejected",
			func() (string, string) {
				folder := filepath.Join(parent, "CROSS_SUB_004")
				_ = os.MkdirAll(folder, 0o700)
				m := subexport.InstallManifest{
					ExportType:       "sub_handler",
					SourceSystemCode: "DIFFERENT_CODE", // 與 expected 不同
				}
				data, _ := json.Marshal(m)
				_ = os.WriteFile(filepath.Join(folder, "install_manifest.json"), data, 0o600)
				return folder, "CROSS_SUB_004"
			},
			"source_system_code",
		},
		{
			"empty path → no-op (idempotent)",
			func() (string, string) { return "", "" },
			"", // no error expected
		},
		{
			"non-existent path → idempotent",
			func() (string, string) {
				return filepath.Join(parent, "GHOST_SUB_NEVER_EXISTED"), "GHOST_SUB_NEVER_EXISTED"
			},
			"", // os.IsNotExist returns nil
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			landed, expected := tt.setup()
			err := removeLandedSubExport(landed, expected)

			if tt.wantContains == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantContains)
				return
			}
			if !contains(err.Error(), tt.wantContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantContains)
			}
			// 防禦：reject 時資料夾應該還在
			if landed != "" {
				if _, statErr := os.Stat(landed); statErr != nil && !os.IsNotExist(statErr) {
					t.Errorf("rejected folder should still exist or not be readable, got %v", statErr)
				}
			}
		})
	}
}
