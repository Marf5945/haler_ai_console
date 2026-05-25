package builtin

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	documentChunkRunes   = 900
	documentChunkOverlap = 140
	defaultVectorLimit   = 5
)

type DocumentChunk struct {
	DocID   string             `json:"doc_id"`
	ChunkID string             `json:"chunk_id"`
	Index   int                `json:"index"`
	Text    string             `json:"text"`
	Vector  map[string]float64 `json:"vector"`
}

type DocumentVectorIndex struct {
	SchemaVersion string          `json:"schema_version"`
	DocID         string          `json:"doc_id"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Chunks        []DocumentChunk `json:"chunks"`
}

type DocumentSearchResult struct {
	DocID       string  `json:"doc_id"`
	DisplayName string  `json:"display_name"`
	Format      string  `json:"format"`
	ChunkID     string  `json:"chunk_id"`
	Snippet     string  `json:"snippet"`
	Score       float64 `json:"score"`
	W3AID       string  `json:"w3a_id"`
}

func BuildAndSaveVectorIndex(store *Store, blob *DocumentBlob) error {
	chunks := BuildDocumentChunks(blob.Meta.DocID, blob.Content)
	index := DocumentVectorIndex{
		SchemaVersion: "document_vector_index.v1",
		DocID:         blob.Meta.DocID,
		UpdatedAt:     time.Now(),
		Chunks:        chunks,
	}
	if err := os.MkdirAll(store.VectorsDir(), 0o700); err != nil {
		return fmt.Errorf("document_vector: mkdir vectors: %w", err)
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("document_vector: marshal: %w", err)
	}
	if err := os.WriteFile(vectorIndexPath(store, blob.Meta.DocID), data, 0o600); err != nil {
		return fmt.Errorf("document_vector: write index: %w", err)
	}
	blob.Meta.ChunkCount = len(chunks)
	blob.Meta.VectorIndexedAt = index.UpdatedAt
	return nil
}

func BuildDocumentChunks(docID, content string) []DocumentChunk {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) == 0 {
		return nil
	}
	step := documentChunkRunes - documentChunkOverlap
	if step <= 0 {
		step = documentChunkRunes
	}
	var chunks []DocumentChunk
	for start, index := 0, 0; start < len(runes); start, index = start+step, index+1 {
		end := start + documentChunkRunes
		if end > len(runes) {
			end = len(runes)
		}
		text := strings.TrimSpace(string(runes[start:end]))
		if text != "" {
			chunks = append(chunks, DocumentChunk{
				DocID:   docID,
				ChunkID: fmt.Sprintf("%s-chunk-%03d", docID, index),
				Index:   index,
				Text:    text,
				Vector:  textVector(text),
			})
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func SearchDocuments(store *Store, query string, limit int) ([]DocumentSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultVectorLimit
	}
	queryVector := textVector(query)
	metas, err := store.List()
	if err != nil {
		return nil, err
	}
	metaByID := make(map[string]DocMeta, len(metas))
	for _, meta := range metas {
		metaByID[meta.DocID] = meta
	}

	entries, err := os.ReadDir(store.VectorsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("document_vector: list indexes: %w", err)
	}
	var results []DocumentSearchResult
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(store.VectorsDir(), entry.Name()))
		if err != nil {
			continue
		}
		var index DocumentVectorIndex
		if err := json.Unmarshal(data, &index); err != nil {
			continue
		}
		meta, ok := metaByID[index.DocID]
		if !ok {
			continue
		}
		for _, chunk := range index.Chunks {
			score := cosine(queryVector, chunk.Vector)
			if score <= 0 {
				continue
			}
			results = append(results, DocumentSearchResult{
				DocID:       index.DocID,
				DisplayName: meta.DisplayName,
				Format:      meta.Format,
				ChunkID:     chunk.ChunkID,
				Snippet:     snippet(chunk.Text),
				Score:       score,
				W3AID:       meta.W3AID,
			})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DisplayName < results[j].DisplayName
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func vectorIndexPath(store *Store, docID string) string {
	return filepath.Join(store.VectorsDir(), filepath.Base(docID)+".json")
}

func textVector(text string) map[string]float64 {
	counts := make(map[string]float64)
	for _, token := range tokenizeForVector(text) {
		counts[token]++
	}
	var norm float64
	for _, count := range counts {
		norm += count * count
	}
	if norm == 0 {
		return counts
	}
	norm = math.Sqrt(norm)
	for token, count := range counts {
		counts[token] = count / norm
	}
	return counts
}

func tokenizeForVector(text string) []string {
	var tokens []string
	var word []rune
	var cjk []rune
	flushWord := func() {
		if len(word) > 0 {
			tokens = append(tokens, strings.ToLower(string(word)))
			word = word[:0]
		}
	}
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			flushWord()
			tokens = append(tokens, string(r))
			cjk = append(cjk, r)
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, unicode.ToLower(r))
			continue
		}
		flushWord()
	}
	flushWord()
	for i := 0; i+1 < len(cjk); i++ {
		tokens = append(tokens, string(cjk[i:i+2]))
	}
	return tokens
}

func cosine(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var score float64
	for token, av := range a {
		score += av * b[token]
	}
	return score
}

func snippet(text string) string {
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if utf8.RuneCountInString(text) <= 180 {
		return text
	}
	runes := []rune(text)
	return string(runes[:180]) + "..."
}
