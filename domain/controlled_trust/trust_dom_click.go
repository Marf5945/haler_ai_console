package controlled_trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TrustDomClickPreference is an ADVANCED preference that allows the system to
// proceed with DOM-confirmed, visually-verified low/medium clicks without
// per-step review. Default: OFF.
//
// Activation conditions (all must be true):
//   - dom_role_match
//   - dom_stable_id_or_label_match
//   - visual_confidence_min 0.60
//   - final_confidence_min 0.78
//   - risk_level_max medium
//
// Stop conditions (immediate halt on any of these):
//   - confirm_dialog_unexpected
//   - credential_request
//   - captcha
//   - payment_confirmation
//   - delete_confirmation
//   - authorization_prompt
//   - permission_prompt
//   - domain_changed
//   - active_window_changed
//
// Corresponds to schema #54 in TASKS_1_2.md.
type TrustDomClickPreference struct {
	Enabled              bool      `json:"enabled"`
	UpdatedAt            time.Time `json:"updated_at"`
	VisualConfidenceMin  float64   `json:"visual_confidence_min"`  // 0.60
	FinalConfidenceMin   float64   `json:"final_confidence_min"`   // 0.78
	RiskLevelMax         string    `json:"risk_level_max"`         // "medium"
}

// stopConditions is the set of events that must immediately halt Trust DOM & Click.
var stopConditions = []string{
	"confirm_dialog_unexpected",
	"credential_request",
	"captcha",
	"payment_confirmation",
	"delete_confirmation",
	"authorization_prompt",
	"permission_prompt",
	"domain_changed",
	"active_window_changed",
}

// TrustDomClickService manages the Trust DOM & Click preference.
type TrustDomClickService struct {
	mu         sync.Mutex
	storePath  string
	preference TrustDomClickPreference
	log        *TrustLog
}

func NewTrustDomClickService(trustDir string, log *TrustLog) *TrustDomClickService {
	svc := &TrustDomClickService{
		storePath: filepath.Join(trustDir, "trust_dom_click_preference.json"),
		log:       log,
		preference: TrustDomClickPreference{
			Enabled:             false, // DEFAULT: OFF (advanced_preference)
			VisualConfidenceMin: 0.60,
			FinalConfidenceMin:  0.78,
			RiskLevelMax:        "medium",
		},
	}
	_ = svc.load()
	return svc
}

// Set enables or disables Trust DOM & Click.
func (s *TrustDomClickService) Set(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preference.Enabled = enabled
	s.preference.UpdatedAt = time.Now()
	if err := s.saveLocked(); err != nil {
		return err
	}
	action := "trust_dom_click_disabled"
	if enabled {
		action = "trust_dom_click_enabled"
	}
	return s.log.Append(TrustLogEntry{
		Type:              action,
		ScopeHash:         "dom_click_preference",
		FinalRiskChanged:  false,
		HardRulesModified: false,
	})
}

// IsActiveFor returns true if Trust DOM & Click is enabled AND the given
// event/condition is not a stop condition AND confidence thresholds are met.
func (s *TrustDomClickService) IsActiveFor(event string, visualConfidence, finalConfidence float64, riskLevel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.preference.Enabled {
		return false
	}
	// Check stop conditions.
	for _, stop := range stopConditions {
		if event == stop {
			return false
		}
	}
	// Check confidence thresholds.
	if visualConfidence < s.preference.VisualConfidenceMin {
		return false
	}
	if finalConfidence < s.preference.FinalConfidenceMin {
		return false
	}
	// Only low/medium risk allowed.
	if riskLevel == "high" || riskLevel == "critical" || riskLevel == "destructive" {
		return false
	}
	return true
}

// MustStop returns true if the given event is a stop condition.
// When true, Trust DOM & Click MUST halt immediately.
func (s *TrustDomClickService) MustStop(event string) bool {
	for _, stop := range stopConditions {
		if event == stop {
			return true
		}
	}
	return false
}

// GetPreference returns the current preference.
func (s *TrustDomClickService) GetPreference() TrustDomClickPreference {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.preference
}

// --- persistence ---

func (s *TrustDomClickService) load() error {
	data, err := os.ReadFile(s.storePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.preference)
}

func (s *TrustDomClickService) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.preference, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storePath, data, 0o600)
}
