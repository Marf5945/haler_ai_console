package visual_learning

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"sort"
	"strings"
)

type AnchorRelocationOptions struct {
	ConfidenceThreshold float64 `json:"confidence_threshold,omitempty"`
	AppearanceThreshold float64 `json:"appearance_threshold,omitempty"`
	CurrentImageData    []byte  `json:"-"`
}

type AnchorRelocationCandidate struct {
	Source            string    `json:"source"`
	BBox              PixelBBox `json:"bbox"`
	Confidence        float64   `json:"confidence"`
	AppearanceScore   float64   `json:"appearance_score,omitempty"`
	PositionScore     float64   `json:"position_score"`
	SizeScore         float64   `json:"size_score"`
	SelectionScore    float64   `json:"selection_score"`
	NeedsConfirmation bool      `json:"needs_confirmation,omitempty"`
}

type AnchorRelocationResult struct {
	OK                bool                        `json:"ok"`
	NeedsConfirmation bool                        `json:"needs_confirmation,omitempty"`
	Method            string                      `json:"method"`
	Reason            string                      `json:"reason,omitempty"`
	DebugImagePath    string                      `json:"debug_image_path,omitempty"`
	Confidence        float64                     `json:"confidence"`
	OriginalPoint     PixelPoint                  `json:"original_point"`
	ExecutionPoint    PixelPoint                  `json:"execution_point"`
	AnchorBBox        PixelBBox                   `json:"anchor_bbox"`
	DetectorBackend   string                      `json:"detector_backend,omitempty"`
	DetectorDegraded  bool                        `json:"detector_degraded,omitempty"`
	Candidates        []AnchorRelocationCandidate `json:"candidates,omitempty"`
}

func ResolveAnchorRelocation(anchor *WindowsClickAnchorResult, recordedWidth, recordedHeight, currentWidth, currentHeight int, detection DetectorResult, shape PipelineResult, options AnchorRelocationOptions) AnchorRelocationResult {
	threshold := options.ConfidenceThreshold
	if threshold <= 0 {
		threshold = 0.5
	}
	appearanceThreshold := options.AppearanceThreshold
	if appearanceThreshold <= 0 {
		appearanceThreshold = 0.58
	}
	result := AnchorRelocationResult{
		Method:           "visual_relocation",
		DetectorBackend:  detection.Backend,
		DetectorDegraded: detection.Degraded,
	}
	if anchor == nil || !anchor.OK {
		result.Reason = "recorded step has no usable visual anchor"
		return result
	}
	if recordedWidth <= 0 || recordedHeight <= 0 || currentWidth <= 0 || currentHeight <= 0 {
		result.Reason = fmt.Sprintf("invalid relocation dimensions recorded=%dx%d current=%dx%d", recordedWidth, recordedHeight, currentWidth, currentHeight)
		return result
	}
	recordedBox := anchor.AnchorBBox
	if recordedBox.W <= 0 || recordedBox.H <= 0 {
		recordedBox = manualClickBox(anchor.ExecutionPoint, recordedWidth, recordedHeight, 28)
	}
	result.OriginalPoint = anchor.ExecutionPoint
	result.AnchorBBox = recordedBox

	candidates := relocationCandidates(anchor, detection, shape, currentWidth, currentHeight, recordedBox, recordedWidth, recordedHeight, options.CurrentImageData)
	result.Candidates = candidates
	if len(candidates) == 0 {
		result.Reason = "no YOLOX/OpenCV candidates were available for relocation"
		return result
	}
	best := candidates[0]
	result.Confidence = best.SelectionScore
	result.ExecutionPoint = bboxCenter(best.BBox)
	result.AnchorBBox = best.BBox
	result.Method = best.Source + "_relocation"
	result.NeedsConfirmation = best.NeedsConfirmation
	if best.SelectionScore < threshold {
		result.NeedsConfirmation = true
		result.Reason = fmt.Sprintf("visual relocation confidence %.2f is below threshold %.2f", best.SelectionScore, threshold)
		return result
	}
	if best.Source == "yolox" && best.AppearanceScore > 0 && best.AppearanceScore < appearanceThreshold {
		result.NeedsConfirmation = true
		result.Reason = fmt.Sprintf("YOLOX candidate appearance %.2f is below threshold %.2f", best.AppearanceScore, appearanceThreshold)
		return result
	}
	if best.NeedsConfirmation {
		result.Reason = "OpenCV fallback matched a candidate; confirmation is required before clicking"
		return result
	}
	result.OK = true
	result.Reason = fmt.Sprintf("YOLOX relocation matched candidate with confidence %.2f", best.SelectionScore)
	return result
}

func relocationCandidates(anchor *WindowsClickAnchorResult, detection DetectorResult, shape PipelineResult, currentWidth, currentHeight int, recordedBox PixelBBox, recordedWidth, recordedHeight int, currentImageData []byte) []AnchorRelocationCandidate {
	var raw []WindowsButtonCandidate
	source := "yolox"
	if detection.Degraded {
		source = "opencv"
	}
	raw = append(raw, proposalCandidates(detection.Proposals, source, currentWidth, currentHeight, PixelPoint{})...)
	raw = append(raw, fingerprintCandidates(shape.Candidates, "opencv_shape", currentWidth, currentHeight, PixelPoint{})...)
	raw = append(raw, cropTemplateCandidates(anchor, currentImageData, currentWidth, currentHeight, recordedBox)...)
	recordedAppearance, hasAppearance := recordedCropSignature(anchor)
	out := make([]AnchorRelocationCandidate, 0, len(raw))
	for _, candidate := range raw {
		if candidate.BBox.W <= 0 || candidate.BBox.H <= 0 {
			continue
		}
		pos := relocationPositionScore(recordedBox, recordedWidth, recordedHeight, candidate.BBox, currentWidth, currentHeight)
		size := relocationSizeScore(recordedBox, recordedWidth, recordedHeight, candidate.BBox, currentWidth, currentHeight)
		conf := clampFloat(candidate.Confidence, 0, 1)
		appearance := 0.0
		if hasAppearance && len(currentImageData) >= currentWidth*currentHeight*4 {
			// Recorded color crop helps separate nearby taskbar icons with similar shapes.
			cropBox := scaledCandidateCropBox(anchor, recordedBox, candidate.BBox, currentWidth, currentHeight)
			appearance = visualSignatureSimilarity(recordedAppearance, rgbaSignature(currentImageData, currentWidth, currentHeight, cropBox))
		}
		score := pos*0.50 + conf*0.30 + size*0.20
		if appearance > 0 {
			score = pos*0.15 + appearance*0.70 + conf*0.10 + size*0.05
			if candidate.Source == "opencv_crop" {
				// Crop matching is a fallback for under-trained targets such as
				// taskbar icons; it should not steal a matching YOLO auto path.
				score = pos*0.15 + appearance*0.70 + conf*0.02 + size*0.05
			}
		}
		needsConfirm := candidate.Source != "yolox"
		out = append(out, AnchorRelocationCandidate{
			Source:            candidate.Source,
			BBox:              candidate.BBox,
			Confidence:        roundFloat(conf, 4),
			AppearanceScore:   roundFloat(appearance, 4),
			PositionScore:     roundFloat(pos, 4),
			SizeScore:         roundFloat(size, 4),
			SelectionScore:    roundFloat(score, 4),
			NeedsConfirmation: needsConfirm,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if math.Abs(out[i].SelectionScore-out[j].SelectionScore) > 0.0001 {
			return out[i].SelectionScore > out[j].SelectionScore
		}
		return !out[i].NeedsConfirmation && out[j].NeedsConfirmation
	})
	if len(out) > 8 {
		return out[:8]
	}
	return out
}

func cropTemplateCandidates(anchor *WindowsClickAnchorResult, currentImageData []byte, width, height int, recordedBox PixelBBox) []WindowsButtonCandidate {
	if anchor == nil || strings.TrimSpace(anchor.CropPNGBase64) == "" || len(currentImageData) < width*height*4 {
		return nil
	}
	templateSig, templateWidth, templateHeight, ok := recordedCropTemplate(anchor)
	if !ok || templateWidth <= 0 || templateHeight <= 0 || templateWidth > width || templateHeight > height {
		return nil
	}
	// The saved crop is a compact color template. It catches taskbar/app icons
	// that YOLO may miss until the model is retrained with those labels.
	bestBox, bestScore := findBestCropTemplateMatch(templateSig, currentImageData, width, height, templateWidth, templateHeight)
	if bestScore < 0.52 {
		return nil
	}
	anchorBox := cropMatchToAnchorBox(anchor, bestBox, recordedBox, width, height)
	return []WindowsButtonCandidate{{
		ID:             "opencv-crop-template-0",
		Source:         "opencv_crop",
		BBox:           anchorBox,
		Confidence:     roundFloat(bestScore, 4),
		ContainsClick:  false,
		ClickDistance:  0,
		CenterDistance: 0,
		AreaReasonable: areaReasonable(anchorBox, width, height),
		SelectionScore: roundFloat(bestScore, 4),
	}}
}

func recordedCropTemplate(anchor *WindowsClickAnchorResult) (visualSignature, int, int, bool) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(anchor.CropPNGBase64))
	if err != nil {
		return visualSignature{}, 0, 0, false
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return visualSignature{}, 0, 0, false
	}
	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return visualSignature{}, 0, 0, false
	}
	return imageSignature(img), bounds.Dx(), bounds.Dy(), true
}

func findBestCropTemplateMatch(template visualSignature, imageData []byte, width, height, templateWidth, templateHeight int) (PixelBBox, float64) {
	stride := clampInt(minInt(templateWidth, templateHeight)/8, 2, 10)
	best := PixelBBox{}
	bestScore := -1.0
	for y := 0; y <= height-templateHeight; y += stride {
		for x := 0; x <= width-templateWidth; x += stride {
			box := PixelBBox{X: x, Y: y, W: templateWidth, H: templateHeight}
			score := visualSignatureSimilarity(template, rgbaSignature(imageData, width, height, box))
			if score > bestScore {
				bestScore = score
				best = box
			}
		}
	}
	return refineCropTemplateMatch(template, imageData, width, height, best, stride, bestScore)
}

func refineCropTemplateMatch(template visualSignature, imageData []byte, width, height int, best PixelBBox, radius int, bestScore float64) (PixelBBox, float64) {
	if best.W <= 0 || best.H <= 0 {
		return best, bestScore
	}
	x0 := clampInt(best.X-radius, 0, width-best.W)
	x1 := clampInt(best.X+radius, 0, width-best.W)
	y0 := clampInt(best.Y-radius, 0, height-best.H)
	y1 := clampInt(best.Y+radius, 0, height-best.H)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			box := PixelBBox{X: x, Y: y, W: best.W, H: best.H}
			score := visualSignatureSimilarity(template, rgbaSignature(imageData, width, height, box))
			if score > bestScore {
				bestScore = score
				best = box
			}
		}
	}
	return best, bestScore
}

func cropMatchToAnchorBox(anchor *WindowsClickAnchorResult, matchedCrop PixelBBox, recordedBox PixelBBox, width, height int) PixelBBox {
	if anchor == nil || anchor.CropBBox.W <= 0 || anchor.CropBBox.H <= 0 || recordedBox.W <= 0 || recordedBox.H <= 0 {
		return clampBBox(matchedCrop, width, height)
	}
	scaleX := float64(matchedCrop.W) / float64(anchor.CropBBox.W)
	scaleY := float64(matchedCrop.H) / float64(anchor.CropBBox.H)
	return clampBBox(PixelBBox{
		X: matchedCrop.X + int(math.Round(float64(recordedBox.X-anchor.CropBBox.X)*scaleX)),
		Y: matchedCrop.Y + int(math.Round(float64(recordedBox.Y-anchor.CropBBox.Y)*scaleY)),
		W: int(math.Round(float64(recordedBox.W) * scaleX)),
		H: int(math.Round(float64(recordedBox.H) * scaleY)),
	}, width, height)
}

type visualSignature struct {
	hist [64]uint16
	grid [192]uint16
}

const visualQuantMax = 65535.0

func recordedCropSignature(anchor *WindowsClickAnchorResult) (visualSignature, bool) {
	if anchor == nil || strings.TrimSpace(anchor.CropPNGBase64) == "" {
		return visualSignature{}, false
	}
	raw, err := base64.StdEncoding.DecodeString(anchor.CropPNGBase64)
	if err != nil {
		return visualSignature{}, false
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return visualSignature{}, false
	}
	return imageSignature(img), true
}

func imageSignature(img image.Image) visualSignature {
	return visualSignature{
		hist: imageHistogram(img),
		grid: imageColorGrid(img),
	}
}

func imageHistogram(img image.Image) [64]uint16 {
	var counts [64]float64
	if img == nil {
		return [64]uint16{}
	}
	bounds := img.Bounds()
	total := 0.0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			idx := colorBin(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			counts[idx]++
			total++
		}
	}
	return quantizeHistogram(counts, total)
}

func rgbaHistogram(imageData []byte, width, height int, box PixelBBox) [64]uint16 {
	var counts [64]float64
	box = clampBBox(box, width, height)
	if width <= 0 || height <= 0 || box.W <= 0 || box.H <= 0 || len(imageData) < width*height*4 {
		return [64]uint16{}
	}
	total := 0.0
	for y := box.Y; y < box.Y+box.H; y++ {
		for x := box.X; x < box.X+box.W; x++ {
			offset := (y*width + x) * 4
			idx := colorBin(imageData[offset], imageData[offset+1], imageData[offset+2])
			counts[idx]++
			total++
		}
	}
	return quantizeHistogram(counts, total)
}

func rgbaSignature(imageData []byte, width, height int, box PixelBBox) visualSignature {
	return visualSignature{
		hist: rgbaHistogram(imageData, width, height, box),
		grid: rgbaColorGrid(imageData, width, height, box),
	}
}

func imageColorGrid(img image.Image) [192]uint16 {
	var grid [192]uint16
	if img == nil {
		return grid
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return grid
	}
	for gy := 0; gy < 8; gy++ {
		y := bounds.Min.Y + clampInt((gy*2+1)*h/16, 0, h-1)
		for gx := 0; gx < 8; gx++ {
			x := bounds.Min.X + clampInt((gx*2+1)*w/16, 0, w-1)
			r, g, b, _ := img.At(x, y).RGBA()
			offset := (gy*8 + gx) * 3
			grid[offset] = quantizeUnit(float64(uint8(r>>8)) / 255.0)
			grid[offset+1] = quantizeUnit(float64(uint8(g>>8)) / 255.0)
			grid[offset+2] = quantizeUnit(float64(uint8(b>>8)) / 255.0)
		}
	}
	return grid
}

func rgbaColorGrid(imageData []byte, width, height int, box PixelBBox) [192]uint16 {
	var grid [192]uint16
	box = clampBBox(box, width, height)
	if width <= 0 || height <= 0 || box.W <= 0 || box.H <= 0 || len(imageData) < width*height*4 {
		return grid
	}
	for gy := 0; gy < 8; gy++ {
		y := box.Y + clampInt((gy*2+1)*box.H/16, 0, box.H-1)
		for gx := 0; gx < 8; gx++ {
			x := box.X + clampInt((gx*2+1)*box.W/16, 0, box.W-1)
			src := (y*width + x) * 4
			offset := (gy*8 + gx) * 3
			grid[offset] = quantizeUnit(float64(imageData[src]) / 255.0)
			grid[offset+1] = quantizeUnit(float64(imageData[src+1]) / 255.0)
			grid[offset+2] = quantizeUnit(float64(imageData[src+2]) / 255.0)
		}
	}
	return grid
}

func colorBin(r, g, b uint8) int {
	return int(r/64)*16 + int(g/64)*4 + int(b/64)
}

func quantizeHistogram(counts [64]float64, total float64) [64]uint16 {
	var hist [64]uint16
	if total <= 0 {
		return hist
	}
	for i, count := range counts {
		hist[i] = quantizeUnit(count / total)
	}
	return hist
}

func histogramSimilarity(a, b [64]uint16) float64 {
	score := 0.0
	for i := range a {
		if a[i] < b[i] {
			score += float64(a[i]) / visualQuantMax
		} else {
			score += float64(b[i]) / visualQuantMax
		}
	}
	return clampFloat(score, 0, 1)
}

func visualSignatureSimilarity(a, b visualSignature) float64 {
	hist := histogramSimilarity(a.hist, b.hist)
	grid := colorGridSimilarity(a.grid, b.grid)
	return clampFloat(hist*0.35+grid*0.65, 0, 1)
}

func colorGridSimilarity(a, b [192]uint16) float64 {
	diff := 0.0
	for i := range a {
		d := int(a[i]) - int(b[i])
		if d < 0 {
			d = -d
		}
		diff += float64(d) / visualQuantMax
	}
	return clampFloat(1.0-diff/float64(len(a)), 0, 1)
}

func quantizeUnit(v float64) uint16 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 65535
	}
	return uint16(math.Round(v * visualQuantMax))
}

func scaledCandidateCropBox(anchor *WindowsClickAnchorResult, recordedBox PixelBBox, candidate PixelBBox, width, height int) PixelBBox {
	if anchor == nil || anchor.CropBBox.W <= 0 || anchor.CropBBox.H <= 0 || recordedBox.W <= 0 || recordedBox.H <= 0 {
		return expandBBox(candidate, width, height, 12)
	}
	scaleX := float64(candidate.W) / float64(recordedBox.W)
	scaleY := float64(candidate.H) / float64(recordedBox.H)
	left := int(math.Round(float64(recordedBox.X-anchor.CropBBox.X) * scaleX))
	top := int(math.Round(float64(recordedBox.Y-anchor.CropBBox.Y) * scaleY))
	right := int(math.Round(float64(anchor.CropBBox.X+anchor.CropBBox.W-recordedBox.X-recordedBox.W) * scaleX))
	bottom := int(math.Round(float64(anchor.CropBBox.Y+anchor.CropBBox.H-recordedBox.Y-recordedBox.H) * scaleY))
	return clampBBox(PixelBBox{
		X: candidate.X - left,
		Y: candidate.Y - top,
		W: candidate.W + left + right,
		H: candidate.H + top + bottom,
	}, width, height)
}

func relocationPositionScore(recordedBox PixelBBox, recordedWidth, recordedHeight int, candidate PixelBBox, currentWidth, currentHeight int) float64 {
	recordedCenter := bboxCenter(recordedBox)
	candidateCenter := bboxCenter(candidate)
	rx := float64(recordedCenter.X) / float64(recordedWidth)
	ry := float64(recordedCenter.Y) / float64(recordedHeight)
	cx := float64(candidateCenter.X) / float64(currentWidth)
	cy := float64(candidateCenter.Y) / float64(currentHeight)
	dist := math.Hypot(rx-cx, ry-cy)
	return clampFloat(1.0-dist/0.35, 0, 1)
}

func relocationSizeScore(recordedBox PixelBBox, recordedWidth, recordedHeight int, candidate PixelBBox, currentWidth, currentHeight int) float64 {
	rw := float64(recordedBox.W) / float64(recordedWidth)
	rh := float64(recordedBox.H) / float64(recordedHeight)
	cw := float64(candidate.W) / float64(currentWidth)
	ch := float64(candidate.H) / float64(currentHeight)
	if rw <= 0 || rh <= 0 || cw <= 0 || ch <= 0 {
		return 0
	}
	delta := math.Abs(math.Log(cw/rw)) + math.Abs(math.Log(ch/rh))
	return clampFloat(1.0-delta/2.2, 0, 1)
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
