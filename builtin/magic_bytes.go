// magic_bytes.go — 檔案格式 magic bytes 驗證。
// 後端二次驗證：防止副檔名偽裝（例如 .txt 改名為 .docx）。
// 策略：淺層快速檢查開頭 bytes，深層由解析器自然驗證。
package builtin

import (
	"fmt"
	"os"
	"strings"
)

// ValidateMagicBytes 檢查檔案開頭是否符合預期格式。
// 純文字格式（txt/md/csv/tsv）不做驗證，直接通過。
func ValidateMagicBytes(path, expectedFormat string) error {
	format := strings.ToLower(expectedFormat)

	// 純文字格式：無 magic bytes 可驗
	switch format {
	case "txt", "md", "csv", "tsv":
		return nil
	}

	// 讀取前 16 bytes 做快速檢查
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("magic_bytes: open %s: %w", path, err)
	}
	defer f.Close()

	head := make([]byte, 16)
	n, err := f.Read(head)
	if err != nil || n < 4 {
		return fmt.Errorf("magic_bytes: 檔案太小或無法讀取")
	}
	head = head[:n]

	switch format {
	// zip 系：docx/xlsx/pptx/odt/ods/odp/epub 都以 PK\x03\x04 開頭
	case "docx", "xlsx", "pptx", "odt", "ods", "odp", "epub":
		if head[0] != 'P' || head[1] != 'K' || head[2] != 3 || head[3] != 4 {
			return fmt.Errorf("magic_bytes: %s 預期 zip 格式（PK header）但開頭不符", format)
		}

	// JSON：第一個非空白字元是 { 或 [
	case "json":
		trimmed := strings.TrimLeft(string(head), " \t\n\r")
		if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
			return fmt.Errorf("magic_bytes: json 預期 { 或 [ 開頭")
		}

	// HTML：包含 < 開頭（寬鬆匹配）
	case "html", "htm":
		trimmed := strings.TrimLeft(string(head), " \t\n\r\xEF\xBB\xBF") // 跳過 BOM
		if len(trimmed) == 0 || trimmed[0] != '<' {
			return fmt.Errorf("magic_bytes: html 預期 < 開頭")
		}

	// RTF：開頭 {\rtf
	case "rtf":
		if !strings.HasPrefix(string(head), "{\\rtf") {
			return fmt.Errorf("magic_bytes: rtf 預期 {\\rtf 開頭")
		}
	}

	return nil
}
