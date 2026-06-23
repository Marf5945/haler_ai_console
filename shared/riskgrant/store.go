// Package riskgrant stores short-lived confirmations for risky tool actions.
package riskgrant

import (
	"strings"
	"sync"
	"time"
)

const DefaultTTL = 5 * time.Minute

// GrantScope is the exact permission boundary for a temporary grant.
type GrantScope struct {
	ToolID         string    `json:"tool_id"`
	ActionTag      string    `json:"action_tag"`
	TargetIdentity string    `json:"target_identity"`
	RiskClass      string    `json:"risk_class"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// Store keeps grants in memory; callers may persist snapshots separately.
type Store struct {
	mu     sync.Mutex
	grants []GrantScope
	ttl    time.Duration
	now    func() time.Time
}

// NewStore creates a grant store with the default 5-minute TTL.
func NewStore() *Store {
	return &Store{ttl: DefaultTTL, now: time.Now}
}

// NewStoreWithClock supports deterministic tests.
func NewStoreWithClock(ttl time.Duration, now func() time.Time) *Store {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if now == nil {
		now = time.Now
	}
	return &Store{ttl: ttl, now: now}
}

// Grant records a confirmed high-risk scope.
func (s *Store) Grant(toolID, actionTag, targetIdentity, riskClass string) GrantScope {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.expireLocked()
	grant := GrantScope{
		ToolID:         strings.TrimSpace(toolID),
		ActionTag:      strings.TrimSpace(actionTag),
		TargetIdentity: strings.TrimSpace(targetIdentity),
		RiskClass:      strings.TrimSpace(riskClass),
		ExpiresAt:      s.now().Add(s.ttl),
	}
	s.grants = append(s.grants, grant)
	return grant
}

// HasValid returns true only for the same tool, action, stable target, and risk.
func (s *Store) HasValid(toolID, actionTag, targetIdentity, riskClass string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.expireLocked()
	for _, grant := range s.grants {
		if grant.ToolID == strings.TrimSpace(toolID) &&
			grant.ActionTag == strings.TrimSpace(actionTag) &&
			grant.TargetIdentity == strings.TrimSpace(targetIdentity) &&
			grant.RiskClass == strings.TrimSpace(riskClass) {
			return true
		}
	}
	return false
}

func (s *Store) expireLocked() {
	now := s.now()
	next := s.grants[:0]
	for _, grant := range s.grants {
		if now.Before(grant.ExpiresAt) {
			next = append(next, grant)
		}
	}
	s.grants = next
}
