// docx_read.go — .docx 純文字抽取。
// .docx = zip 檔案，主要內容在 word/document.xml。
// 只抽 <w:t> 文字節點，以 \n 分隔段落（<w:p>）。
// 不保證：版面、表格、註解、圖片、頁首頁尾。
// 零第三方依賴：archive/zip + encoding/xml。
package builtin

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// ExtractDocxText 從 .docx 檔案抽出純文字。
// 回傳以 \n 分隔的段落文字。
func ExtractDocxText(path string) (string, error) {
	// 開啟 zip
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("docx_read: open zip: %w", err)
	}
	defer r.Close()

	// 找 word/document.xml
	var docFile *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return "", fmt.Errorf("docx_read: word/document.xml not found in %s", path)
	}

	// SEC-16: 檢查解壓大小，防止 zip bomb
	if docFile.UncompressedSize64 > maxZipEntrySize {
		return "", fmt.Errorf("docx_read: document.xml 太大: %d bytes", docFile.UncompressedSize64)
	}

	// 讀取 XML 內容
	rc, err := docFile.Open()
	if err != nil {
		return "", fmt.Errorf("docx_read: open document.xml: %w", err)
	}
	defer rc.Close()

	xmlData, err := io.ReadAll(io.LimitReader(rc, maxZipEntrySize))
	if err != nil {
		return "", fmt.Errorf("docx_read: read document.xml: %w", err)
	}

	// 用 XML decoder 逐 token 解析，抽取 <w:t> 內的文字
	return parseDocumentXML(xmlData)
}

// parseDocumentXML 解析 word/document.xml，抽出段落文字。
// 遇到 <w:p> 開始新段落，遇到 <w:t> 收集文字。
func parseDocumentXML(xmlData []byte) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(xmlData)))

	var paragraphs []string // 每個段落的完整文字
	var currentPara strings.Builder
	inText := false // 是否在 <w:t> 元素內

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("docx_read: xml parse: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			// <w:p> = 新段落開始
			if t.Name.Local == "p" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				// 儲存上一個段落（如果有內容）
				if currentPara.Len() > 0 {
					paragraphs = append(paragraphs, currentPara.String())
					currentPara.Reset()
				}
			}
			// <w:t> = 文字節點開始
			if t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inText = true
			}

		case xml.EndElement:
			// </w:t> = 文字節點結束
			if t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inText = false
			}

		case xml.CharData:
			// 在 <w:t> 內的文字，收集到當前段落
			if inText {
				currentPara.Write(t)
			}
		}
	}

	// 收尾：最後一個段落
	if currentPara.Len() > 0 {
		paragraphs = append(paragraphs, currentPara.String())
	}

	return strings.Join(paragraphs, "\n"), nil
}
