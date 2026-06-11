// vector_y_test.go — Phase B Y' Vector / Vectorizer 行為測試。
//
// 覆蓋：
//   - Vector.Validate：sparse / dense / 雙 backing / 缺 Type 各情境
//   - Vector.Cosine：相容回正數；不相容回 ErrIncompatibleVector
//   - Vector.UnmarshalJSON：legacy flat map → Sparse 自動包；新 schema → 直接 round-trip
//   - chunkParagraphHybrid：段落 / 超長段 / heading / 純空白
//   - IndexNeedsRebuild：模型不同 / hash 不同 / chunker 版本不同
package builtin

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// ─────────────────────────────────────
// Vector.Validate
// ─────────────────────────────────────

func TestVectorValidate(t *testing.T) {
	cases := []struct {
		name    string
		v       Vector
		wantErr bool
	}{
		{"sparse ok", Vector{Sparse: map[string]float64{"a": 1}, Meta: VectorMetadata{Type: "sparse"}}, false},
		{"dense ok", Vector{Dense: []float64{0.1, 0.2}, Meta: VectorMetadata{Type: "dense"}}, false},
		{"quantized dense ok", Vector{DenseQ: &QuantizedDense{Values: []int8{127, -127}, Scale: 0.01}, Meta: VectorMetadata{Type: "dense", Dimension: 2}}, false},
		{"sparse but Sparse nil", Vector{Meta: VectorMetadata{Type: "sparse"}}, true},
		{"sparse and Dense both set", Vector{Sparse: map[string]float64{"a": 1}, Dense: []float64{0.1}, Meta: VectorMetadata{Type: "sparse"}}, true},
		{"dense and DenseQ both set", Vector{Dense: []float64{0.1}, DenseQ: &QuantizedDense{Values: []int8{10}, Scale: 0.01}, Meta: VectorMetadata{Type: "dense"}}, true},
		{"dense length != declared dim", Vector{Dense: []float64{0.1, 0.2}, Meta: VectorMetadata{Type: "dense", Dimension: 5}}, true},
		{"unknown type", Vector{Meta: VectorMetadata{Type: "weird"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.v.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// ─────────────────────────────────────
// Vector.Cosine
// ─────────────────────────────────────

func TestVectorCosineSparseCompatible(t *testing.T) {
	a := Vector{Sparse: map[string]float64{"x": 0.6, "y": 0.8}, Meta: VectorMetadata{Type: "sparse", ModelID: "tfidf-v1"}}
	b := Vector{Sparse: map[string]float64{"x": 0.6, "y": 0.8}, Meta: VectorMetadata{Type: "sparse", ModelID: "tfidf-v1"}}
	score, err := a.Cosine(b)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// 0.36 + 0.64 = 1.0（自己對自己歸一化過）
	if score < 0.99 || score > 1.01 {
		t.Errorf("expected ~1.0, got %v", score)
	}
}

func TestVectorCosineDenseCompatible(t *testing.T) {
	meta := VectorMetadata{Type: "dense", ModelID: "test", Dimension: 3}
	a := Vector{Dense: []float64{1, 0, 0}, Meta: meta}
	b := Vector{Dense: []float64{1, 0, 0}, Meta: meta}
	score, err := a.Cosine(b)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if score < 0.99 {
		t.Errorf("identical dense should cosine ~1, got %v", score)
	}
}

func TestVectorCosineQuantizedDenseCompatible(t *testing.T) {
	meta := VectorMetadata{Type: "dense", ModelID: "test", Dimension: 3}
	query := Vector{Dense: []float64{1, 0, -1}, Meta: meta}
	stored := QuantizeDenseForStorage(Vector{Dense: []float64{1, 0, -1}, Meta: meta})
	if stored.Dense != nil || stored.DenseQ == nil {
		t.Fatalf("expected dense_q storage, got %+v", stored)
	}
	score, err := query.Cosine(stored)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if score < 0.999 {
		t.Errorf("quantized dense should cosine ~1, got %v", score)
	}
}

func TestVectorCosineTypeMismatch(t *testing.T) {
	a := Vector{Sparse: map[string]float64{"x": 1}, Meta: VectorMetadata{Type: "sparse"}}
	b := Vector{Dense: []float64{1, 0}, Meta: VectorMetadata{Type: "dense", Dimension: 2}}
	_, err := a.Cosine(b)
	if !errors.Is(err, ErrIncompatibleVector) {
		t.Errorf("expected ErrIncompatibleVector, got %v", err)
	}
}

func TestVectorCosineDenseModelIDMismatch(t *testing.T) {
	a := Vector{Dense: []float64{1, 0}, Meta: VectorMetadata{Type: "dense", ModelID: "modelA", Dimension: 2}}
	b := Vector{Dense: []float64{1, 0}, Meta: VectorMetadata{Type: "dense", ModelID: "modelB", Dimension: 2}}
	_, err := a.Cosine(b)
	if !errors.Is(err, ErrIncompatibleVector) {
		t.Errorf("expected ErrIncompatibleVector, got %v", err)
	}
}

func TestVectorCosineDenseDimensionMismatch(t *testing.T) {
	a := Vector{Dense: []float64{1, 0}, Meta: VectorMetadata{Type: "dense", ModelID: "m", Dimension: 2}}
	b := Vector{Dense: []float64{1, 0, 0}, Meta: VectorMetadata{Type: "dense", ModelID: "m", Dimension: 3}}
	_, err := a.Cosine(b)
	if !errors.Is(err, ErrIncompatibleVector) {
		t.Errorf("expected ErrIncompatibleVector, got %v", err)
	}
}

// ─────────────────────────────────────
// Vector.UnmarshalJSON 相容
// ─────────────────────────────────────

func TestVectorUnmarshalLegacyFlatMap(t *testing.T) {
	// 舊版 v1：chunk.vector 直接是 flat sparse map
	legacy := []byte(`{"hello": 0.5, "world": 0.5}`)
	var v Vector
	if err := json.Unmarshal(legacy, &v); err != nil {
		t.Fatalf("legacy unmarshal failed: %v", err)
	}
	if v.Meta.Type != "sparse" {
		t.Errorf("expected sparse, got %q", v.Meta.Type)
	}
	if v.Meta.ModelID != legacyTFIDFModelID {
		t.Errorf("expected ModelID %q, got %q", legacyTFIDFModelID, v.Meta.ModelID)
	}
	if len(v.Sparse) != 2 || v.Sparse["hello"] != 0.5 {
		t.Errorf("sparse map content wrong: %+v", v.Sparse)
	}
	if v.Dense != nil {
		t.Errorf("Dense should be nil")
	}
}

func TestVectorUnmarshalNewSchemaRoundTrip(t *testing.T) {
	orig := Vector{
		Sparse: map[string]float64{"x": 0.6, "y": 0.8},
		Meta:   VectorMetadata{Type: "sparse", ModelID: "tfidf-v1"},
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Vector
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Meta.Type != "sparse" || back.Meta.ModelID != "tfidf-v1" {
		t.Errorf("meta lost: %+v", back.Meta)
	}
	if back.Sparse["x"] != 0.6 || back.Sparse["y"] != 0.8 {
		t.Errorf("sparse lost: %+v", back.Sparse)
	}
}

func TestVectorUnmarshalDenseRoundTrip(t *testing.T) {
	orig := Vector{
		Dense: []float64{0.1, 0.2, 0.3},
		Meta:  VectorMetadata{Type: "dense", ModelID: "ollama-nomic", Dimension: 3},
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Vector
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Meta.Dimension != 3 || back.Meta.ModelID != "ollama-nomic" {
		t.Errorf("meta lost: %+v", back.Meta)
	}
	if len(back.Dense) != 3 || back.Dense[1] != 0.2 {
		t.Errorf("dense lost: %+v", back.Dense)
	}
}

func TestVectorQuantizedDenseJSONRoundTrip(t *testing.T) {
	orig := QuantizeDenseForStorage(Vector{
		Dense: []float64{0.25, -0.5, 1},
		Meta:  VectorMetadata{Type: "dense", ModelID: "ollama-nomic", Dimension: 3},
	})
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(data)
	if strings.Contains(raw, `"dense":`) {
		t.Fatalf("quantized storage should omit dense floats: %s", raw)
	}
	if !strings.Contains(raw, `"dense_q":`) {
		t.Fatalf("quantized storage missing dense_q: %s", raw)
	}
	var back Vector
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := back.Validate(); err != nil {
		t.Fatalf("validate back: %v", err)
	}
	score, err := orig.Cosine(back)
	if err != nil {
		t.Fatalf("cosine: %v", err)
	}
	if score < 0.999 {
		t.Fatalf("round-trip cosine = %v", score)
	}
}

// ─────────────────────────────────────
// chunkParagraphHybrid
// ─────────────────────────────────────

func TestChunkParagraphHybrid(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantMin  int // 至少幾個 chunk
		wantMax  int // 至多幾個 chunk
		notEmpty bool
	}{
		{"empty", "", 0, 0, false},
		{"single short paragraph", "這是短段落", 1, 1, true},
		{"two paragraphs joined into one chunk", "短段落 A\n\n短段落 B", 1, 1, true},
		{"single overflow paragraph splits", strings.Repeat("中文。", 800), 2, 10, true},
		{"markdown headings split", "# 標題一\n\n內容 A\n\n# 標題二\n\n內容 B", 2, 6, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := chunkParagraphHybrid(tc.input)
			if tc.notEmpty && len(out) == 0 {
				t.Errorf("expected non-empty output")
			}
			if len(out) < tc.wantMin || (tc.wantMax > 0 && len(out) > tc.wantMax) {
				t.Errorf("chunks=%d, want between [%d, %d]; output=%v", len(out), tc.wantMin, tc.wantMax, out)
			}
			for i, c := range out {
				if strings.TrimSpace(c) == "" {
					t.Errorf("chunk %d is empty after trim", i)
				}
			}
		})
	}
}

func TestChunkParagraphHybridRespectsMaxRunes(t *testing.T) {
	// 一段大文本，期望每個 chunk rune 數 ≤ chunkMaxRunes（容許句切粒度的些許 overshoot）
	input := strings.Repeat("這是一個句子。", 500) // ~3500 runes
	chunks := chunkParagraphHybrid(input)
	for i, c := range chunks {
		if runeLen(c) > chunkMaxRunes+50 { // 50 字 grace 給 sentence-split 粒度
			t.Errorf("chunk %d has %d runes, max %d", i, runeLen(c), chunkMaxRunes)
		}
	}
}

// ─────────────────────────────────────
// IndexNeedsRebuild
// ─────────────────────────────────────

func TestIndexNeedsRebuild(t *testing.T) {
	tfidf := TFIDFVectorizer{}
	const hash = "abcd1234"
	freshIndex := DocumentVectorIndex{
		VectorMeta:     tfidf.Meta(),
		ContentHash:    hash,
		ChunkerVersion: ChunkerVersion,
	}

	cases := []struct {
		name        string
		modify      func(*DocumentVectorIndex)
		queryHash   string
		wantRebuild bool
	}{
		{"all match", func(*DocumentVectorIndex) {}, hash, false},
		{"chunker version drift", func(i *DocumentVectorIndex) { i.ChunkerVersion = "para-hybrid-v0" }, hash, true},
		{"content hash change", func(*DocumentVectorIndex) {}, "different-hash", true},
		{"missing meta (legacy)", func(i *DocumentVectorIndex) { i.VectorMeta = VectorMetadata{} }, hash, true},
		{"model id drift", func(i *DocumentVectorIndex) { i.VectorMeta.ModelID = "tfidf-v0" }, hash, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx := freshIndex
			tc.modify(&idx)
			got := IndexNeedsRebuild(idx, tfidf, tc.queryHash)
			if got != tc.wantRebuild {
				t.Errorf("IndexNeedsRebuild = %v, want %v", got, tc.wantRebuild)
			}
		})
	}
}

func TestIndexNeedsRebuildDenseRequiresQuantizedStorage(t *testing.T) {
	vec := testDenseVectorizer{meta: VectorMetadata{Type: "dense", ModelID: "dense-test", Dimension: 3}}
	const hash = "dense-hash"
	unquantized := DocumentVectorIndex{
		VectorMeta:     vec.Meta(),
		ContentHash:    hash,
		ChunkerVersion: ChunkerVersion,
		Chunks: []DocumentChunk{{
			Vec: Vector{Dense: []float64{1, 0, -1}, Meta: vec.Meta()},
		}},
	}
	if !IndexNeedsRebuild(unquantized, vec, hash) {
		t.Fatalf("dense index with float JSON storage should rebuild into dense_q")
	}

	quantized := unquantized
	quantized.Chunks[0].Vec = QuantizeDenseForStorage(quantized.Chunks[0].Vec)
	if IndexNeedsRebuild(quantized, vec, hash) {
		t.Fatalf("dense_q index should not rebuild when metadata/hash match")
	}
}

type testDenseVectorizer struct {
	meta VectorMetadata
}

func (v testDenseVectorizer) Vectorize(string) (Vector, error) {
	return Vector{Dense: []float64{1, 0, -1}, Meta: v.meta}, nil
}

func (v testDenseVectorizer) Meta() VectorMetadata {
	return v.meta
}

// ─────────────────────────────────────
// TFIDFVectorizer 介面契約
// ─────────────────────────────────────

func TestTFIDFVectorizerContract(t *testing.T) {
	var v Vectorizer = TFIDFVectorizer{}
	out, err := v.Vectorize("hello world hello")
	if err != nil {
		t.Fatalf("vectorize: %v", err)
	}
	if out.Meta.Type != "sparse" {
		t.Errorf("expected sparse, got %q", out.Meta.Type)
	}
	if out.Meta.ModelID != TFIDFModelID {
		t.Errorf("expected modelID %q, got %q", TFIDFModelID, out.Meta.ModelID)
	}
	if err := out.Validate(); err != nil {
		t.Errorf("Vectorize output failed Validate: %v", err)
	}
}
