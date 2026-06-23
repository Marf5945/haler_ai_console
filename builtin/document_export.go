// document_export.go — 文件匯出。
// 從 JSON blob 還原成真實檔案（支援 13 種格式）。
// 匯出到 temp 目錄 → 交給 native drag 處理拖曳。
// 匯出到指定路徑 → Zone B 需使用者確認。
package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExportToTemp 將文件匯出到暫存目錄，回傳暫存檔案路徑。
// 用於 native drag：先匯出到 temp，再由 NSFilePromiseProvider 複製到使用者目標。
func ExportToTemp(store *Store, docID string) (string, error) {
	blob, err := store.Load(docID)
	if err != nil {
		return "", fmt.Errorf("document_export: load %s: %w", docID, err)
	}

	// 建立暫存目錄
	tempDir := filepath.Join(os.TempDir(), "ai-console-doc-export")
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		return "", fmt.Errorf("document_export: mkdir temp: %w", err)
	}

	// 產生輸出檔案名稱（用原始 display name）
	fileName := sanitizeFileName(blob.Meta.DisplayName)
	destPath := filepath.Join(tempDir, fileName)

	if blob.Meta.OriginalPath != "" {
		if _, err := os.Stat(blob.Meta.OriginalPath); err == nil {
			_ = os.Remove(destPath)
			if err := copyRegularFileOverwrite(blob.Meta.OriginalPath, destPath); err != nil {
				return "", err
			}
			_ = copySidecarIfPresentOverwrite(blob.Meta.OriginalPath, destPath)
			return destPath, nil
		}
	}

	// 依格式匯出
	if err := writeExportFile(blob, destPath); err != nil {
		return "", err
	}
	return destPath, nil
}

// ExportToPath 將文件匯出到指定路徑。
// 呼叫端必須先透過 PathGuard 驗證，且 Zone B 需使用者確認。
func ExportToPath(store *Store, guard *PathGuard, docID, destDir string) (string, error) {
	// 驗證匯出路徑
	if err := guard.ValidateExportPath(destDir); err != nil {
		return "", err
	}

	blob, err := store.Load(docID)
	if err != nil {
		return "", fmt.Errorf("document_export: load %s: %w", docID, err)
	}

	fileName := sanitizeFileName(blob.Meta.DisplayName)
	destPath := filepath.Join(destDir, fileName)

	if blob.Meta.OriginalPath != "" {
		if _, err := os.Stat(blob.Meta.OriginalPath); err == nil {
			if err := copyRegularFileOverwrite(blob.Meta.OriginalPath, destPath); err != nil {
				return "", err
			}
			_ = copySidecarIfPresentOverwrite(blob.Meta.OriginalPath, destPath)
			return destPath, nil
		}
	}

	if err := writeExportFile(blob, destPath); err != nil {
		return "", err
	}
	return destPath, nil
}

// writeExportFile 依格式寫出檔案。
func writeExportFile(blob *DocumentBlob, destPath string) error {
	format := strings.ToLower(blob.Meta.Format)

	// 確保副檔名與格式一致的 helper
	ensureExt := func(ext string) string {
		if !strings.HasSuffix(strings.ToLower(destPath), ext) {
			return strings.TrimSuffix(destPath, filepath.Ext(destPath)) + ext
		}
		return destPath
	}

	switch format {
	case "txt", "md":
		// 純文字直接寫入
		if err := os.WriteFile(destPath, []byte(blob.Content), 0o600); err != nil {
			return fmt.Errorf("document_export: write %s: %w", destPath, err)
		}
		return nil

	case "csv":
		// CSV：tab→逗號
		return WriteCSV(blob.Content, ensureExt(".csv"), ',')

	case "tsv":
		// TSV：tab 分隔
		return WriteCSV(blob.Content, ensureExt(".tsv"), '\t')

	case "json":
		// JSON：美化輸出
		return WriteJSONPretty(blob.Content, ensureExt(".json"))

	case "html", "htm":
		// HTML：包成簡單網頁
		title := blob.Meta.DisplayName
		return GenerateHTML(blob.Content, title, ensureExt(".html"))

	case "rtf":
		// RTF 富文字
		return GenerateRTF(blob.Content, ensureExt(".rtf"))

	case "docx":
		// OOXML Word
		return GenerateDocx(blob.Content, ensureExt(".docx"))

	case "xlsx":
		// OOXML Excel（content 是 tab 分隔格式）
		return GenerateXlsx(blob.Content, ensureExt(".xlsx"))

	case "pptx":
		// OOXML PowerPoint
		return GeneratePptx(ensureExt(".pptx"), blob.Content)

	case "odt":
		// OpenDocument Text
		return GenerateOdt(blob.Content, ensureExt(".odt"))

	case "ods":
		// OpenDocument Spreadsheet
		return GenerateOds(blob.Content, ensureExt(".ods"))

	case "odp":
		// OpenDocument Presentation
		return GenerateOdp(blob.Content, ensureExt(".odp"))

	case "epub":
		// EPUB 僅支援匯入，不支援匯出
		return fmt.Errorf("document_export: epub 僅支援匯入，不支援匯出")

	default:
		return fmt.Errorf("document_export: 不支援的匯出格式 %q", format)
	}
}

// sanitizeFileName 清理檔案名稱，移除不安全字元。
func sanitizeFileName(name string) string {
	if name == "" {
		return "untitled.txt"
	}
	// 移除路徑分隔符和常見不安全字元
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
