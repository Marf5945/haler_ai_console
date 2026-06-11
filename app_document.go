// app_document.go — 文件操作 Wails binding 方法。
// 提供 6 個前端可呼叫的文件方法。
// read/list = Zone A 自動，save/export = 需 review card 確認。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/builtin"
	"ui_console/data/storage"

	wailsRT "github.com/wailsapp/wails/v2/pkg/runtime"
)

// DocumentImportResult 前端用的匯入結果（含預覽資訊）。
type DocumentImportResult struct {
	DocID       string `json:"doc_id"`
	DisplayName string `json:"display_name"`
	Format      string `json:"format"`
	WordCount   int    `json:"word_count"`
	Encoding    string `json:"encoding"`
}

// DocumentPreview 寫入確認用的預覽資料。
type DocumentPreview struct {
	DisplayName string `json:"display_name"`
	Format      string `json:"format"`
	WordCount   int    `json:"word_count"`
	Preview     string `json:"preview"` // 前 500 字
	DocID       string `json:"doc_id"`
}

// HandleDocumentDrop 拖入檔案觸發匯入。
// 流程：magic bytes 驗證 → 格式解析 → 存 blob → emit event。
func (a *App) HandleDocumentDrop(filePath string) (*DocumentImportResult, error) {
	store := a.getDocumentStore()
	guard := a.getPathGuard()
	if store == nil || guard == nil {
		return nil, fmt.Errorf("document service 尚未初始化")
	}

	a.maybeEmitConfigMissing(filepath.Base(filePath))
	vec := a.currentVectorizer()
	result, err := builtin.ImportFromDrop(store, guard, filePath, vec)
	a.persistMeasuredDimensionIfNeeded(vec)
	if err != nil {
		return nil, err
	}

	// 通知前端匯入完成（顯示 toast）
	wailsRT.EventsEmit(a.ctx, "document:imported", map[string]interface{}{
		"doc_id":       result.Blob.Meta.DocID,
		"display_name": result.Blob.Meta.DisplayName,
		"format":       result.Blob.Meta.Format,
		"word_count":   result.Blob.Meta.WordCount,
		"encoding":     result.Encoding,
	})

	return &DocumentImportResult{
		DocID:       result.Blob.Meta.DocID,
		DisplayName: result.Blob.Meta.DisplayName,
		Format:      result.Blob.Meta.Format,
		WordCount:   result.Blob.Meta.WordCount,
		Encoding:    result.Encoding,
	}, nil
}

// ListProjectDocuments 列出 store 中所有文件 meta。
// Zone A 自動執行，不需確認。
// Keep the Wails boundary JSON-shaped: builtin.DocMeta contains time.Time and
// must not appear directly in generated bindings.
func (a *App) ListProjectDocuments() (interface{}, error) {
	store := a.getDocumentStore()
	if store == nil {
		return nil, fmt.Errorf("document service 尚未初始化")
	}
	metas, err := store.List()
	return frontendDTO(metas), err
}

// ReadDocumentContent 讀取文件全文內容。
// Zone A 自動執行，不需確認。
func (a *App) ReadDocumentContent(docID string) (string, error) {
	store := a.getDocumentStore()
	if store == nil {
		return "", fmt.Errorf("document service 尚未初始化")
	}
	blob, err := store.Load(docID)
	if err != nil {
		return "", err
	}
	return blob.Content, nil
}

// SaveDocumentFromAgent agent 觸發的文件儲存。
// 組裝 blob + emit review_needed event，前端確認後才真正存檔。
func (a *App) SaveDocumentFromAgent(content, displayName, format string) (*DocumentPreview, error) {
	store := a.getDocumentStore()
	if store == nil {
		return nil, fmt.Errorf("document service 尚未初始化")
	}

	// 先存入 store（預設狀態，前端確認後才算完成）
	blob := &builtin.DocumentBlob{
		Meta: builtin.DocMeta{
			DocID:       fmt.Sprintf("doc-%d", timeNowUnixNano()),
			DisplayName: displayName,
			Format:      format,
			Tags:        []string{"agent_generated"},
		},
		Content: content,
	}

	if err := store.Save(blob); err != nil {
		return nil, err
	}

	// 準備預覽（前 500 字）
	preview := content
	previewRunes := []rune(preview)
	if len(previewRunes) > 500 {
		preview = string(previewRunes[:500]) + "..."
	}

	previewData := &DocumentPreview{
		DisplayName: displayName,
		Format:      format,
		WordCount:   blob.Meta.WordCount,
		Preview:     preview,
		DocID:       blob.Meta.DocID,
	}

	// emit review_needed event → frontend DocumentReviewCard 顯示確認
	wailsRT.EventsEmit(a.ctx, "document:review_needed", previewData)

	return previewData, nil
}

// ExportDocumentToPath 匯出文件到暫存目錄（供 native drag）。
func (a *App) ExportDocumentToPath(docID string) (string, error) {
	store := a.getDocumentStore()
	if store == nil {
		return "", fmt.Errorf("document service 尚未初始化")
	}
	return builtin.ExportToTemp(store, docID)
}

// IsDocumentFormat 前端判斷拖入檔案是否為文件格式。
func (a *App) IsDocumentFormat(filePath string) bool {
	return builtin.IsSupportedFormat(filePath)
}

// --- 內部 helper ---

// getDocumentStore 取得 document store（延遲初始化）。
func (a *App) getDocumentStore() *builtin.Store {
	a.docOnce.Do(func() {
		root := appDataRoot()
		docsDir := filepath.Join(storage.ProjectRoot(root, "default"), "documents")
		store, err := builtin.NewStore(docsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: document store init failed: %v\n", err)
			return
		}
		a.documentStore = store
		a.pathGuard = builtin.NewPathGuard(storage.ProjectRoot(root, "default"))
	})
	return a.documentStore
}

func (a *App) getPathGuard() *builtin.PathGuard {
	a.getDocumentStore() // 確保已初始化
	return a.pathGuard
}

// timeNowUnixNano 用於產生 doc ID（可被 test mock）。
func timeNowUnixNano() int64 {
	return time.Now().UnixNano()
}

// maybeEmitConfigMissing：App 層檢查 EmbeddingConfig 並在必要時發 event。
// 故意在 App 層做，不下推到 builtin——builtin 不應該知道 eventBus / UI 概念。
//
// 規則：ProviderID 空 + PickerDismissed=false → 發 event 給前端開 modal。
// 不阻塞匯入流程：emit 完就 return，匯入照常走 TF-IDF。
//
// displayName 給前端 modal 顯示「是哪份檔觸發了這次提示」用。
func (a *App) maybeEmitConfigMissing(displayName string) {
	if a == nil || a.settingsService == nil || a.eventBus == nil {
		return
	}
	cfg := a.settingsService.EmbeddingConfig()
	if cfg.ProviderID != "" || cfg.PickerDismissed {
		return
	}
	a.eventBus.Emit("embedding:config_missing", map[string]any{
		"displayName": displayName,
	})
}

// currentVectorizer 從 settings 決定 ingest / search 用哪個 vectorizer。
//
// 規則：
//   - EmbeddingConfig.ProviderID == "ollama" 且 ModelID 非空 → OllamaEmbedVectorizer
//   - 其他（沒設定、不認得的 provider）→ TFIDFVectorizer（fallback）
//
// 不快取——每次呼叫重建（Ollama vectorizer 內部有 HTTP client，但 client 本身 reuse connection）。
// 重建成本可忽略；好處是使用者切換 model 後立即生效，不用重啟 app。
func (a *App) currentVectorizer() builtin.Vectorizer {
	if a.settingsService == nil {
		return builtin.TFIDFVectorizer{}
	}
	cfg := a.settingsService.EmbeddingConfig()
	if cfg.ProviderID == "ollama" && cfg.ModelID != "" {
		return builtin.NewOllamaEmbedVectorizer("", cfg.ModelID)
	}
	return builtin.TFIDFVectorizer{}
}

// persistMeasuredDimensionIfNeeded 把 Ollama vectorizer 量到的 dimension 寫回 settings。
// 故意分開呼叫——讓 currentVectorizer() 保持薄，只負責選擇器；測量寫回是 side-effect，由 ingest 流程明確觸發。
func (a *App) persistMeasuredDimensionIfNeeded(vec builtin.Vectorizer) {
	if a.settingsService == nil {
		return
	}
	ollama, ok := vec.(*builtin.OllamaEmbedVectorizer)
	if !ok {
		return
	}
	dim := ollama.MeasuredDimension()
	if dim <= 0 {
		return
	}
	a.settingsService.SaveEmbeddingDimension(dim)
}

// ensureReferenceVectorIndexes 啟動時掃描引用文件目錄，重建「缺索引 / metadata 不一致」者。
//
// Phase B Y' 升級：
//   - 索引存在但 ChunkerVersion / VectorMeta / ContentHash 任一不符 → 重建
//   - 索引不存在 → 建立
//   - 一致 → 跳過（最常見 path）
//
// vectorizer 目前固定 TFIDFVectorizer；M2 會從 settings 讀使用者選的 embed model。
func (a *App) ensureReferenceVectorIndexes() {
	root := appDataRoot()
	refDir := filepath.Join(root, "data", "references", "files")
	vecDir := filepath.Join(root, "data", "references", "vectors")

	entries, err := os.ReadDir(refDir)
	if err != nil {
		return // 目錄不存在則跳過
	}
	_ = os.MkdirAll(vecDir, 0o700)

	vec := a.currentVectorizer()
	defer a.persistMeasuredDimensionIfNeeded(vec)
	for _, entry := range entries {
		if entry.IsDir() || len(entry.Name()) == 0 || entry.Name()[0] == '.' {
			continue
		}
		filePath := filepath.Join(refDir, entry.Name())
		_ = a.indexReferenceFileIfNeeded(filePath, vecDir, vec)
	}
}

func (a *App) indexReferenceFileIfNeeded(filePath, vecDir string, vec builtin.Vectorizer) error {
	if !builtin.IsSearchableFormat(filePath) {
		return nil
	}
	content, err := builtin.ExtractSearchableText(filePath)
	if err != nil || strings.TrimSpace(content) == "" {
		return err
	}
	contentHash := sha256Hex64(content)
	indexPath := filepath.Join(vecDir, filepath.Base(filePath)+".json")
	if data, rerr := os.ReadFile(indexPath); rerr == nil {
		var existing builtin.DocumentVectorIndex
		if json.Unmarshal(data, &existing) == nil {
			if !builtin.IndexNeedsRebuild(existing, vec, contentHash) {
				return nil
			}
		}
	}
	return builtin.BuildAndSaveVectorIndexToDir(vecDir, filepath.Base(filePath), content, vec)
}

// sha256Hex64 — local helper mirroring builtin.sha256Hex（不匯出避免循環）。
func sha256Hex64(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ReferenceVectorsDir 回傳引用文件的向量索引目錄路徑。
func referenceVectorsDir() string {
	return filepath.Join(appDataRoot(), "data", "references", "vectors")
}
