// odt_rw.go — OpenDocument Text (.odt) 讀寫。
// .odt 是 zip 包裝，主要內容在 content.xml。
package builtin

import (
	"fmt"
	"html"
	"strings"
)

// ExtractOdtText 從 .odt 檔案抽取純文字。
func ExtractOdtText(path string) (string, error) {
	// 讀取 zip 內的 content.xml
	data, err := zipReadFile(path, "content.xml")
	if err != nil {
		return "", fmt.Errorf("odt: 無法開啟 zip: %w", err)
	}
	if data == nil {
		return "", fmt.Errorf("odt: 找不到 content.xml")
	}

	// ODF 文字在 <text:p> 節點內，用完整 namespace 抽取
	ns := "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
	text := xmlExtractText(data, ns, "p", "p")

	// 備援：namespace 不匹配時改用 local name 比對
	if strings.TrimSpace(text) == "" {
		text = xmlExtractText(data, "", "p", "p")
	}

	return text, nil
}

// GenerateOdt 建立最小 .odt 檔案，content 每行一段。
func GenerateOdt(content string, destPath string) error {
	// 組裝 content.xml — 每行變成 <text:p>
	lines := strings.Split(content, "\n")
	var body strings.Builder
	for _, ln := range lines {
		// 跳脫 XML 特殊字元
		body.WriteString("<text:p>")
		body.WriteString(html.EscapeString(ln))
		body.WriteString("</text:p>\n")
	}

	contentXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content
  xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
<office:body><office:text>
` + body.String() + `</office:text></office:body>
</office:document-content>`

	// manifest.xml 宣告內容類型
	manifestXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0">
  <manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
</manifest:manifest>`

	// 建立 zip 寫入各 entry
	entries := map[string]string{
		"mimetype":              "application/vnd.oasis.opendocument.text",
		"META-INF/manifest.xml": manifestXML,
		"content.xml":           contentXML,
	}

	if err := writeMinimalZip(destPath, entries); err != nil {
		return fmt.Errorf("odt: 寫入 zip 失敗: %w", err)
	}
	return nil
}
