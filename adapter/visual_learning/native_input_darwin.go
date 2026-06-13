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
// 權限（缺一不可，缺哪個都會給明確錯誤訊息）：
//   - 輔助使用 Accessibility：CGEventPost 回放點擊、AX 視窗 raise。
//   - 輸入監控 Input Monitoring：listen-only CGEventTap（macOS 10.15+）。
//     注意：只請求 Accessibility 不會帶出輸入監控授權，必須另外呼叫
//     CGRequestListenEventAccess()。
//   - 螢幕錄製 Screen Recording：CaptureWindow 視覺重定位 +
//     CGWindowList 的 kCGWindowName。缺權限時 replay 退回原始螢幕座標。
//
// 座標空間：CGEventTap / CGWindowList bounds 都是「point」；
//   截圖（SCK / CGWindowListCreateImage）是「pixel」（Retina 為 2x）。
//   WindowCapture.Scale 記錄 pixels-per-point，所有 anchor 計算統一在
//   capture pixel 空間，回放執行點再除以 Scale 還原為 point。
//
// 截圖：macOS 14+ 優先走 ScreenCaptureKit（screencapture_darwin.m），
//   舊系統 fallback 到 dlsym 的 CGWindowListCreateImage（14+ 已棄用）。
//
// 執行緒：event tap 掛在某條 OS thread 的 CFRunLoop 上，故 Start() 內
//   以 runtime.LockOSThread() 鎖住一條 goroutine 跑 CFRunLoopRun()，
//   Stop() 由另一條 goroutine 呼叫 CFRunLoopStop() 喚醒結束。

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation -framework CoreGraphics

#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <unistd.h>
#include <dlfcn.h>
#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include "screencapture_darwin.h"

// Go 端匯出的回呼（cgo 產生對應 C symbol）。
extern void goNativeClickCallback(double x, double y, int button, int clickCount, uintptr_t handle);

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
    int clickCount = (int)CGEventGetIntegerValueField(event, kCGMouseEventClickState);
    goNativeClickCallback((double)loc.x, (double)loc.y, button, clickCount, (uintptr_t)refcon);
    return event; // listen-only：原樣放行，不影響使用者操作
}

// ── 權限 ────────────────────────────────────────────────────────────

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

// 輸入監控（listen-only event tap 所需，10.15+）。回 1=已授權。
static int nativePreflightListenAccess(void) {
    if (@available(macOS 10.15, *)) {
        return CGPreflightListenEventAccess() ? 1 : 0;
    }
    return 1;
}

static int nativeRequestListenAccess(void) {
    if (@available(macOS 10.15, *)) {
        return CGRequestListenEventAccess() ? 1 : 0;
    }
    return 1;
}

// 螢幕錄製（截圖 / kCGWindowName 所需，11.0+ 公開 API）。回 1=已授權。
static int nativePreflightScreenCapture(void) {
    if (@available(macOS 11.0, *)) {
        return CGPreflightScreenCaptureAccess() ? 1 : 0;
    }
    return 1;
}

static int nativeRequestScreenCapture(void) {
    if (@available(macOS 11.0, *)) {
        return CGRequestScreenCaptureAccess() ? 1 : 0;
    }
    return 1;
}

// ── event tap 生命週期 ──────────────────────────────────────────────

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

// ── 視窗解析 ────────────────────────────────────────────────────────

typedef struct {
    int    found;
    int    pid;
    long   windowNumber;
    double x, y, w, h;
    char   owner[256];
    char   title[256];
} WinInfo;

static void nativeFillWinInfoFromDict(CFDictionaryRef d, WinInfo *out) {
    CFDictionaryRef boundsDict = (CFDictionaryRef)CFDictionaryGetValue(d, kCGWindowBounds);
    if (boundsDict) {
        CGRect r;
        if (CGRectMakeWithDictionaryRepresentation(boundsDict, &r)) {
            out->x = r.origin.x; out->y = r.origin.y;
            out->w = r.size.width; out->h = r.size.height;
        }
    }
    CFNumberRef num = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowNumber);
    if (num) { long wn = 0; CFNumberGetValue(num, kCFNumberLongType, &wn); out->windowNumber = wn; }
    CFNumberRef pidRef = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowOwnerPID);
    if (pidRef) { int pid = 0; CFNumberGetValue(pidRef, kCFNumberIntType, &pid); out->pid = pid; }
    CFStringRef owner = (CFStringRef)CFDictionaryGetValue(d, kCGWindowOwnerName);
    if (owner) CFStringGetCString(owner, out->owner, sizeof(out->owner), kCFStringEncodingUTF8);
    CFStringRef title = (CFStringRef)CFDictionaryGetValue(d, kCGWindowName); // 需「螢幕錄製」權限才有值
    if (title) CFStringGetCString(title, out->title, sizeof(out->title), kCFStringEncodingUTF8);
}

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
    nativeFillWinInfoFromDict(d, out);
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
        nativeFillWinInfoFromDict(d, out);
        break;
    }
    CFRelease(list);
}

// 以 process(owner) 名稱 + 視窗標題重新尋找視窗（錄到的 windowNumber 在目標
// app 重開後就失效，盲點舊座標前先試著找回正確視窗）。
// 評分：owner 相符必要；標題完全相符 > 前綴/包含 > 同 owner 第一個（最前面）。
static void nativeFindWindowByOwnerTitle(const char *ownerWanted, const char *titleWanted, WinInfo *out) {
    memset(out, 0, sizeof(WinInfo));
    if (!ownerWanted || !ownerWanted[0]) return;
    CFArrayRef list = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID);
    if (!list) return;
    CFIndex count = CFArrayGetCount(list);
    int bestScore = 0;
    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef d = (CFDictionaryRef)CFArrayGetValueAtIndex(list, i);
        int layer = 0;
        CFNumberRef layerRef = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowLayer);
        if (layerRef) CFNumberGetValue(layerRef, kCFNumberIntType, &layer);
        if (layer != 0) continue;

        WinInfo cur;
        memset(&cur, 0, sizeof(cur));
        nativeFillWinInfoFromDict(d, &cur);
        if (cur.w <= 1 || cur.h <= 1) continue;
        if (strcasecmp(cur.owner, ownerWanted) != 0) continue;

        int score = 1; // owner 相符
        if (titleWanted && titleWanted[0] && cur.title[0]) {
            if (strcasecmp(cur.title, titleWanted) == 0) {
                score = 4;
            } else if (strcasestr(cur.title, titleWanted) || strcasestr(titleWanted, cur.title)) {
                score = 3;
            }
        }
        if (score > bestScore) {
            bestScore = score;
            cur.found = 1;
            *out = cur;
            if (score >= 4) break;
        }
    }
    CFRelease(list);
}

// 最前面的 layer-0 視窗（用來驗證 activation 是否成功）。
static void nativeFrontmostWindow(WinInfo *out) {
    memset(out, 0, sizeof(WinInfo));
    CFArrayRef list = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID);
    if (!list) return;
    CFIndex count = CFArrayGetCount(list);
    for (CFIndex i = 0; i < count; i++) {
        CFDictionaryRef d = (CFDictionaryRef)CFArrayGetValueAtIndex(list, i);
        int layer = 0;
        CFNumberRef layerRef = (CFNumberRef)CFDictionaryGetValue(d, kCGWindowLayer);
        if (layerRef) CFNumberGetValue(layerRef, kCFNumberIntType, &layer);
        if (layer != 0) continue;
        out->found = 1;
        nativeFillWinInfoFromDict(d, out);
        break;
    }
    CFRelease(list);
}

// ── 視窗 activation（回放前把目標帶到前景；Windows 版的 SetForegroundWindow 對齊）──

// 以 AX API 把 pid 的 app 設為 frontmost，並 raise 與錄製視窗最匹配的視窗。
// 需要 Accessibility 授權。回 1=有嘗試（app element 建立成功），0=失敗。
static int nativeActivateWindow(int pid, double wx, double wy, double ww, double wh, const char *title) {
    if (pid <= 0) return 0;
    AXUIElementRef app = AXUIElementCreateApplication((pid_t)pid);
    if (!app) return 0;

    AXUIElementSetAttributeValue(app, kAXFrontmostAttribute, kCFBooleanTrue);

    CFArrayRef windows = NULL;
    if (AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef *)&windows) == kAXErrorSuccess && windows) {
        CFIndex count = CFArrayGetCount(windows);
        AXUIElementRef best = NULL;
        int bestScore = 0;
        for (CFIndex i = 0; i < count; i++) {
            AXUIElementRef win = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);
            int score = 1;
            // 標題比對
            if (title && title[0]) {
                CFTypeRef t = NULL;
                if (AXUIElementCopyAttributeValue(win, kAXTitleAttribute, &t) == kAXErrorSuccess && t) {
                    if (CFGetTypeID(t) == CFStringGetTypeID()) {
                        char buf[256] = {0};
                        if (CFStringGetCString((CFStringRef)t, buf, sizeof(buf), kCFStringEncodingUTF8)) {
                            if (strcasecmp(buf, title) == 0) score = 4;
                            else if (buf[0] && (strcasestr(buf, title) || strcasestr(title, buf))) score = 3;
                        }
                    }
                    CFRelease(t);
                }
            }
            // 幾何比對（位置/尺寸接近錄到的視窗 → 加分）
            if (score < 4 && ww > 0 && wh > 0) {
                CFTypeRef posRef = NULL, sizeRef = NULL;
                CGPoint pos = CGPointZero;
                CGSize size = CGSizeZero;
                int haveGeom = 0;
                if (AXUIElementCopyAttributeValue(win, kAXPositionAttribute, &posRef) == kAXErrorSuccess && posRef) {
                    if (AXValueGetValue((AXValueRef)posRef, kAXValueTypeCGPoint, &pos)) haveGeom++;
                    CFRelease(posRef);
                }
                if (AXUIElementCopyAttributeValue(win, kAXSizeAttribute, &sizeRef) == kAXErrorSuccess && sizeRef) {
                    if (AXValueGetValue((AXValueRef)sizeRef, kAXValueTypeCGSize, &size)) haveGeom++;
                    CFRelease(sizeRef);
                }
                if (haveGeom == 2 &&
                    pos.x > wx - 8 && pos.x < wx + 8 &&
                    pos.y > wy - 8 && pos.y < wy + 8 &&
                    size.width > ww - 8 && size.width < ww + 8 &&
                    size.height > wh - 8 && size.height < wh + 8) {
                    score = score > 2 ? score : 2;
                }
            }
            if (score > bestScore) {
                bestScore = score;
                best = win;
                if (score >= 4) break;
            }
        }
        if (best) {
            AXUIElementSetAttributeValue(best, kAXMainAttribute, kCFBooleanTrue);
            AXUIElementPerformAction(best, kAXRaiseAction);
        }
        CFRelease(windows);
    }
    CFRelease(app);
    return 1;
}

// ── 螢幕 / 滑鼠 ─────────────────────────────────────────────────────

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

// 點擊（clicks=1 單擊、2 雙擊）。雙擊必須在每個 down/up 事件設
// kCGMouseEventClickState，否則目標 app 只會看到兩次單擊。
static int nativePostMouseClick(double x, double y, int rightButton, int clicks) {
    if (clicks < 1) clicks = 1;
    if (clicks > 3) clicks = 3;
    CGPoint p = CGPointMake(x, y);
    CGMouseButton button = rightButton ? kCGMouseButtonRight : kCGMouseButtonLeft;
    CGEventType downType = rightButton ? kCGEventRightMouseDown : kCGEventLeftMouseDown;
    CGEventType upType = rightButton ? kCGEventRightMouseUp : kCGEventLeftMouseUp;

    for (int c = 1; c <= clicks; c++) {
        CGEventRef down = CGEventCreateMouseEvent(NULL, downType, p, button);
        CGEventRef up = CGEventCreateMouseEvent(NULL, upType, p, button);
        if (!down || !up) {
            if (down) CFRelease(down);
            if (up) CFRelease(up);
            return 0;
        }
        CGEventSetIntegerValueField(down, kCGMouseEventClickState, c);
        CGEventSetIntegerValueField(up, kCGMouseEventClickState, c);
        CGEventPost(kCGHIDEventTap, down);
        CFRelease(down);
        usleep(85000);
        CGEventPost(kCGHIDEventTap, up);
        CFRelease(up);
        if (c < clicks) usleep(60000); // 雙擊間隔需小於系統 double-click 門檻
    }
    return 1;
}

// ── 視窗截圖 ────────────────────────────────────────────────────────

// CGImage → RGBA bytes（premultiplied, big-endian RGBA8888）。
static int nativeImageToRGBA(CGImageRef image, uint8_t **outData, int *outW, int *outH) {
    size_t width = CGImageGetWidth(image);
    size_t height = CGImageGetHeight(image);
    if (width == 0 || height == 0 || width > 10000 || height > 10000) return 0;

    size_t bytesPerRow = width * 4;
    size_t total = bytesPerRow * height;
    uint8_t *data = (uint8_t *)calloc(total, 1);
    if (!data) return 0;

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
        return 0;
    }
    CGContextDrawImage(ctx, CGRectMake(0, 0, width, height), image);
    CGContextRelease(ctx);

    *outData = data;
    *outW = (int)width;
    *outH = (int)height;
    return 1;
}

// 截取單一視窗。macOS 14+ 走 ScreenCaptureKit，否則 fallback 到
// CGWindowListCreateImage（dlsym：新 SDK 已標棄用、未來可能移除 symbol）。
static int nativeCaptureWindowRGBA(long windowNumber, uint8_t **outData, int *outW, int *outH, WinInfo *outInfo) {
    *outData = NULL;
    *outW = 0;
    *outH = 0;
    if (outInfo) nativeWindowInfoByNumber(windowNumber, outInfo);

    CGImageRef image = ScWindowCaptureCGImage((uint32_t)windowNumber);
    if (!image) {
        typedef CGImageRef (*CreateWindowImageFn)(CGRect, CGWindowListOption, CGWindowID, CGWindowImageOption);
        CreateWindowImageFn createImage = (CreateWindowImageFn)dlsym(RTLD_DEFAULT, "CGWindowListCreateImage");
        if (!createImage) return 0;
        image = createImage(
            CGRectNull,
            kCGWindowListOptionIncludingWindow,
            (CGWindowID)windowNumber,
            kCGWindowImageBoundsIgnoreFraming);
    }
    if (!image) return 0;

    int ok = nativeImageToRGBA(image, outData, outW, outH);
    CGImageRelease(image);
    return ok;
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

var (
	nativeAccessibilityPromptShown atomic.Bool
	nativeListenPromptShown        atomic.Bool
	nativeScreenCapturePromptShown atomic.Bool
)

// NativeInput records OS-level click input on macOS via a Quartz CGEventTap.
// Replay activates the target window (AX raise + frontmost) before posting
// clicks, and supports visual relocation when window capture is available.
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
// 若缺少授權（CGEventTapCreate 回 NULL），回明確錯誤。
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
	//
	// 注意：listen-only event tap 需要的是「輸入監控」，跟 Accessibility 是
	// 兩個獨立授權，兩個都要請求。
	if C.nativeCheckAccessibility() == 0 && nativeAccessibilityPromptShown.CompareAndSwap(false, true) {
		C.nativeRequestAccessibility()
	}
	if C.nativePreflightListenAccess() == 0 && nativeListenPromptShown.CompareAndSwap(false, true) {
		C.nativeRequestListenAccess()
	}

	tap := C.nativeTapSetup(C.uintptr_t(n.handle))
	if tap == nil {
		missing := make([]string, 0, 2)
		if C.nativeCheckAccessibility() == 0 {
			missing = append(missing, "輔助使用 Accessibility")
		}
		if C.nativePreflightListenAccess() == 0 {
			missing = append(missing, "輸入監控 Input Monitoring")
		}
		detail := "Accessibility 與 Input Monitoring"
		if len(missing) > 0 {
			detail = strings.Join(missing, "、")
		}
		ready <- fmt.Errorf("native input: CGEventTapCreate failed — 請到 系統設定 → 隱私權與安全性 開啟「%s」後重啟 app（缺輸入監控時錄製收不到任何外部點擊）", detail)
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
func (n *NativeInput) emitClick(x, y, button, clickCount int) {
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
		ClickCount:    clickCount,
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

// ResolveWindow 比對錄到的 window handle 與目前桌面：
//  1. windowNumber 還活著且 owner 沒變 → 直接用；
//  2. 否則用 process(owner) + title 重新找（目標 app 重開後 windowNumber 必失效）。
func (n *NativeInput) ResolveWindow(handle uintptr, process, title string) (ResolvedWindow, bool) {
	process = strings.TrimSpace(process)
	title = strings.TrimSpace(title)

	if handle != 0 {
		var info C.WinInfo
		if C.nativeWindowInfoByNumber(C.long(handle), &info) != 0 && info.found != 0 && info.w > 1 && info.h > 1 {
			owner := C.GoString(&info.owner[0])
			// windowNumber 會被系統回收重用；owner 變了就視為失效。
			if process == "" || strings.EqualFold(owner, process) {
				return resolvedFromWinInfo(&info, false), true
			}
		}
	}

	if process == "" {
		return ResolvedWindow{}, false
	}
	cProcess := C.CString(process)
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cProcess))
	defer C.free(unsafe.Pointer(cTitle))
	var info C.WinInfo
	C.nativeFindWindowByOwnerTitle(cProcess, cTitle, &info)
	if info.found == 0 {
		return ResolvedWindow{}, false
	}
	return resolvedFromWinInfo(&info, true), true
}

func resolvedFromWinInfo(info *C.WinInfo, refound bool) ResolvedWindow {
	owner := C.GoString(&info.owner[0])
	title := C.GoString(&info.title[0])
	if title == "" {
		title = owner
	}
	return ResolvedWindow{
		Handle:  uintptr(info.windowNumber),
		PID:     int(info.pid),
		Title:   title,
		Process: owner,
		Rect: PixelBBox{
			X: int(info.x),
			Y: int(info.y),
			W: int(info.w),
			H: int(info.h),
		},
		Refound: refound,
	}
}

// activateStepWindow 回放點擊前把目標視窗帶到前景（對齊 Windows 版
// SetForegroundWindow + waitForForegroundWindow 行為）。
// 回傳 (foregroundOK, frontTitle, frontProcess)。
func (n *NativeInput) activateStepWindow(step LearningReplayStep) (bool, string, string) {
	resolved, ok := n.ResolveWindow(step.WindowHandle, step.WindowProcess, step.WindowTitle)
	if !ok {
		front := frontmostWindowInfo()
		return false, front.Title, front.Process
	}

	// 已在前景就不用動。
	if front := frontmostWindowInfo(); front.PID != 0 && front.PID == resolved.PID {
		return true, front.Title, front.Process
	}

	cTitle := C.CString(resolved.Title)
	C.nativeActivateWindow(
		C.int(resolved.PID),
		C.double(resolved.Rect.X), C.double(resolved.Rect.Y),
		C.double(resolved.Rect.W), C.double(resolved.Rect.H),
		cTitle,
	)
	C.free(unsafe.Pointer(cTitle))

	deadline := time.Now().Add(900 * time.Millisecond)
	for time.Now().Before(deadline) {
		front := frontmostWindowInfo()
		if front.PID != 0 && front.PID == resolved.PID {
			return true, front.Title, front.Process
		}
		time.Sleep(50 * time.Millisecond)
	}
	front := frontmostWindowInfo()
	return false, front.Title, front.Process
}

func frontmostWindowInfo() ResolvedWindow {
	var info C.WinInfo
	C.nativeFrontmostWindow(&info)
	if info.found == 0 {
		return ResolvedWindow{}
	}
	return resolvedFromWinInfo(&info, false)
}

// Click / MoveCursorOnly：以 macOS 全域螢幕座標（point）重放。
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
			Error:         "native input: 回放需要「輔助使用 Accessibility」權限（系統設定 → 隱私權與安全性 → 輔助使用），開啟後請重啟 app",
			WindowTitle:   step.WindowTitle,
			WindowProcess: step.WindowProcess,
		}
	}

	// 關鍵：先把目標視窗帶到前景。沒有這步，使用者錄完示範切回本 app 後，
	// 目標視窗被蓋住，所有點擊都會打在本 app 或別的視窗上。
	foregroundOK, frontTitle, frontProcess := n.activateStepWindow(step)

	n.moveCursorSmooth(step.X, step.Y)
	time.Sleep(450 * time.Millisecond)
	rightButton := 0
	if strings.EqualFold(step.Button, "right") || strings.EqualFold(step.Action, "right_click") {
		rightButton = 1
	}
	clicks := 1
	if strings.EqualFold(step.Action, string(MouseEventDoubleClick)) {
		clicks = 2
	}
	if C.nativePostMouseClick(C.double(step.X), C.double(step.Y), C.int(rightButton), C.int(clicks)) == 0 {
		return NativeReplayResult{
			OK:                false,
			Method:            "native",
			Index:             step.Index,
			Label:             step.Label,
			X:                 step.X,
			Y:                 step.Y,
			Error:             "native input: failed to create macOS mouse event",
			WindowTitle:       step.WindowTitle,
			WindowProcess:     step.WindowProcess,
			ForegroundOK:      foregroundOK,
			ForegroundTitle:   frontTitle,
			ForegroundProcess: frontProcess,
		}
	}
	time.Sleep(220 * time.Millisecond)
	warning := ""
	if !foregroundOK {
		warning = "target window could not be confirmed as frontmost before the click; the click may have hit another window"
	}
	return NativeReplayResult{
		OK:                true,
		Method:            "native",
		Index:             step.Index,
		Label:             step.Label,
		X:                 step.X,
		Y:                 step.Y,
		WindowTitle:       step.WindowTitle,
		WindowProcess:     step.WindowProcess,
		ForegroundOK:      foregroundOK,
		ForegroundTitle:   frontTitle,
		ForegroundProcess: frontProcess,
		Warning:           warning,
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
			Error:   "native input: 回放預覽需要「輔助使用 Accessibility」權限，開啟後請重啟 app",
		}
	}
	// 預覽也先把目標帶到前景，使用者才能看到游標停在正確元件上。
	n.activateStepWindow(step)
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

// CaptureWindow 截取單一視窗（macOS 14+ 用 ScreenCaptureKit，否則 legacy）。
// 回傳的 WindowCapture.Scale = 截圖 pixel 寬 / 視窗 point 寬（Retina ≈ 2）。
func (n *NativeInput) CaptureWindow(hwnd uintptr) (WindowCapture, error) {
	if hwnd == 0 {
		return WindowCapture{}, fmt.Errorf("native capture: window handle is required")
	}
	if C.nativePreflightScreenCapture() == 0 {
		if nativeScreenCapturePromptShown.CompareAndSwap(false, true) {
			C.nativeRequestScreenCapture()
		}
		return WindowCapture{}, fmt.Errorf("native capture: 缺少「螢幕錄製 Screen Recording」權限（系統設定 → 隱私權與安全性 → 螢幕錄製），視覺重定位停用，回放將退回原始座標")
	}
	var data *C.uint8_t
	var width, height C.int
	var info C.WinInfo
	ok := C.nativeCaptureWindowRGBA(C.long(hwnd), &data, &width, &height, &info)
	if ok == 0 || data == nil || width <= 0 || height <= 0 {
		return WindowCapture{}, fmt.Errorf("native capture: window capture failed (window %d may be closed)", hwnd)
	}
	defer C.free(unsafe.Pointer(data))
	size := int(width) * int(height) * 4
	imageData := C.GoBytes(unsafe.Pointer(data), C.int(size))
	owner := C.GoString(&info.owner[0])
	title := C.GoString(&info.title[0])
	if title == "" {
		title = owner
	}
	scale := 1.0
	if info.found != 0 && info.w > 0 {
		scale = float64(width) / float64(info.w)
		// 截圖可能比視窗 bounds 多/少 1-2px（陰影裁切），吸附到常見倍率。
		for _, snap := range []float64{1, 2, 3} {
			if scale > snap*0.92 && scale < snap*1.08 {
				scale = snap
				break
			}
		}
		if scale <= 0 {
			scale = 1
		}
	}
	return WindowCapture{
		ImageData: imageData,
		Width:     int(width),
		Height:    int(height),
		Scale:     scale,
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
