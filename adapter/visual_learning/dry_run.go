package visual_learning

import "time"

// DryRunDecision is what the system decides after a dry-run positioning pass.
type DryRunDecision string

const (
	DryRunConfirmed  DryRunDecision = "confirmed"   // user confirmed position
	DryRunRelocate   DryRunDecision = "relocate"    // user asked to re-position
	DryRunManual     DryRunDecision = "manual"      // user specifies manually
	DryRunStop       DryRunDecision = "stop"        // user aborted
	DryRunLocateFail DryRunDecision = "locate_fail" // system could not locate; halted
)

// DryRunResult records the outcome of a new-device first-replay positioning pass.
// Corresponds to schema #53 in TASKS_1_2.md.
//
// RULE: On a new device the first replay ONLY positions — it does NOT click.
// The user must confirm before any click is executed.
// Confidence is a four-stage calculation:
//
//	final_confidence = base_confidence - device_penalty + runtime_evidence
type DryRunResult struct {
	ActionID        string         `json:"action_id"`
	DeviceProfileID string         `json:"device_profile_id"`
	BaseConfidence  float64        `json:"base_confidence"`
	DevicePenalty   float64        `json:"device_penalty"`
	RuntimeEvidence float64        `json:"runtime_evidence"`
	FinalConfidence float64        `json:"final_confidence"`
	Decision        DryRunDecision `json:"decision"`
	OverlayShown    bool           `json:"overlay_shown"` // overlay must be shown before user input
	CreatedAt       time.Time      `json:"created_at"`
}

// DryRunConfig holds thresholds for dry-run behaviour.
type DryRunConfig struct {
	// MinConfidenceToExecute is the minimum final_confidence required
	// before an action may proceed to actual click execution.
	MinConfidenceToExecute float64
}

var DefaultDryRunConfig = DryRunConfig{
	MinConfidenceToExecute: 0.78,
}

// ComputeConfidence calculates the four-stage confidence score.
// Returns the final_confidence value.
func ComputeConfidence(base, devicePenalty, runtimeEvidence float64) float64 {
	result := base - devicePenalty + runtimeEvidence
	if result < 0 {
		return 0
	}
	if result > 1 {
		return 1
	}
	return result
}

// NewDryRunResult builds a result record for a positioning pass.
// Decision is set to DryRunLocateFail if final_confidence is below threshold.
// The overlay MUST be shown before calling this — overlay enforcement is
// the responsibility of the UI layer.
func NewDryRunResult(actionID, deviceProfileID string, base, penalty, runtimeEvidence float64, cfg DryRunConfig) DryRunResult {
	final := ComputeConfidence(base, penalty, runtimeEvidence)
	decision := DryRunLocateFail
	if final >= cfg.MinConfidenceToExecute {
		decision = DryRunConfirmed // still requires user confirmation — this just means positioning succeeded
	}
	return DryRunResult{
		ActionID:        actionID,
		DeviceProfileID: deviceProfileID,
		BaseConfidence:  base,
		DevicePenalty:   penalty,
		RuntimeEvidence: runtimeEvidence,
		FinalConfidence: final,
		Decision:        decision,
		OverlayShown:    true, // caller must set this to true only after UI shows overlay
		CreatedAt:       time.Now(),
	}
}

// TargetRemap is a pending request to update a region mapping after locate failure.
// It does NOT overwrite the existing dictionary entry — it enters pending review.
type TargetRemap struct {
	ID              string    `json:"id"`
	ActionID        string    `json:"action_id"`
	DeviceProfileID string    `json:"device_profile_id"`
	NewBBox         BBox      `json:"new_bbox"`
	Reason          string    `json:"reason"`
	Status          string    `json:"status"` // "pending_target_remap"
	CreatedAt       time.Time `json:"created_at"`
}
