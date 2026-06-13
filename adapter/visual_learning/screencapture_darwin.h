// screencapture_darwin.h — ScreenCaptureKit 單視窗截圖 C API（macOS 14+）。
//
// CGWindowListCreateImage 已於 macOS 14 棄用，且在缺「螢幕錄製」權限或新版
// macOS 上可能直接回 NULL。此 bridge 在 macOS 14+ 改走 SCScreenshotManager，
// 舊系統由呼叫端 fallback 回 legacy 路徑。
//
// 依專案規範（§14.6.1 同 CoreML bridge）：Objective-C 實作放獨立 .m 檔。
#ifndef SCREENCAPTURE_DARWIN_H
#define SCREENCAPTURE_DARWIN_H

#include <stdint.h>
#include <CoreGraphics/CoreGraphics.h>

#ifdef __cplusplus
extern "C" {
#endif

// ScWindowCaptureCGImage captures one window by CGWindowID via
// ScreenCaptureKit. Returns a +1 retained CGImageRef, or NULL when:
//   - macOS < 14.0 (caller should fall back to CGWindowListCreateImage)
//   - Screen Recording permission is missing
//   - the window is gone / capture timed out
// Caller owns the returned image (CGImageRelease).
CGImageRef ScWindowCaptureCGImage(uint32_t windowID);

#ifdef __cplusplus
}
#endif

#endif // SCREENCAPTURE_DARWIN_H
