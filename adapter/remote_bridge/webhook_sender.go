// remote_bridge/webhook_sender.go — Generic Webhook Sender（§12A.2）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 平台無關的 HTTP webhook 發送核心。                          │
// │ sender 不知道 Telegram/Discord/LINE 等平台細節，            │
// │ 只處理 HTTP POST/GET + timeout + response 回傳。           │
// │                                                             │
// │ 設計原則：                                                  │
// │  • WebhookRequest 只含 url/method/headers/body/timeout     │
// │  • 無任何平台欄位                                          │
// │  • 新增平台只需加 preset，不改本檔案                       │
// │                                                             │
// │ SEC-04: 加入 SSRF 防護（ValidateURL）與 JSON 安全檢查。    │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ui_console/internal/urlsafe"
)

// ──────────────────────────────────────────────
// 請求 / 回應結構
// ──────────────────────────────────────────────

// WebhookRequest 是平台無關的 HTTP 請求描述。
type WebhookRequest struct {
	URL            string            `json:"url"`
	Method         string            `json:"method"`          // POST / GET
	Headers        map[string]string `json:"headers"`         // 自訂 header
	Body           string            `json:"body"`            // 請求 body（應由 json.Marshal 產生）
	TimeoutSeconds int               `json:"timeout_seconds"` // 逾時秒數（預設 10）
	DevMode        bool              `json:"dev_mode"`        // SEC-04: true 時允許 localhost/private（需前端確認）
}

// WebhookResponse 是 HTTP 回應的結構化表示。
type WebhookResponse struct {
	StatusCode   int    `json:"status_code"`
	ResponseBody string `json:"response_body"`
	Error        string `json:"error,omitempty"`
}

// ──────────────────────────────────────────────
// 發送函式
// ──────────────────────────────────────────────

// SendWebhook 執行單次 HTTP 請求，回傳結構化回應。
// SEC-04: 發送前先驗證 URL，防止 SSRF。
// DevMode=false → 僅允許 https 公網（PolicyWebhook）。
// DevMode=true  → 允許 localhost/private（PolicyWebhookDev），需前端已確認。
func SendWebhook(req WebhookRequest) (WebhookResponse, error) {
	// SEC-04: URL SSRF 防護
	policy := urlsafe.PolicyWebhook
	if req.DevMode {
		policy = urlsafe.PolicyWebhookDev
	}
	if err := urlsafe.ValidateURL(req.URL, policy); err != nil {
		return WebhookResponse{Error: fmt.Sprintf("SSRF blocked: %v", err)},
			fmt.Errorf("webhook URL validation failed: %w", err)
	}

	// 預設值
	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}
	method := req.Method
	if method == "" {
		method = "POST"
	}

	// 建立 HTTP client（每次請求獨立 timeout）
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 建立 HTTP 請求
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}
	httpReq, err := http.NewRequest(method, req.URL, bodyReader)
	if err != nil {
		return WebhookResponse{Error: err.Error()}, err
	}

	// 設定 headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	// 預設 Content-Type
	if httpReq.Header.Get("Content-Type") == "" && req.Body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 發送請求
	resp, err := client.Do(httpReq)
	if err != nil {
		return WebhookResponse{Error: err.Error()}, err
	}
	defer resp.Body.Close()

	// 讀取回應（限制 1MB 防止 OOM）
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return WebhookResponse{
			StatusCode: resp.StatusCode,
			Error:      err.Error(),
		}, err
	}

	return WebhookResponse{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(body),
	}, nil
}
