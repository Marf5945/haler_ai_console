// document_vector.go — 文件拆 chunk + TF-IDF 向量索引 + cosine 搜尋。
package builtin

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	chunkMaxRunes      = 1200 // 單一 chunk 上限（rune 計）；Y' 預設
	chunkMinRunes      = 200  // 太小的尾段如果跟前段合併會超界就還是獨立成 chunk
	defaultVectorLimit = 5
)

// 保留舊命名以免外部 link（雖然此 repo 內無 caller），避免下游 break。
const (
	documentChunkRunes   = chunkMaxRunes
	documentChunkOverlap = 0 // 段落 hybrid 不再用 overlap；保留欄位避免 break
)

type DocumentChunk struct {
	DocID       string `json:"doc_id"`
	ChunkID     string `json:"chunk_id"`
	Index       int    `json:"index"`
	Text        string `json:"text"`
	ContentHash string `json:"content_hash,omitempty"` // SHA256 of Text，用於 hash diff 增量更新
	Vec         Vector `json:"vector"`                 // JSON key 沿用 "vector" 以相容舊索引
}

// UnmarshalJSON 對 DocumentChunk 做向後相容：舊版 chunk JSON 內 "vector" 是
// flat map[string]float64；Vector.UnmarshalJSON 已能消化兩種 schema，所以這裡
// 預設行為就夠用。本方法明確列在這裡只是文件用途——未來改 Field 名再啟動。

type DocumentVectorIndex struct {
	SchemaVersion  string          `json:"schema_version"`
	DocID          string          `json:"doc_id"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Chunks         []DocumentChunk `json:"chunks"`
	VectorMeta     VectorMetadata  `json:"vector_meta,omitempty"`     // 哪個 vectorizer 建的
	ContentHash    string          `json:"content_hash,omitempty"`    // 整檔 SHA256
	ChunkerVersion string          `json:"chunker_version,omitempty"` // 切塊演算法版本
}

// ChunkerVersion — 切塊演算法版本標記。換 chunker 邏輯時 bump，所有舊索引自動視為過期重建。
const ChunkerVersion = "para-hybrid-v1"

// IndexNeedsRebuild 比對現有索引與當前 vectorizer/chunker/content hash，回傳是否該重建。
// 沒讀到 metadata（舊索引）或任一不符 → 重建。
func IndexNeedsRebuild(existing DocumentVectorIndex, vec Vectorizer, contentHash string) bool {
	if existing.ChunkerVersion != ChunkerVersion {
		return true
	}
	if existing.ContentHash == "" || existing.ContentHash != contentHash {
		return true
	}
	want := vec.Meta()
	got := existing.VectorMeta
	if want.Type != got.Type || want.ModelID != got.ModelID {
		return true
	}
	if want.Type == "dense" && want.Dimension != got.Dimension {
		return true
	}
	if want.Type == "dense" && !indexUsesQuantizedDense(existing) {
		return true
	}
	return false
}

func indexUsesQuantizedDense(index DocumentVectorIndex) bool {
	for _, chunk := range index.Chunks {
		if chunk.Vec.Meta.Type != "dense" {
			continue
		}
		if chunk.Vec.DenseQ == nil || len(chunk.Vec.DenseQ.Values) == 0 {
			return false
		}
	}
	return true
}

type DocumentSearchResult struct {
	DocID       string  `json:"doc_id"`
	DisplayName string  `json:"display_name"`
	Format      string  `json:"format"`
	ChunkID     string  `json:"chunk_id"`
	Snippet     string  `json:"snippet"`
	Score       float64 `json:"score"`
	W3AID       string  `json:"w3a_id"`
	Source      string  `json:"source"` // "document" 或 "reference"
}

// BuildAndSaveVectorIndex 為文件建索引並寫入 JSON。
func BuildAndSaveVectorIndex(store *Store, blob *DocumentBlob, vec Vectorizer) error {
	chunks, err := BuildDocumentChunks(blob.Meta.DocID, blob.Content, vec)
	if err != nil {
		return fmt.Errorf("document_vector: build chunks: %w", err)
	}
	index := DocumentVectorIndex{
		SchemaVersion:  "document_vector_index.v2",
		DocID:          blob.Meta.DocID,
		UpdatedAt:      time.Now(),
		Chunks:         quantizeDenseChunksForStorage(chunks),
		VectorMeta:     vec.Meta(),
		ContentHash:    sha256Hex(blob.Content),
		ChunkerVersion: ChunkerVersion,
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

// BuildAndSaveVectorIndexToDir 對任意目錄建索引（用於 references/vectors/）。
func BuildAndSaveVectorIndexToDir(vectorsDir, docID, content string, vec Vectorizer) error {
	chunks, err := BuildDocumentChunks(docID, content, vec)
	if err != nil {
		return fmt.Errorf("document_vector: build chunks: %w", err)
	}
	index := DocumentVectorIndex{
		SchemaVersion:  "document_vector_index.v2",
		DocID:          docID,
		UpdatedAt:      time.Now(),
		Chunks:         quantizeDenseChunksForStorage(chunks),
		VectorMeta:     vec.Meta(),
		ContentHash:    sha256Hex(content),
		ChunkerVersion: ChunkerVersion,
	}
	if err := os.MkdirAll(vectorsDir, 0o700); err != nil {
		return fmt.Errorf("document_vector: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("document_vector: marshal: %w", err)
	}
	path := filepath.Join(vectorsDir, filepath.Base(docID)+".json")
	return os.WriteFile(path, data, 0o600)
}

func quantizeDenseChunksForStorage(chunks []DocumentChunk) []DocumentChunk {
	for i := range chunks {
		chunks[i].Vec = QuantizeDenseForStorage(chunks[i].Vec)
	}
	return chunks
}

// BuildDocumentChunks 用「段落 hybrid」策略拆 chunk 並向量化。
//
// 策略：
//  1. 先依「空行或 markdown heading」切段落。
//  2. 段落 rune 數 ≤ chunkMaxRunes 直接整段當一個 chunk；
//     多個小段落會被攢起來，直到「加下一段就超 chunkMaxRunes」才 flush。
//  3. 單一段落超 chunkMaxRunes 時，依中英句末標點（。！？.!?）切，
//     再 group 到不超過 chunkMaxRunes。
//  4. ChunkerVersion 被 bump 時所有舊索引視為過期。
//
// 為什麼不直接用 LLM 判斷最長句？M1 先 deterministic 穩定；之後 M3 可選擇性升級。
func BuildDocumentChunks(docID, content string, vec Vectorizer) ([]DocumentChunk, error) {
	pieces := chunkParagraphHybrid(content)
	if len(pieces) == 0 {
		return nil, nil
	}
	chunks := make([]DocumentChunk, 0, len(pieces))
	for i, text := range pieces {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		vecOut, err := vec.Vectorize(text)
		if err != nil {
			return nil, fmt.Errorf("vectorize chunk %d: %w", i, err)
		}
		chunks = append(chunks, DocumentChunk{
			DocID:       docID,
			ChunkID:     fmt.Sprintf("%s-chunk-%03d", docID, i),
			Index:       i,
			Text:        text,
			ContentHash: sha256Hex(text),
			Vec:         vecOut,
		})
	}
	return chunks, nil
}

// chunkParagraphHybrid 是 Phase B Y' 的切塊核心；純文字進、字串切片出，不碰 vectorizer。
// 公開供 test 使用。
func chunkParagraphHybrid(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	paragraphs := splitParagraphs(content)
	var out []string
	var buf strings.Builder
	bufRunes := 0
	flush := func() {
		if bufRunes > 0 {
			out = append(out, strings.TrimSpace(buf.String()))
			buf.Reset()
			bufRunes = 0
		}
	}
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		paraRunes := runeLen(para)
		// markdown heading 強制成新 chunk 起點：先把前一段落區塊封口。
		// 這樣 retrieval 拉到「# 標題二」就只會帶到它底下的內容，不會混到上面那節。
		if isMarkdownHeading(para) {
			flush()
		}
		// 單段就超界 → flush 現有 + 段內按句切
		if paraRunes > chunkMaxRunes {
			flush()
			for _, frag := range splitBySentence(para, chunkMaxRunes) {
				out = append(out, frag)
			}
			continue
		}
		// 加進 buf 會超界 → flush 再開新一塊
		if bufRunes > 0 && bufRunes+paraRunes+2 > chunkMaxRunes {
			flush()
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
			bufRunes += 2
		}
		buf.WriteString(para)
		bufRunes += paraRunes
	}
	flush()
	return out
}

// splitParagraphs 依空行（連續 \n）切段落。Markdown heading（# / ## / ...）會單獨成段。
func splitParagraphs(content string) []string {
	// 標準化 line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	rawParts := strings.Split(content, "\n\n")
	var out []string
	for _, p := range rawParts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// markdown heading 行單獨成段（# / ## / ### + 空格）
		lines := strings.Split(p, "\n")
		var buf strings.Builder
		for _, line := range lines {
			tl := strings.TrimSpace(line)
			if isMarkdownHeading(tl) {
				if buf.Len() > 0 {
					out = append(out, strings.TrimSpace(buf.String()))
					buf.Reset()
				}
				out = append(out, tl)
				continue
			}
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(line)
		}
		if buf.Len() > 0 {
			out = append(out, strings.TrimSpace(buf.String()))
		}
	}
	return out
}

// splitBySentence 把超長段落按中英句末標點切，再 group 到 ≤ maxRunes。
// 不丟資料：標點本身保留在前段。
func splitBySentence(para string, maxRunes int) []string {
	if runeLen(para) <= maxRunes {
		return []string{para}
	}
	// 先切句
	var sentences []string
	var cur strings.Builder
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			sentences = append(sentences, s)
		}
		cur.Reset()
	}
	for _, r := range para {
		cur.WriteRune(r)
		switch r {
		case '。', '！', '？', '.', '!', '?', '\n':
			flush()
		}
	}
	flush()
	// 再 group
	var out []string
	var buf strings.Builder
	bufRunes := 0
	flushBuf := func() {
		if bufRunes > 0 {
			out = append(out, strings.TrimSpace(buf.String()))
			buf.Reset()
			bufRunes = 0
		}
	}
	for _, s := range sentences {
		sRunes := runeLen(s)
		if sRunes > maxRunes {
			// 連單句都超界 → 硬切成 maxRunes 一塊
			flushBuf()
			runes := []rune(s)
			for i := 0; i < len(runes); i += maxRunes {
				end := i + maxRunes
				if end > len(runes) {
					end = len(runes)
				}
				out = append(out, string(runes[i:end]))
			}
			continue
		}
		if bufRunes > 0 && bufRunes+sRunes+1 > maxRunes {
			flushBuf()
		}
		if buf.Len() > 0 {
			buf.WriteString(" ")
			bufRunes++
		}
		buf.WriteString(s)
		bufRunes += sRunes
	}
	flushBuf()
	return out
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// isMarkdownHeading 認得 # / ## / ... / ###### 後接空格的標題行。
// 不是 markdown 但長得像的也算（無傷大雅，最多多切一段）。
func isMarkdownHeading(line string) bool {
	if line == "" || line[0] != '#' {
		return false
	}
	i := 0
	for i < len(line) && i < 6 && line[i] == '#' {
		i++
	}
	return i > 0 && i < len(line) && line[i] == ' '
}

// SearchDocuments 在 store 的向量索引中搜尋，回傳 top-N 結果。
func SearchDocuments(store *Store, query string, vec Vectorizer, limit int) ([]DocumentSearchResult, error) {
	return SearchDocumentsInDir(store.VectorsDir(), query, vec, limit, func(docID string) (string, string, string) {
		metas, err := store.List()
		if err != nil {
			return docID, "", ""
		}
		for _, m := range metas {
			if m.DocID == docID {
				return m.DisplayName, m.Format, m.W3AID
			}
		}
		return docID, "", ""
	}, "document")
}

// ── 向量索引記憶體快取（stage 0：只解速度瓶頸，不改索引格式、不加依賴）──
// 動機：SearchDocumentsInDir 舊版每次查詢都重讀＋重新 json.Unmarshal 整個語料，
// 解析才是熱路徑成本（cosine 數學是零頭）。這裡把解析後的 index 快取在記憶體，
// 用檔案 mtime+size 當失效鍵——只 stat、不讀內容；重建索引會改 mtime，快取自動失效。
// LRU 上限避免跟著使用者成長到無限大。回傳的 index 之 Chunks 與快取共享底層陣列，
// 搜尋只讀不改，故共享安全。
const vectorIndexCacheMaxEntries = 512

type vectorIndexCacheEntry struct {
	modTime time.Time
	size    int64
	index   DocumentVectorIndex
	elem    *list.Element // 在 LRU 佇列中的位置（Value 存檔案路徑）
}

var vectorIndexCache = struct {
	sync.Mutex
	m  map[string]*vectorIndexCacheEntry
	ll *list.List // 前=最近使用、後=最久未用
}{m: make(map[string]*vectorIndexCacheEntry), ll: list.New()}

// loadVectorIndexCached 回傳解析後索引：命中且 mtime+size 未變就免碰磁碟。
func loadVectorIndexCached(path string, info os.FileInfo) (DocumentVectorIndex, error) {
	vectorIndexCache.Lock()
	defer vectorIndexCache.Unlock()

	if e, ok := vectorIndexCache.m[path]; ok && e.modTime.Equal(info.ModTime()) && e.size == info.Size() {
		vectorIndexCache.ll.MoveToFront(e.elem) // 標記最近使用
		return e.index, nil
	}

	// miss / 過期：讀檔重新解析
	data, err := os.ReadFile(path)
	if err != nil {
		return DocumentVectorIndex{}, err
	}
	var idx DocumentVectorIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return DocumentVectorIndex{}, err
	}

	if e, ok := vectorIndexCache.m[path]; ok {
		e.modTime, e.size, e.index = info.ModTime(), info.Size(), idx
		vectorIndexCache.ll.MoveToFront(e.elem)
	} else {
		e := &vectorIndexCacheEntry{modTime: info.ModTime(), size: info.Size(), index: idx}
		e.elem = vectorIndexCache.ll.PushFront(path)
		vectorIndexCache.m[path] = e
		// 超過上限就淘汰最久未用的一筆
		for vectorIndexCache.ll.Len() > vectorIndexCacheMaxEntries {
			oldest := vectorIndexCache.ll.Back()
			if oldest == nil {
				break
			}
			vectorIndexCache.ll.Remove(oldest)
			delete(vectorIndexCache.m, oldest.Value.(string))
		}
	}
	return idx, nil
}

// SearchDocumentsInDir 在任意 vectors 目錄搜尋（通用）。
func SearchDocumentsInDir(vectorsDir, query string, vec Vectorizer, limit int, metaLookup func(string) (string, string, string), source string) ([]DocumentSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultVectorLimit
	}
	queryVec, err := vec.Vectorize(query)
	if err != nil {
		return nil, fmt.Errorf("document_vector: vectorize query: %w", err)
	}
	entries, err := os.ReadDir(vectorsDir)
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
		// 走記憶體快取：mtime+size 未變就免讀磁碟、免重解析（解析才是熱路徑瓶頸）。
		info, ierr := entry.Info()
		if ierr != nil {
			continue
		}
		index, lerr := loadVectorIndexCached(filepath.Join(vectorsDir, entry.Name()), info)
		if lerr != nil {
			continue
		}
		// 整個 index 的 vectorizer 跟 query 不符就跳過——不要花時間算每個 chunk。
		if !indexMetaCompatible(queryVec.Meta, index.VectorMeta) {
			continue
		}
		displayName, format, w3aID := metaLookup(index.DocID)
		for _, chunk := range index.Chunks {
			score, cerr := queryVec.Cosine(chunk.Vec)
			if cerr != nil || score <= 0 {
				continue
			}
			results = append(results, DocumentSearchResult{
				DocID:       index.DocID,
				DisplayName: displayName,
				Format:      format,
				ChunkID:     chunk.ChunkID,
				Snippet:     snippet(chunk.Text),
				Score:       score,
				W3AID:       w3aID,
				Source:      source,
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

// VectorIndexPath 公開版，供外部清理索引。
func VectorIndexPath(store *Store, docID string) string {
	return vectorIndexPath(store, docID)
}

// textVector 產生歸一化 TF 稀疏向量（內部使用，由 TFIDFVectorizer 呼叫）。
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

func snippet(text string) string {
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if utf8.RuneCountInString(text) <= 180 {
		return text
	}
	runes := []rune(text)
	return string(runes[:180]) + "..."
}

// sha256Hex 回 hex(SHA256(s))。用於 content hash diff。
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// indexMetaCompatible 在 search loop 內快速判斷一個 index 是否值得繼續算 cosine。
// 規則：Type 必須相同；dense 額外要求 ModelID + Dimension 對得上；
//
//	legacy / 缺 Type 的舊索引一律當 sparse 處理（兼容 v1）。
func indexMetaCompatible(queryMeta, indexMeta VectorMetadata) bool {
	qt := queryMeta.Type
	it := indexMeta.Type
	if it == "" {
		it = "sparse" // legacy v1
	}
	if qt != it {
		return false
	}
	if qt == "dense" {
		if queryMeta.ModelID != indexMeta.ModelID {
			return false
		}
		if queryMeta.Dimension != indexMeta.Dimension {
			return false
		}
	}
	return true
}
