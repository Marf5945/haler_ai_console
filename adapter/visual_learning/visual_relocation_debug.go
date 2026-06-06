package visual_learning

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// SaveAnchorRelocationDebugOverlay writes the captured replay image with all
// relocation candidates overlaid, so click drift can be inspected after replay.
func SaveAnchorRelocationDebugOverlay(path string, imageData []byte, width, height int, result AnchorRelocationResult) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("debug overlay: invalid image size %dx%d", width, height)
	}
	need := width * height * 4
	if len(imageData) < need {
		return fmt.Errorf("debug overlay: image data too short: got %d need %d", len(imageData), need)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, imageData[:need])

	for _, candidate := range result.Candidates {
		candidateColor := color.RGBA{255, 170, 0, 255}
		if candidate.Source == "yolox" {
			candidateColor = color.RGBA{255, 220, 0, 255}
		}
		drawDebugBox(img, candidate.BBox, candidateColor, 2)
	}
	if result.AnchorBBox.W > 0 && result.AnchorBBox.H > 0 {
		drawDebugBox(img, result.AnchorBBox, color.RGBA{0, 220, 90, 255}, 3)
	}
	drawDebugCross(img, result.ExecutionPoint, color.RGBA{255, 30, 30, 255}, 10)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func drawDebugBox(img *image.RGBA, box PixelBBox, c color.RGBA, thickness int) {
	bounds := img.Bounds()
	box = clampBBox(box, bounds.Dx(), bounds.Dy())
	if box.W <= 0 || box.H <= 0 {
		return
	}
	for i := 0; i < thickness; i++ {
		x0 := box.X + i
		y0 := box.Y + i
		x1 := box.X + box.W - 1 - i
		y1 := box.Y + box.H - 1 - i
		for x := x0; x <= x1; x++ {
			setDebugPixel(img, x, y0, c)
			setDebugPixel(img, x, y1, c)
		}
		for y := y0; y <= y1; y++ {
			setDebugPixel(img, x0, y, c)
			setDebugPixel(img, x1, y, c)
		}
	}
}

func drawDebugCross(img *image.RGBA, p PixelPoint, c color.RGBA, radius int) {
	for delta := -radius; delta <= radius; delta++ {
		setDebugPixel(img, p.X+delta, p.Y, c)
		setDebugPixel(img, p.X, p.Y+delta, c)
	}
}

func setDebugPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetRGBA(x, y, c)
	}
}
