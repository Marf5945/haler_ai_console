//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// v3.1.16 Phase 3 — Windows 後台浮動頭像基礎實作。
//
// 對齊 macOS 端 applyFloatingAvatarWindow 的介面：進入後台頭像時把主視窗
// 切成「無邊框 + 永遠置頂」，還原時切回一般有框視窗。透明（per-pixel alpha）
// 屬於後續強化；本版先做基礎的 borderless + topmost，64x64 縮放與拖曳沿用
// 前端的 Wails runtime（WindowSetSize / WindowSetPosition）。
//
// 視窗以標題列字串定位（對應 wails_main.go 的 Title）。

const winAvatarWindowTitle = "HaLer AI Console"

const (
	gwlStyle  = ^uintptr(15) // -16
	wsPopup   = uintptr(0x80000000)
	wsVisible = uintptr(0x10000000)
	wsOverlap = uintptr(0x00CF0000) // WS_OVERLAPPEDWINDOW

	hwndTopmost   = ^uintptr(0) // -1
	hwndNoTopmost = ^uintptr(1) // -2

	swpNoMove      = uintptr(0x0002)
	swpNoSize      = uintptr(0x0001)
	swpFrameChange = uintptr(0x0020)
	swpShowWindow  = uintptr(0x0040)
)

var (
	avatarUser32          = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW       = avatarUser32.NewProc("FindWindowW")
	procGetWindowLongPtrW = avatarUser32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = avatarUser32.NewProc("SetWindowLongPtrW")
	procSetWindowPos      = avatarUser32.NewProc("SetWindowPos")
)

func winFindAvatarWindow() uintptr {
	title, err := syscall.UTF16PtrFromString(winAvatarWindowTitle)
	if err != nil {
		return 0
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	return hwnd
}

func applyFloatingAvatarWindow(on bool) {
	hwnd := winFindAvatarWindow()
	if hwnd == 0 {
		return
	}
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	if on {
		// 去邊框（保留可見），切成 popup。
		style = (style &^ wsOverlap) | wsPopup | wsVisible
		procSetWindowLongPtrW.Call(hwnd, gwlStyle, style)
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0,
			swpNoMove|swpNoSize|swpFrameChange|swpShowWindow)
	} else {
		// 還原一般有框視窗。
		style = (style &^ wsPopup) | wsOverlap | wsVisible
		procSetWindowLongPtrW.Call(hwnd, gwlStyle, style)
		procSetWindowPos.Call(hwnd, hwndNoTopmost, 0, 0, 0, 0,
			swpNoMove|swpNoSize|swpFrameChange|swpShowWindow)
	}
}
