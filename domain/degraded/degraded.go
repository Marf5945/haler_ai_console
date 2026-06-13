// Package degraded implements the Wails-native degraded mode controller.
//
// When degraded mode is active:
//   - High-risk capabilities are disabled (tool execution blocked).
//   - All open ReviewCards are invalidated (作廢).
//   - DAG resume is forbidden; running DAGs are halted.
//   - Status Rail shows degraded indicator.
//
// Entry conditions (any one triggers degraded mode):
//   - Adapter connectivity loss (all adapters offline).
//   - Critical config corruption detected.
//   - Manual trigger via Wails binding (admin/testing).
//
// Exit conditions:
//   - At least one adapter returns to online status.
//   - Manual exit via Wails binding.
//
// Legacy reference: TASKS_1_1.md 遺留能力 #5.
package degraded

import (
	"sync"
	"time"
)

// Reason describes why degraded mode was entered.
type Reason string

const (
	ReasonAllAdaptersOffline Reason = "all_adapters_offline"
	ReasonConfigCorruption   Reason = "config_corruption"
	ReasonManualTrigger      Reason = "manual_trigger"
)

// State represents the current degraded mode status.
type State struct {
	Active     bool      `json:"active"`
	Reason     Reason    `json:"reason,omitempty"`
	EnteredAt  time.Time `json:"entered_at,omitempty"`
	BlockedOps []string  `json:"blocked_ops"` // list of blocked operation categories
}

// Service manages degraded mode transitions.
type Service struct {
	mu    sync.Mutex
	state State
}

// NewService creates a degraded mode service (starts in normal mode).
func NewService() *Service {
	return &Service{
		state: State{Active: false, BlockedOps: []string{}},
	}
}

// GetState returns the current degraded mode state (read-only).
func (s *Service) GetState() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// Enter activates degraded mode with the given reason.
// Idempotent — if already active, updates reason but does not reset timer.
func (s *Service) Enter(reason Reason) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.Active {
		s.state.EnteredAt = time.Now()
	}
	s.state.Active = true
	s.state.Reason = reason
	s.state.BlockedOps = []string{
		"tool_execution",
		"dag_resume",
		"review_card_action",
		"package_install",
		"trust_elevation",
	}
	return s.state
}

// Exit deactivates degraded mode. Returns the new (normal) state.
func (s *Service) Exit() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = State{Active: false, BlockedOps: []string{}}
	return s.state
}

// IsBlocked checks whether a specific operation category is blocked.
// Use before executing any high-risk action.
func (s *Service) IsBlocked(opCategory string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.Active {
		return false
	}
	for _, blocked := range s.state.BlockedOps {
		if blocked == opCategory {
			return true
		}
	}
	return false
}
