// remote_bridge/dispatch.go — 通知分發器（§12A.6–12A.9, §12A.11）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 當 Laptop Controller 需要把進度/警告/Review Card 推送到     │
// │ 使用者手機時，由本檔案的 Dispatcher 負責組裝與發送。        │
// │                                                             │
// │ 發送決策流程：                                              │
// │  1. 檢查是否有啟用通道（GetActiveChannel）                  │
// │  2. 檢查通道權限模式是否允許該通知類型（§12A.3）            │
// │  3. 組裝內容（buildContent）                                │
// │     - critical_minimal_alert → 固定文案，不含敏感資訊       │
// │     - 其他 → 依 title/risk/action/nodes 組裝                │
// │  4. 動態縮減（shrinkAndSplit，§12A.9 優先順序）             │
// │     保留：title > risk > action > node title > status > …   │
// │     超長：截斷 details → 仍超過則分段                       │
// │  5. 寫入稽核日誌                                            │
// │  6. 實際 HTTP POST（MVP TODO — 預留位置）                   │
// │                                                             │
// │ 緊急分發（§12A.7）：                                        │
// │  System Warning / Review Card / Stop Recovery /              │
// │  critical_minimal_alert 直接發送，不等聚合週期。             │
// │                                                             │
// │ 各通道訊息長度上限：                                        │
// │  Telegram 4096 / Discord 2000 / LINE 1000                   │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// 分發內容類型
// ──────────────────────────────────────────────

// NotificationType 通知分發的觸發類型。
type NotificationType string

const (
	NotifyProgressUpdate  NotificationType = "progress_update"
	NotifyResultSummary   NotificationType = "result_summary"
	NotifySystemWarning   NotificationType = "system_warning"
	NotifyStopRecovery    NotificationType = "stop_recovery_notice"
	NotifyCriticalMinimal NotificationType = "critical_minimal_alert"
	NotifyReviewCard      NotificationType = "review_card"
	NotifyHeartbeat       NotificationType = "heartbeat"
)

// IsEmergency 判斷是否為緊急分發類型（§12A.7）。
func (n NotificationType) IsEmergency() bool {
	switch n {
	case NotifySystemWarning, NotifyReviewCard, NotifyStopRecovery, NotifyCriticalMinimal:
		return true
	default:
		return false
	}
}

// ──────────────────────────────────────────────
// 分發請求
// ──────────────────────────────────────────────

// DispatchRequest 通知分發請求。
type DispatchRequest struct {
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	RiskClass   string           `json:"risk_class,omitempty"`
	Action      string           `json:"action,omitempty"` // 使用者需執行的動作
	DAGRunID    string           `json:"dag_run_id,omitempty"`
	ReviewID    string           `json:"review_id,omitempty"`
	NodeTitles  []string         `json:"node_titles,omitempty"`
	ShortReason string           `json:"short_reason,omitempty"`
	ErrorCode   string           `json:"error_code,omitempty"`
	Details     string           `json:"details,omitempty"` // 可被截斷的詳細資訊
}

// ──────────────────────────────────────────────
// 分發器
// ──────────────────────────────────────────────

// Dispatcher 負責將通知推送到啟用的通道。
type Dispatcher struct {
	service *Service
}

// NewDispatcher 建立分發器。
func NewDispatcher(service *Service) *Dispatcher {
	return &Dispatcher{service: service}
}

// Dispatch 分發通知到當前啟用的通道。
// 如果通道為 notification_only，只允許接收特定類型的通知。
func (d *Dispatcher) Dispatch(req DispatchRequest) (DispatchRecord, error) {
	active := d.service.GetActiveChannel()
	if active == nil {
		return DispatchRecord{}, fmt.Errorf("no active channel")
	}

	// §12A.3: notification_only 通道的允許類型
	if active.Mode == ModeNotificationOnly {
		allowed := map[NotificationType]bool{
			NotifyProgressUpdate:  true,
			NotifyResultSummary:   true,
			NotifySystemWarning:   true,
			NotifyStopRecovery:    true,
			NotifyCriticalMinimal: true,
		}
		if !allowed[req.Type] {
			return DispatchRecord{}, fmt.Errorf("notification_only mode does not allow %s", req.Type)
		}
	}

	// §12A.11: critical_minimal_alert — 不含敏感細節，僅通知回到筆電
	content := d.buildContent(req, active)

	// §12A.9: 動態縮減
	maxLen := channelMaxLength(active.Channel)
	parts := d.shrinkAndSplit(content, maxLen)

	now := time.Now()
	dispatchID := fmt.Sprintf("disp_%d", now.UnixMilli())

	record := DispatchRecord{
		DispatchID:    dispatchID,
		Channel:       active.Channel,
		ChannelIDHash: hashString(active.ID),
		Mode:          active.Mode,
		RiskClass:     req.RiskClass,
		Redacted:      req.Type == NotifyCriticalMinimal,
		PartsCount:    len(parts),
		ReviewID:      req.ReviewID,
		DAGRunID:      req.DAGRunID,
		ContentHash:   hashContent(content),
		CreatedAt:     now,
	}

	// 稽核記錄
	d.service.auditLog.Append(AuditEntry{
		DispatchID:         dispatchID,
		Channel:            active.Channel,
		ChannelIDHash:      hashString(active.ID),
		Mode:               active.Mode,
		RiskClass:          req.RiskClass,
		RedactionApplied:   record.Redacted,
		PartsCount:         len(parts),
		ReviewID:           req.ReviewID,
		DAGRunID:           req.DAGRunID,
		Outcome:            "dispatched",
		ControllerDecision: fmt.Sprintf("dispatch_%s", req.Type),
		CreatedAt:          now,
	})

	// 透過 generic webhook sender 逐段發送
	for i, part := range parts {
		webhookReq := d.buildWebhookRequest(active, part, i, len(parts))
		resp, err := SendWebhook(webhookReq)
		if err != nil {
			record.ContentHash = hashContent(content)
			return record, fmt.Errorf("webhook send part %d failed: %w", i, err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return record, fmt.Errorf("webhook send part %d: HTTP %d", i, resp.StatusCode)
		}
	}

	return record, nil
}

// buildWebhookRequest 根據啟用通道建構 WebhookRequest。
// 從 credential store 載入 URL，填入 generic sender 格式。
func (d *Dispatcher) buildWebhookRequest(ch *ChannelBinding, content string, partIndex, totalParts int) WebhookRequest {
	bodyContent := content
	if totalParts > 1 {
		bodyContent = fmt.Sprintf("[%d/%d] %s", partIndex+1, totalParts, content)
	}

	customWebhookConfig := ch.CustomWebhookConfig
	if ch.SetupMode == "developer" {
		if raw, err := d.service.secrets.Load(fmt.Sprintf("remote_bridge:%s:custom_config", ch.ID)); err == nil {
			var stored WebhookRequest
			if json.Unmarshal([]byte(raw), &stored) == nil {
				customWebhookConfig = &stored
			}
		}
	}

	if customWebhookConfig != nil {
		req := *customWebhookConfig
		if req.Headers != nil {
			headers := make(map[string]string, len(req.Headers))
			for k, v := range req.Headers {
				headers[k] = v
			}
			req.Headers = headers
		}
		if strings.TrimSpace(req.URL) == "" {
			req.URL, _ = d.service.secrets.Load("remote_bridge:" + ch.ID)
		}
		if strings.TrimSpace(req.Body) == "" {
			req.Body = fmt.Sprintf(`{"text":"%s","part":%d,"total":%d}`, escapeJSON(content), partIndex+1, totalParts)
		} else {
			req.Body = renderWebhookBodyTemplate(req.Body, bodyContent, partIndex, totalParts)
		}
		return req
	}

	if ch.SetupMode == "quick" && ch.PresetID != "" {
		if preset, ok := GetPreset(ch.PresetID); ok {
			fields := map[string]string{}
			for _, field := range preset.RequiredFields {
				if value, err := d.service.secrets.Load(fmt.Sprintf("remote_bridge:%s:%s", ch.ID, field)); err == nil {
					fields[field] = value
				}
			}
			if req, err := BuildWebhookRequest(preset, fields, bodyContent); err == nil {
				return req
			}
		}
	}

	// 從 credential 載入通道 URL/token
	rawURL, _ := d.service.secrets.Load("remote_bridge:" + ch.ID)

	return WebhookRequest{
		URL:            rawURL,
		Method:         "POST",
		Headers:        map[string]string{"Content-Type": "application/json"},
		Body:           fmt.Sprintf(`{"text":"%s","part":%d,"total":%d}`, escapeJSON(content), partIndex+1, totalParts),
		TimeoutSeconds: 10,
	}
}

func renderWebhookBodyTemplate(template string, content string, partIndex, totalParts int) string {
	body := strings.ReplaceAll(template, "{{.Content}}", escapeJSON(content))
	body = strings.ReplaceAll(body, "{{.PartIndex}}", fmt.Sprintf("%d", partIndex+1))
	body = strings.ReplaceAll(body, "{{.TotalParts}}", fmt.Sprintf("%d", totalParts))
	return body
}

// ──────────────────────────────────────────────
// 內容建構
// ──────────────────────────────────────────────

func (d *Dispatcher) buildContent(req DispatchRequest, channel *ChannelBinding) string {
	// §12A.11: critical_minimal_alert 只能包含最低限度資訊
	if req.Type == NotifyCriticalMinimal {
		return "已暫停：偵測到需要你手動處理的高敏感操作。\n請回到筆電確認。"
	}

	content := req.Title
	if req.RiskClass != "" {
		content += fmt.Sprintf("\n風險等級：%s", req.RiskClass)
	}
	if req.Action != "" {
		content += fmt.Sprintf("\n需要操作：%s", req.Action)
	}
	for _, title := range req.NodeTitles {
		content += fmt.Sprintf("\n  ▸ %s", title)
	}
	if req.ShortReason != "" {
		content += fmt.Sprintf("\n原因：%s", req.ShortReason)
	}
	if req.ErrorCode != "" {
		content += fmt.Sprintf("\n錯誤碼：%s", req.ErrorCode)
	}
	if req.Details != "" {
		content += fmt.Sprintf("\n\n%s", req.Details)
	}
	return content
}

// ──────────────────────────────────────────────
// §12A.9 動態縮減與多段分發
// ──────────────────────────────────────────────

// channelMaxLength 取得各通道的訊息長度上限。
func channelMaxLength(channel ChannelType) int {
	switch channel {
	case ChannelTelegram:
		return 4096
	case ChannelDiscord:
		return 2000
	case ChannelLINE:
		return 1000
	case ChannelQQ:
		return 2000
	default:
		return 2000
	}
}

// shrinkAndSplit 按照 §12A.9 優先順序縮減，超過上限則分段。
func (d *Dispatcher) shrinkAndSplit(content string, maxLen int) []string {
	if len(content) <= maxLen {
		return []string{content}
	}

	// 先截斷 details
	truncated := content
	if len(truncated) > maxLen {
		truncated = truncated[:maxLen-3] + "..."
	}

	if len(truncated) <= maxLen {
		return []string{truncated}
	}

	// 分段
	var parts []string
	for len(content) > 0 {
		end := maxLen
		if end > len(content) {
			end = len(content)
		}
		parts = append(parts, content[:end])
		content = content[end:]
	}
	return parts
}

// hashContent 計算內容雜湊。
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("sha256:%x", h[:8])
}
