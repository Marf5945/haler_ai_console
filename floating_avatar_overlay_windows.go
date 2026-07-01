//go:build windows

package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	avatarOverlayClassName = "HaLerFloatingAvatarOverlay"

	csDblClks = 0x0008

	wsPopup   = uintptr(0x80000000)
	wsVisible = uintptr(0x10000000)

	wsExTopmost    = uintptr(0x00000008)
	wsExTool       = uintptr(0x00000080)
	wsExLayered    = uintptr(0x00080000)
	wsExNoActivate = uintptr(0x08000000)

	wmClose           = 0x0010
	wmDestroy         = 0x0002
	wmNCHitTest       = 0x0084
	wmLButtonDblClk   = 0x0203
	wmNCLButtonDblClk = 0x00A3
	wmRButtonUp       = 0x0205
	wmNCRButtonUp     = 0x00A5
	htCaption         = 2

	mfString     = uintptr(0x00000000)
	tpmRight     = uintptr(0x0002)
	tpmReturnCmd = uintptr(0x0100)
	menuRestore  = uintptr(1001)

	ulwAlpha   = uintptr(0x00000002)
	acSrcOver  = 0x00
	acSrcAlpha = 0x01

	dibRGBColors = uintptr(0)
	biRGB        = uint32(0)
)

var (
	overlayUser32 = windows.NewLazySystemDLL("user32.dll")
	overlayGDI32  = windows.NewLazySystemDLL("gdi32.dll")
	overlayKernel = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW    = overlayUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = overlayUser32.NewProc("CreateWindowExW")
	procDefWindowProcW      = overlayUser32.NewProc("DefWindowProcW")
	procDestroyWindow       = overlayUser32.NewProc("DestroyWindow")
	procPostQuitMessage     = overlayUser32.NewProc("PostQuitMessage")
	procGetMessageW         = overlayUser32.NewProc("GetMessageW")
	procTranslateMessage    = overlayUser32.NewProc("TranslateMessage")
	procDispatchMessageW    = overlayUser32.NewProc("DispatchMessageW")
	procUpdateLayeredWindow = overlayUser32.NewProc("UpdateLayeredWindow")
	procGetDC               = overlayUser32.NewProc("GetDC")
	procReleaseDC           = overlayUser32.NewProc("ReleaseDC")
	procPostMessageW        = overlayUser32.NewProc("PostMessageW")
	procSetForegroundWindow = overlayUser32.NewProc("SetForegroundWindow")
	procOverlayGetCursorPos = overlayUser32.NewProc("GetCursorPos")
	procGetWindowRect       = overlayUser32.NewProc("GetWindowRect")
	procCreatePopupMenu     = overlayUser32.NewProc("CreatePopupMenu")
	procAppendMenuW         = overlayUser32.NewProc("AppendMenuW")
	procTrackPopupMenu      = overlayUser32.NewProc("TrackPopupMenu")
	procDestroyMenu         = overlayUser32.NewProc("DestroyMenu")
	procGetModuleHandleW    = overlayKernel.NewProc("GetModuleHandleW")
	procCreateCompatibleDC  = overlayGDI32.NewProc("CreateCompatibleDC")
	procCreateDIBSection    = overlayGDI32.NewProc("CreateDIBSection")
	procSelectObject        = overlayGDI32.NewProc("SelectObject")
	procDeleteObject        = overlayGDI32.NewProc("DeleteObject")
	procDeleteDC            = overlayGDI32.NewProc("DeleteDC")

	avatarOverlayWndProc = syscall.NewCallback(avatarOverlayWindowProc)

	overlayMu        sync.Mutex
	overlayHWND      uintptr
	overlayAction    func(string)
	overlayClassOnce sync.Once
	overlayClassErr  error
)

type overlayPoint struct {
	X int32
	Y int32
}

type overlaySize struct {
	CX int32
	CY int32
}

type overlayBlendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type overlayMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      overlayPoint
}

type overlayRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type overlayWndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type overlayBitmapInfoHeader struct {
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

type overlayBitmapInfo struct {
	Header overlayBitmapInfoHeader
	Colors [1]uint32
}

func showFloatingAvatarOverlay(imagePath string, x int, y int, maxW int, maxH int, onAction func(string)) error {
	img, err := loadOverlayImage(imagePath, maxW, maxH)
	if err != nil {
		return err
	}
	closeFloatingAvatarOverlay()

	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		runFloatingAvatarOverlay(img, x, y, onAction, ready)
	}()
	return <-ready
}

func closeFloatingAvatarOverlay() {
	overlayMu.Lock()
	hwnd := overlayHWND
	overlayMu.Unlock()
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	}
}

func floatingAvatarOverlayPosition() (int, int) {
	overlayMu.Lock()
	hwnd := overlayHWND
	overlayMu.Unlock()
	if hwnd == 0 {
		return 0, 0
	}
	var rect overlayRect
	ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 {
		return 0, 0
	}
	return int(rect.Left), int(rect.Top)
}

func runFloatingAvatarOverlay(img image.Image, x int, y int, onAction func(string), ready chan<- error) {
	if err := registerFloatingAvatarOverlayClass(); err != nil {
		ready <- err
		return
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		ready <- fmt.Errorf("floating avatar overlay image is empty")
		return
	}

	className, _ := syscall.UTF16PtrFromString(avatarOverlayClassName)
	title, _ := syscall.UTF16PtrFromString("HaLer Floating Avatar")
	hwnd, _, err := procCreateWindowExW.Call(
		wsExLayered|wsExTopmost|wsExTool|wsExNoActivate,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsPopup|wsVisible,
		uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(width)), uintptr(int32(height)),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		ready <- fmt.Errorf("CreateWindowExW failed: %w", err)
		return
	}

	overlayMu.Lock()
	overlayHWND = hwnd
	overlayAction = onAction
	overlayMu.Unlock()

	if err := updateOverlayBitmap(hwnd, img, x, y); err != nil {
		procDestroyWindow.Call(hwnd)
		ready <- err
		return
	}
	ready <- nil

	var msg overlayMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	overlayMu.Lock()
	if overlayHWND == hwnd {
		overlayHWND = 0
		overlayAction = nil
	}
	overlayMu.Unlock()
}

func registerFloatingAvatarOverlayClass() error {
	overlayClassOnce.Do(func() {
		className, _ := syscall.UTF16PtrFromString(avatarOverlayClassName)
		instance, _, _ := procGetModuleHandleW.Call(0)
		wc := overlayWndClassEx{
			Size:      uint32(unsafe.Sizeof(overlayWndClassEx{})),
			Style:     csDblClks,
			WndProc:   avatarOverlayWndProc,
			Instance:  instance,
			ClassName: className,
		}
		atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			overlayClassErr = fmt.Errorf("RegisterClassExW failed: %w", err)
		}
	})
	return overlayClassErr
}

func avatarOverlayWindowProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	switch msg {
	case wmNCHitTest:
		return htCaption
	case wmLButtonDblClk, wmNCLButtonDblClk:
		emitOverlayAction("restore")
		procDestroyWindow.Call(hwnd)
		return 0
	case wmRButtonUp, wmNCRButtonUp:
		action := trackOverlayMenu(hwnd)
		if action == "restore" {
			emitOverlayAction(action)
			procDestroyWindow.Call(hwnd)
		} else if action != "" {
			emitOverlayAction(action)
		}
		return 0
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
}

func emitOverlayAction(action string) {
	overlayMu.Lock()
	callback := overlayAction
	overlayMu.Unlock()
	if callback != nil {
		go callback(action)
	}
}

func trackOverlayMenu(hwnd uintptr) string {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return ""
	}
	defer procDestroyMenu.Call(menu)
	appendOverlayMenuItem(menu, menuRestore, "Restore UI")

	var pt overlayPoint
	procOverlayGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmRight|tpmReturnCmd, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	switch cmd {
	case menuRestore:
		return "restore"
	default:
		return ""
	}
}

func appendOverlayMenuItem(menu uintptr, id uintptr, label string) {
	text, _ := syscall.UTF16PtrFromString(label)
	procAppendMenuW.Call(menu, mfString, id, uintptr(unsafe.Pointer(text)))
}

func updateOverlayBitmap(hwnd uintptr, img image.Image, x int, y int) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		return fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, screenDC)

	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(memDC)

	var bits unsafe.Pointer
	bmi := overlayBitmapInfo{
		Header: overlayBitmapInfoHeader{
			Size:        uint32(unsafe.Sizeof(overlayBitmapInfoHeader{})),
			Width:       int32(width),
			Height:      -int32(height),
			Planes:      1,
			BitCount:    32,
			Compression: biRGB,
			SizeImage:   uint32(width * height * 4),
		},
	}
	hbmp, _, _ := procCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bmi)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbmp == 0 || bits == nil {
		return fmt.Errorf("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(hbmp)
	oldObj, _, _ := procSelectObject.Call(memDC, hbmp)
	if oldObj != 0 {
		defer procSelectObject.Call(memDC, oldObj)
	}

	fillPremultipliedBGRA(bits, img)

	dst := overlayPoint{X: int32(x), Y: int32(y)}
	size := overlaySize{CX: int32(width), CY: int32(height)}
	src := overlayPoint{X: 0, Y: 0}
	blend := overlayBlendFunction{BlendOp: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: acSrcAlpha}
	ok, _, err := procUpdateLayeredWindow.Call(
		hwnd,
		screenDC,
		uintptr(unsafe.Pointer(&dst)),
		uintptr(unsafe.Pointer(&size)),
		memDC,
		uintptr(unsafe.Pointer(&src)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ulwAlpha,
	)
	if ok == 0 {
		return fmt.Errorf("UpdateLayeredWindow failed: %w", err)
	}
	return nil
}

func fillPremultipliedBGRA(bits unsafe.Pointer, img image.Image) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	buf := unsafe.Slice((*byte)(bits), width*height*4)
	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			alpha := uint32(a >> 8)
			buf[index+0] = byte((uint32(b>>8) * alpha) / 255)
			buf[index+1] = byte((uint32(g>>8) * alpha) / 255)
			buf[index+2] = byte((uint32(r>>8) * alpha) / 255)
			buf[index+3] = byte(alpha)
			index += 4
		}
	}
}

func loadOverlayImage(path string, maxW int, maxH int) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return fitOverlayImage(img, maxW, maxH), nil
}

func fitOverlayImage(img image.Image, maxW int, maxH int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 || maxW <= 0 || maxH <= 0 || (width <= maxW && height <= maxH) {
		return img
	}
	scaleW := float64(maxW) / float64(width)
	scaleH := float64(maxH) / float64(height)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	nextW := max(1, int(float64(width)*scale))
	nextH := max(1, int(float64(height)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, nextW, nextH))
	for y := 0; y < nextH; y++ {
		srcY := bounds.Min.Y + y*height/nextH
		for x := 0; x < nextW; x++ {
			srcX := bounds.Min.X + x*width/nextW
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst
}
