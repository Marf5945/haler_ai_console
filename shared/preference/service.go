// Package preference implements the v3.3.2 P0.2 drag-position preference
// resolution contract.
//
// Priority order (highest → lowest):
//   explicit sub preference > current workflow main preference >
//   global user preference > routing score
//
// 此套件只管「排序 / rank」——工具可用性（Available / NeedsReauth）
// 由 adapter_registry / tools 套件統一管理。
package preference

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"ui_console/data/storage"
)

// PreferenceRank is the position in the toolbar (0 = leftmost = highest rank).
type PreferenceRank int

// PreferenceScope defines how broadly a preference applies.
type PreferenceScope string

const (
	ScopeSub      PreferenceScope = "sub"      // sub-workflow explicit
	ScopeMain     PreferenceScope = "main"      // current workflow / DAG revision
	ScopeGlobal   PreferenceScope = "global"    // global user preference
	ScopeRouting  PreferenceScope = "routing"   // routing score (automatic)
)

// ToolPreferenceEntry records a user's preferred position for one tool.
type ToolPreferenceEntry struct {
	ToolID      string          `json:"tool_id"`
	Rank        PreferenceRank  `json:"rank"`
	Scope       PreferenceScope `json:"scope"`
	DAGRevision string          `json:"dag_revision,omitempty"` // bound for ScopeMain
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ToolVisibilityState 是 preference 層回傳的工具排序狀態。
// 只包含排序資訊——工具可用性（Available / NeedsReauth）
// 由 adapter_registry / tools 套件管理，不在此重複。
type ToolVisibilityState struct {
	ToolID string         `json:"tool_id"`
	Rank   PreferenceRank `json:"rank"`
}

// PreferenceStore persists the ordered preference list.
type PreferenceStore struct {
	mu      sync.Mutex
	store   *storage.JSONStore[[]ToolPreferenceEntry]
	entries []ToolPreferenceEntry
}

// NewPreferenceStore creates (or loads) a preference store.
func NewPreferenceStore(dataRoot string) *PreferenceStore {
	ps := &PreferenceStore{
		store: storage.NewJSONStore[[]ToolPreferenceEntry](
			filepath.Join(dataRoot, "data", "preferences", "tool_preference.json"),
		),
	}
	if loaded, err := ps.store.Load(); err == nil && loaded != nil {
		ps.entries = loaded
	}
	return ps
}

// Resolve returns the effective rank for a tool given its DAG revision context.
// Priority: sub > main (matching dag_revision) > global > routing score.
func (ps *PreferenceStore) Resolve(toolID, dagRevision string, routingScore float64) PreferenceRank {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// 1. Explicit sub preference
	for _, e := range ps.entries {
		if e.ToolID == toolID && e.Scope == ScopeSub {
			return e.Rank
		}
	}
	// 2. Main preference bound to this DAG revision
	for _, e := range ps.entries {
		if e.ToolID == toolID && e.Scope == ScopeMain && e.DAGRevision == dagRevision {
			return e.Rank
		}
	}
	// 3. Global preference
	for _, e := range ps.entries {
		if e.ToolID == toolID && e.Scope == ScopeGlobal {
			return e.Rank
		}
	}
	// 4. Routing score as fallback rank (lower score → higher rank number / lower priority)
	return PreferenceRank(int(1000 - routingScore*10))
}

// SetPreference records a user's explicit rank change for a tool.
// Changing scope from ScopeMain to a broader scope requires user confirmation
// (enforced by the UI — this method accepts the confirmed result).
func (ps *PreferenceStore) SetPreference(entry ToolPreferenceEntry) error {
	if entry.ToolID == "" {
		return fmt.Errorf("preference: tool_id is required")
	}
	if entry.Scope == ScopeMain && entry.DAGRevision == "" {
		return fmt.Errorf("preference: main scope requires dag_revision")
	}
	entry.UpdatedAt = time.Now()

	ps.mu.Lock()
	defer ps.mu.Unlock()

	for i, e := range ps.entries {
		if e.ToolID == entry.ToolID && e.Scope == entry.Scope && e.DAGRevision == entry.DAGRevision {
			ps.entries[i] = entry
			return ps.store.SaveRaw(ps.entries)
		}
	}
	ps.entries = append(ps.entries, entry)
	return ps.store.SaveRaw(ps.entries)
}

// BuildVisibilityList 回傳所有指定工具的排序狀態。
// 只處理 rank（排序）——工具可用性由 adapter_registry / tools 套件管理。
func (ps *PreferenceStore) BuildVisibilityList(toolIDs []string, dagRevision string) []ToolVisibilityState {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	result := make([]ToolVisibilityState, 0, len(toolIDs))
	for _, id := range toolIDs {
		rank := ps.resolveLocked(id, dagRevision)
		result = append(result, ToolVisibilityState{
			ToolID: id,
			Rank:   rank,
		})
	}
	return result
}

func (ps *PreferenceStore) resolveLocked(toolID, dagRevision string) PreferenceRank {
	for _, e := range ps.entries {
		if e.ToolID == toolID && e.Scope == ScopeSub {
			return e.Rank
		}
	}
	for _, e := range ps.entries {
		if e.ToolID == toolID && e.Scope == ScopeMain && e.DAGRevision == dagRevision {
			return e.Rank
		}
	}
	for _, e := range ps.entries {
		if e.ToolID == toolID && e.Scope == ScopeGlobal {
			return e.Rank
		}
	}
	return PreferenceRank(999) // unranked
}

// persistence 由 storage.JSONStore 處理。
