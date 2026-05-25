//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework UniformTypeIdentifiers
#include <stdlib.h>
#include <stdio.h>
#include <dispatch/dispatch.h>
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>

@interface AIConsoleDragSource : NSObject <NSDraggingSource, NSFilePromiseProviderDelegate>
@property dispatch_semaphore_t done;
@property NSDragOperation operation;
@property BOOL promiseFinished;
@property BOOL promiseFailed;
@property BOOL sourceIsDirectory;
@property (nonatomic, strong) NSString *sourcePath;
@property (nonatomic, strong) NSString *fileName;
@property (nonatomic, strong) NSString *landedPath;
@end

@implementation AIConsoleDragSource
- (NSDragOperation)draggingSession:(NSDraggingSession *)session sourceOperationMaskForDraggingContext:(NSDraggingContext)context {
    return NSDragOperationCopy;
}
- (BOOL)ignoreModifierKeysForDraggingSession:(NSDraggingSession *)session {
    return YES;
}
- (void)draggingSession:(NSDraggingSession *)session endedAtPoint:(NSPoint)screenPoint operation:(NSDragOperation)operation {
    self.operation = operation;
    if (operation == NSDragOperationNone && self.done != nil) {
        dispatch_semaphore_signal(self.done);
    }
}
- (NSString *)filePromiseProvider:(NSFilePromiseProvider *)filePromiseProvider fileNameForType:(NSString *)fileType {
    return self.fileName;
}
- (void)filePromiseProvider:(NSFilePromiseProvider *)filePromiseProvider writePromiseToURL:(NSURL *)url completionHandler:(void (^)(NSError * _Nullable error))completionHandler {
    NSError *error = nil;
    NSURL *sourceURL = [NSURL fileURLWithPath:self.sourcePath isDirectory:self.sourceIsDirectory];
    [[NSFileManager defaultManager] copyItemAtURL:sourceURL toURL:url error:&error];
    self.landedPath = url.path;
    self.promiseFinished = YES;
    self.promiseFailed = error != nil;
    completionHandler(error);
    if (self.done != nil) {
        dispatch_semaphore_signal(self.done);
    }
}
@end

static int AIConsoleStartFilePromiseDrag(const char *cpath, char *message, int messageLen, char *landedPath, int landedPathLen) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:cpath];
        if (path == nil || path.length == 0) {
            snprintf(message, messageLen, "empty export path");
            return 0;
        }
        BOOL isDirectory = NO;
        if (![[NSFileManager defaultManager] fileExistsAtPath:path isDirectory:&isDirectory]) {
            snprintf(message, messageLen, "export folder does not exist");
            return 0;
        }

        if ([NSThread isMainThread]) {
            snprintf(message, messageLen, "native drag cannot start from main thread");
            return 0;
        }

        __block int ok = 0;
        __block NSString *detail = nil;
        AIConsoleDragSource *source = [AIConsoleDragSource new];
        source.done = dispatch_semaphore_create(0);
        source.operation = NSDragOperationNone;
        source.promiseFinished = NO;
        source.promiseFailed = NO;
        source.sourceIsDirectory = isDirectory;
        source.sourcePath = path;
        source.fileName = [path lastPathComponent];
        source.landedPath = nil;

        dispatch_block_t startDrag = ^{
            NSWindow *window = [NSApp keyWindow] ?: [NSApp mainWindow];
            NSView *view = window.contentView;
            if (view == nil) {
                detail = @"no active NSView";
                dispatch_semaphore_signal(source.done);
                return;
            }

            // Finder asks the provider to create this item at the drop target.
            NSString *fileType = isDirectory ? @"public.folder" : @"public.data";
            NSFilePromiseProvider *provider = [[NSFilePromiseProvider alloc] initWithFileType:fileType delegate:source];
            NSDraggingItem *dragItem = [[NSDraggingItem alloc] initWithPasteboardWriter:provider];
            NSImage *icon = [[NSWorkspace sharedWorkspace] iconForFile:path];
            [icon setSize:NSMakeSize(64, 64)];

            NSPoint mouse = [view convertPoint:window.mouseLocationOutsideOfEventStream fromView:nil];
            NSRect frame = NSMakeRect(mouse.x - 24, mouse.y - 24, 64, 64);
            [dragItem setDraggingFrame:frame contents:icon];

            NSEvent *event = [NSEvent mouseEventWithType:NSEventTypeLeftMouseDragged
                                                location:[window mouseLocationOutsideOfEventStream]
                                           modifierFlags:0
                                               timestamp:[[NSProcessInfo processInfo] systemUptime]
                                            windowNumber:window.windowNumber
                                                 context:nil
                                             eventNumber:0
                                              clickCount:1
                                                pressure:1.0];
            if (event == nil) {
                detail = @"unable to create drag event";
                dispatch_semaphore_signal(source.done);
                return;
            }

            NSDraggingSession *session = [view beginDraggingSessionWithItems:@[dragItem] event:event source:source];
            session.animatesToStartingPositionsOnCancelOrFail = YES;
            ok = session != nil ? 1 : 0;
            if (!ok) {
                detail = @"beginDraggingSession returned nil";
                dispatch_semaphore_signal(source.done);
            }
        };

        dispatch_async(dispatch_get_main_queue(), startDrag);

        dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, 120 * NSEC_PER_SEC);
        long waitResult = dispatch_semaphore_wait(source.done, timeout);

        if (!ok) {
            const char *err = detail ? [detail UTF8String] : "native drag failed";
            snprintf(message, messageLen, "%s", err);
            return 0;
        }
        if (waitResult != 0) {
            snprintf(message, messageLen, "native drag timed out");
            return 0;
        }
        if (source.promiseFinished && !source.promiseFailed) {
            const char *landed = source.landedPath ? [source.landedPath UTF8String] : "";
            snprintf(landedPath, landedPathLen, "%s", landed);
            snprintf(message, messageLen, "macOS native file promise completed");
            return 1;
        }
        if (source.operation == NSDragOperationNone && !source.promiseFinished) {
            snprintf(message, messageLen, "native drag cancelled");
            return -1;
        }
        if (source.promiseFailed || !source.promiseFinished) {
            snprintf(message, messageLen, "native file promise failed");
            return 0;
        }
        snprintf(message, messageLen, "native drag finished without a file promise");
        return 0;
    }
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

func startNativeFileDrag(path string) nativeDragResult {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	buf := make([]C.char, 512)
	landed := make([]C.char, 2048)
	ok := C.AIConsoleStartFilePromiseDrag(cPath, &buf[0], C.int(len(buf)), &landed[0], C.int(len(landed)))
	message := C.GoString(&buf[0])
	if ok == 1 {
		return nativeDragResult{
			Status:           nativeDragStatusSuccess,
			FallbackRequired: false,
			Message:          message,
			LandedPath:       C.GoString(&landed[0]),
		}
	}
	if ok == -1 {
		return nativeDragResult{
			Status:           nativeDragStatusCancelled,
			FallbackRequired: false,
			Message:          message,
		}
	}

	if strings.TrimSpace(message) == "" {
		message = "macOS 原生拖曳啟動失敗，已改用資料夾選擇器"
	}
	return nativeDragResult{
		Status:           nativeDragStatusFailed,
		FallbackRequired: true,
		Message:          message,
	}
}
