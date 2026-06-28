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
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
