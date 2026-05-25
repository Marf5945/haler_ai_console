// remote_bridge/types.go — §12A Remote Bridge Communication 核心型別。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 本檔案是 Remote Bridge 模組的「資料字典」，定義所有跨檔案    │
// │ 共用的型別與常數。其他 .go 檔案只 import 本套件即可取用。    │
// │                                                             │
// │ 架構層次（由外到內）：                                      │
// │  1. ChannelType / ChannelMode  — 列舉常數                  │
// │  2. ChannelBinding             — 一個已註冊通道的完整狀態    │
// │  3. MessageEnvelope            — 手機→筆電的結構化訊息      │
// │  4. DispatchRecord             — 筆電→手機的分發記錄        │
// │  5. AuditEntry                 — 稽核日誌單筆               │
// │  6. ConnectionTestResult       — 連線測試回傳               │
// │  7. AggregationConfig          — 通知聚合設定               │
// │                                                             │
// │ Spec 對照：§12A.2–§12A.12                                  │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import "time"

// ──────────────────────────────────────────────
// 通道類型（§12A.2）
// ──────────────────────────────────────────────

// ChannelType 描述內建通訊軟體通道。
type ChannelType string

const (
	ChannelTelegram ChannelType = "telegram"
	ChannelDiscord  ChannelType = "discord"
	ChannelLINE     ChannelType = "line"
	ChannelTeams    ChannelType = "teams"
	ChannelQQ       ChannelType = "qq"
)

// AllChannelTypes 回傳所有內建通道類型。
func AllChannelTypes() []ChannelType {
	return []ChannelType{ChannelTelegram, ChannelDiscord, ChannelLINE, ChannelTeams, ChannelQQ}
}

// Label 回傳通道的使用者友善名稱。
func (c ChannelType) Label() string {
	switch c {
	case ChannelTelegram:
		return "Telegram"
	case ChannelDiscord:
		return "Discord"
	case ChannelLINE:
		return "LINE"
	case ChannelTeams:
		return "Teams"
	case ChannelQQ:
		return "QQ"
	default:
		return string(c)
	}
}

// Icon 回傳用於 UI 的文字 icon。
func (c ChannelType) Icon() string {
	switch c {
	case ChannelTelegram:
		return "TG"
	case ChannelDiscord:
		return "DC"
	case ChannelLINE:
		return "LN"
	case ChannelTeams:
		return "TM"
	case ChannelQQ:
		return "QQ"
	default:
		return "??"
	}
}

// ──────────────────────────────────────────────
// 權限模式（§12A.3）
// ──────────────────────────────────────────────

// ChannelMode 描述通道的權限等級。
type ChannelMode string

const (
	ModeNotificationOnly ChannelMode = "notification_only"
	ModeRemoteTaskSubmit ChannelMode = "remote_task_submit"
	ModeRemoteReview     ChannelMode = "remote_review"
)

// AllModes 回傳所有權限模式（用於 UI 選單）。
func AllModes() []ChannelMode {
	return []ChannelMode{ModeNotificationOnly, ModeRemoteTaskSubmit, ModeRemoteReview}
}

// Label 回傳模式的中文標籤。
func (m ChannelMode) Label() string {
	switch m {
	case ModeNotificationOnly:
		return "僅通知"
	case ModeRemoteTaskSubmit:
		return "遠端提交任務"
	case ModeRemoteReview:
		return "遠端審查"
	default:
		return string(m)
	}
}

// ──────────────────────────────────────────────
// 通道綁定（§12A.4）
// ──────────────────────────────────────────────
//
// ChannelBinding 是通道的「身分證」：
//   - 由 service.RegisterChannel() 建立
//   - 持久化到 remote_bridge_bindings.json
//   - 前端透過 ListRemoteBridgeChannels() 取得陣列後渲染 icon
//   - Active 欄位保證全域只有一個通道為 true（單一啟用限制）
//   - IsUsable() 檢查三重閘門：未撤銷 + 未過期 + 測試通過

// ChannelBinding 代表一個已設定的通訊通道。
type ChannelBinding struct {
	ID                string      `json:"id"`                             // 唯一識別碼
	DisplayName       string      `json:"display_name,omitempty"`         // 使用者自訂顯示名稱
	Channel           ChannelType `json:"channel"`                        // telegram / discord / line
	Mode              ChannelMode `json:"mode"`                           // 目前權限模式
	Active            bool        `json:"active"`                         // 是否啟用接收/發送
	Primary           bool        `json:"primary"`                        // 是否為唯一主頻道
	AllowedUserIDHash string      `json:"allowed_user_id_hash,omitempty"` // SHA-256 雜湊
	AllowedChatIDHash string      `json:"allowed_chat_id_hash,omitempty"` // SHA-256 雜湊
	MaxRemoteRisk     string      `json:"max_remote_risk"`                // high_non_destructive 上限
	ExpiresAt         *time.Time  `json:"expires_at,omitempty"`           // 過期時間
	Revoked           bool        `json:"revoked"`                        // 是否已撤銷
	CreatedAt         time.Time   `json:"created_at"`                     // 建立時間
	TestedAt          *time.Time  `json:"tested_at,omitempty"`            // 最近一次連線測試
	TestPassed        bool        `json:"test_passed"`                    // 連線測試是否通過

	// §12A.2 使用者分流模式（v1.7.1 新增）
	SetupMode           string          `json:"setup_mode,omitempty"`            // "quick" | "developer"
	PresetID            string          `json:"preset_id,omitempty"`             // quick mode 使用的 preset ID
	CustomWebhookConfig *WebhookRequest `json:"custom_webhook_config,omitempty"` // developer mode 自訂設定
}

// IsExpired 檢查通道綁定是否已過期。
func (b *ChannelBinding) IsExpired() bool {
	if b.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*b.ExpiresAt)
}

// IsUsable 檢查通道是否可用（未撤銷、未過期、測試通過）。
func (b *ChannelBinding) IsUsable() bool {
	return !b.Revoked && !b.IsExpired() && b.TestPassed
}

// ──────────────────────────────────────────────
// 訊息封裝（§12A.5）
// ──────────────────────────────────────────────
//
// 手機端發送的每則訊息，必須包裝成 MessageEnvelope 才能進入
// Laptop Controller 的處理流程。Controller 會根據 MessageType
// 決定後續動作（風險閘門、Review Card、DAG 排程等）。
// 禁止直接攜帶 raw shell command / tool_id / 座標等。

// MessageCategory 描述遠端訊息類別。
type MessageCategory string

const (
	CategoryRemoteIntent    MessageCategory = "remote_intent"
	CategoryReviewResponse  MessageCategory = "review_response"
	CategoryContinueRequest MessageCategory = "continue_request"
	CategoryRetryRequest    MessageCategory = "retry_request"
	CategoryStatusAck       MessageCategory = "status_ack"
)

// MessageEnvelope 遠端訊息結構化封裝（§12A.5）。
type MessageEnvelope struct {
	MessageType      MessageCategory        `json:"message_type"`
	SessionID        string                 `json:"session_id"`
	Channel          ChannelType            `json:"channel"`
	ChannelID        string                 `json:"channel_id"`
	ProjectID        string                 `json:"project_id"`
	Intent           string                 `json:"intent,omitempty"`
	ClientCapability map[string]bool        `json:"client_capability,omitempty"`
	Payload          map[string]interface{} `json:"payload,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// ──────────────────────────────────────────────
// 通知分發（§12A.6–12A.9）
// ──────────────────────────────────────────────

// DispatchRecord 記錄一次訊息分發。
type DispatchRecord struct {
	DispatchID    string      `json:"dispatch_id"`
	Channel       ChannelType `json:"channel"`
	ChannelIDHash string      `json:"channel_id_hash"`
	Mode          ChannelMode `json:"mode"`
	RiskClass     string      `json:"risk_class"`
	Redacted      bool        `json:"redaction_applied"`
	PartsCount    int         `json:"parts_count"`
	PartIndex     int         `json:"part_index,omitempty"`
	TotalParts    int         `json:"total_parts,omitempty"`
	ReviewID      string      `json:"review_id,omitempty"`
	DAGRunID      string      `json:"dag_run_id,omitempty"`
	ContentHash   string      `json:"content_hash,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

// ──────────────────────────────────────────────
// 稽核記錄（§12A.12）
// ──────────────────────────────────────────────

// AuditEntry Remote Bridge 稽核日誌的單筆記錄。
type AuditEntry struct {
	DispatchID         string      `json:"dispatch_id"`
	Channel            ChannelType `json:"channel"`
	ChannelIDHash      string      `json:"channel_id_hash"`
	Mode               ChannelMode `json:"mode"`
	RiskClass          string      `json:"risk_class"`
	RedactionApplied   bool        `json:"redaction_applied"`
	PartsCount         int         `json:"parts_count"`
	ReviewID           string      `json:"review_id,omitempty"`
	DAGRunID           string      `json:"dag_run_id,omitempty"`
	Outcome            string      `json:"outcome"` // accepted / rejected / ignored / failed / identity_mismatch / permission_mismatch
	ControllerDecision string      `json:"controller_decision,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
}

// ──────────────────────────────────────────────
// 連線測試結果
// ──────────────────────────────────────────────

// ConnectionTestResult 連線測試回傳結構。
type ConnectionTestResult struct {
	Success      bool        `json:"success"`
	Channel      ChannelType `json:"channel"`
	ErrorMessage string      `json:"error_message,omitempty"`
	TestedAt     time.Time   `json:"tested_at"`
}

// ──────────────────────────────────────────────
// 聚合設定（§12A.6）
// ──────────────────────────────────────────────

// AggregationConfig 通知聚合設定。
type AggregationConfig struct {
	AggregateEveryNodes    int  `json:"aggregate_every_nodes"`    // 預設 5
	HeartbeatSeconds       int  `json:"heartbeat_seconds"`        // 預設 30
	MinHeartbeatSeconds    int  `json:"min_heartbeat_seconds"`    // 最低 5
	FastHeartbeatConfirmed bool `json:"fast_heartbeat_confirmed"` // 使用者是否確認快速心跳
}

// DefaultAggregationConfig 回傳預設聚合設定。
func DefaultAggregationConfig() AggregationConfig {
	return AggregationConfig{
		AggregateEveryNodes: 5,
		HeartbeatSeconds:    30,
		MinHeartbeatSeconds: 5,
	}
}
