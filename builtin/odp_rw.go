// odp_rw.go — OpenDocument Presentation (.odp) 讀寫。
// .odp 是 zip 包裝，簡報內容在 content.xml。
package builtin

import (
	"fmt"
	"html"
	"strings"
)

// ExtractOdpText 從 .odp 檔案抽取純文字。
// 每個 draw:page 觸發段落分隔，文字從 <text:p> 抽取。
func ExtractOdpText(path string) (string, error) {
	// 讀取 zip 內的 content.xml
	data, err := zipReadFile(path, "content.xml")
	if err != nil {
		return "", fmt.Errorf("odp: 無法開啟 zip: %w", err)
	}
	if data == nil {
		return "", fmt.Errorf("odp: 找不到 content.xml")
	}

	// ODP 文字在 <text:p> 內，以 <draw:page> 分段
	ns := "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
	text := xmlExtractText(data, ns, "p", "page")

	// 備援：namespace 不匹配時用 local name
	if strings.TrimSpace(text) == "" {
		text = xmlExtractText(data, "", "p", "page")
	}

	return text, nil
}

// GenerateOdp 建立最小單頁 .odp 檔案，content 每行一段。
func GenerateOdp(content string, destPath string) error {
	// 每行變成 <text:p>
	lines := strings.Split(content, "\n")
	var paras strings.Builder
	for _, ln := range lines {
		// 跳脫 XML 特殊字元
		paras.WriteString("      <text:p>")
		paras.WriteString(html.EscapeString(ln))
		paras.WriteString("</text:p>\n")
	}

	// 組裝 content.xml — 單一 slide 含一個 text-box
	contentXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content
  xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0"
  xmlns:presentation="urn:oasis:names:tc:opendocument:xmlns:presentation:1.0"
  xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0">
<office:body><office:presentation>
<draw:page draw:name="Slide1">
  <draw:frame draw:name="content" svg:x="2cm" svg:y="2cm" svg:width="20cm" svg:height="14cm">
    <draw:text-box>
` + paras.String() + `    </draw:text-box>
  </draw:frame>
</draw:page>
</office:presentation></office:body>
</office:document-content>`

	// manifest.xml 宣告內容類型
	manifestXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0">
  <manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.presentation"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
</manifest:manifest>`

	// 寫出 zip
	entries := map[string]string{
		"mimetype":              "application/vnd.oasis.opendocument.presentation",
		"META-INF/manifest.xml": manifestXML,
		"content.xml":           contentXML,
	}

	if err := writeMinimalZip(destPath, entries); err != nil {
		return fmt.Errorf("odp: 寫入 zip 失敗: %w", err)
	}
	return nil
}
