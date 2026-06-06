package visual_learning

import (
	"encoding/base64"
	"math"
	"testing"
)

func TestResolveWindowsClickAnchorPrefersContainingYOLOXBox(t *testing.T) {
	imageData := solidRGBA(200, 100)
	detection := DetectorResult{
		Backend: "directml",
		Proposals: []RegionProposal{
			{
				BBox:       BBox{X: 0.05, Y: 0.05, W: 0.9, H: 0.9},
				RawScore:   0.99,
				ProposalID: "container",
			},
			{
				BBox:       BBox{X: 0.40, Y: 0.30, W: 0.30, H: 0.20},
				RawScore:   0.86,
				ProposalID: "button",
			},
		},
	}

	result, err := ResolveWindowsClickAnchor(imageData, 200, 100, 95, 40, detection, PipelineResult{}, WindowsClickAnchorOptions{})
	if err != nil {
		t.Fatalf("ResolveWindowsClickAnchor returned error: %v", err)
	}
	if result.Mode != "yolox_contains" {
		t.Fatalf("mode = %q, want yolox_contains", result.Mode)
	}
	if result.AnchorBBox != (PixelBBox{X: 80, Y: 30, W: 60, H: 20}) {
		t.Fatalf("anchor bbox = %+v", result.AnchorBBox)
	}
	if result.ExecutionHint != "click_bbox_center" {
		t.Fatalf("execution hint = %q", result.ExecutionHint)
	}
	if result.ExecutionPoint != (PixelPoint{X: 110, Y: 40}) {
		t.Fatalf("execution point = %+v", result.ExecutionPoint)
	}
	if result.NeedsReview {
		t.Fatalf("containing YOLOX selection should not be marked degraded review")
	}
	if result.CropPNGBase64 == "" {
		t.Fatalf("expected color crop png base64")
	}
	if _, err := base64.StdEncoding.DecodeString(result.CropPNGBase64); err != nil {
		t.Fatalf("crop is not valid base64: %v", err)
	}
}

func TestResolveWindowsClickAnchorUsesShapeFallbackWhenYOLOXMisses(t *testing.T) {
	imageData := solidRGBA(300, 200)
	shape := PipelineResult{
		Candidates: []UIFingerprint{
			NewUIFingerprint("shape-button", "pure_go_pipeline", BBox{X: 0.30, Y: 0.40, W: 0.20, H: 0.10}, 0.62),
		},
	}

	result, err := ResolveWindowsClickAnchor(imageData, 300, 200, 96, 88, DetectorResult{Backend: "directml"}, shape, WindowsClickAnchorOptions{})
	if err != nil {
		t.Fatalf("ResolveWindowsClickAnchor returned error: %v", err)
	}
	if result.Mode != "shape_fallback" {
		t.Fatalf("mode = %q, want shape_fallback", result.Mode)
	}
	if result.AnchorBBox != (PixelBBox{X: 90, Y: 80, W: 60, H: 20}) {
		t.Fatalf("anchor bbox = %+v", result.AnchorBBox)
	}
	if !result.NeedsReview {
		t.Fatalf("shape fallback should be marked degraded review")
	}
	if result.OCRStatus != "not_used" {
		t.Fatalf("ocr status = %q", result.OCRStatus)
	}
}

func TestResolveWindowsClickAnchorUsesTightShapeWhenYOLOXSelectsTaskbarGroup(t *testing.T) {
	imageData := solidRGBA(1920, 60)
	detection := DetectorResult{
		Backend: "directml",
		Proposals: []RegionProposal{
			{
				BBox:       BBox{X: 1017.0 / 1920.0, Y: 14.0 / 60.0, W: 416.0 / 1920.0, H: 30.0 / 60.0},
				RawScore:   0.39,
				ProposalID: "taskbar-icon-group",
			},
		},
	}
	shape := PipelineResult{
		Candidates: []UIFingerprint{
			NewUIFingerprint("chrome-icon", "pure_go_pipeline_icon", BBox{X: 1142.0 / 1920.0, Y: 16.0 / 60.0, W: 28.0 / 1920.0, H: 28.0 / 60.0}, 0.62),
		},
	}

	result, err := ResolveWindowsClickAnchor(imageData, 1920, 60, 1144, 40, detection, shape, WindowsClickAnchorOptions{})
	if err != nil {
		t.Fatalf("ResolveWindowsClickAnchor returned error: %v", err)
	}
	if result.Mode != "shape_fallback" {
		t.Fatalf("mode = %q, want shape_fallback", result.Mode)
	}
	if result.AnchorBBox != (PixelBBox{X: 1142, Y: 16, W: 28, H: 28}) {
		t.Fatalf("anchor bbox = %+v", result.AnchorBBox)
	}
	if result.ExecutionPoint != (PixelPoint{X: 1156, Y: 30}) {
		t.Fatalf("execution point = %+v", result.ExecutionPoint)
	}
	if !result.NeedsReview {
		t.Fatalf("OpenCV/icon override should be marked review")
	}
}

func TestResolveWindowsClickAnchorManualBoxWhenNoObjectFound(t *testing.T) {
	imageData := solidRGBA(120, 80)

	result, err := ResolveWindowsClickAnchor(imageData, 120, 80, 8, 6, DetectorResult{}, PipelineResult{}, WindowsClickAnchorOptions{ManualBoxSize: 20})
	if err != nil {
		t.Fatalf("ResolveWindowsClickAnchor returned error: %v", err)
	}
	if result.Mode != "manual_click_box" {
		t.Fatalf("mode = %q, want manual_click_box", result.Mode)
	}
	if result.ExecutionHint != "fast_click_original_point" {
		t.Fatalf("execution hint = %q", result.ExecutionHint)
	}
	if result.ExecutionPoint != (PixelPoint{X: 8, Y: 6}) {
		t.Fatalf("execution point = %+v", result.ExecutionPoint)
	}
	if result.AnchorBBox != (PixelBBox{X: 0, Y: 0, W: 18, H: 16}) {
		t.Fatalf("manual bbox = %+v", result.AnchorBBox)
	}
	if !result.NeedsReview {
		t.Fatalf("manual click box should be marked review")
	}
}

func TestResolveWindowsClickAnchorNearMissYOLOX(t *testing.T) {
	imageData := solidRGBA(200, 100)
	detection := DetectorResult{
		Backend: "directml",
		Proposals: []RegionProposal{
			{
				BBox:       BBox{X: 0.50, Y: 0.40, W: 0.20, H: 0.20},
				RawScore:   0.88,
				ProposalID: "near",
			},
		},
	}

	result, err := ResolveWindowsClickAnchor(imageData, 200, 100, 96, 50, detection, PipelineResult{}, WindowsClickAnchorOptions{NearMissPx: 5})
	if err != nil {
		t.Fatalf("ResolveWindowsClickAnchor returned error: %v", err)
	}
	if result.Mode != "yolox_near_miss" {
		t.Fatalf("mode = %q, want yolox_near_miss", result.Mode)
	}
	if !result.NeedsReview {
		t.Fatalf("near miss should be marked review")
	}
}

func TestResolveAnchorRelocationUsesYOLOXAboveThreshold(t *testing.T) {
	anchor := &WindowsClickAnchorResult{
		OK:             true,
		Mode:           "yolox_contains",
		ExecutionPoint: PixelPoint{X: 110, Y: 40},
		AnchorBBox:     PixelBBox{X: 80, Y: 30, W: 60, H: 20},
	}
	detection := DetectorResult{
		Backend: "directml",
		Proposals: []RegionProposal{
			{BBox: BBox{X: 0.41, Y: 0.31, W: 0.31, H: 0.21}, RawScore: 0.86, ProposalID: "button"},
		},
	}
	result := ResolveAnchorRelocation(anchor, 200, 100, 400, 200, detection, PipelineResult{}, AnchorRelocationOptions{ConfidenceThreshold: 0.5})
	if !result.OK {
		t.Fatalf("expected YOLOX relocation to pass: %#v", result)
	}
	if result.NeedsConfirmation {
		t.Fatalf("YOLOX high confidence should not require confirmation")
	}
	if result.ExecutionPoint.X == 0 || result.ExecutionPoint.Y == 0 {
		t.Fatalf("expected execution point, got %#v", result.ExecutionPoint)
	}
}

func TestResolveAnchorRelocationRequiresConfirmationForOpenCV(t *testing.T) {
	anchor := &WindowsClickAnchorResult{
		OK:             true,
		Mode:           "yolox_contains",
		ExecutionPoint: PixelPoint{X: 110, Y: 40},
		AnchorBBox:     PixelBBox{X: 80, Y: 30, W: 60, H: 20},
	}
	shape := PipelineResult{
		Candidates: []UIFingerprint{
			NewUIFingerprint("shape-button", "pure_go_pipeline", BBox{X: 0.41, Y: 0.31, W: 0.31, H: 0.21}, 0.9),
		},
	}
	result := ResolveAnchorRelocation(anchor, 200, 100, 400, 200, DetectorResult{Backend: "directml"}, shape, AnchorRelocationOptions{ConfidenceThreshold: 0.5})
	if result.OK {
		t.Fatalf("OpenCV fallback should not auto-pass: %#v", result)
	}
	if !result.NeedsConfirmation {
		t.Fatalf("OpenCV fallback should require confirmation: %#v", result)
	}
}

func TestResolveAnchorRelocationUsesRecordedCropAppearance(t *testing.T) {
	recordedImage := solidRGBA(200, 80)
	fillRGBA(recordedImage, 200, PixelBBox{X: 80, Y: 20, W: 30, H: 30}, 220, 40, 40)
	currentImage := solidRGBA(200, 80)
	fillRGBA(currentImage, 200, PixelBBox{X: 40, Y: 20, W: 30, H: 30}, 220, 40, 40)
	fillRGBA(currentImage, 200, PixelBBox{X: 80, Y: 20, W: 30, H: 30}, 40, 80, 220)

	anchor := &WindowsClickAnchorResult{
		OK:             true,
		Mode:           "shape_fallback",
		ExecutionPoint: PixelPoint{X: 95, Y: 35},
		AnchorBBox:     PixelBBox{X: 80, Y: 20, W: 30, H: 30},
		CropBBox:       PixelBBox{X: 72, Y: 12, W: 46, H: 46},
		CropPNGBase64:  cropPNGBase64(recordedImage, 200, 80, PixelBBox{X: 72, Y: 12, W: 46, H: 46}),
	}
	shape := PipelineResult{
		Candidates: []UIFingerprint{
			NewUIFingerprint("matching-red", "pure_go_pipeline", BBox{X: 0.20, Y: 0.25, W: 0.15, H: 0.375}, 0.9),
			NewUIFingerprint("nearby-blue", "pure_go_pipeline", BBox{X: 0.40, Y: 0.25, W: 0.15, H: 0.375}, 0.9),
		},
	}

	result := ResolveAnchorRelocation(anchor, 200, 80, 200, 80, DetectorResult{Backend: "directml"}, shape, AnchorRelocationOptions{
		ConfidenceThreshold: 0.5,
		CurrentImageData:    currentImage,
	})
	if result.AnchorBBox.X != 40 {
		t.Fatalf("expected appearance match at x=40, got bbox %#v candidates %#v", result.AnchorBBox, result.Candidates)
	}
	if result.Candidates[0].AppearanceScore <= result.Candidates[len(result.Candidates)-1].AppearanceScore {
		t.Fatalf("expected matching candidate to beat non-matching appearance: %#v", result.Candidates)
	}
}

func TestResolveAnchorRelocationUsesSpatialAppearanceWhenHistogramTies(t *testing.T) {
	recordedImage := solidRGBA(240, 100)
	fillRGBA(recordedImage, 240, PixelBBox{X: 80, Y: 20, W: 20, H: 40}, 230, 40, 40)
	fillRGBA(recordedImage, 240, PixelBBox{X: 100, Y: 20, W: 20, H: 40}, 40, 90, 230)

	currentImage := solidRGBA(240, 100)
	fillRGBA(currentImage, 240, PixelBBox{X: 40, Y: 20, W: 20, H: 40}, 40, 90, 230)
	fillRGBA(currentImage, 240, PixelBBox{X: 60, Y: 20, W: 20, H: 40}, 230, 40, 40)
	fillRGBA(currentImage, 240, PixelBBox{X: 140, Y: 20, W: 20, H: 40}, 230, 40, 40)
	fillRGBA(currentImage, 240, PixelBBox{X: 160, Y: 20, W: 20, H: 40}, 40, 90, 230)

	anchor := &WindowsClickAnchorResult{
		OK:             true,
		Mode:           "yolox_contains",
		ExecutionPoint: PixelPoint{X: 100, Y: 40},
		AnchorBBox:     PixelBBox{X: 80, Y: 20, W: 40, H: 40},
		CropBBox:       PixelBBox{X: 80, Y: 20, W: 40, H: 40},
		CropPNGBase64:  cropPNGBase64(recordedImage, 240, 100, PixelBBox{X: 80, Y: 20, W: 40, H: 40}),
	}
	detection := DetectorResult{
		Backend: "directml",
		Proposals: []RegionProposal{
			{BBox: BBox{X: 40.0 / 240.0, Y: 0.20, W: 40.0 / 240.0, H: 0.40}, RawScore: 0.95, ProposalID: "same-hist-wrong-order"},
			{BBox: BBox{X: 140.0 / 240.0, Y: 0.20, W: 40.0 / 240.0, H: 0.40}, RawScore: 0.90, ProposalID: "same-hist-right-order"},
		},
	}

	result := ResolveAnchorRelocation(anchor, 240, 100, 240, 100, detection, PipelineResult{}, AnchorRelocationOptions{
		ConfidenceThreshold: 0.5,
		CurrentImageData:    currentImage,
	})
	if !result.OK {
		t.Fatalf("expected YOLOX relocation to pass: %#v", result)
	}
	if result.AnchorBBox.X != 140 {
		t.Fatalf("expected spatial appearance match at x=140, got bbox %#v candidates %#v", result.AnchorBBox, result.Candidates)
	}
	if result.Candidates[0].AppearanceScore <= result.Candidates[len(result.Candidates)-1].AppearanceScore {
		t.Fatalf("expected spatial signature to break histogram tie: %#v", result.Candidates)
	}
}

func TestResolveAnchorRelocationUsesCropTemplateFallback(t *testing.T) {
	recordedImage := solidRGBA(160, 60)
	drawIconPattern(recordedImage, 160, 48, 18)
	currentImage := solidRGBA(220, 60)
	drawIconPattern(currentImage, 220, 128, 18)

	anchor := &WindowsClickAnchorResult{
		OK:             true,
		Mode:           "shape_fallback",
		ExecutionPoint: PixelPoint{X: 56, Y: 26},
		AnchorBBox:     PixelBBox{X: 48, Y: 18, W: 16, H: 16},
		CropBBox:       PixelBBox{X: 40, Y: 10, W: 32, H: 32},
		CropPNGBase64:  cropPNGBase64(recordedImage, 160, 60, PixelBBox{X: 40, Y: 10, W: 32, H: 32}),
	}

	result := ResolveAnchorRelocation(anchor, 160, 60, 220, 60, DetectorResult{Backend: "directml"}, PipelineResult{}, AnchorRelocationOptions{
		ConfidenceThreshold: 0.5,
		CurrentImageData:    currentImage,
	})
	if !result.NeedsConfirmation {
		t.Fatalf("crop fallback should require confirmation: %#v", result)
	}
	if result.Method != "opencv_crop_relocation" {
		t.Fatalf("method = %q, want opencv_crop_relocation", result.Method)
	}
	if result.AnchorBBox.X != 128 || math.Abs(float64(result.AnchorBBox.Y-18)) > 2 {
		t.Fatalf("expected crop template to relocate icon bbox, got %#v candidates %#v", result.AnchorBBox, result.Candidates)
	}
}

func solidRGBA(width, height int) []byte {
	data := make([]byte, width*height*4)
	for i := 0; i < width*height; i++ {
		offset := i * 4
		data[offset] = 20
		data[offset+1] = 120
		data[offset+2] = 220
		data[offset+3] = 255
	}
	return data
}

func drawIconPattern(data []byte, width, x, y int) {
	fillRGBA(data, width, PixelBBox{X: x, Y: y, W: 8, H: 8}, 230, 40, 40)
	fillRGBA(data, width, PixelBBox{X: x + 8, Y: y, W: 8, H: 8}, 40, 180, 80)
	fillRGBA(data, width, PixelBBox{X: x, Y: y + 8, W: 8, H: 8}, 40, 90, 230)
	fillRGBA(data, width, PixelBBox{X: x + 8, Y: y + 8, W: 8, H: 8}, 245, 210, 40)
}

func fillRGBA(data []byte, width int, box PixelBBox, r, g, b byte) {
	for y := box.Y; y < box.Y+box.H; y++ {
		for x := box.X; x < box.X+box.W; x++ {
			offset := (y*width + x) * 4
			data[offset] = r
			data[offset+1] = g
			data[offset+2] = b
			data[offset+3] = 255
		}
	}
}
