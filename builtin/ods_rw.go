// ods_rw.go — OpenDocument Spreadsheet (.ods) 讀寫。
// .ods 是 zip 包裝，表格資料在 content.xml。
package builtin

import (
	"fmt"
	"html"
	"strings"
)

// ExtractOdsText 從 .ods 檔案抽取純文字。
// 每個 table-row 變一行，cell 內文字以空白串接。
func ExtractOdsText(path string) (string, error) {
	// 讀取 zip 內的 content.xml
	data, err := zipReadFile(path, "content.xml")
	if err != nil {
		return "", fmt.Errorf("ods: 無法開啟 zip: %w", err)
	}
	if data == nil {
		return "", fmt.Errorf("ods: 找不到 content.xml")
	}

	// ODS 文字在 <text:p> 內，段落分隔以 <table:table-row> 為界
	ns := "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
	text := xmlExtractText(data, ns, "p", "table-row")

	// 備援：namespace 不匹配時用 local name
	if strings.TrimSpace(text) == "" {
		text = xmlExtractText(data, "", "p", "table-row")
	}

	return text, nil
}

// GenerateOds 建立最小 .ods 檔案。
// content 格式：tab 分隔欄位，換行分隔列。
func GenerateOds(content string, destPath string) error {
	// 拆解每列每欄，組裝 table XML
	rows := strings.Split(content, "\n")
	var tableBody strings.Builder
	for _, row := range rows {
		// 空列也產生空 row 節點
		tableBody.WriteString("<table:table-row>\n")
		cells := strings.Split(row, "\t")
		for _, cell := range cells {
			// 每個 cell 用 <text:p> 包文字
			tableBody.WriteString("  <table:table-cell><text:p>")
			tableBody.WriteString(html.EscapeString(cell))
			tableBody.WriteString("</text:p></table:table-cell>\n")
		}
		tableBody.WriteString("</table:table-row>\n")
	}

	contentXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content
  xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0">
<office:body><office:spreadsheet>
<table:table table:name="Sheet1">
` + tableBody.String() + `</table:table>
</office:spreadsheet></office:body>
</office:document-content>`

	// manifest.xml 宣告內容類型
	manifestXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0">
  <manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.spreadsheet"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
</manifest:manifest>`

	// 寫出 zip
	entries := map[string]string{
		"mimetype":              "application/vnd.oasis.opendocument.spreadsheet",
		"META-INF/manifest.xml": manifestXML,
		"content.xml":           contentXML,
	}

	if err := writeMinimalZip(destPath, entries); err != nil {
		return fmt.Errorf("ods: 寫入 zip 失敗: %w", err)
	}
	return nil
}
