//go:build windows

package visual_learning

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	whMouseLL                      = 14
	wmQuit                         = 0x0012
	wmLButtonUp                    = 0x0202
	hcAction                       = 0
	smXVirtual                     = 76
	smYVirtual                     = 77
	smCXVirtual                    = 78
	smCYVirtual                    = 79
	gaRoot                         = 2
	processQueryLimitedInformation = 0x1000
	inputMouse                     = 0
	mouseLeftDown                  = 0x0002
	mouseLeftUp                    = 0x0004
	srccopy                        = 0x00CC0020
	biRGB                          = 0
	dibRGBColors                   = 0
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	gdi32                          = syscall.NewLazyDLL("gdi32.dll")
	procSetWindowsHookExW          = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx        = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx             = user32.NewProc("CallNextHookEx")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procPostThreadMessageW         = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadID         = kernel32.NewProc("GetCurrentThreadId")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessID   = user32.NewProc("GetWindowThreadProcessId")
	procWindowFromPoint            = user32.NewProc("WindowFromPoint")
	procGetAncestor                = user32.NewProc("GetAncestor")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procGetSystemMetrics           = user32.NewProc("GetSystemMetrics")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	procSetCursorPos               = user32.NewProc("SetCursorPos")
	procSendInput                  = user32.NewProc("SendInput")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procGetDC                      = user32.NewProc("GetDC")
	procReleaseDC                  = user32.NewProc("ReleaseDC")
	procCreateCompatibleDC         = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC                   = gdi32.NewProc("DeleteDC")
	procCreateCompatibleBitmap     = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject               = gdi32.NewProc("SelectObject")
	procDeleteObject               = gdi32.NewProc("DeleteObject")
	procBitBlt                     = gdi32.NewProc("BitBlt")
	procGetDIBits                  = gdi32.NewProc("GetDIBits")
)

type nativePoint struct {
	X int32
	Y int32
}

type mouseLLHookStruct struct {
	Pt          nativePoint
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type nativeMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      nativePoint
}

type nativeRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

type mouseInput struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type input struct {
	Type uint32
	Mi   mouseInput
}

// NativeInput records and replays OS-level click input on Windows.
type NativeInput struct {
	mu       sync.Mutex
	hook     uintptr
	threadID uint32
	done     chan struct{}
	callback uintptr
	onClick  func(NativeClickEvent)
	selfExe  string
}

func NewNativeInput() *NativeInput {
	exe, _ := os.Executable()
	return &NativeInput{selfExe: strings.ToLower(filepath.Base(exe))}
}

func (n *NativeInput) Start(onClick func(NativeClickEvent)) error {
	n.mu.Lock()
	if n.done != nil {
		n.mu.Unlock()
		return nil
	}
	n.onClick = onClick
	ready := make(chan error, 1)
	done := make(chan struct{})
	n.done = done
	n.mu.Unlock()

	go n.hookLoop(ready)
	if err := <-ready; err != nil {
		n.mu.Lock()
		if n.done == done {
			n.done = nil
			n.threadID = 0
			n.hook = 0
			n.callback = 0
			n.onClick = nil
		}
		n.mu.Unlock()
		return err
	}
	return nil
}

func (n *NativeInput) Stop() error {
	n.mu.Lock()
	done := n.done
	threadID := n.threadID
	n.mu.Unlock()
	if done == nil {
		return nil
	}
	if threadID != 0 {
		procPostThreadMessageW.Call(uintptr(threadID), wmQuit, 0, 0)
	}
	select {
	case <-done:
	case <-time.After(1200 * time.Millisecond):
		return fmt.Errorf("native input: recorder stop timed out")
	}
	n.mu.Lock()
	n.done = nil
	n.threadID = 0
	n.hook = 0
	n.callback = 0
	n.onClick = nil
	n.mu.Unlock()
	return nil
}

func (n *NativeInput) Click(step LearningReplayStep) NativeReplayResult {
	if step.CoordinateSpace != "screen" && step.Source != "native" {
		return NativeReplayResult{
			OK:      false,
			Skipped: true,
			Method:  "native",
			Index:   step.Index,
			Label:   step.Label,
			X:       step.X,
			Y:       step.Y,
			Error:   "not a native screen-coordinate step",
		}
	}
	foregroundOK := true
	foregroundTitle := ""
	foregroundProcess := ""
	warning := ""
	if step.WindowHandle != 0 {
		procSetForegroundWindow.Call(uintptr(step.WindowHandle))
		foregroundOK, foregroundTitle, foregroundProcess = waitForForegroundWindow(uintptr(step.WindowHandle), 900*time.Millisecond)
		if !foregroundOK {
			warning = "target window was not confirmed as foreground before native click"
		}
	}
	if err := n.moveCursorSmooth(step.X, step.Y); err != nil {
		return NativeReplayResult{
			OK:                false,
			Method:            "native",
			Index:             step.Index,
			Label:             step.Label,
			X:                 step.X,
			Y:                 step.Y,
			Error:             err.Error(),
			Warning:           warning,
			WindowTitle:       step.WindowTitle,
			WindowProcess:     step.WindowProcess,
			ForegroundOK:      foregroundOK,
			ForegroundTitle:   foregroundTitle,
			ForegroundProcess: foregroundProcess,
		}
	}
	time.Sleep(450 * time.Millisecond)
	if err := sendMouseButton(mouseLeftDown); err != nil {
		return NativeReplayResult{
			OK:                false,
			Method:            "native",
			Index:             step.Index,
			Label:             step.Label,
			X:                 step.X,
			Y:                 step.Y,
			Error:             err.Error(),
			Warning:           warning,
			WindowTitle:       step.WindowTitle,
			WindowProcess:     step.WindowProcess,
			ForegroundOK:      foregroundOK,
			ForegroundTitle:   foregroundTitle,
			ForegroundProcess: foregroundProcess,
		}
	}
	mouseDownSent := true
	defer func() {
		if mouseDownSent {
			_ = sendMouseButton(mouseLeftUp)
		}
	}()
	time.Sleep(85 * time.Millisecond)
	if err := sendMouseButton(mouseLeftUp); err != nil {
		return NativeReplayResult{
			OK:                false,
			Method:            "native",
			Index:             step.Index,
			Label:             step.Label,
			X:                 step.X,
			Y:                 step.Y,
			Error:             err.Error(),
			Warning:           warning,
			WindowTitle:       step.WindowTitle,
			WindowProcess:     step.WindowProcess,
			ForegroundOK:      foregroundOK,
			ForegroundTitle:   foregroundTitle,
			ForegroundProcess: foregroundProcess,
		}
	}
	mouseDownSent = false
	time.Sleep(220 * time.Millisecond)
	return NativeReplayResult{
		OK:                true,
		Method:            "native",
		Index:             step.Index,
		Label:             step.Label,
		X:                 step.X,
		Y:                 step.Y,
		Warning:           warning,
		WindowTitle:       step.WindowTitle,
		WindowProcess:     step.WindowProcess,
		ForegroundOK:      foregroundOK,
		ForegroundTitle:   foregroundTitle,
		ForegroundProcess: foregroundProcess,
	}
}

func (n *NativeInput) MoveCursorOnly(step LearningReplayStep) NativeReplayResult {
	if err := n.moveCursorSmooth(step.X, step.Y); err != nil {
		return NativeReplayResult{
			OK:      false,
			Skipped: true,
			Method:  "native_preview",
			Index:   step.Index,
			Label:   step.Label,
			X:       step.X,
			Y:       step.Y,
			Error:   err.Error(),
		}
	}
	return NativeReplayResult{
		OK:      true,
		Skipped: true,
		Method:  "native_preview",
		Index:   step.Index,
		Label:   step.Label,
		X:       step.X,
		Y:       step.Y,
	}
}

func (n *NativeInput) CaptureWindow(hwnd uintptr) (WindowCapture, error) {
	if hwnd == 0 {
		return WindowCapture{}, fmt.Errorf("native capture: window handle is required")
	}
	procSetForegroundWindow.Call(hwnd)
	_, title, procName := waitForForegroundWindow(hwnd, 700*time.Millisecond)
	rect, err := windowRect(hwnd)
	if err != nil {
		return WindowCapture{}, err
	}
	if rect.W <= 0 || rect.H <= 0 {
		return WindowCapture{}, fmt.Errorf("native capture: invalid window rect %+v", rect)
	}
	screenDC, _, err := procGetDC.Call(0)
	if screenDC == 0 {
		return WindowCapture{}, fmt.Errorf("native capture: GetDC failed: %v", err)
	}
	defer procReleaseDC.Call(0, screenDC)

	memDC, _, err := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return WindowCapture{}, fmt.Errorf("native capture: CreateCompatibleDC failed: %v", err)
	}
	defer procDeleteDC.Call(memDC)

	bmp, _, err := procCreateCompatibleBitmap.Call(screenDC, uintptr(rect.W), uintptr(rect.H))
	if bmp == 0 {
		return WindowCapture{}, fmt.Errorf("native capture: CreateCompatibleBitmap failed: %v", err)
	}
	defer procDeleteObject.Call(bmp)

	oldObj, _, _ := procSelectObject.Call(memDC, bmp)
	if oldObj != 0 {
		defer procSelectObject.Call(memDC, oldObj)
	}

	if r, _, err := procBitBlt.Call(memDC, 0, 0, uintptr(rect.W), uintptr(rect.H), screenDC, uintptr(rect.X), uintptr(rect.Y), srccopy); r == 0 {
		return WindowCapture{}, fmt.Errorf("native capture: BitBlt failed: %v", err)
	}

	data := make([]byte, rect.W*rect.H*4)
	info := bitmapInfo{}
	info.Header.Size = uint32(unsafe.Sizeof(info.Header))
	info.Header.Width = int32(rect.W)
	info.Header.Height = -int32(rect.H)
	info.Header.Planes = 1
	info.Header.BitCount = 32
	info.Header.Compression = biRGB
	info.Header.SizeImage = uint32(len(data))
	if r, _, err := procGetDIBits.Call(
		memDC,
		bmp,
		0,
		uintptr(rect.H),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(unsafe.Pointer(&info)),
		dibRGBColors,
	); r == 0 {
		return WindowCapture{}, fmt.Errorf("native capture: GetDIBits failed: %v", err)
	}
	for i := 0; i+3 < len(data); i += 4 {
		data[i], data[i+2] = data[i+2], data[i]
		if data[i+3] == 0 {
			data[i+3] = 255
		}
	}
	if title == "" || procName == "" {
		_, title, procName = windowInfo(hwnd)
	}
	return WindowCapture{
		ImageData:     data,
		Width:         rect.W,
		Height:        rect.H,
		WindowRect:    rect,
		WindowTitle:   title,
		WindowProcess: procName,
	}, nil
}

func sendMouseButton(flags uint32) error {
	in := input{Type: inputMouse, Mi: mouseInput{DwFlags: flags}}
	if r, _, err := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&in)),
		unsafe.Sizeof(in),
	); r != 1 {
		return fmt.Errorf("SendInput failed: %v", err)
	}
	return nil
}

func waitForForegroundWindow(target uintptr, timeout time.Duration) (bool, string, string) {
	deadline := time.Now().Add(timeout)
	lastTitle := ""
	lastProcess := ""
	for {
		hwnd, title, procName := foregroundWindowInfo()
		lastTitle = title
		lastProcess = procName
		if hwnd == target {
			return true, title, procName
		}
		if time.Now().After(deadline) {
			return false, lastTitle, lastProcess
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (n *NativeInput) hookLoop(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := procGetCurrentThreadID.Call()
	n.mu.Lock()
	n.threadID = uint32(threadID)
	n.callback = syscall.NewCallback(func(code int, wParam uintptr, lParam uintptr) uintptr {
		if code >= hcAction && wParam == wmLButtonUp {
			info := (*mouseLLHookStruct)(unsafe.Pointer(lParam))
			n.emitClick(int(info.Pt.X), int(info.Pt.Y))
		}
		next, _, _ := procCallNextHookEx.Call(0, uintptr(code), wParam, lParam)
		return next
	})
	callback := n.callback
	n.mu.Unlock()

	hook, _, err := procSetWindowsHookExW.Call(whMouseLL, callback, 0, 0)
	if hook == 0 {
		ready <- fmt.Errorf("native input: SetWindowsHookExW failed: %w", err)
		close(n.done)
		return
	}
	n.mu.Lock()
	n.hook = hook
	n.mu.Unlock()
	ready <- nil
	defer func() {
		procUnhookWindowsHookEx.Call(hook)
		n.mu.Lock()
		done := n.done
		n.mu.Unlock()
		if done != nil {
			close(done)
		}
	}()

	var msg nativeMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
	}
}

func (n *NativeInput) emitClick(x, y int) {
	hwnd, title, procName := windowInfoFromPoint(x, y)
	if hwnd == 0 {
		hwnd, title, procName = foregroundWindowInfo()
	}
	if hwnd == 0 {
		return
	}
	if n.selfExe != "" && strings.EqualFold(filepath.Base(procName), n.selfExe) {
		return
	}
	screenX := systemMetric(smXVirtual)
	screenY := systemMetric(smYVirtual)
	rect, _ := windowRect(hwnd)
	event := NativeClickEvent{
		Timestamp:     time.Now(),
		X:             x,
		Y:             y,
		Button:        "left",
		WindowTitle:   title,
		WindowProcess: procName,
		WindowHandle:  hwnd,
		ScreenX:       screenX,
		ScreenY:       screenY,
		ScreenWidth:   systemMetric(smCXVirtual),
		ScreenHeight:  systemMetric(smCYVirtual),
		WindowRect:    rect,
	}
	n.mu.Lock()
	onClick := n.onClick
	n.mu.Unlock()
	if onClick != nil {
		go onClick(event)
	}
}

func (n *NativeInput) moveCursorSmooth(targetX, targetY int) error {
	startX, startY := currentCursorPosition()
	if startX == targetX && startY == targetY {
		return nil
	}
	const steps = 28
	const stepDelay = 28 * time.Millisecond
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		// Ease out a little so the user can see where the cursor will land.
		eased := 1 - (1-t)*(1-t)
		x := startX + int(float64(targetX-startX)*eased)
		y := startY + int(float64(targetY-startY)*eased)
		if r, _, err := procSetCursorPos.Call(uintptr(x), uintptr(y)); r == 0 {
			return fmt.Errorf("SetCursorPos failed: %v", err)
		}
		time.Sleep(stepDelay)
	}
	if r, _, err := procSetCursorPos.Call(uintptr(targetX), uintptr(targetY)); r == 0 {
		return fmt.Errorf("SetCursorPos failed: %v", err)
	}
	return nil
}

func currentCursorPosition() (int, int) {
	var pt nativePoint
	if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		return 0, 0
	}
	return int(pt.X), int(pt.Y)
}

func foregroundWindowInfo() (uintptr, string, string) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0, "", ""
	}
	return windowInfo(hwnd)
}

func windowInfoFromPoint(x, y int) (uintptr, string, string) {
	hwnd, _, _ := procWindowFromPoint.Call(packPoint(x, y))
	if hwnd == 0 {
		return 0, "", ""
	}
	if root, _, _ := procGetAncestor.Call(hwnd, gaRoot); root != 0 {
		hwnd = root
	}
	return windowInfo(hwnd)
}

func packPoint(x, y int) uintptr {
	return uintptr(uint32(int32(x))) | (uintptr(uint32(int32(y))) << 32)
}

func windowInfo(hwnd uintptr) (uintptr, string, string) {
	title := windowTitle(hwnd)
	var pid uint32
	procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return hwnd, title, processPath(pid)
}

func windowRect(hwnd uintptr) (PixelBBox, error) {
	var rect nativeRect
	if r, _, err := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); r == 0 {
		return PixelBBox{}, fmt.Errorf("native window rect: GetWindowRect failed: %v", err)
	}
	return PixelBBox{
		X: int(rect.Left),
		Y: int(rect.Top),
		W: int(rect.Right - rect.Left),
		H: int(rect.Bottom - rect.Top),
	}, nil
}

func windowTitle(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, int(length)+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func processPath(pid uint32) string {
	if pid == 0 {
		return ""
	}
	handle, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return ""
	}
	defer procCloseHandle.Call(handle)
	buf := make([]uint16, syscall.MAX_PATH)
	size := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(handle, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 || size == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

func systemMetric(index int) int {
	r, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int(r)
}
