//go:build !darwin && !windows && !linux

package main

// applyFloatingAvatarWindow 在其餘平台（非 macOS / Windows / Linux）暫為 no-op。
//
// macOS 見 floating_avatar_darwin.go、Linux 見 floating_avatar_linux.go、
// Windows 見 floating_avatar_windows.go。這些以外的平台進入後台頭像時，
// 視窗縮放與 always-on-top 仍沿用 Wails runtime（WindowSetSize /
// WindowSetAlwaysOnTop），只是不做原生無框透明處理。
func applyFloatingAvatarWindow(on bool) {}
