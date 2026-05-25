// xlsx_rw.go — Excel .xlsx 讀寫。
// .xlsx = zip 檔案，主要內容：
//   xl/sharedStrings.xml — 字串表（所有儲存格文字集中在這）
//   xl/worksheets/sheet1.xml — 工作表（儲存格用 <v> 索引指向字串表）
// Import：抽取所有字串。Export：產生最小單工作表 xlsx。
package builtin

import (
	"fmt"
	"html"
	"strings"
)

const xlsxSharedStringsNS = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"

// ExtractXlsxText 從 .xlsx 抽出所有儲存格文字。
// 策略：讀 sharedStrings.xml 取所有 <t> 文字，以 \n 分隔。
func ExtractXlsxText(path string) (string, error) {
	data, err := zipReadFile(path, "xl/sharedStrings.xml")
	if err != nil {
		return "", fmt.Errorf("xlsx_rw: read sharedStrings: %w", err)
	}
	if data == nil {
		// 有些 xlsx 沒有 sharedStrings（全部是數字），嘗試直接讀 sheet
		return extractXlsxFromSheet(path)
	}

	// 抽取 <t> 節點文字
	text := xmlExtractText(data, xlsxSharedStringsNS, "t", "si")
	if text == "" {
		// fallback: 不限 namespace 抽取
		text = xmlExtractText(data, "", "t", "si")
	}
	return text, nil
}

// extractXlsxFromSheet 從 sheet1 直接抽取 <v> 值（fallback）。
func extractXlsxFromSheet(path string) (string, error) {
	data, err := zipReadFile(path, "xl/worksheets/sheet1.xml")
	if err != nil || data == nil {
		return "", fmt.Errorf("xlsx_rw: no sharedStrings or sheet1 found")
	}
	return xmlExtractText(data, "", "v", "row"), nil
}

// GenerateXlsx 從純文字產生最小 .xlsx 檔案。
// content 格式：tab 分隔欄位，\n 分隔列。
func GenerateXlsx(content string, destPath string) error {
	lines := strings.Split(content, "\n")

	// 收集所有唯一字串（建立 sharedStrings）
	var sharedStrings []string
	ssIndex := map[string]int{}
	for _, line := range lines {
		for _, cell := range strings.Split(line, "\t") {
			if _, exists := ssIndex[cell]; !exists {
				ssIndex[cell] = len(sharedStrings)
				sharedStrings = append(sharedStrings, cell)
			}
		}
	}

	// 建立 sharedStrings.xml
	var ssXML strings.Builder
	ssXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	fmt.Fprintf(&ssXML, `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`,
		countCells(lines), len(sharedStrings))
	for _, s := range sharedStrings {
		ssXML.WriteString(`<si><t>` + html.EscapeString(s) + `</t></si>`)
	}
	ssXML.WriteString(`</sst>`)

	// 建立 sheet1.xml
	var sheetXML strings.Builder
	sheetXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheetXML.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	sheetXML.WriteString(`<sheetData>`)
	for rowIdx, line := range lines {
		fmt.Fprintf(&sheetXML, `<row r="%d">`, rowIdx+1)
		cells := strings.Split(line, "\t")
		for colIdx, cell := range cells {
			colLetter := string(rune('A' + colIdx)) // A-Z 足夠第一版
			if colIdx > 25 {
				colLetter = "Z"
			}
			ref := fmt.Sprintf("%s%d", colLetter, rowIdx+1)
			idx := ssIndex[cell]
			fmt.Fprintf(&sheetXML, `<c r="%s" t="s"><v>%d</v></c>`, ref, idx)
		}
		sheetXML.WriteString(`</row>`)
	}
	sheetXML.WriteString(`</sheetData></worksheet>`)

	entries := map[string]string{
		"[Content_Types].xml":            xlsxContentTypes,
		"_rels/.rels":                    xlsxRels,
		"xl/_rels/workbook.xml.rels":     xlsxWorkbookRels,
		"xl/workbook.xml":               xlsxWorkbook,
		"xl/styles.xml":                 xlsxStyles,
		"xl/sharedStrings.xml":          ssXML.String(),
		"xl/worksheets/sheet1.xml":      sheetXML.String(),
	}
	return writeMinimalZip(destPath, entries)
}

// countCells 計算總儲存格數。
func countCells(lines []string) int {
	n := 0
	for _, line := range lines {
		n += len(strings.Split(line, "\t"))
	}
	return n
}

// --- xlsx 固定 XML 模板 ---

const xlsxContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`

const xlsxRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const xlsxWorkbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`

const xlsxWorkbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets>
</workbook>`

const xlsxStyles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>
<fills count="1"><fill><patternFill patternType="none"/></fill></fills>
<borders count="1"><border/></borders>
<cellStyleXfs count="1"><xf/></cellStyleXfs>
<cellXfs count="1"><xf/></cellXfs>
</styleSheet>`

