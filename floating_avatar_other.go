//go:build !darwin && !windows

package main

// applyFloatingAvatarWindow 在非 macOS 平台暫為 no-op。
//
// Windows 的真浮窗（WS_EX_LAYERED | WS_EX_TOPMOST + per-pixel alpha）
// 規劃於 v3.1.16 Phase 3 stub；目前 Windows/Linux 進入後台頭像時，
// 視窗縮放與 always-on-top 仍沿用 Wails runtime（WindowSetSize /
// WindowSetAlwaysOnTop），只是不做原生無框透明處理。
func applyFloatingAvatarWindow(on bool) {}
