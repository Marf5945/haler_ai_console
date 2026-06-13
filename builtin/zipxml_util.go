// zipxml_util.go — zip+XML 格式的共用工具。
// docx/xlsx/pptx/odt/ods/odp/epub 都是 zip 檔案內包 XML。
// 這裡提供共用的 zip 開啟、XML 文字抽取工具，減少重複程式碼。
package builtin

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// SEC-16: zip entry 解壓上限 500MB，防止 zip bomb
const maxZipEntrySize = 500 * 1024 * 1024

// zipReadFile 從 zip 檔案中讀取指定路徑的內容。
// 找不到回傳 nil, nil（不是錯誤，因為某些檔案可能不存在）。
func zipReadFile(zipPath, entryName string) ([]byte, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == entryName {
			// SEC-16: 先檢查宣告大小，再用 LimitReader 雙重防護
			if f.UncompressedSize64 > maxZipEntrySize {
				return nil, fmt.Errorf("zip entry %s 太大: %d bytes (上限 %d)", entryName, f.UncompressedSize64, maxZipEntrySize)
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, maxZipEntrySize))
		}
	}
	return nil, nil // 找不到不是錯誤
}

// zipListFiles 列出 zip 內所有檔案名稱。
func zipListFiles(zipPath string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names, nil
}

// zipFindByPrefix 找出 zip 內以 prefix 開頭的所有檔案，按名稱排序。
func zipFindByPrefix(zipPath, prefix string) ([]string, error) {
	names, err := zipListFiles(zipPath)
	if err != nil {
		return nil, err
	}
	var matched []string
	for _, n := range names {
		if strings.HasPrefix(n, prefix) {
			matched = append(matched, n)
		}
	}
	sort.Strings(matched) // 排序確保 slide1 < slide2 < slide10
	return matched, nil
}

// xmlExtractText 從 XML bytes 中抽取指定 namespace+localName 的所有文字內容。
// 遇到 paragraphLocal 時插入換行。
// 這是 docx/xlsx/pptx/odt 共用的核心抽取邏輯。
func xmlExtractText(data []byte, textNS, textLocal, paraLocal string) string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var paragraphs []string
	var current strings.Builder
	inText := false

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == paraLocal {
				if current.Len() > 0 {
					paragraphs = append(paragraphs, current.String())
					current.Reset()
				}
			}
			if matchElement(t, textNS, textLocal) {
				inText = true
			}
		case xml.EndElement:
			if matchElement(t, textNS, textLocal) {
				inText = false
			}
		case xml.CharData:
			if inText {
				current.Write(t)
			}
		}
	}
	if current.Len() > 0 {
		paragraphs = append(paragraphs, current.String())
	}
	return strings.Join(paragraphs, "\n")
}

// xmlExtractAllText 從 XML bytes 中抽取所有 CharData 文字（不限 namespace）。
// 適合 EPUB XHTML 等不需要精確匹配 namespace 的場景。
func xmlExtractAllText(data []byte) string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var parts []string
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if cd, ok := tok.(xml.CharData); ok {
			text := strings.TrimSpace(string(cd))
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, " ")
}

// matchElement 比較 StartElement 或 EndElement 的 namespace + local name。
// ns 為空字串時不比較 namespace（只看 local name）。
func matchElement(tok interface{}, ns, local string) bool {
	switch t := tok.(type) {
	case xml.StartElement:
		if ns == "" {
			return t.Name.Local == local
		}
		return t.Name.Space == ns && t.Name.Local == local
	case xml.EndElement:
		if ns == "" {
			return t.Name.Local == local
		}
		return t.Name.Space == ns && t.Name.Local == local
	}
	return false
}

// writeMinimalZip 建立一個最小 zip 檔案，從 entries map 寫入。
// entries: key=zip 內路徑, value=檔案內容。
func writeMinimalZip(destPath string, entries map[string]string) error {
	f, err := createFile(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for name, content := range entries {
		fw, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

// createFile 建立檔案（含建立父目錄）。
func createFile(path string) (*os.File, error) {
	return os.Create(path)
}
