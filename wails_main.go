package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// cspMiddleware 注入 Content-Security-Policy header。
// SEC-12: 防止 XSS / 注入攻擊。
func cspMiddleware(next http.Handler) http.Handler {
	// style-src 'unsafe-inline': Wails runtime 及部分 UI 框架需要 inline style。
	// img-src data:/blob: : 前端可能使用 data URI 或 blob 圖片。
	const csp = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"connect-src 'self'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self' data:; " +
		"object-src 'none'; " +
		"base-uri 'self'"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
}

func main() {
	// 釘選對話框（popout）子行程模式：只開小視窗殼，不跑主控台/排程/sidecar。
	// 注意：必須在 NewApp / SingleInstanceLock 之前分流，否則會被單一實例鎖擋下。
	if popArgs := parsePopoutArgs(os.Args); popArgs != nil {
		popoutAssets, subErr := fs.Sub(assets, "frontend/dist")
		if subErr != nil {
			println("Error:", subErr.Error())
			return
		}
		if runErr := runPopoutWindow(popoutAssets, popArgs); runErr != nil {
			println("Error:", runErr.Error())
		}
		return
	}

	// Create an instance of the app structure
	app := NewApp()
	// Phase G：LaunchAgent 以 --scheduled-wake 喚醒 → 隱藏視窗最小背景模式啟動。
	scheduledWake := hasScheduledWakeFlag(os.Args)
	frontendAssets, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		println("Error:", err.Error())
		return
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "HaLer AI Console",
		Width:     1536,
		Height:    860,
		MinWidth:  1180,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets:     frontendAssets,
			Middleware: cspMiddleware, // SEC-12: CSP header 注入
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		BackgroundColour: &options.RGBA{R: 5, G: 5, B: 5, A: 255},
		// 後台浮動頭像需要真透明：WebView 設為透明，避免 WKWebView 自身白底
		// 蓋住前端的 transparent CSS。視窗外框/紅綠燈改由執行期 cgo 切換
		// （floating_avatar_darwin.go），一般主控台維持正常有框視窗。
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
		},
		// v2.7：Windows 照 mac 接法。WebView2 底色設為可透明（wails 會把所有
		// WindowSetBackgroundColour 的 alpha 強制為 0，實際外觀由前端 CSS 決定，
		// 主控台 CSS 是不透明 #050505 所以平時看不出差別）。
		// WindowIsTranslucent 讓建窗時帶 WS_EX_NOREDIRECTIONBITMAP（此旗標只能在
		// 建窗當下給，執行期補不了），配 BackdropType None = 乾淨真透明、不帶
		// Mica/壓克力模糊。無框＋置頂由 floating_avatar_windows.go 於執行期切換。
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.None,
		},
		// Phase G：排程喚醒時不跳視窗（隱藏啟動）。
		StartHidden: scheduledWake,
		// 單一實例鎖：避免每次喚醒堆積行程／重複執行；第二實例會被導回第一實例。
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "com.aiconsole.singleinstance",
			OnSecondInstanceLaunch: app.onSecondInstanceLaunch,
		},
		OnStartup: app.startup,
		// §30: 關閉視窗時攔截，讓前端顯示「存成 sub」對話框
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			return app.beforeClose(ctx)
		},
		// 釘選視窗是子行程：主程式退出時一併收掉，避免孤兒視窗。
		OnShutdown: func(ctx context.Context) {
			popouts.shutdown()
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
