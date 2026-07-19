//go:build !dev

package main

// contentSecurityPolicy — production CSP（SEC-12 嚴格版，與原設定完全相同）。
// style-src 'unsafe-inline': Wails runtime 及部分 UI 框架需要 inline style。
// img-src data:/blob: : 前端可能使用 data URI 或 blob 圖片。
// script-src 'wasm-unsafe-eval': Inochi2D avatar runtime (inox2d) 需要編譯
// WebAssembly；此指令只放行 wasm 編譯，JS eval() 仍被禁止。
func contentSecurityPolicy() string {
	return "default-src 'self'; " +
		"script-src 'self' 'wasm-unsafe-eval'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"connect-src 'self'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self' data:; " +
		"object-src 'none'; " +
		"base-uri 'self'"
}
