package blackboard

import (
	"encoding/json"
	"os"
)

// maxProcessedIDs bounds the dedup window kept in the cursor (spec §11).
const maxProcessedIDs = 512

// Cursor is each participant's local read position. Replaying the same
// batch with the same cursor must not change any outcome (idempotency).
type Cursor struct {
	DocID            string   `json:"doc_id"`
	LastSeenEventID  string   `json:"last_seen_event_id"`
	ProcessedIDs     []string `json:"processed_ids"`
	LastProjectionID string   `json:"last_projection_id"`
}

// MarkProcessed records an event id, keeping the window bounded.
func (c *Cursor) MarkProcessed(id string) {
	for _, p := range c.ProcessedIDs {
		if p == id {
			return
		}
	}
	c.ProcessedIDs = append(c.ProcessedIDs, id)
	if len(c.ProcessedIDs) > maxProcessedIDs {
		c.ProcessedIDs = c.ProcessedIDs[len(c.ProcessedIDs)-maxProcessedIDs:]
	}
	c.LastSeenEventID = id
}

// Processed reports whether an id is inside the dedup window.
func (c *Cursor) Processed(id string) bool {
	for _, p := range c.ProcessedIDs {
		if p == id {
			return true
		}
	}
	return false
}

// LoadCursor reads a cursor file; a missing file yields a zero cursor.
func LoadCursor(path string) (Cursor, error) {
	var c Cursor
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

// SaveCursor writes the cursor atomically.
func SaveCursor(path string, c Cursor) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, b)
}
