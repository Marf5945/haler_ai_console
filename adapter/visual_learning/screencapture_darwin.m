// screencapture_darwin.m — ScreenCaptureKit 單視窗截圖實作（macOS 14+）。
//
// 流程：SCShareableContent（async）→ 找到 CGWindowID 對應的 SCWindow
//   → SCContentFilter(desktopIndependentWindow) → SCScreenshotManager
//   → CGImageRef（pixel 尺寸 = window point 尺寸 × pointPixelScale）。
//
// 連結策略：所有 SCK class 都用 NSClassFromString 在執行期解析，
//   「不」在連結期綁定 ScreenCaptureKit symbol（cgo 的 LDFLAGS 安全白名單
//   不接受 -weak_framework，強連結又會讓 app 在舊 macOS 上無法啟動）。
//   framework 本體由 scLoadFramework() 在第一次截圖時 dlopen 載入，
//   因此 LDFLAGS 完全不需要 ScreenCaptureKit，任何 Go 版本都不需 allowlist。
//   標頭僅用於型別/selector 宣告，不會產生 link 依賴（cgo 未開 -fmodules，
//   不會 autolink）。
//
// 同步化：SCK 的 API 都是 async callback，這裡用 dispatch_semaphore 等待，
//   並透過 heap 上的 atomic refcount holder 避免 timeout 後 callback 遲到
//   寫入已釋放記憶體（timeout 路徑只會小幅延後釋放，不會 crash）。
//
// 本套件以 -fobjc-arc 編譯（見 coreml_bridge_darwin.go 的 #cgo CFLAGS）。

#import <Foundation/Foundation.h>
#import <CoreGraphics/CoreGraphics.h>
#include <stdatomic.h>
#include <stdlib.h>
#include <dlfcn.h>
#include "screencapture_darwin.h"

#if __has_include(<ScreenCaptureKit/ScreenCaptureKit.h>)
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#define UICONSOLE_HAS_SCK 1
#endif

#ifdef UICONSOLE_HAS_SCK

// scHolder 讓「等待端」與「callback 端」各持一個引用；最後釋放者負責清理。
typedef struct {
    void *_Atomic object;   // retained CFTypeRef，由 callback 寫入
    _Atomic int refs;
} scHolder;

static scHolder *scHolderCreate(void) {
    scHolder *h = calloc(1, sizeof(scHolder));
    if (h) atomic_store(&h->refs, 2);
    return h;
}

// scHolderTake 取走 object 所有權（之後 holder 不再負責釋放它）。
static void *scHolderTake(scHolder *h) {
    return atomic_exchange(&h->object, NULL);
}

static void scHolderRelease(scHolder *h) {
    if (!h) return;
    if (atomic_fetch_sub(&h->refs, 1) == 1) {
        void *obj = atomic_exchange(&h->object, NULL);
        if (obj) CFRelease((CFTypeRef)obj);
        free(h);
    }
}

API_AVAILABLE(macos(14.0))
static SCShareableContent *scFetchShareableContent(Class contentCls) {
    scHolder *h = scHolderCreate();
    if (!h) return nil;
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    [contentCls getShareableContentExcludingDesktopWindows:NO
                                       onScreenWindowsOnly:YES
                                         completionHandler:^(SCShareableContent *content, NSError *error) {
        if (content && !error) {
            atomic_store(&h->object, (void *)CFBridgingRetain(content));
        }
        dispatch_semaphore_signal(sem);
        scHolderRelease(h);
    }];
    SCShareableContent *result = nil;
    if (dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, (int64_t)(4 * NSEC_PER_SEC))) == 0) {
        void *obj = scHolderTake(h);
        if (obj) result = CFBridgingRelease(obj); // 轉回 ARC 管理
    }
    scHolderRelease(h);
    return result;
}

API_AVAILABLE(macos(14.0))
static CGImageRef scCaptureWithFilter(Class screenshotCls, SCContentFilter *filter, SCStreamConfiguration *config) {
    scHolder *h = scHolderCreate();
    if (!h) return NULL;
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    [screenshotCls captureImageWithFilter:filter
                            configuration:config
                        completionHandler:^(CGImageRef image, NSError *error) {
        if (image && !error) {
            atomic_store(&h->object, (void *)CGImageRetain(image));
        }
        dispatch_semaphore_signal(sem);
        scHolderRelease(h);
    }];
    CGImageRef result = NULL;
    if (dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, (int64_t)(4 * NSEC_PER_SEC))) == 0) {
        result = (CGImageRef)scHolderTake(h);
    }
    scHolderRelease(h);
    return result;
}

#endif // UICONSOLE_HAS_SCK

// scLoadFramework：執行期 dlopen ScreenCaptureKit（取代 -weak_framework；
// Go 1.26 cgo 白名單連 -Wl, 形式都拒收）。載入一次後常駐；
// 失敗（舊 macOS 無此 framework）回 NULL，呼叫端照走 legacy fallback。
static void *scLoadFramework(void) {
    static void *handle = NULL;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        handle = dlopen("/System/Library/Frameworks/ScreenCaptureKit.framework/Versions/A/ScreenCaptureKit", RTLD_LAZY);
        if (!handle) {
            handle = dlopen("/System/Library/Frameworks/ScreenCaptureKit.framework/ScreenCaptureKit", RTLD_LAZY);
        }
    });
    return handle;
}

CGImageRef ScWindowCaptureCGImage(uint32_t windowID) {
#ifdef UICONSOLE_HAS_SCK
    if (@available(macOS 14.0, *)) {
        @autoreleasepool {
            if (!scLoadFramework()) return NULL;
            Class contentCls = NSClassFromString(@"SCShareableContent");
            Class filterCls = NSClassFromString(@"SCContentFilter");
            Class configCls = NSClassFromString(@"SCStreamConfiguration");
            Class screenshotCls = NSClassFromString(@"SCScreenshotManager");
            if (!contentCls || !filterCls || !configCls || !screenshotCls) return NULL;

            SCShareableContent *content = scFetchShareableContent(contentCls);
            if (!content) return NULL;
            SCWindow *target = nil;
            for (SCWindow *w in content.windows) {
                if (w.windowID == windowID) { target = w; break; }
            }
            if (!target) return NULL;

            SCContentFilter *filter = [[filterCls alloc] initWithDesktopIndependentWindow:target];
            if (!filter) return NULL;
            SCStreamConfiguration *config = [[configCls alloc] init];
            if (!config) return NULL;
            float scale = filter.pointPixelScale;
            if (scale <= 0) scale = 1;
            size_t w = (size_t)(target.frame.size.width * scale);
            size_t hpx = (size_t)(target.frame.size.height * scale);
            if (w == 0 || hpx == 0 || w > 10000 || hpx > 10000) return NULL;
            config.width = w;
            config.height = hpx;
            config.showsCursor = NO;
            return scCaptureWithFilter(screenshotCls, filter, config);
        }
    }
#endif
    (void)windowID;
    return NULL;
}
