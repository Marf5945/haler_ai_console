// document_import.go — 拖入匯入主流程。
// 使用者將檔案拖入 UI → 系統辨識格式 → 編碼轉換 → 存成 DocumentBlob。
// 支援 13 種格式：txt, md, csv, tsv, json, html, docx, xlsx, pptx, odt, ods, odp, epub。
package builtin

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// SupportedImportFormats 支援匯入的格式。
var SupportedImportFormats = map[string]bool{
	".txt":  true,
	".md":   true,
	".csv":  true,
	".tsv":  true,
	".json": true,
	".html": true,
	".htm":  true,
	".docx": true,
	".xlsx": true,
	".pptx": true,
	".odt":  true,
	".ods":  true,
	".odp":  true,
	".epub": true,
}

// ImportResult 匯入結果，前端用來顯示 toast。
type ImportResult struct {
	Blob          *DocumentBlob `json:"blob"`
	Encoding      string        `json:"encoding"`       // 偵測到的原始編碼
	SizeClass     SizeClass     `json:"size_class"`     // 檔案大小等級
	AlreadyCached bool          `json:"already_cached"` // 內容已存在於文件快取
}

// ImportFromDrop 從拖入的檔案路徑匯入文件。
// 完整流程：驗證路徑 → 辨識格式 → 讀取+編碼偵測 → 組裝 blob → 存入 store。
func ImportFromDrop(store *Store, guard *PathGuard, filePath string, vec Vectorizer) (*ImportResult, error) {
	// 1. 路徑安全驗證
	if err := guard.ValidateImportPath(filePath); err != nil {
		return nil, err
	}

	// 2. 辨識格式
	ext := strings.ToLower(filepath.Ext(filePath))
	if !SupportedImportFormats[ext] {
		return nil, fmt.Errorf("document_import: 不支援的格式 %q", ext)
	}

	// 2.5 magic bytes 二次驗證（防副檔名偽裝）
	format := strings.TrimPrefix(ext, ".")
	if err := ValidateMagicBytes(filePath, format); err != nil {
		return nil, fmt.Errorf("document_import: %w", err)
	}

	// 3. 依格式讀取內容
	var content string
	var detectedEncoding string
	var sizeClass SizeClass

	switch ext {
	case ".txt", ".md":
		// 純文字：讀 raw bytes + 編碼偵測
		raw, sc, err := ReadWithGuard(filePath, 0)
		if err != nil {
			return nil, fmt.Errorf("document_import: read %s: %w", filePath, err)
		}
		sizeClass = sc
		converted, encoding, err := DetectAndConvert(raw)
		if err != nil {
			return nil, fmt.Errorf("document_import: encoding %s: %w", filePath, err)
		}
		content = converted
		detectedEncoding = encoding

	case ".csv":
		// CSV：用 encoding/csv 解析
		text, err := ReadCSV(filePath, ',')
		if err != nil {
			return nil, fmt.Errorf("document_import: csv %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "csv"
		sizeClass = SizeInline

	case ".tsv":
		// TSV：tab 分隔
		text, err := ReadCSV(filePath, '\t')
		if err != nil {
			return nil, fmt.Errorf("document_import: tsv %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "tsv"
		sizeClass = SizeInline

	case ".json":
		// JSON：遞迴抽所有字串值
		text, err := ExtractJSONText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: json %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "json-utf8"
		sizeClass = SizeInline

	case ".html", ".htm":
		// HTML：去 tag 取純文字
		text, err := ExtractHTMLTextFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: html %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "html"
		sizeClass = SizeInline

	case ".docx":
		// OOXML Word：抽 <w:t> 文字
		text, err := ExtractDocxText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: docx %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "docx-xml"
		sizeClass = SizeInline

	case ".xlsx":
		// OOXML Excel：抽 sharedStrings 文字
		text, err := ExtractXlsxText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: xlsx %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "xlsx-xml"
		sizeClass = SizeInline

	case ".pptx":
		// OOXML PowerPoint：抽所有投影片 <a:t>
		text, err := ExtractPptxText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: pptx %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "pptx-xml"
		sizeClass = SizeInline

	case ".odt":
		// OpenDocument Text
		text, err := ExtractOdtText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: odt %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "odt-xml"
		sizeClass = SizeInline

	case ".ods":
		// OpenDocument Spreadsheet
		text, err := ExtractOdsText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: ods %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "ods-xml"
		sizeClass = SizeInline

	case ".odp":
		// OpenDocument Presentation
		text, err := ExtractOdpText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: odp %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "odp-xml"
		sizeClass = SizeInline

	case ".epub":
		// EPUB 電子書
		text, err := ExtractEpubText(filePath)
		if err != nil {
			return nil, fmt.Errorf("document_import: epub %s: %w", filePath, err)
		}
		content = text
		detectedEncoding = "epub-xhtml"
		sizeClass = SizeInline
	}

	// 4. 組裝 DocumentBlob
	now := time.Now()
	docID := fmt.Sprintf("doc-%d", now.UnixNano())
	contentHashValue := contentHash(content)
	if existing, ok, err := store.FindByContentHash(contentHashValue); err != nil {
		return nil, err
	} else if ok {
		return &ImportResult{
			Blob:          existing,
			Encoding:      detectedEncoding,
			SizeClass:     sizeClass,
			AlreadyCached: true,
		}, nil
	}

	displayName, cachedPath, originalHash, w3aID, err := cacheOriginalFile(store, filePath)
	if err != nil {
		return nil, err
	}

	blob := &DocumentBlob{
		SchemaVersion: "document_blob.v1",
		Meta: DocMeta{
			DocID:        docID,
			DisplayName:  displayName,
			Format:       format,
			CreatedAt:    now,
			ContentHash:  contentHashValue,
			OriginalHash: originalHash,
			OriginalPath: cachedPath,
			W3AID:        w3aID,
			SourceHint:   fmt.Sprintf("drag_import from %s", filepath.Base(filePath)),
			Tags:         []string{"cached_original"},
		},
		Content: content,
	}

	if err := BuildAndSaveVectorIndex(store, blob, vec); err != nil {
		return nil, fmt.Errorf("document_import: vector index: %w", err)
	}

	// 5. 存入 store（Save 會自動算 hash、word count、updated_at）
	if err := store.Save(blob); err != nil {
		return nil, fmt.Errorf("document_import: save: %w", err)
	}

	return &ImportResult{
		Blob:      blob,
		Encoding:  detectedEncoding,
		SizeClass: sizeClass,
	}, nil
}

// IsSupportedFormat 快速判斷副檔名是否支援匯入。
// 前端 drop zone 在拖入時可先呼叫此函式過濾。
func IsSupportedFormat(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return SupportedImportFormats[ext]
}
