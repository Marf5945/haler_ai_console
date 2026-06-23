// pptx_rw.go — PowerPoint .pptx 讀寫。
// 匯入：抽取所有投影片文字。匯出：產生單頁 .pptx。
package builtin

import (
	"fmt"
	"html"
	"strings"
)

// drawingMLNS 是 DrawingML 的 namespace（a:t 節點）。
const drawingMLNS = "http://schemas.openxmlformats.org/drawingml/2006/main"

// ExtractPptxText 從 .pptx 抽取所有投影片的文字內容。
// 每張投影片的段落以換行分隔，投影片之間也用換行分隔。
func ExtractPptxText(path string) (string, error) {
	// 找出所有 slide 檔案
	entries, err := zipFindByPrefix(path, "ppt/slides/slide")
	if err != nil {
		return "", fmt.Errorf("pptx 找投影片失敗: %w", err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("pptx 無投影片")
	}

	var allText []string
	for _, entry := range entries {
		// 讀取單張投影片 XML
		data, err := zipReadFile(path, entry)
		if err != nil {
			return "", fmt.Errorf("pptx 讀 %s 失敗: %w", entry, err)
		}
		if data == nil {
			continue // 檔案不存在，跳過
		}

		// 用 namespace 抽取 <a:t> 文字
		text := xmlExtractText(data, drawingMLNS, "t", "p")
		if text == "" {
			// 備援：不指定 namespace 再試一次
			text = xmlExtractText(data, "", "t", "p")
		}
		if text != "" {
			allText = append(allText, text)
		}
	}
	return strings.Join(allText, "\n"), nil
}

// GeneratePptx 產生一個最小的單頁 .pptx，每行文字各自成一個段落。
func GeneratePptx(destPath, content string) error {
	lines := strings.Split(content, "\n")

	// 組合投影片 XML 中的段落
	var parasBuf strings.Builder
	for _, line := range lines {
		escaped := html.EscapeString(line)
		parasBuf.WriteString(`<a:p><a:r><a:rPr lang="zh-TW" dirty="0"/><a:t>`)
		parasBuf.WriteString(escaped)
		parasBuf.WriteString(`</a:t></a:r></a:p>`)
	}
	parasXML := parasBuf.String()

	// 各 zip 檔案內容
	entries := map[string]string{
		// [Content_Types].xml — 宣告各類型
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
  <Override PartName="/ppt/presProps.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presProps+xml"/>
</Types>`,

		// _rels/.rels — 根關聯，指向 presentation.xml
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`,

		// ppt/_rels/presentation.xml.rels — presentation 的關聯
		"ppt/_rels/presentation.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/presProps" Target="presProps.xml"/>
</Relationships>`,

		// ppt/presentation.xml — 最小 presentation
		"ppt/presentation.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>
    <p:sldId id="256" r:id="rId2"/>
  </p:sldIdLst>
</p:presentation>`,

		// ppt/presProps.xml — 空的簡報屬性
		"ppt/presProps.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentationPr xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"/>`,

		// ppt/slides/slide1.xml — 實際內容投影片
		"ppt/slides/slide1.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr/>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="2" name="Content"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="457200" y="457200"/><a:ext cx="8229600" cy="5486400"/></a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0"/>
          <a:lstStyle/>
          ` + parasXML + `
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`,
	}

	// 寫入 zip
	if err := writeMinimalZip(destPath, entries); err != nil {
		return fmt.Errorf("pptx 寫檔失敗: %w", err)
	}
	return nil
}
