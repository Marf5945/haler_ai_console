// remote_bridge/connection.go — 連線測試。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 使用者在引用連結貼 URL、通過 detector 辨識後，              │
// │ 本檔案負責「真的打一次 HTTP 請求」確認 endpoint 可達。       │
// │                                                             │
// │ 各平台測試策略（MVP）：                                     │
// │  • Telegram : GET /bot<TOKEN>/getMe → 驗證 {"ok":true}     │
// │  • Discord  : GET webhook URL → 200 表示 webhook 存在      │
// │  • LINE     : HEAD notify-api/api/status → 200 或 401      │
// │              （401 = endpoint 存在但需 token，視為可達）     │
// │  • Teams    : HEAD/GET webhook URL，4xx 也視為 endpoint 可達 │
// │                                                             │
// │ 呼叫鏈：                                                    │
// │  前端 onConfirm → Wails TestRemoteBridgeConnection           │
// │    → service.TestChannelConnection → tester.TestConnection   │
// │    → testTelegram / testDiscord / testLINE (本檔案)          │
// │                                                             │
// │ 測試通過後，service.RegisterChannel 才會建立 ChannelBinding  │
// │ 並讓前端黃框區域長出 icon。                                  │
// │                                                             │
// │ 注意：MVP 不實際送出使用者可見訊息（不 POST sendMessage）。  │
// │ Timeout 10 秒，失敗回傳 ConnectionTestResult.Success=false。 │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ui_console/internal/urlsafe"
)

// ──────────────────────────────────────────────
// 連線測試器
// ──────────────────────────────────────────────

// ConnectionTester 執行各通道的連線測試。
type ConnectionTester struct {
	client *http.Client
}

// NewConnectionTester 建立連線測試器。
// SEC-05: 使用 Safe Client（PolicyWebhook，僅公網 HTTPS + 連線當下 IP 篩選）。
func NewConnectionTester() *ConnectionTester {
	return &ConnectionTester{
		client: urlsafe.NewSafeClient(urlsafe.PolicyWebhook, "connection_test", 10*time.Second),
	}
}

// TestConnection 根據通道類型執行對應的連線測試。
// URL 必須先通過 DetectChannel 取得通道類型。
func (ct *ConnectionTester) TestConnection(rawURL string, channel ChannelType) ConnectionTestResult {
	now := time.Now()

	if err := ValidateURLFormat(rawURL); err != nil {
		return ConnectionTestResult{
			Success:      false,
			Channel:      channel,
			ErrorMessage: err.Error(),
			TestedAt:     now,
		}
	}

	// SEC-05: 字面值 SSRF 快速失敗；實質防線仍在 Safe Client 的連線時 IP 篩選。
	if err := urlsafe.ValidateURL(rawURL, urlsafe.PolicyWebhook); err != nil {
		return ConnectionTestResult{
			Success:      false,
			Channel:      channel,
			ErrorMessage: err.Error(),
			TestedAt:     now,
		}
	}

	var err error
	switch channel {
	case ChannelTelegram:
		err = ct.testTelegram(rawURL)
	case ChannelDiscord:
		err = ct.testDiscord(rawURL)
	case ChannelLINE:
		err = ct.testLINE(rawURL)
	case ChannelTeams:
		err = ct.testTeams(rawURL)
	case ChannelQQ:
		err = ct.testQQ(rawURL)
	default:
		err = fmt.Errorf("unsupported channel: %s", channel)
	}

	if err != nil {
		return ConnectionTestResult{
			Success:      false,
			Channel:      channel,
			ErrorMessage: err.Error(),
			TestedAt:     now,
		}
	}
	return ConnectionTestResult{
		Success:  true,
		Channel:  channel,
		TestedAt: now,
	}
}

// testTelegram 用 getMe API 測試 Telegram Bot Token 是否有效。
func (ct *ConnectionTester) testTelegram(rawURL string) error {
	token, err := ExtractBotToken(rawURL)
	if err != nil {
		return fmt.Errorf("無法解析 Telegram Bot Token：%w", err)
	}

	getMeURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token)
	resp, err := ct.client.Get(getMeURL)
	if err != nil {
		return fmt.Errorf("Telegram 連線失敗：%w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram 回傳錯誤（%d）：%s", resp.StatusCode, truncate(string(body), 200))
	}

	// 檢查 {"ok": true}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.OK {
		return fmt.Errorf("Telegram getMe 回傳非成功狀態")
	}
	return nil
}

// testDiscord 用 GET webhook URL 測試 Discord Webhook 是否有效。
func (ct *ConnectionTester) testDiscord(rawURL string) error {
	// Discord webhook GET returns webhook info without sending a message.
	resp, err := ct.client.Get(rawURL)
	if err != nil {
		return fmt.Errorf("Discord 連線失敗：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("Discord 回傳錯誤（%d）：%s", resp.StatusCode, truncate(string(body), 200))
}

// testLINE 測試 LINE Notify 或 Messaging API。
// LINE Notify: POST 帶 Authorization header 測試 /api/status。
// LINE Messaging API: 需要 Channel Access Token，MVP 先做 HEAD 測試。
func (ct *ConnectionTester) testLINE(rawURL string) error {
	parsed := strings.ToLower(rawURL)

	if strings.Contains(parsed, "notify-api.line.me") {
		// LINE Notify — 測試 /api/status
		statusURL := "https://notify-api.line.me/api/status"
		// 從 URL 或 header 取 token（MVP：假設 token 在 query param 或需另外提供）
		// 先做基本的 HEAD 測試
		resp, err := ct.client.Head(statusURL)
		if err != nil {
			return fmt.Errorf("LINE Notify 連線失敗：%w", err)
		}
		defer resp.Body.Close()

		// 200 或 401 都表示 endpoint 可達（401 = token 需提供但 endpoint 活著）
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
			return nil
		}
		return fmt.Errorf("LINE Notify 回傳異常（%d）", resp.StatusCode)
	}

	// LINE Messaging API
	resp, err := ct.client.Head(rawURL)
	if err != nil {
		return fmt.Errorf("LINE API 連線失敗：%w", err)
	}
	defer resp.Body.Close()

	// 405 Method Not Allowed 也算 endpoint 存在
	if resp.StatusCode < 500 {
		return nil
	}
	return fmt.Errorf("LINE API 回傳伺服器錯誤（%d）", resp.StatusCode)
}

// testTeams 不送出可見訊息，只確認 webhook endpoint 可連線。
func (ct *ConnectionTester) testTeams(rawURL string) error {
	resp, err := ct.client.Head(rawURL)
	if err != nil {
		resp, err = ct.client.Get(rawURL)
	}
	if err != nil {
		return fmt.Errorf("Teams 連線失敗：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 500 {
		return nil
	}
	return fmt.Errorf("Teams Webhook 回傳伺服器錯誤（%d）", resp.StatusCode)
}

func (ct *ConnectionTester) testQQ(rawURL string) error {
	resp, err := ct.client.Head(rawURL)
	if err != nil {
		resp, err = ct.client.Get(rawURL)
	}
	if err != nil {
		return fmt.Errorf("QQ Bot API 連線失敗：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 500 {
		return nil
	}
	return fmt.Errorf("QQ Bot API 回傳伺服器錯誤（%d）", resp.StatusCode)
}

// truncate 截斷字串至指定長度。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
