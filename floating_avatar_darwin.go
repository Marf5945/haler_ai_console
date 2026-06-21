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
