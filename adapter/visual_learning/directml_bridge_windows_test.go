//go:build windows

package visual_learning

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectMLBridgeLoadAndInferOptIn(t *testing.T) {
	if os.Getenv("UI_CONSOLE_TEST_DIRECTML") != "1" {
		t.Skip("set UI_CONSOLE_TEST_DIRECTML=1 to verify the bundled ONNX Runtime DirectML path")
	}

	status := CheckDirectMLRuntime()
	if !status.Available {
		t.Fatalf("DirectML runtime unavailable: %s", status.Reason)
	}

	engine := NewInferenceEngine()
	defer engine.Close()

	modelPath := filepath.Join("..", "..", "assets", "models", "yolox_button_s.onnx")
	if err := engine.LoadModel(modelPath); err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}

	size := DefaultYOLOXButtonSConfig.InputSize
	rgba := make([]byte, size*size*4)
	for i := 3; i < len(rgba); i += 4 {
		rgba[i] = 255
	}

	raw, err := engine.Infer(rgba, size, size)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}
	if err := raw.Validate(); err != nil {
		t.Fatalf("RawTensor validation failed: %v", err)
	}
}
