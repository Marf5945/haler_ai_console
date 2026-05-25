package controlled_trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DeviceTrustProfile records the hardware characteristics of a known device.
// It can reduce device_penalty for DPI / scale differences — but it
// CANNOT cancel the first-replay dry-run requirement on any new device.
// Corresponds to schema #54 in TASKS_1_2.md.
type DeviceTrustProfile struct {
	DeviceProfileID           string    `json:"device_profile_id"`
	DPIBucket                 string    `json:"dpi_bucket"`            // e.g. "1x", "2x", "3x"
	DisplayScaleBucket        string    `json:"display_scale_bucket"`  // e.g. "100%", "125%", "150%"
	AppBundleHash             string    `json:"app_bundle_hash"`
	AccessibilityTreeAvailable bool     `json:"accessibility_tree_available"`
	FirstSeenAt               time.Time `json:"first_seen_at"`
	IsNewDevice               bool      `json:"is_new_device"` // always true until first dry-run completes
}

// Penalty limits for device profile adjustments (spec #32).
const (
	maxDPIPenaltyReduction         = 0.08
	maxResolutionPenaltyReduction  = 0.05
)

// DeviceProfileService manages device trust profiles.
type DeviceProfileService struct {
	mu        sync.Mutex
	storePath string
	profiles  []DeviceTrustProfile
	log       *TrustLog
}

func NewDeviceProfileService(trustDir string, log *TrustLog) *DeviceProfileService {
	svc := &DeviceProfileService{
		storePath: filepath.Join(trustDir, "device_trust_profiles.json"),
		log:       log,
	}
	_ = svc.load()
	return svc
}

// RegisterDevice records a new or updated device profile.
// New devices are always marked is_new_device=true until CompleteDryRun is called.
func (s *DeviceProfileService) RegisterDevice(profileID, dpiBucket, scaleBucket, appBundleHash string, a11yAvailable bool) (*DeviceTrustProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if device already exists.
	for i := range s.profiles {
		if s.profiles[i].DeviceProfileID == profileID {
			// Update existing profile.
			s.profiles[i].DPIBucket = dpiBucket
			s.profiles[i].DisplayScaleBucket = scaleBucket
			s.profiles[i].AppBundleHash = appBundleHash
			s.profiles[i].AccessibilityTreeAvailable = a11yAvailable
			p := s.profiles[i]
			return &p, s.saveLocked()
		}
	}

	// New device.
	profile := DeviceTrustProfile{
		DeviceProfileID:            profileID,
		DPIBucket:                  dpiBucket,
		DisplayScaleBucket:         scaleBucket,
		AppBundleHash:              appBundleHash,
		AccessibilityTreeAvailable: a11yAvailable,
		FirstSeenAt:                time.Now(),
		IsNewDevice:                true, // must complete dry-run before this becomes false
	}
	s.profiles = append(s.profiles, profile)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	_ = s.log.Append(TrustLogEntry{
		Type:              "device_profile_registered",
		ScopeHash:         profileID,
		FinalRiskChanged:  false,
		HardRulesModified: false,
	})
	return &profile, nil
}

// CompleteDryRun marks a device as no longer new after the user completes first-replay dry-run.
// This does NOT cancel any risk gate or confidence gate requirements.
func (s *DeviceProfileService) CompleteDryRun(profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.profiles {
		if s.profiles[i].DeviceProfileID == profileID {
			s.profiles[i].IsNewDevice = false
			return s.saveLocked()
		}
	}
	return fmt.Errorf("device_profile: %q not found", profileID)
}

// ComputePenaltyReduction returns how much device_penalty may be reduced for a profile.
// Caps are enforced per spec: DPI max 0.08, resolution max 0.05.
func (s *DeviceProfileService) ComputePenaltyReduction(profileID, referenceDPI, referenceScale string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.profiles {
		if p.DeviceProfileID != profileID {
			continue
		}
		reduction := 0.0
		if p.DPIBucket != referenceDPI {
			reduction += maxDPIPenaltyReduction
		}
		if p.DisplayScaleBucket != referenceScale {
			reduction += maxResolutionPenaltyReduction
		}
		return reduction
	}
	return 0
}

// GetProfile returns a device profile by ID.
func (s *DeviceProfileService) GetProfile(profileID string) (*DeviceTrustProfile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.profiles {
		if p.DeviceProfileID == profileID {
			copy := p
			return &copy, true
		}
	}
	return nil, false
}

// --- persistence ---

func (s *DeviceProfileService) load() error {
	data, err := os.ReadFile(s.storePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.profiles)
}

func (s *DeviceProfileService) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storePath, data, 0o600)
}
