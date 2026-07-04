package debugtrace

import (
	"sync"
	"time"
)

// Event is an in-memory diagnostic event. The public build keeps this as a
// process-local buffer only; it does not expose a local monitor server.
type Event struct {
	ID      int         `json:"id"`
	Time    string      `json:"time"`
	TraceID string      `json:"trace_id"`
	Node    string      `json:"node"`
	Data    interface{} `json:"data"`
}

var store = struct {
	sync.Mutex
	nextID int
	events []Event
}{
	events: make([]Event, 0, 256),
}

// Start is retained for compatibility with older call sites. It intentionally
// does not start a network listener.
func Start(_ string) {}

func Record(node, traceID string, data interface{}) {
	store.Lock()
	defer store.Unlock()
	store.nextID++
	store.events = append(store.events, Event{
		ID:      store.nextID,
		Time:    time.Now().Format(time.RFC3339Nano),
		TraceID: traceID,
		Node:    node,
		Data:    data,
	})
	if len(store.events) > 500 {
		store.events = store.events[len(store.events)-500:]
	}
}

func EventsSnapshot() []Event {
	store.Lock()
	defer store.Unlock()
	return append([]Event(nil), store.events...)
}
