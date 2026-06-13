// native_drag_dropdir_test.go — Linux 落點解析純邏輯測試（跨平台可跑）。
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxDropDirCandidatesOrder(t *testing.T) {
	got := linuxDropDirCandidates("/home/u", "/home/u/桌面")
	want := []string{"/home/u/桌面", filepath.Join("/home/u", "Desktop"), filepath.Join("/home/u", "Downloads"), "/home/u"}
	if len(got) != len(want) {
		t.Fatalf("候選數錯誤：%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("順序錯誤 #%d：got %q want %q", i, got[i], want[i])
		}
	}
}

func TestLinuxDropDirCandidatesEmpty(t *testing.T) {
	if got := linuxDropDirCandidates("", ""); len(got) != 0 {
		t.Fatalf("無 home/xdg 應回空，got %v", got)
	}
	// 只有 home、無 xdg：不含 XDG 桌面項。
	got := linuxDropDirCandidates("/home/u", "")
	if len(got) != 3 || got[0] != filepath.Join("/home/u", "Desktop") {
		t.Fatalf("無 xdg 時候選錯誤：%v", got)
	}
}

func TestFirstExistingDir(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "Desktop")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	// 第一個不存在、第二個存在 → 取第二個。
	got, ok := firstExistingDir([]string{filepath.Join(tmp, "missing"), real, tmp})
	if !ok || got != real {
		t.Fatalf("應取第一個存在目錄 %q，got %q ok=%v", real, got, ok)
	}
	// 全不存在 → false。
	if _, ok := firstExistingDir([]string{filepath.Join(tmp, "nope")}); ok {
		t.Fatal("全不存在應回 false")
	}
	// 檔案不算目錄。
	f := filepath.Join(tmp, "afile")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := firstExistingDir([]string{f}); ok {
		t.Fatal("檔案不應被當作目錄落點")
	}
}
