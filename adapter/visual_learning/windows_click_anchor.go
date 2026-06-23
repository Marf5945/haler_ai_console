package visual_learning

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
	"strings"
)

// WindowsClickAnchorOptions controls the Windows-first click-to-anchor resolver.
//
// This interface is intentionally Windows-scoped for the first implementation:
// native replay currently uses Windows screen coordinates, while macOS still
// has a native executor stub. The code below does not call Win32 APIs; it
// defines the data contract and selection rules used by the Windows recorder
// and replay planner.
type WindowsClickAnchorOptions struct {
	NearMissPx          int `json:"near_miss_px,omitempty"`
	ShapeFallbackRadius int `json:"shape_fallback_radius,omitempty"`
	ManualBoxSize       int `json:"manual_box_size,omitempty"`
	CropPadding         int `json:"crop_padding,omitempty"`
}

// PixelPoint is an absolute screen/image coordinate.
type PixelPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PixelBBox is an absolute pixel bounding box in the current screenshot.
type PixelBBox struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// WindowsButtonCandidate is a YOLOX/OpenCV candidate considered for one click.
type WindowsButtonCandidate struct {
	ID             string    `json:"id"`
	Source         string    `json:"source"`
	BBox           PixelBBox `json:"bbox"`
	Confidence     float64   `json:"confidence"`
	ContainsClick  bool      `json:"contains_click"`
	ClickDistance  float64   `json:"click_distance"`
	CenterDistance float64   `json:"center_distance"`
	AreaReasonable bool      `json:"area_reasonable"`
	SelectionScore float64   `json:"selection_score"`
}

// WindowsClickAnchorResult is the small, token-efficient visual anchor package
// produced after a user click during Windows learning mode.
type WindowsClickAnchorResult struct {
	Platform         string                   `json:"platform"`
	OK               bool                     `json:"ok"`
	Mode             string                   `json:"mode"`
	Reason           string                   `json:"reason,omitempty"`
	Click            PixelPoint               `json:"click"`
	ExecutionPoint   PixelPoint               `json:"execution_point"`
	ExecutionHint    string                   `json:"execution_hint"`
	AnchorBBox       PixelBBox                `json:"anchor_bbox"`
	CropBBox         PixelBBox                `json:"crop_bbox"`
	CropPNGBase64    string                   `json:"crop_png_base64,omitempty"`
	Candidates       []WindowsButtonCandidate `json:"candidates,omitempty"`
	OCRStatus        string                   `json:"ocr_status"`
	OCRNote          string                   `json:"ocr_note"`
	DetectorBackend  string                   `json:"detector_backend,omitempty"`
	DetectorDegraded bool                     `json:"detector_degraded"`
	NeedsReview      bool                     `json:"needs_review"`
	// ImageWidth/ImageHeight record the pixel size of the screenshot the anchor
	// coordinates live in. Replay must scale from THIS space (not the window's
	// point-space rect) so point/pixel mismatches (e.g. Retina 2x) cannot skew
	// relocation.
	ImageWidth  int `json:"image_width,omitempty"`
	ImageHeight int `json:"image_height,omitempty"`
}

// ResolveWindowsClickAnchor applies the Windows learning-mode selection rules:
//  1. prefer YOLOX boxes that contain the click point;
//  2. if no YOLOX box contains the click, use a very close YOLOX near-miss;
//  3. if the model missed a visible object/text-like shape, use OpenCV shape
//     fallback and crop that region in color;
//  4. if nothing object-like is found, mark a small manual click box and replay
//     at the original click point.
//
// OCR is deliberately not used here. The result carries OCRStatus/OCRNote so a
// later optional OCR enhancer can add text without making this interface depend
// on Tesseract, cloud OCR, or another heavyweight runtime.
func ResolveWindowsClickAnchor(
	imageData []byte,
	width int,
	height int,
	clickX int,
	clickY int,
	detection DetectorResult,
	shape PipelineResult,
	options WindowsClickAnchorOptions,
) (WindowsClickAnchorResult, error) {
	opts := normalizeWindowsClickAnchorOptions(options)
	result := WindowsClickAnchorResult{
		Platform:         "windows",
		Click:            PixelPoint{X: clickX, Y: clickY},
		ImageWidth:       width,
		ImageHeight:      height,
		OCRStatus:        "not_used",
		OCRNote:          "OCR is optional and not used by this Windows-first anchor interface; shape/OCR text can be added later as an enhancer.",
		DetectorBackend:  detection.Backend,
		DetectorDegraded: detection.Degraded,
	}
	if width <= 0 || height <= 0 {
		return result, fmt.Errorf("windows click anchor: invalid screenshot size %dx%d", width, height)
	}
	click := clampPoint(PixelPoint{X: clickX, Y: clickY}, width, height)
	result.Click = click

	yoloCandidates := make([]WindowsButtonCandidate, 0, len(detection.Proposals))
	shapeCandidates := make([]WindowsButtonCandidate, 0, len(shape.Candidates)+len(detection.Proposals))
	if !detection.Degraded {
		yoloCandidates = proposalCandidates(detection.Proposals, "yolox", width, height, click)
	} else {
		shapeCandidates = append(shapeCandidates, proposalCandidates(detection.Proposals, "detector_degraded_shape", width, height, click)...)
	}
	shapeCandidates = append(shapeCandidates, fingerprintCandidates(shape.Candidates, "shape_fallback", width, height, click)...)

	if selected, ok := bestContainingCandidate(yoloCandidates); ok {
		if refined, refinedOK := bestTightShapeOverride(selected, shapeCandidates, width, height); refinedOK {
			result = fillWindowsAnchorSelection(result, refined, "shape_fallback", "click_bbox_center", true)
			result.Reason = "YOLOX selected an oversized group; using the tighter OpenCV/icon box around the click"
		} else {
			result = fillWindowsAnchorSelection(result, selected, "yolox_contains", "click_bbox_center", false)
		}
	} else if selected, ok := bestNearCandidate(yoloCandidates, float64(opts.NearMissPx)); ok {
		result = fillWindowsAnchorSelection(result, selected, "yolox_near_miss", "click_bbox_center", true)
		result.Reason = "click was just outside the YOLOX box; using the nearest high-confidence box"
	} else if selected, ok := bestShapeFallback(shapeCandidates, float64(opts.ShapeFallbackRadius)); ok {
		result = fillWindowsAnchorSelection(result, selected, "shape_fallback", "click_bbox_center", true)
		result.Reason = "YOLOX did not contain the click; an object/text-like shape near the click was framed instead"
	} else {
		box := manualClickBox(click, width, height, opts.ManualBoxSize)
		result.OK = true
		result.Mode = "manual_click_box"
		result.Reason = "no button/object candidate was found near the click; preserving the user click as a small manual anchor"
		result.ExecutionPoint = click
		result.ExecutionHint = "fast_click_original_point"
		result.AnchorBBox = box
		result.NeedsReview = true
	}

	result.Candidates = mergeCandidateLists(yoloCandidates, shapeCandidates)
	result.CropBBox = expandBBox(result.AnchorBBox, width, height, opts.CropPadding)
	result.CropPNGBase64 = cropPNGBase64(imageData, width, height, result.CropBBox)
	return result, nil
}

func normalizeWindowsClickAnchorOptions(opts WindowsClickAnchorOptions) WindowsClickAnchorOptions {
	if opts.NearMissPx <= 0 {
		opts.NearMissPx = 12
	}
	if opts.ShapeFallbackRadius <= 0 {
		opts.ShapeFallbackRadius = 48
	}
	if opts.ManualBoxSize <= 0 {
		opts.ManualBoxSize = 28
	}
	if opts.CropPadding <= 0 {
		opts.CropPadding = 12
	}
	return opts
}

func proposalCandidates(proposals []RegionProposal, source string, width, height int, click PixelPoint) []WindowsButtonCandidate {
	candidates := make([]WindowsButtonCandidate, 0, len(proposals))
	for i, proposal := range proposals {
		id := strings.TrimSpace(proposal.ProposalID)
		if id == "" {
			id = fmt.Sprintf("%s-%d", source, i)
		}
		candidates = append(candidates, makeWindowsCandidate(id, source, proposal.BBox, proposal.RawScore, width, height, click))
	}
	return candidates
}

func fingerprintCandidates(fingerprints []UIFingerprint, source string, width, height int, click PixelPoint) []WindowsButtonCandidate {
	candidates := make([]WindowsButtonCandidate, 0, len(fingerprints))
	for i, fp := range fingerprints {
		id := strings.TrimSpace(fp.RegionID)
		if id == "" {
			id = fmt.Sprintf("%s-%d", source, i)
		}
		candidateSource := strings.TrimSpace(fp.Source)
		if candidateSource == "" {
			candidateSource = source
		}
		candidates = append(candidates, makeWindowsCandidate(id, candidateSource, fp.BBoxRelative, fp.Confidence, width, height, click))
	}
	return candidates
}

func makeWindowsCandidate(id, source string, bbox BBox, confidence float64, width, height int, click PixelPoint) WindowsButtonCandidate {
	pixelBox := bboxToPixelBBox(bbox, width, height)
	centerDistance := distanceToBBoxCenter(click, pixelBox)
	clickDistance := distanceToBBox(click, pixelBox)
	contains := bboxContains(pixelBox, click)
	reasonable := areaReasonable(pixelBox, width, height)
	score := candidateSelectionScore(confidence, centerDistance, reasonable, contains)
	return WindowsButtonCandidate{
		ID:             id,
		Source:         source,
		BBox:           pixelBox,
		Confidence:     confidence,
		ContainsClick:  contains,
		ClickDistance:  roundFloat(clickDistance, 2),
		CenterDistance: roundFloat(centerDistance, 2),
		AreaReasonable: reasonable,
		SelectionScore: roundFloat(score, 4),
	}
}

func bestContainingCandidate(candidates []WindowsButtonCandidate) (WindowsButtonCandidate, bool) {
	return bestCandidate(candidates, func(c WindowsButtonCandidate) bool {
		return c.ContainsClick
	})
}

func bestNearCandidate(candidates []WindowsButtonCandidate, maxDistance float64) (WindowsButtonCandidate, bool) {
	return bestCandidate(candidates, func(c WindowsButtonCandidate) bool {
		return c.ClickDistance > 0 && c.ClickDistance <= maxDistance
	})
}

func bestShapeFallback(candidates []WindowsButtonCandidate, maxDistance float64) (WindowsButtonCandidate, bool) {
	return bestCandidate(candidates, func(c WindowsButtonCandidate) bool {
		return c.ContainsClick || c.ClickDistance <= maxDistance
	})
}

func bestTightShapeOverride(yolo WindowsButtonCandidate, shapes []WindowsButtonCandidate, width, height int) (WindowsButtonCandidate, bool) {
	if !looksLikeOversizedGroup(yolo, width, height) {
		return WindowsButtonCandidate{}, false
	}
	return bestCandidate(shapes, func(c WindowsButtonCandidate) bool {
		if !c.ContainsClick || !c.AreaReasonable {
			return false
		}
		if !looksLikeTightIconBox(c.BBox) {
			return false
		}
		if area(c.BBox)*4 > area(yolo.BBox) {
			return false
		}
		return c.CenterDistance+8 < yolo.CenterDistance
	})
}

func looksLikeOversizedGroup(candidate WindowsButtonCandidate, width, height int) bool {
	box := candidate.BBox
	if box.W <= 0 || box.H <= 0 || width <= 0 || height <= 0 {
		return false
	}
	aspect := float64(box.W) / float64(box.H)
	areaFraction := float64(area(box)) / float64(width*height)
	if candidate.CenterDistance < 32 {
		return false
	}
	if box.W >= 140 && box.H <= 80 && aspect >= 3.0 {
		return true
	}
	if areaFraction >= 0.025 && (box.W >= 120 || box.H >= 120) {
		return true
	}
	return false
}

func looksLikeTightIconBox(box PixelBBox) bool {
	if box.W < 12 || box.H < 12 || box.W > 96 || box.H > 96 {
		return false
	}
	aspect := float64(box.W) / float64(box.H)
	return aspect >= 0.55 && aspect <= 1.85
}

func bestCandidate(candidates []WindowsButtonCandidate, keep func(WindowsButtonCandidate) bool) (WindowsButtonCandidate, bool) {
	filtered := make([]WindowsButtonCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if keep(candidate) && candidate.BBox.W > 0 && candidate.BBox.H > 0 {
			filtered = append(filtered, candidate)
		}
	}
	if len(filtered) == 0 {
		return WindowsButtonCandidate{}, false
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		a := filtered[i]
		b := filtered[j]
		if a.AreaReasonable != b.AreaReasonable {
			return a.AreaReasonable
		}
		if math.Abs(a.SelectionScore-b.SelectionScore) > 0.0001 {
			return a.SelectionScore > b.SelectionScore
		}
		if math.Abs(a.Confidence-b.Confidence) > 0.0001 {
			return a.Confidence > b.Confidence
		}
		return area(a.BBox) < area(b.BBox)
	})
	return filtered[0], true
}

func fillWindowsAnchorSelection(result WindowsClickAnchorResult, selected WindowsButtonCandidate, mode, hint string, degraded bool) WindowsClickAnchorResult {
	result.OK = true
	result.Mode = mode
	result.ExecutionHint = hint
	result.AnchorBBox = selected.BBox
	result.ExecutionPoint = bboxCenter(selected.BBox)
	result.NeedsReview = degraded
	return result
}

func mergeCandidateLists(groups ...[]WindowsButtonCandidate) []WindowsButtonCandidate {
	var merged []WindowsButtonCandidate
	seen := map[string]bool{}
	for _, group := range groups {
		for _, candidate := range group {
			key := fmt.Sprintf("%s:%d:%d:%d:%d", candidate.Source, candidate.BBox.X, candidate.BBox.Y, candidate.BBox.W, candidate.BBox.H)
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, candidate)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].SelectionScore > merged[j].SelectionScore
	})
	if len(merged) > 12 {
		return merged[:12]
	}
	return merged
}

func bboxToPixelBBox(b BBox, width, height int) PixelBBox {
	x := int(math.Round(b.X * float64(width)))
	y := int(math.Round(b.Y * float64(height)))
	w := int(math.Round(b.W * float64(width)))
	h := int(math.Round(b.H * float64(height)))
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	if x > width {
		x = width
	}
	if y > height {
		y = height
	}
	if x+w > width {
		w = width - x
	}
	if y+h > height {
		h = height - y
	}
	return PixelBBox{X: x, Y: y, W: w, H: h}
}

func candidateSelectionScore(confidence, distance float64, areaOK bool, contains bool) float64 {
	conf := clampFloat(confidence, 0, 1)
	distanceScore := 1.0 / (1.0 + distance/24.0)
	areaScore := 0.35
	if areaOK {
		areaScore = 1.0
	}
	containsScore := 0.0
	if contains {
		containsScore = 1.0
	}
	return conf*0.55 + areaScore*0.25 + distanceScore*0.12 + containsScore*0.08
}

func areaReasonable(box PixelBBox, width, height int) bool {
	if box.W < 8 || box.H < 8 {
		return false
	}
	if width <= 0 || height <= 0 {
		return false
	}
	areaFraction := float64(area(box)) / float64(width*height)
	if areaFraction < 0.00004 || areaFraction > 0.12 {
		return false
	}
	aspect := float64(box.W) / float64(box.H)
	return aspect >= 0.35 && aspect <= 18
}

func bboxContains(box PixelBBox, point PixelPoint) bool {
	return point.X >= box.X && point.X <= box.X+box.W && point.Y >= box.Y && point.Y <= box.Y+box.H
}

func bboxCenter(box PixelBBox) PixelPoint {
	return PixelPoint{X: box.X + box.W/2, Y: box.Y + box.H/2}
}

func distanceToBBoxCenter(point PixelPoint, box PixelBBox) float64 {
	center := bboxCenter(box)
	return math.Hypot(float64(point.X-center.X), float64(point.Y-center.Y))
}

func distanceToBBox(point PixelPoint, box PixelBBox) float64 {
	if bboxContains(box, point) {
		return 0
	}
	dx := 0
	if point.X < box.X {
		dx = box.X - point.X
	} else if point.X > box.X+box.W {
		dx = point.X - (box.X + box.W)
	}
	dy := 0
	if point.Y < box.Y {
		dy = box.Y - point.Y
	} else if point.Y > box.Y+box.H {
		dy = point.Y - (box.Y + box.H)
	}
	return math.Hypot(float64(dx), float64(dy))
}

func manualClickBox(point PixelPoint, width, height, size int) PixelBBox {
	half := size / 2
	box := PixelBBox{X: point.X - half, Y: point.Y - half, W: size, H: size}
	return clampBBox(box, width, height)
}

func expandBBox(box PixelBBox, width, height, padding int) PixelBBox {
	return clampBBox(PixelBBox{
		X: box.X - padding,
		Y: box.Y - padding,
		W: box.W + padding*2,
		H: box.H + padding*2,
	}, width, height)
}

func clampBBox(box PixelBBox, width, height int) PixelBBox {
	if box.X < 0 {
		box.W += box.X
		box.X = 0
	}
	if box.Y < 0 {
		box.H += box.Y
		box.Y = 0
	}
	if box.X > width {
		box.X = width
		box.W = 0
	}
	if box.Y > height {
		box.Y = height
		box.H = 0
	}
	if box.X+box.W > width {
		box.W = width - box.X
	}
	if box.Y+box.H > height {
		box.H = height - box.Y
	}
	if box.W < 0 {
		box.W = 0
	}
	if box.H < 0 {
		box.H = 0
	}
	return box
}

func clampPoint(point PixelPoint, width, height int) PixelPoint {
	if point.X < 0 {
		point.X = 0
	}
	if point.Y < 0 {
		point.Y = 0
	}
	if point.X >= width {
		point.X = width - 1
	}
	if point.Y >= height {
		point.Y = height - 1
	}
	return point
}

func area(box PixelBBox) int {
	return box.W * box.H
}

func cropPNGBase64(imageData []byte, width, height int, box PixelBBox) string {
	if width <= 0 || height <= 0 || box.W <= 0 || box.H <= 0 {
		return ""
	}
	if len(imageData) < width*height*4 {
		return ""
	}
	img := image.NewRGBA(image.Rect(0, 0, box.W, box.H))
	for y := 0; y < box.H; y++ {
		for x := 0; x < box.W; x++ {
			src := ((box.Y+y)*width + (box.X + x)) * 4
			img.SetRGBA(x, y, color.RGBA{
				R: imageData[src],
				G: imageData[src+1],
				B: imageData[src+2],
				A: imageData[src+3],
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func roundFloat(value float64, places int) float64 {
	if places <= 0 {
		return math.Round(value)
	}
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}
