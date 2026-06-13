//go:build !darwin && !windows && !linux

package main

func startNativeFileDrag(path string) nativeDragResult {
	return nativeDragResult{
		Status:           nativeDragStatusFailed,
		FallbackRequired: true,
		Message:          "此平台尚未支援原生拖曳匯出，已改用資料夾選擇器",
	}
}
