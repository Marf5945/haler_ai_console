package visual_learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CandidateAgeStatus is the lifecycle stage of a pending action candidate.
// Lifecycle: fresh_pending (0–14d) → stale_pending (14–30d) → archived_pending (30d+)
type CandidateAgeStatus string

const (
	AgeStatusFresh    CandidateAgeStatus = "fresh_pending"
	AgeStatusStale    CandidateAgeStatus = "stale_pending"
	AgeStatusArchived CandidateAgeStatus = "archived_pending"
)

// Lifecycle thresholds (in days).
const (
	freshDays  = 14
	staleDays  = 30
	stalePenalty = -0.10 // confidence penalty applied when stale
)

// Capacity limits per project.
const (
	softLimitPerProject    = 100
	hardLimitPerProject    = 300
	softLimitPerSubagent   = 30
	hardLimitPerSubagent   = 100
)

// PendingCandidateRecord wraps a pending action candidate with lifecycle metadata.
// Corresponds to schema #53 in TASKS_1_2.md.
type PendingCandidateRecord struct {
	ID               string             `json:"id"`
	Type             string             `json:"type"` // "action" | "label" | "fingerprint_patch" | "target_remap" | "risk_word"
	Status           CandidateAgeStatus `json:"status"`
	AgeDays          int                `json:"age_days"`
	ConfidencePenalty float64           `json:"confidence_penalty"`
	CreatedAt        time.Time          `json:"created_at"`
	ArchivedAt       *time.Time         `json:"archived_at,omitempty"`
	SubagentID       string             `json:"subagent_id,omitempty"`
}

// PendingCandidateManager enforces lifecycle rules and capacity limits.
type PendingCandidateManager struct {
	mu          sync.Mutex
	pendingPath string
	records     []PendingCandidateRecord
}

func NewPendingCandidateManager(learnDir string) *PendingCandidateManager {
	m := &PendingCandidateManager{
		pendingPath: filepath.Join(learnDir, "pending", "pending_action_candidate.json"),
	}
	_ = m.load()
	return m
}

// Add inserts a new pending candidate, enforcing hard limits.
// When hard limit is reached, the oldest stale entry is archived first.
func (m *PendingCandidateManager) Add(candidateType, subagentID string) (*PendingCandidateRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.refreshAgesLocked()

	active := m.activeCountLocked()
	if active >= hardLimitPerProject {
		if err := m.archiveOldestStaleLocked(); err != nil {
			return nil, fmt.Errorf("pending_candidate: hard limit %d reached and no stale entries to archive", hardLimitPerProject)
		}
	}

	rec := PendingCandidateRecord{
		ID:         fmt.Sprintf("pc-%d", time.Now().UnixNano()),
		Type:       candidateType,
		Status:     AgeStatusFresh,
		AgeDays:    0,
		CreatedAt:  time.Now(),
		SubagentID: subagentID,
	}
	m.records = append(m.records, rec)
	return &rec, m.saveLocked()
}

// RefreshAges updates age_status and applies confidence penalties.
// Call this before any read that depends on lifecycle state.
func (m *PendingCandidateManager) RefreshAges() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshAgesLocked()
	_ = m.saveLocked()
}

// List returns all non-archived pending records.
func (m *PendingCandidateManager) List() []PendingCandidateRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshAgesLocked()
	var result []PendingCandidateRecord
	for _, r := range m.records {
		if r.Status != AgeStatusArchived {
			result = append(result, r)
		}
	}
	return result
}

// ActiveCount returns the number of non-archived candidates.
func (m *PendingCandidateManager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeCountLocked()
}

// --- internal ---

func (m *PendingCandidateManager) refreshAgesLocked() {
	now := time.Now()
	for i := range m.records {
		r := &m.records[i]
		if r.Status == AgeStatusArchived {
			continue
		}
		days := int(now.Sub(r.CreatedAt).Hours() / 24)
		r.AgeDays = days
		switch {
		case days < freshDays:
			r.Status = AgeStatusFresh
			r.ConfidencePenalty = 0
		case days < staleDays:
			r.Status = AgeStatusStale
			r.ConfidencePenalty = stalePenalty
		default:
			t := now
			r.Status = AgeStatusArchived
			r.ArchivedAt = &t
		}
	}
}

func (m *PendingCandidateManager) activeCountLocked() int {
	count := 0
	for _, r := range m.records {
		if r.Status != AgeStatusArchived {
			count++
		}
	}
	return count
}

func (m *PendingCandidateManager) archiveOldestStaleLocked() error {
	now := time.Now()
	for i := range m.records {
		if m.records[i].Status == AgeStatusStale {
			m.records[i].Status = AgeStatusArchived
			m.records[i].ArchivedAt = &now
			return m.saveLocked()
		}
	}
	return fmt.Errorf("no stale entries to archive")
}

func (m *PendingCandidateManager) load() error {
	data, err := os.ReadFile(m.pendingPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.records)
}

func (m *PendingCandidateManager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.pendingPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.pendingPath, data, 0o600)
}
