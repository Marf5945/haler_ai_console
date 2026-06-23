// Package scheduler 提供排程任務的動作執行機制。
// 本檔案定義了 Action 介面及其各種實作，包含事件匯流排動作、技能動作、回呼動作，
// 以及將 ActionType 對應到具體 Action 實作的 ActionResolver。
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"ui_console/shared/eventbus"
)

// ---------------------------------------------------------------------------
// Action 介面
// ---------------------------------------------------------------------------

// Action 定義排程器可執行的動作介面。
// 所有具體的動作類型（事件匯流排、技能、回呼）都必須實作此介面。
type Action interface {
	// Execute 執行動作。payload 為 JSON 格式字串，具體結構依動作類型而異。
	Execute(ctx context.Context, payload string) error
}

// ---------------------------------------------------------------------------
// EventBusAction — 透過事件匯流排發送事件的動作
// ---------------------------------------------------------------------------

// eventBusPayload 定義 EventBusAction 的 payload JSON 結構。
type eventBusPayload struct {
	EventName string      `json:"event_name"` // 事件名稱
	Data      interface{} `json:"data"`       // 事件攜帶的資料
}

// EventBusAction 透過 eventbus.Bus 發送事件的動作實作。
// payload JSON 格式：{"event_name": "xxx", "data": {...}}
type EventBusAction struct {
	bus *eventbus.Bus // 事件匯流排實例
}

// NewEventBusAction 建立新的 EventBusAction。
func NewEventBusAction(bus *eventbus.Bus) *EventBusAction {
	return &EventBusAction{bus: bus}
}

// Execute 解析 payload JSON 並透過事件匯流排發送事件。
// payload 必須包含 event_name 欄位；data 欄位為選填。
func (a *EventBusAction) Execute(ctx context.Context, payload string) error {
	if a.bus == nil {
		return fmt.Errorf("EventBusAction 缺少 event bus")
	}

	var p eventBusPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("解析 EventBusAction payload 失敗: %w", err)
	}
	if p.EventName == "" {
		return fmt.Errorf("EventBusAction payload 缺少 event_name 欄位")
	}

	// 檢查 context 是否已取消
	select {
	case <-ctx.Done():
		return fmt.Errorf("EventBusAction 執行被取消: %w", ctx.Err())
	default:
	}

	a.bus.Emit(p.EventName, p.Data)
	return nil
}

// ---------------------------------------------------------------------------
// SkillExecutor 介面 — 抽象技能執行入口
// ---------------------------------------------------------------------------

// SkillExecutor 抽象 skill 執行入口，避免直接依賴 skill_step.Router 的具體簽名。
// 應用層（如 app.go）負責提供此介面的具體實作，將呼叫轉發至 skill_step.Router。
type SkillExecutor interface {
	// ExecuteSkill 執行指定的技能動作。
	// actionTarget 為技能目標識別字串，sessionID 為排程工作階段識別碼。
	ExecuteSkill(ctx context.Context, actionTarget string, sessionID string) error
}

// ---------------------------------------------------------------------------
// SkillAction — 透過 SkillExecutor 執行技能的動作
// ---------------------------------------------------------------------------

// skillPayload 定義 SkillAction 的 payload JSON 結構。
type skillPayload struct {
	ActionTarget string `json:"action_target"` // 技能目標識別字串
	SessionID    string `json:"session_id"`    // 工作階段識別碼
}

// SkillAction 透過 SkillExecutor 介面執行技能的動作實作。
// payload JSON 格式：{"action_target": "xxx", "session_id": "scheduler"}
type SkillAction struct {
	executor SkillExecutor // 技能執行器
}

// NewSkillAction 建立新的 SkillAction。
func NewSkillAction(executor SkillExecutor) *SkillAction {
	return &SkillAction{executor: executor}
}

// Execute 解析 payload JSON 並透過 SkillExecutor 執行技能。
// payload 必須包含 action_target 欄位；session_id 若未提供則預設為 "scheduler"。
func (a *SkillAction) Execute(ctx context.Context, payload string) error {
	if a.executor == nil {
		return fmt.Errorf("SkillAction 缺少 skill executor")
	}

	var p skillPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("解析 SkillAction payload 失敗: %w", err)
	}
	if p.ActionTarget == "" {
		return fmt.Errorf("SkillAction payload 缺少 action_target 欄位")
	}

	// 若未指定 session_id，使用預設值
	if p.SessionID == "" {
		p.SessionID = "scheduler"
	}

	return a.executor.ExecuteSkill(ctx, p.ActionTarget, p.SessionID)
}

// ---------------------------------------------------------------------------
// CallbackRegistry — 回呼函式註冊表
// ---------------------------------------------------------------------------

// CallbackFunc 定義回呼函式的型別。
type CallbackFunc func(ctx context.Context, args string) error

// CallbackRegistry 管理已註冊的回呼函式，支援執行緒安全的讀寫操作。
type CallbackRegistry struct {
	mu        sync.RWMutex            // 讀寫鎖，保護 callbacks 的並行存取
	callbacks map[string]CallbackFunc // 回呼函式對應表，key 為回呼名稱
}

// NewCallbackRegistry 建立新的 CallbackRegistry，初始化內部映射表。
func NewCallbackRegistry() *CallbackRegistry {
	return &CallbackRegistry{
		callbacks: make(map[string]CallbackFunc),
	}
}

// Register 註冊一個具名回呼函式。若名稱已存在，會覆蓋原有的回呼。
func (r *CallbackRegistry) Register(name string, fn CallbackFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks[name] = fn
}

// Get 依名稱查詢已註冊的回呼函式。
// 回傳回呼函式及是否存在的布林值。
func (r *CallbackRegistry) Get(name string) (CallbackFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.callbacks[name]
	return fn, ok
}

// List 回傳所有已註冊的回呼名稱，依字母順序排列。
func (r *CallbackRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.callbacks))
	for name := range r.callbacks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ---------------------------------------------------------------------------
// CallbackAction — 透過回呼註冊表執行回呼的動作
// ---------------------------------------------------------------------------

// callbackPayload 定義 CallbackAction 的 payload JSON 結構。
type callbackPayload struct {
	CallbackName string `json:"callback_name"` // 回呼函式名稱
	Args         string `json:"args"`          // 傳遞給回呼的引數字串
}

// CallbackAction 透過 CallbackRegistry 查找並執行回呼函式的動作實作。
// payload JSON 格式：{"callback_name": "xxx", "args": "..."}
type CallbackAction struct {
	registry *CallbackRegistry // 回呼註冊表
}

// NewCallbackAction 建立新的 CallbackAction。
func NewCallbackAction(registry *CallbackRegistry) *CallbackAction {
	return &CallbackAction{registry: registry}
}

// Execute 解析 payload JSON，從註冊表中查找回呼函式並執行。
// payload 必須包含 callback_name 欄位且該回呼必須已註冊。
func (a *CallbackAction) Execute(ctx context.Context, payload string) error {
	if a.registry == nil {
		return fmt.Errorf("CallbackAction 缺少 callback registry")
	}

	var p callbackPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return fmt.Errorf("解析 CallbackAction payload 失敗: %w", err)
	}
	if p.CallbackName == "" {
		return fmt.Errorf("CallbackAction payload 缺少 callback_name 欄位")
	}

	fn, ok := a.registry.Get(p.CallbackName)
	if !ok {
		return fmt.Errorf("回呼 %q 尚未註冊", p.CallbackName)
	}

	return fn(ctx, p.Args)
}

// ---------------------------------------------------------------------------
// ActionResolver — 依 ActionType 解析對應的 Action 實作
// ---------------------------------------------------------------------------

// ActionResolver 根據 ActionType 回傳對應的 Action 實作。
// 集中管理所有動作類型與其實作的對應關係。
type ActionResolver struct {
	eventBusAction *EventBusAction // 事件匯流排動作
	skillAction    *SkillAction    // 技能動作
	callbackAction *CallbackAction // 回呼動作
}

// NewActionResolver 建立新的 ActionResolver。
// 參數：
//   - bus: 事件匯流排實例，用於 EventBus 類型動作
//   - skillExec: 技能執行器介面，用於 Skill 類型動作
//   - callbacks: 回呼註冊表，用於 Callback 類型動作
func NewActionResolver(bus *eventbus.Bus, skillExec SkillExecutor, callbacks *CallbackRegistry) *ActionResolver {
	return &ActionResolver{
		eventBusAction: NewEventBusAction(bus),
		skillAction:    NewSkillAction(skillExec),
		callbackAction: NewCallbackAction(callbacks),
	}
}

// Resolve 根據 ActionType 回傳對應的 Action 實作。
// 支援的類型：ActionEvent、ActionSkill、ActionCallback。
// 若遇到不支援的類型，回傳錯誤。
func (r *ActionResolver) Resolve(actionType ActionType) (Action, error) {
	switch actionType {
	case ActionEvent:
		return r.eventBusAction, nil
	case ActionSkill:
		return r.skillAction, nil
	case ActionCallback:
		return r.callbackAction, nil
	default:
		return nil, fmt.Errorf("不支援的動作類型: %q", actionType)
	}
}
