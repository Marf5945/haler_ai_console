package visual_learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ElementType is the first axis of the canonical label schema.
type ElementType string

const (
	ElementButton    ElementType = "button"
	ElementInput     ElementType = "input"
	ElementLink      ElementType = "link"
	ElementCheckbox  ElementType = "checkbox"
	ElementDropdown  ElementType = "dropdown"
	ElementModal     ElementType = "modal"
	ElementIcon      ElementType = "icon"
	ElementContainer ElementType = "container"
	ElementText      ElementType = "text"
	ElementUnknown   ElementType = "unknown"
)

// ActionSemantic is the second axis of the canonical label schema.
type ActionSemantic string

const (
	ActionSubmit   ActionSemantic = "submit"
	ActionCancel   ActionSemantic = "cancel"
	ActionNavigate ActionSemantic = "navigate"
	ActionToggle   ActionSemantic = "toggle"
	ActionExpand   ActionSemantic = "expand"
	ActionClose    ActionSemantic = "close"
	ActionScroll   ActionSemantic = "scroll"
	ActionRead     ActionSemantic = "read"
	ActionUnknown  ActionSemantic = "unknown"
)

// CanonicalLabel is the combination of element_type × action_semantic.
// LLM free-form descriptions MUST be mapped to a CanonicalLabel.
// New labels MUST enter pending_label_candidate — never overwrite the schema directly.
type CanonicalLabel struct {
	ElementType    ElementType    `json:"element_type"`
	ActionSemantic ActionSemantic `json:"action_semantic"`
	Description    string         `json:"description,omitempty"`
}

// LabelCandidate is a proposed new label awaiting review.
type LabelCandidate struct {
	ID             string         `json:"id"`
	ProposedLabel  CanonicalLabel `json:"proposed_label"`
	SourceRegionID string         `json:"source_region_id"`
	LLMDescription string         `json:"llm_description"`
	Confidence     float64        `json:"confidence"`
	Status         string         `json:"status"` // "pending" | "approved" | "rejected"
	CreatedAt      time.Time      `json:"created_at"`
}

// CanonicalLabelService manages the schema and pending candidates.
type CanonicalLabelService struct {
	mu          sync.Mutex
	schemaPath  string
	pendingPath string
	schema      []CanonicalLabel
}

func NewCanonicalLabelService(learnDir string) *CanonicalLabelService {
	dictDir := filepath.Join(learnDir, "dictionaries")
	svc := &CanonicalLabelService{
		schemaPath:  filepath.Join(dictDir, "canonical_label_schema.json"),
		pendingPath: filepath.Join(learnDir, "pending", "pending_label_candidate.json"),
	}
	_ = svc.loadSchema()
	return svc
}

// MapLabel maps an LLM free-form description to the closest canonical label.
// If no mapping can be determined, it creates a pending_label_candidate.
// It does NOT write free keys directly to the schema.
func (s *CanonicalLabelService) MapLabel(llmDescription, sourceRegionID string, confidence float64) (CanonicalLabel, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Simple keyword-based mapping (production would use embeddings).
	label := matchLabel(llmDescription)
	if label.ElementType != ElementUnknown {
		return label, true
	}

	// Cannot map — create pending candidate instead.
	candidate := LabelCandidate{
		ID:             fmt.Sprintf("lc-%d", time.Now().UnixNano()),
		ProposedLabel:  CanonicalLabel{ElementType: ElementUnknown, ActionSemantic: ActionUnknown},
		SourceRegionID: sourceRegionID,
		LLMDescription: llmDescription,
		Confidence:     confidence,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}
	_ = s.appendPendingLocked(candidate)
	return CanonicalLabel{ElementType: ElementUnknown, ActionSemantic: ActionUnknown}, false
}

// GetSchema returns the current canonical label schema.
func (s *CanonicalLabelService) GetSchema() []CanonicalLabel {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]CanonicalLabel(nil), s.schema...)
}

// matchLabel performs keyword-based canonical mapping.
func matchLabel(desc string) CanonicalLabel {
	lower := toLower(desc)
	switch {
	case contains(lower, "button") && contains(lower, "submit"):
		return CanonicalLabel{ElementType: ElementButton, ActionSemantic: ActionSubmit}
	case contains(lower, "button") && contains(lower, "cancel"):
		return CanonicalLabel{ElementType: ElementButton, ActionSemantic: ActionCancel}
	case contains(lower, "link"):
		return CanonicalLabel{ElementType: ElementLink, ActionSemantic: ActionNavigate}
	case contains(lower, "checkbox"):
		return CanonicalLabel{ElementType: ElementCheckbox, ActionSemantic: ActionToggle}
	case contains(lower, "dropdown") || contains(lower, "select"):
		return CanonicalLabel{ElementType: ElementDropdown, ActionSemantic: ActionExpand}
	case contains(lower, "modal"):
		return CanonicalLabel{ElementType: ElementModal, ActionSemantic: ActionClose}
	case contains(lower, "input") || contains(lower, "text field"):
		return CanonicalLabel{ElementType: ElementInput, ActionSemantic: ActionRead}
	default:
		return CanonicalLabel{ElementType: ElementUnknown, ActionSemantic: ActionUnknown}
	}
}

func (s *CanonicalLabelService) loadSchema() error {
	data, err := os.ReadFile(s.schemaPath)
	if os.IsNotExist(err) {
		s.schema = defaultSchema()
		return s.saveSchemaLocked()
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.schema)
}

func (s *CanonicalLabelService) saveSchemaLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.schemaPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.schema, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.schemaPath, data, 0o600)
}

func (s *CanonicalLabelService) appendPendingLocked(c LabelCandidate) error {
	if err := os.MkdirAll(filepath.Dir(s.pendingPath), 0o700); err != nil {
		return err
	}
	var existing []LabelCandidate
	if data, err := os.ReadFile(s.pendingPath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	existing = append(existing, c)
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.pendingPath, data, 0o600)
}

func defaultSchema() []CanonicalLabel {
	return []CanonicalLabel{
		{ElementType: ElementButton, ActionSemantic: ActionSubmit, Description: "Submit / confirm button"},
		{ElementType: ElementButton, ActionSemantic: ActionCancel, Description: "Cancel / dismiss button"},
		{ElementType: ElementLink, ActionSemantic: ActionNavigate, Description: "Navigation link"},
		{ElementType: ElementCheckbox, ActionSemantic: ActionToggle, Description: "Toggle checkbox"},
		{ElementType: ElementDropdown, ActionSemantic: ActionExpand, Description: "Expandable dropdown"},
		{ElementType: ElementModal, ActionSemantic: ActionClose, Description: "Modal close trigger"},
		{ElementType: ElementInput, ActionSemantic: ActionRead, Description: "Text input field"},
	}
}

// --- helpers (no external deps) ---

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
