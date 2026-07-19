package blackboard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MirrorRecord is one line of blackboard_events.jsonl: the local, trusted
// copy of every event ever successfully parsed (spec §12).
type MirrorRecord struct {
	ID   string          `json:"id"`
	Seq  int64           `json:"seq"`
	Hash string          `json:"hash"`
	Raw  json.RawMessage `json:"raw"`
}

// Mirror persists the local event mirror, dead letters, and the canonical
// sequence index inside one directory.
type Mirror struct {
	Dir string
}

// NewMirror returns a Mirror rooted at dir (created lazily).
func NewMirror(dir string) *Mirror { return &Mirror{Dir: dir} }

func (m *Mirror) eventsPath() string     { return filepath.Join(m.Dir, "blackboard_events.jsonl") }
func (m *Mirror) deadLetterPath() string { return filepath.Join(m.Dir, "blackboard_dead_letter.jsonl") }
func (m *Mirror) seqIndexPath() string   { return filepath.Join(m.Dir, "blackboard_seq_index.json") }

// seqIndex maps event id -> canonical_seq. Once assigned, a seq never
// changes (spec §3): archival or document reshuffling cannot reorder
// history.
type seqIndex struct {
	Next int64            `json:"next"`
	IDs  map[string]int64 `json:"ids"`
}

func (m *Mirror) loadSeqIndex() (*seqIndex, error) {
	idx := &seqIndex{Next: 1, IDs: map[string]int64{}}
	b, err := os.ReadFile(m.seqIndexPath())
	if os.IsNotExist(err) {
		return idx, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, idx); err != nil {
		return nil, fmt.Errorf("corrupt seq index: %w", err)
	}
	if idx.IDs == nil {
		idx.IDs = map[string]int64{}
	}
	if idx.Next <= 0 {
		idx.Next = 1
	}
	return idx, nil
}

func (m *Mirror) saveSeqIndex(idx *seqIndex) error {
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(m.seqIndexPath(), b)
}

// AssignSeq gives every parsed event its permanent canonical_seq. Events
// already known keep their seq; new events are numbered in document order
// (admission order). The updated index is persisted before returning.
func (m *Mirror) AssignSeq(events []ParsedEvent) ([]SeqEvent, error) {
	idx, err := m.loadSeqIndex()
	if err != nil {
		return nil, err
	}
	out := make([]SeqEvent, 0, len(events))
	changed := false
	for _, pe := range events {
		seq, ok := idx.IDs[pe.Event.ID]
		if !ok {
			seq = idx.Next
			idx.Next++
			idx.IDs[pe.Event.ID] = seq
			changed = true
		}
		out = append(out, SeqEvent{ParsedEvent: pe, Seq: seq})
	}
	if changed {
		if err := m.saveSeqIndex(idx); err != nil {
			return nil, err
		}
	}
	// Replays must be ordered by canonical seq, not by this round's DocPos.
	sortSeqEvents(out)
	return out, nil
}

func sortSeqEvents(evs []SeqEvent) {
	// Insertion sort: rounds are small and mostly ordered already.
	for i := 1; i < len(evs); i++ {
		for j := i; j > 0 && evs[j].Seq < evs[j-1].Seq; j-- {
			evs[j], evs[j-1] = evs[j-1], evs[j]
		}
	}
}

// mirrorRecords loads the trusted first-seen record for each event ID. The
// hash is retained so a later sync cannot silently replace an old event while
// keeping its identity.
func (m *Mirror) mirrorRecords() (map[string]MirrorRecord, error) {
	records := map[string]MirrorRecord{}
	f, err := os.Open(m.eventsPath())
	if os.IsNotExist(err) {
		return records, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var rec MirrorRecord
		if json.Unmarshal(sc.Bytes(), &rec) == nil && rec.ID != "" {
			if _, exists := records[rec.ID]; !exists {
				records[rec.ID] = rec
			}
		}
	}
	return records, sc.Err()
}

// MirrorIDs returns the set of event ids present in the local mirror.
func (m *Mirror) MirrorIDs() (map[string]bool, error) {
	records, err := m.mirrorRecords()
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(records))
	for id := range records {
		ids[id] = true
	}
	return ids, nil
}

// ReconcileEvents mirrors new events and rejects cross-round content changes
// for an existing ID. Conflicts project the trusted first-seen mirror record,
// never the edited shared copy (spec §5 and §12).
func (m *Mirror) ReconcileEvents(events []SeqEvent) (accepted []SeqEvent, conflicts []DeadLetter, written int, err error) {
	known, err := m.mirrorRecords()
	if err != nil {
		return nil, nil, 0, err
	}
	var fresh []SeqEvent
	for _, se := range events {
		record, exists := known[se.Event.ID]
		if !exists {
			accepted = append(accepted, se)
			fresh = append(fresh, se)
			continue
		}
		if record.Hash == se.Hash {
			if record.Seq > 0 {
				se.Seq = record.Seq
			}
			accepted = append(accepted, se)
			continue
		}

		// Keep the mirror as the projection source. The edited shared payload is
		// preserved only in the local dead letter for diagnosis.
		var original Event
		if err := json.Unmarshal(record.Raw, &original); err != nil {
			return nil, nil, 0, fmt.Errorf("corrupt mirror event %s: %w", record.ID, err)
		}
		original.Raw = append(json.RawMessage(nil), record.Raw...)
		seq := record.Seq
		if seq <= 0 {
			seq = se.Seq
		}
		accepted = append(accepted, SeqEvent{ParsedEvent: ParsedEvent{
			Event: original, DocPos: se.DocPos, Line: se.Line,
			Hash: record.Hash, RawJSON: append([]byte(nil), record.Raw...),
		}, Seq: seq})
		conflicts = append(conflicts, DeadLetter{
			Line: se.Line, EventID: se.Event.ID, Reason: ReasonConflict,
			Detail:  "same id changed hash after it was mirrored; trusted first-seen content retained",
			RawText: truncate(string(se.RawJSON), 500),
		})
	}
	if len(fresh) == 0 {
		return accepted, conflicts, 0, nil
	}
	if err := os.MkdirAll(m.Dir, 0o755); err != nil {
		return nil, nil, 0, err
	}
	f, err := os.OpenFile(m.eventsPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, 0, err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, se := range fresh {
		rec := MirrorRecord{ID: se.Event.ID, Seq: se.Seq, Hash: se.Hash, Raw: se.RawJSON}
		b, err := json.Marshal(rec)
		if err != nil {
			return nil, nil, 0, err
		}
		if _, err := w.Write(b); err != nil {
			return nil, nil, 0, err
		}
		if err := w.WriteByte('\n'); err != nil {
			return nil, nil, 0, err
		}
	}
	if err := w.Flush(); err != nil {
		return nil, nil, 0, err
	}
	return accepted, conflicts, len(fresh), nil
}

// AppendDeadLetters quarantines unparseable entries locally. Dead letters
// never go back into the shared log (spec §5).
func (m *Mirror) AppendDeadLetters(dls []DeadLetter) error {
	if len(dls) == 0 {
		return nil
	}
	if err := os.MkdirAll(m.Dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(m.deadLetterPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, dl := range dls {
		b, err := json.Marshal(dl)
		if err != nil {
			return err
		}
		w.Write(b)
		w.WriteByte('\n')
	}
	return w.Flush()
}

// IntegrityCheck compares the shared log against the mirror: ids present
// locally but missing from the document indicate hand-deletion or sync
// loss. Never auto-restore — surface a warning instead (spec §12).
func (m *Mirror) IntegrityCheck(currentIDs map[string]bool) ([]string, error) {
	known, err := m.MirrorIDs()
	if err != nil {
		return nil, err
	}
	var missing []string
	for id := range known {
		if !currentIDs[id] {
			missing = append(missing, id)
		}
	}
	sortStrings(missing)
	return missing, nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
