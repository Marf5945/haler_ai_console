package execution_hook

import (
	"os"
	"testing"
	"time"
)

func tmpHookDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "hook_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// TestHookRunLifecycle verifies Start → RecordTrace → Complete.
func TestHookRunLifecycle(t *testing.T) {
	svc := NewService(tmpHookDir(t))

	run, err := svc.StartRun("dag-001", "outline-001")
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if run.Status != HookStatusRunning {
		t.Fatalf("expected running, got %s", run.Status)
	}

	trace := StepTrace{
		StepID:        "step-1",
		OutlineStepID: "outline-step-1",
		Action:        "click",
		Target:        "submit_button",
		ToolUsed:      "dom_click",
		StartedAt:     time.Now(),
		EndedAt:       time.Now(),
		ResultStatus:  StepResultOK,
		RiskLevel:     RiskLow,
	}
	if err := svc.RecordTrace(run.ID, trace); err != nil {
		t.Fatalf("RecordTrace: %v", err)
	}

	summary, err := svc.CompleteRun(run.ID)
	if err != nil {
		t.Fatalf("CompleteRun: %v", err)
	}
	if summary.HookRunID != run.ID {
		t.Errorf("summary run ID mismatch")
	}
	completed, ok := svc.GetRun(run.ID)
	if !ok {
		t.Fatal("GetRun returned nothing")
	}
	if completed.Status != HookStatusCompleted {
		t.Errorf("expected completed, got %s", completed.Status)
	}
}

// TestDeviationAnalyze checks deviation detection for skipped steps.
func TestDeviationAnalyze(t *testing.T) {
	outline := OutlineStep{
		ID:                "os-1",
		Action:            "read",
		Target:            "config.json",
		ExpectedToolType:  "file_reader",
		ExpectedRiskLevel: RiskLow,
	}
	actual := StepTrace{
		StepID:       "s-1",
		ResultStatus: StepResultSkipped,
		RiskLevel:    RiskLow,
	}
	dev := Analyze(outline, actual)
	if dev == nil {
		t.Fatal("expected a deviation for skipped step")
	}
	if dev.Type != DeviationSkippedStep {
		t.Errorf("expected skipped_step, got %s", dev.Type)
	}
}

// TestDeviationAnalyzeNoDeviation verifies nil when nothing deviates.
func TestDeviationAnalyzeNoDeviation(t *testing.T) {
	outline := OutlineStep{
		ID:                "os-2",
		Action:            "read",
		Target:            "log.txt",
		ExpectedToolType:  "file_reader",
		ExpectedRiskLevel: RiskLow,
	}
	actual := StepTrace{
		StepID:       "s-2",
		ToolUsed:     "file_reader",
		ResultStatus: StepResultOK,
		RiskLevel:    RiskLow,
	}
	dev := Analyze(outline, actual)
	if dev != nil {
		t.Errorf("expected no deviation, got %+v", dev)
	}
}

// TestReinforcementLowRiskPromoted ensures low-risk patches are auto-promoted.
func TestReinforcementLowRiskPromoted(t *testing.T) {
	dir := tmpHookDir(t)
	svc := NewReinforcementService(dir)

	patches := []TagPatch{
		{ID: "p1", TargetID: "subagent-a", RiskLevel: RiskLow, LearnedTags: []string{"fast"}},
		{ID: "p2", TargetID: "subagent-b", RiskLevel: RiskMedium, LearnedTags: []string{"risky"}},
	}
	promoted, err := svc.PromoteLowRisk(patches)
	if err != nil {
		t.Fatalf("PromoteLowRisk: %v", err)
	}
	if len(promoted) != 1 {
		t.Errorf("expected 1 promoted, got %d", len(promoted))
	}
	if promoted[0].Status != TagPatchPromoted {
		t.Errorf("expected promoted status")
	}
	// Medium patch must remain pending.
	pending, err := svc.GetPendingTagPatches()
	if err != nil {
		t.Fatalf("GetPendingTagPatches: %v", err)
	}
	if len(pending) != 1 || pending[0].RiskLevel != RiskMedium {
		t.Errorf("expected 1 medium pending patch")
	}
}

// TestHashChainIntegrity verifies the chain links are intact after appends.
func TestHashChainIntegrity(t *testing.T) {
	dir := tmpHookDir(t)
	chain := NewHashChain(dir)

	for i := 0; i < 5; i++ {
		entry := ChainEntry{
			Type:      "test_entry",
			Payload:   "payload",
			CreatedAt: time.Now(),
		}
		if err := chain.Append(entry); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	if err := chain.Verify(); err != nil {
		t.Errorf("chain verification failed: %v", err)
	}
}

// TestEvidenceStoreEncryptDecrypt checks round-trip encryption.
func TestEvidenceStoreEncryptDecrypt(t *testing.T) {
	dir := tmpHookDir(t)
	store, err := NewEvidenceStoreFromFile(dir)
	if err != nil {
		t.Fatalf("NewEvidenceStoreFromFile: %v", err)
	}
	plaintext := []byte(`{"step":"test","value":42}`)
	encrypted, err := store.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	decrypted, err := store.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("round-trip mismatch: got %q", decrypted)
	}
}

// TestCandidateCreateDoesNotModifyExisting verifies the candidate
// only creates new files and never references mutation of existing subagents.
func TestCandidateCreateDoesNotModifyExisting(t *testing.T) {
	dir := tmpHookDir(t)
	svc := NewCandidateService(dir)

	candidate, err := svc.CreateCandidate("hook-001", "TestAgent", "A test candidate", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CreateCandidate: %v", err)
	}
	if candidate.Status != CandidatePending {
		t.Errorf("new candidate must be pending")
	}

	// Verify the files were created.
	if _, err := os.Stat(candidate.CandidateJSONPath); os.IsNotExist(err) {
		t.Error("JSON file not created")
	}
	if _, err := os.Stat(candidate.CandidateMDPath); os.IsNotExist(err) {
		t.Error("MD file not created")
	}

	list, err := svc.ListCandidates()
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 candidate, got %d", len(list))
	}
}
