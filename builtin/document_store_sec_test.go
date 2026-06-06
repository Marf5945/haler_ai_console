package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

// SEC-13 驗證：blobPath 使用 filepath.Base 防止路徑穿越。
func TestBlobPath_PathTraversal(t *testing.T) {
	// 建立暫存目錄模擬 docsDir
	tmpDir, err := os.MkdirTemp("", "sec13-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := &Store{docsDir: tmpDir}

	tests := []struct {
		name     string
		docID    string
		wantBase string // 預期路徑只包含這個檔名
	}{
		{"正常 docID", "doc-1716000000", "doc-1716000000.json"},
		{"路徑穿越 ../", "../../etc/passwd", "passwd.json"},
		{"路徑穿越 subdir/", "subdir/evil", "evil.json"},
		{"路徑穿越 absolute", "/etc/shadow", "shadow.json"},
		{"空字串", "", "..json"}, // filepath.Base("") 回傳 "."
		{"只有 dots", "..", "...json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.blobPath(tt.docID)
			// 確認路徑在 docsDir 內
			dir := filepath.Dir(got)
			if dir != tmpDir {
				t.Errorf("blobPath(%q) dir=%q, want %q — 路徑穿越!", tt.docID, dir, tmpDir)
			}
			base := filepath.Base(got)
			if base != tt.wantBase {
				t.Errorf("blobPath(%q) base=%q, want %q", tt.docID, base, tt.wantBase)
			}
		})
	}
}
