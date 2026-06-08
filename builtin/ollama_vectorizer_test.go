// ollama_vectorizer_test.go — M2 OllamaEmbedVectorizer 用 httptest mock 全 case 覆蓋。
//
// 涵蓋：
//  1. 成功（新端點 /api/embed 回 embeddings: [[...]]）
//  2. 成功（舊端點 fallback embedding: [...]）
//  3. 5xx 連線錯誤 → ErrOllamaUnavailable
//  4. 404 / model 沒裝 → ErrOllamaUnavailable
//  5. 200 但 JSON 壞 → ErrOllamaBadResponse
//  6. 200 但空 embeddings → ErrOllamaBadResponse
//  7. NaN / Inf → ErrOllamaBadResponse
//  8. Dimension 第一次量到後固定；第二次不同維度 → ErrOllamaBadResponse
//  9. text 空 / ModelID 空 → 立即錯誤，不打網路
package builtin

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper — 建一個會回指定 JSON 的 server。
func newMockServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		// 讀完 request body 確保 reader 不被 leak
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// ─────────────────────────────────────
// 成功路徑
// ─────────────────────────────────────

func TestOllamaVectorize_NewEndpointShape(t *testing.T) {
	body := `{"embeddings": [[0.1, 0.2, 0.3]]}`
	srv := newMockServer(t, http.StatusOK, body)
	defer srv.Close()

	v := NewOllamaEmbedVectorizer(srv.URL, "nomic-embed-text")
	out, err := v.Vectorize("hello")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Meta.Type != "dense" {
		t.Errorf("expected dense, got %q", out.Meta.Type)
	}
	if out.Meta.ModelID != "ollama-nomic-embed-text" {
		t.Errorf("modelID wrong: %q", out.Meta.ModelID)
	}
	if out.Meta.Dimension != 3 {
		t.Errorf("dim wrong: %d", out.Meta.Dimension)
	}
	if len(out.Dense) != 3 || out.Dense[1] != 0.2 {
		t.Errorf("dense wrong: %v", out.Dense)
	}
}

func TestOllamaVectorize_LegacyEndpointShape(t *testing.T) {
	// 舊端點回 singular embedding，新 vectorizer 也要相容
	body := `{"embedding": [0.5, 0.5]}`
	srv := newMockServer(t, http.StatusOK, body)
	defer srv.Close()

	v := NewOllamaEmbedVectorizer(srv.URL, "old-model")
	out, err := v.Vectorize("test")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out.Dense) != 2 {
		t.Errorf("dim wrong: %d", len(out.Dense))
	}
}

// ─────────────────────────────────────
// 連線 / server 錯誤
// ─────────────────────────────────────

func TestOllamaVectorize_5xxUnavailable(t *testing.T) {
	srv := newMockServer(t, http.StatusInternalServerError, `{"error":"oops"}`)
	defer srv.Close()
	v := NewOllamaEmbedVectorizer(srv.URL, "any")
	_, err := v.Vectorize("test")
	if !errors.Is(err, ErrOllamaUnavailable) {
		t.Errorf("expected ErrOllamaUnavailable, got %v", err)
	}
}

func TestOllamaVectorize_404ModelMissing(t *testing.T) {
	srv := newMockServer(t, http.StatusNotFound, `{"error":"model not found"}`)
	defer srv.Close()
	v := NewOllamaEmbedVectorizer(srv.URL, "missing")
	_, err := v.Vectorize("test")
	if !errors.Is(err, ErrOllamaUnavailable) {
		t.Errorf("expected ErrOllamaUnavailable for 404, got %v", err)
	}
}

func TestOllamaVectorize_ConnectionRefused(t *testing.T) {
	// 沒有 server 在跑的 endpoint
	v := NewOllamaEmbedVectorizer("http://127.0.0.1:1", "any") // port 1 不太可能有人佔
	_, err := v.Vectorize("test")
	if !errors.Is(err, ErrOllamaUnavailable) {
		t.Errorf("expected ErrOllamaUnavailable for connection refused, got %v", err)
	}
}

// ─────────────────────────────────────
// 200 但異常
// ─────────────────────────────────────

func TestOllamaVectorize_MalformedJSON(t *testing.T) {
	srv := newMockServer(t, http.StatusOK, `{not json}`)
	defer srv.Close()
	v := NewOllamaEmbedVectorizer(srv.URL, "any")
	_, err := v.Vectorize("test")
	if !errors.Is(err, ErrOllamaBadResponse) {
		t.Errorf("expected ErrOllamaBadResponse for malformed JSON, got %v", err)
	}
}

func TestOllamaVectorize_EmptyEmbeddings(t *testing.T) {
	srv := newMockServer(t, http.StatusOK, `{"embeddings": []}`)
	defer srv.Close()
	v := NewOllamaEmbedVectorizer(srv.URL, "any")
	_, err := v.Vectorize("test")
	if !errors.Is(err, ErrOllamaBadResponse) {
		t.Errorf("expected ErrOllamaBadResponse for empty embeddings, got %v", err)
	}
}

func TestOllamaVectorize_NaNRejected(t *testing.T) {
	// NaN 沒辦法直接放 JSON literal，用手寫 string
	srv := newMockServer(t, http.StatusOK, `{"embeddings": [[1.0, "NaN", 2.0]]}`)
	defer srv.Close()
	v := NewOllamaEmbedVectorizer(srv.URL, "any")
	_, err := v.Vectorize("test")
	// JSON parse 會先報錯（因為字串不能塞進 []float64），所以 err 還是 BadResponse
	if !errors.Is(err, ErrOllamaBadResponse) {
		t.Errorf("expected ErrOllamaBadResponse for NaN-shaped JSON, got %v", err)
	}
}

func TestOllamaVectorize_ValidateNaNDirectly(t *testing.T) {
	// 直接驗 validateDenseEmbedding 的 NaN / Inf 路徑
	err := validateDenseEmbedding([]float64{1.0, math.NaN()})
	if !errors.Is(err, ErrOllamaBadResponse) {
		t.Errorf("NaN should fail validate, got %v", err)
	}
	err = validateDenseEmbedding([]float64{1.0, math.Inf(1)})
	if !errors.Is(err, ErrOllamaBadResponse) {
		t.Errorf("Inf should fail validate, got %v", err)
	}
	if err := validateDenseEmbedding([]float64{1.0, 0.5}); err != nil {
		t.Errorf("normal floats should pass, got %v", err)
	}
}

// ─────────────────────────────────────
// Dimension lock-in
// ─────────────────────────────────────

func TestOllamaVectorize_DimensionLockIn(t *testing.T) {
	// 模擬：第一次 server 回 3 維、第二次回 5 維（不該發生但要擋）
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`{"embeddings": [[0.1, 0.2, 0.3]]}`))
		} else {
			_, _ = w.Write([]byte(`{"embeddings": [[0.1, 0.2, 0.3, 0.4, 0.5]]}`))
		}
	}))
	defer srv.Close()

	v := NewOllamaEmbedVectorizer(srv.URL, "drift-model")

	out1, err := v.Vectorize("first")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if out1.Meta.Dimension != 3 {
		t.Errorf("first dim should be 3, got %d", out1.Meta.Dimension)
	}
	if v.MeasuredDimension() != 3 {
		t.Errorf("MeasuredDimension should be 3, got %d", v.MeasuredDimension())
	}

	_, err = v.Vectorize("second")
	if !errors.Is(err, ErrOllamaBadResponse) {
		t.Errorf("dim drift should fail with ErrOllamaBadResponse, got %v", err)
	}
}

// ─────────────────────────────────────
// 預先檢查（不打網路）
// ─────────────────────────────────────

func TestOllamaVectorize_EmptyTextRejected(t *testing.T) {
	v := NewOllamaEmbedVectorizer("http://example.invalid", "any")
	_, err := v.Vectorize("   ")
	if err == nil || !strings.Contains(err.Error(), "empty text") {
		t.Errorf("expected empty text error, got %v", err)
	}
}

func TestOllamaVectorize_EmptyModelRejected(t *testing.T) {
	v := NewOllamaEmbedVectorizer("http://example.invalid", "")
	_, err := v.Vectorize("hello")
	if err == nil || !strings.Contains(err.Error(), "ModelID required") {
		t.Errorf("expected ModelID required error, got %v", err)
	}
}

// ─────────────────────────────────────
// Meta() before / after measurement
// ─────────────────────────────────────

func TestOllamaVectorize_MetaBeforeMeasurement(t *testing.T) {
	v := NewOllamaEmbedVectorizer("http://example.invalid", "test-model")
	m := v.Meta()
	if m.Type != "dense" {
		t.Errorf("expected dense, got %q", m.Type)
	}
	if m.ModelID != "ollama-test-model" {
		t.Errorf("expected ollama-test-model, got %q", m.ModelID)
	}
	if m.Dimension != 0 {
		t.Errorf("Dimension before measurement should be 0, got %d", m.Dimension)
	}
}

func TestOllamaVectorize_MetaReflectsMeasurement(t *testing.T) {
	body := `{"embeddings": [[0.1, 0.2, 0.3, 0.4]]}`
	srv := newMockServer(t, http.StatusOK, body)
	defer srv.Close()
	v := NewOllamaEmbedVectorizer(srv.URL, "test")
	_, _ = v.Vectorize("x")
	if v.Meta().Dimension != 4 {
		t.Errorf("Meta().Dimension should reflect 4, got %d", v.Meta().Dimension)
	}
}

// ─────────────────────────────────────
// JSON request body shape sanity check
// ─────────────────────────────────────

func TestOllamaVectorize_RequestBodyShape(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"embeddings":[[0.5]]}`))
	}))
	defer srv.Close()
	v := NewOllamaEmbedVectorizer(srv.URL, "nomic-embed-text")
	_, _ = v.Vectorize("hello world")
	if captured["model"] != "nomic-embed-text" {
		t.Errorf("model field wrong: %v", captured["model"])
	}
	if captured["input"] != "hello world" {
		t.Errorf("input field wrong: %v", captured["input"])
	}
}
