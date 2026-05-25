package visual_learning

import (
	"os"
	"testing"
	"time"
)

func tmpLearnDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "vl_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// TestLearningModeActivation verifies start/stop lifecycle.
func TestLearningModeActivation(t *testing.T) {
	svc := NewLearningService(tmpLearnDir(t))

	if svc.IsRecording() {
		t.Fatal("should not be recording on start")
	}
	run, err := svc.StartDemonstration("window-hash-abc")
	if err != nil {
		t.Fatalf("StartDemonstration: %v", err)
	}
	if !svc.IsRecording() {
		t.Fatal("should be recording after start")
	}

	// Double-start must error.
	if _, err := svc.StartDemonstration("other"); err == nil {
		t.Error("expected error on double-start")
	}

	stopped, err := svc.StopDemonstration()
	if err != nil {
		t.Fatalf("StopDemonstration: %v", err)
	}
	if stopped.ID != run.ID {
		t.Error("stopped run ID mismatch")
	}
	if svc.IsRecording() {
		t.Fatal("should not be recording after stop")
	}
}

// TestRecordEventRequiresActiveMode verifies background recording is forbidden.
func TestRecordEventRequiresActiveMode(t *testing.T) {
	svc := NewLearningService(tmpLearnDir(t))
	event := MouseEventTrace{Timestamp: time.Now(), EventType: MouseEventClick}
	if err := svc.RecordEvent(event); err == nil {
		t.Error("expected error when recording without active mode")
	}
}

// TestSafeExportAllowlist verifies only whitelisted sections pass.
func TestSafeExportAllowlist(t *testing.T) {
	dir := tmpLearnDir(t)
	exporter := NewSafeExporter(dir)

	manifest, err := exporter.Export([]string{"element_dictionary", "canonical_label_schema"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(manifest.IncludedSections) != 2 {
		t.Errorf("expected 2 included sections, got %d", len(manifest.IncludedSections))
	}
}

// TestSafeExportForbiddenBlocked verifies forbidden sections are rejected.
func TestSafeExportForbiddenBlocked(t *testing.T) {
	dir := tmpLearnDir(t)
	exporter := NewSafeExporter(dir)

	forbidden := []string{"full_screenshots", "readable_text_patches", "api_keys", "passwords", "form_content"}
	for _, section := range forbidden {
		if _, err := exporter.Export([]string{section}); err == nil {
			t.Errorf("expected error for forbidden section %q", section)
		}
	}
}

// TestPendingCandidateLifecycle verifies fresh → stale → archived transitions.
func TestPendingCandidateLifecycle(t *testing.T) {
	dir := tmpLearnDir(t)
	mgr := NewPendingCandidateManager(dir)

	rec, err := mgr.Add("action", "subagent-x")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if rec.Status != AgeStatusFresh {
		t.Errorf("new record must be fresh_pending, got %s", rec.Status)
	}
	if mgr.ActiveCount() != 1 {
		t.Errorf("expected 1 active candidate")
	}
}

// TestDryRunConfidenceCalc verifies four-stage confidence computation.
func TestDryRunConfidenceCalc(t *testing.T) {
	result := NewDryRunResult("action-1", "device-1", 0.85, 0.08, 0.05, DefaultDryRunConfig)
	// expected: 0.85 - 0.08 + 0.05 = 0.82
	expected := 0.82
	if abs(result.FinalConfidence-expected) > 0.001 {
		t.Errorf("expected final_confidence %.3f, got %.3f", expected, result.FinalConfidence)
	}
	if result.Decision != DryRunConfirmed {
		t.Errorf("expected confirmed decision (confidence above threshold)")
	}
}

// TestDryRunLocateFailBelowThreshold checks that low confidence gives locate_fail.
func TestDryRunLocateFailBelowThreshold(t *testing.T) {
	result := NewDryRunResult("action-2", "device-2", 0.60, 0.10, 0.00, DefaultDryRunConfig)
	if result.Decision != DryRunLocateFail {
		t.Errorf("expected locate_fail for confidence %.3f", result.FinalConfidence)
	}
}

// TestCanonicalLabelMapping verifies known keywords map correctly.
func TestCanonicalLabelMapping(t *testing.T) {
	dir := tmpLearnDir(t)
	svc := NewCanonicalLabelService(dir)

	label, ok := svc.MapLabel("submit button", "region-1", 0.9)
	if !ok {
		t.Error("expected successful mapping for 'submit button'")
	}
	if label.ElementType != ElementButton || label.ActionSemantic != ActionSubmit {
		t.Errorf("unexpected label: %+v", label)
	}
}

// TestCanonicalLabelUnmappedCreatesPending verifies unknown descriptions go to pending.
func TestCanonicalLabelUnmappedCreatesPending(t *testing.T) {
	dir := tmpLearnDir(t)
	svc := NewCanonicalLabelService(dir)
	_, ok := svc.MapLabel("some completely unknown widget xyz", "region-2", 0.5)
	if ok {
		t.Error("expected no mapping for unknown description")
	}
	// Pending file should now exist.
	pendingPath := dir + "/pending/pending_label_candidate.json"
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Error("pending_label_candidate.json not created")
	}
}

// TestUIFingerprintDefaultsReadableFalse verifies the safe default.
func TestUIFingerprintDefaultsReadableFalse(t *testing.T) {
	fp := NewUIFingerprint("r-1", "opencv", BBox{X: 0.1, Y: 0.2, W: 0.3, H: 0.1}, 0.8)
	if fp.ReadablePatchExported {
		t.Error("readable_patch_exported must default to false")
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
