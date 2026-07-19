package blackboard

import (
	"context"
)

// SyncResult is one synchronization round's full outcome.
type SyncResult struct {
	State       MeetingState
	Parse       ParseResult
	NewMirrored int
	MissingIDs  []string // in mirror but absent from shared log (spec §12)
}

// Synchronizer ties Store + parser + mirror + projector together. It is
// carrier-agnostic: swap the Store (local file, synced folder, cloud doc)
// and nothing else changes.
type Synchronizer struct {
	Store  Store
	Mirror *Mirror
	// Trusted adjudicates claimed identities (spec §6.2). nil trusts all —
	// acceptable only for single-machine or fully-trusted transports.
	Trusted TrustFunc
}

// Sync performs one full round: read → parse → assign canonical seq →
// mirror → integrity check → project. Safe to call repeatedly; replaying
// the same log yields a byte-identical Meeting State (spec §16).
func (s *Synchronizer) Sync(ctx context.Context) (*SyncResult, error) {
	content, _, err := s.Store.ReadLog(ctx)
	if err != nil {
		return nil, err
	}
	res := ParseLog(content)

	seqEvents, err := s.Mirror.AssignSeq(res.Events)
	if err != nil {
		return nil, err
	}
	seqEvents, conflicts, written, err := s.Mirror.ReconcileEvents(seqEvents)
	if err != nil {
		return nil, err
	}
	res.DeadLetters = append(res.DeadLetters, conflicts...)
	if err := s.Mirror.AppendDeadLetters(res.DeadLetters); err != nil {
		return nil, err
	}

	current := make(map[string]bool, len(seqEvents))
	for _, se := range seqEvents {
		current[se.Event.ID] = true
	}
	missing, err := s.Mirror.IntegrityCheck(current)
	if err != nil {
		return nil, err
	}
	// Intentionally removed events (archive, redaction) are not incidents.
	if tombs, terr := s.Mirror.LoadTombstones(); terr != nil {
		return nil, terr
	} else if len(tombs) > 0 {
		kept := missing[:0]
		for _, id := range missing {
			if !tombs[id] {
				kept = append(kept, id)
			}
		}
		missing = kept
	}

	state := Project(seqEvents, s.Trusted, len(res.DeadLetters))
	return &SyncResult{
		State:       state,
		Parse:       res,
		NewMirrored: written,
		MissingIDs:  missing,
	}, nil
}
