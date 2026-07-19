// Package eventbus provides a unified app-level event bus backed by Wails runtime.EventsEmit.
// It replaces the legacy WebSocket event log with typed, in-process event dispatch.
//
// Usage (Go side):
//
//	bus := eventbus.New(ctx)  // ctx from app.startup
//	bus.Emit(EventAdapterStatusChanged, payload)
//
// Usage (Frontend):
//
//	import { EventsOn } from '../wailsjs/runtime/runtime';
//	EventsOn("adapter:status_changed", (payload) => { ... });
//
// Event naming convention: "domain:action" (e.g. "dag:node_completed", "review:card_added").
// All events are fire-and-forget; subscribers receive a JSON-serializable payload.
package eventbus

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Event name constants — centralised so both Go emitters and JS subscribers reference the same strings.
const (
	// Adapter events
	EventAdapterListChanged        = "adapter:list_changed"
	EventAdapterStatusChanged      = "adapter:status_changed"
	EventAdapterModelChoiceCleared = "adapter:model_choice_cleared"

	// DAG events
	EventDagRunStarted    = "dag:run_started"
	EventDagNodeCompleted = "dag:node_completed"
	EventDagRunCompleted  = "dag:run_completed"
	EventDagRunFailed     = "dag:run_failed"

	// Review events
	EventReviewCardAdded       = "review:card_added"
	EventReviewCardResolved    = "review:card_resolved"
	EventReviewCardInvalidated = "review:card_invalidated"

	// Status Rail events
	EventStatusRailUpdated = "statusrail:updated"

	// Degraded mode events
	EventDegradedModeEntered = "degraded:entered"
	EventDegradedModeExited  = "degraded:exited"

	// Memory / health events
	EventMemoryHealthChanged = "memory:health_changed"

	// Sidecar / execution events (#I-805)
	EventExecutionInterrupted = "dag:execution_interrupted"
	EventSidecarStateChanged  = "sidecar:state_changed"

	// CLI 授權事件：當 CLI（如 Gemini）需要瀏覽器 OAuth 授權時，
	// 由 Go 端 emit 此事件，前端顯示授權對話框。
	EventCLIAuthRequired = "cli:auth_required"

	// Digest events (#I-1001)
	EventDigestAutoArchived = "digest:auto_archived"
	EventDigestUpdated      = "digest:updated"

	// Embedding events (Phase B M3)
	EventEmbeddingConfigMissing = "embedding:config_missing"
	EventEmbeddingPullStarted   = "embedding:pull_started"
	EventEmbeddingPullProgress  = "embedding:pull_progress"
	EventEmbeddingPullDone      = "embedding:pull_done"
	EventEmbeddingPullFailed    = "embedding:pull_failed"

	// Web search setup
	EventWebSearchConfigRequired = "web_search:config_required"

	// GGUF 匯入（引用連結貼 .gguf → 下載 → ollama create → 本地模型）
	EventGGUFImportStarted  = "gguf:import_started"
	EventGGUFImportProgress = "gguf:import_progress"
	EventGGUFImportDone     = "gguf:import_done"
	EventGGUFImportFailed   = "gguf:import_failed"
)

// Bus is the central event dispatcher. Holds the Wails context for EventsEmit.
type Bus struct {
	mu   sync.Mutex
	ctx  context.Context
	sink Sink
}

// Sink is the small injection point used by in-process consumers and tests.
// Production buses leave it nil and continue through Wails runtime.EventsEmit.
type Sink func(eventName string, payload interface{})

// New creates a Bus. Pass the context received in app.startup.
// If ctx is nil (e.g. during unit tests), Emit becomes a no-op.
func New(ctx context.Context) *Bus {
	return &Bus{ctx: ctx}
}

// NewWithSink creates a bus whose events are delivered synchronously to sink.
// This keeps event-condition tests independent from a live Wails runtime.
func NewWithSink(sink Sink) *Bus {
	return &Bus{sink: sink}
}

// SetContext allows deferred context injection (useful when Bus is created before startup).
func (b *Bus) SetContext(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ctx = ctx
}

// Emit sends an event to all frontend subscribers.
// eventName should be one of the Event* constants above.
// payload must be JSON-serializable (struct, map, string, etc.).
func (b *Bus) Emit(eventName string, payload interface{}) {
	b.mu.Lock()
	ctx := b.ctx
	sink := b.sink
	b.mu.Unlock()
	if sink != nil {
		sink(eventName, payload)
		return
	}
	if ctx == nil {
		return // no-op in test or pre-startup
	}
	runtime.EventsEmit(ctx, eventName, payload)
}
