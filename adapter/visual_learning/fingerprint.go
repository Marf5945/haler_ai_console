package visual_learning

import "time"

// UIFingerprint is the visual identity record for a UI region.
// Corresponds to schema #20 / #53 in TASKS_1_2.md.
// readable_patch_exported is ALWAYS false by default and must never be set
// without an explicit safe_export pass.
type UIFingerprint struct {
	RegionID              string    `json:"region_id"`
	ElementTypeGuess      string    `json:"element_type_guess"`
	BBoxRelative          BBox      `json:"bbox_relative"`
	Anchor                string    `json:"anchor"`
	ShapeHash             string    `json:"shape_hash"`
	EdgeHash64            string    `json:"edge_hash_64"`
	EdgeHash128           string    `json:"edge_hash_128"`
	IconHash              string    `json:"icon_hash"`
	ColorCode             string    `json:"color_code"`
	TextHint              string    `json:"text_hint"`
	MotionDelta           float64   `json:"motion_delta"`
	Source                string    `json:"source"` // "opencv" | "yolo" | "manual"
	Confidence            float64   `json:"confidence"`
	ReadablePatchExported bool      `json:"readable_patch_exported"` // always false by default
	CreatedAt             time.Time `json:"created_at"`
}

// BBox is a relative bounding box (0.0–1.0 normalised coordinates).
type BBox struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// NewUIFingerprint creates a fingerprint with safe defaults.
// ReadablePatchExported is always initialised to false.
func NewUIFingerprint(regionID, source string, bbox BBox, confidence float64) UIFingerprint {
	return UIFingerprint{
		RegionID:              regionID,
		BBoxRelative:          bbox,
		Source:                source,
		Confidence:            confidence,
		ReadablePatchExported: false, // immutable default per spec
		CreatedAt:             time.Now(),
	}
}
