package debugtrace

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// DEBUG_TRACE_REMOVE: This package is a temporary local trace viewer for the
// UI -> Wails -> Go -> sidecar -> CLI path. Remove this folder and all
// DEBUG_TRACE_REMOVE call sites when the diagnostic work is done.
// Keep while cleanup is in progress: this is active diagnostic infrastructure,
// not an unused watcher. The monitor URL may be stale after the app exits.

const DefaultAddr = "127.0.0.1:48765"

// LinkSnapshot is the current monitor-link register exposed to UI/debug code.
type LinkSnapshot struct {
	URL       string `json:"url"`
	Addr      string `json:"addr"`
	Started   bool   `json:"started"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updated_at"`
	LastError string `json:"last_error,omitempty"`
}

type Event struct {
	ID      int         `json:"id"`
	Time    string      `json:"time"`
	TraceID string      `json:"trace_id"`
	Node    string      `json:"node"`
	Data    interface{} `json:"data"`
}

var store = struct {
	sync.Mutex
	nextID    int
	events    []Event
	started   bool
	addr      string
	url       string
	version   int
	updatedAt string
	lastError string
}{
	events: make([]Event, 0, 256),
}

func Start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", pageHandler)
	mux.HandleFunc("/events", eventsHandler)
	mux.HandleFunc("/trace", traceHandler)
	mux.HandleFunc("/clear", clearHandler)

	store.Lock()
	if store.started {
		store.Unlock()
		return
	}
	store.Unlock()

	listener, actualAddr, listenErr := listenWithFallback(addr)
	store.Lock()
	defer store.Unlock()
	if listenErr != nil {
		store.lastError = listenErr.Error()
		store.updatedAt = time.Now().Format(time.RFC3339Nano)
		store.version++
		log.Printf("DEBUG_TRACE_REMOVE: trace viewer unavailable: %v", listenErr)
		return
	}
	store.started = true
	store.addr = actualAddr
	store.url = "http://" + actualAddr
	store.lastError = ""
	store.updatedAt = time.Now().Format(time.RFC3339Nano)
	store.version++
	_ = os.Setenv("AI_CONSOLE_TRACE_URL", store.url)

	go func() {
		log.Printf("DEBUG_TRACE_REMOVE: trace viewer listening at http://%s", actualAddr)
		if err := http.Serve(listener, mux); err != nil && err != http.ErrServerClosed {
			log.Printf("DEBUG_TRACE_REMOVE: trace viewer stopped: %v", err)
		}
	}()
}

func URL() string {
	store.Lock()
	defer store.Unlock()
	if store.url != "" {
		return store.url
	}
	if store.addr != "" {
		return "http://" + store.addr
	}
	if env := os.Getenv("AI_CONSOLE_TRACE_URL"); env != "" {
		return env
	}
	return "http://" + DefaultAddr
}

// Snapshot returns the monitor-link register for UI and diagnostics.
func Snapshot() LinkSnapshot {
	store.Lock()
	defer store.Unlock()
	url := store.url
	addr := store.addr
	if url == "" {
		url = os.Getenv("AI_CONSOLE_TRACE_URL")
	}
	if url == "" {
		url = "http://" + DefaultAddr
	}
	if addr == "" {
		addr = DefaultAddr
	}
	return LinkSnapshot{
		URL:       url,
		Addr:      addr,
		Started:   store.started,
		Version:   store.version,
		UpdatedAt: store.updatedAt,
		LastError: store.lastError,
	}
}

func listenWithFallback(addr string) (net.Listener, string, error) {
	if addr == "" {
		addr = DefaultAddr
	}
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		return listener, listener.Addr().String(), nil
	}
	fallback, fallbackErr := net.Listen("tcp", "127.0.0.1:0")
	if fallbackErr != nil {
		return nil, "", fmt.Errorf("listen %s failed: %w; fallback failed: %v", addr, err, fallbackErr)
	}
	log.Printf("DEBUG_TRACE_REMOVE: trace viewer addr %s unavailable (%v), using %s", addr, err, fallback.Addr().String())
	return fallback, fallback.Addr().String(), nil
}

func URLFromEnvOrDefault() string {
	if env := os.Getenv("AI_CONSOLE_TRACE_URL"); env != "" {
		return env
	}
	return URL()
}

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

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	store.Lock()
	events := append([]Event(nil), store.events...)
	store.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events)
}

func traceHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Node    string      `json:"node"`
		TraceID string      `json:"trace_id"`
		Data    interface{} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	Record(req.Node, req.TraceID, req.Data)
	w.WriteHeader(http.StatusNoContent)
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	store.Lock()
	store.nextID = 0
	store.events = store.events[:0]
	store.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func pageHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, tracePageHTML)
}

const tracePageHTML = `<!doctype html>
<html lang="zh-Hant">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>CLI Trace Debug</title>
  <style>
    body { margin: 0; font: 14px/1.45 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #111; color: #eee; }
    header { position: sticky; top: 0; display: flex; gap: 12px; align-items: center; padding: 12px 16px; background: #1c1c1c; border-bottom: 1px solid #333; }
    h1 { margin: 0; font-size: 16px; }
    button { border: 1px solid #555; background: #242424; color: #eee; border-radius: 6px; padding: 6px 10px; cursor: pointer; }
    main { padding: 16px; }
    .event { border: 1px solid #333; border-radius: 8px; margin: 0 0 10px; background: #181818; overflow: hidden; }
    .meta { display: grid; grid-template-columns: 70px 190px 1fr 220px; gap: 10px; padding: 8px 10px; background: #202020; color: #bbb; }
    .node { color: #7dd3fc; font-weight: 700; }
    pre { margin: 0; padding: 10px; white-space: pre-wrap; word-break: break-word; color: #f6f6f6; }
    .empty { color: #aaa; }
  </style>
</head>
<body>
  <header>
    <h1>CLI Trace Debug</h1>
    <button id="clear">清除</button>
    <span id="count"></span>
  </header>
  <main id="events"><p class="empty">等待 trace...</p></main>
  <script>
    const root = document.getElementById('events');
    const count = document.getElementById('count');
    document.getElementById('clear').onclick = () => fetch('/clear', {method: 'POST'});
    async function load() {
      const events = await fetch('/events', {cache: 'no-store'}).then(r => r.json()).catch(() => []);
      count.textContent = events.length + ' events';
      if (!events.length) {
        root.innerHTML = '<p class="empty">等待 trace...</p>';
        return;
      }
      root.innerHTML = events.slice().reverse().map(e => (
        '<section class="event">' +
        '<div class="meta"><span>#' + e.id + '</span><span>' + e.time + '</span><span class="node">' + escapeHTML(e.node || '') + '</span><span>' + escapeHTML(e.trace_id || '') + '</span></div>' +
        '<pre>' + escapeHTML(JSON.stringify(e.data, null, 2)) + '</pre>' +
        '</section>'
      )).join('');
    }
    function escapeHTML(s) {
      return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
    }
    setInterval(load, 700);
    load();
  </script>
</body>
</html>`
