// Package visual_learning implements Visual Capture Replay Learning Mode (v3.1.0).
// Users must explicitly activate Learning Mode — it must NEVER record in the background.
// LLM / OCR processing is deferred (not required during demonstration).
package visual_learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LearningRunStatus tracks the lifecycle of a recording session.
type LearningRunStatus string

const (
	LearningRunActive    LearningRunStatus = "active"
	LearningRunStopped   LearningRunStatus = "stopped"
	LearningRunProcessed LearningRunStatus = "processed"
)

// MouseEventType classifies pointer interactions.
type MouseEventType string

const (
	MouseEventClick      MouseEventType = "click"
	MouseEventRightClick MouseEventType = "right_click"
	MouseEventDoubleClick MouseEventType = "double_click"
	MouseEventDrag       MouseEventType = "drag"
	MouseEventScroll     MouseEventType = "scroll"
)

// LearningRun is the top-level record for one user-initiated demonstration session.
// Corresponds to schema #53 in TASKS_1_2.md.
type LearningRun struct {
	ID               string            `json:"id"`
	ActiveWindowHash string            `json:"active_window_hash"`
	StartedAt        time.Time         `json:"started_at"`
	StoppedAt        *time.Time        `json:"stopped_at,omitempty"`
	Status           LearningRunStatus `json:"status"`
	TracePath        string            `json:"trace_path"`
}

// MouseEventTrace records a single pointer event during a demonstration.
// Corresponds to schema #53 in TASKS_1_2.md.
// LLM / OCR processing must happen AFTER demonstration, not here.
type MouseEventTrace struct {
	Timestamp      time.Time      `json:"timestamp"`
	EventType      MouseEventType `json:"event_type"`
	X              int            `json:"x"`
	Y              int            `json:"y"`
	Button         string         `json:"button"`
	TargetRegionID string         `json:"target_region_id"`
	BeforeHash     string         `json:"before_hash"`
	AfterHash      string         `json:"after_hash"`
}

// LearningService manages Learning Mode activation and trace recording.
type LearningService struct {
	mu          sync.Mutex
	learnDir    string
	activeRun   *LearningRun
	isRecording bool
}

// NewLearningService creates a service whose data is stored under:
//
//	<projectRoot>/data/projects/<project>/visual_learning/learning_runs/
func NewLearningService(projectRoot string) *LearningService {
	return &LearningService{
		learnDir: filepath.Join(projectRoot, "data", "visual_learning", "learning_runs"),
	}
}

// IsRecording returns whether Learning Mode is currently active.
func (s *LearningService) IsRecording() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRecording
}

// StartDemonstration activates Learning Mode.
// Only one run may be active at a time; user must call StopDemonstration before starting again.
func (s *LearningService) StartDemonstration(activeWindowHash string) (*LearningRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRecording {
		return nil, fmt.Errorf("learning mode: already recording run %q; stop it first", s.activeRun.ID)
	}

	id := fmt.Sprintf("learn-%d", time.Now().UnixNano())
	tracePath := filepath.Join(s.learnDir, id, "encrypted_learning_trace.jsonl")
	run := &LearningRun{
		ID:               id,
		ActiveWindowHash: activeWindowHash,
		StartedAt:        time.Now(),
		Status:           LearningRunActive,
		TracePath:        tracePath,
	}
	if err := s.saveRunLocked(run); err != nil {
		return nil, err
	}
	s.activeRun = run
	s.isRecording = true
	return run, nil
}

// StopDemonstration deactivates Learning Mode and finalises the run.
func (s *LearningService) StopDemonstration() (*LearningRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRecording {
		return nil, fmt.Errorf("learning mode: no active recording")
	}
	now := time.Now()
	s.activeRun.StoppedAt = &now
	s.activeRun.Status = LearningRunStopped
	if err := s.saveRunLocked(s.activeRun); err != nil {
		return nil, err
	}
	run := s.activeRun
	s.activeRun = nil
	s.isRecording = false
	return run, nil
}

// RecordEvent appends a mouse event trace to the active run's trace file.
// Returns an error if Learning Mode is not active — background recording is FORBIDDEN.
func (s *LearningService) RecordEvent(event MouseEventTrace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRecording {
		return fmt.Errorf("learning mode: recording not active; cannot record event")
	}
	if err := os.MkdirAll(filepath.Dir(s.activeRun.TracePath), 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.activeRun.TracePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// ActiveRun returns a copy of the currently active run, or nil.
func (s *LearningService) ActiveRun() *LearningRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeRun == nil {
		return nil
	}
	copy := *s.activeRun
	return &copy
}

// --- persistence ---

func (s *LearningService) saveRunLocked(run *LearningRun) error {
	dir := filepath.Join(s.learnDir, run.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "run.json"), data, 0o600)
}
