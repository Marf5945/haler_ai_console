//go:build darwin

package visual_learning

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework Vision -framework CoreGraphics
#include "coreml_bridge_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"runtime"
	"unsafe"
)

// appleVisionOCRProvider uses Apple's Vision framework for OCR on macOS.
// It is optional: visual replay must treat recognized text as a hint, not as
// the authority for deciding where to click.
type appleVisionOCRProvider struct{}

func NewNativeOCRProvider() OCRProvider {
	return appleVisionOCRProvider{}
}

func (appleVisionOCRProvider) Status() OCRStatus {
	return OCRStatus{
		Available: true,
		Platform:  "darwin",
		Source:    "apple-vision",
	}
}

func (p appleVisionOCRProvider) Recognize(imageData []byte) ([]OCRResult, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("%w: empty image data", ErrOCRUnavailable)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var outJSON *C.char
	var cErrMsg *C.char
	rc := C.VisionOCR_RecognizeImage(
		(*C.uint8_t)(unsafe.Pointer(&imageData[0])),
		C.int(len(imageData)),
		&outJSON,
		&cErrMsg,
	)
	if rc != 0 {
		reason := "unknown Apple Vision OCR error"
		if cErrMsg != nil {
			reason = C.GoString(cErrMsg)
			C.CoreML_FreeString(cErrMsg)
		}
		return nil, fmt.Errorf("%w: %s", ErrOCRUnavailable, reason)
	}
	defer C.CoreML_FreeString(outJSON)

	var results []OCRResult
	if err := json.Unmarshal([]byte(C.GoString(outJSON)), &results); err != nil {
		return nil, fmt.Errorf("apple vision OCR: decode result: %w", err)
	}
	for i := range results {
		results[i].Source = p.Status().Source
	}
	return results, nil
}
