// epub_read.go — EPUB 電子書文字抽取（僅匯入，不匯出）。
// EPUB 本質是 zip，內含 XHTML 章節檔案。
package builtin

import (
	"encoding/xml"
	"fmt"
	"path"
	"strings"
)

// ExtractEpubText 從 EPUB 檔案抽取所有章節的純文字。
func ExtractEpubText(filePath string) (string, error) {
	// 步驟一：找到 OPF 檔案路徑
	opfPath, err := epubFindOPFPath(filePath)
	if err != nil {
		return "", fmt.Errorf("epub: 找不到 OPF 檔案: %w", err)
	}

	// 步驟二：從 OPF 解析章節清單
	chapters, err := epubFindChapters(filePath, opfPath)
	if err != nil {
		return "", fmt.Errorf("epub: 解析章節失敗: %w", err)
	}

	// 步驟三：逐一讀取章節並抽取文字
	var parts []string
	for _, ch := range chapters {
		data, err := zipReadFile(filePath, ch)
		if err != nil {
			continue // 跳過讀取失敗的章節
		}
		if data == nil {
			continue // 檔案不存在，跳過
		}
		text := xmlExtractAllText(data)
		if text != "" {
			parts = append(parts, text)
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("epub: 無法抽取任何文字內容")
	}
	return strings.Join(parts, "\n\n"), nil
}

// epubFindOPFPath 透過 container.xml 找到 OPF 檔案的路徑。
// 若 container.xml 不存在，退而掃描 zip 內的 .opf 檔案。
func epubFindOPFPath(zipPath string) (string, error) {
	// 嘗試讀取標準路徑 container.xml
	data, err := zipReadFile(zipPath, "META-INF/container.xml")
	if err != nil {
		return "", err
	}

	if data != nil {
		// 解析 container.xml，找 rootfile 的 full-path 屬性
		opf := parseContainerXML(data)
		if opf != "" {
			return opf, nil
		}
	}

	// 退路：掃描 zip 找 .opf 檔案
	names, err := zipListFiles(zipPath)
	if err != nil {
		return "", err
	}
	for _, n := range names {
		if strings.HasSuffix(strings.ToLower(n), ".opf") {
			return n, nil // 找到第一個 .opf
		}
	}
	return "", fmt.Errorf("找不到 .opf 檔案")
}

// parseContainerXML 從 container.xml 中解析出 rootfile full-path。
func parseContainerXML(data []byte) string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "rootfile" {
				// 取得 full-path 屬性
				for _, attr := range se.Attr {
					if attr.Name.Local == "full-path" {
						return attr.Value
					}
				}
			}
		}
	}
	return ""
}

// epubFindChapters 從 OPF manifest 中找出所有 HTML/XHTML 章節檔案路徑。
func epubFindChapters(zipPath, opfPath string) ([]string, error) {
	data, err := zipReadFile(zipPath, opfPath)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("OPF 檔案不存在: %s", opfPath)
	}

	// OPF 所在的目錄，用於解析相對路徑
	opfDir := path.Dir(opfPath)

	// 解析 manifest 中的 item 元素
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var chapters []string
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "item" {
			continue
		}

		// 檢查 media-type 是否為 HTML 類型
		var href, mediaType string
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "href":
				href = attr.Value
			case "media-type":
				mediaType = attr.Value
			}
		}

		// 篩選 HTML/XHTML 內容
		if href != "" && (strings.Contains(mediaType, "html")) {
			// 解析相對路徑：以 OPF 目錄為基準
			fullPath := href
			if opfDir != "." && opfDir != "" {
				fullPath = opfDir + "/" + href
			}
			// 清理路徑（處理 ../ 等）
			fullPath = path.Clean(fullPath)
			chapters = append(chapters, fullPath)
		}
	}

	if len(chapters) == 0 {
		return nil, fmt.Errorf("OPF manifest 中找不到 HTML 章節")
	}
	return chapters, nil
}
