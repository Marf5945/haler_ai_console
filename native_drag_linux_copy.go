//go:build linux

// native_drag_linux_copy.go — Linux 落點複製 fallback（linux 無論有無 gtk tag 都編入）。
// 抽成共用函式，讓 `!gtk` 的預設版與 `gtk` 原生版都能在失敗時回落同一套邏輯，
// 保證 `-tags gtk` 建置永遠不會比 fallback 差（防回退）。
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// linuxDesktopCopyFallback 把匯出來源複製到桌面/下載/家目錄第一個可寫處。
// reason 會帶進 Message，讓前端說明為何是複製而非真正拖放。
func linuxDesktopCopyFallback(path, reason string) nativeDragResult {
	info, err := os.Stat(path)
	if err != nil {
		writeNativeDragPhase("linux-fallback-source-missing", err.Error())
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "；原始匯出來源不存在"}
	}

	dropDir, ok := firstExistingDir(linuxDropDirCandidates(os.Getenv("HOME"), resolveLinuxXDGDesktop()))
	if !ok {
		writeNativeDragPhase("linux-fallback-no-dropdir", "")
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "；找不到可寫入的桌面或家目錄"}
	}

	target := filepath.Join(dropDir, filepath.Base(path))
	if _, err := os.Stat(target); err == nil {
		// 拒絕覆蓋，對齊 Windows fallback 行為。
		writeNativeDragPhase("linux-fallback-target-exists", fmt.Sprintf("target=%q", target))
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "；落點已存在同名項目：" + target}
	}

	if info.IsDir() {
		err = copySubExportDirectory(path, target)
	} else {
		err = copySubExportFile(path, target, info.Mode())
	}
	if err != nil {
		writeNativeDragPhase("linux-fallback-copy-error", err.Error())
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "；複製到桌面失敗：" + err.Error()}
	}

	writeNativeDragPhase("linux-fallback-copied", fmt.Sprintf("target=%q", target))
	return nativeDragResult{
		Status:           nativeDragStatusSuccess,
		FallbackRequired: true, // 已落地但非真正拖放，前端據此提示落點
		Message:          reason + "，已複製到 " + dropDir,
		LandedPath:       target,
		DropTargetKind:   "linux-desktop-fallback",
		DropTargetDir:    dropDir,
	}
}
