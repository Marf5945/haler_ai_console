//go:build !darwin && !windows

package visual_learning

type nativeOCRStubProvider struct{}

func NewNativeOCRProvider() OCRProvider {
	return nativeOCRStubProvider{}
}

func (nativeOCRStubProvider) Status() OCRStatus {
	return OCRStatus{
		Available: false,
		Platform:  "unsupported",
		Source:    "disabled",
		Reason:    "platform-native OCR is only wired for macOS in this build",
	}
}

func (p nativeOCRStubProvider) Recognize(imageData []byte) ([]OCRResult, error) {
	return emptyOCRResult(p.Status().Source, p.Status().Reason)
}
