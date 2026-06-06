// docx_write.go — 最小 .docx 模板產生。
// .docx 本質是 zip，裡面放幾個 XML 檔案。
// 這裡只產生能被 Word/Pages/LibreOffice 開啟的最小結構。
// 零第三方依賴：archive/zip + encoding/xml + html。
package builtin

import (
	"archive/zip"
	"fmt"
	"html"
	"os"
	"strings"
)

// GenerateDocx 從純文字產生最小 .docx 檔案。
// 每行文字 = 一個段落（<w:p>）。
// destPath 是輸出 .docx 的完整路徑。
func GenerateDocx(content string, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("docx_write: create %s: %w", destPath, err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// 1. [Content_Types].xml — 宣告 zip 內每種檔案的 content type
	if err := writeZipEntry(w, "[Content_Types].xml", contentTypesXML); err != nil {
		return err
	}

	// 2. _rels/.rels — 根關聯，指向 word/document.xml
	if err := writeZipEntry(w, "_rels/.rels", relsXML); err != nil {
		return err
	}

	// 3. word/_rels/document.xml.rels — 文件關聯（指向 styles.xml）
	if err := writeZipEntry(w, "word/_rels/document.xml.rels", documentRelsXML); err != nil {
		return err
	}

	// 4. word/styles.xml — 最小樣式（空但必須存在，否則 Word 會警告）
	if err := writeZipEntry(w, "word/styles.xml", stylesXML); err != nil {
		return err
	}

	// 5. word/document.xml — 主文件，包含實際文字
	docXML := buildDocumentXML(content)
	if err := writeZipEntry(w, "word/document.xml", docXML); err != nil {
		return err
	}

	return nil
}

// buildDocumentXML 將純文字轉成 document.xml。
// 每行 → 一個 <w:p> 段落；空行 → 空段落。
func buildDocumentXML(content string) string {
	lines := strings.Split(content, "\n")

	var body strings.Builder
	for _, line := range lines {
		escaped := html.EscapeString(line) // 處理 <>&" 等特殊字元
		body.WriteString(`<w:p><w:r><w:t xml:space="preserve">`)
		body.WriteString(escaped)
		body.WriteString(`</w:t></w:r></w:p>`)
	}

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas"` +
		` xmlns:mo="http://schemas.microsoft.com/office/mac/office/2008/main"` +
		` xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"` +
		` xmlns:mv="urn:schemas-microsoft-com:mac:vml"` +
		` xmlns:o="urn:schemas-microsoft-com:office:office"` +
		` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"` +
		` xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"` +
		` xmlns:v="urn:schemas-microsoft-com:vml"` +
		` xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"` +
		` xmlns:w10="urn:schemas-microsoft-com:office:word"` +
		` xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"` +
		` xmlns:wne="http://schemas.microsoft.com/office/word/2006/wordml">` +
		`<w:body>` + body.String() + `</w:body></w:document>`
}

// writeZipEntry 寫入一個 zip 項目。
func writeZipEntry(w *zip.Writer, name, content string) error {
	f, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("docx_write: create entry %s: %w", name, err)
	}
	_, err = f.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("docx_write: write entry %s: %w", name, err)
	}
	return nil
}

// --- 最小 docx 所需的固定 XML 模板 ---

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:docDefaults>
    <w:rPrDefault>
      <w:rPr>
        <w:sz w:val="24"/>
      </w:rPr>
    </w:rPrDefault>
  </w:docDefaults>
</w:styles>`
