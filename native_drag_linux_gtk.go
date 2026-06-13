//go:build linux && gtk

// native_drag_linux_gtk.go — Linux GTK 原生拖放來源「骨架」（需 `-tags gtk` 建置）。
//
// ⚠️ 未經 GUI 實測：本檔提供正確的 GTK3 drag-source API 序列與防回退結構，
// 但「取得真實 GtkWidget + 觸發 GdkEvent」與「drag-end 結果回報」兩處需在有
// pkg-config gtk+-3.0、GTK dev headers、Linux 桌面 session 的機器上由工程師接線並實測。
//
// 防回退保證：任何一步缺件或失敗，一律回落 linuxDesktopCopyFallback，
// 故 `-tags gtk` 建置永遠不會比預設 fallback 差。
//
// 與預設版互斥：native_drag_linux.go 帶 `linux && !gtk`，本檔帶 `linux && gtk`，
// 兩者不會同時編入，startNativeFileDrag 不會重複定義。
package main

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <stdlib.h>
#include <string.h>

// drag-data-get：拖放過程被要求供資料時，把來源檔轉成 file:// URI 放入 selection。
static void ai_console_drag_data_get(GtkWidget *widget, GdkDragContext *context,
                                     GtkSelectionData *data, guint info, guint time,
                                     gpointer user_data) {
    const char *path = (const char *)user_data;
    if (path == NULL) return;
    char *uri = g_filename_to_uri(path, NULL, NULL);
    if (uri != NULL) {
        char *uris[2] = { uri, NULL };
        gtk_selection_data_set_uris(data, uris);
        g_free(uri);
    }
}

// drag-end：清理 callback 持有的 path 副本。
// TODO(工程師)：在此（或 drag-failed）以 semaphore/channel 通知 Go 端實際 drop 結果與落點。
static void ai_console_drag_end(GtkWidget *widget, GdkDragContext *context, gpointer user_data) {
    if (user_data != NULL) free(user_data);
}

// ai_console_start_gtk_drag：啟動 GTK 拖放會話。
//   widget_ptr / event_ptr：真實 GtkWidget* 與觸發用 GdkEvent*（button-press 當下），
//     由 Wails linux 執行期提供（TODO：尚未接線，傳 0 時本函式回 0）。
//   回傳 1=拖放會話已啟動；0=無法啟動（缺件等）→ Go 端回落 fallback。
static int ai_console_start_gtk_drag(guintptr widget_ptr, guintptr event_ptr, const char *path) {
    if (widget_ptr == 0 || event_ptr == 0 || path == NULL) {
        return 0; // 尚未接到真實 widget/event → 交 Go 端回落，不勉強啟動
    }
    GtkWidget *widget = (GtkWidget *)widget_ptr;
    GdkEvent *event = (GdkEvent *)event_ptr;

    GtkTargetList *targets = gtk_target_list_new(NULL, 0);
    gtk_target_list_add_uri_targets(targets, 0);

    // path 需在 drag-data-get 前持續有效；複製一份交 callback，drag-end 釋放。
    char *path_copy = strdup(path);
    g_signal_connect(widget, "drag-data-get", G_CALLBACK(ai_console_drag_data_get), path_copy);
    g_signal_connect(widget, "drag-end", G_CALLBACK(ai_console_drag_end), path_copy);

    GdkDragContext *ctx = gtk_drag_begin_with_coordinates(
        widget, targets, GDK_ACTION_COPY, 1, event, -1, -1);
    gtk_target_list_unref(targets);
    return ctx != NULL ? 1 : 0;
}
*/
import "C"

import "unsafe"

// gtkDragHandles 回傳啟動拖放所需的真實 GtkWidget* 與 GdkEvent* 指標。
// TODO(工程師)：接 Wails linux runtime——在 webview button-press 當下取得承載
// webview 的 GtkWidget 與該事件的 GdkEvent 指標。接線前回 0,0，使下方安全回落。
func gtkDragHandles() (widget, event uintptr) {
	return 0, 0
}

func startNativeFileDrag(path string) nativeDragResult {
	widget, event := gtkDragHandles()
	if widget == 0 || event == 0 {
		// 尚未接線：絕不比 fallback 差。
		return linuxDesktopCopyFallback(path, "GTK 原生拖放尚未接線（缺 widget/event）")
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	started := C.ai_console_start_gtk_drag(C.guintptr(widget), C.guintptr(event), cPath)
	if started != 1 {
		return linuxDesktopCopyFallback(path, "GTK 拖放啟動失敗")
	}

	// TODO(工程師)：等待 drag-end / drag-failed 訊號回報實際 drop 結果與落點，
	// 期間以 semaphore/channel 同步（參照 darwin 的 dispatch_semaphore 模式）。
	// 在完整結果回報接線前，仍回落確保使用者一定拿得到檔案（防回退）。
	return linuxDesktopCopyFallback(path, "GTK 拖放已啟動，落點回報尚未接線")
}
