// app_document.go — 文件操作 Wails binding 方法。
// 提供 6 個前端可呼叫的文件方法。
// read/list = Zone A 自動，save/export = 需 review card 確認。
package main

import (
	"fmt"
	"os"
	"path/filepath"
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

	result, err := builtin.ImportFromDrop(store, guard, filePath)
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
