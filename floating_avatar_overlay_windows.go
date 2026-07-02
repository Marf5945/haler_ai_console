//go:build windows

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
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
	wmKeyDown         = 0x0100
	wmChar            = 0x0102
	wmLButtonUp       = 0x0202
	wmLButtonDblClk   = 0x0203
	wmNCLButtonDblClk = 0x00A3
	wmRButtonUp       = 0x0205
	wmNCRButtonUp     = 0x00A5
	wmMouseWheel      = 0x020A
	htClient          = 1
	htCaption         = 2
	vkReturn          = 0x0D
	vkEscape          = 0x1B

	mfString     = uintptr(0x00000000)
	tpmRight     = uintptr(0x0002)
	tpmReturnCmd = uintptr(0x0100)
	menuHead     = uintptr(1001)
	menuFull     = uintptr(1002)
	menuRestore  = uintptr(1003)
	menuQuit     = uintptr(1004)
	menuChat     = uintptr(1005)

	ulwAlpha   = uintptr(0x00000002)
	acSrcOver  = 0x00
	acSrcAlpha = 0x01

	dibRGBColors = uintptr(0)
	biRGB        = uint32(0)

	bkTransparent = uintptr(1)
	dtLeft        = uint32(0x00000000)
	dtCenter      = uint32(0x00000001)
	dtTop         = uint32(0x00000000)
	dtVCenter     = uint32(0x00000004)
	dtWordBreak   = uint32(0x00000010)
	dtSingleLine  = uint32(0x00000020)
	dtEndEllipsis = uint32(0x00008000)

	overlayPanelWidth  = 360
	overlayPanelHeight = 326
	overlayBubbleWidth = 224
	overlayGap         = 14
	overlayReplyLineH  = 24

	fontQualityAntialiased = uintptr(4)
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
	procDrawTextW           = overlayUser32.NewProc("DrawTextW")
	procGetModuleHandleW    = overlayKernel.NewProc("GetModuleHandleW")
	procCreateCompatibleDC  = overlayGDI32.NewProc("CreateCompatibleDC")
	procCreateDIBSection    = overlayGDI32.NewProc("CreateDIBSection")
	procSelectObject        = overlayGDI32.NewProc("SelectObject")
	procDeleteObject        = overlayGDI32.NewProc("DeleteObject")
	procDeleteDC            = overlayGDI32.NewProc("DeleteDC")
	procSetBkMode           = overlayGDI32.NewProc("SetBkMode")
	procSetTextColor        = overlayGDI32.NewProc("SetTextColor")
	procCreateFontW         = overlayGDI32.NewProc("CreateFontW")
	procGdiFlush            = overlayGDI32.NewProc("GdiFlush")

	avatarOverlayWndProc = syscall.NewCallback(avatarOverlayWindowProc)

	overlayMu        sync.Mutex
	overlayHWND      uintptr
	overlayAction    func(string, string)
	overlayMode      string
	overlayChatMode  bool
	overlayClassOnce sync.Once
	overlayClassErr  error
	overlayState     = floatingAvatarOverlayState{
		PersonaName: "厭世大叔",
		Placeholder: "輸入想聊的內容，只暫存最近 30 條，關掉或切換人格就忘記。",
	}
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

func (r overlayRect) contains(x int, y int) bool {
	if r.Right <= r.Left || r.Bottom <= r.Top {
		return false
	}
	return int32(x) >= r.Left && int32(x) <= r.Right && int32(y) >= r.Top && int32(y) <= r.Bottom
}

func (r overlayRect) isEmpty() bool {
	return r.Right <= r.Left || r.Bottom <= r.Top
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

type floatingAvatarOverlayState struct {
	Avatar          image.Image
	Mode            string
	AvatarScreenX   int
	AvatarScreenY   int
	AvatarLocalX    int
	AvatarLocalY    int
	PanelOpen       bool
	Draft           string
	ReplyText       string
	PersonaName     string
	Placeholder     string
	CloseRect       overlayRect
	PhotoRect       overlayRect
	ReplyRect       overlayRect
	SubmitRect      overlayRect
	TextRect        overlayRect
	ReplyScroll     int
	ReplyMaxScroll  int
	LastWindowLeft  int
	LastWindowTop   int
	LastWindowWidth int
	LastWindowH     int
}

type overlayTextRun struct {
	Text    string
	Rect    overlayRect
	Color   uint32
	Size    int32
	Weight  int32
	Center  bool
	VCenter bool
}

type overlayRender struct {
	Image          image.Image
	TextRuns       []overlayTextRun
	WindowX        int
	WindowY        int
	AvatarLocalX   int
	AvatarLocalY   int
	CloseRect      overlayRect
	PhotoRect      overlayRect
	ReplyRect      overlayRect
	SubmitRect     overlayRect
	TextRect       overlayRect
	ReplyMaxScroll int
}

func showFloatingAvatarOverlay(imagePath string, mode string, x int, y int, maxW int, maxH int, onAction func(string, string)) error {
	img, err := loadOverlayImage(imagePath, mode, maxW, maxH)
	if err != nil {
		return err
	}
	closeFloatingAvatarOverlay()

	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		runFloatingAvatarOverlay(img, mode, x, y, onAction, ready)
	}()
	return <-ready
}

func closeFloatingAvatarOverlay() {
	overlayMu.Lock()
	hwnd := overlayHWND
	overlayMu.Unlock()
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			overlayMu.Lock()
			closed := overlayHWND == 0
			overlayMu.Unlock()
			if closed {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func floatingAvatarOverlayPosition() (int, int) {
	overlayMu.Lock()
	hwnd := overlayHWND
	localX := overlayState.AvatarLocalX
	localY := overlayState.AvatarLocalY
	overlayMu.Unlock()
	if hwnd == 0 {
		return 0, 0
	}
	var rect overlayRect
	ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 {
		return 0, 0
	}
	return int(rect.Left) + localX, int(rect.Top) + localY
}

func setFloatingAvatarOverlayChatMode(enabled bool) {
	overlayMu.Lock()
	overlayChatMode = enabled
	if !enabled {
		overlayState.PanelOpen = false
		overlayState.Draft = ""
		overlayState.ReplyText = ""
		overlayState.ReplyScroll = 0
	}
	overlayMu.Unlock()
	redrawFloatingAvatarOverlay()
}

func setFloatingAvatarOverlayMetadata(personaName string, replyText string, placeholder string) {
	overlayMu.Lock()
	if strings.TrimSpace(personaName) != "" {
		overlayState.PersonaName = strings.TrimSpace(personaName)
	}
	nextReply := strings.TrimSpace(replyText)
	if nextReply != overlayState.ReplyText {
		overlayState.ReplyScroll = 0
	}
	overlayState.ReplyText = nextReply
	if strings.TrimSpace(placeholder) != "" {
		overlayState.Placeholder = strings.TrimSpace(placeholder)
	}
	overlayMu.Unlock()
	redrawFloatingAvatarOverlay()
}

func runFloatingAvatarOverlay(img image.Image, mode string, x int, y int, onAction func(string, string), ready chan<- error) {
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
	overlayMode = normalizeOverlayMode(mode)
	overlayState.Avatar = img
	overlayState.Mode = normalizeOverlayMode(mode)
	overlayState.AvatarScreenX = x
	overlayState.AvatarScreenY = y
	overlayMu.Unlock()

	render := composeFloatingAvatarOverlayRender()
	if err := updateOverlayBitmap(hwnd, render.Image, render.WindowX, render.WindowY, render.TextRuns); err != nil {
		procDestroyWindow.Call(hwnd)
		ready <- err
		return
	}
	storeOverlayRender(render)
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
		return overlayHitTest(hwnd, lParam)
	case wmLButtonDblClk, wmNCLButtonDblClk:
		openNativeOverlayChatPanel()
		emitOverlayAction("chat_on", "")
		return 0
	case wmLButtonUp:
		x := int(int16(lParam & 0xffff))
		y := int(int16((lParam >> 16) & 0xffff))
		handleOverlayClick(x, y)
		return 0
	case wmMouseWheel:
		handleOverlayMouseWheel(hwnd, wParam, lParam)
		return 0
	case wmKeyDown:
		if wParam == vkReturn {
			submitOverlayDraft()
			return 0
		}
		if wParam == vkEscape {
			closeNativeOverlayChatPanel()
			return 0
		}
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	case wmChar:
		handleOverlayChar(rune(wParam))
		return 0
	case wmRButtonUp, wmNCRButtonUp:
		action := trackOverlayMenuWithState(hwnd)
		if action == "restore" {
			emitOverlayAction(action, "")
			procDestroyWindow.Call(hwnd)
		} else if action != "" {
			emitOverlayAction(action, "")
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

func emitOverlayAction(action string, text string) {
	overlayMu.Lock()
	callback := overlayAction
	overlayMu.Unlock()
	if callback != nil {
		go callback(action, text)
	}
}

func overlayHitTest(hwnd uintptr, lParam uintptr) uintptr {
	overlayMu.Lock()
	panelOpen := overlayState.PanelOpen
	closeRect := overlayState.CloseRect
	photoRect := overlayState.PhotoRect
	replyRect := overlayState.ReplyRect
	submitRect := overlayState.SubmitRect
	textRect := overlayState.TextRect
	replyMaxScroll := overlayState.ReplyMaxScroll
	overlayMu.Unlock()
	x, y := overlayLocalPointFromLParam(hwnd, lParam)
	if replyMaxScroll > 0 && replyRect.contains(x, y) {
		return htClient
	}
	if !panelOpen {
		return htCaption
	}
	if closeRect.contains(x, y) || photoRect.contains(x, y) || submitRect.contains(x, y) || textRect.contains(x, y) {
		return htClient
	}
	return htCaption
}

func overlayLocalPointFromLParam(hwnd uintptr, lParam uintptr) (int, int) {
	screenX := int(int16(lParam & 0xffff))
	screenY := int(int16((lParam >> 16) & 0xffff))
	var rect overlayRect
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ok == 0 {
		return screenX, screenY
	}
	return screenX - int(rect.Left), screenY - int(rect.Top)
}

func trackOverlayMenuWithState(hwnd uintptr) string {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return ""
	}
	defer procDestroyMenu.Call(menu)
	overlayMu.Lock()
	currentMode := overlayMode
	chatActive := overlayChatMode
	overlayMu.Unlock()
	appendOverlayMenuItem(menu, menuHead, checkedOverlayLabel(currentMode == "head", "\u982d\u50cf"))
	appendOverlayMenuItem(menu, menuFull, checkedOverlayLabel(currentMode == "full", "\u5168\u8eab"))
	appendOverlayMenuItem(menu, menuChat, checkedOverlayLabel(chatActive, "\u9592\u804a\u6a21\u5f0f"))
	appendOverlayMenuItem(menu, menuRestore, "\u6574\u500b UI")
	appendOverlayMenuItem(menu, menuQuit, "\u95dc\u9589")

	var pt overlayPoint
	procOverlayGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmRight|tpmReturnCmd, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	switch cmd {
	case menuHead:
		return "head"
	case menuFull:
		return "full"
	case menuChat:
		return "chat"
	case menuRestore:
		return "restore"
	case menuQuit:
		return "quit"
	default:
		return ""
	}
}

func checkedOverlayLabel(active bool, label string) string {
	if active {
		return "\u2713 " + label
	}
	return label
}

func trackOverlayMenu(hwnd uintptr) string {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return ""
	}
	defer procDestroyMenu.Call(menu)
	appendOverlayMenuItem(menu, menuHead, "頭像")
	appendOverlayMenuItem(menu, menuFull, "全身像")
	appendOverlayMenuItem(menu, menuChat, "閒聊模式")
	appendOverlayMenuItem(menu, menuRestore, "回到介面")
	appendOverlayMenuItem(menu, menuQuit, "關閉程式")

	var pt overlayPoint
	procOverlayGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmRight|tpmReturnCmd, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	switch cmd {
	case menuHead:
		return "head"
	case menuFull:
		return "full"
	case menuChat:
		return "chat"
	case menuRestore:
		return "restore"
	case menuQuit:
		return "quit"
	default:
		return ""
	}
}

func appendOverlayMenuItem(menu uintptr, id uintptr, label string) {
	text, _ := syscall.UTF16PtrFromString(label)
	procAppendMenuW.Call(menu, mfString, id, uintptr(unsafe.Pointer(text)))
}

func openNativeOverlayChatPanel() {
	overlayMu.Lock()
	overlayChatMode = true
	overlayState.PanelOpen = true
	overlayMu.Unlock()
	redrawFloatingAvatarOverlay()
}

func closeNativeOverlayChatPanel() {
	overlayMu.Lock()
	overlayState.PanelOpen = false
	overlayState.Draft = ""
	overlayMu.Unlock()
	redrawFloatingAvatarOverlay()
}

func handleOverlayClick(x int, y int) {
	overlayMu.Lock()
	panelOpen := overlayState.PanelOpen
	closeRect := overlayState.CloseRect
	photoRect := overlayState.PhotoRect
	submitRect := overlayState.SubmitRect
	textRect := overlayState.TextRect
	overlayMu.Unlock()
	if !panelOpen {
		return
	}
	if closeRect.contains(x, y) {
		closeNativeOverlayChatPanel()
		return
	}
	if photoRect.contains(x, y) {
		emitOverlayAction("photo", "")
		return
	}
	if submitRect.contains(x, y) {
		submitOverlayDraft()
		return
	}
	if textRect.contains(x, y) {
		procSetForegroundWindow.Call(overlayHWND)
	}
}

func handleOverlayMouseWheel(hwnd uintptr, wParam uintptr, lParam uintptr) {
	x, y := overlayLocalPointFromLParam(hwnd, lParam)
	delta := int(int16((wParam >> 16) & 0xffff))
	if delta == 0 {
		return
	}
	steps := delta / 120
	if steps == 0 {
		if delta > 0 {
			steps = 1
		} else {
			steps = -1
		}
	}
	overlayMu.Lock()
	replyRect := overlayState.ReplyRect
	maxScroll := overlayState.ReplyMaxScroll
	if maxScroll <= 0 || !replyRect.contains(x, y) {
		overlayMu.Unlock()
		return
	}
	next := clampInt(overlayState.ReplyScroll-steps, 0, maxScroll)
	if next == overlayState.ReplyScroll {
		overlayMu.Unlock()
		return
	}
	overlayState.ReplyScroll = next
	overlayMu.Unlock()
	redrawFloatingAvatarOverlay()
}

func handleOverlayChar(ch rune) {
	if ch == '\r' || ch == '\n' {
		submitOverlayDraft()
		return
	}
	overlayMu.Lock()
	panelOpen := overlayState.PanelOpen
	if !panelOpen {
		overlayMu.Unlock()
		return
	}
	switch ch {
	case '\b':
		overlayState.Draft = trimLastRune(overlayState.Draft)
	case 0x1b:
		overlayState.PanelOpen = false
		overlayState.Draft = ""
		overlayState.ReplyScroll = 0
	default:
		if ch >= 0x20 && len([]rune(overlayState.Draft)) < 300 {
			overlayState.Draft += string(ch)
		}
	}
	overlayMu.Unlock()
	redrawFloatingAvatarOverlay()
}

func submitOverlayDraft() {
	overlayMu.Lock()
	text := strings.TrimSpace(overlayState.Draft)
	if text != "" {
		overlayState.Draft = ""
	}
	overlayMu.Unlock()
	if text == "" {
		redrawFloatingAvatarOverlay()
		return
	}
	redrawFloatingAvatarOverlay()
	emitOverlayAction("chat_submit", text)
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func redrawFloatingAvatarOverlay() {
	overlayMu.Lock()
	hwnd := overlayHWND
	overlayMu.Unlock()
	if hwnd == 0 {
		return
	}
	refreshOverlayAvatarScreenPosition(hwnd)
	render := composeFloatingAvatarOverlayRender()
	if render.Image == nil {
		return
	}
	if err := updateOverlayBitmap(hwnd, render.Image, render.WindowX, render.WindowY, render.TextRuns); err == nil {
		storeOverlayRender(render)
	}
}

func storeOverlayRender(render overlayRender) {
	bounds := render.Image.Bounds()
	overlayMu.Lock()
	overlayState.AvatarLocalX = render.AvatarLocalX
	overlayState.AvatarLocalY = render.AvatarLocalY
	overlayState.CloseRect = render.CloseRect
	overlayState.PhotoRect = render.PhotoRect
	overlayState.ReplyRect = render.ReplyRect
	overlayState.SubmitRect = render.SubmitRect
	overlayState.TextRect = render.TextRect
	overlayState.ReplyMaxScroll = render.ReplyMaxScroll
	overlayState.LastWindowLeft = render.WindowX
	overlayState.LastWindowTop = render.WindowY
	overlayState.LastWindowWidth = bounds.Dx()
	overlayState.LastWindowH = bounds.Dy()
	overlayMu.Unlock()
}

func refreshOverlayAvatarScreenPosition(hwnd uintptr) {
	var rect overlayRect
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ok == 0 {
		return
	}
	overlayMu.Lock()
	overlayState.AvatarScreenX = int(rect.Left) + overlayState.AvatarLocalX
	overlayState.AvatarScreenY = int(rect.Top) + overlayState.AvatarLocalY
	overlayMu.Unlock()
}

func composeFloatingAvatarOverlayRender() overlayRender {
	overlayMu.Lock()
	state := overlayState
	chatActive := overlayChatMode
	overlayMu.Unlock()
	avatar := state.Avatar
	if avatar == nil {
		return overlayRender{}
	}
	avatarBounds := avatar.Bounds()
	avatarW := avatarBounds.Dx()
	avatarH := avatarBounds.Dy()
	replyText := strings.TrimSpace(state.ReplyText)
	panelOpen := state.PanelOpen
	if !panelOpen && replyText == "" {
		return overlayRender{
			Image:        avatar,
			WindowX:      state.AvatarScreenX,
			WindowY:      state.AvatarScreenY,
			AvatarLocalX: 0,
			AvatarLocalY: 0,
		}
	}

	leftW := overlayBubbleWidth
	if replyText == "" {
		leftW = 0
	}
	rightW := 0
	if panelOpen {
		rightW = overlayPanelWidth
	}
	canvasW := leftW + avatarW + rightW + 28
	if leftW > 0 {
		canvasW += overlayGap
	}
	if rightW > 0 {
		canvasW += overlayGap
	}
	canvasH := max(avatarH+40, overlayPanelHeight+24)
	if !panelOpen {
		canvasH = max(avatarH+28, 108)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	avatarX := 8
	if leftW > 0 {
		avatarX += leftW + overlayGap
	}
	avatarY := max(12, (canvasH-avatarH)/2)
	if state.Mode == "head" && panelOpen {
		avatarY = 96
	}
	if state.Mode == "full" {
		avatarY = max(8, (canvasH-avatarH)/2)
	}
	draw.Draw(canvas, image.Rect(avatarX, avatarY, avatarX+avatarW, avatarY+avatarH), avatar, avatarBounds.Min, draw.Over)

	textRuns := []overlayTextRun{}
	var replyRect overlayRect
	replyMaxScroll := 0
	if replyText != "" {
		bubbleH := min(116, max(84, 42+estimateTextLines(replyText, 14)*overlayReplyLineH))
		bubbleY := max(8, avatarY+4)
		bubble := image.Rect(8, bubbleY, 8+overlayBubbleWidth, bubbleY+bubbleH)
		drawRoundedRect(canvas, bubble, 7, color.RGBA{R: 20, G: 14, B: 7, A: 255})
		drawRoundedRectOutline(canvas, bubble, 7, color.RGBA{R: 205, G: 142, B: 39, A: 220})
		textRect := insetRect(bubble, 12, 13)
		visibleLines := overlayVisibleTextLines(textRect, overlayReplyLineH)
		visibleReply, maxScroll, scroll := scrollableOverlayText(replyText, 14, visibleLines, state.ReplyScroll)
		replyMaxScroll = max(replyMaxScroll, maxScroll)
		drawOverlayScrollThumb(canvas, bubble, scroll, maxScroll, visibleLines)
		replyRect = rect(bubble.Min.X, bubble.Min.Y, bubble.Max.X, bubble.Max.Y)
		textRuns = append(textRuns, overlayTextRun{
			Text:  visibleReply,
			Rect:  textRect,
			Color: rgb(255, 248, 239),
			Size:  15,
		})
	}

	var closeRect, photoRect, submitRect, textRect overlayRect
	if panelOpen {
		panelX := avatarX + avatarW + overlayGap
		panelY := 18
		panel := image.Rect(panelX, panelY, panelX+overlayPanelWidth, panelY+overlayPanelHeight)
		drawRoundedRect(canvas, panel, 8, color.RGBA{R: 10, G: 7, B: 5, A: 255})
		drawRoundedRectOutline(canvas, panel, 8, color.RGBA{R: 179, G: 109, B: 23, A: 230})
		closeRect = rect(panel.Max.X-33, panel.Min.Y+12, panel.Max.X-10, panel.Min.Y+35)
		drawRoundedRect(canvas, image.Rect(int(closeRect.Left), int(closeRect.Top), int(closeRect.Right), int(closeRect.Bottom)), 11, color.RGBA{R: 69, G: 45, B: 14, A: 255})
		drawRoundedRectOutline(canvas, image.Rect(int(closeRect.Left), int(closeRect.Top), int(closeRect.Right), int(closeRect.Bottom)), 11, color.RGBA{R: 234, G: 180, B: 85, A: 230})
		textRuns = append(textRuns,
			overlayTextRun{Text: "閒聊模式", Rect: rect(panelX+14, panelY+16, panelX+126, panelY+38), Color: rgb(160, 145, 124), Size: 13, Weight: 500},
			overlayTextRun{Text: safeOverlayTitle(state.PersonaName), Rect: rect(panelX+14, panelY+39, panelX+244, panelY+70), Color: rgb(255, 186, 85), Size: 19, Weight: 700},
			overlayTextRun{Text: "×", Rect: closeRect, Color: rgb(255, 225, 169), Size: 19, Weight: 500, Center: true, VCenter: true},
		)
		latest := replyText
		if latest == "" {
			latest = "主人今天好嗎？"
		}
		quote := image.Rect(panelX+12, panelY+74, panelX+overlayPanelWidth-14, panelY+144)
		drawRoundedRect(canvas, quote, 7, color.RGBA{R: 33, G: 23, B: 13, A: 255})
		quoteTextRect := rect(quote.Min.X+12, quote.Min.Y+38, quote.Max.X-14, quote.Max.Y-10)
		visibleLines := overlayVisibleTextLines(quoteTextRect, overlayReplyLineH)
		visibleLatest, maxScroll, scroll := scrollableOverlayText(latest, 22, visibleLines, state.ReplyScroll)
		replyMaxScroll = max(replyMaxScroll, maxScroll)
		drawOverlayScrollThumb(canvas, quote, scroll, maxScroll, visibleLines)
		if replyRect.isEmpty() {
			replyRect = rect(quote.Min.X, quote.Min.Y, quote.Max.X, quote.Max.Y)
		}
		textRuns = append(textRuns,
			overlayTextRun{Text: safeOverlayTitle(state.PersonaName) + "：", Rect: rect(quote.Min.X+12, quote.Min.Y+12, quote.Max.X-10, quote.Min.Y+36), Color: rgb(110, 215, 207), Size: 17, Weight: 700},
			overlayTextRun{Text: visibleLatest, Rect: quoteTextRect, Color: rgb(255, 248, 239), Size: 15},
		)
		input := image.Rect(panelX+12, panelY+156, panelX+overlayPanelWidth-14, panelY+260)
		drawRoundedRect(canvas, input, 7, color.RGBA{R: 8, G: 8, B: 8, A: 255})
		drawRoundedRectOutline(canvas, input, 7, color.RGBA{R: 81, G: 61, B: 45, A: 240})
		draft := state.Draft
		if strings.TrimSpace(draft) == "" {
			draft = state.Placeholder
			if strings.TrimSpace(draft) == "" {
				draft = "輸入想聊的內容，只暫存最近 30 條，關掉或切換人格就忘記。"
			}
			textRuns = append(textRuns, overlayTextRun{Text: draft, Rect: insetRect(input, 12, 14), Color: rgb(150, 139, 125), Size: 15})
		} else {
			textRuns = append(textRuns, overlayTextRun{Text: draft, Rect: insetRect(input, 12, 14), Color: rgb(255, 248, 239), Size: 16})
		}
		textRect = rect(input.Min.X, input.Min.Y, input.Max.X, input.Max.Y)
		photo := image.Rect(panelX+overlayPanelWidth-156, panelY+270, panelX+overlayPanelWidth-88, panelY+306)
		drawRoundedRect(canvas, photo, 9, color.RGBA{R: 30, G: 39, B: 38, A: 255})
		drawRoundedRectOutline(canvas, photo, 9, color.RGBA{R: 110, G: 215, B: 207, A: 220})
		photoRect = rect(photo.Min.X, photo.Min.Y, photo.Max.X, photo.Max.Y)
		textRuns = append(textRuns, overlayTextRun{Text: "拍照", Rect: photoRect, Color: rgb(214, 255, 249), Size: 16, Weight: 700, Center: true, VCenter: true})
		submit := image.Rect(panelX+overlayPanelWidth-80, panelY+270, panelX+overlayPanelWidth-14, panelY+306)
		drawRoundedRect(canvas, submit, 9, color.RGBA{R: 83, G: 49, B: 10, A: 255})
		drawRoundedRectOutline(canvas, submit, 9, color.RGBA{R: 230, G: 158, B: 52, A: 255})
		submitRect = rect(submit.Min.X, submit.Min.Y, submit.Max.X, submit.Max.Y)
		textRuns = append(textRuns, overlayTextRun{Text: "送出", Rect: submitRect, Color: rgb(255, 248, 239), Size: 16, Weight: 700, Center: true, VCenter: true})
		if chatActive {
			badge := image.Rect(panelX+overlayPanelWidth-124, panelY+13, panelX+overlayPanelWidth-42, panelY+37)
			drawRoundedRect(canvas, badge, 11, color.RGBA{R: 41, G: 26, B: 8, A: 255})
			textRuns = append(textRuns, overlayTextRun{Text: "✓ 閒聊", Rect: rect(badge.Min.X, badge.Min.Y, badge.Max.X, badge.Max.Y), Color: rgb(255, 210, 129), Size: 14, Weight: 600, Center: true, VCenter: true})
		}
	}
	windowX := state.AvatarScreenX - avatarX
	windowY := state.AvatarScreenY - avatarY
	return overlayRender{
		Image:          canvas,
		TextRuns:       textRuns,
		WindowX:        windowX,
		WindowY:        windowY,
		AvatarLocalX:   avatarX,
		AvatarLocalY:   avatarY,
		CloseRect:      closeRect,
		PhotoRect:      photoRect,
		ReplyRect:      replyRect,
		SubmitRect:     submitRect,
		TextRect:       textRect,
		ReplyMaxScroll: replyMaxScroll,
	}
}

func updateOverlayBitmap(hwnd uintptr, img image.Image, x int, y int, textRuns []overlayTextRun) error {
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
	if len(textRuns) > 0 {
		// GDI 文字會把畫到的像素 alpha 寫成 0，UpdateLayeredWindow（AC_SRC_ALPHA）
		// 會把這些像素與桌面「加色混合」，文字看起來半透明又模糊。
		// 先快照像素，畫完字後把「被 GDI 改過的像素」alpha 修回 255
		//（文字都畫在不透明面板/氣泡上，直接設 255 是正確的）。
		snapshot := snapshotOverlayPixels(bits, width*height*4)
		drawOverlayTextRuns(memDC, textRuns)
		procGdiFlush.Call() // 確保 GDI 批次繪圖已寫入 DIB，才能讀 bits 修 alpha
		repairOverlayTextAlpha(bits, snapshot)
	}

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

// snapshotOverlayPixels 複製 DIB 像素，供文字繪製後比對哪些像素被 GDI 改過。
func snapshotOverlayPixels(bits unsafe.Pointer, size int) []byte {
	src := unsafe.Slice((*byte)(bits), size)
	dup := make([]byte, size)
	copy(dup, src)
	return dup
}

// repairOverlayTextAlpha 把 GDI DrawText 動過的像素 alpha 修回 255。
// GDI 不寫 alpha（會歸零），若不修，layered window 會把文字像素當半透明
// 與桌面混色，造成文字發虛、模糊（小字尤其明顯）。
func repairOverlayTextAlpha(bits unsafe.Pointer, snapshot []byte) {
	px := unsafe.Slice((*byte)(bits), len(snapshot))
	for i := 0; i+3 < len(px); i += 4 {
		if px[i] != snapshot[i] || px[i+1] != snapshot[i+1] || px[i+2] != snapshot[i+2] {
			px[i+3] = 255
		}
	}
}

func drawOverlayTextRuns(memDC uintptr, runs []overlayTextRun) {
	fontName, _ := syscall.UTF16PtrFromString("Microsoft JhengHei UI")
	procSetBkMode.Call(memDC, bkTransparent)
	for _, run := range runs {
		if strings.TrimSpace(run.Text) == "" {
			continue
		}
		weight := run.Weight
		if weight == 0 {
			weight = 400
		}
		height := -run.Size
		if height == 0 {
			height = -14
		}
		font, _, _ := procCreateFontW.Call(
			uintptr(height), 0, 0, 0, uintptr(weight),
			0, 0, 0, 1, 0, 0, fontQualityAntialiased, 0,
			uintptr(unsafe.Pointer(fontName)),
		)
		oldFont := uintptr(0)
		if font != 0 {
			oldFont, _, _ = procSelectObject.Call(memDC, font)
		}
		procSetTextColor.Call(memDC, uintptr(run.Color))
		text, _ := syscall.UTF16PtrFromString(run.Text)
		rc := run.Rect
		flags := dtLeft | dtTop | dtWordBreak
		if run.Center {
			flags &^= dtLeft
			flags |= dtCenter
		}
		if run.VCenter {
			flags &^= dtTop | dtWordBreak
			flags |= dtVCenter | dtSingleLine | dtEndEllipsis
		}
		procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(text)), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), uintptr(flags))
		if oldFont != 0 {
			procSelectObject.Call(memDC, oldFont)
		}
		if font != 0 {
			procDeleteObject.Call(font)
		}
	}
}

func rect(left int, top int, right int, bottom int) overlayRect {
	return overlayRect{Left: int32(left), Top: int32(top), Right: int32(right), Bottom: int32(bottom)}
}

func insetRect(r image.Rectangle, x int, y int) overlayRect {
	return rect(r.Min.X+x, r.Min.Y+y, r.Max.X-x, r.Max.Y-y)
}

func rgb(r byte, g byte, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func safeOverlayTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "厭世大叔"
	}
	runes := []rune(value)
	if len(runes) > 14 {
		return string(runes[:14])
	}
	return value
}

func estimateTextLines(text string, charsPerLine int) int {
	chars := len([]rune(strings.TrimSpace(text)))
	if chars == 0 || charsPerLine <= 0 {
		return 1
	}
	return max(1, (chars+charsPerLine-1)/charsPerLine)
}

func overlayVisibleTextLines(r overlayRect, lineHeight int) int {
	height := int(r.Bottom - r.Top)
	return max(1, (height-4)/max(1, lineHeight))
}

func scrollableOverlayText(text string, charsPerLine int, visibleLines int, scroll int) (string, int, int) {
	lines := wrapOverlayTextLines(text, charsPerLine)
	if len(lines) == 0 {
		return "", 0, 0
	}
	visibleLines = max(1, visibleLines)
	maxScroll := max(0, len(lines)-visibleLines)
	scroll = clampInt(scroll, 0, maxScroll)
	end := min(len(lines), scroll+visibleLines)
	return strings.Join(lines[scroll:end], "\n"), maxScroll, scroll
}

func wrapOverlayTextLines(text string, charsPerLine int) []string {
	charsPerLine = max(1, charsPerLine)
	lines := []string{}
	for _, rawLine := range strings.Split(strings.TrimSpace(text), "\n") {
		runes := []rune(strings.TrimSpace(rawLine))
		if len(runes) == 0 {
			lines = append(lines, "")
			continue
		}
		for len(runes) > 0 {
			n := min(charsPerLine, len(runes))
			lines = append(lines, string(runes[:n]))
			runes = runes[n:]
		}
	}
	return lines
}

func drawOverlayScrollThumb(img *image.RGBA, r image.Rectangle, scroll int, maxScroll int, visibleLines int) {
	if maxScroll <= 0 {
		return
	}
	track := image.Rect(r.Max.X-8, r.Min.Y+10, r.Max.X-5, r.Max.Y-10)
	if track.Dy() <= 8 {
		return
	}
	totalLines := max(visibleLines+maxScroll, 1)
	thumbH := max(16, track.Dy()*visibleLines/totalLines)
	thumbH = min(thumbH, track.Dy())
	travel := max(0, track.Dy()-thumbH)
	thumbY := track.Min.Y
	if maxScroll > 0 {
		thumbY += travel * clampInt(scroll, 0, maxScroll) / maxScroll
	}
	drawRoundedRect(img, image.Rect(track.Min.X, thumbY, track.Max.X, thumbY+thumbH), 2, color.RGBA{R: 255, G: 213, B: 143, A: 185})
}

func drawRoundedRect(img *image.RGBA, r image.Rectangle, radius int, c color.RGBA) {
	r = r.Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	radius = max(0, min(radius, min(r.Dx(), r.Dy())/2))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if roundedRectContains(x, y, r, radius) {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func drawRoundedRectOutline(img *image.RGBA, r image.Rectangle, radius int, c color.RGBA) {
	for i := 0; i < 1; i++ {
		drawRectOutline(img, r.Inset(i), radius, c)
	}
}

func drawRectOutline(img *image.RGBA, r image.Rectangle, radius int, c color.RGBA) {
	r = r.Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	inner := r.Inset(1)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if roundedRectContains(x, y, r, radius) && !roundedRectContains(x, y, inner, max(0, radius-1)) {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func roundedRectContains(x int, y int, r image.Rectangle, radius int) bool {
	if radius <= 0 {
		return image.Pt(x, y).In(r)
	}
	cx := x
	if x < r.Min.X+radius {
		cx = r.Min.X + radius
	} else if x >= r.Max.X-radius {
		cx = r.Max.X - radius - 1
	}
	cy := y
	if y < r.Min.Y+radius {
		cy = r.Min.Y + radius
	} else if y >= r.Max.Y-radius {
		cy = r.Max.Y - radius - 1
	}
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= radius*radius
}

func blendPixel(img *image.RGBA, x int, y int, c color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) || c.A == 0 {
		return
	}
	i := img.PixOffset(x, y)
	dstA := uint32(img.Pix[i+3])
	srcA := uint32(c.A)
	outA := srcA + dstA*(255-srcA)/255
	if outA == 0 {
		return
	}
	img.Pix[i+0] = byte((uint32(c.R)*srcA + uint32(img.Pix[i+0])*dstA*(255-srcA)/255) / outA)
	img.Pix[i+1] = byte((uint32(c.G)*srcA + uint32(img.Pix[i+1])*dstA*(255-srcA)/255) / outA)
	img.Pix[i+2] = byte((uint32(c.B)*srcA + uint32(img.Pix[i+2])*dstA*(255-srcA)/255) / outA)
	img.Pix[i+3] = byte(outA)
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

func loadOverlayImage(path string, mode string, maxW int, maxH int) (image.Image, error) {
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
		srcY := (float64(y)+0.5)*float64(height)/float64(nextH) - 0.5
		y0 := clampInt(int(srcY), 0, height-1)
		y1 := clampInt(y0+1, 0, height-1)
		fy := srcY - float64(y0)
		if fy < 0 {
			fy = 0
		}
		for x := 0; x < nextW; x++ {
			srcX := (float64(x)+0.5)*float64(width)/float64(nextW) - 0.5
			x0 := clampInt(int(srcX), 0, width-1)
			x1 := clampInt(x0+1, 0, width-1)
			fx := srcX - float64(x0)
			if fx < 0 {
				fx = 0
			}
			c00 := rgba8(img.At(bounds.Min.X+x0, bounds.Min.Y+y0))
			c10 := rgba8(img.At(bounds.Min.X+x1, bounds.Min.Y+y0))
			c01 := rgba8(img.At(bounds.Min.X+x0, bounds.Min.Y+y1))
			c11 := rgba8(img.At(bounds.Min.X+x1, bounds.Min.Y+y1))
			dst.SetRGBA(x, y, bilinearRGBA(c00, c10, c01, c11, fx, fy))
		}
	}
	return dst
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func rgba8(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: byte(r >> 8), G: byte(g >> 8), B: byte(b >> 8), A: byte(a >> 8)}
}

func bilinearRGBA(c00 color.RGBA, c10 color.RGBA, c01 color.RGBA, c11 color.RGBA, fx float64, fy float64) color.RGBA {
	return color.RGBA{
		R: byte(bilinearChannel(c00.R, c10.R, c01.R, c11.R, fx, fy)),
		G: byte(bilinearChannel(c00.G, c10.G, c01.G, c11.G, fx, fy)),
		B: byte(bilinearChannel(c00.B, c10.B, c01.B, c11.B, fx, fy)),
		A: byte(bilinearChannel(c00.A, c10.A, c01.A, c11.A, fx, fy)),
	}
}

func bilinearChannel(c00 byte, c10 byte, c01 byte, c11 byte, fx float64, fy float64) uint8 {
	top := float64(c00)*(1-fx) + float64(c10)*fx
	bottom := float64(c01)*(1-fx) + float64(c11)*fx
	value := top*(1-fy) + bottom*fy
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value + 0.5)
}
