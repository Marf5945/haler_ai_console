// reference_finalize_sec_test.go — SEC-W02 同類修補（reference file）table test
//
// 目的：驗證 FinalizeNativeReferenceFileExport 的 "remove" / "cancel" 分支
// 對被誘導的目錄 / 不一致 basename 路徑會 reject。
//
// 不抽 helper，直接 table-driven 呼叫 binding。
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFinalizeNativeReferenceFileExport_RemoveCase(t *testing.T) {
	dir := t.TempDir()
	app := &App{}

	t.Run("remove valid file → succeeds", func(t *testing.T) {
		path := filepath.Join(dir, "ref1.txt")
		_ = os.WriteFile(path, []byte("hello"), 0o600)
		if err := app.FinalizeNativeReferenceFileExport("remove", path, ""); err != nil {
			t.Errorf("expected success, got %v", err)
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Errorf("expected file removed")
		}
	})

	t.Run("remove directory → reject", func(t *testing.T) {
		subdir := filepath.Join(dir, "subdir_to_protect")
		if err := os.MkdirAll(subdir, 0o700); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(subdir, "important.txt"), []byte("DO NOT DELETE"), 0o600)
		err := app.FinalizeNativeReferenceFileExport("remove", subdir, "")
		if err == nil || !contains(err.Error(), "is a directory") {
			t.Errorf("expected 'is a directory' error, got %v", err)
		}
		// 重要：subdir 內檔案必須還在
		if _, statErr := os.Stat(filepath.Join(subdir, "important.txt")); statErr != nil {
			t.Errorf("subdir content should be intact, got %v", statErr)
		}
	})

	t.Run("remove non-existent → success (idempotent)", func(t *testing.T) {
		if err := app.FinalizeNativeReferenceFileExport("remove", filepath.Join(dir, "gone.txt"), ""); err != nil {
			t.Errorf("expected idempotent success, got %v", err)
		}
	})

	t.Run("remove empty path → reject", func(t *testing.T) {
		err := app.FinalizeNativeReferenceFileExport("remove", "", "")
		if err == nil || !contains(err.Error(), "source path is empty") {
			t.Errorf("expected empty path error, got %v", err)
		}
	})
}

func TestFinalizeNativeReferenceFileExport_CancelCase(t *testing.T) {
	dir := t.TempDir()
	app := &App{}

	t.Run("cancel valid landed file → succeeds", func(t *testing.T) {
		source := filepath.Join(dir, "ref2.txt")
		_ = os.WriteFile(source, []byte("source"), 0o600)
		landed := filepath.Join(dir, "ref2.txt") // 同 basename
		// (在這個 test 內，source 與 landed 同位置；驗 basename 一致即可)
		if err := app.FinalizeNativeReferenceFileExport("cancel", source, landed); err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("cancel basename mismatch → reject", func(t *testing.T) {
		source := filepath.Join(dir, "original.txt")
		_ = os.WriteFile(source, []byte("src"), 0o600)
		landed := filepath.Join(dir, "different_name.txt")
		_ = os.WriteFile(landed, []byte("evil-landed"), 0o600)
		err := app.FinalizeNativeReferenceFileExport("cancel", source, landed)
		if err == nil || !contains(err.Error(), "basename mismatch") {
			t.Errorf("expected basename mismatch, got %v", err)
		}
		// landed 應該還在
		if _, statErr := os.Stat(landed); statErr != nil {
			t.Errorf("landed should not be deleted on mismatch, got %v", statErr)
		}
	})

	t.Run("cancel directory → reject", func(t *testing.T) {
		landed := filepath.Join(dir, "evil_dir")
		if err := os.MkdirAll(landed, 0o700); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(landed, "guard.txt"), []byte("DO NOT DELETE"), 0o600)
		err := app.FinalizeNativeReferenceFileExport("cancel", "anything.txt", landed)
		if err == nil || !contains(err.Error(), "is a directory") {
			t.Errorf("expected 'is a directory' error, got %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(landed, "guard.txt")); statErr != nil {
			t.Errorf("dir content should survive, got %v", statErr)
		}
	})

	t.Run("cancel non-existent landed → success (idempotent)", func(t *testing.T) {
		if err := app.FinalizeNativeReferenceFileExport("cancel", "src.txt", filepath.Join(dir, "ghost.txt")); err != nil {
			t.Errorf("expected idempotent success, got %v", err)
		}
	})
}
