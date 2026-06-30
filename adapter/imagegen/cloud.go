// cloud.go — 雲端產圖後端（OpenAI images 相容）。
//
// 對接「POST {endpoint} → {data:[{b64_json|url}]}」這類相容介面（OpenAI DALL·E、
// 多數聚合服務、自架相容端點皆適用）。egress 由呼叫端注入 *http.Client，app 層用
// urlsafe.NewSafeClient(PolicyCloudAPI,...)（僅公網 https、封鎖 localhost/LAN）。
//
// 動漫風由「模型(model) + 提示詞畫風」決定，與本 adapter 無關；BuildScenePrompt
// 已內建 anime 前綴。
package imagegen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CloudAdapter 對接 OpenAI images 相容的雲端產圖端點。
type CloudAdapter struct {
	endpoint   string // 完整 URL，例 https://api.openai.com/v1/images/generations
	apiKey     string
	model      string // 例 dall-e-3 / 服務商指定的模型名
	httpClient *http.Client
}

// NewCloudAdapter 建立雲端 adapter。httpClient 不可為 nil。
func NewCloudAdapter(endpoint, apiKey, model string, httpClient *http.Client) *CloudAdapter {
	return &CloudAdapter{
		endpoint:   strings.TrimSpace(endpoint),
		apiKey:     strings.TrimSpace(apiKey),
		model:      strings.TrimSpace(model),
		httpClient: httpClient,
	}
}

func (c *CloudAdapter) Name() string { return "cloud" }

// Health 雲端端點不一定有健康檢查；這裡僅驗證設定齊全。
func (c *CloudAdapter) Health(ctx context.Context) error {
	if c.endpoint == "" {
		return fmt.Errorf("未設定雲端產圖端點")
	}
	if c.apiKey == "" {
		return fmt.Errorf("未設定雲端產圖金鑰")
	}
	return nil
}

// Generate 呼叫雲端端點產圖。
func (c *CloudAdapter) Generate(ctx context.Context, req Request) (Result, error) {
	if c.httpClient == nil {
		return Result{}, fmt.Errorf("httpClient 未設定")
	}
	if err := c.Health(ctx); err != nil {
		return Result{}, err
	}
	model := firstNonEmpty(req.Model, c.model)
	if model == "" {
		return Result{}, fmt.Errorf("未指定雲端產圖模型")
	}

	width := orDefaultInt(req.Width, 512)
	height := orDefaultInt(req.Height, 768)
	body, err := json.Marshal(map[string]any{
		"model":           model,
		"prompt":          req.Positive,
		"n":               1,
		"size":            fmt.Sprintf("%dx%d", width, height),
		"response_format": "b64_json",
	})
	if err != nil {
		return Result{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("雲端產圖請求失敗: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Result{}, fmt.Errorf("雲端產圖回應 %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	var out struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024*1024)).Decode(&out); err != nil {
		return Result{}, fmt.Errorf("解析雲端回應失敗: %w", err)
	}
	if len(out.Data) == 0 {
		return Result{}, fmt.Errorf("雲端未回傳圖片")
	}

	// 優先 b64_json；否則用 url 再抓一次。
	if b64 := strings.TrimSpace(out.Data[0].B64JSON); b64 != "" {
		png, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return Result{}, fmt.Errorf("解碼雲端圖片失敗: %w", err)
		}
		return Result{PNG: png, Seed: req.Seed, Model: model}, nil
	}
	if u := strings.TrimSpace(out.Data[0].URL); u != "" {
		png, err := c.fetchURL(ctx, u)
		if err != nil {
			return Result{}, err
		}
		return Result{PNG: png, Seed: req.Seed, Model: model}, nil
	}
	return Result{}, fmt.Errorf("雲端回應沒有可用圖片資料")
}

func (c *CloudAdapter) fetchURL(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下載雲端圖片失敗: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下載雲端圖片回應 %d", resp.StatusCode)
	}
	png, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, err
	}
	if len(png) == 0 {
		return nil, fmt.Errorf("雲端圖片為空")
	}
	return png, nil
}
