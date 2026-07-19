//go:build dev

package main

// contentSecurityPolicy — dev CSP（僅 `wails dev` 編譯進來；production 走
// csp_policy_prod.go 的嚴格版，SEC-12 邊界不變）。
//
// 放寬原因：wails dev 的 vite 開發伺服器需要
//   1. inline preamble script（@vitejs/plugin-react）→ script-src 'unsafe-inline'
//   2. HMR WebSocket（ws://localhost:5173、ws://wails.localhost:*）→ connect-src ws:
//   3. 從 http://localhost:5173 載入模組 → script/connect 放行該 origin
func contentSecurityPolicy() string {
	return "default-src 'self' http://localhost:5173; " +
		"script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval' http://localhost:5173; " +
		"style-src 'self' 'unsafe-inline' http://localhost:5173; " +
		"connect-src 'self' ws: http://localhost:5173; " +
		"img-src 'self' data: blob: http://localhost:5173; " +
		"font-src 'self' data: http://localhost:5173; " +
		"object-src 'none'; " +
		"base-uri 'self'"
}
