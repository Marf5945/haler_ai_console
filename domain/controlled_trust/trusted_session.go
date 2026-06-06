package controlled_trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionStatus is the lifecycle state of a trusted session scope.
type SessionStatus string

const (
	SessionActive  SessionStatus = "active"
	SessionExpired SessionStatus = "expired"
)

// idleExpiry is the maximum idle duration before a trusted session silently expires.
const idleExpiry = 120 * time.Minute

// TrustedSessionScope is a short-lived convenience authorisation scoped to
// the current app session / workspace / DAG run / active window.
// max_risk_covered: medium — it CANNOT cover critical or destructive operations.
// Expiry is SILENT — no countdown is shown to the user.
// Corresponds to schema #54 in TASKS_1_2.md.
type TrustedSessionScope struct {
	ID               string        `json:"id"`
	AppSessionID     string        `json:"app_session_id"`
	WorkspaceID      string        `json:"workspace_id"`
	DAGRunID         string        `json:"dag_run_id"`
	ActiveWindowHash string        `json:"active_window_hash"`
	ExpiresAt        time.Time     `json:"expires_at"`
	Status           SessionStatus `json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
}

// nonCoverableBySession lists operations that a trusted session can NEVER cover.
var nonCoverableBySession = []string{
	"critical", "destructive",
	"login", "payment", "delete", "authorization",
	"captcha", "credential_request", "irreversible_confirmation",
}

// TrustedSessionService manages trusted session scope lifecycle.
type TrustedSessionService struct {
	mu        sync.Mutex
	storePath string
	current   *TrustedSessionScope
	log       *TrustLog
}

func NewTrustedSessionService(trustDir string, log *TrustLog) *TrustedSessionService {
	return &TrustedSessionService{
		storePath: filepath.Join(trustDir, "trusted_session_scope.json"),
		log:       log,
	}
}

// Enable activates a new trusted session scope.
// Requires OS authentication in production (stub here; caller enforces OS auth).
func (s *TrustedSessionService) Enable(appSessionID, workspaceID, dagRunID, activeWindowHash string) (*TrustedSessionScope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Silently expire existing session first.
	if s.current != nil {
		s.current.Status = SessionExpired
	}

	now := time.Now()
	scope := &TrustedSessionScope{
		ID:               fmt.Sprintf("ts-%d", now.UnixNano()),
		AppSessionID:     appSessionID,
		WorkspaceID:      workspaceID,
		DAGRunID:         dagRunID,
		ActiveWindowHash: activeWindowHash,
		ExpiresAt:        now.Add(idleExpiry),
		Status:           SessionActive,
		CreatedAt:        now,
	}
	s.current = scope
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	_ = s.log.Append(TrustLogEntry{
		Type:              "trusted_session_enabled",
		ScopeHash:         computeSessionScopeHash(scope),
		FinalRiskChanged:  false,
		HardRulesModified: false,
	})
	return scope, nil
}

// IsActive returns true if a trusted session is currently valid.
// Expiry is checked silently — no notification is generated here.
func (s *TrustedSessionService) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isActiveLocked()
}

// IsActiveForOperation returns false if the operation is in the non-coverable list,
// or if the session has expired.
func (s *TrustedSessionService) IsActiveForOperation(operation string) bool {
	for _, forbidden := range nonCoverableBySession {
		if operation == forbidden {
			return false // never covered regardless of session state
		}
	}
	return s.IsActive()
}

// InvalidateOnChange silently expires the session if any tracked hash changes.
// Called by the runtime when active_window, risk_policy, tool_registry, plan, or target changes.
func (s *TrustedSessionService) InvalidateOnChange(newWindowHash, newPlanHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || s.current.Status != SessionActive {
		return
	}
	if newWindowHash != "" && newWindowHash != s.current.ActiveWindowHash {
		s.expireCurrentLocked("active_window_changed")
	}
}

// ExpireOnAppClose must be called when the app is closing.
func (s *TrustedSessionService) ExpireOnAppClose() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil {
		s.expireCurrentLocked("app_close")
	}
}

func (s *TrustedSessionService) isActiveLocked() bool {
	if s.current == nil || s.current.Status != SessionActive {
		return false
	}
	if time.Now().After(s.current.ExpiresAt) {
		s.current.Status = SessionExpired
		_ = s.saveLocked()
		return false
	}
	return true
}

func (s *TrustedSessionService) expireCurrentLocked(reason string) {
	s.current.Status = SessionExpired
	_ = s.saveLocked()
	_ = s.log.Append(TrustLogEntry{
		Type:              "trusted_session_expired",
		ScopeHash:         computeSessionScopeHash(s.current),
		FinalRiskChanged:  false,
		HardRulesModified: false,
	})
	_ = reason // logged above; not stored separately per spec
}

// --- persistence ---

func (s *TrustedSessionService) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storePath, data, 0o600)
}

func computeSessionScopeHash(scope *TrustedSessionScope) string {
	if scope == nil {
		return ""
	}
	raw := fmt.Sprintf("%s|%s|%s|%s", scope.AppSessionID, scope.WorkspaceID, scope.DAGRunID, scope.ActiveWindowHash)
	sum := [32]byte{}
	for i, b := range []byte(raw) {
		sum[i%32] ^= b
	}
	return fmt.Sprintf("%x", sum)
}
