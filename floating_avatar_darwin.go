//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <dispatch/dispatch.h>
#import <AppKit/AppKit.h>

// 取得主視窗：優先 mainWindow / keyWindow，再退回第一個可見視窗。
static NSWindow *AIConsoleResolveWindow(void) {
    NSWindow *w = [NSApp mainWindow];
    if (w != nil) return w;
    w = [NSApp keyWindow];
    if (w != nil) return w;
    for (NSWindow *cand in [NSApp windows]) {
        if ([cand isVisible]) return cand;
    }
    if ([[NSApp windows] count] > 0) {
        return [[NSApp windows] objectAtIndex:0];
    }
    return nil;
}

// 遞迴找出 WKWebView（Wails 把 webview 掛在 contentView 底下）。
static NSView *AIConsoleFindWebView(NSView *root) {
    if (root == nil) return nil;
    if ([NSStringFromClass([root class]) containsString:@"WKWebView"]) return root;
    for (NSView *sub in [root subviews]) {
        NSView *hit = AIConsoleFindWebView(sub);
        if (hit != nil) return hit;
    }
    return nil;
}

// 後台浮窗透明加固：WKWebView 一旦出現 WebGL 等加速內容（全身像＋動態圖像），
// 若 drawsBackground / underPageBackgroundColor / layer 底色沒清乾淨，
// 合成器會把整片 webview 畫成不透明黑塊。這裡一次清到底；還原時恢復系統底色。
static void AIConsoleHardenWebViewTransparency(NSWindow *w, int transparent) {
    NSView *content = [w contentView];
    NSView *web = AIConsoleFindWebView(content);
    if (web != nil) {
        if (transparent) {
            @try {
                [web setValue:[NSNumber numberWithBool:NO] forKey:@"drawsBackground"];
            } @catch (NSException *e) {}  // WebKit 版本差異，忽略
        }
        @try {
            if ([web respondsToSelector:@selector(setUnderPageBackgroundColor:)]) {
                [web setValue:(transparent ? [NSColor clearColor] : [NSColor windowBackgroundColor])
                       forKey:@"underPageBackgroundColor"];
            }
        } @catch (NSException *e) {}  // 忽略
        if ([web layer] != nil) {
            [[web layer] setOpaque:NO];
            [[web layer] setBackgroundColor:(transparent ? [[NSColor clearColor] CGColor] : NULL)];
        }
    }
    if (content != nil && [content layer] != nil) {
        [[content layer] setBackgroundColor:(transparent ? [[NSColor clearColor] CGColor] : NULL)];
    }
}

// 切換浮動頭像視窗型態。所有 AppKit 操作都丟回 main thread，
// 因為這支函式由 Go bound method（非主執行緒）觸發。
static void AIConsoleSetFloatingAvatar(int on) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *w = AIConsoleResolveWindow();
        if (w == nil) return;

        if (on) {
            // 保留 titled（讓視窗仍可成為 key，迷你框才能輸入文字），
            // 但隱藏標題列與紅綠燈，內容延伸到整片視窗。
            w.titlebarAppearsTransparent = YES;
            w.titleVisibility = NSWindowTitleHidden;
            w.movableByWindowBackground = YES;
            w.styleMask |= NSWindowStyleMaskFullSizeContentView;
            [[w standardWindowButton:NSWindowCloseButton] setHidden:YES];
            [[w standardWindowButton:NSWindowMiniaturizeButton] setHidden:YES];
            [[w standardWindowButton:NSWindowZoomButton] setHidden:YES];

            // 真透明：視窗非不透明 + 清空背景 + 去陰影。
            [w setOpaque:NO];
            [w setHasShadow:NO];
            [w setBackgroundColor:[NSColor clearColor]];

            // 永遠置頂 + 跨所有 Space + 蓋在全螢幕 app 上。
            [w setLevel:NSStatusWindowLevel];
            [w setCollectionBehavior:
                NSWindowCollectionBehaviorCanJoinAllSpaces |
                NSWindowCollectionBehaviorStationary |
                NSWindowCollectionBehaviorFullScreenAuxiliary |
                NSWindowCollectionBehaviorIgnoresCycle];

            // WebGL（動態全身像）掛載後仍要維持真透明。
            AIConsoleHardenWebViewTransparency(w, 1);
        } else {
            // 還原一般主控台視窗。
            w.titlebarAppearsTransparent = NO;
            w.titleVisibility = NSWindowTitleVisible;
            w.movableByWindowBackground = NO;
            w.styleMask &= ~NSWindowStyleMaskFullSizeContentView;
            [[w standardWindowButton:NSWindowCloseButton] setHidden:NO];
            [[w standardWindowButton:NSWindowMiniaturizeButton] setHidden:NO];
            [[w standardWindowButton:NSWindowZoomButton] setHidden:NO];

            [w setOpaque:YES];
            [w setHasShadow:YES];
            [w setBackgroundColor:[NSColor windowBackgroundColor]];

            [w setLevel:NSNormalWindowLevel];
            [w setCollectionBehavior:NSWindowCollectionBehaviorDefault];

            AIConsoleHardenWebViewTransparency(w, 0);
        }
    });
}
*/
import "C"

// applyFloatingAvatarWindow 在 macOS 上切換主視窗的浮動頭像型態。
func applyFloatingAvatarWindow(on bool) {
	flag := C.int(0)
	if on {
		flag = C.int(1)
	}
	C.AIConsoleSetFloatingAvatar(flag)
}
