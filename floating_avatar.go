// floating_avatar.go — v3.1.16 Phase 3：後台浮動頭像的原生視窗切換（跨平台入口）。
//
// 進入後台頭像模式時，主視窗需要從「一般有框、不透明」切成
// 「無框、透明、永遠置頂、可跨所有桌面/Space」的桌面浮窗；
// 還原主系統時再切回一般視窗。實際視窗操作依平台分檔：
//   - floating_avatar_darwin.go：macOS 真實作（cgo / AppKit）
//   - floating_avatar_other.go ：Windows/Linux stub（暫為 no-op）
//
// 前端在 enterFloatingAvatarMode / restoreFloatingAvatarWindow 透過
// 這兩個 bound method 呼叫。
package main

// EnterFloatingAvatarNative 進入後台頭像：把主視窗切成無框透明置頂浮窗。
func (a *App) EnterFloatingAvatarNative() {
	applyFloatingAvatarWindow(true)
}

// ExitFloatingAvatarNative 還原主系統：恢復一般有框、不透明視窗。
func (a *App) ExitFloatingAvatarNative() {
	applyFloatingAvatarWindow(false)
}
