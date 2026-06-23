// Package execution_hook provides step-level observation for AI Console.
// Hook records actual step traces and compares against step_outline predictions.
// It must NOT execute steps, NOT make final decisions, and must NOT modify
// tool_registry, risk_policy, DAG nodes, memory, or existing subagents during a run.
package execution_hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HookStatus represents the lifecycle state of a hook run.
type HookStatus string

const (
	HookStatusRunning   HookStatus = "running"
	HookStatusCompleted HookStatus = "completed"
	HookStatusAborted   HookStatus = "aborted"
)

// RiskLevel mirrors the global risk classification.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// StepResultStatus represents the outcome of a step execution.
type StepResultStatus string

const (
	StepResultOK      StepResultStatus = "ok"
	StepResultFailed  StepResultStatus = "failed"
	StepResultSkipped StepResultStatus = "skipped"
)

// HookRun is the top-level record for one hook observation session.
// Corresponds to schema #51 in TASKS_1_2.md.
type HookRun struct {
	ID           string     `json:"id"`
	DAGRunID     string     `json:"dag_run_id"`
	OutlineID    string     `json:"outline_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       HookStatus `json:"status"`
	EvidencePath string     `json:"evidence_path"`
	SummaryHash  string     `json:"summary_hash"`
}

// StepTrace records what actually happened during a single step execution.
// Corresponds to schema #51 in TASKS_1_2.md.
type StepTrace struct {
	StepID        string           `json:"step_id"`
	OutlineStepID string           `json:"outline_step_id"`
	Action        string           `json:"action"`
	Target        string           `json:"target"`
	ToolUsed      string           `json:"tool_used"`
	StartedAt     time.Time        `json:"started_at"`
	EndedAt       time.Time        `json:"ended_at"`
	ResultStatus  StepResultStatus `json:"result_status"`
	RiskLevel     RiskLevel        `json:"risk_level"`
	TraceHash     string           `json:"trace_hash"`
}

// HookSummary is the post-run summary produced after a subagent completes.
type HookSummary struct {
	HookRunID   string      `json:"hook_run_id"`
	Deviations  []Deviation `json:"deviations"`
	TagPatches  []TagPatch  `json:"tag_patches"`
	GeneratedAt time.Time   `json:"generated_at"`
	SummaryHash string      `json:"summary_hash"`
}

// SummaryIndex is stored alongside the evidence to allow fast lookup.
type SummaryIndex struct {
	Entries []SummaryIndexEntry `json:"entries"`
}

type SummaryIndexEntry struct {
	HookRunID   string    `json:"hook_run_id"`
	DAGRunID    string    `json:"dag_run_id"`
	CompletedAt time.Time `json:"completed_at"`
	SummaryHash string    `json:"summary_hash"`
}

// Service manages hook run lifecycle and storage for one project.
type Service struct {
	mu          sync.Mutex
	projectRoot string
	hookDir     string
	runs        map[string]*HookRun
	summaryIdx  SummaryIndex
}

// NewService creates a hook Service rooted at projectRoot.
// Storage will be initialised under:
//
//	<projectRoot>/data/projects/<project>/execution_hook_runs/
func NewService(projectRoot string) *Service {
	if projectRoot == "" {
		projectRoot = "."
	}
	hookDir := filepath.Join(projectRoot, "data", "execution_hook_runs")
	svc := &Service{
		projectRoot: projectRoot,
		hookDir:     hookDir,
		runs:        make(map[string]*HookRun),
	}
	_ = svc.loadIndex()
	return svc
}

// StartRun initialises a new HookRun and persists it.
func (s *Service) StartRun(dagRunID, outlineID string) (*HookRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("hook-%d", time.Now().UnixNano())
	evidencePath := filepath.Join(s.hookDir, id, "encrypted_hook_evidence.jsonl")
	run := &HookRun{
		ID:           id,
		DAGRunID:     dagRunID,
		OutlineID:    outlineID,
		StartedAt:    time.Now(),
		Status:       HookStatusRunning,
		EvidencePath: evidencePath,
	}
	s.runs[id] = run
	if err := s.saveRunLocked(run); err != nil {
		return nil, err
	}
	return run, nil
}

// RecordTrace appends a StepTrace to the hook run's evidence file.
// This does NOT modify tool_registry, risk_policy, DAG, memory, or subagents.
func (s *Service) RecordTrace(hookRunID string, trace StepTrace) error {
	s.mu.Lock()
	run, ok := s.runs[hookRunID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("hook run %q not found", hookRunID)
	}

	// Compute trace hash for integrity.
	trace.TraceHash = computeHash(trace.StepID + trace.Action + trace.Target + trace.StartedAt.String())

	// Append to evidence JSONL (plain — encryption layer in evidence_store.go).
	if err := os.MkdirAll(filepath.Dir(run.EvidencePath), 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(run.EvidencePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// CompleteRun marks the run as completed and writes a summary.
// Patch promotion (if any) happens HERE — after the run, never during.
func (s *Service) CompleteRun(hookRunID string) (*HookSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[hookRunID]
	if !ok {
		return nil, fmt.Errorf("hook run %q not found", hookRunID)
	}
	now := time.Now()
	run.CompletedAt = &now
	run.Status = HookStatusCompleted

	summary := &HookSummary{
		HookRunID:   hookRunID,
		GeneratedAt: now,
	}
	summary.SummaryHash = computeHash(hookRunID + now.String())
	run.SummaryHash = summary.SummaryHash

	if err := s.saveRunLocked(run); err != nil {
		return nil, err
	}
	s.summaryIdx.Entries = append(s.summaryIdx.Entries, SummaryIndexEntry{
		HookRunID:   hookRunID,
		DAGRunID:    run.DAGRunID,
		CompletedAt: now,
		SummaryHash: summary.SummaryHash,
	})
	_ = s.saveIndexLocked()
	return summary, nil
}

// GetRun returns a copy of a HookRun by ID.
func (s *Service) GetRun(hookRunID string) (*HookRun, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[hookRunID]
	if !ok {
		return nil, false
	}
	copy := *r
	return &copy, true
}

// --- persistence helpers ---

func (s *Service) saveRunLocked(run *HookRun) error {
	dir := filepath.Join(s.hookDir, run.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "hook_run.json")
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *Service) saveIndexLocked() error {
	if err := os.MkdirAll(s.hookDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.summaryIdx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.hookDir, "hook_summary_index.json"), data, 0o600)
}

func (s *Service) loadIndex() error {
	path := filepath.Join(s.hookDir, "hook_summary_index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.summaryIdx)
}
