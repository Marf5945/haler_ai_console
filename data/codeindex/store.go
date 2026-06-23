package codeindex

import (
	"encoding/json"
	"os"
	"path/filepath"

	domain "ui_console/domain/codeindex"
)

const snapshotFile = "snapshot.json"

type Store struct {
	rootDir      string
	snapshotPath string
}

func NewStore(projectRoot string) *Store {
	rootDir := filepath.Join(projectRoot, "runtime", "codeindex")
	return &Store{
		rootDir:      rootDir,
		snapshotPath: filepath.Join(rootDir, snapshotFile),
	}
}

func (s *Store) EnsureLayout() error {
	return os.MkdirAll(s.rootDir, 0o755)
}

func (s *Store) Load() (domain.Snapshot, error) {
	if err := s.EnsureLayout(); err != nil {
		return domain.Snapshot{}, err
	}
	data, err := os.ReadFile(s.snapshotPath)
	if os.IsNotExist(err) {
		return domain.Snapshot{Version: 1}, nil
	}
	if err != nil {
		return domain.Snapshot{}, err
	}
	var snapshot domain.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return domain.Snapshot{}, err
	}
	if snapshot.Version == 0 {
		snapshot.Version = 1
	}
	return snapshot, nil
}

func (s *Store) Save(snapshot domain.Snapshot) error {
	if err := s.EnsureLayout(); err != nil {
		return err
	}
	snapshot.Version = 1
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.snapshotPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.snapshotPath)
}
