package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// cspMiddleware 注入 Content-Security-Policy header。
// SEC-12: 防止 XSS / 注入攻擊，同時保留 debug trace 連線。
// connect-src 包含 http://127.0.0.1:* 以支援會自動換 port 的工程師監視頁。
// 當 DEBUG_TRACE_REMOVE 完成時，移除該例外並收緊為 connect-src 'self'。
func cspMiddleware(next http.Handler) http.Handler {
	// style-src 'unsafe-inline': Wails runtime 及部分 UI 框架需要 inline style。
	// img-src data:/blob: : 前端可能使用 data URI 或 blob 圖片。
	const csp = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"connect-src 'self' http://127.0.0.1:*; " +
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
	frontendAssets, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		println("Error:", err.Error())
		return
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "AI Console",
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
		OnStartup:        app.startup,
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
