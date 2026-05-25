package audit_log

import "os"

// ──────────────────────────────────────────────
// 內部設定
// ──────────────────────────────────────────────

// config holds all configurable behavior for an AppendLog.
type config[T any] struct {
	filePerm         os.FileMode
	dirPerm          os.FileMode
	skipCorruptLines bool
	beforeAppend     func(*T) error
	hashChain        *hashChainHooks[T]
	chainState       *ChainState
}

func defaultConfig[T any]() config[T] {
	return config[T]{
		filePerm: 0o600,
		dirPerm:  0o700,
	}
}

// ──────────────────────────────────────────────
// Option 函數
// ──────────────────────────────────────────────

// Option configures an AppendLog.
type Option[T any] func(*config[T])

// WithFilePermission sets the file permission for the log file.
// Default: 0600.
func WithFilePermission[T any](perm os.FileMode) Option[T] {
	return func(c *config[T]) {
		c.filePerm = perm
	}
}

// WithDirPermission sets the directory permission for MkdirAll.
// Default: 0700.
func WithDirPermission[T any](perm os.FileMode) Option[T] {
	return func(c *config[T]) {
		c.dirPerm = perm
	}
}

// WithSkipCorruptLines causes ReadAll to skip lines that fail JSON parsing
// instead of returning an error.
func WithSkipCorruptLines[T any]() Option[T] {
	return func(c *config[T]) {
		c.skipCorruptLines = true
	}
}

// WithBeforeAppend registers a hook that runs before each Append.
// The hook receives a pointer to the entry and may mutate it (e.g. set timestamps,
// auto-generate IDs) or return an error to reject the entry (e.g. invariant violation).
func WithBeforeAppend[T any](fn func(*T) error) Option[T] {
	return func(c *config[T]) {
		c.beforeAppend = fn
	}
}

// ──────────────────────────────────────────────
// Hash Chain 支援
// ──────────────────────────────────────────────

// ChainState holds mutable state for hash chain linking.
// Consumers initialise it with their genesis hash.
type ChainState struct {
	LastHash string
	Length   int
}

// hashChainHooks contains the three callbacks that implement hash chain linking.
type hashChainHooks[T any] struct {
	// beforeWrite sets PreviousHash and computes Hash on the entry before write.
	beforeWrite func(entry *T, state *ChainState)
	// afterWrite updates ChainState after successful write (e.g. advance lastHash).
	afterWrite func(entry *T, state *ChainState)
	// loadTail extracts hash + length from the last persisted entry.
	loadTail func(entry *T, state *ChainState)
}

// WithHashChain enables hash chain linking.
//
// Parameters:
//   - genesisHash: the initial "previous hash" for the first entry.
//   - beforeWrite: called before marshal; should set previous_hash and compute hash on entry.
//   - afterWrite: called after successful write; should update state.LastHash.
//   - loadTail: called during New() to recover state from the last entry on disk.
func WithHashChain[T any](
	genesisHash string,
	beforeWrite func(entry *T, state *ChainState),
	afterWrite func(entry *T, state *ChainState),
	loadTail func(entry *T, state *ChainState),
) Option[T] {
	return func(c *config[T]) {
		c.chainState = &ChainState{
			LastHash: genesisHash,
			Length:   0,
		}
		c.hashChain = &hashChainHooks[T]{
			beforeWrite: beforeWrite,
			afterWrite:  afterWrite,
			loadTail:    loadTail,
		}
	}
}
