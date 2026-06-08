// review/card.go — v3.6.1 完整 Review Card 定義（§5.1）。
// 從 service.go 拆出 Card 結構，擴充所有 spec 必要欄位。
package review

import (
	"ui_console/domain/risk"
)

// ──────────────────────────────────────────────
// Standard Review Card（§5.1）
// ──────────────────────────────────────────────

// Card 是 v3.6.1 完整的 Review Card 結構。
// 包含所有 §5.1 必要欄位 + 向下相容的 Level 欄位。
type Card struct {
	// 唯一識別碼
	ID        string `json:"review_id"`
	CreatedAt string `json:"created_at"`

	// 風險分類（v3.6.1 七級）
	RiskClass risk.RiskClass `json:"risk_class"`

	// 操作描述
	Operation string `json:"operation"`
	Target    string `json:"target"`
	Reason    string `json:"reason"` // 使用者可讀的理由

	// 按鈕標籤
	AcceptLabel string `json:"accept_label"`
	RejectLabel string `json:"reject_label"`

	// 後果說明
	AcceptEffect string `json:"accept_effect"`
	RejectEffect string `json:"reject_effect"`

	// 復原與備份
	RollbackAvailable bool `json:"rollback_available"`
	BackupAvailable   bool `json:"backup_available"`

	// 雙步驟確認（§4.6 v3.6.1）
	// 僅 security_boundary_rewrite 為 true
	RequiresDualStep bool `json:"requires_dual_step"`
	CooldownSeconds  int  `json:"cooldown_seconds"` // 預設 3 秒

	// 來源資訊（向下相容）
	SourceType string `json:"source_type,omitempty"`
	SourceID   string `json:"source_id,omitempty"`

	// 工程細節（僅在「展開」時顯示）
	EngineerReason string `json:"engineer_reason,omitempty"`
	LogLocation    string `json:"log_location,omitempty"`

	// 狀態
	Resolved   bool   `json:"resolved"`
	ResolvedAt string `json:"resolved_at,omitempty"`

	// 雙步驟狀態追蹤
	DualStepState *DualStepState `json:"dual_step_state,omitempty"`

	// 向下相容：舊的 Level 欄位映射
	LegacyLevel Level `json:"level,omitempty"`
}

// ──────────────────────────────────────────────
// 雙步驟確認狀態（§4.6）
// ──────────────────────────────────────────────

// DualStepState 追蹤雙步驟確認的進度。
type DualStepState struct {
	// Step 1: 使用者點擊「我了解，繼續」的時間
	Step1ConfirmedAt string `json:"step1_confirmed_at,omitempty"`

	// Step 2: 使用者長按「執行」的時間
	Step2ExecutedAt string `json:"step2_executed_at,omitempty"`

	// 驗證用的 hash（controller 在 Step 2 時比對）
	ReviewIDAtStep1  string `json:"review_id_at_step1,omitempty"`
	ScopeHashAtStep1 string `json:"scope_hash_at_step1,omitempty"`
	RiskPolicyHash   string `json:"risk_policy_hash,omitempty"`
	ToolRegistryHash string `json:"tool_registry_hash,omitempty"`
	TargetHashSet    string `json:"target_hash_set,omitempty"`

	// 是否已失效（hash 不一致時設為 true）
	Invalidated bool `json:"invalidated,omitempty"`
}

// ──────────────────────────────────────────────
// Card 建構輔助
// ──────────────────────────────────────────────

// CardParams 建立 Card 時的必要參數。
type CardParams struct {
	RiskClass      risk.RiskClass
	Operation      string
	Target         string
	Reason         string
	AcceptLabel    string
	RejectLabel    string
	AcceptEffect   string
	RejectEffect   string
	RollbackAvail  bool
	BackupAvail    bool
	SourceType     string
	SourceID       string
	EngineerReason string
	LogLocation    string
}

// NewCard 根據參數建立完整的 Review Card。
// 自動根據 RiskClass 設定 RequiresDualStep 和 CooldownSeconds。
func NewCard(params CardParams) Card {
	requiresDual := risk.RequiresDualStep(params.RiskClass)
	cooldown := 0
	if requiresDual {
		cooldown = 3 // §4.6: 3 秒冷卻
	}

	return Card{
		ID:                generateID("rev"),
		CreatedAt:         nowRFC3339(),
		RiskClass:         params.RiskClass,
		Operation:         params.Operation,
		Target:            params.Target,
		Reason:            params.Reason,
		AcceptLabel:       params.AcceptLabel,
		RejectLabel:       params.RejectLabel,
		AcceptEffect:      params.AcceptEffect,
		RejectEffect:      params.RejectEffect,
		RollbackAvailable: params.RollbackAvail,
		BackupAvailable:   params.BackupAvail,
		RequiresDualStep:  requiresDual,
		CooldownSeconds:   cooldown,
		SourceType:        params.SourceType,
		SourceID:          params.SourceID,
		EngineerReason:    params.EngineerReason,
		LogLocation:       params.LogLocation,
		Resolved:          false,
		LegacyLevel:       riskToLegacyLevel(params.RiskClass),
	}
}

// riskToLegacyLevel 將 v3.6.1 風險等級映射到舊的三級 Level（向下相容）。
func riskToLegacyLevel(rc risk.RiskClass) Level {
	switch rc {
	case risk.Low:
		return LevelBackground
	case risk.Medium:
		return LevelPending
	default:
		return LevelBlocking
	}
}
