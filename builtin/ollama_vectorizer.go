// ollama_vectorizer.go — Phase B M2：用本機 Ollama 產生 dense embedding。
//
// 設計原則：
//   - 純 vectorizer 實作；不依賴 App / settings / Wails。caller 自己決定何時建構。
//   - 用 /api/embed（新端點）— /api/embeddings 是舊端點，未來會 deprecate。
//     Response 是 {"embeddings": [[...]]}（batch-capable），相容 fallback 單筆 {"embedding": [...]}。
//   - Dimension 是「第一次成功 embed 時量到」才寫進來；外層不能直接設。
//     第二次以後若回不同維度 → 視為 corruption，回 ErrOllamaBadResponse。
//   - HTTP timeout 30s + LimitReader 4 MiB，避免被 hang / OOM。
//   - NaN/Inf 一律 reject，避免下游 cosine 變 NaN。
package builtin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"ui_console/internal/urlsafe"
)

const (
	defaultOllamaEndpoint       = "http://localhost:11434"
	ollamaEmbedHTTPTimeout      = 30 * time.Second
	ollamaEmbedMaxResponseBytes = 4 * 1024 * 1024 // 4 MiB
)

// ErrOllamaUnavailable 表示 Ollama server 連不上 / 模型沒裝 / 5xx。
// 外層拿到應該 fallback 到 TF-IDF（不要當 fatal）。
var ErrOllamaUnavailable = errors.New("ollama: server unreachable or model missing")

// ErrOllamaBadResponse 表示 server 回了但格式異常（壞 JSON、空向量、NaN、維度漂移）。
// 這通常代表 server 端 bug 或 model 對不上，外層應該 surface 給使用者。
var ErrOllamaBadResponse = errors.New("ollama: invalid embedding response")

// OllamaEmbedVectorizer 用 Ollama HTTP API 產生 dense 向量。
//
// 使用方式：
//
//	v := NewOllamaEmbedVectorizer("", "nomic-embed-text")
//	vec, err := v.Vectorize("hello")
type OllamaEmbedVectorizer struct {
	Endpoint string // 例 "http://localhost:11434"；空字串自動填預設
	ModelID  string // 例 "nomic-embed-text"；不可空

	client *http.Client

	dimMu       sync.Mutex
	measuredDim int // 0 = 尚未量過；非 0 = 鎖定值
}

// NewOllamaEmbedVectorizer 建構。Endpoint 空字串 → defaultOllamaEndpoint。
func NewOllamaEmbedVectorizer(endpoint, modelID string) *OllamaEmbedVectorizer {
	if endpoint == "" {
		endpoint = defaultOllamaEndpoint
	}
	return &OllamaEmbedVectorizer{
		Endpoint: endpoint,
		ModelID:  modelID,
		// SEC-05 2a: 本機 embedding 走 PolicyLocalLLM。
		client: urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "ollama_embed", ollamaEmbedHTTPTimeout),
	}
}

// Meta 回 metadata。Dimension 在第一次成功 Vectorize 後才會非零。
// 重要：外部讀到 Dimension==0 就代表「還沒測過」，不要拿去比較。
func (o *OllamaEmbedVectorizer) Meta() VectorMetadata {
	o.dimMu.Lock()
	dim := o.measuredDim
	o.dimMu.Unlock()
	return VectorMetadata{
		Type:      "dense",
		ModelID:   "ollama-" + o.ModelID,
		Dimension: dim,
	}
}

// MeasuredDimension 回目前已鎖定的維度（0 表示尚未量到）。
// 給 backend 量完後寫回 settings 用。
func (o *OllamaEmbedVectorizer) MeasuredDimension() int {
	o.dimMu.Lock()
	defer o.dimMu.Unlock()
	return o.measuredDim
}

// Vectorize 打 POST /api/embed，回 Vector{Dense: ..., Meta: ...}。
//
// 錯誤分類：
//   - text 空 / ModelID 空 → 立即回 error，不打網路
//   - 連線錯誤 / 5xx → 包 ErrOllamaUnavailable
//   - 200 但內容怪 → 包 ErrOllamaBadResponse
func (o *OllamaEmbedVectorizer) Vectorize(text string) (Vector, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Vector{}, errors.New("ollama: empty text")
	}
	if o.ModelID == "" {
		return Vector{}, errors.New("ollama: ModelID required")
	}

	payload, err := json.Marshal(map[string]any{
		"model": o.ModelID,
		"input": text, // /api/embed 支援 string 或 []string
	})
	if err != nil {
		return Vector{}, fmt.Errorf("ollama: marshal request: %w", err)
	}

	endpoint := strings.TrimRight(o.Endpoint, "/") + "/api/embed"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return Vector{}, fmt.Errorf("ollama: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return Vector{}, fmt.Errorf("%w: %v", ErrOllamaUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, ollamaEmbedMaxResponseBytes))
	if err != nil {
		return Vector{}, fmt.Errorf("ollama: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 404 通常代表 model 沒 pull；其他 5xx / 4xx 都歸 unavailable。
		return Vector{}, fmt.Errorf("%w: HTTP %d: %s", ErrOllamaUnavailable, resp.StatusCode, truncateForError(string(body), 200))
	}

	var parsed struct {
		Embeddings [][]float64 `json:"embeddings"`
		Embedding  []float64   `json:"embedding"` // 舊端點 fallback
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Vector{}, fmt.Errorf("%w: %v", ErrOllamaBadResponse, err)
	}

	var dense []float64
	switch {
	case len(parsed.Embeddings) > 0 && len(parsed.Embeddings[0]) > 0:
		dense = parsed.Embeddings[0]
	case len(parsed.Embedding) > 0:
		dense = parsed.Embedding
	default:
		return Vector{}, fmt.Errorf("%w: no embedding in response", ErrOllamaBadResponse)
	}

	if err := validateDenseEmbedding(dense); err != nil {
		return Vector{}, err
	}

	// 第一次成功 → 鎖定 dimension。
	// 第二次以後如果不一樣 → 嚴重狀況（換 model 或 server 壞），回 error 讓上層處理。
	o.dimMu.Lock()
	if o.measuredDim == 0 {
		o.measuredDim = len(dense)
	} else if o.measuredDim != len(dense) {
		got, want := len(dense), o.measuredDim
		o.dimMu.Unlock()
		return Vector{}, fmt.Errorf("%w: dimension drift (got %d, locked %d)", ErrOllamaBadResponse, got, want)
	}
	dim := o.measuredDim
	o.dimMu.Unlock()

	return Vector{
		Dense: dense,
		Meta: VectorMetadata{
			Type:      "dense",
			ModelID:   "ollama-" + o.ModelID,
			Dimension: dim,
		},
	}, nil
}

// validateDenseEmbedding：非空 + 無 NaN/Inf。
// 不檢查正規化（Ollama 通常已歸一化，但某些模型不會——cosine 內部會處理）。
func validateDenseEmbedding(dense []float64) error {
	if len(dense) == 0 {
		return fmt.Errorf("%w: zero-length embedding", ErrOllamaBadResponse)
	}
	for i, v := range dense {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%w: NaN/Inf at index %d", ErrOllamaBadResponse, i)
		}
	}
	return nil
}

// truncateForError 截短 server response 以免 error message 太長爆 log。
func truncateForError(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
