// job.go — 排程工作（Job）與執行紀錄（JobExecution）的資料結構定義。
//
// 設計原則：
//   - 零外部依賴：僅使用 Go 標準函式庫
//   - UUID 透過 crypto/rand 產生，無需第三方套件
//   - 時間欄位一律使用 RFC 3339 格式字串，便於 JSON 序列化
package scheduler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// --------------------------------------------------------------------------
// ActionType — 工作觸發時執行的動作類別
// --------------------------------------------------------------------------

// ActionType 定義排程觸發後要執行的動作種類。
type ActionType string

const (
	// ActionEvent 表示發送事件通知。
	ActionEvent ActionType = "event"

	// ActionSkill 表示呼叫技能模組。
	ActionSkill ActionType = "skill"

	// ActionCallback 表示執行回呼函式。
	ActionCallback ActionType = "callback"
)

// --------------------------------------------------------------------------
// Job — 排程工作定義
// --------------------------------------------------------------------------

// Job 描述一個排程工作的完整狀態。
// 所有時間欄位均以 RFC 3339 字串儲存，方便直接進行 JSON 序列化。
type Job struct {
	// ID 為工作的唯一識別碼（UUID v4 格式）。
	ID string `json:"id"`

	// Name 為工作的顯示名稱，供使用者辨識用途。
	Name string `json:"name"`

	// CronExpr 為原始 cron 表達式字串（五欄位格式或快捷別名）。
	CronExpr string `json:"cron_expr"`

	// Enabled 表示此工作是否啟用。停用的工作不會被排程器觸發。
	Enabled bool `json:"enabled"`

	// ActionType 指定觸發時要執行的動作類別。
	ActionType ActionType `json:"action_type"`

	// ActionPayload 為動作的 JSON 酬載，內容依 ActionType 而異。
	ActionPayload string `json:"action_payload"`

	// LastFired 為上次觸發時間（RFC 3339 格式）。若從未觸發過則為空字串。
	LastFired string `json:"last_fired"`

	// NextFire 為下次預計觸發時間（RFC 3339 格式）。
	NextFire string `json:"next_fire"`

	// CreatedAt 為工作建立時間（RFC 3339 格式）。
	CreatedAt string `json:"created_at"`

	// ConsecutiveFailures 為連續失敗次數。成功執行後會重設為 0。
	ConsecutiveFailures int `json:"consecutive_failures"`

	// RiskClass 由 risk.ClassifyOperation 自動算出
	RiskClass string `json:"risk_class"`

	// PayloadHash 為 sha256(ActionPayload)
	PayloadHash string `json:"payload_hash"`

	// ProjectID 建立時所屬專案
	ProjectID string `json:"project_id"`
}

// --------------------------------------------------------------------------
// JobExecution — 單次執行紀錄
// --------------------------------------------------------------------------

// 執行狀態常數
const (
	ExecStatusSuccess   = "success"
	ExecStatusFailed    = "failed"
	ExecStatusSkipped   = "skipped"
	ExecStatusCancelled = "cancelled_by_user"
)

// JobExecution 記錄一次排程工作執行的結果。
type JobExecution struct {
	// JobID 為對應工作的 UUID。
	JobID string `json:"job_id"`

	// FiredAt 為實際觸發時間（RFC 3339 格式）。
	FiredAt string `json:"fired_at"`

	// Duration 為執行耗時（毫秒）。
	Duration int64 `json:"duration_ms"`

	// Status 為執行結果狀態：success / failed / skipped / cancelled_by_user
	Status string `json:"status"`

	// Error 為錯誤訊息。成功時為空字串，序列化時省略。
	Error string `json:"error,omitempty"`

	// Retried 表示此次執行是否為重試。
	Retried bool `json:"retried"`
}

// --------------------------------------------------------------------------
// NewJob — 建立新的排程工作
// --------------------------------------------------------------------------

// NewJob 建立一個新的排程工作。
//
// 參數：
//   - name:          工作顯示名稱
//   - cronExpr:      cron 表達式（五欄位格式或快捷別名）
//   - actionType:    動作類別（event / skill / callback）
//   - actionPayload: 動作的 JSON 酬載
//   - riskClass:     風險等級（由呼叫端傳入）
//   - projectID:     所屬專案 ID
//
// 此函式會：
//  1. 使用 crypto/rand 產生 UUID v4 作為工作 ID
//  2. 透過 ParseCron 驗證 cron 表達式的合法性
//  3. 以當前時間設定 CreatedAt，並計算 NextFire
//  4. 自動計算 PayloadHash
//  5. 預設啟用工作（Enabled = true）
//
// 若 cron 表達式不合法或 UUID 產生失敗，將回傳錯誤。
func NewJob(name, cronExpr string, actionType ActionType, actionPayload string, optional ...string) (*Job, error) {
	if !IsValidActionType(actionType) {
		return nil, fmt.Errorf("scheduler: 不支援的動作類型: %q", actionType)
	}

	// 產生 UUID v4
	id, err := generateUUID()
	if err != nil {
		return nil, fmt.Errorf("scheduler: 產生 UUID 失敗: %w", err)
	}

	// 解析並驗證 cron 表達式
	parsed, err := ParseCron(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("scheduler: cron 表達式無效: %w", err)
	}

	now := time.Now()
	riskClass := ""
	projectID := ""
	if len(optional) > 0 {
		riskClass = optional[0]
	}
	if len(optional) > 1 {
		projectID = optional[1]
	}

	// 計算下次觸發時間
	nextFire := parsed.NextAfter(now)
	var nextFireStr string
	if !nextFire.IsZero() {
		nextFireStr = nextFire.Format(time.RFC3339)
	}

	return &Job{
		ID:            id,
		Name:          name,
		CronExpr:      cronExpr,
		Enabled:       true,
		ActionType:    actionType,
		ActionPayload: actionPayload,
		LastFired:     "",
		NextFire:      nextFireStr,
		CreatedAt:     now.Format(time.RFC3339),
		RiskClass:     riskClass,
		PayloadHash:   computePayloadHash(actionPayload),
		ProjectID:     projectID,
	}, nil
}

// IsValidActionType reports whether actionType is one of the scheduler's
// supported action kinds.
func IsValidActionType(actionType ActionType) bool {
	switch actionType {
	case ActionEvent, ActionSkill, ActionCallback:
		return true
	default:
		return false
	}
}

// --------------------------------------------------------------------------
// computePayloadHash — 計算 ActionPayload 的 SHA-256 雜湊
// --------------------------------------------------------------------------

// computePayloadHash 回傳 payload 的 SHA-256 hex 字串。
func computePayloadHash(payload string) string {
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

// --------------------------------------------------------------------------
// generateUUID — 使用 crypto/rand 產生 UUID v4
// --------------------------------------------------------------------------

// generateUUID 產生符合 RFC 4122 的 UUID v4（隨機型）。
// 使用 crypto/rand 作為隨機來源，確保加密安全等級的隨機性。
//
// UUID v4 格式：xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
//   - 第 13 字元固定為 '4'（版本號）
//   - 第 17 字元的高兩位元固定為 '10'（變體標記）
func generateUUID() (string, error) {
	// UUID 為 128 位元 = 16 位元組
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		return "", fmt.Errorf("讀取隨機來源失敗: %w", err)
	}

	// 設定版本號（第 7 位元組的高四位元 = 0100，即版本 4）
	uuid[6] = (uuid[6] & 0x0f) | 0x40

	// 設定變體標記（第 9 位元組的高兩位元 = 10）
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	// 格式化為標準 UUID 字串：8-4-4-4-12
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16],
	), nil
}
