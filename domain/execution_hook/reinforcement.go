package execution_hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TagPatchStatus tracks the review state of a patch proposal.
type TagPatchStatus string

const (
	TagPatchPending  TagPatchStatus = "pending"
	TagPatchPromoted TagPatchStatus = "promoted"
	TagPatchRejected TagPatchStatus = "rejected"
)

// TagPatch represents a proposed update to learned_tags or preference_score.
// Corresponds to schema #51 in TASKS_1_2.md.
// RULE: low-risk patches may be auto-promoted after run completion.
//
//	medium/high/critical patches must go through pending_review.
type TagPatch struct {
	ID                   string         `json:"id"`
	TargetID             string         `json:"target_id"`
	LearnedTags          []string       `json:"learned_tags"`
	PreferenceScoreDelta float64        `json:"preference_score_delta"`
	RiskLevel            RiskLevel      `json:"risk_level"`
	Status               TagPatchStatus `json:"status"`
	CreatedAt            time.Time      `json:"created_at"`
	PromotedAt           *time.Time     `json:"promoted_at,omitempty"`
}

// RegistryPatchProposal represents a proposed schema change for a tool.
// Corresponds to schema #51 in TASKS_1_2.md.
// These are PROPOSALS only — they must never automatically update tool_registry.json.
type RegistryPatchProposal struct {
	ID                  string         `json:"id"`
	ToolID              string         `json:"tool_id"`
	ProposedSchemaDelta string         `json:"proposed_schema_delta"`
	EvidenceHash        string         `json:"evidence_hash"`
	Status              TagPatchStatus `json:"status"`
	CreatedAt           time.Time      `json:"created_at"`
}

// ReinforcementService handles tag patch promotion after a run completes.
// It MUST NOT be called during step execution — only after subagent completes.
type ReinforcementService struct {
	hookDir string
	chain   *HashChain
}

func NewReinforcementService(hookDir string) *ReinforcementService {
	return &ReinforcementService{
		hookDir: hookDir,
		chain:   NewHashChain(hookDir),
	}
}

// PromoteLowRisk auto-promotes patches with risk_level == low.
// All other risk levels go to pending_review and must not be auto-promoted.
// Every promotion (regardless of auto or manual) is written to the hash chain.
func (r *ReinforcementService) PromoteLowRisk(patches []TagPatch) ([]TagPatch, error) {
	promoted := make([]TagPatch, 0, len(patches))
	pending := make([]TagPatch, 0, len(patches))

	for _, p := range patches {
		if p.RiskLevel == RiskLow {
			now := time.Now()
			p.Status = TagPatchPromoted
			p.PromotedAt = &now
			// Write to hash chain — mandatory for every auto-promotion.
			entry := ChainEntry{
				Type:      "tag_patch_promoted",
				Payload:   fmt.Sprintf("patch=%s target=%s tags=%v delta=%.4f", p.ID, p.TargetID, p.LearnedTags, p.PreferenceScoreDelta),
				CreatedAt: now,
			}
			if err := r.chain.Append(entry); err != nil {
				return nil, fmt.Errorf("hash chain write failed for patch %s: %w", p.ID, err)
			}
			promoted = append(promoted, p)
		} else {
			p.Status = TagPatchPending
			pending = append(pending, p)
		}
	}

	if err := r.savePendingReview(pending); err != nil {
		return nil, err
	}
	return promoted, nil
}

func (r *ReinforcementService) savePendingReview(patches []TagPatch) error {
	if len(patches) == 0 {
		return nil
	}
	if err := os.MkdirAll(r.hookDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(patches, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.hookDir, "pending_tag_patch.json"), data, 0o600)
}

// GetPendingTagPatches reads the current pending patches from disk.
func (r *ReinforcementService) GetPendingTagPatches() ([]TagPatch, error) {
	path := filepath.Join(r.hookDir, "pending_tag_patch.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var patches []TagPatch
	if err := json.Unmarshal(data, &patches); err != nil {
		return nil, err
	}
	return patches, nil
}

// SaveRegistryProposals writes tool registry patch proposals to disk.
// These are read-only proposals — they do NOT touch tool_registry.json.
func (r *ReinforcementService) SaveRegistryProposals(proposals []RegistryPatchProposal) error {
	if err := os.MkdirAll(r.hookDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(proposals, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.hookDir, "tool_registry_patch_proposal.json"), data, 0o600)
}

// GetRegistryProposals reads tool registry patch proposals from the canonical
// execution hook storage file. A missing file means no proposals have been
// generated yet, so callers receive an empty list instead of an error.
func (r *ReinforcementService) GetRegistryProposals() ([]RegistryPatchProposal, error) {
	path := filepath.Join(r.hookDir, "tool_registry_patch_proposal.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []RegistryPatchProposal{}, nil
	}
	if err != nil {
		return nil, err
	}
	var proposals []RegistryPatchProposal
	if err := json.Unmarshal(data, &proposals); err != nil {
		return nil, err
	}
	return proposals, nil
}
