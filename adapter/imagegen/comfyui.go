// comfyui.go — 本機 ComfyUI 的 txt2img 後端實作。
//
// 流程：POST /prompt 送入一張最小 txt2img workflow → 輪詢 /history/{id} 等出圖
// → GET /view 取回 PNG。所有 HTTP 走呼叫端注入的 *http.Client（app 層用
// urlsafe.NewSafeClient(PolicyLocalLLM,...)，只放行 loopback、擋 LAN/metadata）。
package imagegen

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ComfyUIAdapter 對接本機 ComfyUI server。
type ComfyUIAdapter struct {
	baseURL    string       // 例 http://127.0.0.1:8188
	httpClient *http.Client // 由呼叫端注入（含 egress policy 與 timeout）
	checkpoint string       // 預設 checkpoint（ckpt_name）
	clientID   string       // ComfyUI client_id，用於 /history 對位

	// 可調參數（建構時帶預設）。
	pollInterval time.Duration
	pollTimeout  time.Duration
}

// NewComfyUIAdapter 建立 ComfyUI adapter。httpClient 不可為 nil。
func NewComfyUIAdapter(baseURL string, httpClient *http.Client, checkpoint, clientID string) *ComfyUIAdapter {
	return &ComfyUIAdapter{
		baseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient:   httpClient,
		checkpoint:   strings.TrimSpace(checkpoint),
		clientID:     firstNonEmpty(clientID, "ai-console-keepsake"),
		pollInterval: 1 * time.Second,
		pollTimeout:  3 * time.Minute,
	}
}

// WithPolling 覆寫輪詢節奏（主要供測試縮短）。
func (c *ComfyUIAdapter) WithPolling(interval, timeout time.Duration) *ComfyUIAdapter {
	if interval > 0 {
		c.pollInterval = interval
	}
	if timeout > 0 {
		c.pollTimeout = timeout
	}
	return c
}

func (c *ComfyUIAdapter) Name() string { return "comfyui" }

// Health 以 /system_stats 確認 server 可達。
func (c *ComfyUIAdapter) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/system_stats", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ComfyUI 不可達: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ComfyUI 回應異常: %d", resp.StatusCode)
	}
	return nil
}

// Generate 執行一次 txt2img。
func (c *ComfyUIAdapter) Generate(ctx context.Context, req Request) (Result, error) {
	if c.httpClient == nil {
		return Result{}, fmt.Errorf("httpClient 未設定")
	}
	seed := req.Seed
	if seed < 0 {
		seed = randomSeed()
	}
	model := firstNonEmpty(req.Model, c.checkpoint)
	if model == "" {
		return Result{}, fmt.Errorf("未指定 ComfyUI checkpoint")
	}

	graph := buildTxt2ImgGraph(req, seed, model)
	promptID, err := c.submit(ctx, graph)
	if err != nil {
		return Result{}, err
	}
	img, err := c.waitForImage(ctx, promptID)
	if err != nil {
		return Result{}, err
	}
	png, err := c.fetchImage(ctx, img)
	if err != nil {
		return Result{}, err
	}
	return Result{PNG: png, Seed: seed, Model: model}, nil
}

// ── 內部：送 prompt ──

func (c *ComfyUIAdapter) submit(ctx context.Context, graph map[string]any) (string, error) {
	body, err := json.Marshal(map[string]any{
		"prompt":    graph,
		"client_id": c.clientID,
	})
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/prompt", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("送出產圖請求失敗: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("ComfyUI /prompt 回應 %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	var out struct {
		PromptID string `json:"prompt_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("解析 prompt_id 失敗: %w", err)
	}
	if out.PromptID == "" {
		return "", fmt.Errorf("ComfyUI 未回傳 prompt_id")
	}
	return out.PromptID, nil
}

// imageRef 是 /history 回來的一張圖位置。
type imageRef struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

func (c *ComfyUIAdapter) waitForImage(ctx context.Context, promptID string) (imageRef, error) {
	deadline := time.Now().Add(c.pollTimeout)
	for {
		select {
		case <-ctx.Done():
			return imageRef{}, ctx.Err()
		default:
		}
		ref, done, err := c.pollHistory(ctx, promptID)
		if err != nil {
			return imageRef{}, err
		}
		if done {
			if ref.Filename == "" {
				return imageRef{}, fmt.Errorf("ComfyUI 完成但無輸出圖")
			}
			return ref, nil
		}
		if time.Now().After(deadline) {
			return imageRef{}, fmt.Errorf("等待 ComfyUI 出圖逾時")
		}
		select {
		case <-ctx.Done():
			return imageRef{}, ctx.Err()
		case <-time.After(c.pollInterval):
		}
	}
}

func (c *ComfyUIAdapter) pollHistory(ctx context.Context, promptID string) (imageRef, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/history/"+url.PathEscape(promptID), nil)
	if err != nil {
		return imageRef{}, false, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return imageRef{}, false, fmt.Errorf("輪詢 ComfyUI history 失敗: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return imageRef{}, false, nil // 尚未就緒，繼續等
	}
	// /history/{id} → { "<id>": { "outputs": { "<node>": { "images": [imageRef...] } } } }
	var hist map[string]struct {
		Outputs map[string]struct {
			Images []imageRef `json:"images"`
		} `json:"outputs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hist); err != nil {
		return imageRef{}, false, fmt.Errorf("解析 history 失敗: %w", err)
	}
	entry, ok := hist[promptID]
	if !ok {
		return imageRef{}, false, nil // 還沒寫進 history
	}
	for _, node := range entry.Outputs {
		for _, img := range node.Images {
			if img.Filename != "" {
				return img, true, nil
			}
		}
	}
	// 有 entry 但沒圖：當作完成但空。
	return imageRef{}, true, nil
}

func (c *ComfyUIAdapter) fetchImage(ctx context.Context, ref imageRef) ([]byte, error) {
	q := url.Values{}
	q.Set("filename", ref.Filename)
	q.Set("subfolder", ref.Subfolder)
	if ref.Type != "" {
		q.Set("type", ref.Type)
	} else {
		q.Set("type", "output")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/view?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下載產圖失敗: %w", err)
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ComfyUI /view 回應 %d", resp.StatusCode)
	}
	// 上限 32MB，避免異常巨大回應吃爆記憶體。
	png, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, err
	}
	if len(png) == 0 {
		return nil, fmt.Errorf("ComfyUI 回傳空圖")
	}
	return png, nil
}

// buildTxt2ImgGraph 組出最小 txt2img workflow（ComfyUI API 格式）。
func buildTxt2ImgGraph(req Request, seed int64, model string) map[string]any {
	width := orDefaultInt(req.Width, 512)
	height := orDefaultInt(req.Height, 768)
	steps := orDefaultInt(req.Steps, 24)
	cfg := req.CFG
	if cfg <= 0 {
		cfg = 7.0
	}
	sampler := firstNonEmpty(req.Sampler, "euler")
	negative := firstNonEmpty(req.Negative, DefaultNegative)

	return map[string]any{
		"4": map[string]any{
			"class_type": "CheckpointLoaderSimple",
			"inputs":     map[string]any{"ckpt_name": model},
		},
		"5": map[string]any{
			"class_type": "EmptyLatentImage",
			"inputs":     map[string]any{"width": width, "height": height, "batch_size": 1},
		},
		"6": map[string]any{
			"class_type": "CLIPTextEncode",
			"inputs":     map[string]any{"text": req.Positive, "clip": []any{"4", 1}},
		},
		"7": map[string]any{
			"class_type": "CLIPTextEncode",
			"inputs":     map[string]any{"text": negative, "clip": []any{"4", 1}},
		},
		"3": map[string]any{
			"class_type": "KSampler",
			"inputs": map[string]any{
				"seed":         seed,
				"steps":        steps,
				"cfg":          cfg,
				"sampler_name": sampler,
				"scheduler":    "normal",
				"denoise":      1.0,
				"model":        []any{"4", 0},
				"positive":     []any{"6", 0},
				"negative":     []any{"7", 0},
				"latent_image": []any{"5", 0},
			},
		},
		"8": map[string]any{
			"class_type": "VAEDecode",
			"inputs":     map[string]any{"samples": []any{"3", 0}, "vae": []any{"4", 2}},
		},
		"9": map[string]any{
			"class_type": "SaveImage",
			"inputs":     map[string]any{"filename_prefix": "keepsake", "images": []any{"8", 0}},
		},
	}
}

func orDefaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func randomSeed() int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return time.Now().UnixNano()
	}
	return n.Int64()
}

func drain(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
}
