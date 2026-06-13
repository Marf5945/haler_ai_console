// native_drag_dropdir.go — Linux 桌面匯出落點解析（平台中立、可跨平台單元測試）。
// 刻意不帶 GOOS 檔名後綴：邏輯為純 Go，讓 Windows/macOS 也能跑測試；
// 實際只有 native_drag_linux.go 會呼叫。
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// linuxDropDirCandidates 回傳 Linux 桌面匯出落點的候選順序：
// XDG 桌面 → ~/Desktop → ~/Downloads → ~（家目錄）。
// 抽成純函式（吃 home 與 xdgDesktop 兩個輸入）便於測試，不在內部讀環境。
func linuxDropDirCandidates(home, xdgDesktop string) []string {
	var c []string
	if x := strings.TrimSpace(xdgDesktop); x != "" {
		c = append(c, x)
	}
	if h := strings.TrimSpace(home); h != "" {
		c = append(c, filepath.Join(h, "Desktop"), filepath.Join(h, "Downloads"), h)
	}
	return c
}

// firstExistingDir 依序回傳第一個存在的目錄；都不存在回 false。
func firstExistingDir(candidates []string) (string, bool) {
	for _, d := range candidates {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d, true
		}
	}
	return "", false
}

// resolveLinuxXDGDesktop 先試 `xdg-user-dir DESKTOP`，失敗退回 $XDG_DESKTOP_DIR。
// 這是唯一會碰外部命令/環境的地方，與純邏輯分離。
func resolveLinuxXDGDesktop() string {
	if out, err := exec.Command("xdg-user-dir", "DESKTOP").Output(); err == nil {
		if d := strings.TrimSpace(string(out)); d != "" {
			return d
		}
	}
	return strings.TrimSpace(os.Getenv("XDG_DESKTOP_DIR"))
}
