package execution_hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CandidateStatus tracks the review state of a subagent candidate.
type CandidateStatus string

const (
	CandidatePending  CandidateStatus = "pending"
	CandidateReviewed CandidateStatus = "reviewed"
	CandidateRejected CandidateStatus = "rejected"
)

// SubagentCandidate represents a proposed new subagent derived from hook evidence.
// Corresponds to schema #51 in TASKS_1_2.md.
//
// SAFETY RULE: A candidate ONLY creates files under new_subagent_candidates/.
// It NEVER replaces, disables, or deletes an existing subagent.
type SubagentCandidate struct {
	ID                string          `json:"id"`
	SourceHookRunID   string          `json:"source_hook_run_id"`
	Name              string          `json:"name"`
	Summary           string          `json:"summary"`
	CandidateJSONPath string          `json:"candidate_json_path"`
	CandidateMDPath   string          `json:"candidate_md_path"`
	Status            CandidateStatus `json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
}

// CandidateService manages new_subagent_candidate creation.
type CandidateService struct {
	hookDir string
}

func NewCandidateService(hookDir string) *CandidateService {
	return &CandidateService{hookDir: hookDir}
}

// CreateCandidate writes a new candidate JSON + MD file pair.
// It only creates files — it never modifies, disables, or removes existing subagents.
func (cs *CandidateService) CreateCandidate(hookRunID, name, summary string, detail map[string]interface{}) (*SubagentCandidate, error) {
	candidateDir := filepath.Join(cs.hookDir, "new_subagent_candidates")
	if err := os.MkdirAll(candidateDir, 0o700); err != nil {
		return nil, err
	}

	id := fmt.Sprintf("candidate_%d", time.Now().UnixNano())
	jsonPath := filepath.Join(candidateDir, id+".json")
	mdPath := filepath.Join(candidateDir, id+".md")

	candidate := &SubagentCandidate{
		ID:                id,
		SourceHookRunID:   hookRunID,
		Name:              name,
		Summary:           summary,
		CandidateJSONPath: jsonPath,
		CandidateMDPath:   mdPath,
		Status:            CandidatePending,
		CreatedAt:         time.Now(),
	}

	// Write JSON detail file.
	detail["candidate_meta"] = candidate
	jsonData, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(jsonPath, jsonData, 0o600); err != nil {
		return nil, err
	}

	// Write human-readable MD file.
	mdContent := fmt.Sprintf("# Subagent Candidate: %s\n\n"+
		"**ID:** %s  \n"+
		"**Source Hook Run:** %s  \n"+
		"**Created:** %s  \n"+
		"**Status:** %s  \n\n"+
		"## Summary\n\n%s\n\n"+
		"> This is a CANDIDATE only. It requires review before use.\n"+
		"> It does NOT replace or disable any existing subagent.\n",
		name, id, hookRunID, candidate.CreatedAt.Format(time.RFC3339), candidate.Status, summary)
	if err := os.WriteFile(mdPath, []byte(mdContent), 0o600); err != nil {
		return nil, err
	}

	return candidate, nil
}

// ListCandidates returns all candidates currently stored in the candidates dir.
func (cs *CandidateService) ListCandidates() ([]SubagentCandidate, error) {
	candidateDir := filepath.Join(cs.hookDir, "new_subagent_candidates")
	entries, err := os.ReadDir(candidateDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var candidates []SubagentCandidate
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(candidateDir, e.Name()))
		if err != nil {
			continue
		}
		var wrapper struct {
			CandidateMeta SubagentCandidate `json:"candidate_meta"`
		}
		if err := json.Unmarshal(data, &wrapper); err == nil {
			candidates = append(candidates, wrapper.CandidateMeta)
		}
	}
	return candidates, nil
}
