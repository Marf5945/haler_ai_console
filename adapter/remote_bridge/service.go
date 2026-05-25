// remote_bridge/service.go — Remote Bridge 主服務（§12A）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 本檔案是 Remote Bridge 模組的「大腦」，統合所有子元件：      │
// │  • detector.go   → URL 辨識                                │
// │  • connection.go → 連線測試                                │
// │  • credential.go → 憑證安全儲存                            │
// │  • audit.go      → 稽核日誌                                │
// │  • dispatch.go   → 通知分發（聚合/緊急/多段）              │
// │                                                             │
// │ 核心原則（§12A.1）：                                        │
// │  「手機可以遞紙條。筆電 Controller 才能審紙條、開門、派工具」│
// │                                                             │
// │ Wails binding 暴露方法（前端透過 app.go 代理呼叫）：        │
// │  • DetectChannelFromURL   — 純辨識，不改狀態                │
// │  • TestChannelConnection  — 偵測+連線測試                   │
// │  • RegisterChannel        — 寫入綁定+憑證+稽核              │
// │  • ActivateChannel        — 啟用（自動停用其他，單一限制）   │
// │  • DeactivateChannel      — 停用                            │
// │  • SwitchMode             — 切換 notification_only /        │
// │                             remote_task_submit /            │
// │                             remote_review                  │
// │  • ListChannels           — 列出所有未撤銷通道              │
// │  • GetActiveChannel       — 取得唯一啟用通道                │
// │  • RemoveChannel          — 撤銷+清除憑證                   │
// │  • GetRecentAudit         — 稽核記錄查詢                    │
// │                                                             │
// │ 持久化：remote_bridge_bindings.json（通道綁定清單）         │
// │ 每次異動都先 load() → 修改 → save() + audit.Append()        │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ui_console/domain/credential"
)

// ──────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────

const bindingsFile = "remote_bridge_bindings.json"

// Service 是 Remote Bridge Communication 的核心服務。
// TASKS_1_6_3 Step 3：credentialStore → secrets（注入共用 SecretStore）。
type Service struct {
	mu          sync.Mutex
	projectRoot string
	bindings    []ChannelBinding
	loaded      bool
	secrets     credential.SecretStore // 統一加密憑證儲存（注入）
	auditLog    *AuditLog
	tester      *ConnectionTester
	aggregation AggregationConfig
}

// NewService 建立 Remote Bridge 服務。
// secrets 由外部注入（domain/credential.Store）。
func NewService(projectRoot string, secrets credential.SecretStore) *Service {
	return &Service{
		projectRoot: projectRoot,
		secrets:     secrets,
		auditLog:    NewAuditLog(projectRoot),
		tester:      NewConnectionTester(),
		aggregation: DefaultAggregationConfig(),
	}
}

// ──────────────────────────────────────────────
// 持久化
// ──────────────────────────────────────────────

func (s *Service) bindingsPath() string {
	return filepath.Join(s.projectRoot, "remote_bridge", bindingsFile)
}

func (s *Service) load() error {
	if s.loaded {
		return nil
	}
	data, err := os.ReadFile(s.bindingsPath())
	if os.IsNotExist(err) {
		s.bindings = []ChannelBinding{}
		s.loaded = true
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		s.bindings = []ChannelBinding{}
		s.loaded = true
		return nil
	}
	if err := json.Unmarshal(data, &s.bindings); err != nil {
		s.bindings = []ChannelBinding{}
	}
	s.loaded = true
	return nil
}

func (s *Service) save() error {
	dir := filepath.Dir(s.bindingsPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.bindings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.bindingsPath(), data, 0600)
}

// ──────────────────────────────────────────────
// 通道偵測（前端呼叫）
// ──────────────────────────────────────────────

// DetectChannelFromURL 從 URL 偵測通道類型（Wails binding）。
func (s *Service) DetectChannelFromURL(rawURL string) DetectResult {
	return DetectChannel(rawURL)
}

// ──────────────────────────────────────────────
// 連線測試（前端呼叫）
// ──────────────────────────────────────────────

// TestChannelConnection 對指定 URL 執行連線測試（Wails binding）。
func (s *Service) TestChannelConnection(rawURL string) ConnectionTestResult {
	result := DetectChannel(rawURL)
	if !result.Matched {
		return ConnectionTestResult{
			Success:      false,
			ErrorMessage: "無法辨識通訊軟體類型，請確認 URL 格式",
			TestedAt:     time.Now(),
		}
	}
	return s.tester.TestConnection(rawURL, result.Channel)
}

// ──────────────────────────────────────────────
// 通道註冊
// ──────────────────────────────────────────────

// RegisterChannelWithMode 以指定模式註冊通道（§12A.2 使用者分流）。
// setupMode: "quick" 或 "developer"
// Quick Mode：presetID + fields（只填 RequiredFields）
// Developer Mode：完整 WebhookRequest 自訂設定
func (s *Service) RegisterChannelWithMode(setupMode, presetID string, fields map[string]string, customConfig *WebhookRequest) (ChannelBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return ChannelBinding{}, fmt.Errorf("load bindings: %w", err)
	}

	var channelType ChannelType
	if setupMode == "quick" {
		// Quick Mode：使用 preset
		preset, ok := GetPreset(presetID)
		if !ok {
			return ChannelBinding{}, fmt.Errorf("unknown preset: %s", presetID)
		}
		channelType = ChannelType(preset.PlatformID)

		// 驗證必填欄位
		for _, f := range preset.RequiredFields {
			if _, exists := fields[f]; !exists {
				return ChannelBinding{}, fmt.Errorf("missing required field: %s", f)
			}
		}
	} else if setupMode == "developer" {
		// Developer Mode：完整自訂
		if customConfig == nil || customConfig.URL == "" {
			return ChannelBinding{}, fmt.Errorf("developer mode requires custom webhook config with URL")
		}
		channelType = "custom"
	} else {
		return ChannelBinding{}, fmt.Errorf("invalid setup_mode: %s (must be 'quick' or 'developer')", setupMode)
	}

	// 檢查同類型通道衝突（custom 允許多個）
	if channelType != "custom" {
		for _, b := range s.bindings {
			if b.Channel == channelType && !b.Revoked {
				return ChannelBinding{}, fmt.Errorf("已存在 %s 通道，請先移除", channelType.Label())
			}
		}
	}

	now := time.Now()
	expiry := now.Add(30 * 24 * time.Hour)
	var persistedCustomConfig *WebhookRequest
	if setupMode == "developer" && customConfig != nil {
		persistedCustomConfig = &WebhookRequest{
			Method:         customConfig.Method,
			TimeoutSeconds: customConfig.TimeoutSeconds,
			DevMode:        customConfig.DevMode,
		}
	}
	binding := ChannelBinding{
		ID:                  fmt.Sprintf("rb_%s_%d", channelType, now.UnixMilli()),
		DisplayName:         defaultChannelDisplayName(channelType),
		Channel:             channelType,
		Mode:                ModeNotificationOnly,
		Active:              false,
		MaxRemoteRisk:       "high_non_destructive",
		ExpiresAt:           &expiry,
		CreatedAt:           now,
		TestedAt:            &now,
		TestPassed:          true,
		SetupMode:           setupMode,
		PresetID:            presetID,
		CustomWebhookConfig: persistedCustomConfig,
	}

	// 儲存敏感欄位到 device-local secret
	if setupMode == "quick" {
		for k, v := range fields {
			if err := s.secrets.Store(fmt.Sprintf("remote_bridge:%s:%s", binding.ID, k), v); err != nil {
				return ChannelBinding{}, fmt.Errorf("store field secret: %w", err)
			}
		}
	} else {
		// Developer mode：完整自訂 webhook config 可能包含 URL/header token，只存 SecretStore。
		configRaw, err := json.Marshal(customConfig)
		if err != nil {
			return ChannelBinding{}, fmt.Errorf("marshal custom webhook config: %w", err)
		}
		if err := s.secrets.Store("remote_bridge:"+binding.ID+":custom_config", string(configRaw)); err != nil {
			return ChannelBinding{}, fmt.Errorf("store custom webhook config: %w", err)
		}
		// 保留舊 ref 供向後相容與純 URL fallback 使用。
		if err := s.secrets.Store("remote_bridge:"+binding.ID, customConfig.URL); err != nil {
			return ChannelBinding{}, fmt.Errorf("store credential: %w", err)
		}
	}

	s.bindings = append(s.bindings, binding)
	if err := s.save(); err != nil {
		return ChannelBinding{}, fmt.Errorf("save bindings: %w", err)
	}

	s.auditLog.Append(AuditEntry{
		DispatchID:         binding.ID,
		Channel:            binding.Channel,
		ChannelIDHash:      hashString(binding.ID),
		Mode:               binding.Mode,
		Outcome:            "registered",
		ControllerDecision: fmt.Sprintf("channel_registered_%s_%s", setupMode, presetID),
		CreatedAt:          now,
	})

	return binding, nil
}

// RegisterChannel 註冊一個通道（URL 已通過連線測試後呼叫）。
// 同一通道類型只允許一個綁定。（向後相容舊 API）
func (s *Service) RegisterChannel(rawURL string) (ChannelBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return ChannelBinding{}, fmt.Errorf("load bindings: %w", err)
	}

	detected := DetectChannel(rawURL)
	if !detected.Matched {
		return ChannelBinding{}, fmt.Errorf("無法辨識通訊軟體類型")
	}

	// 檢查是否已有同類型通道
	for _, b := range s.bindings {
		if b.Channel == detected.Channel && !b.Revoked {
			return ChannelBinding{}, fmt.Errorf("已存在 %s 通道，請先移除", detected.Channel.Label())
		}
	}

	now := time.Now()
	expiry := now.Add(30 * 24 * time.Hour) // 預設 30 天到期
	binding := ChannelBinding{
		ID:            fmt.Sprintf("rb_%s_%d", detected.Channel, now.UnixMilli()),
		DisplayName:   defaultChannelDisplayName(detected.Channel),
		Channel:       detected.Channel,
		Mode:          ModeNotificationOnly, // §12A.2：所有通道預設 notification_only
		Active:        false,
		MaxRemoteRisk: "high_non_destructive",
		ExpiresAt:     &expiry,
		CreatedAt:     now,
		TestedAt:      &now,
		TestPassed:    true,
	}

	// 儲存憑證到本機（namespaced ref）
	if err := s.secrets.Store("remote_bridge:"+binding.ID, rawURL); err != nil {
		return ChannelBinding{}, fmt.Errorf("store credential: %w", err)
	}

	s.bindings = append(s.bindings, binding)
	if err := s.save(); err != nil {
		return ChannelBinding{}, fmt.Errorf("save bindings: %w", err)
	}

	// 稽核
	s.auditLog.Append(AuditEntry{
		DispatchID:         binding.ID,
		Channel:            binding.Channel,
		ChannelIDHash:      hashString(binding.ID),
		Mode:               binding.Mode,
		Outcome:            "registered",
		ControllerDecision: "channel_registered",
		CreatedAt:          now,
	})

	return binding, nil
}

// RenameChannel 更新通道顯示名稱。
func (s *Service) RenameChannel(channelID string, displayName string) (ChannelBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return ChannelBinding{}, err
	}

	name := normalizeChannelDisplayName(displayName)
	for i := range s.bindings {
		if s.bindings[i].ID == channelID && !s.bindings[i].Revoked {
			s.bindings[i].DisplayName = name
			if err := s.save(); err != nil {
				return ChannelBinding{}, err
			}
			s.auditLog.Append(AuditEntry{
				DispatchID:         s.bindings[i].ID,
				Channel:            s.bindings[i].Channel,
				ChannelIDHash:      hashString(s.bindings[i].ID),
				Mode:               s.bindings[i].Mode,
				Outcome:            "renamed",
				ControllerDecision: "channel_renamed",
				CreatedAt:          time.Now(),
			})
			return s.bindings[i], nil
		}
	}
	return ChannelBinding{}, fmt.Errorf("channel %s not found", channelID)
}

func defaultChannelDisplayName(channel ChannelType) string {
	if channel == "custom" {
		return "自訂 Webhook"
	}
	return channel.Label()
}

func normalizeChannelDisplayName(displayName string) string {
	name := strings.TrimSpace(displayName)
	if name == "" {
		return "未命名通訊"
	}
	return name
}

// ──────────────────────────────────────────────
// 啟用 / 停用（§12A 單一通道啟用限制）
// ──────────────────────────────────────────────

// ActivateChannel 啟用指定通道。主頻道由 SetPrimaryChannel 獨立管理。
func (s *Service) ActivateChannel(channelID string) (ChannelBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return ChannelBinding{}, err
	}

	var target *ChannelBinding
	for i := range s.bindings {
		if s.bindings[i].ID == channelID {
			target = &s.bindings[i]
		}
	}

	if target == nil {
		return ChannelBinding{}, fmt.Errorf("channel %s not found", channelID)
	}
	if !target.IsUsable() {
		return ChannelBinding{}, fmt.Errorf("channel %s is not usable (revoked/expired/untested)", channelID)
	}

	target.Active = true

	if err := s.save(); err != nil {
		return ChannelBinding{}, err
	}

	s.auditLog.Append(AuditEntry{
		DispatchID:         target.ID,
		Channel:            target.Channel,
		ChannelIDHash:      hashString(target.ID),
		Mode:               target.Mode,
		Outcome:            "activated",
		ControllerDecision: "channel_activated",
		CreatedAt:          time.Now(),
	})

	return *target, nil
}

// SetPrimaryChannel marks one enabled channel as the single primary channel.
func (s *Service) SetPrimaryChannel(channelID string) (ChannelBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return ChannelBinding{}, err
	}

	var target *ChannelBinding
	for i := range s.bindings {
		if s.bindings[i].ID == channelID {
			target = &s.bindings[i]
		} else {
			s.bindings[i].Primary = false
		}
	}
	if target == nil {
		return ChannelBinding{}, fmt.Errorf("channel %s not found", channelID)
	}
	if !target.IsUsable() {
		return ChannelBinding{}, fmt.Errorf("channel %s is not usable (revoked/expired/untested)", channelID)
	}
	target.Active = true
	target.Primary = true
	if err := s.save(); err != nil {
		return ChannelBinding{}, err
	}
	s.auditLog.Append(AuditEntry{
		DispatchID:         target.ID,
		Channel:            target.Channel,
		ChannelIDHash:      hashString(target.ID),
		Mode:               target.Mode,
		Outcome:            "primary_selected",
		ControllerDecision: "channel_primary_selected",
		CreatedAt:          time.Now(),
	})
	return *target, nil
}

// DeactivateChannel 停用指定通道。
func (s *Service) DeactivateChannel(channelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return err
	}

	for i := range s.bindings {
		if s.bindings[i].ID == channelID {
			s.bindings[i].Active = false
			s.bindings[i].Primary = false
			s.auditLog.Append(AuditEntry{
				DispatchID:         s.bindings[i].ID,
				Channel:            s.bindings[i].Channel,
				ChannelIDHash:      hashString(s.bindings[i].ID),
				Mode:               s.bindings[i].Mode,
				Outcome:            "deactivated",
				ControllerDecision: "channel_deactivated",
				CreatedAt:          time.Now(),
			})
			return s.save()
		}
	}
	return fmt.Errorf("channel %s not found", channelID)
}

// ──────────────────────────────────────────────
// 模式切換（§12A.3）
// ──────────────────────────────────────────────

// SwitchMode 切換通道權限模式。
func (s *Service) SwitchMode(channelID string, mode ChannelMode) (ChannelBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return ChannelBinding{}, err
	}

	// 驗證模式合法性
	validMode := false
	for _, m := range AllModes() {
		if m == mode {
			validMode = true
			break
		}
	}
	if !validMode {
		return ChannelBinding{}, fmt.Errorf("invalid mode: %s", mode)
	}

	for i := range s.bindings {
		if s.bindings[i].ID == channelID {
			oldMode := s.bindings[i].Mode
			s.bindings[i].Mode = mode

			if err := s.save(); err != nil {
				return ChannelBinding{}, err
			}

			s.auditLog.Append(AuditEntry{
				DispatchID:         s.bindings[i].ID,
				Channel:            s.bindings[i].Channel,
				ChannelIDHash:      hashString(s.bindings[i].ID),
				Mode:               mode,
				Outcome:            "mode_switched",
				ControllerDecision: fmt.Sprintf("%s→%s", oldMode, mode),
				CreatedAt:          time.Now(),
			})

			return s.bindings[i], nil
		}
	}
	return ChannelBinding{}, fmt.Errorf("channel %s not found", channelID)
}

// ──────────────────────────────────────────────
// 查詢
// ──────────────────────────────────────────────

// ListChannels 列出所有已註冊通道（Wails binding）。
func (s *Service) ListChannels() []ChannelBinding {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return nil
	}

	// 過濾已撤銷的
	var result []ChannelBinding
	for _, b := range s.bindings {
		if !b.Revoked {
			result = append(result, b)
		}
	}
	return result
}

// GetActiveChannel 取得主頻道；若尚未指定，fallback 到第一個啟用通道。
func (s *Service) GetActiveChannel() *ChannelBinding {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return nil
	}
	for _, b := range s.bindings {
		if b.Primary && b.Active && !b.Revoked && !b.IsExpired() {
			copied := b
			return &copied
		}
	}
	for _, b := range s.bindings {
		if b.Active && !b.Revoked && !b.IsExpired() {
			copied := b
			return &copied
		}
	}
	return nil
}

// GetChannelByID 取得指定通道。測試傳送等明確指定場景不依賴目前啟用通道。
func (s *Service) GetChannelByID(channelID string) *ChannelBinding {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return nil
	}
	for _, b := range s.bindings {
		if b.ID == channelID && !b.Revoked && !b.IsExpired() {
			copied := b
			return &copied
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// 移除通道
// ──────────────────────────────────────────────

// RemoveChannel 撤銷並移除通道。
func (s *Service) RemoveChannel(channelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.load(); err != nil {
		return err
	}

	found := false
	for i := range s.bindings {
		if s.bindings[i].ID == channelID {
			s.bindings[i].Revoked = true
			s.bindings[i].Active = false
			s.bindings[i].Primary = false
			found = true

			s.auditLog.Append(AuditEntry{
				DispatchID:         s.bindings[i].ID,
				Channel:            s.bindings[i].Channel,
				ChannelIDHash:      hashString(s.bindings[i].ID),
				Mode:               s.bindings[i].Mode,
				Outcome:            "removed",
				ControllerDecision: "channel_revoked",
				CreatedAt:          time.Now(),
			})
			break
		}
	}
	if !found {
		return fmt.Errorf("channel %s not found", channelID)
	}

	// 移除本機憑證（namespaced ref）
	s.secrets.Delete("remote_bridge:" + channelID)
	for _, field := range []string{
		"bot_token", "chat_id", "webhook_url", "channel_access_token",
		"recipient_id", "channel_secret", "bot_app_id", "channel_id",
	} {
		_ = s.secrets.Delete(fmt.Sprintf("remote_bridge:%s:%s", channelID, field))
	}

	return s.save()
}

// ──────────────────────────────────────────────
// 稽核查詢
// ──────────────────────────────────────────────

// GetRecentAudit 取得最近 N 筆稽核記錄（Wails binding）。
func (s *Service) GetRecentAudit(n int) []AuditEntry {
	entries, err := s.auditLog.ReadRecent(n)
	if err != nil {
		return nil
	}
	return entries
}

// ──────────────────────────────────────────────
// 輔助
// ──────────────────────────────────────────────

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("sha256:%x", h[:8]) // 前 8 bytes 足夠用於日誌辨識
}
