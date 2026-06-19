// search_summary_finalize_sec_test.go — 搜尋摘要拖曳 finalize 的安全行為。
//
// 對齊 reference_finalize_sec_test.go 的 table 風格，重點驗 cancel 分支：
// basename / checksum / 目錄 都要能擋下誤刪；copy 不動檔；暫存父層會被清掉。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// sha256HexOf 算 bytes 的 sha256（hex），測試自用。
func sha256HexOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestFinalizeNativeSearchSummaryExport_CancelCase(t *testing.T) {
	app := &App{}

	t.Run("cancel 正常（basename+checksum 對得上）→ 刪除", func(t *testing.T) {
		dir := t.TempDir()
		content := []byte("# 天氣・台北\n28C 晴時多雲\n")
		landed := filepath.Join(dir, "天氣-台北.md")
		_ = os.WriteFile(landed, content, 0o600)
		temp := filepath.Join(dir, "天氣-台北.md") // 同 basename 即可
		if err := app.FinalizeNativeSearchSummaryExport("cancel", temp, landed, sha256HexOf(content)); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if _, statErr := os.Stat(landed); !os.IsNotExist(statErr) {
			t.Errorf("expected landed removed")
		}
	})

	t.Run("cancel basename 不一致 → 拒絕且保留檔", func(t *testing.T) {
		dir := t.TempDir()
		content := []byte("x")
		landed := filepath.Join(dir, "different_name.md")
		_ = os.WriteFile(landed, content, 0o600)
		temp := filepath.Join(dir, "天氣-台北.md")
		err := app.FinalizeNativeSearchSummaryExport("cancel", temp, landed, sha256HexOf(content))
		if err == nil || !contains(err.Error(), "basename mismatch") {
			t.Errorf("expected basename mismatch, got %v", err)
		}
		if _, statErr := os.Stat(landed); statErr != nil {
			t.Errorf("landed should survive on mismatch, got %v", statErr)
		}
	})

	t.Run("cancel checksum 不符（同名但內容不同）→ 拒絕且保留檔", func(t *testing.T) {
		dir := t.TempDir()
		// 落地資料夾裡剛好有個同名的「使用者既有檔」，內容不同。
		landed := filepath.Join(dir, "天氣-台北.md")
		_ = os.WriteFile(landed, []byte("使用者自己的重要筆記"), 0o600)
		temp := filepath.Join(dir, "天氣-台北.md")
		ourChecksum := sha256HexOf([]byte("# 天氣・台北\n28C\n")) // 我們拖出那份的 checksum
		err := app.FinalizeNativeSearchSummaryExport("cancel", temp, landed, ourChecksum)
		if err == nil || !contains(err.Error(), "checksum mismatch") {
			t.Errorf("expected checksum mismatch, got %v", err)
		}
		if _, statErr := os.Stat(landed); statErr != nil {
			t.Errorf("使用者既有檔不可被刪, got %v", statErr)
		}
	})

	t.Run("cancel 目錄 → 拒絕", func(t *testing.T) {
		dir := t.TempDir()
		landed := filepath.Join(dir, "天氣-台北.md") // 故意做成目錄
		_ = os.MkdirAll(landed, 0o700)
		_ = os.WriteFile(filepath.Join(landed, "guard.txt"), []byte("DO NOT DELETE"), 0o600)
		err := app.FinalizeNativeSearchSummaryExport("cancel", landed, landed, "")
		if err == nil || !contains(err.Error(), "is a directory") {
			t.Errorf("expected directory refused, got %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(landed, "guard.txt")); statErr != nil {
			t.Errorf("dir content should survive, got %v", statErr)
		}
	})

	t.Run("cancel 落地檔不存在 → idempotent 成功", func(t *testing.T) {
		dir := t.TempDir()
		if err := app.FinalizeNativeSearchSummaryExport("cancel", "x.md", filepath.Join(dir, "ghost.md"), ""); err != nil {
			t.Errorf("expected idempotent success, got %v", err)
		}
	})

	t.Run("cancel 空落地路徑 → 成功（無事可做）", func(t *testing.T) {
		if err := app.FinalizeNativeSearchSummaryExport("cancel", "x.md", "", ""); err != nil {
			t.Errorf("expected success on empty landed, got %v", err)
		}
	})
}

func TestFinalizeNativeSearchSummaryExport_CopyAndUnknown(t *testing.T) {
	app := &App{}

	t.Run("copy → 保留落地檔", func(t *testing.T) {
		dir := t.TempDir()
		landed := filepath.Join(dir, "天氣-台北.md")
		_ = os.WriteFile(landed, []byte("keep me"), 0o600)
		if err := app.FinalizeNativeSearchSummaryExport("copy", "", landed, ""); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if _, statErr := os.Stat(landed); statErr != nil {
			t.Errorf("copy 不應刪除落地檔, got %v", statErr)
		}
	})

	t.Run("未知 action → 報錯", func(t *testing.T) {
		err := app.FinalizeNativeSearchSummaryExport("nuke", "", "", "")
		if err == nil || !contains(err.Error(), "unknown action") {
			t.Errorf("expected unknown action error, got %v", err)
		}
	})
}

// 暫存父層在 finalize 後必須被清掉（僅限我們自己的暫存根底下）。
func TestFinalizeNativeSearchSummaryExport_TempCleanup(t *testing.T) {
	app := &App{}
	parent := filepath.Join(os.TempDir(), searchSummaryExportTempRoot, "export-unittest-cleanup")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	temp := filepath.Join(parent, "天氣-台北.md")
	_ = os.WriteFile(temp, []byte("tmp"), 0o600)
	t.Cleanup(func() { _ = os.RemoveAll(parent) })

	if err := app.FinalizeNativeSearchSummaryExport("copy", temp, "", ""); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}
	if _, statErr := os.Stat(parent); !os.IsNotExist(statErr) {
		t.Errorf("暫存父層應被清掉, stat err=%v", statErr)
	}
}

// 暫存路徑落在我們的根底下才會被當成可清的父層；外部路徑一律不回傳。
func TestSearchSummaryTempParent_Guard(t *testing.T) {
	if got := searchSummaryTempParent("/etc/passwd"); got != "" {
		t.Errorf("外部路徑不可被視為暫存父層, got %q", got)
	}
	if got := searchSummaryTempParent(""); got != "" {
		t.Errorf("空路徑應回空, got %q", got)
	}
	good := filepath.Join(os.TempDir(), searchSummaryExportTempRoot, "export-1", "a.md")
	if got := searchSummaryTempParent(good); got == "" {
		t.Errorf("暫存根底下的路徑應回父層, got 空")
	}
}
