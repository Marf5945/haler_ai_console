// remote_bridge/telegram_onboarding.go — Telegram 快速模式 Chat ID 自動偵測（§12A.2）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ Telegram 專屬的引導流程，自動偵測 chat_id。                 │
// │ 此 helper 不進入 generic sender core。                      │
// │                                                             │
// │ 流程：                                                      │
// │  1. 使用者貼 Bot Token                                     │
// │  2. getMe 驗證 token 有效                                  │
// │  3. 前端提示使用者到 Telegram 對 bot 輸入 /start           │
// │  4. 輪詢 getUpdates ���選 /start message                   │
// │  5. 找到候選 chat_id → 使用者確認                          │
// │  6. 存入 device-local secret                               │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ui_console/domain/credential"
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// OnboardingSession 代表一次 Telegram onboarding 會話。
type OnboardingSession struct {
	BotUsername string `json:"bot_username"`
	BotID       int64  `json:"bot_id"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error,omitempty"`
}

// ChatCandidate 是 getUpdates 中找到的候選 chat。
type ChatCandidate struct {
	ChatID    int64  `json:"chat_id"`
	ChatTitle string `json:"chat_title"` // 群組名 or 使用者名
	ChatType  string `json:"chat_type"`  // private / group / supergroup
	Username  string `json:"username"`
}

// ──────────────────────────────────────────────
// Onboarding 核心邏輯
// ──────────────────────────────────────────────

// StartTelegramOnboarding 驗證 bot token 有效性（呼叫 getMe）。
func StartTelegramOnboarding(botToken string) (OnboardingSession, error) {
	if botToken == "" {
		return OnboardingSession{Error: "bot token 不能為空"}, fmt.Errorf("empty bot token")
	}

	// 呼叫 Telegram getMe API
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", botToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return OnboardingSession{Error: "無���連接 Telegram API"}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"result"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return OnboardingSession{Error: "解析回應失敗"}, err
	}

	if !result.OK {
		errMsg := result.Description
		if errMsg == "" {
			errMsg = "token 無效"
		}
		return OnboardingSession{Error: errMsg, Valid: false}, fmt.Errorf("getMe failed: %s", errMsg)
	}

	return OnboardingSession{
		BotUsername: result.Result.Username,
		BotID:       result.Result.ID,
		Valid:       true,
	}, nil
}

// PollTelegramChatID 輪詢 getUpdates，��選含 /start 的 message。
func PollTelegramChatID(botToken string) ([]ChatCandidate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=1&allowed_updates=[\"message\"]", botToken)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var updates struct {
		OK     bool `json:"ok"`
		Result []struct {
			Message struct {
				Text string `json:"text"`
				Chat struct {
					ID       int64  `json:"id"`
					Title    string `json:"title"`
					Type     string `json:"type"`
					Username string `json:"username"`
				} `json:"chat"`
				From struct {
					Username string `json:"username"`
				} `json:"from"`
			} `json:"message"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &updates); err != nil {
		return nil, err
	}

	// 篩選含 /start 的 message
	seen := make(map[int64]bool)
	var candidates []ChatCandidate
	for _, u := range updates.Result {
		if !strings.HasPrefix(u.Message.Text, "/start") {
			continue
		}
		chatID := u.Message.Chat.ID
		if seen[chatID] {
			continue
		}
		seen[chatID] = true

		title := u.Message.Chat.Title
		if title == "" {
			title = u.Message.From.Username
		}
		candidates = append(candidates, ChatCandidate{
			ChatID:    chatID,
			ChatTitle: title,
			ChatType:  u.Message.Chat.Type,
			Username:  u.Message.Chat.Username,
		})
	}

	return candidates, nil
}

// ConfirmTelegramChatID 確認 chat_id 並存入 device-local secret。
// Audit / project export 只存 sha256(chat_id)。
func ConfirmTelegramChatID(secrets credential.SecretStore, channelID string, chatID string) error {
	if chatID == "" {
		return fmt.Errorf("chat_id 不能為空")
	}

	// 存入 device-local secret（namespaced）
	ref := fmt.Sprintf("remote_bridge:%s:chat_id", channelID)
	if err := secrets.Store(ref, chatID); err != nil {
		return fmt.Errorf("store chat_id: %w", err)
	}

	return nil
}

// HashChatID 計算 chat_id 的 SHA-256 雜湊（供 audit 和 export 使用）。
func HashChatID(chatID string) string {
	h := sha256.Sum256([]byte(chatID))
	return fmt.Sprintf("sha256:%x", h)
}
