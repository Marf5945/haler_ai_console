// csv_rw.go — CSV/TSV 讀寫。
// Import：讀取所有儲存格，以 tab 分隔欄位、換行分隔列。
// Export：從純文字（每行 tab 分隔）產生 CSV/TSV 檔案。
// 依賴：encoding/csv（Go 標準庫）。
package builtin

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// ReadCSV 讀取 CSV 或 TSV 檔案，回傳純文字（tab 分隔欄位、\n 分隔列）。
// delimiter: ',' for CSV, '\t' for TSV。
func ReadCSV(path string, delimiter rune) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("csv_rw: open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = delimiter
	reader.LazyQuotes = true // 容忍不標準的引號（常見於使用者自建 CSV）

	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("csv_rw: parse %s: %w", path, err)
	}

	// 每列的欄位用 tab 連接，列之間用 \n
	var lines []string
	for _, row := range records {
		lines = append(lines, strings.Join(row, "\t"))
	}
	return strings.Join(lines, "\n"), nil
}

// WriteCSV 將純文字寫成 CSV 或 TSV 檔案。
// content 格式：tab 分隔欄位、\n 分隔列。
func WriteCSV(content string, path string, delimiter rune) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("csv_rw: create %s: %w", path, err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = delimiter

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if err := writer.Write(fields); err != nil {
			return fmt.Errorf("csv_rw: write row: %w", err)
		}
	}
	writer.Flush()
	return writer.Error()
}
