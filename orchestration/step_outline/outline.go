// Package step_outline manages step outlines used by execution_hook for prefetch
// and deviation analysis. Outlines are read-only prediction structures:
// - They are NOT executable.
// - They do NOT directly create DAG nodes.
// - A missing step does NOT block execution.
package step_outline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ui_console/domain/execution_hook"
)

// OutlineSource identifies where a step outline was generated from.
// Weight order (highest to lowest): main_agent > semantic_router > reused > cli_assist.
type OutlineSource string

const (
	SourceMainAgent      OutlineSource = "main_agent_outline"      // weight 4
	SourceSemanticRouter OutlineSource = "semantic_router_outline" // weight 3
	SourceReused         OutlineSource = "reused_outline"          // weight 2
	SourceCLIAssist      OutlineSource = "cli_assist_outline"      // weight 1
)

// sourceWeight maps each source to its priority weight.
var sourceWeight = map[OutlineSource]int{
	SourceMainAgent:      4,
	SourceSemanticRouter: 3,
	SourceReused:         2,
	SourceCLIAssist:      1,
}

// OutlineStep is one step in an outline. Each step has exactly one action + target
// (single_path rule). Multi-branch logic must call separate subagents.
// Corresponds to schema #52 in TASKS_1_2.md.
type OutlineStep = execution_hook.OutlineStep

// StepOutline is the full predicted execution plan for a DAG run.
// Corresponds to schema #52 in TASKS_1_2.md.
type StepOutline struct {
	ID           string        `json:"id"`
	Source       OutlineSource `json:"source"`
	SourceWeight int           `json:"source_weight"`
	CreatedAt    time.Time     `json:"created_at"`
	Steps        []OutlineStep `json:"steps"`
	OutlineHash  string        `json:"outline_hash"`
}

// Service manages step outline storage and retrieval for one project.
type Service struct {
	mu         sync.Mutex
	outlineDir string
}

// NewService creates an outline Service. Outlines are stored under:
//
//	<projectRoot>/data/step_outlines/
func NewService(projectRoot string) *Service {
	return &Service{
		outlineDir: filepath.Join(projectRoot, "data", "step_outlines"),
	}
}

// Create validates and stores a new StepOutline.
// Returns an error if the single_path rule is violated (any step with
// empty action+target, or multi-target syntax detected).
func (s *Service) Create(source OutlineSource, steps []OutlineStep) (*StepOutline, error) {
	if err := validateSinglePath(steps); err != nil {
		return nil, err
	}
	weight, ok := sourceWeight[source]
	if !ok {
		return nil, fmt.Errorf("step_outline: unknown source %q", source)
	}

	id := fmt.Sprintf("outline-%d", time.Now().UnixNano())
	outline := &StepOutline{
		ID:           id,
		Source:       source,
		SourceWeight: weight,
		CreatedAt:    time.Now(),
		Steps:        steps,
	}
	outline.OutlineHash = computeOutlineHash(outline)

	s.mu.Lock()
	defer s.mu.Unlock()
	return outline, s.saveLocked(outline)
}

// Get loads a StepOutline by ID.
func (s *Service) Get(id string) (*StepOutline, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(id)
}

// BestFor returns the highest-weight outline stored for a given DAGRunID.
// If none exists, it returns nil (missing outline does NOT block execution).
func (s *Service) BestFor(dagRunID string) *StepOutline {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.outlineDir)
	if err != nil {
		return nil
	}
	var best *StepOutline
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		o, err := s.loadLocked(e.Name()[:len(e.Name())-5])
		if err != nil {
			continue
		}
		if best == nil || o.SourceWeight > best.SourceWeight {
			best = o
		}
	}
	return best
}

// --- validation ---

// validateSinglePath enforces the single_path rule:
// each step must have exactly one action + one target, no multi-target syntax.
func validateSinglePath(steps []OutlineStep) error {
	for i, step := range steps {
		if step.Action == "" {
			return fmt.Errorf("step_outline: step %d has empty action (single_path rule)", i)
		}
		if step.Target == "" {
			return fmt.Errorf("step_outline: step %d has empty target (single_path rule)", i)
		}
		// Detect common multi-target syntax (commas, slashes, "and").
		if containsMultiTarget(step.Target) {
			return fmt.Errorf("step_outline: step %d target %q looks like multi-target; use separate subagents", i, step.Target)
		}
	}
	return nil
}

func containsMultiTarget(target string) bool {
	for _, ch := range []byte{',', '|'} {
		for _, b := range []byte(target) {
			if b == ch {
				return true
			}
		}
	}
	return false
}

// --- persistence ---

func (s *Service) saveLocked(o *StepOutline) error {
	if err := os.MkdirAll(s.outlineDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.outlineDir, o.ID+".json"), data, 0o600)
}

func (s *Service) loadLocked(id string) (*StepOutline, error) {
	data, err := os.ReadFile(filepath.Join(s.outlineDir, id+".json"))
	if err != nil {
		return nil, err
	}
	var o StepOutline
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

// computeOutlineHash produces a deterministic fingerprint for the outline.
func computeOutlineHash(o *StepOutline) string {
	raw := fmt.Sprintf("%s|%s|%d|%s", o.ID, o.Source, len(o.Steps), o.CreatedAt.UTC().Format(time.RFC3339Nano))
	// Simple XOR-based fingerprint (not cryptographic; use for change detection only).
	var sum [32]byte
	for i, b := range []byte(raw) {
		sum[i%32] ^= b
	}
	return fmt.Sprintf("%x", sum)
}
