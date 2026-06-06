package scheduler

import (
	"context"
	"errors"
	"testing"

	"ui_console/shared/eventbus"
)

type mockSkillExecutor struct {
	actionTarget string
	sessionID    string
	calls        int
	err          error
}

func (m *mockSkillExecutor) ExecuteSkill(ctx context.Context, actionTarget string, sessionID string) error {
	m.calls++
	m.actionTarget = actionTarget
	m.sessionID = sessionID
	return m.err
}

func TestActionCallbackExecutesRegisteredFunction(t *testing.T) {
	registry := NewCallbackRegistry()
	action := NewCallbackAction(registry)
	var gotArgs string
	registry.Register("sample", func(ctx context.Context, args string) error {
		gotArgs = args
		return nil
	})

	if err := action.Execute(context.Background(), `{"callback_name":"sample","args":"{\"ok\":true}"}`); err != nil {
		t.Fatalf("CallbackAction.Execute returned error: %v", err)
	}
	if gotArgs != `{"ok":true}` {
		t.Fatalf("callback args = %q, want JSON string", gotArgs)
	}
}

func TestActionCallbackMissingReturnsError(t *testing.T) {
	action := NewCallbackAction(NewCallbackRegistry())
	if err := action.Execute(context.Background(), `{"callback_name":"missing"}`); err == nil {
		t.Fatalf("expected missing callback error")
	}
}

func TestActionSkillExecutesResolver(t *testing.T) {
	exec := &mockSkillExecutor{}
	action := NewSkillAction(exec)

	if err := action.Execute(context.Background(), `{"action_target":"查詢ㄌ天氣","session_id":"scheduler"}`); err != nil {
		t.Fatalf("SkillAction.Execute returned error: %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("skill executor calls = %d, want 1", exec.calls)
	}
	if exec.actionTarget != "查詢ㄌ天氣" || exec.sessionID != "scheduler" {
		t.Fatalf("skill executor got target=%q session=%q", exec.actionTarget, exec.sessionID)
	}
}

func TestActionSkillPropagatesExecutorError(t *testing.T) {
	wantErr := errors.New("boom")
	action := NewSkillAction(&mockSkillExecutor{err: wantErr})

	if err := action.Execute(context.Background(), `{"action_target":"查詢ㄌ天氣"}`); !errors.Is(err, wantErr) {
		t.Fatalf("SkillAction.Execute error = %v, want %v", err, wantErr)
	}
}

func TestActionEventBusValidatesPayload(t *testing.T) {
	action := NewEventBusAction(eventbus.New(nil))
	if err := action.Execute(context.Background(), `{"event_name":"scheduler:test","data":{"ok":true}}`); err != nil {
		t.Fatalf("EventBusAction.Execute returned error: %v", err)
	}
	if err := action.Execute(context.Background(), `{"data":{}}`); err == nil {
		t.Fatalf("expected missing event_name error")
	}
}

func TestActionResolverRejectsUnsupportedType(t *testing.T) {
	resolver := NewActionResolver(eventbus.New(nil), &mockSkillExecutor{}, NewCallbackRegistry())
	if _, err := resolver.Resolve(ActionType("bad")); err == nil {
		t.Fatalf("expected unsupported action type error")
	}
}
