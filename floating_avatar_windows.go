//go:build windows

package main

// WebView2 does not reliably honor layered-window transparency when Wails
// resizes the main window into a compact avatar popup. The frontend disables
// the native compact-window path on Windows, and this no-op prevents accidental
// calls from producing a black always-on-top rectangle.
func applyFloatingAvatarWindow(on bool) {}
