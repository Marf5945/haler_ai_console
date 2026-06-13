package execution_hook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"ui_console/audit_log"
)

// ChainEntry is one immutable record in the append-only hash chain.
// The hash links to the previous entry, ensuring tampering is detectable.
type ChainEntry struct {
	Index        int       `json:"index"`
	Type         string    `json:"type"`
	Payload      string    `json:"payload"`
	PreviousHash string    `json:"previous_hash"`
	Hash         string    `json:"hash"`
	CreatedAt    time.Time `json:"created_at"`
}

// HashChain implements the proposal append-only hash chain.
// Default retention: 2 years (enforced by retention sweep — not yet implemented).
// 重構：v4.0 委託 audit_log.AppendLog[ChainEntry] 共用抽象。
type HashChain struct {
	inner *audit_log.AppendLog[ChainEntry]
}

const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// NewHashChain creates (or re-opens) the hash chain stored at
// <hookDir>/hook_proposal_hash_chain.jsonl.
func NewHashChain(hookDir string) *HashChain {
	p := filepath.Join(hookDir, "hook_proposal_hash_chain.jsonl")
	return &HashChain{
		inner: audit_log.New[ChainEntry](
			p,
			audit_log.WithHashChain[ChainEntry](
				genesisHash,
				// beforeWrite: 設定 Index, PreviousHash, Hash
				func(entry *ChainEntry, state *audit_log.ChainState) {
					entry.Index = state.Length
					entry.PreviousHash = state.LastHash
					entry.Hash = computeChainHash(*entry)
				},
				// afterWrite: 更新 state
				func(entry *ChainEntry, state *audit_log.ChainState) {
					state.LastHash = entry.Hash
					state.Length++
				},
				// loadTail: 從最後一筆恢復
				func(entry *ChainEntry, state *audit_log.ChainState) {
					state.LastHash = entry.Hash
					state.Length = entry.Index + 1
				},
			),
		),
	}
}

// Append adds a new entry to the chain. Thread-safe.
func (hc *HashChain) Append(entry ChainEntry) error {
	return hc.inner.Append(entry)
}

// Verify reads the full chain and checks every hash link.
// Returns nil if the chain is intact.
func (hc *HashChain) Verify() error {
	entries, err := hc.inner.ReadAll()
	if err != nil {
		return err
	}

	prev := genesisHash
	for _, entry := range entries {
		if entry.PreviousHash != prev {
			return fmt.Errorf("hash chain: broken link at index %d", entry.Index)
		}
		expected := computeChainHash(entry)
		if entry.Hash != expected {
			return fmt.Errorf("hash chain: hash mismatch at index %d", entry.Index)
		}
		prev = entry.Hash
	}
	return nil
}

// computeChainHash produces the SHA-256 digest for a chain entry.
func computeChainHash(e ChainEntry) string {
	raw := fmt.Sprintf("%d|%s|%s|%s|%s", e.Index, e.Type, e.Payload, e.PreviousHash, e.CreatedAt.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// computeHash is a lightweight helper used elsewhere in the package.
func computeHash(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}
