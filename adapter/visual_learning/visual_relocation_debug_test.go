package visual_learning

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAnchorRelocationDebugOverlay(t *testing.T) {
	width, height := 80, 48
	imageData := make([]byte, width*height*4)
	for i := 0; i+3 < len(imageData); i += 4 {
		imageData[i] = 240
		imageData[i+1] = 240
		imageData[i+2] = 240
		imageData[i+3] = 255
	}
	result := AnchorRelocationResult{
		ExecutionPoint: PixelPoint{X: 40, Y: 24},
		AnchorBBox:     PixelBBox{X: 30, Y: 15, W: 20, H: 18},
		Candidates: []AnchorRelocationCandidate{
			{Source: "yolox", BBox: PixelBBox{X: 28, Y: 14, W: 22, H: 19}},
			{Source: "opencv_shape", BBox: PixelBBox{X: 5, Y: 5, W: 12, H: 12}},
		},
	}
	out := filepath.Join(t.TempDir(), "overlay.png")
	if err := SaveAnchorRelocationDebugOverlay(out, imageData, width, height, result); err != nil {
		t.Fatalf("SaveAnchorRelocationDebugOverlay failed: %v", err)
	}
	file, err := os.Open(out)
	if err != nil {
		t.Fatalf("open overlay: %v", err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode overlay: %v", err)
	}
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Fatalf("overlay size = %dx%d, want %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), width, height)
	}
}
