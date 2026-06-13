// persona_avatar/service.go — Avatar 核心服務（§10.1–§10.3）。
// 管理 provider 選擇、狀態映射、config 讀寫。
package persona_avatar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ──────────────────────────────────────────────
// Service 主體
// ──────────────────────────────────────────────

// Service 管理 persona avatar 的讀取與設定。
type Service struct {
	baseDir string // 應用程式根目錄
}

const lockedAvatarPersonaID = "persona-a"

// NewService 建立 avatar 服務。baseDir 為應用程式根目錄。
func NewService(baseDir string) *Service {
	return &Service{baseDir: baseDir}
}

// ──────────────────────────────────────────────
// 路徑輔助
// ──────────────────────────────────────────────

func (s *Service) avatarDir(personaID string) string {
	return filepath.Join(s.baseDir, "data", "personas", personaID, "avatar")
}

func (s *Service) configPath(personaID string) string {
	return filepath.Join(s.avatarDir(personaID), "avatar_config.json")
}

// ──────────────────────────────────────────────
// 讀取設定
// ──────────────────────────────────────────────

// GetCurrentAvatar 取得指定 persona 的目前 avatar 設定。
// 若無設定檔，回傳 pixel fallback 預設值。
func (s *Service) GetCurrentAvatar(personaID string) AvatarConfig {
	if personaID == lockedAvatarPersonaID {
		return s.defaultConfig(personaID)
	}
	data, err := os.ReadFile(s.configPath(personaID))
	if err != nil {
		return s.defaultConfig(personaID)
	}
	var config AvatarConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return s.defaultConfig(personaID)
	}
	config.PixelPack = NormalizePixelPack(config.PersonaID, config.PixelPack)
	return config
}

// defaultConfig 回傳 pixel fallback 預設設定。
func (s *Service) defaultConfig(personaID string) AvatarConfig {
	return AvatarConfig{
		AvatarProvider: ProviderPixel,
		PersonaID:      personaID,
		PixelPack:      DefaultPixelPack(personaID),
		OutputSize:     128,
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}
}

// DefaultPixelPack keeps the built-in personas visually stable: A is the
// wolfdog brand, B is the uncle, and C is the secretary.
func DefaultPixelPack(personaID string) string {
	switch personaID {
	case "persona-b":
		return "uncle"
	case "persona-c":
		return "secretary"
	default:
		return "wolf"
	}
}

func NormalizePixelPack(personaID, pack string) string {
	switch pack {
	case "wolf", "uncle", "secretary":
		return pack
	default:
		return DefaultPixelPack(personaID)
	}
}

// ──────────────────────────────────────────────
// Provider 選擇邏輯（§10.2）
// ──────────────────────────────────────────────

// ResolveProvider 根據目前設定決定實際使用的 provider。
// 優先順序：user_image_api > static_image > built_in_pixel。
func (s *Service) ResolveProvider(personaID string) AvatarProvider {
	config := s.GetCurrentAvatar(personaID)

	// 1. Image API 已設定且有 credential
	if config.AvatarProvider == ProviderImageAPI &&
		config.CredentialRef != "" && config.APIEndpoint != "" {
		return ProviderImageAPI
	}

	// 2. 靜態圖片存在
	if config.StaticAvatarPath != "" {
		staticPath := config.StaticAvatarPath
		if !filepath.IsAbs(staticPath) {
			staticPath = filepath.Join(s.baseDir, staticPath)
		}
		if _, err := os.Stat(staticPath); err == nil {
			return ProviderStaticImage
		}
	}

	// 3. Pixel fallback
	return ProviderPixel
}

// ──────────────────────────────────────────────
// 設定 Provider
// ──────────────────────────────────────────────

// SetProvider 更新 avatar provider。
func (s *Service) SetProvider(personaID string, provider AvatarProvider) error {
	if personaID == lockedAvatarPersonaID {
		return fmt.Errorf("persona avatar is locked: %s", personaID)
	}
	config := s.GetCurrentAvatar(personaID)
	config.AvatarProvider = provider
	config.PixelPack = NormalizePixelPack(personaID, config.PixelPack)
	config.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.saveConfig(personaID, config)
}

// SetPixelPack stores the built-in dynamic avatar pack per persona instead of
// using a process-wide preference that would make A/B/C overwrite each other.
func (s *Service) SetPixelPack(personaID, pack string) error {
	if personaID == lockedAvatarPersonaID {
		return fmt.Errorf("persona avatar is locked: %s", personaID)
	}
	config := s.GetCurrentAvatar(personaID)
	config.AvatarProvider = ProviderPixel
	config.PixelPack = NormalizePixelPack(personaID, pack)
	config.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.saveConfig(personaID, config)
}

// saveConfig 寫入 avatar_config.json。
func (s *Service) saveConfig(personaID string, config AvatarConfig) error {
	dir := s.avatarDir(personaID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("建立 avatar 目錄失敗: %w", err)
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath(personaID), data, 0o600)
}

// ──────────────────────────────────────────────
// 狀態觸發器映射（§10.3 單向）
// ──────────────────────────────────────────────

// GetStateTrigger 根據系統的 task 狀態和風險狀態，
// 回傳對應的 avatar 表情觸發器。
// 這是單向映射：系統狀態 → avatar 表情。
// 回傳值不得回寫到任何系統狀態（§10.3 禁止）。
func GetStateTrigger(taskState, riskState string) AvatarStateTrigger {
	// 風險狀態優先（安全相關的表情最重要）
	switch riskState {
	case "critical_runtime_action", "security_boundary_rewrite":
		return StateBlocked
	case "user_owned_asset_destructive", "subagent_lifecycle_removal":
		return StateWarning
	case "high_non_destructive":
		return StateWarning
	}

	// 任務狀態
	switch taskState {
	case "running", "executing", "tool_running", "testing", "editing", "building", "starting_service":
		return StateWorking
	case "thinking", "planning", "llm_waiting", "analyzing_files", "learning":
		return StateThinking
	case "completed", "success", "test_passed", "build_success", "discord_sent", "praised":
		return StateHappy
	case "idle", "waiting":
		return StateIdle
	case "sleeping", "inactive", "app_minimized", "fresh_window":
		return StateSleepy
	case "sad", "failed", "build_failed", "test_failed", "cancelled", "llm_error", "api_error", "major_failure", "scolded":
		return StateSad
	case "speechless", "Speechless", "empty_message", "unknown_command", "joke":
		return StateSpeechless
	case "blocked", "permission_required", "credential_missing", "discord_auth_failed", "confirmation_required":
		return StateBlocked
	case "low_risk_confirmation", "missing_config", "adapter_degraded", "unverified_source":
		return StateWarning
	}

	return StateIdle
}
