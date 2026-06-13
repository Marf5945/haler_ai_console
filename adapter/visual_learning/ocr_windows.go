//go:build windows

package visual_learning

// windowsNativeOCRProvider is intentionally disabled for the first pass.
//
// Windows native OCR exists through Microsoft APIs such as Windows.Media.Ocr,
// but those WinRT APIs require package identity for desktop apps. The current
// Wails Windows build is a plain desktop executable, so this provider exposes a
// clear capability status without adding a brittle dependency or pretending OCR
// is available.
type windowsNativeOCRProvider struct{}

func NewNativeOCRProvider() OCRProvider {
	return windowsNativeOCRProvider{}
}

func (windowsNativeOCRProvider) Status() OCRStatus {
	return OCRStatus{
		Available: false,
		Platform:  "windows",
		Source:    "windows-native",
		Reason:    "Windows.Media.Ocr requires package identity for desktop apps; native OCR is not enabled in this Wails exe build",
	}
}

func (p windowsNativeOCRProvider) Recognize(imageData []byte) ([]OCRResult, error) {
	return emptyOCRResult(p.Status().Source, p.Status().Reason)
}
