//go:build darwin

package visual_learning

// macOS 原生點擊錄製器（對齊 Windows 版 native_input_windows.go 的契約）。
//
// 機制：Quartz CGEventTap 在 session 層級「監聽」全域滑鼠放開事件
//   （kCGEventLeftMouseUp / kCGEventRightMouseUp，listen-only 不攔截），
//   每次放開就用 CGWindowList 解析游標下最前面的外部視窗，組成
//   NativeClickEvent 交給 onClick。這讓使用者在 app 外部視窗的點擊
//   也能進入 learning trace。
//
// 權限：CGEventTap 需要「輔助使用 / Accessibility」授權
//   （系統設定 → 隱私權與安全性 → 輔助使用），部分機種還需「輸入監控
//   / Input Monitoring」。未授權時第一次 Start() 會請 macOS 顯示授權提示，
//   後續 Start() 只回明確錯誤 → 上層 emit native_recorder_degraded，避免
//   權限彈窗被反覆錄進示範流程。
//
// 執行緒：event tap 掛在某條 OS thread 的 CFRunLoop 上，故 Start() 內
//   以 runtime.LockOSThread() 鎖住一條 goroutine 跑 CFRunLoopRun()，
//   Stop() 由另一條 goroutine 呼叫 CFRunLoopStop() 喚醒結束。
//
// 限制：CaptureWindow 需要「螢幕錄製」權限；缺權限時 replay 會退回
//   原始螢幕座標點擊。

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework CoreGraphics

#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <dlfcn.h>
#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>

// Go 端匯出的回呼（cgo 產生對應 C symbol）。
extern void goNativeClickCallback(double x, double y, int button, uintptr_t handle);

typedef struct {
    CFMachPortRef      tap;
    CFRunLoopSourceRef source;
    CFRunLoopRef       runloop;
} NativeTap;

// 單一錄製器假設：保存 tap 供 timeout 後重新啟用。
static CFMachPortRef gTap = NULL;

static CGEventRef nativeTapCallback(CGEventTapProxy proxy, CGEventType type,
                                    CGEventRef event, void *refcon) {
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (gTap) CGEventTapEnable(gTap, true); // 被系統停用後自動恢復
        return event;
    }
    int button = 0; // 0 = left, 1 = right
    if (type == kCGEventRightMouseUp) button = 1;
    CGPoint loc = CGEventGetLocation(event);
    goNativeClickCallback((double)loc.x, (double)loc.y, button, (uintptr_t)refcon);
    return event; // listen-only：原樣放行，不影響使用者操作
}

// 檢查 Accessibility 授權，不彈出系統提示。回 1=已信任。
static int nativeCheckAccessibility(void) {
    return AXIsProcessTrusted() ? 1 : 0;
}

// 主動請求 Accessibility 授權（未授權時跳出系統提示）。回 1=已信任。
static int nativeRequestAccessibility(void) {
    CFStringRef keys[1]  = { kAXTrustedCheckOptionPrompt };
    CFBooleanRef vals[1] = { kCFBooleanTrue };
    CFDictionaryRef opts = CFDictionaryCreate(kCFAllocatorDefault,
        (const void **)keys, (const void **)vals, 1,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    Boolean trusted = AXIsProcessTrustedWithOptions(opts);
    CFRelease(opts);
    return trusted ? 1 : 0;
}

// 在「目前這條 thread」的 run loop 上建立並啟用 tap。
// 成功回 NativeTap*（malloc），失敗回 NULL（通常是缺權限）。非阻塞。
static NativeTap *nativeTapSetup(uintptr_t handle) {
    CGEventMask mask = CGEventMaskBit(kCGEventLeftMouseUp) |
                       CGEventMaskBit(kCGEventRightMouseUp);
    CFMachPortRef tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap,
                                         kCGEventTapOptionListenOnly, mask,
                                         nativeTapCallback, (void *)handle);
    if (!tap) return NULL;

    NativeTap *nt = (NativeTap *)calloc(1, sizeof(NativeTap));
    nt->tap = tap;
    nt->source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
    nt->runloop = CFRunLoopGetCurrent();
    CFRetain(nt->runloop);
    CFRunLoopAddSource(nt->runloop, nt->source, kCFRunLoopCommonModes);
    CGEventTapEnable(tap, true);
    gTap = tap;
    return nt;
}

// 阻塞跑目前 thread 的 run loop，直到 nativeTapStop 被呼叫。
static void nativeTapRunLoop(void) { CFRunLoopRun(); }

// 由另一條 thread 安全呼叫：請 run loop 停止並喚醒。
static void nativeTapStop(NativeTap *nt) {
    if (nt && nt->runloop) {
        CFRunLoopStop(nt->runloop);
        CFRunLoopWakeUp(nt->runloop);
    }
}

static void nativeTapTeardown(NativeTap *nt) {
    if (!nt) return;
    if (nt->tap) CGEventTapEnable(nt->tap, false);
    if (gTap == nt->tap) gTap = NULL;
    if (nt->source) {
        if (nt->runloop) CFRunLoopRemoveSource(nt->runloop, nt->source, kCFRunLoopCommonModes);
        CFRelease(nt->source);
    }
    if (nt->tap) CFRelease(nt->tap);
    if (nt->runloop) CFRelease(nt->runloop);
    free(nt);
}

typedef struct {
    int    found;
    int    pid;
    long   windowNumber;
    double x, y, w, h;
    char   owner[256];
    char   title[256];
} WinInfo;

static int nativeWindowInfoByNumber(long windowNumber, WinInfo *out) {
    memset(out, 0, sizeof(WinInfo));
    CFArrayRef list = CGWindowListCopyWindowInfo(
        kCGWindowListOptionIncludingWindow,
        (CGWindowID)windowNumber);
    if (!list) return 0;
    CFIndex count = CFArrayGetCount(list);
    if (count <= 0) {
        CFRelease(list);
        return 0;
    }
    CFDictionaryRef d = (CFDictionaryRef)CFArrayGetValueAtIndex(list, 0);
    out->found = 1;
    out->windowNumber = windowNumber;

    CFDictionaryRef boundsDict = (CFDictionaryRef)CFDictionaryGetValue(d, kCGWindowBounds);
    if (boundsDict) {
        CGRect r;
        if (CGRectMakeWithDictionaryRepresentation(boundsDict, &r)) {
            out->x = r.origin.x; out->y = r.origin.y;
            out->w = r.size.width; out->h = r.size.height;
        }
    }
    CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowOwnerPID);
    if (pidRef) { int pid = 0; CFNumberGetValue(pidRef, kCFNumberIntType, &pid); out->pid = pid; }
    CFStringRef owner = (CFStringRef)CFDictionaryGetValue(d, kCGWindowOwnerName);
    if (owner) CFStringGetCString(owner, out->owner, sizeof(out->owner), kCFStringEncodingUTF8);
    CFStringRef title = (CFStringRef)CFDictionaryGetValue(d, kCGWindowName);
    if (title) CFStringGetCString(title, out->title, sizeof(out->title), kCFStringEncodingUTF8);
    CFRelease(list);
    return 1;
}

// 找游標下、layer 0（一般視窗）、最前面、且 bounds 含點的視窗資訊。
static void nativeWindowAtPoint(double px, double py, WinInfo *out) {
    memset(out, 0, sizeof(WinInfo));
    CFArrayRef list = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID);
    if (!list) return;
    CFIndex count = CFArrayGetCount(list); // 前到後（z-order）
    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef d = (CFDictionaryRef)CFArrayGetValueAtIndex(list, i);

        int layer = 0;
        CFNumberRef layerRef = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowLayer);
        if (layerRef) CFNumberGetValue(layerRef, kCFNumberIntType, &layer);
        if (layer != 0) continue; // 略過選單列、Dock、浮層等

        CFDictionaryRef boundsDict = (CFDictionaryRef)CFDictionaryGetValue(d, kCGWindowBounds);
        if (!boundsDict) continue;
        CGRect r;
        if (!CGRectMakeWithDictionaryRepresentation(boundsDict, &r)) continue;
        if (px < r.origin.x || py < r.origin.y ||
            px > r.origin.x + r.size.width || py > r.origin.y + r.size.height) continue;

        out->found = 1;
        out->x = r.origin.x; out->y = r.origin.y;
        out->w = r.size.width; out->h = r.size.height;

        CFNumberRef num = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowNumber);
        if (num) { long wn = 0; CFNumberGetValue(num, kCFNumberLongType, &wn); out->windowNumber = wn; }
        CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowOwnerPID);
        if (pidRef) { int pid = 0; CFNumberGetValue(pidRef, kCFNumberIntType, &pid); out->pid = pid; }
        CFStringRef owner = (CFStringRef)CFDictionaryGetValue(d, kCGWindowOwnerName);
        if (owner) CFStringGetCString(owner, out->owner, sizeof(out->owner), kCFStringEncodingUTF8);
        CFStringRef title = (CFStringRef)CFDictionaryGetValue(d, kCGWindowName); // 需「螢幕錄製」權限才有值
        if (title) CFStringGetCString(title, out->title, sizeof(out->title), kCFStringEncodingUTF8);
        break;
    }
    CFRelease(list);
}

static void nativeMainDisplaySize(int *w, int *h) {
    CGDirectDisplayID d = CGMainDisplayID();
    *w = (int)CGDisplayPixelsWide(d);
    *h = (int)CGDisplayPixelsHigh(d);
}

static void nativePostMouseMove(double x, double y) {
    CGPoint p = CGPointMake(x, y);
    CGEventRef e = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, p, kCGMouseButtonLeft);
    if (!e) return;
    CGEventPost(kCGHIDEventTap, e);
    CFRelease(e);
}

static int nativeCurrentMouseLocation(double *x, double *y) {
    CGEventRef e = CGEventCreate(NULL);
    if (!e) return 0;
    CGPoint p = CGEventGetLocation(e);
    CFRelease(e);
    *x = p.x;
    *y = p.y;
    return 1;
}

static int nativePostMouseClick(double x, double y, int rightButton) {
    CGPoint p = CGPointMake(x, y);
    CGMouseButton button = rightButton ? kCGMouseButtonRight : kCGMouseButtonLeft;
    CGEventType downType = rightButton ? kCGEventRightMouseDown : kCGEventLeftMouseDown;
    CGEventType upType = rightButton ? kCGEventRightMouseUp : kCGEventLeftMouseUp;

    CGEventRef down = CGEventCreateMouseEvent(NULL, downType, p, button);
    CGEventRef up = CGEventCreateMouseEvent(NULL, upType, p, button);
    if (!down || !up) {
        if (down) CFRelease(down);
        if (up) CFRelease(up);
        return 0;
    }
    CGEventPost(kCGHIDEventTap, down);
    CFRelease(down);
    usleep(85000);
    CGEventPost(kCGHIDEventTap, up);
    CFRelease(up);
    return 1;
}

static int nativeCaptureWindowRGBA(long windowNumber, uint8_t **outData, int *outW, int *outH, WinInfo *outInfo) {
    *outData = NULL;
    *outW = 0;
    *outH = 0;
    if (outInfo) nativeWindowInfoByNumber(windowNumber, outInfo);

    typedef CGImageRef (*CreateWindowImageFn)(CGRect, CGWindowListOption, CGWindowID, CGWindowImageOption);
    CreateWindowImageFn createImage = (CreateWindowImageFn)dlsym(RTLD_DEFAULT, "CGWindowListCreateImage");
    if (!createImage) return 0;

    CGImageRef image = createImage(
        CGRectNull,
        kCGWindowListOptionIncludingWindow,
        (CGWindowID)windowNumber,
        kCGWindowImageBoundsIgnoreFraming);
    if (!image) return 0;

    size_t width = CGImageGetWidth(image);
    size_t height = CGImageGetHeight(image);
    if (width == 0 || height == 0 || width > 10000 || height > 10000) {
        CGImageRelease(image);
        return 0;
    }

    size_t bytesPerRow = width * 4;
    size_t total = bytesPerRow * height;
    uint8_t *data = (uint8_t *)calloc(total, 1);
    if (!data) {
        CGImageRelease(image);
        return 0;
    }

    CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
    CGContextRef ctx = CGBitmapContextCreate(
        data,
        width,
        height,
        8,
        bytesPerRow,
        cs,
        kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
    CGColorSpaceRelease(cs);
    if (!ctx) {
        free(data);
        CGImageRelease(image);
        return 0;
    }

    CGContextDrawImage(ctx, CGRectMake(0, 0, width, height), image);
    CGContextRelease(ctx);
    CGImageRelease(image);

    *outData = data;
    *outW = (int)width;
    *outH = (int)height;
    return 1;
}
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"runtime/cgo"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var nativeAccessibilityPromptShown atomic.Bool

// NativeInput records OS-level click input on macOS via a Quartz CGEventTap.
// Replay supports visual relocation when window capture is available, otherwise
// it falls back to screen-coordinate mouse move/click.
type NativeInput struct {
	mu      sync.Mutex
	onClick func(NativeClickEvent)
	handle  cgo.Handle
	tap     *C.NativeTap
	done    chan struct{}
	selfPID int
}

func NewNativeInput() *NativeInput {
	return &NativeInput{selfPID: os.Getpid()}
}

// Start 安裝 CGEventTap 並開始錄製外部視窗點擊。
// 若缺少 Accessibility 授權（CGEventTapCreate 回 NULL），回明確錯誤。
func (n *NativeInput) Start(onClick func(NativeClickEvent)) error {
	n.mu.Lock()
	if n.done != nil {
		n.mu.Unlock()
		return nil // 已在錄製
	}
	n.onClick = onClick
	done := make(chan struct{})
	n.done = done
	n.handle = cgo.NewHandle(n)
	n.mu.Unlock()

	ready := make(chan error, 1)
	go n.runLoop(ready, done)

	if err := <-ready; err != nil {
		n.mu.Lock()
		if n.done == done {
			n.done = nil
			if n.handle != 0 {
				n.handle.Delete()
				n.handle = 0
			}
			n.onClick = nil
		}
		n.mu.Unlock()
		return err
	}
	return nil
}

func (n *NativeInput) runLoop(ready chan<- error, done chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 只在本次 app 生命週期第一次缺授權時跳出系統提示；之後由 UI 顯示
	// native_recorder_degraded，避免權限視窗反覆干擾並被錄進示範。
	if C.nativeCheckAccessibility() == 0 && nativeAccessibilityPromptShown.CompareAndSwap(false, true) {
		C.nativeRequestAccessibility()
	}

	tap := C.nativeTapSetup(C.uintptr_t(n.handle))
	if tap == nil {
		ready <- fmt.Errorf("native input: CGEventTapCreate failed — grant Accessibility " +
			"(System Settings → Privacy & Security → Accessibility) and, if needed, Input Monitoring, then restart")
		n.closeDone(done)
		return
	}

	n.mu.Lock()
	n.tap = tap
	n.mu.Unlock()

	ready <- nil
	C.nativeTapRunLoop() // 阻塞到 Stop()

	C.nativeTapTeardown(tap)
	n.closeDone(done)
}

func (n *NativeInput) closeDone(done chan struct{}) {
	n.mu.Lock()
	if n.done == done {
		select {
		case <-done:
		default:
			close(done)
		}
	}
	n.mu.Unlock()
}

// Stop 停止 run loop 並拆除 tap。
func (n *NativeInput) Stop() error {
	n.mu.Lock()
	done := n.done
	tap := n.tap
	n.mu.Unlock()
	if done == nil {
		return nil
	}
	if tap != nil {
		C.nativeTapStop(tap)
	}
	timedOut := false
	select {
	case <-done:
	case <-time.After(1500 * time.Millisecond):
		timedOut = true
	}
	// 不論是否逾時都要重置狀態，否則下次 Start() 會誤判仍在錄製而直接
	// return nil（不建立新 tap），導致後續示範靜默錄不到任何 step。
	n.mu.Lock()
	if n.done == done {
		n.done = nil
		n.tap = nil
		n.onClick = nil
		if !timedOut && n.handle != 0 {
			n.handle.Delete()
			n.handle = 0
		}
		// 逾時時保留 handle：tap 可能仍存活，立即 Delete 會使後續回呼存取
		// 已刪除的 handle 而 panic。下次 Start() 會建立新 handle，舊的小幅
		// 洩漏在這條罕見路徑上可接受。
	}
	n.mu.Unlock()
	if timedOut {
		return fmt.Errorf("native input: recorder stop timed out")
	}
	return nil
}

// emitClick 由 C 回呼觸發（event tap thread）。解析視窗並送出事件。
func (n *NativeInput) emitClick(x, y, button int) {
	var info C.WinInfo
	C.nativeWindowAtPoint(C.double(x), C.double(y), &info)
	if info.found == 0 {
		return
	}
	if int(info.pid) == n.selfPID {
		return // 略過點到自己 app 的情況（app 內點擊由 WebView 處理）
	}

	owner := C.GoString(&info.owner[0])
	title := C.GoString(&info.title[0])
	if title == "" {
		title = owner
	}
	if isMacOSPermissionPromptWindow(owner, title) {
		return
	}

	var sw, sh C.int
	C.nativeMainDisplaySize(&sw, &sh)

	btn := "left"
	if button == 1 {
		btn = "right"
	}

	event := NativeClickEvent{
		Timestamp:     time.Now(),
		X:             x,
		Y:             y,
		Button:        btn,
		WindowTitle:   title,
		WindowProcess: owner,
		WindowHandle:  uintptr(info.windowNumber),
		ScreenX:       0,
		ScreenY:       0,
		ScreenWidth:   int(sw),
		ScreenHeight:  int(sh),
		WindowRect: PixelBBox{
			X: int(info.x),
			Y: int(info.y),
			W: int(info.w),
			H: int(info.h),
		},
	}

	n.mu.Lock()
	cb := n.onClick
	n.mu.Unlock()
	if cb != nil {
		go cb(event)
	}
}

// Click / MoveCursorOnly：以 macOS 全域螢幕座標重放。
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
	if C.nativeCheckAccessibility() == 0 {
		return NativeReplayResult{
			OK:            false,
			Method:        "native",
			Index:         step.Index,
			Label:         step.Label,
			X:             step.X,
			Y:             step.Y,
			Error:         "native input: Accessibility permission is required for macOS replay; grant it and restart the app",
			WindowTitle:   step.WindowTitle,
			WindowProcess: step.WindowProcess,
		}
	}
	n.moveCursorSmooth(step.X, step.Y)
	time.Sleep(450 * time.Millisecond)
	rightButton := 0
	if strings.EqualFold(step.Button, "right") || strings.EqualFold(step.Action, "right_click") {
		rightButton = 1
	}
	if C.nativePostMouseClick(C.double(step.X), C.double(step.Y), C.int(rightButton)) == 0 {
		return NativeReplayResult{
			OK:            false,
			Method:        "native",
			Index:         step.Index,
			Label:         step.Label,
			X:             step.X,
			Y:             step.Y,
			Error:         "native input: failed to create macOS mouse event",
			WindowTitle:   step.WindowTitle,
			WindowProcess: step.WindowProcess,
		}
	}
	time.Sleep(220 * time.Millisecond)
	return NativeReplayResult{
		OK:                true,
		Method:            "native",
		Index:             step.Index,
		Label:             step.Label,
		X:                 step.X,
		Y:                 step.Y,
		WindowTitle:       step.WindowTitle,
		WindowProcess:     step.WindowProcess,
		ForegroundOK:      false,
		ForegroundTitle:   step.WindowTitle,
		ForegroundProcess: step.WindowProcess,
		Warning:           "macOS native replay used recorded screen coordinates; foreground window was not independently verified",
	}
}

func (n *NativeInput) MoveCursorOnly(step LearningReplayStep) NativeReplayResult {
	if C.nativeCheckAccessibility() == 0 {
		return NativeReplayResult{
			OK:      false,
			Skipped: true,
			Method:  "native_preview",
			Index:   step.Index,
			Label:   step.Label,
			X:       step.X,
			Y:       step.Y,
			Error:   "native input: Accessibility permission is required for macOS replay preview; grant it and restart the app",
		}
	}
	n.moveCursorSmooth(step.X, step.Y)
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
	var data *C.uint8_t
	var width, height C.int
	var info C.WinInfo
	ok := C.nativeCaptureWindowRGBA(C.long(hwnd), &data, &width, &height, &info)
	if ok == 0 || data == nil || width <= 0 || height <= 0 {
		return WindowCapture{}, fmt.Errorf("native capture: CGWindowListCreateImage failed; grant Screen Recording permission if visual relocation is needed")
	}
	defer C.free(unsafe.Pointer(data))
	size := int(width) * int(height) * 4
	imageData := C.GoBytes(unsafe.Pointer(data), C.int(size))
	owner := C.GoString(&info.owner[0])
	title := C.GoString(&info.title[0])
	if title == "" {
		title = owner
	}
	return WindowCapture{
		ImageData: imageData,
		Width:     int(width),
		Height:    int(height),
		WindowRect: PixelBBox{
			X: int(info.x),
			Y: int(info.y),
			W: int(info.w),
			H: int(info.h),
		},
		WindowTitle:   title,
		WindowProcess: owner,
	}, nil
}

func (n *NativeInput) moveCursorSmooth(targetX, targetY int) {
	var sx, sy C.double
	if C.nativeCurrentMouseLocation(&sx, &sy) == 0 {
		C.nativePostMouseMove(C.double(targetX), C.double(targetY))
		return
	}
	startX := float64(sx)
	startY := float64(sy)
	if int(startX) == targetX && int(startY) == targetY {
		return
	}
	const steps = 28
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		eased := 1 - (1-t)*(1-t)
		x := startX + (float64(targetX)-startX)*eased
		y := startY + (float64(targetY)-startY)*eased
		C.nativePostMouseMove(C.double(x), C.double(y))
		time.Sleep(28 * time.Millisecond)
	}
	C.nativePostMouseMove(C.double(targetX), C.double(targetY))
}

func isMacOSPermissionPromptWindow(owner, title string) bool {
	text := strings.ToLower(strings.TrimSpace(owner + " " + title))
	return strings.Contains(text, "universalaccessauthwarn")
}

// handleToNativeInput 從 cgo.Handle 還原 *NativeInput（供匯出回呼使用）。
func handleToNativeInput(h uintptr) (*NativeInput, bool) {
	v := cgo.Handle(h).Value()
	n, ok := v.(*NativeInput)
	return n, ok
}
