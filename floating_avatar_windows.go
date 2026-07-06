//go:build windows

package main

// floating_avatar_windows.go — v2.7：Windows 後台浮動頭像，照 macOS 接法實作。
//
// macOS（floating_avatar_darwin.go）的做法是把「主視窗本體」切成
// 無框、真透明、永遠置頂的浮窗，讓 React 前端（含 Pixi 動態全身像）
// 繼續在同一個 webview 裡渲染。這裡用 Win32 做等價切換：
//
//   mac (AppKit)                            windows (Win32)
//   ─────────────────────────────           ─────────────────────────────
//   隱藏標題列/紅綠燈 + FullSizeContent  →  拔 WS_CAPTION/WS_THICKFRAME/
//                                           WS_SYSMENU/WS_MIN/MAXBOX，改 WS_POPUP
//   setLevel:NSStatusWindowLevel         →  SetWindowPos(HWND_TOPMOST)
//   CollectionBehavior IgnoresCycle      →  WS_EX_TOOLWINDOW（躲工作列/Alt-Tab）
//   setOpaque:NO + clearColor            →  建窗時 WindowIsTranslucent
//                                           （WS_EX_NOREDIRECTIONBITMAP，見
//                                           wails_main.go）＋ WebviewIsTransparent
//
// 真透明的來源是 wails 建窗選項（該旗標只能在建窗當下給），這裡只負責
// 執行期可逆的部分：無框 ↔ 有框、置頂 ↔ 一般、tool window ↔ 一般。
// 刻意「不用」WS_EX_LAYERED：WebView2 與 layered window 不相容，會整片黑
// （這正是舊版把此函式做成 no-op 的原因；overlay 靜態 PNG 路徑才用 layered）。
//
// 與 mac 版相同，呼叫端（App.jsx）會在視窗尺寸變更後重呼叫一次做幂等重套，
// 因此本實作必須可重入：原始樣式只在第一次進入時保存。

import (
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	favGWLStyle   = -16 // GWL_STYLE
	favGWLExStyle = -20 // GWL_EXSTYLE

	favWSCaption     = uintptr(0x00C00000) // WS_CAPTION
	favWSThickFrame  = uintptr(0x00040000) // WS_THICKFRAME（可拉伸框）
	favWSSysMenu     = uintptr(0x00080000) // WS_SYSMENU
	favWSMinimizeBox = uintptr(0x00020000) // WS_MINIMIZEBOX
	favWSMaximizeBox = uintptr(0x00010000) // WS_MAXIMIZEBOX

	favSWPNoSize       = uintptr(0x0001)
	favSWPNoMove       = uintptr(0x0002)
	favSWPNoActivate   = uintptr(0x0010)
	favSWPFrameChanged = uintptr(0x0020)
	favSWPShowWindow   = uintptr(0x0040)

	favGWOwner = uintptr(4) // GW_OWNER
)

// HWND_TOPMOST = -1、HWND_NOTOPMOST = -2（以 two's complement 表示）。
var (
	favHWNDTopmost   = ^uintptr(0)
	favHWNDNoTopmost = ^uintptr(1)
)

var (
	favUser32                       = windows.NewLazySystemDLL("user32.dll")
	favProcEnumWindows              = favUser32.NewProc("EnumWindows")
	favProcGetWindowThreadProcessID = favUser32.NewProc("GetWindowThreadProcessId")
	favProcIsWindowVisible          = favUser32.NewProc("IsWindowVisible")
	favProcGetWindow                = favUser32.NewProc("GetWindow")
	favProcGetClassNameW            = favUser32.NewProc("GetClassNameW")
	favProcGetWindowLongPtrW        = favUser32.NewProc("GetWindowLongPtrW")
	favProcSetWindowLongPtrW        = favUser32.NewProc("SetWindowLongPtrW")
	favProcSetWindowPos             = favUser32.NewProc("SetWindowPos")
)

// favState 記住主視窗與進入浮窗前的原始樣式，restore 時原樣放回。
var favState struct {
	sync.Mutex
	hwnd    uintptr
	saved   bool
	style   uintptr
	exStyle uintptr
}

// EnumWindows 的 callback 只建一次（syscall.NewCallback 不可重複大量配置）。
var (
	favEnumOnce     sync.Once
	favEnumCallback uintptr
	favEnumResult   uintptr
)

func favEnumProc(hwnd uintptr, _ uintptr) uintptr {
	var pid uint32
	favProcGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid))) //nolint:errcheck
	if pid != windows.GetCurrentProcessId() {
		return 1 // 續列
	}
	if visible, _, _ := favProcIsWindowVisible.Call(hwnd); visible == 0 {
		return 1
	}
	if owner, _, _ := favProcGetWindow.Call(hwnd, favGWOwner); owner != 0 {
		return 1 // 跳過附屬視窗（對話框等）
	}
	// 跳過原生 overlay（舊路徑）與釘選子視窗，只認 wails 主視窗類別。
	var buf [64]uint16
	length, _, _ := favProcGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if length > 0 {
		switch windows.UTF16ToString(buf[:length]) {
		case avatarOverlayClassName:
			return 1
		case "wailsWindow": // wails v2 預設主視窗類別
			favEnumResult = hwnd
			return 0 // 找到，停止列舉
		}
	}
	// 類別名不認得時先記著當備援（同 PID、可見、無 owner 的頂層視窗）。
	if favEnumResult == 0 {
		favEnumResult = hwnd
	}
	return 1
}

// favMainWindowHWND 找出本行程的 wails 主視窗（對應 mac 版 AIConsoleResolveWindow）。
func favMainWindowHWND() uintptr {
	favEnumOnce.Do(func() {
		favEnumCallback = syscall.NewCallback(favEnumProc)
	})
	favEnumResult = 0
	favProcEnumWindows.Call(favEnumCallback, 0) //nolint:errcheck
	return favEnumResult
}

func favGetWindowLongPtr(hwnd uintptr, index int) uintptr {
	value, _, _ := favProcGetWindowLongPtrW.Call(hwnd, uintptr(index))
	return value
}

func favSetWindowLongPtr(hwnd uintptr, index int, value uintptr) {
	favProcSetWindowLongPtrW.Call(hwnd, uintptr(index), value) //nolint:errcheck
}

// applyFloatingAvatarWindow 在 Windows 上切換主視窗的浮動頭像型態。
// on=true：無框（WS_POPUP）＋置頂＋tool window；on=false：原樣還原。
// 真透明由建窗選項提供（wails_main.go），前端 CSS 決定哪些像素透明。
func applyFloatingAvatarWindow(on bool) {
	favState.Lock()
	defer favState.Unlock()

	hwnd := favState.hwnd
	if hwnd == 0 {
		hwnd = favMainWindowHWND()
		favState.hwnd = hwnd
	}
	if hwnd == 0 {
		return
	}

	if on {
		// 原始樣式只在第一次進入時保存；之後的呼叫是尺寸變更後的幂等重套。
		if !favState.saved {
			favState.style = favGetWindowLongPtr(hwnd, favGWLStyle)
			favState.exStyle = favGetWindowLongPtr(hwnd, favGWLExStyle)
			favState.saved = true
		}
		frame := favWSCaption | favWSThickFrame | favWSSysMenu | favWSMinimizeBox | favWSMaximizeBox
		favSetWindowLongPtr(hwnd, favGWLStyle, (favState.style&^frame)|wsPopup|wsVisible)
		// 加 WS_EX_TOOLWINDOW（≈ mac 的 IgnoresCycle：躲工作列與 Alt-Tab）。
		// 不加 WS_EX_NOACTIVATE——迷你框要能收鍵盤輸入（對應 mac 保留 titled 讓視窗可成為 key）。
		favSetWindowLongPtr(hwnd, favGWLExStyle, favState.exStyle|wsExTool)
		favProcSetWindowPos.Call(hwnd, favHWNDTopmost, 0, 0, 0, 0, //nolint:errcheck
			favSWPNoMove|favSWPNoSize|favSWPNoActivate|favSWPFrameChanged|favSWPShowWindow)
		return
	}

	// 還原一般有框主控台視窗。
	if favState.saved {
		favSetWindowLongPtr(hwnd, favGWLStyle, favState.style|wsVisible)
		favSetWindowLongPtr(hwnd, favGWLExStyle, favState.exStyle)
		favState.saved = false
	}
	favProcSetWindowPos.Call(hwnd, favHWNDNoTopmost, 0, 0, 0, 0, //nolint:errcheck
		favSWPNoMove|favSWPNoSize|favSWPNoActivate|favSWPFrameChanged|favSWPShowWindow)
}
