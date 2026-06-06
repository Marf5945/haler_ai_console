package main

import (
	"testing"

	"ui_console/adapter/adapter_registry"
	"ui_console/domain/degraded"
	"ui_console/shared/eventbus"
	"ui_console/shared/health"
	"ui_console/shared/onboarding"
	"ui_console/domain/review"
)

// =============================================================================
// Integration Tests — 遺留待重建能力 (#1–#8)
// Covers: StatusRail, Review, DAG, Adapter, Degraded mode
// =============================================================================

// --- #1 Adapter Registry ---

func TestAdapterRegistry_ListAvailable(t *testing.T) {
	svc := adapter_registry.NewService(t.TempDir())
	if adapters := svc.ListAvailable(); len(adapters) != 0 {
		t.Fatalf("new registry should not seed mock adapters, got %d", len(adapters))
	}
	if err := svc.Register("claude-cli", "Claude", "C"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	adapters := svc.ListAvailable()
	if len(adapters) != 1 {
		t.Fatalf("expected registered adapter only, got %d", len(adapters))
	}
	// Verify Claude is in the list
	found := false
	for _, a := range adapters {
		if a.Name == "Claude" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Claude adapter in default list")
	}
}

func TestAdapterRegistry_SetStatus(t *testing.T) {
	svc := adapter_registry.NewService(t.TempDir())
	if err := svc.Register("claude-cli", "Claude", "C"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	err := svc.SetStatus("claude-cli", adapter_registry.StatusOffline)
	if err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}
	a, _ := svc.GetStatus("claude-cli")
	if a.Status != adapter_registry.StatusOffline {
		t.Errorf("expected offline, got %s", a.Status)
	}
}

func TestAdapterRegistry_NotFound(t *testing.T) {
	svc := adapter_registry.NewService(t.TempDir())
	_, err := svc.GetStatus("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent adapter")
	}
}

// --- #2 Review Card ---

func TestReviewCard_AddAndList(t *testing.T) {
	svc := review.NewService()
	svc.AddLegacyCard(review.LevelBlocking, "test", "src-1", "高風險操作", "engineer detail")
	svc.AddLegacyCard(review.LevelPending, "test", "src-2", "一般待審", "")

	open := svc.ListOpen()
	if len(open) != 2 {
		t.Fatalf("expected 2 open cards, got %d", len(open))
	}
}

func TestReviewCard_Resolve(t *testing.T) {
	svc := review.NewService()
	card := svc.AddLegacyCard(review.LevelBlocking, "test", "src-1", "reason", "")
	if !svc.HasBlocking() {
		t.Error("expected HasBlocking=true")
	}
	svc.Resolve(card.ID)
	if svc.HasBlocking() {
		t.Error("expected HasBlocking=false after resolve")
	}
}

func TestReviewCard_InvalidateAll(t *testing.T) {
	svc := review.NewService()
	svc.AddLegacyCard(review.LevelBlocking, "test", "src-1", "", "")
	svc.AddLegacyCard(review.LevelPending, "test", "src-2", "", "")
	count := svc.InvalidateAll()
	if count != 2 {
		t.Errorf("expected 2 invalidated, got %d", count)
	}
	if len(svc.ListOpen()) != 0 {
		t.Error("expected no open cards after invalidation")
	}
}

// --- #3 DAG Events (via EventBus) ---

func TestEventBus_EmitNoContext(t *testing.T) {
	// With nil context, Emit should be a no-op (no panic)
	bus := eventbus.New(nil)
	bus.Emit(eventbus.EventDagRunStarted, map[string]string{"run_id": "test-1"})
	// No assertion needed — just verifying no panic
}

// --- #4 Memory Health / Config Public ---

func TestHealthService_GetMemoryHealth(t *testing.T) {
	svc := health.NewService(t.TempDir(), false)
	h := svc.GetMemoryHealth()
	if h.HeapAllocMB <= 0 {
		t.Error("expected positive heap allocation")
	}
	if h.NumGoroutines <= 0 {
		t.Error("expected at least one goroutine")
	}
}

func TestHealthService_GetConfigPublic(t *testing.T) {
	svc := health.NewService(t.TempDir(), true)
	cfg := svc.GetConfigPublic()
	if cfg.AppVersion == "" {
		t.Error("expected non-empty app version")
	}
	if !cfg.DevMode {
		t.Error("expected dev mode true")
	}
}

// --- #5 Degraded Mode ---

func TestDegradedMode_EnterExit(t *testing.T) {
	svc := degraded.NewService()
	state := svc.GetState()
	if state.Active {
		t.Error("expected inactive on init")
	}

	state = svc.Enter(degraded.ReasonAllAdaptersOffline)
	if !state.Active {
		t.Error("expected active after Enter")
	}
	if len(state.BlockedOps) == 0 {
		t.Error("expected blocked operations list")
	}

	if !svc.IsBlocked("tool_execution") {
		t.Error("expected tool_execution blocked in degraded mode")
	}
	if !svc.IsBlocked("dag_resume") {
		t.Error("expected dag_resume blocked in degraded mode")
	}

	state = svc.Exit()
	if state.Active {
		t.Error("expected inactive after Exit")
	}
	if svc.IsBlocked("tool_execution") {
		t.Error("expected tool_execution not blocked after exit")
	}
}

func TestDegradedMode_InvalidatesReviewCards(t *testing.T) {
	reviewSvc := review.NewService()
	degradedSvc := degraded.NewService()

	reviewSvc.AddLegacyCard(review.LevelBlocking, "test", "s1", "", "")
	reviewSvc.AddLegacyCard(review.LevelPending, "test", "s2", "", "")

	// Simulate entering degraded mode (app.go does this)
	degradedSvc.Enter(degraded.ReasonManualTrigger)
	count := reviewSvc.InvalidateAll()
	if count != 2 {
		t.Errorf("expected 2 cards invalidated, got %d", count)
	}
}

// --- #6 Onboarding / Read-only Mode ---

func TestOnboarding_FirstRun(t *testing.T) {
	svc := onboarding.NewService(t.TempDir())
	state := svc.GetState()
	if !state.IsFirstRun {
		t.Error("expected first run on fresh directory")
	}
	if !state.ReadOnlyMode {
		t.Error("expected read-only mode on first run")
	}
	if len(state.Steps) == 0 {
		t.Error("expected onboarding steps")
	}
}

func TestOnboarding_CompleteSteps(t *testing.T) {
	dir := t.TempDir()
	svc := onboarding.NewService(dir)

	for _, step := range svc.GetState().Steps {
		svc.CompleteStep(step.ID)
	}

	// After all steps, still read-only until MarkComplete
	if !svc.IsReadOnly() {
		t.Error("expected still read-only until MarkComplete")
	}

	err := svc.MarkComplete()
	if err != nil {
		t.Fatalf("MarkComplete failed: %v", err)
	}
	if svc.IsReadOnly() {
		t.Error("expected read-only=false after MarkComplete")
	}

	// Second instantiation should detect first_run_done marker
	svc2 := onboarding.NewService(dir)
	if svc2.GetState().IsFirstRun {
		t.Error("expected first_run=false after MarkComplete")
	}
}

// --- #7 EventBus Constants ---

func TestEventBus_ConstantsExist(t *testing.T) {
	// Verify all expected event constants are defined
	events := []string{
		eventbus.EventAdapterListChanged,
		eventbus.EventAdapterStatusChanged,
		eventbus.EventDagRunStarted,
		eventbus.EventDagNodeCompleted,
		eventbus.EventDagRunCompleted,
		eventbus.EventDagRunFailed,
		eventbus.EventReviewCardAdded,
		eventbus.EventReviewCardResolved,
		eventbus.EventStatusRailUpdated,
		eventbus.EventDegradedModeEntered,
		eventbus.EventDegradedModeExited,
		eventbus.EventMemoryHealthChanged,
	}
	for _, e := range events {
		if e == "" {
			t.Error("found empty event constant")
		}
	}
}

// --- StatusRail integration (existing service, verifying binding works) ---

func TestStatusRail_ViewNotEmpty(t *testing.T) {
	app := NewApp()
	state := app.GetConsoleState()
	if state.StatusRail.Text == "" {
		t.Error("expected non-empty status rail text")
	}
	if len(state.Adapters) == 0 {
		t.Error("expected adapters from registry")
	}
}
