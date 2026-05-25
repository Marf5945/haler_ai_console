package visual_learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ActionRiskLevel mirrors the global risk classification for action entries.
type ActionRiskLevel string

const (
	ActionRiskLow      ActionRiskLevel = "low"
	ActionRiskMedium   ActionRiskLevel = "medium"
	ActionRiskHigh     ActionRiskLevel = "high"
	ActionRiskCritical ActionRiskLevel = "critical"
)

// ActionStep is one step in a replayable action script.
type ActionStep struct {
	Order          int    `json:"order"`
	ElementID      string `json:"element_id"` // reference to ElementDictionary entry
	CanonicalLabel CanonicalLabel `json:"canonical_label"`
	WaitAfterMs    int    `json:"wait_after_ms"`
}

// ActionDictionaryEntry records a replayable UI action script.
// Status "pending_action_candidate" means the entry CANNOT be executed yet.
// Only "formal" entries (status=approved) may be executed — after Review or user confirm.
// Corresponds to schema #53 in TASKS_1_2.md.
type ActionDictionaryEntry struct {
	ID                  string          `json:"id"`
	Steps               []ActionStep    `json:"steps"`
	Status              string          `json:"status"` // "pending_action_candidate" | "approved" | "rejected"
	RiskLevel           ActionRiskLevel `json:"risk_level"`
	SourceLearningRunID string          `json:"source_learning_run_id"`
	CreatedAt           time.Time       `json:"created_at"`
	ReviewedAt          *time.Time      `json:"reviewed_at,omitempty"`
}

// ActionDictionary manages the action dictionary for one project.
// Separation from ElementDictionary is intentional (spec §22).
type ActionDictionary struct {
	mu           sync.Mutex
	formalPath   string
	pendingPath  string
	formalEntries []ActionDictionaryEntry
}

func NewActionDictionary(learnDir string) *ActionDictionary {
	d := &ActionDictionary{
		formalPath:  filepath.Join(learnDir, "dictionaries", "action_dictionary.json"),
		pendingPath: filepath.Join(learnDir, "pending", "pending_action_candidate.json"),
	}
	_ = d.loadFormal()
	return d
}

// AddPendingCandidate records a new action as pending — it cannot be executed.
// All new actions from Learning Mode start as pending_action_candidate.
func (d *ActionDictionary) AddPendingCandidate(steps []ActionStep, riskLevel ActionRiskLevel, learningRunID string) (*ActionDictionaryEntry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	entry := ActionDictionaryEntry{
		ID:                  fmt.Sprintf("action-%d", time.Now().UnixNano()),
		Steps:               steps,
		Status:              "pending_action_candidate",
		RiskLevel:           riskLevel,
		SourceLearningRunID: learningRunID,
		CreatedAt:           time.Now(),
	}
	return &entry, d.appendPendingLocked(entry)
}

// Approve moves a pending candidate to the formal dictionary after Review.
// The formal entry may then be executed.
func (d *ActionDictionary) Approve(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Load pending, find entry, move to formal.
	pending := d.loadPendingLocked()
	var found *ActionDictionaryEntry
	var remaining []ActionDictionaryEntry
	for _, e := range pending {
		if e.ID == id {
			found = &e
		} else {
			remaining = append(remaining, e)
		}
	}
	if found == nil {
		return fmt.Errorf("action_dictionary: pending entry %q not found", id)
	}
	now := time.Now()
	found.Status = "approved"
	found.ReviewedAt = &now
	d.formalEntries = append(d.formalEntries, *found)
	if err := d.saveFormalLocked(); err != nil {
		return err
	}
	return d.savePendingLocked(remaining)
}

// ListFormal returns all approved (executable) action entries.
func (d *ActionDictionary) ListFormal() []ActionDictionaryEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]ActionDictionaryEntry(nil), d.formalEntries...)
}

// ListPending returns all pending action candidates.
func (d *ActionDictionary) ListPending() []ActionDictionaryEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.loadPendingLocked()
}

// --- persistence ---

func (d *ActionDictionary) appendPendingLocked(entry ActionDictionaryEntry) error {
	existing := d.loadPendingLocked()
	existing = append(existing, entry)
	return d.savePendingLocked(existing)
}

func (d *ActionDictionary) loadPendingLocked() []ActionDictionaryEntry {
	data, err := os.ReadFile(d.pendingPath)
	if err != nil {
		return nil
	}
	var entries []ActionDictionaryEntry
	_ = json.Unmarshal(data, &entries)
	return entries
}

func (d *ActionDictionary) savePendingLocked(entries []ActionDictionaryEntry) error {
	if err := os.MkdirAll(filepath.Dir(d.pendingPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.pendingPath, data, 0o600)
}

func (d *ActionDictionary) loadFormal() error {
	data, err := os.ReadFile(d.formalPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &d.formalEntries)
}

func (d *ActionDictionary) saveFormalLocked() error {
	if err := os.MkdirAll(filepath.Dir(d.formalPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(d.formalEntries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.formalPath, data, 0o600)
}
