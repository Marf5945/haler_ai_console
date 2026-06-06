// Package onboarding manages the first-run experience and read-only mode.
//
// On first launch (no .first_run_done marker):
//   - App enters read-only mode: user can browse UI but cannot execute tools,
//     modify preferences, or trigger DAG runs.
//   - A guided onboarding flow is presented (steps stored here).
//   - After completing onboarding, MarkComplete() is called, which writes
//     the marker file and exits read-only mode.
//
// Read-only mode can also be entered manually (e.g. demo mode, shared screens).
//
// This module does NOT depend on browser preload (Wails embeds frontend).
// Legacy reference: TASKS_1_1.md 遺留能力 #6.
package onboarding

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Step represents a single onboarding guide step.
type Step struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// State is the current onboarding / read-only status.
type State struct {
	IsFirstRun   bool   `json:"is_first_run"`
	ReadOnlyMode bool   `json:"read_only_mode"`
	Steps        []Step `json:"steps"`
	CurrentStep  int    `json:"current_step"` // 0-based index
}

// Service manages onboarding state.
type Service struct {
	mu       sync.Mutex
	dataRoot string
	state    State
}

// NewService creates an onboarding service. Detects first-run from marker file.
func NewService(dataRoot string) *Service {
	isFirstRun := !fileExists(filepath.Join(dataRoot, "data", ".first_run_done"))
	steps := defaultSteps()
	return &Service{
		dataRoot: dataRoot,
		state: State{
			IsFirstRun:   isFirstRun,
			ReadOnlyMode: isFirstRun, // first-run → read-only until onboarding complete
			Steps:        steps,
			CurrentStep:  0,
		},
	}
}

// GetState returns the current onboarding state (read-only binding).
func (s *Service) GetState() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// IsReadOnly returns whether the app is currently in read-only mode.
// All mutation operations (tool execution, DAG start, preference save) should check this.
func (s *Service) IsReadOnly() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.ReadOnlyMode
}

// CompleteStep marks a step as completed and advances to the next.
func (s *Service) CompleteStep(stepID string) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, step := range s.state.Steps {
		if step.ID == stepID {
			s.state.Steps[i].Completed = true
			break
		}
	}
	// Advance current step to next incomplete
	for i, step := range s.state.Steps {
		if !step.Completed {
			s.state.CurrentStep = i
			return s.state
		}
	}
	// All steps complete → exit read-only
	s.state.CurrentStep = len(s.state.Steps)
	return s.state
}

// GoBack moves to the previous step (uncompletes it). Noop if already at step 0.
func (s *Service) GoBack() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.CurrentStep > 0 {
		// 先取消目前步驟的完成狀態（如果已完成）
		cur := s.state.CurrentStep
		if cur < len(s.state.Steps) {
			s.state.Steps[cur].Completed = false
		}
		// 回到上一步
		s.state.CurrentStep = cur - 1
		s.state.Steps[s.state.CurrentStep].Completed = false
	}
	return s.state
}

// MarkComplete finishes onboarding, writes marker, exits read-only mode.
func (s *Service) MarkComplete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ReadOnlyMode = false
	s.state.IsFirstRun = false
	// Write marker file
	markerPath := filepath.Join(s.dataRoot, "data", ".first_run_done")
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(markerPath, []byte(time.Now().Format(time.RFC3339)), 0o600)
}

// EnterReadOnly manually enables read-only mode (e.g. demo mode).
func (s *Service) EnterReadOnly() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ReadOnlyMode = true
}

// ExitReadOnly manually exits read-only mode.
func (s *Service) ExitReadOnly() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ReadOnlyMode = false
}

func defaultSteps() []Step {
	return []Step{
		{ID: "welcome", Title: "歡迎", Description: "認識 AI Console 的基本介面", Completed: false},
		{ID: "adapter", Title: "選擇 Adapter", Description: "選擇你要使用的 CLI（Claude / Codex / Gemini）", Completed: false},
		{ID: "persona", Title: "設定人格", Description: "設定助手的名稱與回應風格", Completed: false},
		{ID: "tool", Title: "試用工具", Description: "開啟工具面板，嘗試使用一個工具", Completed: false},
		{ID: "complete", Title: "完成", Description: "你已準備好開始使用！", Completed: false},
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
