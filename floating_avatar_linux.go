//go:build linux

package main

// floating_avatar_linux.go — 後台浮動頭像的 Linux（GTK3 / WebKitGTK）原生實作。
//
// 這是 floating_avatar_darwin.go 的 Linux 對應版本。macOS 上黑塊的病因是
// WKWebView 掛上 WebGL（動態全身像）後，合成器把沒清乾淨背景的透明視窗畫成
// 不透明黑；Linux 上 Wails 用 GTK3 + WebKitGTK，症狀與解法一模一樣：
//
//   - 視窗要能真透明，GtkWindow 必須改用 RGBA visual 並 app-paintable；
//   - WebKitWebView 一旦出現 WebGL 內容，若不把 web view 背景設成完全透明，
//     WebKitGTK 的合成器同樣會塗成不透明黑（webkit_web_view_set_background_color
//     設 alpha=0 是 WebKitGTK 端已知解法，對應 macOS 的 drawsBackground=NO）。
//
// 為避免對 webkit2gtk 的版本（4.0 / 4.1）產生硬連結耦合，這裡只用 pkg-config
// 連 gtk+-3.0，WebKitGTK 的函式在執行期以 dlsym 從已載入的行程解析
// （Wails 本身已把 webkit 連進來），因此不需要 webkit 標頭或連結旗標。

/*
#define _GNU_SOURCE
#cgo pkg-config: gtk+-3.0
#cgo LDFLAGS: -ldl

#include <gtk/gtk.h>
#include <dlfcn.h>
#include <string.h>

// 取得主視窗：優先第一個可見的 GtkWindow toplevel，再退回第一個 toplevel。
static GtkWidget *AIConsoleResolveWindow(void) {
    GList *tops = gtk_window_list_toplevels();
    GtkWidget *found = NULL;
    for (GList *l = tops; l != NULL; l = l->next) {
        GtkWidget *w = GTK_WIDGET(l->data);
        if (GTK_IS_WINDOW(w) && gtk_widget_get_visible(w)) {
            found = w;
            break;
        }
    }
    if (found == NULL && tops != NULL) {
        found = GTK_WIDGET(tops->data);
    }
    g_list_free(tops);
    return found;
}

// 遞迴找出 WebKitWebView（Wails 把 web view 掛在 GtkBox/GtkOverlay 底下）。
static GtkWidget *AIConsoleFindWebView(GtkWidget *root) {
    if (root == NULL) return NULL;
    const char *tn = G_OBJECT_TYPE_NAME(root);
    if (tn != NULL && strstr(tn, "WebKitWebView") != NULL) {
        return root;
    }
    if (GTK_IS_CONTAINER(root)) {
        GList *kids = gtk_container_get_children(GTK_CONTAINER(root));
        GtkWidget *hit = NULL;
        for (GList *l = kids; l != NULL; l = l->next) {
            hit = AIConsoleFindWebView(GTK_WIDGET(l->data));
            if (hit != NULL) break;
        }
        g_list_free(kids);
        return hit;
    }
    return NULL;
}

// GdkRGBA 的記憶體佈局：四個 gdouble（red, green, blue, alpha）。
// 自行宣告避免 include gdk 版本差異。
typedef struct { double red, green, blue, alpha; } AIConsoleRGBA;
// 對應 void webkit_web_view_set_background_color(WebKitWebView*, const GdkRGBA*)
typedef void (*AIConsoleSetBgFn)(void *web_view, const AIConsoleRGBA *rgba);

// 後台浮窗透明加固：把 WebKitWebView 背景設成完全透明（root cause 黑塊根治）；
// 還原時設回不透明白底。函式指標以 dlsym 解析並快取。
static void AIConsoleHardenWebViewTransparency(GtkWidget *web, int transparent) {
    if (web == NULL) return;
    static AIConsoleSetBgFn set_bg = NULL;
    static int resolved = 0;
    if (!resolved) {
        set_bg = (AIConsoleSetBgFn) dlsym(RTLD_DEFAULT, "webkit_web_view_set_background_color");
        resolved = 1;
    }
    if (set_bg == NULL) return;
    AIConsoleRGBA c;
    if (transparent) {
        c.red = 0.0; c.green = 0.0; c.blue = 0.0; c.alpha = 0.0;
    } else {
        c.red = 1.0; c.green = 1.0; c.blue = 1.0; c.alpha = 1.0;
    }
    set_bg((void *)web, &c);
}

// 實際切換型態；必須在 GTK 主執行緒執行（由 g_idle_add 排程進來）。
static void AIConsoleApplyFloatingAvatar(int on) {
    GtkWidget *win = AIConsoleResolveWindow();
    if (win == NULL) return;

    if (on) {
        // 真透明：改用 RGBA visual + app-paintable（對應 macOS setOpaque:NO）。
        GdkScreen *screen = gtk_widget_get_screen(win);
        if (screen != NULL) {
            GdkVisual *rgba = gdk_screen_get_rgba_visual(screen);
            if (rgba != NULL) {
                gtk_widget_set_visual(win, rgba);
            }
        }
        gtk_widget_set_app_paintable(win, TRUE);

        // 無框 + 永遠置頂 + 跨所有工作區 + 不進工作列/分頁器。
        gtk_window_set_decorated(GTK_WINDOW(win), FALSE);
        gtk_window_set_keep_above(GTK_WINDOW(win), TRUE);
        gtk_window_stick(GTK_WINDOW(win));
        gtk_window_set_skip_taskbar_hint(GTK_WINDOW(win), TRUE);
        gtk_window_set_skip_pager_hint(GTK_WINDOW(win), TRUE);

        // WebGL（動態全身像）掛載後維持真透明。
        AIConsoleHardenWebViewTransparency(AIConsoleFindWebView(win), 1);
    } else {
        // 還原一般主控台視窗。
        gtk_window_set_decorated(GTK_WINDOW(win), TRUE);
        gtk_window_set_keep_above(GTK_WINDOW(win), FALSE);
        gtk_window_unstick(GTK_WINDOW(win));
        gtk_window_set_skip_taskbar_hint(GTK_WINDOW(win), FALSE);
        gtk_window_set_skip_pager_hint(GTK_WINDOW(win), FALSE);
        gtk_widget_set_app_paintable(win, FALSE);

        AIConsoleHardenWebViewTransparency(AIConsoleFindWebView(win), 0);
    }

    gtk_widget_queue_draw(win);
}

// g_idle_add 回呼：回 G_SOURCE_REMOVE 表示只跑一次。
static gboolean AIConsoleFloatingAvatarIdle(gpointer data) {
    AIConsoleApplyFloatingAvatar(GPOINTER_TO_INT(data));
    return G_SOURCE_REMOVE;
}

// 由 Go bound method（非 GTK 主執行緒）呼叫；把操作排回主迴圈執行緒。
static void AIConsoleSetFloatingAvatar(int on) {
    g_idle_add(AIConsoleFloatingAvatarIdle, GINT_TO_POINTER(on));
}
*/
import "C"

// applyFloatingAvatarWindow 在 Linux 上切換主視窗的浮動頭像型態。
func applyFloatingAvatarWindow(on bool) {
	flag := C.int(0)
	if on {
		flag = C.int(1)
	}
	C.AIConsoleSetFloatingAvatar(flag)
}
