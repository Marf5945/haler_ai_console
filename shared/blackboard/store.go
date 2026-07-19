package blackboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChangeSignal wakes the synchronizer: it means "the log may have changed",
// never "here is the data" (spec: watchers are wake-up calls, not streams).
type ChangeSignal struct {
	Revision string
	At       time.Time
}

// Watcher is the pluggable change-detection interface. Implementations:
// PollingWatcher (any Store), and later push-based ones (Drive watch, ...).
type Watcher interface {
	Start(ctx context.Context) (<-chan ChangeSignal, error)
	Stop()
}

// Store abstracts the shared exchange document carrier. The engine never
// assumes a particular backend; local folders, synced folders, Google Docs,
// WebDAV or a self-hosted server are all adapters over this interface.
type Store interface {
	// ReadLog returns the full exchange document and an opaque revision tag.
	ReadLog(ctx context.Context) (content string, revision string, err error)
	// AppendEvent appends an event before the sentinel and verifies the
	// write by reading back (spec §14). It must never modify old events.
	AppendEvent(ctx context.Context, ev Event) error
	// Revision returns the current opaque revision tag cheaply.
	Revision(ctx context.Context) (string, error)
}

// ---------------------------------------------------------------------------
// FileStore: exchange document as a Markdown file in any folder. Point the
// path into a synced folder (Drive/Dropbox/Syncthing/self-hosted mount) and
// the same file becomes a cloud blackboard with zero engine changes.
// ---------------------------------------------------------------------------

// FileStore implements Store over a single Markdown file.
type FileStore struct {
	Path string
	// LockTimeout bounds how long AppendEvent waits for the sidecar lock.
	LockTimeout time.Duration
}

// NewFileStore returns a FileStore with sane defaults.
func NewFileStore(path string) *FileStore {
	return &FileStore{Path: path, LockTimeout: 5 * time.Second}
}

// initialContent is written when the exchange file does not exist yet.
const initialContent = "# Blackboard Exchange Log\n\n## Event Log\n\n" + Sentinel + "\n"

// ReadLog implements Store.
func (s *FileStore) ReadLog(ctx context.Context) (string, string, error) {
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return initialContent, revisionOf(initialContent), nil
	}
	if err != nil {
		return "", "", err
	}
	c := string(b)
	return c, revisionOf(c), nil
}

// Revision implements Store.
func (s *FileStore) Revision(ctx context.Context) (string, error) {
	_, rev, err := s.ReadLog(ctx)
	return rev, err
}

// AppendEvent implements Store: lock, re-read, insert before sentinel,
// atomic replace, read back and verify id + hash (spec §14).
func (s *FileStore) AppendEvent(ctx context.Context, ev Event) error {
	block, err := FormatEvent(ev)
	if err != nil {
		return fmt.Errorf("format event: %w", err)
	}
	// The exchange file may point into a not-yet-created local or sync folder.
	// Create only its direct parent here; FileStore needs no separate setup layer.
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("create exchange directory: %w", err)
	}
	unlock, err := s.acquireLock(ctx)
	if err != nil {
		return err
	}
	defer unlock()

	content, _, err := s.ReadLog(ctx)
	if err != nil {
		return err
	}
	idx := strings.LastIndex(content, Sentinel)
	if idx < 0 {
		// Sentinel was hand-edited away: recoverable, append at end.
		content = strings.TrimRight(content, "\n") + "\n\n"
		idx = len(content)
		content += Sentinel + "\n"
	}
	updated := content[:idx] + block + content[idx:]

	if err := atomicWrite(s.Path, []byte(updated)); err != nil {
		return err
	}

	// Read-back verification.
	verify, _, err := s.ReadLog(ctx)
	if err != nil {
		return fmt.Errorf("readback: %w", err)
	}
	wantHash := HashJSON(mustRawOf(ev))
	for _, pe := range ParseLog(verify).Events {
		if pe.Event.ID == ev.ID {
			if pe.Hash != wantHash {
				return fmt.Errorf("readback hash mismatch for %s", ev.ID)
			}
			return nil
		}
	}
	return fmt.Errorf("readback: event %s not found after append", ev.ID)
}

func mustRawOf(ev Event) []byte {
	// FormatEvent already validated; marshal cannot realistically fail here.
	// HashJSON compacts before hashing, so indentation differences are moot.
	b, _ := json.Marshal(ev)
	return b
}

func (s *FileStore) acquireLock(ctx context.Context) (func(), error) {
	lockPath := s.Path + ".lock"
	deadline := time.Now().Add(s.LockTimeout)
	if s.LockTimeout <= 0 {
		deadline = time.Now().Add(5 * time.Second)
	}
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintf(f, "pid=%d t=%s\n", os.Getpid(), time.Now().Format(time.RFC3339))
			f.Close()
			return func() { os.Remove(lockPath) }, nil
		}
		if time.Now().After(deadline) {
			// A stale lock older than 30s is treated as abandoned.
			if fi, statErr := os.Stat(lockPath); statErr == nil && time.Since(fi.ModTime()) > 30*time.Second {
				os.Remove(lockPath)
				continue
			}
			return nil, fmt.Errorf("could not acquire lock %s", lockPath)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".bbm_tmp_*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func revisionOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// PollingWatcher: works with any Store. Push-based watchers can replace it
// behind the same interface without touching the synchronizer.
// ---------------------------------------------------------------------------

// PollingWatcher polls Store.Revision and emits a ChangeSignal on change.
type PollingWatcher struct {
	Store    Store
	Interval time.Duration

	cancel context.CancelFunc
}

// NewPollingWatcher returns a watcher with the given poll interval.
func NewPollingWatcher(store Store, interval time.Duration) *PollingWatcher {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &PollingWatcher{Store: store, Interval: interval}
}

// Start implements Watcher. The returned channel closes when the context is
// cancelled or Stop is called.
func (w *PollingWatcher) Start(ctx context.Context) (<-chan ChangeSignal, error) {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	ch := make(chan ChangeSignal, 1)

	last, err := w.Store.Revision(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	go func() {
		defer close(ch)
		ticker := time.NewTicker(w.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rev, err := w.Store.Revision(ctx)
				if err != nil || rev == last {
					continue
				}
				last = rev
				select {
				case ch <- ChangeSignal{Revision: rev, At: time.Now()}:
				default: // a pending signal already wakes the synchronizer
				}
			}
		}
	}()
	return ch, nil
}

// Stop implements Watcher.
func (w *PollingWatcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}
