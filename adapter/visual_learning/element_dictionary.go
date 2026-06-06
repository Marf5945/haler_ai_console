package visual_learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DictEntryStatus tracks whether a dictionary entry has passed review.
type DictEntryStatus string

const (
	DictEntryPending  DictEntryStatus = "pending"
	DictEntryApproved DictEntryStatus = "approved"
	DictEntryRejected DictEntryStatus = "rejected"
)

// ElementDictionaryEntry records a discovered UI element.
// Status remains "pending" until a Review passes — it is NOT usable for automation
// until approved.
// Corresponds to schema #53 in TASKS_1_2.md.
type ElementDictionaryEntry struct {
	ID             string          `json:"id"`
	FingerprintID  string          `json:"fingerprint_id"`
	CanonicalLabel CanonicalLabel  `json:"canonical_label"`
	Status         DictEntryStatus `json:"status"`
	Confidence     float64         `json:"confidence"`
	CreatedAt      time.Time       `json:"created_at"`
	ReviewedAt     *time.Time      `json:"reviewed_at,omitempty"`
}

// ElementDictionary manages the element dictionary for one project.
type ElementDictionary struct {
	mu      sync.Mutex
	path    string
	entries []ElementDictionaryEntry
}

func NewElementDictionary(learnDir string) *ElementDictionary {
	d := &ElementDictionary{
		path: filepath.Join(learnDir, "dictionaries", "element_dictionary.json"),
	}
	_ = d.load()
	return d
}

// Add inserts a new entry with status=pending.
// Entries are NOT usable for automation until Review approves them.
func (d *ElementDictionary) Add(fingerprintID string, label CanonicalLabel, confidence float64) (*ElementDictionaryEntry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	entry := ElementDictionaryEntry{
		ID:             fmt.Sprintf("elem-%d", time.Now().UnixNano()),
		FingerprintID:  fingerprintID,
		CanonicalLabel: label,
		Status:         DictEntryPending,
		Confidence:     confidence,
		CreatedAt:      time.Now(),
	}
	d.entries = append(d.entries, entry)
	return &entry, d.saveLocked()
}

// Approve marks an entry as approved after Review. Only approved entries may be
// used for automation.
func (d *ElementDictionary) Approve(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.entries {
		if d.entries[i].ID == id {
			now := time.Now()
			d.entries[i].Status = DictEntryApproved
			d.entries[i].ReviewedAt = &now
			return d.saveLocked()
		}
	}
	return fmt.Errorf("element_dictionary: entry %q not found", id)
}

// List returns a copy of all entries, optionally filtered by status.
func (d *ElementDictionary) List(status DictEntryStatus) []ElementDictionaryEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	var result []ElementDictionaryEntry
	for _, e := range d.entries {
		if status == "" || e.Status == status {
			result = append(result, e)
		}
	}
	return result
}

func (d *ElementDictionary) load() error {
	data, err := os.ReadFile(d.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &d.entries)
}

func (d *ElementDictionary) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(d.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(d.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.path, data, 0o600)
}
