package blackboard

// archive.go — 歸檔（spec §13）與真刪除 redaction（spec §8）。
// 兩者都需要重寫共享文件，因此僅支援實作 RewritableStore 的載體。
// canonical_seq 已由 seq index 凍結，搬動文件不影響既有順序（spec §3）。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RewritableStore is an optional Store capability: replacing the whole log
// document (archival, redaction). Append-only participants never need it.
type RewritableStore interface {
	Store
	ReplaceLog(ctx context.Context, content string) error
}

// ReplaceLog implements RewritableStore for FileStore (lock + atomic write).
func (s *FileStore) ReplaceLog(ctx context.Context, content string) error {
	unlock, err := s.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	return atomicWrite(s.Path, []byte(content))
}

// ArchiveConfig sets the archival thresholds (spec §13).
type ArchiveConfig struct {
	MaxEvents  int // trigger: more events than this
	MaxBytes   int // trigger: log content larger than this
	KeepEvents int // events retained in the main document
}

// DefaultArchiveConfig returns the spec §13 defaults: 500 / 300KB / keep 200.
func DefaultArchiveConfig() ArchiveConfig {
	return ArchiveConfig{MaxEvents: 500, MaxBytes: 300 * 1024, KeepEvents: 200}
}

// ArchiveResult describes one executed archival.
type ArchiveResult struct {
	ArchivePath   string `json:"archive_path"`
	ArchivedCount int    `json:"archived_count"`
	MarkerID      string `json:"marker_id"`
}

// reFence re-renders a parsed event block verbatim from its raw JSON so the
// content hash (and any signature) survives the move.
func reFence(pe ParsedEvent) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### BBM %s kind:%s\n", pe.Event.ID, pe.Event.Kind)
	b.WriteString("```json\n")
	b.Write(pe.RawJSON)
	b.WriteString("\n```\n\n")
	return b.String()
}

// ArchiveIfNeeded checks the thresholds and, when exceeded, moves the oldest
// events into an archive file, leaves an archive_marker, and tombstones the
// moved ids so integrity checks stay silent. markerActor must be a human or
// coordinator (permission table). Returns nil when no archival was needed.
func (s *Synchronizer) ArchiveIfNeeded(ctx context.Context, cfg ArchiveConfig, markerActor Actor, archiveDir string, now time.Time) (*ArchiveResult, error) {
	rw, ok := s.Store.(RewritableStore)
	if !ok {
		return nil, fmt.Errorf("store does not support rewriting; cannot archive")
	}
	if cfg.MaxEvents <= 0 || cfg.KeepEvents < 0 {
		return nil, fmt.Errorf("invalid archive config")
	}
	content, _, err := s.Store.ReadLog(ctx)
	if err != nil {
		return nil, err
	}
	res := ParseLog(content)
	if len(res.Events) <= cfg.MaxEvents && (cfg.MaxBytes <= 0 || len(content) <= cfg.MaxBytes) {
		return nil, nil
	}
	if len(res.Events) <= cfg.KeepEvents {
		return nil, nil // oversized but nothing old enough to move
	}

	cut := res.Events[:len(res.Events)-cfg.KeepEvents]
	kept := res.Events[len(res.Events)-cfg.KeepEvents:]

	// Write the archive document first (never lose data).
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, err
	}
	archivePath := filepath.Join(archiveDir,
		fmt.Sprintf("blackboard_archive_%s.md", now.UTC().Format("20060102_150405")))
	var ab strings.Builder
	fmt.Fprintf(&ab, "# Blackboard Archive %s\n\n", now.UTC().Format("2006-01-02"))
	ab.WriteString("<!-- read-only archive; do not append here -->\n\n")
	for _, pe := range cut {
		ab.WriteString(reFence(pe))
	}
	if err := atomicWrite(archivePath, []byte(ab.String())); err != nil {
		return nil, err
	}

	markerID, err := NewEventID(now)
	if err != nil {
		return nil, err
	}
	marker := Event{
		V: SchemaVersion, ID: markerID, Kind: KindArchiveMarker,
		Actor: markerActor, CreatedAt: now.UTC().Format(time.RFC3339),
		Summary:    fmt.Sprintf("archived %d events", len(cut)),
		Range:      &EventRange{From: cut[0].Event.ID, To: cut[len(cut)-1].Event.ID},
		ArchiveRef: &Ref{Type: "file", Path: archivePath},
	}
	markerBlock, err := FormatEvent(marker)
	if err != nil {
		return nil, err
	}

	var nb strings.Builder
	nb.WriteString("# Blackboard Exchange Log\n\n## Event Log\n\n")
	nb.WriteString(markerBlock)
	for _, pe := range kept {
		nb.WriteString(reFence(pe))
	}
	nb.WriteString(Sentinel + "\n")
	if err := rw.ReplaceLog(ctx, nb.String()); err != nil {
		return nil, err
	}

	ids := make([]string, len(cut))
	for i, pe := range cut {
		ids[i] = pe.Event.ID
	}
	if err := s.Mirror.AddTombstones(ids...); err != nil {
		return nil, err
	}
	return &ArchiveResult{ArchivePath: archivePath, ArchivedCount: len(cut), MarkerID: markerID}, nil
}

// RemoveEventFromLog cuts one event block out of the live log region.
// Returns the new content and whether the event was found.
func RemoveEventFromLog(content, id string) (string, bool) {
	live := len(content)
	if idx := strings.Index(content, Sentinel); idx >= 0 {
		live = idx
	}
	heading := "### BBM " + id + " "
	start := strings.Index(content[:live], heading)
	if start < 0 {
		return content, false
	}
	next := strings.Index(content[start+1:live], "### BBM ")
	end := live
	if next >= 0 {
		end = start + 1 + next
	}
	return content[:start] + content[end:], true
}

// Redact performs a host-approved real deletion (spec §8): mirror backup via
// a fresh sync, remove the block, tombstone the id, then append a
// redaction_applied event (which never contains the sensitive original).
// The approver must be a human or coordinator.
func (s *Synchronizer) Redact(ctx context.Context, targetID, reason string, approver Actor, now time.Time) (string, error) {
	if approver.Type != ActorHuman && approver.Type != ActorCoordinator {
		return "", fmt.Errorf("redaction requires human or coordinator approval")
	}
	rw, ok := s.Store.(RewritableStore)
	if !ok {
		return "", fmt.Errorf("store does not support rewriting; cannot redact")
	}
	// Backup before deletion (spec §8): a sync mirrors every parsed event.
	if _, err := s.Sync(ctx); err != nil {
		return "", fmt.Errorf("pre-redaction backup sync: %w", err)
	}
	content, _, err := s.Store.ReadLog(ctx)
	if err != nil {
		return "", err
	}
	updated, found := RemoveEventFromLog(content, targetID)
	if !found {
		return "", fmt.Errorf("event %s not found in live log (already archived or removed?)", targetID)
	}
	if err := rw.ReplaceLog(ctx, updated); err != nil {
		return "", err
	}
	if err := s.Mirror.AddTombstones(targetID); err != nil {
		return "", err
	}

	redactionID, err := NewEventID(now)
	if err != nil {
		return "", err
	}
	// Invariant (spec §8 / §35.8): the receipt legitimately keeps target_id as
	// the audit case number — the redacted event ID may lawfully remain in the
	// log. Only the sensitive *content* must disappear (the block is removed
	// above); the receipt itself never carries the original body.
	ev := Event{
		V: SchemaVersion, ID: redactionID, Kind: KindRedactionApplied,
		Actor: approver, CreatedAt: now.UTC().Format(time.RFC3339),
		TargetID: targetID, Reason: reason,
	}
	if err := s.Store.AppendEvent(ctx, ev); err != nil {
		return "", fmt.Errorf("append redaction_applied: %w", err)
	}
	return redactionID, nil
}
