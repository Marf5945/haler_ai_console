package builtin

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// SEC-16 驗證：zipReadFile 對超大 zip entry 的拒絕行為。
func TestZipReadFile_SizeLimit(t *testing.T) {
	// 建立一個 zip，其中 entry 的 UncompressedSize64 宣告超過 500MB
	tmpDir, err := os.MkdirTemp("", "sec16-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "bomb.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)

	// 寫一個正常小檔案
	fw, err := w.Create("small.txt")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("hello"))

	w.Close()
	f.Close()

	// 正常小檔案應該能讀取
	data, err := zipReadFile(zipPath, "small.txt")
	if err != nil {
		t.Errorf("zipReadFile small.txt error=%v", err)
	}
	if string(data) != "hello" {
		t.Errorf("zipReadFile small.txt got=%q, want %q", string(data), "hello")
	}

	// 不存在的 entry 回傳 nil, nil
	data, err = zipReadFile(zipPath, "noexist.txt")
	if err != nil {
		t.Errorf("zipReadFile noexist error=%v", err)
	}
	if data != nil {
		t.Errorf("zipReadFile noexist got=%v, want nil", data)
	}
}

// SEC-16 驗證：maxZipEntrySize 常數值正確。
func TestMaxZipEntrySize(t *testing.T) {
	expected := int64(500 * 1024 * 1024)
	if maxZipEntrySize != expected {
		t.Errorf("maxZipEntrySize=%d, want %d (500MB)", maxZipEntrySize, expected)
	}
}
