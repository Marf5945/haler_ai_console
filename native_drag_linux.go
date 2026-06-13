//go:build linux && !gtk

// native_drag_linux.go — Linux 預設建置（無 gtk tag）：直接走桌面複製 fallback。
// 真正的 GTK 原生拖放在 native_drag_linux_gtk.go（需 `-tags gtk` 建置）。
// 兩檔以 gtk build tag 互斥，避免 startNativeFileDrag 重複定義。
package main

func startNativeFileDrag(path string) nativeDragResult {
	return linuxDesktopCopyFallback(path, "Linux 尚未支援原生拖放")
}
