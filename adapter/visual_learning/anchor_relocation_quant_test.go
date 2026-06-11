package visual_learning

import (
	"image"
	"image/color"
	"testing"
)

func TestVisualSignatureQuantizedSelfSimilarity(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	sig := imageSignature(img)
	redBin := colorBin(255, 0, 0)
	if sig.hist[redBin] == 0 {
		t.Fatalf("red histogram bin was not populated: %+v", sig.hist)
	}
	if sig.grid[0] != 65535 || sig.grid[1] != 0 || sig.grid[2] != 0 {
		t.Fatalf("expected quantized red grid cell, got r=%d g=%d b=%d", sig.grid[0], sig.grid[1], sig.grid[2])
	}
	if score := visualSignatureSimilarity(sig, sig); score < 0.999 {
		t.Fatalf("self similarity = %v, want ~1", score)
	}
}

func TestColorGridSimilarityQuantizedExtremes(t *testing.T) {
	var white, black [192]uint16
	for i := range white {
		white[i] = 65535
	}
	if score := colorGridSimilarity(white, white); score < 0.999 {
		t.Fatalf("identical white grid score = %v, want ~1", score)
	}
	if score := colorGridSimilarity(white, black); score > 0.001 {
		t.Fatalf("opposite grids score = %v, want ~0", score)
	}
}
