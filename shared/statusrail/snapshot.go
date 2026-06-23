package statusrail

import "sync"

type Snapshot struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type SnapshotBuffer struct {
	mu    sync.Mutex
	items []Snapshot
	limit int
}

func NewSnapshotBuffer(limit int) *SnapshotBuffer {
	if limit < 1 {
		limit = 2
	}
	return &SnapshotBuffer{limit: limit}
}

func (b *SnapshotBuffer) Add(role, text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items = append(b.items, Snapshot{Role: role, Text: text})
	if len(b.items) > b.limit {
		b.items = append([]Snapshot(nil), b.items[len(b.items)-b.limit:]...)
	}
}

func (b *SnapshotBuffer) Items() []Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Snapshot(nil), b.items...)
}
