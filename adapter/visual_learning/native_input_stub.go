//go:build !windows && !darwin

package visual_learning

import "fmt"

// NativeInput is the cross-platform placeholder. macOS will later use the
// same methods with Quartz/CGEvent so the Wails/frontend contract stays stable.
type NativeInput struct{}

func NewNativeInput() *NativeInput {
	return &NativeInput{}
}

func (n *NativeInput) Start(onClick func(NativeClickEvent)) error {
	return fmt.Errorf("native input recorder is not implemented on this platform")
}

func (n *NativeInput) Stop() error {
	return nil
}

func (n *NativeInput) Click(step LearningReplayStep) NativeReplayResult {
	return NativeReplayResult{
		OK:      false,
		Skipped: true,
		Method:  "native",
		Index:   step.Index,
		Label:   step.Label,
		X:       step.X,
		Y:       step.Y,
		Error:   fmt.Sprintf("native input executor is not implemented on this platform"),
	}
}

func (n *NativeInput) MoveCursorOnly(step LearningReplayStep) NativeReplayResult {
	return NativeReplayResult{
		OK:      false,
		Skipped: true,
		Method:  "native_preview",
		Index:   step.Index,
		Label:   step.Label,
		X:       step.X,
		Y:       step.Y,
		Error:   fmt.Sprintf("native input preview is not implemented on this platform"),
	}
}

func (n *NativeInput) CaptureWindow(hwnd uintptr) (WindowCapture, error) {
	return WindowCapture{}, fmt.Errorf("native window capture is not implemented on this platform")
}

func (n *NativeInput) ResolveWindow(handle uintptr, process, title string) (ResolvedWindow, bool) {
	return ResolvedWindow{}, false
}
