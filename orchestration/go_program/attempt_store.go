package go_program

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type AttemptRecord struct {
	Attempt        int               `json:"attempt"`
	Hash           string            `json:"hash"`
	ErrorSignature string            `json:"error_signature,omitempty"`
	Files          []string          `json:"files"`
	Manifest       Manifest          `json:"manifest"`
	Toolchain      Toolchain         `json:"toolchain"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type AttemptStore struct {
	Root string
}

func NewAttemptStore(root string) *AttemptStore {
	return &AttemptStore{Root: root}
}

func (s *AttemptStore) SaveAttempt(attempt int, manifest Manifest, toolchain Toolchain, files map[string]string, errorSignature string) (*AttemptRecord, error) {
	if attempt <= 0 {
		return nil, fmt.Errorf("go_program: attempt must be positive")
	}
	dir := filepath.Join(s.Root, fmt.Sprintf("attempt-%d", attempt))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("go_program: create attempt dir: %w", err)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := writeAttemptFile(dir, name, files[name]); err != nil {
			return nil, err
		}
	}
	attemptManifest := manifest
	attemptManifest.SourceDir = dir
	hash, goFiles, err := SourceHash(dir, attemptManifest, toolchain)
	if err != nil {
		return nil, err
	}
	rec := &AttemptRecord{
		Attempt:        attempt,
		Hash:           hash,
		ErrorSignature: errorSignature,
		Files:          relFiles(dir, goFiles),
		Manifest:       attemptManifest,
		Toolchain:      toolchain,
	}
	data, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "attempt.json"), data, 0o600); err != nil {
		return nil, fmt.Errorf("go_program: write attempt metadata: %w", err)
	}
	return rec, nil
}

func writeAttemptFile(root, name, content string) error {
	clean := filepath.Clean(name)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("go_program: invalid attempt file path %q", name)
	}
	path := filepath.Join(root, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func RepeatedProgress(records []AttemptRecord) bool {
	if len(records) < 2 {
		return false
	}
	last := records[len(records)-1]
	prev := records[len(records)-2]
	if last.Hash != "" && last.Hash == prev.Hash {
		return true
	}
	return last.ErrorSignature != "" && last.ErrorSignature == prev.ErrorSignature
}
