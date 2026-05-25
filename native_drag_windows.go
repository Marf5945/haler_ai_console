//go:build windows

package main

func startNativeFileDrag(path string) nativeDragResult {
	// Windows will use OLE/COM CF_HDROP in the next bridge pass.
	return nativeDragResult{
		Status:           nativeDragStatusFailed,
		FallbackRequired: true,
		Message:          "Windows 原生 CF_HDROP bridge 尚未接上，已改用資料夾選擇器",
	}
}
