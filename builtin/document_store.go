// document_store.go — 合併式 JSON blob 儲存服務。
// 每個文件存成一個 {doc_id}.json，集中在 documents/ 目錄下。
// 所有公開方法持有 mu 鎖，可安全用於多 goroutine。
package builtin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Store 管理 documents/ 目錄下的所有 DocumentBlob。
type Store struct {
	mu       sync.Mutex
	docsDir  string // documents/ 的絕對路徑
}

// NewStore 建立儲存服務。docsDir 不存在時會自動建立。
func NewStore(docsDir string) (*Store, error) {
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		return nil, fmt.Errorf("builtin.store: mkdir %s: %w", docsDir, err)
	}
	if err := os.MkdirAll(filepath.Join(docsDir, "originals"), 0o700); err != nil {
		return nil, fmt.Errorf("builtin.store: mkdir originals: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(docsDir, "vectors"), 0o700); err != nil {
		return nil, fmt.Errorf("builtin.store: mkdir vectors: %w", err)
	}
	return &Store{docsDir: docsDir}, nil
}

func (s *Store) OriginalsDir() string {
	return filepath.Join(s.docsDir, "originals")
}

func (s *Store) VectorsDir() string {
	return filepath.Join(s.docsDir, "vectors")
}

// Save 寫入一個 DocumentBlob。若 DocID 已存在則覆蓋。
// 自動更新 UpdatedAt、ContentHash、WordCount。
func (s *Store) Save(blob *DocumentBlob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 確保 schema version
	blob.SchemaVersion = "document_blob.v1"

	// 自動計算 hash 和字數
	blob.Meta.ContentHash = contentHash(blob.Content)
	blob.Meta.WordCount = countWords(blob.Content)
	blob.Meta.UpdatedAt = time.Now()

	// 序列化寫入
	data, err := json.MarshalIndent(blob, "", "  ")
	if err != nil {
		return fmt.Errorf("builtin.store: marshal %s: %w", blob.Meta.DocID, err)
	}
	path := s.blobPath(blob.Meta.DocID)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("builtin.store: write %s: %w", path, err)
	}
	return nil
}

// Load 讀取單一 DocumentBlob（含 content）。
func (s *Store) Load(docID string) (*DocumentBlob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.blobPath(docID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("builtin.store: read %s: %w", docID, err)
	}
	var blob DocumentBlob
	if err := json.Unmarshal(data, &blob); err != nil {
		return nil, fmt.Errorf("builtin.store: parse %s: %w", docID, err)
	}
	return &blob, nil
}

// List 回傳所有文件的 DocMeta（不載入 content，節省記憶體）。
func (s *Store) List() ([]DocMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.docsDir)
	if err != nil {
		return nil, fmt.Errorf("builtin.store: list %s: %w", s.docsDir, err)
	}

	var metas []DocMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.docsDir, entry.Name()))
		if err != nil {
			continue // 跳過讀取失敗的檔案，不中斷整個列表
		}
		// 只解析 meta 欄位，不解析 content（省記憶體）
		var partial struct {
			Meta DocMeta `json:"meta"`
		}
		if err := json.Unmarshal(data, &partial); err != nil {
			continue
		}
		metas = append(metas, partial.Meta)
	}
	return metas, nil
}

func (s *Store) FindByContentHash(hash string) (*DocumentBlob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, false, nil
	}
	entries, err := os.ReadDir(s.docsDir)
	if err != nil {
		return nil, false, fmt.Errorf("builtin.store: list %s: %w", s.docsDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.docsDir, entry.Name()))
		if err != nil {
			continue
		}
		var blob DocumentBlob
		if err := json.Unmarshal(data, &blob); err != nil {
			continue
		}
		if blob.Meta.ContentHash == hash {
			return &blob, true, nil
		}
	}
	return nil, false, nil
}

// Delete 刪除文件 blob 並清理對應的向量索引。
func (s *Store) Delete(docID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.blobPath(docID)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("builtin.store: delete %s: %w", docID, err)
	}
	// 清理向量索引（忽略不存在的情況）
	_ = os.Remove(filepath.Join(s.VectorsDir(), filepath.Base(docID)+".json"))
	return nil
}

// RemoveVectorIndex 只清理向量索引，不刪 blob。
func (s *Store) RemoveVectorIndex(docID string) {
	_ = os.Remove(filepath.Join(s.VectorsDir(), filepath.Base(docID)+".json"))
}

// blobPath 回傳 docID 對應的 JSON 檔案路徑。
func (s *Store) blobPath(docID string) string {
	docID = filepath.Base(docID) // SEC-13: 防止路徑穿越
	return filepath.Join(s.docsDir, docID+".json")
}

// contentHash 計算 SHA-256。
func contentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// countWords 計算字數。
// 中文：每個 rune 算一字。英文：以空白分隔。混合文本取較大值。
func countWords(content string) int {
	// 簡單策略：rune 數量（對中文友好，英文會偏高但可接受）
	return utf8.RuneCountInString(strings.TrimSpace(content))
}
