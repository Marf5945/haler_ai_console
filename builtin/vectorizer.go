// vectorizer.go — 向量化介面（Phase B Y' 統一設計）。
//
// 一句話原則：所有向量都用 Vector struct 包；sparse / dense 兩種 backing 互斥。
// Vector.Cosine() 內部自己 dispatch + 相容性檢查；caller 看不到 dispatch code。
//
// 目前實作：
//   - TFIDFVectorizer（sparse，零外部依賴，預設）
//   - 未來：OllamaEmbedVectorizer（dense，HTTP 打 ollama）
//
// 向後相容：
//   - 舊索引 JSON 內 `vector: {詞: weight}` 走 Vector.UnmarshalJSON 自動包成
//     `Vector{Sparse: ..., Meta: {Type: "sparse"}}`，不需要遷移檔案。
package builtin

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// ErrIncompatibleVector 是 Cosine 在 metadata 不相容時回的 sentinel。
// SearchDocumentsInDir 收到這個 error 就跳過該 chunk / index，不視為 fatal。
var ErrIncompatibleVector = errors.New("vector: incompatible metadata")

// VectorMetadata 描述向量來自哪個 vectorizer。
// 相容性檢查靠 Type / ModelID / Dimension 三者比對。
type VectorMetadata struct {
	Type      string `json:"type"`                // "sparse" | "dense"
	ModelID   string `json:"model_id,omitempty"`  // 例 "tfidf-v1" / "ollama-nomic-embed-text"
	Dimension int    `json:"dimension,omitempty"` // dense 才有意義（768 / 1024 …）
}

// Vector 是 sparse / dense 的 tagged union。
// 不變式：Sparse 和 Dense 恰好只有一邊有值（Validate() 強制）。
type Vector struct {
	Sparse map[string]float64 `json:"sparse,omitempty"`
	Dense  []float64          `json:"dense,omitempty"`
	Meta   VectorMetadata     `json:"meta"`
}

// Validate 檢查不變式：Type 必須是 sparse/dense；對應的 backing 不可空。
// 由 Vectorizer 實作在 Vectorize 結束前自己呼叫；caller 也可用來 sanity check 從 disk 讀回的索引。
func (v Vector) Validate() error {
	switch v.Meta.Type {
	case "sparse":
		if v.Sparse == nil {
			return errors.New("vector: sparse type but Sparse is nil")
		}
		if v.Dense != nil {
			return errors.New("vector: sparse type must not have Dense set")
		}
	case "dense":
		if v.Dense == nil {
			return errors.New("vector: dense type but Dense is nil")
		}
		if v.Sparse != nil {
			return errors.New("vector: dense type must not have Sparse set")
		}
		if v.Meta.Dimension > 0 && len(v.Dense) != v.Meta.Dimension {
			return fmt.Errorf("vector: dense length %d != declared dimension %d", len(v.Dense), v.Meta.Dimension)
		}
	default:
		return fmt.Errorf("vector: unknown type %q", v.Meta.Type)
	}
	return nil
}

// IsEmpty 判斷是否為零值 Vector（無 Type、無 backing）。用於 JSON 反序列化檢查。
func (v Vector) IsEmpty() bool {
	return v.Meta.Type == "" && len(v.Sparse) == 0 && len(v.Dense) == 0
}

// Cosine 計算兩個 Vector 的 cosine similarity。
// 相容性規則：Type 必須相同；dense 還要求 ModelID + Dimension 都對得上。
// 不相容回 ErrIncompatibleVector，caller 應跳過該對比、不視為 fatal。
func (v Vector) Cosine(other Vector) (float64, error) {
	if v.Meta.Type != other.Meta.Type {
		return 0, ErrIncompatibleVector
	}
	switch v.Meta.Type {
	case "sparse":
		// sparse TF-IDF 不要求 model_id 相同（同一個 tokenizer 就 OK）。
		return cosineSparse(v.Sparse, other.Sparse), nil
	case "dense":
		if v.Meta.ModelID != other.Meta.ModelID {
			return 0, ErrIncompatibleVector
		}
		if len(v.Dense) != len(other.Dense) {
			return 0, ErrIncompatibleVector
		}
		return cosineDense(v.Dense, other.Dense), nil
	default:
		return 0, ErrIncompatibleVector
	}
}

func cosineSparse(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var score float64
	for token, av := range a {
		score += av * b[token]
	}
	return score
}

func cosineDense(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// UnmarshalJSON 接受兩種 schema：
//  1. 新版：{"sparse": {...} | "dense": [...], "meta": {...}}
//  2. 舊版：{"詞": 0.12, ...}  ← TF-IDF v1 索引直接是 flat map
//
// 偵測規則：有 "meta" key 走新版；否則當 legacy flat sparse map 包進來。
func (v *Vector) UnmarshalJSON(data []byte) error {
	// probe "meta" 是否存在（不解析其他欄位以保持快速）
	var probe struct {
		Meta *VectorMetadata `json:"meta"`
	}
	_ = json.Unmarshal(data, &probe)
	if probe.Meta != nil {
		type alias Vector
		var a alias
		if err := json.Unmarshal(data, &a); err != nil {
			return err
		}
		*v = Vector(a)
		return nil
	}
	// Legacy: flat map → 包成 sparse Vector
	var sparse map[string]float64
	if err := json.Unmarshal(data, &sparse); err != nil {
		return fmt.Errorf("vector: cannot parse as new or legacy schema: %w", err)
	}
	*v = Vector{
		Sparse: sparse,
		Meta:   VectorMetadata{Type: "sparse", ModelID: legacyTFIDFModelID},
	}
	return nil
}

// legacyTFIDFModelID 是讀到舊索引時自動標記的 ModelID，方便升級後分辨。
const legacyTFIDFModelID = "tfidf-legacy"

// TFIDFModelID 是新建 TF-IDF 索引時用的 ModelID（換 tokenizer 就 bump 版本）。
const TFIDFModelID = "tfidf-v1"

// Vectorizer 將文字轉為 Vector，並能自報 metadata（讓索引記錄是誰建的）。
type Vectorizer interface {
	Vectorize(text string) (Vector, error)
	Meta() VectorMetadata
}

// TFIDFVectorizer — 基於詞頻的本地稀疏向量化（零依賴）。
// 內部呼叫 textVector（在 document_vector.go 內）產生 sparse map。
type TFIDFVectorizer struct{}

// Vectorize 回傳歸一化 TF 稀疏向量包成 Vector。
func (TFIDFVectorizer) Vectorize(text string) (Vector, error) {
	sparse := textVector(text)
	v := Vector{
		Sparse: sparse,
		Meta:   VectorMetadata{Type: "sparse", ModelID: TFIDFModelID},
	}
	// Validate 在這裡其實一定過——保留以便未來換 tokenizer 時及早抓 bug。
	if err := v.Validate(); err != nil {
		return Vector{}, err
	}
	return v, nil
}

// Meta 回 TF-IDF 自報的 metadata。
func (TFIDFVectorizer) Meta() VectorMetadata {
	return VectorMetadata{Type: "sparse", ModelID: TFIDFModelID}
}
