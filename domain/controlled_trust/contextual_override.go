// Package controlled_trust implements the Controlled Trust / Draft Sandbox module (v3.2.0).
// Core principle: the system may trust the user's intention, but must not let intention
// rewrite hard safety boundaries.
package controlled_trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OverrideStatus is the lifecycle state of a contextual risk override.
type OverrideStatus string

const (
	OverrideActive      OverrideStatus = "active"
	OverrideExpired     OverrideStatus = "expired"
	OverrideInvalidated OverrideStatus = "invalidated"
)

// AllowedRiskLevel enumerates the risk levels that MAY be covered by an override.
// critical, destructive, and listed forbidden operations can NEVER be covered.
type AllowedRiskLevel string

const (
	OverrideLow             AllowedRiskLevel = "low"
	OverrideMedium          AllowedRiskLevel = "medium"
	OverrideHighNonDestructive AllowedRiskLevel = "high_non_destructive"
)

// neverOverridable lists operations that can NEVER be covered by a contextual override.
var neverOverridable = []string{
	"critical", "destructive", "delete_permanent",
	"auth", "credential", "token", "api_key",
	"permission_grant", "payment", "external_share",
	"security_challenge", "captcha", "irreversible_confirmation",
}

// OverrideScope binds the override to a specific execution context.
// All fields are REQUIRED — a missing field invalidates the scope.
// Corresponds to schema #54 in TASKS_1_2.md.
type OverrideScope struct {
	DAGRunID          string `json:"dag_run_id"`
	WorkspaceID       string `json:"workspace_id"`
	OperationFamily   string `json:"operation_family"`
	TargetHashSet     string `json:"target_hash_set"`
	PlanHash          string `json:"plan_hash"`
	RiskPolicyHash    string `json:"risk_policy_hash"`
	ToolRegistryHash  string `json:"tool_registry_hash"`
	DeviceProfileID   string `json:"device_profile_id"`
	Expiry            time.Time `json:"expiry"`
}

// ContextualRiskOverride allows reducing repeated review interruptions for
// clearly intentional workflows. It ONLY reduces review frequency — it does NOT
// lower final_risk or disable hard rules.
// Corresponds to schema #54 in TASKS_1_2.md.
type ContextualRiskOverride struct {
	ID                 string           `json:"id"`
	Scope              OverrideScope    `json:"scope"`
	AllowedRisk        AllowedRiskLevel `json:"allowed_risk"`
	Status             OverrideStatus   `json:"status"`
	CreatedAt          time.Time        `json:"created_at"`
	InvalidatedReason  string           `json:"invalidated_reason,omitempty"`
}

// ContextualOverrideService manages contextual risk overrides.
type ContextualOverrideService struct {
	mu        sync.Mutex
	storePath string
	overrides []ContextualRiskOverride
	log       *TrustLog
}

func NewContextualOverrideService(trustDir string, log *TrustLog) *ContextualOverrideService {
	svc := &ContextualOverrideService{
		storePath: filepath.Join(trustDir, "contextual_risk_overrides.json"),
		log:       log,
	}
	_ = svc.load()
	return svc
}

// Enable creates a new contextual risk override after validating the scope.
// Returns an error if any forbidden operation family is requested.
func (s *ContextualOverrideService) Enable(scope OverrideScope, allowedRisk AllowedRiskLevel) (*ContextualRiskOverride, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if err := validateAllowedRisk(scope.OperationFamily, allowedRisk); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	override := ContextualRiskOverride{
		ID:          fmt.Sprintf("override-%d", time.Now().UnixNano()),
		Scope:       scope,
		AllowedRisk: allowedRisk,
		Status:      OverrideActive,
		CreatedAt:   time.Now(),
	}
	s.overrides = append(s.overrides, override)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	// Write to trust log — mandatory for all trust decisions.
	_ = s.log.Append(TrustLogEntry{
		Type:              "contextual_override_enabled",
		ScopeHash:         computeScopeHash(scope),
		FinalRiskChanged:  false, // override never changes final_risk
		HardRulesModified: false,
	})
	return &override, nil
}

// Disable deactivates an override by ID.
func (s *ContextualOverrideService) Disable(overrideID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.overrides {
		if s.overrides[i].ID == overrideID {
			s.overrides[i].Status = OverrideExpired
			if err := s.saveLocked(); err != nil {
				return err
			}
			_ = s.log.Append(TrustLogEntry{
				Type:              "contextual_override_disabled",
				ScopeHash:         computeScopeHash(s.overrides[i].Scope),
				FinalRiskChanged:  false,
				HardRulesModified: false,
			})
			return nil
		}
	}
	return fmt.Errorf("contextual_override: %q not found", overrideID)
}

// InvalidateIfScopeChanged checks whether the given hashes differ from those
// recorded in an active override, and invalidates it if so.
func (s *ContextualOverrideService) InvalidateIfScopeChanged(overrideID, planHash, toolHash, riskHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.overrides {
		o := &s.overrides[i]
		if o.ID != overrideID || o.Status != OverrideActive {
			continue
		}
		if o.Scope.PlanHash != planHash || o.Scope.ToolRegistryHash != toolHash || o.Scope.RiskPolicyHash != riskHash {
			o.Status = OverrideInvalidated
			o.InvalidatedReason = "scope hash changed (plan, tool registry, or risk policy)"
			_ = s.saveLocked()
			_ = s.log.Append(TrustLogEntry{
				Type:              "contextual_override_invalidated",
				ScopeHash:         computeScopeHash(o.Scope),
				FinalRiskChanged:  false,
				HardRulesModified: false,
			})
		}
	}
}

// ListActive returns all currently active overrides.
func (s *ContextualOverrideService) ListActive() []ContextualRiskOverride {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var result []ContextualRiskOverride
	for _, o := range s.overrides {
		if o.Status == OverrideActive && now.Before(o.Scope.Expiry) {
			result = append(result, o)
		}
	}
	return result
}

// --- validation ---

func validateScope(scope OverrideScope) error {
	if scope.DAGRunID == "" || scope.WorkspaceID == "" || scope.OperationFamily == "" ||
		scope.TargetHashSet == "" || scope.PlanHash == "" || scope.RiskPolicyHash == "" ||
		scope.ToolRegistryHash == "" || scope.DeviceProfileID == "" {
		return fmt.Errorf("contextual_override: all scope fields are required")
	}
	if scope.Expiry.IsZero() {
		return fmt.Errorf("contextual_override: expiry must be set")
	}
	return nil
}

func validateAllowedRisk(operationFamily string, allowedRisk AllowedRiskLevel) error {
	for _, forbidden := range neverOverridable {
		if operationFamily == forbidden {
			return fmt.Errorf("contextual_override: operation_family %q is never overridable", operationFamily)
		}
	}
	switch allowedRisk {
	case OverrideLow, OverrideMedium, OverrideHighNonDestructive:
		return nil
	default:
		return fmt.Errorf("contextual_override: allowed_risk %q is not permitted", allowedRisk)
	}
}

// --- persistence ---

func (s *ContextualOverrideService) load() error {
	data, err := os.ReadFile(s.storePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.overrides)
}

func (s *ContextualOverrideService) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.overrides, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storePath, data, 0o600)
}
