// stop_recovery/service.go — Stop Recovery UX（§21）。
// 當 CLI sidecar 崩潰或偵測到 critical_runtime_action 時顯示恢復卡片。
// 精簡四項動作 + 禁止動作清單。
package stop_recovery

import (
	"fmt"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// 允許 / 禁止動作定義
// ──────────────────────────────────────────────

// RecoveryAction 恢復動作。
type RecoveryAction string

const (
	// 允許的四項動作
	ActionHandleManuallyThenResume RecoveryAction = "user_handle_manually_then_resume"
	ActionDryRunCurrentStep        RecoveryAction = "dry_run_current_step"
	ActionDiscardSandbox           RecoveryAction = "discard_sandbox"
	ActionRestartSidecar           RecoveryAction = "restart_sidecar_runner"
)

// AllowedActions 所有允許的恢復動作。
var AllowedActions = []RecoveryAction{
	ActionHandleManuallyThenResume,
	ActionDryRunCurrentStep,
	ActionDiscardSandbox,
	ActionRestartSidecar,
}

// forbiddenActions 禁止的恢復動作（§21 硬規則）。
var forbiddenActions = map[string]bool{
	"ignore_and_continue_for_critical":         true,
	"auto_solve_captcha":                       true,
	"auto_login":                               true,
	"auto_confirm_payment":                     true,
	"auto_grant_permission":                    true,
}

// ──────────────────────────────────────────────
// Stop Recovery Card 結構
// ──────────────────────────────────────────────

// StopReason 停止原因。
type StopReason string

const (
	ReasonSidecarCrash         StopReason = "sidecar_crashed"
	ReasonCriticalRuntimeAction StopReason = "critical_runtime_action"
	ReasonUserStop             StopReason = "user_stop"
	ReasonResumeGuardFailed    StopReason = "resume_guard_failed"
)

// StopRecoveryCard 恢復卡片。
type StopRecoveryCard struct {
	ID               string             `json:"id"`
	StopReason       StopReason         `json:"stop_reason"`
	DetectedSignal   string             `json:"detected_signal"`    // 偵測到的信號描述
	SafeNextActions  []ActionOption     `json:"safe_next_actions"`  // 可用的恢復動作
	ResumeConditions []ResumeCondition  `json:"resume_conditions"`  // §21 恢復前置條件清單
	UserMessage      string             `json:"user_message"`       // 中文使用者提示
	CreatedAt        string             `json:"created_at"`
	Resolved         bool               `json:"resolved"`
	ResolvedAction   string             `json:"resolved_action"`
	ResolvedAt       string             `json:"resolved_at"`
}

// ResumeCondition 恢復前置條件（前端渲染為 checklist）。
type ResumeCondition struct {
	Description string `json:"description"` // 中文描述
	Met         bool   `json:"met"`         // controller 即時判斷是否已滿足
}

// ActionOption 單一動作選項（含 UI 顯示資訊）。
type ActionOption struct {
	Action      RecoveryAction `json:"action"`
	Label       string         `json:"label"`        // 中文按鈕標籤
	Description string         `json:"description"`  // 中文說明
}

// ──────────────────────────────────────────────
// Service 核心
// ──────────────────────────────────────────────

// Service 管理 Stop Recovery 流程。
type Service struct {
	mu    sync.Mutex
	cards map[string]*StopRecoveryCard
}

// NewService 建立 Stop Recovery service。
func NewService() *Service {
	return &Service{
		cards: make(map[string]*StopRecoveryCard),
	}
}

// ──────────────────────────────────────────────
// 建立恢復卡片
// ──────────────────────────────────────────────

// CreateCard 建立恢復卡片。
// 根據停止原因自動組合可用的恢復動作。
func (s *Service) CreateCard(reason StopReason, signal string) *StopRecoveryCard {
	s.mu.Lock()
	defer s.mu.Unlock()

	card := &StopRecoveryCard{
		ID:             fmt.Sprintf("stop-%d", time.Now().UnixNano()),
		StopReason:     reason,
		DetectedSignal: signal,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	// 根據原因設定使用者提示和可用動作
	switch reason {
	case ReasonSidecarCrash:
		card.UserMessage = "CLI 執行環境意外停止。請選擇恢復方式："
		card.SafeNextActions = []ActionOption{
			{Action: ActionRestartSidecar, Label: "重新啟動執行環境", Description: "重新啟動 CLI sidecar，從上次中斷處繼續"},
			{Action: ActionHandleManuallyThenResume, Label: "手動處理後恢復", Description: "自行處理問題後，點擊恢復繼續執行"},
			{Action: ActionDiscardSandbox, Label: "丟棄本次作業", Description: "放棄目前的草稿沙盒，不保留任何變更"},
		}
		card.ResumeConditions = []ResumeCondition{
			{Description: "Sidecar process 已重新啟動", Met: false},
			{Description: "DAG checkpoint 可讀取", Met: true},
		}

	case ReasonCriticalRuntimeAction:
		card.UserMessage = fmt.Sprintf("偵測到需要人工介入的操作：%s。系統已暫停，請選擇：", signal)
		card.SafeNextActions = []ActionOption{
			{Action: ActionHandleManuallyThenResume, Label: "手動處理後恢復", Description: "自行完成該操作後，點擊恢復繼續"},
			{Action: ActionDryRunCurrentStep, Label: "模擬執行（不實際操作）", Description: "以 dry-run 模式執行當前步驟，確認結果後再決定"},
			{Action: ActionDiscardSandbox, Label: "丟棄本次作業", Description: "放棄目前的草稿沙盒���不保留任何變更"},
		}
		card.ResumeConditions = []ResumeCondition{
			{Description: "使用者已在外部完成該操作（登入/付款/授權/CAPTCHA）", Met: false},
			{Description: "操作結果已確認無誤", Met: false},
		}

	case ReasonUserStop:
		card.UserMessage = "您已手動停止執行。請選擇後續動作："
		card.SafeNextActions = []ActionOption{
			{Action: ActionHandleManuallyThenResume, Label: "稍後恢復", Description: "確認環境後恢復執行"},
			{Action: ActionDiscardSandbox, Label: "丟棄本次作業", Description: "放棄目前的草稿沙盒"},
		}
		card.ResumeConditions = []ResumeCondition{
			{Description: "使用者確認環境無異常", Met: false},
		}

	case ReasonResumeGuardFailed:
		card.UserMessage = "偵測到環境已變更（記憶/工具/政策/來源信任），無法安全恢復。"
		card.SafeNextActions = []ActionOption{
			{Action: ActionDiscardSandbox, Label: "丟棄並重新開始", Description: "放棄目前的 DAG 執行，重新建立"},
			{Action: ActionHandleManuallyThenResume, Label: "檢查變更後恢復", Description: "確認環境變更無影響後強制恢復"},
		}
		card.ResumeConditions = []ResumeCondition{
			{Description: "環境 hash 已重新計算並確認安全", Met: false},
			{Description: "變更內容已由使用者確認", Met: false},
		}

	default:
		card.UserMessage = "執行已停止，請選擇恢復方式。"
		card.SafeNextActions = buildDefaultActions()
	}

	s.cards[card.ID] = card
	return card
}

// ──────────────────────────────────────────────
// 解決恢復卡片
// ──────────────────────────────────────────────

// ResolveCard 解決恢復卡片（使用者選擇了動作）。
// §21 + TASKS_1_7：加入顯式 guard，拒絕不合理的 reason+action 組合。
func (s *Service) ResolveCard(cardID string, action RecoveryAction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 檢查是否為禁止動作
	if forbiddenActions[string(action)] {
		return fmt.Errorf("禁止的恢復動作: %s", action)
	}

	// 檢查是否為允許的動作
	if !isAllowedAction(action) {
		return fmt.Errorf("不允許的恢復動作: %s", action)
	}

	card, ok := s.cards[cardID]
	if !ok {
		return fmt.Errorf("恢復卡片不存在: %s", cardID)
	}
	if card.Resolved {
		return fmt.Errorf("恢復卡片已解決: %s", cardID)
	}

	// 顯式 guard：critical_runtime_action 不可使用 restart_sidecar
	// sidecar 本身正常，restart 無意義且可能丟失 session
	if card.StopReason == ReasonCriticalRuntimeAction && action == ActionRestartSidecar {
		return fmt.Errorf("critical_runtime_action 不允許 restart_sidecar_runner：sidecar 正常運作中")
	}

	// 顯式 guard：resume_guard_failed 不可使用 restart_sidecar
	if card.StopReason == ReasonResumeGuardFailed && action == ActionRestartSidecar {
		return fmt.Errorf("resume_guard_failed 不允許 restart_sidecar_runner：問題不在 sidecar")
	}

	card.Resolved = true
	card.ResolvedAction = string(action)
	card.ResolvedAt = time.Now().Format(time.RFC3339)
	return nil
}

// ──────────────────────────────────────────────
// 查詢
// ──────────────────────────────────────────────

// ListOpen 列出所有未解決的恢復卡片。
func (s *Service) ListOpen() []*StopRecoveryCard {
	s.mu.Lock()
	defer s.mu.Unlock()

	var open []*StopRecoveryCard
	for _, card := range s.cards {
		if !card.Resolved {
			open = append(open, card)
		}
	}
	return open
}

// GetCard 取得指定恢復卡片。
func (s *Service) GetCard(cardID string) (*StopRecoveryCard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("恢復卡片不存在: %s", cardID)
	}
	return card, nil
}

// HasOpen 是否有未解決的恢復卡片。
func (s *Service) HasOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, card := range s.cards {
		if !card.Resolved {
			return true
		}
	}
	return false
}

// IsForbiddenAction 檢查動作是否被禁止。
func IsForbiddenAction(action string) bool {
	return forbiddenActions[action]
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

func isAllowedAction(action RecoveryAction) bool {
	for _, a := range AllowedActions {
		if a == action {
			return true
		}
	}
	return false
}

func buildDefaultActions() []ActionOption {
	return []ActionOption{
		{Action: ActionHandleManuallyThenResume, Label: "手動處理後恢復", Description: "自行處理後恢復執行"},
		{Action: ActionDryRunCurrentStep, Label: "模擬執行", Description: "以 dry-run 模式執行"},
		{Action: ActionDiscardSandbox, Label: "丟棄作業", Description: "放棄目前的草稿沙盒"},
		{Action: ActionRestartSidecar, Label: "重啟執行環境", Description: "重新啟動 CLI sidecar"},
	}
}
