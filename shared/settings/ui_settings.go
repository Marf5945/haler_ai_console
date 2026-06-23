package settings

// ui_settings.go — v3.3.2 P0.5 UI settings restore defaults contract.
//
// Rules:
//   - UI-panel-related settings are stored separately at data/preferences/ui_settings.json.
//   - RestoreDefaults ONLY resets UI settings and preference ordering.
//   - RestoreDefaults must NOT delete: memory, DAG, installed Skill/MCP,
//     tool registry, review log, personas, or conversation history.
//   - Before applying an agent-supplied custom style, a diff preview must be shown;
//     on failure the previous ui_settings.json must be restored.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ui_console/data/storage"
)

// UISettings holds all UI-only preferences for the Console panel.
type UISettings struct {
	PanelLanguage string    `json:"panel_language"`
	RoleLanguage  string    `json:"role_language"`
	FontPreset    string    `json:"font_preset"`
	FontScale     string    `json:"font_scale"`
	PanelStyle    string    `json:"panel_style"`
	AppIcon       string    `json:"app_icon"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Skill Context Orchestration — 初次使用說明卡已顯示旗標。
	// true = 使用者已看過說明，不再顯示。持久化於 ui_settings.json。
	SkillFirstUseExplained bool `json:"skill_first_use_explained"`
}

// defaultUISettings returns the factory-default UI settings.
func defaultUISettings() UISettings {
	return UISettings{
		PanelLanguage: "繁中",
		RoleLanguage:  "自動",
		FontPreset:    "預設",
		FontScale:     "100%",
		PanelStyle:    "喔黏菊",
		AppIcon:       "default",
	}
}

// UISettingsService manages the isolated UI settings file.
type UISettingsService struct {
	mu      sync.Mutex
	store   *storage.JSONStore[UISettings]
	backup  string // holds previous version for rollback
	current UISettings

	// ── #46 Live Preview 狀態 ──
	livePreviewActive bool        // Live Preview 是否正在進行
	previewTimer      *time.Timer // 600 秒自動回滾計時器
}

// NewUISettingsService creates (or loads) a UI settings service.
func NewUISettingsService(root string) *UISettingsService {
	svc := &UISettingsService{
		store: storage.NewJSONStore[UISettings](
			filepath.Join(root, "data", "preferences", "ui_settings.json"),
		),
		backup:  filepath.Join(root, "data", "preferences", "ui_settings.backup.json"),
		current: defaultUISettings(),
	}
	if loaded, err := svc.store.Load(); err == nil && loaded.PanelLanguage != "" {
		loaded.PanelStyle = normalizeUIStyle(loaded.PanelStyle)
		svc.current = loaded
	} else {
		_ = svc.store.Save(svc.current)
	}
	return svc
}

// Get returns the current UI settings.
func (s *UISettingsService) Get() UISettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// RestoreDefaults resets UI settings and preference ordering to defaults.
// It MUST NOT touch memory, DAG, installed tools, tool registry, review log,
// personas, or conversation history — only ui_settings.json.
func (s *UISettingsService) RestoreDefaults() (UISettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = defaultUISettings()
	s.current.UpdatedAt = time.Now()
	return s.current, s.saveLocked()
}

// ApplyStyleDiff applies an agent-supplied custom style after showing a diff
// preview. diffJSON is the proposed delta; on error the previous settings are
// automatically restored.
//
// The caller (app.go) MUST show the diff preview to the user and receive
// explicit confirmation before calling this method.
func (s *UISettingsService) ApplyStyleDiff(diffJSON string) (UISettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Back up current settings before applying.
	if err := s.writeFileLocked(s.backup, s.current); err != nil {
		return s.current, fmt.Errorf("ui_settings: cannot create backup before applying diff: %w", err)
	}

	var delta map[string]interface{}
	if err := json.Unmarshal([]byte(diffJSON), &delta); err != nil {
		return s.current, fmt.Errorf("ui_settings: invalid diff JSON: %w", err)
	}

	// Apply supported fields from delta.
	if v, ok := delta["panel_language"].(string); ok && v != "" {
		s.current.PanelLanguage = v
	}
	if v, ok := delta["role_language"].(string); ok && v != "" {
		s.current.RoleLanguage = v
	}
	if v, ok := delta["font_preset"].(string); ok && v != "" {
		s.current.FontPreset = v
	}
	if v, ok := delta["font_scale"].(string); ok && v != "" {
		s.current.FontScale = v
	}
	if v, ok := delta["panel_style"].(string); ok && v != "" {
		s.current.PanelStyle = normalizeUIStyle(v)
	}
	if v, ok := delta["app_icon"].(string); ok && v != "" {
		s.current.AppIcon = v
	}
	s.current.UpdatedAt = time.Now()

	if err := s.saveLocked(); err != nil {
		// Rollback to backup.
		_ = s.rollbackLocked()
		return s.current, fmt.Errorf("ui_settings: failed to save; rolled back: %w", err)
	}
	return s.current, nil
}

// RollbackStyle reverts to the most recent backup of ui_settings.json.
func (s *UISettingsService) RollbackStyle() (UISettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current, s.rollbackLocked()
}

// ══════════════════════════════════════════════
// #46 Live Preview — 600 秒自動回滾計時器
// ══════════════════════════════════════════════
//
// Agent 或外部流程寫入自訂風格前，必須透過 Live Preview 流程：
//   1. StartLivePreview(diffJSON) → 立即套用新樣式，啟動 600 秒計時器
//   2. 使用者選擇「保留」→ CommitPreview() 確認套用
//   3. 使用者選擇「復原」→ CancelPreview() 手動回滾
//   4. 600 秒內無操作 → 自動回滾到前一版

const livePreviewTimeout = 600 * time.Second

// StartLivePreview 啟動 Live Preview：套用新樣式並開始 600 秒倒數。
// 600 秒內必須呼叫 CommitPreview 或 CancelPreview，否則自動回滾。
func (s *UISettingsService) StartLivePreview(diffJSON string) (UISettings, error) {
	// 先套用 diff（ApplyStyleDiff 內部會備份）
	result, err := s.ApplyStyleDiff(diffJSON)
	if err != nil {
		return result, err
	}

	s.mu.Lock()
	// 取消之前的計時器（若有）
	if s.previewTimer != nil {
		s.previewTimer.Stop()
	}
	s.livePreviewActive = true

	// 啟動 600 秒自動回滾計時器
	s.previewTimer = time.AfterFunc(livePreviewTimeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.livePreviewActive {
			_ = s.rollbackLocked()
			s.livePreviewActive = false
			s.previewTimer = nil
		}
	})
	s.mu.Unlock()

	return result, nil
}

// CommitPreview 確認保留 Live Preview 的樣式。
// 停止自動回滾計時器，刪除備份。
func (s *UISettingsService) CommitPreview() (UISettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.previewTimer != nil {
		s.previewTimer.Stop()
		s.previewTimer = nil
	}
	s.livePreviewActive = false
	// 刪除備份（不再需要回滾）
	_ = os.Remove(s.backup)
	return s.current, nil
}

// CancelPreview 手動取消 Live Preview，回滾到前一版。
func (s *UISettingsService) CancelPreview() (UISettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.previewTimer != nil {
		s.previewTimer.Stop()
		s.previewTimer = nil
	}
	s.livePreviewActive = false
	return s.current, s.rollbackLocked()
}

// IsLivePreviewActive 回傳 Live Preview 是否正在進行。
func (s *UISettingsService) IsLivePreviewActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.livePreviewActive
}

// MarkSkillFirstUseExplained 將初次使用說明標記為已顯示。
// 前端在使用者關閉說明卡後呼叫此方法，下次啟動不再顯示。
func (s *UISettingsService) MarkSkillFirstUseExplained() (UISettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current.SkillFirstUseExplained = true
	s.current.UpdatedAt = time.Now()
	return s.current, s.saveLocked()
}

// --- internal helpers ---

func (s *UISettingsService) saveLocked() error {
	return s.store.SaveRaw(s.current)
}

func (s *UISettingsService) writeFileLocked(path string, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *UISettingsService) rollbackLocked() error {
	data, err := os.ReadFile(s.backup)
	if err != nil {
		return fmt.Errorf("ui_settings: no backup to roll back to: %w", err)
	}
	var prev UISettings
	if err := json.Unmarshal(data, &prev); err != nil {
		return err
	}
	prev.PanelStyle = normalizeUIStyle(prev.PanelStyle)
	s.current = prev
	return s.saveLocked()
}

func normalizeUIStyle(style string) string {
	if style == "" || style == "預設" || style == "喔黏橘" {
		return "喔黏菊"
	}
	return style
}
