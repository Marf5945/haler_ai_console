package visual_learning

import (
	"errors"
	"fmt"
)

// ErrOCRUnavailable means native OCR is not usable in the current runtime.
// OCR is an optional hint for visual anchors, never a required click signal.
var ErrOCRUnavailable = errors.New("native OCR unavailable")

// OCRStatus describes whether the platform-native OCR provider can be used.
type OCRStatus struct {
	Available bool   `json:"available"`
	Platform  string `json:"platform"`
	Source    string `json:"source"`
	Reason    string `json:"reason,omitempty"`
}

// OCRResult is one recognized text span from a cropped UI image.
// BBox is normalized [x, y, width, height]. Providers may leave it empty.
type OCRResult struct {
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"`
	BBox       []float64 `json:"bbox,omitempty"`
}

// OCRProvider is the small platform boundary for native OCR.
// Implementations must use OS-provided APIs or return ErrOCRUnavailable.
type OCRProvider interface {
	Status() OCRStatus
	Recognize(imageData []byte) ([]OCRResult, error)
}

func emptyOCRResult(source, reason string) ([]OCRResult, error) {
	return nil, fmt.Errorf("%w: %s (%s)", ErrOCRUnavailable, reason, source)
}
