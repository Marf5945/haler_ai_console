package blackboard

// tombstone.go — 有意移除的事件（歸檔、redaction）記入墓碑名單，
// integrity check 不再視為手滑事故（spec §12、§13）。
// 獨立檔案，避免與 mirror.go 的演進互相衝突。

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (m *Mirror) tombstonesPath() string {
	return filepath.Join(m.Dir, "blackboard_tombstones.json")
}

type tombstoneFile struct {
	IDs []string `json:"ids"`
}

// LoadTombstones returns the set of intentionally removed event ids.
func (m *Mirror) LoadTombstones() (map[string]bool, error) {
	out := map[string]bool{}
	b, err := os.ReadFile(m.tombstonesPath())
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	var tf tombstoneFile
	if err := json.Unmarshal(b, &tf); err != nil {
		return nil, err
	}
	for _, id := range tf.IDs {
		out[id] = true
	}
	return out, nil
}

// AddTombstones records ids as intentionally removed (idempotent).
func (m *Mirror) AddTombstones(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	existing, err := m.LoadTombstones()
	if err != nil {
		return err
	}
	changed := false
	for _, id := range ids {
		if !existing[id] {
			existing[id] = true
			changed = true
		}
	}
	if !changed {
		return nil
	}
	all := make([]string, 0, len(existing))
	for id := range existing {
		all = append(all, id)
	}
	sortStrings(all)
	b, err := json.MarshalIndent(tombstoneFile{IDs: all}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.Dir, 0o755); err != nil {
		return err
	}
	return atomicWrite(m.tombstonesPath(), b)
}
