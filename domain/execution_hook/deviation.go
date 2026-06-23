package execution_hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DeviationType enumerates the kinds of deviation that can occur between
// a step_outline prediction and an actual step trace.
type DeviationType string

const (
	DeviationMergedSteps     DeviationType = "merged_steps"
	DeviationSplitStep       DeviationType = "split_step"
	DeviationSkippedStep     DeviationType = "skipped_step"
	DeviationInsertedStep    DeviationType = "inserted_step"
	DeviationToolTypeChanged DeviationType = "tool_type_changed"
	DeviationSchemaImproved  DeviationType = "schema_improved"
	DeviationLatencyImproved DeviationType = "latency_improved"
	DeviationTokenImproved   DeviationType = "token_improved"
	DeviationRiskIncreased   DeviationType = "risk_increased"
	DeviationRiskDecreased   DeviationType = "risk_decreased"
	DeviationResultMismatch  DeviationType = "result_mismatch"
)

// PatchType classifies what kind of patch a deviation suggests.
type PatchType string

const (
	PatchTypeTagUpdate      PatchType = "tag_update"
	PatchTypeRegistryUpdate PatchType = "registry_update"
	PatchTypeNewCandidate   PatchType = "new_candidate"
	PatchTypeNone           PatchType = "none"
)

// Deviation records a single observed difference between outline and actual.
// Corresponds to schema #51 in TASKS_1_2.md.
type Deviation struct {
	Type               DeviationType `json:"type"`
	OutlineStepID      string        `json:"outline_step_id"`
	ActualStepID       string        `json:"actual_step_id"`
	Confidence         float64       `json:"confidence"`
	Reason             string        `json:"reason"`
	RiskDelta          float64       `json:"risk_delta"` // positive = risk increased
	SuggestedPatchType PatchType     `json:"suggested_patch_type"`
}

// OutlineStep is the shared type used by both execution_hook and step_outline.
// Defined here to avoid import cycles (step_outline imports execution_hook).
// Corresponds to schema #52 in TASKS_1_2.md.
type OutlineStep struct {
	ID                string    `json:"id"`
	Action            string    `json:"action"`
	Target            string    `json:"target"`
	ExpectedToolType  string    `json:"expected_tool_type"`
	ExpectedRiskLevel RiskLevel `json:"expected_risk_level"`
	Dependencies      []string  `json:"dependencies,omitempty"`
}

// DeviationStore persists deviation records for a hook run.
type DeviationStore struct {
	hookDir   string
	hookRunID string
}

func NewDeviationStore(hookDir, hookRunID string) *DeviationStore {
	return &DeviationStore{hookDir: hookDir, hookRunID: hookRunID}
}

// Analyze compares an outline step against the actual trace and produces a
// Deviation if a meaningful difference is found.
// This function must NOT call LLM, must NOT modify any registry or policy.
func Analyze(outlineStep OutlineStep, actual StepTrace) *Deviation {
	var devType DeviationType
	var reason string
	var riskDelta float64
	var patchType PatchType

	switch {
	case actual.ResultStatus == StepResultSkipped && outlineStep.Action != "":
		devType = DeviationSkippedStep
		reason = fmt.Sprintf("step %q was skipped; outline expected action %q", actual.StepID, outlineStep.Action)
		patchType = PatchTypeTagUpdate

	case actual.ToolUsed != outlineStep.ExpectedToolType && outlineStep.ExpectedToolType != "":
		devType = DeviationToolTypeChanged
		reason = fmt.Sprintf("tool changed from %q to %q", outlineStep.ExpectedToolType, actual.ToolUsed)
		patchType = PatchTypeRegistryUpdate

	case actual.RiskLevel == RiskCritical && outlineStep.ExpectedRiskLevel != RiskCritical:
		devType = DeviationRiskIncreased
		reason = "actual risk level elevated to critical"
		riskDelta = 1.0
		patchType = PatchTypeNone // critical — do not auto-promote

	case actual.RiskLevel == RiskLow && outlineStep.ExpectedRiskLevel == RiskHigh:
		devType = DeviationRiskDecreased
		reason = "actual risk level lower than outlined"
		riskDelta = -1.0
		patchType = PatchTypeTagUpdate

	default:
		return nil // no meaningful deviation detected
	}

	return &Deviation{
		Type:               devType,
		OutlineStepID:      outlineStep.ID,
		ActualStepID:       actual.StepID,
		Confidence:         0.75, // baseline; real scoring requires post-run LLM analysis
		Reason:             reason,
		RiskDelta:          riskDelta,
		SuggestedPatchType: patchType,
	}
}

// Save appends deviations to the hook run's pending_tag_patch.json.
// Only low-risk deviations may be auto-promoted (see reinforcement.go).
func (ds *DeviationStore) Save(deviations []Deviation) error {
	if len(deviations) == 0 {
		return nil
	}
	dir := filepath.Join(ds.hookDir, ds.hookRunID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(ds.hookDir, "pending_tag_patch.json")
	data, err := json.MarshalIndent(map[string]interface{}{
		"hook_run_id": ds.hookRunID,
		"deviations":  deviations,
		"recorded_at": time.Now(),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
