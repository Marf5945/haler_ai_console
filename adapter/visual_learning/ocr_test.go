package visual_learning

import (
	"errors"
	"testing"
)

func TestNativeOCRProviderStatus(t *testing.T) {
	status := NewNativeOCRProvider().Status()
	if status.Source == "" {
		t.Fatalf("expected OCR status source")
	}
	if status.Platform == "" {
		t.Fatalf("expected OCR status platform")
	}
}

func TestNativeOCRRejectsEmptyImage(t *testing.T) {
	_, err := NewNativeOCRProvider().Recognize(nil)
	if err == nil {
		t.Fatalf("expected empty OCR image to fail")
	}
	if !errors.Is(err, ErrOCRUnavailable) {
		t.Fatalf("expected ErrOCRUnavailable, got %v", err)
	}
}
