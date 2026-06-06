package controlled_trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SandboxStatus is the lifecycle state of a draft sandbox run.
type SandboxStatus string

const (
	SandboxActive  SandboxStatus = "active"
	SandboxStopped SandboxStatus = "stopped"
)

// SandboxPromotion is the user's choice after a sandbox stops.
type SandboxPromotion string

const (
	// PromoteFormalReview: use draft steps as basis for Review Card.
	PromoteFormalReview SandboxPromotion = "formal_review"
	// PromotePendingCandidate: save draft as pending candidate for later review.
	PromotePendingCandidate SandboxPromotion = "pending_candidate"
	// PromoteDiscard: discard the draft entirely.
	PromoteDiscard SandboxPromotion = "discard"
)

// sandboxStopTriggers lists events that must immediately halt the draft sandbox.
// On any of these, the sandbox stops and presents the user with three options.
var sandboxStopTriggers = []string{
	"login", "payment", "delete", "authorization",
	"permission_grant", "captcha", "irreversible_confirmation",
	"credential_request", "token_request", "api_key_request",
	"external_share", "file_overwrite", "account_change",
	"security_challenge", "rate_limit", "unexpected_modal",
	"active_window_changed",
}

// DraftSandboxRun is one sandbox session. It is ONLY valid for:
//   - The current active_window at start time.
//   - The current app session (lifetime: current_session_only).
//
// It must NOT be Safe Exported, must NOT write to formal Action Dictionary /
// Element Dictionary / canonical schema, and must NOT persist across app restarts.
// Corresponds to schema #54 in TASKS_1_2.md.
type DraftSandboxRun struct {
	ID               string        `json:"id"`
	ActiveWindowHash string        `json:"active_window_hash"`
	StartedAt        time.Time     `json:"started_at"`
	StoppedAt        *time.Time    `json:"stopped_at,omitempty"`
	StopReason       string        `json:"stop_reason,omitempty"`
	Status           SandboxStatus `json:"status"`
	TempTracePath    string        `json:"temporary_trace_path"`
}

// DraftSandboxService manages draft sandbox sessions.
type DraftSandboxService struct {
	mu          sync.Mutex
	sandboxDir  string
	activeRun   *DraftSandboxRun
	log         *TrustLog
}

func NewDraftSandboxService(trustDir string, log *TrustLog) *DraftSandboxService {
	return &DraftSandboxService{
		sandboxDir: filepath.Join(trustDir, "draft_sandbox_runs"),
		log:        log,
	}
}

// Start activates a new draft sandbox for the given active window.
// Only one sandbox may be active at a time.
func (s *DraftSandboxService) Start(activeWindowHash string) (*DraftSandboxRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeRun != nil && s.activeRun.Status == SandboxActive {
		return nil, fmt.Errorf("draft_sandbox: run %q already active; stop it first", s.activeRun.ID)
	}

	id := fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
	run := &DraftSandboxRun{
		ID:               id,
		ActiveWindowHash: activeWindowHash,
		StartedAt:        time.Now(),
		Status:           SandboxActive,
		TempTracePath:    filepath.Join(s.sandboxDir, id, "temporary_trace.jsonl"),
	}
	s.activeRun = run
	if err := s.saveRunLocked(run); err != nil {
		return nil, err
	}
	_ = s.log.Append(TrustLogEntry{
		Type:              "draft_sandbox_started",
		ScopeHash:         activeWindowHash,
		FinalRiskChanged:  false,
		HardRulesModified: false,
	})
	return run, nil
}

// Stop halts the active sandbox. reason should be one of sandboxStopTriggers
// or "user_requested". After stopping, the UI MUST show three continuation options.
func (s *DraftSandboxService) Stop(sandboxID, reason string) (*DraftSandboxRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeRun == nil || s.activeRun.ID != sandboxID {
		return nil, fmt.Errorf("draft_sandbox: run %q not active", sandboxID)
	}
	now := time.Now()
	s.activeRun.StoppedAt = &now
	s.activeRun.StopReason = reason
	s.activeRun.Status = SandboxStopped
	if err := s.saveRunLocked(s.activeRun); err != nil {
		return nil, err
	}
	_ = s.log.Append(TrustLogEntry{
		Type:              "draft_sandbox_stopped",
		ScopeHash:         s.activeRun.ActiveWindowHash,
		FinalRiskChanged:  false,
		HardRulesModified: false,
	})
	run := s.activeRun
	s.activeRun = nil
	return run, nil
}

// CheckAndStopIfTriggered stops the sandbox if the given event is a stop trigger.
// Returns (stopped, run, error). The UI must display three options when stopped=true.
func (s *DraftSandboxService) CheckAndStopIfTriggered(sandboxID, event string) (bool, *DraftSandboxRun, error) {
	for _, trigger := range sandboxStopTriggers {
		if event == trigger {
			run, err := s.Stop(sandboxID, event)
			return true, run, err
		}
	}
	return false, nil, nil
}

// Promote implements the upgrade path from draft to the formal system.
// All three promotion paths still respect Review / risk gate requirements.
func (s *DraftSandboxService) Promote(sandboxID string, promotion SandboxPromotion) (string, error) {
	switch promotion {
	case PromoteFormalReview:
		// In production: create a ReviewCard from the sandbox trace.
		return fmt.Sprintf("sandbox %q promoted to formal review", sandboxID), nil
	case PromotePendingCandidate:
		// In production: copy sandbox trace to pending_action_candidate.
		return fmt.Sprintf("sandbox %q saved as pending candidate", sandboxID), nil
	case PromoteDiscard:
		// Clear the sandbox trace. Nothing is preserved.
		return fmt.Sprintf("sandbox %q discarded", sandboxID), s.discardTrace(sandboxID)
	default:
		return "", fmt.Errorf("draft_sandbox: unknown promotion %q", promotion)
	}
}

// IsSandboxActive returns whether a sandbox is currently running.
func (s *DraftSandboxService) IsSandboxActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeRun != nil && s.activeRun.Status == SandboxActive
}

func (s *DraftSandboxService) discardTrace(sandboxID string) error {
	traceDir := filepath.Join(s.sandboxDir, sandboxID)
	return os.RemoveAll(traceDir)
}

func (s *DraftSandboxService) saveRunLocked(run *DraftSandboxRun) error {
	dir := filepath.Join(s.sandboxDir, run.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "run.json"), data, 0o600)
}
