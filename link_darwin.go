//go:build darwin && suppress_duplicate_link_warnings

package main

// 抑制 macOS 新連結器(ld_prime / Xcode 15+)對 Wails cgo 重複帶入
// 函式庫(例如 -lobjc)所發出的：
//   ld: warning: ignoring duplicate libraries: '-lobjc'
// 這純粹是 warning、不影響打包；在此把抑制旗標釘進 darwin link 階段,
// 讓任何建置方式(wails build / wails dev / IDE)都不再出現。
//
// 注意：Go 的 cgo 安全檢查預設不允許 -Wl,... 出現在 #cgo LDFLAGS，
// 因此此檔只能在明確允許該 flag 的建置環境中 opt-in。

/*
#cgo LDFLAGS: -Wl,-no_warn_duplicate_libraries
*/
import "C"
