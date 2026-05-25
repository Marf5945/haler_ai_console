package subexport

import (
	"os"
	"path/filepath"
	"testing"
)

// SEC-14 驗證：RemoveSubFromSystem 拒絕路徑穿越。
func TestRemoveSubFromSystem_PathTraversal(t *testing.T) {
	// 建立暫存 projectRoot
	tmpDir, err := os.MkdirTemp("", "sec14-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 建立 callable 結構
	callableDir := filepath.Join(tmpDir, "subagents", "callable")
	os.MkdirAll(callableDir, 0o700)

	// 建立一個合法 sub
	legitimateSub := filepath.Join(callableDir, "sub-123")
	os.MkdirAll(legitimateSub, 0o700)
	os.WriteFile(filepath.Join(legitimateSub, "test.txt"), []byte("data"), 0o600)

	// 建立一個不在 callable 下的目錄（攻擊目標）
	victimDir := filepath.Join(tmpDir, "important-data")
	os.MkdirAll(victimDir, 0o700)
	os.WriteFile(filepath.Join(victimDir, "secret.txt"), []byte("secret"), 0o600)

	tests := []struct {
		name    string
		subID   string
		wantErr bool
	}{
		{"正常 subID", "sub-123", false},
		{"路徑穿越 ../", "../../important-data", true},
		{"路徑穿越 ../../", "../../../tmp", true},
		{"包含斜線", "sub/../../etc", true},
		{"只有 dots", "..", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoveSubFromSystem(tmpDir, tt.subID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveSubFromSystem(%q) error=%v, wantErr=%v", tt.subID, err, tt.wantErr)
			}
		})
	}

	// 驗證攻擊目標仍然存在
	if _, err := os.Stat(victimDir); os.IsNotExist(err) {
		t.Fatal("CRITICAL: victimDir was deleted by path traversal attack!")
	}
}

// SEC-15 驗證：exportDirPerm 是 0o700。
func TestExportDirPerm(t *testing.T) {
	if exportDirPerm != 0o700 {
		t.Errorf("exportDirPerm=%o, want 0700", exportDirPerm)
	}
}
