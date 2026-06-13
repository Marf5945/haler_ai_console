// xlsx_rw.go — Excel .xlsx 讀寫。
// .xlsx = zip 檔案，主要內容：
//
//	xl/sharedStrings.xml — 字串表（所有儲存格文字集中在這）
//	xl/worksheets/sheet1.xml — 工作表（儲存格用 <v> 索引指向字串表）
//
// Import：讀成二維表格或 tab 分隔文字。Export：產生最小單工作表 xlsx。
package builtin

import (
	"encoding/xml"
	"fmt"
	"html"
	"path"
	"sort"
	"strconv"
	"strings"
)

// ExtractXlsxText 從 .xlsx 抽出所有儲存格文字。
// 策略：讀第一個工作表，回傳 tab 分隔欄位、\n 分隔列的文字。
func ExtractXlsxText(path string) (string, error) {
	grid, err := ReadXlsxSheetGrid(path, "")
	if err != nil {
		return "", err
	}
	return xlsxGridToTabText(grid), nil
}

// ReadXlsxSheetGrid 將指定工作表讀成二維字串表格 grid[列][欄]。
// sheetName 為空時讀第一個工作表。
func ReadXlsxSheetGrid(filePath, sheetName string) ([][]string, error) {
	target, err := xlsxResolveSheetTarget(filePath, sheetName)
	if err != nil {
		return nil, err
	}
	shared, err := xlsxReadSharedStrings(filePath)
	if err != nil {
		return nil, err
	}
	data, err := zipReadFile(filePath, target)
	if err != nil {
		return nil, fmt.Errorf("xlsx_rw: read %s: %w", target, err)
	}
	if data == nil {
		return nil, fmt.Errorf("xlsx_rw: worksheet %s not found", target)
	}

	var sheet struct {
		Rows []struct {
			R     string `xml:"r,attr"`
			Cells []struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				V  string `xml:"v"`
				Is struct {
					T string `xml:"t"`
					R []struct {
						T string `xml:"t"`
					} `xml:"r"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(data, &sheet); err != nil {
		return nil, fmt.Errorf("xlsx_rw: parse worksheet XML: %w", err)
	}

	var grid [][]string
	for rowPos, row := range sheet.Rows {
		rowIdx := rowPos
		if n, err := strconv.Atoi(strings.TrimSpace(row.R)); err == nil && n > 0 {
			rowIdx = n - 1
		}
		cells := map[int]string{}
		maxCol := -1
		for cellPos, c := range row.Cells {
			colIdx := cellPos
			if c.R != "" {
				parsedRow, parsedCol, err := parseXlsxA1CellRef(c.R)
				if err != nil {
					continue
				}
				rowIdx = parsedRow
				colIdx = parsedCol
			}
			if colIdx > maxCol {
				maxCol = colIdx
			}
			cells[colIdx] = xlsxCellText(c.T, c.V, c.Is.T, xlsxRichTextRuns(c.Is.R), shared)
		}
		for len(grid) <= rowIdx {
			grid = append(grid, nil)
		}
		if maxCol < 0 {
			grid[rowIdx] = []string{}
			continue
		}
		line := make([]string, maxCol+1)
		for col, v := range cells {
			line[col] = v
		}
		grid[rowIdx] = line
	}
	return grid, nil
}

// ConvertXlsxToCSV 將第一個工作表轉成 CSV，方便系統讀取。
func ConvertXlsxToCSV(xlsxPath, csvPath string) error {
	return ConvertXlsxSheetToCSV(xlsxPath, csvPath, "", ',')
}

// ConvertXlsxSheetToCSV 將指定工作表轉成 CSV/TSV。
func ConvertXlsxSheetToCSV(xlsxPath, csvPath, sheetName string, delimiter rune) error {
	grid, err := ReadXlsxSheetGrid(xlsxPath, sheetName)
	if err != nil {
		return err
	}
	return WriteCSVRecords(grid, csvPath, delimiter)
}

// ConvertCSVToXlsx 將 CSV 轉回單工作表 XLSX。
func ConvertCSVToXlsx(csvPath, xlsxPath string) error {
	return ConvertDelimitedToXlsx(csvPath, xlsxPath, ',')
}

// ConvertDelimitedToXlsx 將 CSV/TSV 轉回單工作表 XLSX。
func ConvertDelimitedToXlsx(inputPath, xlsxPath string, delimiter rune) error {
	records, err := ReadCSVRecords(inputPath, delimiter)
	if err != nil {
		return err
	}
	return GenerateStyledXlsx(XlsxSpec{Rows: recordsToXlsxRows(records)}, xlsxPath)
}

func xlsxGridToTabText(grid [][]string) string {
	lines := make([]string, 0, len(grid))
	for _, row := range grid {
		lines = append(lines, strings.Join(row, "\t"))
	}
	return strings.Join(lines, "\n")
}

// GenerateXlsx 從純文字產生最小 .xlsx 檔案。
// content 格式：tab 分隔欄位，\n 分隔列。
func GenerateXlsx(content string, destPath string) error {
	return GenerateStyledXlsx(XlsxSpec{Rows: tsvToXlsxRows(content)}, destPath)
}

// XlsxSpec 是結構化 .xlsx 產生格式。Rows 適合表格資料，Cells 適合指定 A1 儲存格。
type XlsxSpec struct {
	SheetName string               `json:"sheet_name,omitempty"`
	Rows      [][]XlsxCell         `json:"rows,omitempty"`
	Cells     map[string]XlsxCell  `json:"cells,omitempty"`
	Styles    map[string]XlsxStyle `json:"styles,omitempty"`
	ColWidths map[string]float64   `json:"col_widths,omitempty"`
}

type XlsxCell struct {
	Value interface{} `json:"value"`
	Style string      `json:"style,omitempty"`
}

type XlsxStyle struct {
	Bold      bool   `json:"bold,omitempty"`
	FontColor string `json:"font_color,omitempty"`
	FillColor string `json:"fill_color,omitempty"`
	Align     string `json:"align,omitempty"`
	NumFmt    string `json:"num_fmt,omitempty"`
}

// GenerateStyledXlsx 從結構化資料產生單工作表 .xlsx，支援基本樣式與欄寬。
func GenerateStyledXlsx(spec XlsxSpec, destPath string) error {
	lines, err := xlsxSpecRows(spec)
	if err != nil {
		return err
	}

	// 收集所有唯一字串（建立 sharedStrings）
	var sharedStrings []string
	ssIndex := map[string]int{}
	cellCount := 0
	for _, row := range lines {
		for _, cell := range row {
			value := xlsxCellValueString(cell.Value)
			if _, exists := ssIndex[value]; !exists {
				ssIndex[value] = len(sharedStrings)
				sharedStrings = append(sharedStrings, value)
			}
			cellCount++
		}
	}

	// 建立 sharedStrings.xml
	var ssXML strings.Builder
	ssXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	fmt.Fprintf(&ssXML, `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`,
		cellCount, len(sharedStrings))
	for _, s := range sharedStrings {
		ssXML.WriteString(`<si><t>` + html.EscapeString(s) + `</t></si>`)
	}
	ssXML.WriteString(`</sst>`)

	styleXML, styleIDs := buildXlsxStyles(spec.Styles)

	// 建立 sheet1.xml
	var sheetXML strings.Builder
	sheetXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheetXML.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	writeXlsxColumns(&sheetXML, spec.ColWidths)
	sheetXML.WriteString(`<sheetData>`)
	for rowIdx, row := range lines {
		fmt.Fprintf(&sheetXML, `<row r="%d">`, rowIdx+1)
		for colIdx, cell := range row {
			colLetter := xlsxColumnName(colIdx)
			ref := fmt.Sprintf("%s%d", colLetter, rowIdx+1)
			idx := ssIndex[xlsxCellValueString(cell.Value)]
			styleAttr := ""
			if styleID, ok := styleIDs[cell.Style]; ok && styleID > 0 {
				styleAttr = fmt.Sprintf(` s="%d"`, styleID)
			}
			fmt.Fprintf(&sheetXML, `<c r="%s" t="s"%s><v>%d</v></c>`, ref, styleAttr, idx)
		}
		sheetXML.WriteString(`</row>`)
	}
	sheetXML.WriteString(`</sheetData></worksheet>`)

	entries := map[string]string{
		"[Content_Types].xml":        xlsxContentTypes,
		"_rels/.rels":                xlsxRels,
		"xl/_rels/workbook.xml.rels": xlsxWorkbookRels,
		"xl/workbook.xml":            xlsxWorkbookXML(spec.SheetName),
		"xl/styles.xml":              styleXML,
		"xl/sharedStrings.xml":       ssXML.String(),
		"xl/worksheets/sheet1.xml":   sheetXML.String(),
	}
	return writeMinimalZip(destPath, entries)
}

// GenerateMultiSheetXlsx 從多個 XlsxSpec 產生「多工作表」.xlsx。
// 每個 spec 對應一個工作表；所有工作表共用一份字串表與樣式表。
// 純標準庫，無新增依賴，重用 GenerateStyledXlsx 既有的 helper。
func GenerateMultiSheetXlsx(specs []XlsxSpec, destPath string) error {
	if len(specs) == 0 {
		return GenerateStyledXlsx(XlsxSpec{}, destPath)
	}

	// 合併所有工作表的樣式，共用一份 styles.xml
	mergedStyles := map[string]XlsxStyle{}
	for _, spec := range specs {
		for name, st := range spec.Styles {
			mergedStyles[name] = st
		}
	}
	styleXML, styleIDs := buildXlsxStyles(mergedStyles)

	// 全活頁簿共用字串表
	var sharedStrings []string
	ssIndex := map[string]int{}
	cellCount := 0

	type builtSheet struct {
		name string
		xml  string
	}
	sheets := make([]builtSheet, 0, len(specs))
	usedNames := map[string]bool{}

	for si, spec := range specs {
		lines, err := xlsxSpecRows(spec)
		if err != nil {
			return err
		}

		var sheetXML strings.Builder
		sheetXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
		sheetXML.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
		writeXlsxColumns(&sheetXML, spec.ColWidths)
		sheetXML.WriteString(`<sheetData>`)
		for rowIdx, row := range lines {
			fmt.Fprintf(&sheetXML, `<row r="%d">`, rowIdx+1)
			for colIdx, cell := range row {
				value := xlsxCellValueString(cell.Value)
				if _, exists := ssIndex[value]; !exists {
					ssIndex[value] = len(sharedStrings)
					sharedStrings = append(sharedStrings, value)
				}
				cellCount++
				ref := fmt.Sprintf("%s%d", xlsxColumnName(colIdx), rowIdx+1)
				styleAttr := ""
				if styleID, ok := styleIDs[cell.Style]; ok && styleID > 0 {
					styleAttr = fmt.Sprintf(` s="%d"`, styleID)
				}
				fmt.Fprintf(&sheetXML, `<c r="%s" t="s"%s><v>%d</v></c>`, ref, styleAttr, ssIndex[value])
			}
			sheetXML.WriteString(`</row>`)
		}
		sheetXML.WriteString(`</sheetData></worksheet>`)

		// 工作表名稱：清理 + 去重（Excel 不允許重名）
		name := sanitizeXlsxSheetName(spec.SheetName)
		if spec.SheetName == "" {
			name = fmt.Sprintf("Sheet%d", si+1)
		}
		base := name
		for n := 2; usedNames[name]; n++ {
			name = sanitizeXlsxSheetName(fmt.Sprintf("%s_%d", base, n))
		}
		usedNames[name] = true

		sheets = append(sheets, builtSheet{name: name, xml: sheetXML.String()})
	}

	// sharedStrings.xml
	var ssXML strings.Builder
	ssXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	fmt.Fprintf(&ssXML, `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`,
		cellCount, len(sharedStrings))
	for _, s := range sharedStrings {
		ssXML.WriteString(`<si><t xml:space="preserve">` + html.EscapeString(s) + `</t></si>`)
	}
	ssXML.WriteString(`</sst>`)

	// [Content_Types].xml：每個工作表一筆 Override
	var ctXML strings.Builder
	ctXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	ctXML.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	ctXML.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	ctXML.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	ctXML.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	for i := range sheets {
		fmt.Fprintf(&ctXML, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i+1)
	}
	ctXML.WriteString(`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	ctXML.WriteString(`<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>`)
	ctXML.WriteString(`</Types>`)

	// xl/workbook.xml：列出所有工作表
	var wbXML strings.Builder
	wbXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	wbXML.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for i, s := range sheets {
		fmt.Fprintf(&wbXML, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, html.EscapeString(s.name), i+1, i+1)
	}
	wbXML.WriteString(`</sheets></workbook>`)

	// xl/_rels/workbook.xml.rels：工作表 rId1..N，接著 styles、sharedStrings
	var relsXML strings.Builder
	relsXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	relsXML.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := range sheets {
		fmt.Fprintf(&relsXML, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i+1, i+1)
	}
	fmt.Fprintf(&relsXML, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, len(sheets)+1)
	fmt.Fprintf(&relsXML, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>`, len(sheets)+2)
	relsXML.WriteString(`</Relationships>`)

	entries := map[string]string{
		"[Content_Types].xml":        ctXML.String(),
		"_rels/.rels":                xlsxRels,
		"xl/_rels/workbook.xml.rels": relsXML.String(),
		"xl/workbook.xml":            wbXML.String(),
		"xl/styles.xml":              styleXML,
		"xl/sharedStrings.xml":       ssXML.String(),
	}
	for i, s := range sheets {
		entries[fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)] = s.xml
	}
	return writeMinimalZip(destPath, entries)
}

func tsvToXlsxRows(content string) [][]XlsxCell {
	lines := strings.Split(content, "\n")
	rows := make([][]XlsxCell, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		row := make([]XlsxCell, 0, len(parts))
		for _, cell := range parts {
			row = append(row, XlsxCell{Value: cell})
		}
		rows = append(rows, row)
	}
	return rows
}

func recordsToXlsxRows(records [][]string) [][]XlsxCell {
	rows := make([][]XlsxCell, 0, len(records))
	for _, record := range records {
		row := make([]XlsxCell, 0, len(record))
		for _, cell := range record {
			row = append(row, XlsxCell{Value: cell})
		}
		rows = append(rows, row)
	}
	return rows
}

func xlsxResolveSheetTarget(filePath, sheetName string) (string, error) {
	wbData, err := zipReadFile(filePath, "xl/workbook.xml")
	if err != nil || wbData == nil {
		return "xl/worksheets/sheet1.xml", nil
	}
	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := xml.Unmarshal(wbData, &wb); err != nil || len(wb.Sheets) == 0 {
		return "xl/worksheets/sheet1.xml", nil
	}

	relData, _ := zipReadFile(filePath, "xl/_rels/workbook.xml.rels")
	relMap := map[string]string{}
	if relData != nil {
		var rels struct {
			Rel []struct {
				ID     string `xml:"Id,attr"`
				Target string `xml:"Target,attr"`
			} `xml:"Relationship"`
		}
		if xml.Unmarshal(relData, &rels) == nil {
			for _, r := range rels.Rel {
				relMap[r.ID] = r.Target
			}
		}
	}

	pick := wb.Sheets[0]
	if strings.TrimSpace(sheetName) != "" {
		for _, s := range wb.Sheets {
			if strings.TrimSpace(s.Name) == strings.TrimSpace(sheetName) {
				pick = s
				break
			}
		}
	}
	target := relMap[pick.RID]
	if target == "" {
		return "xl/worksheets/sheet1.xml", nil
	}
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/"), nil
	}
	if strings.HasPrefix(target, "xl/") {
		return path.Clean(target), nil
	}
	return path.Clean("xl/" + target), nil
}

func xlsxReadSharedStrings(filePath string) ([]string, error) {
	data, err := zipReadFile(filePath, "xl/sharedStrings.xml")
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var sst struct {
		SI []struct {
			T string `xml:"t"`
			R []struct {
				T string `xml:"t"`
			} `xml:"r"`
		} `xml:"si"`
	}
	if err := xml.Unmarshal(data, &sst); err != nil {
		return nil, fmt.Errorf("xlsx_rw: parse sharedStrings: %w", err)
	}
	out := make([]string, 0, len(sst.SI))
	for _, si := range sst.SI {
		if si.T != "" || len(si.R) == 0 {
			out = append(out, si.T)
			continue
		}
		out = append(out, strings.Join(xlsxRichTextRuns(si.R), ""))
	}
	return out, nil
}

func xlsxRichTextRuns(runs []struct {
	T string `xml:"t"`
}) []string {
	out := make([]string, 0, len(runs))
	for _, r := range runs {
		out = append(out, r.T)
	}
	return out
}

func xlsxCellText(cellType, rawValue, inlineText string, inlineRuns []string, shared []string) string {
	switch cellType {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(rawValue))
		if err == nil && idx >= 0 && idx < len(shared) {
			return shared[idx]
		}
		return ""
	case "inlineStr":
		if inlineText != "" || len(inlineRuns) == 0 {
			return inlineText
		}
		return strings.Join(inlineRuns, "")
	default:
		return rawValue
	}
}

func xlsxSpecRows(spec XlsxSpec) ([][]XlsxCell, error) {
	rows := make([][]XlsxCell, len(spec.Rows))
	for i, row := range spec.Rows {
		rows[i] = append([]XlsxCell(nil), row...)
	}
	refs := make([]string, 0, len(spec.Cells))
	for ref := range spec.Cells {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		rowIdx, colIdx, err := parseXlsxA1CellRef(ref)
		if err != nil {
			return nil, err
		}
		for len(rows) <= rowIdx {
			rows = append(rows, nil)
		}
		for len(rows[rowIdx]) <= colIdx {
			rows[rowIdx] = append(rows[rowIdx], XlsxCell{})
		}
		rows[rowIdx][colIdx] = spec.Cells[ref]
	}
	if len(rows) == 0 {
		return [][]XlsxCell{{}}, nil
	}
	return rows, nil
}

func parseXlsxA1CellRef(ref string) (int, int, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, 0, fmt.Errorf("xlsx_rw: empty cell ref")
	}
	i := 0
	col := 0
	for i < len(ref) {
		ch := ref[i]
		if ch >= 'a' && ch <= 'z' {
			ch -= 'a' - 'A'
		}
		if ch < 'A' || ch > 'Z' {
			break
		}
		col = col*26 + int(ch-'A'+1)
		i++
	}
	if col == 0 || i == len(ref) {
		return 0, 0, fmt.Errorf("xlsx_rw: invalid cell ref %q", ref)
	}
	row := 0
	for ; i < len(ref); i++ {
		ch := ref[i]
		if ch < '0' || ch > '9' {
			return 0, 0, fmt.Errorf("xlsx_rw: invalid cell ref %q", ref)
		}
		row = row*10 + int(ch-'0')
	}
	if row <= 0 {
		return 0, 0, fmt.Errorf("xlsx_rw: invalid cell ref %q", ref)
	}
	return row - 1, col - 1, nil
}

func xlsxCellValueString(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func xlsxColumnName(idx int) string {
	if idx < 0 {
		return "A"
	}
	name := ""
	for idx >= 0 {
		name = string(rune('A'+idx%26)) + name
		idx = idx/26 - 1
	}
	return name
}

func writeXlsxColumns(b *strings.Builder, widths map[string]float64) {
	if len(widths) == 0 {
		return
	}
	keys := make([]string, 0, len(widths))
	for key := range widths {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	b.WriteString(`<cols>`)
	for _, key := range keys {
		width := widths[key]
		if width <= 0 {
			continue
		}
		col, err := xlsxColumnIndex(key)
		if err != nil {
			continue
		}
		fmt.Fprintf(b, `<col min="%d" max="%d" width="%.2f" customWidth="1"/>`, col+1, col+1, width)
	}
	b.WriteString(`</cols>`)
}

func xlsxColumnIndex(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, fmt.Errorf("xlsx_rw: empty column")
	}
	col := 0
	for _, r := range strings.ToUpper(ref) {
		if r < 'A' || r > 'Z' {
			return 0, fmt.Errorf("xlsx_rw: invalid column %q", ref)
		}
		col = col*26 + int(r-'A'+1)
	}
	return col - 1, nil
}

func buildXlsxStyles(styles map[string]XlsxStyle) (string, map[string]int) {
	names := make([]string, 0, len(styles))
	for name := range styles {
		if strings.TrimSpace(name) != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	styleIDs := map[string]int{"": 0}
	var fonts []string
	var fills []string
	var xfs []string
	fonts = append(fonts, `<font><sz val="11"/><name val="Calibri"/></font>`)
	fills = append(fills,
		`<fill><patternFill patternType="none"/></fill>`,
		`<fill><patternFill patternType="gray125"/></fill>`,
	)
	xfs = append(xfs, `<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>`)

	for _, name := range names {
		style := styles[name]
		fontID := len(fonts)
		fillID := 0
		fonts = append(fonts, xlsxFontXML(style))
		if normalizeXlsxColor(style.FillColor) != "" {
			fillID = len(fills)
			fills = append(fills, xlsxFillXML(style.FillColor))
		}
		attrs := fmt.Sprintf(`numFmtId="0" fontId="%d" fillId="%d" borderId="0" xfId="0"`, fontID, fillID)
		apply := ` applyFont="1"`
		if fillID > 0 {
			apply += ` applyFill="1"`
		}
		alignment := xlsxAlignmentXML(style.Align)
		if alignment != "" {
			apply += ` applyAlignment="1"`
			xfs = append(xfs, `<xf `+attrs+apply+`>`+alignment+`</xf>`)
		} else {
			xfs = append(xfs, `<xf `+attrs+apply+`/>`)
		}
		styleIDs[name] = len(xfs) - 1
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	fmt.Fprintf(&b, `<fonts count="%d">%s</fonts>`, len(fonts), strings.Join(fonts, ""))
	fmt.Fprintf(&b, `<fills count="%d">%s</fills>`, len(fills), strings.Join(fills, ""))
	b.WriteString(`<borders count="1"><border/></borders>`)
	b.WriteString(`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>`)
	fmt.Fprintf(&b, `<cellXfs count="%d">%s</cellXfs>`, len(xfs), strings.Join(xfs, ""))
	b.WriteString(`</styleSheet>`)
	return b.String(), styleIDs
}

func xlsxFontXML(style XlsxStyle) string {
	var b strings.Builder
	b.WriteString(`<font><sz val="11"/><name val="Calibri"/>`)
	if style.Bold {
		b.WriteString(`<b/>`)
	}
	if color := normalizeXlsxColor(style.FontColor); color != "" {
		fmt.Fprintf(&b, `<color rgb="%s"/>`, color)
	}
	b.WriteString(`</font>`)
	return b.String()
}

func xlsxFillXML(color string) string {
	color = normalizeXlsxColor(color)
	return `<fill><patternFill patternType="solid"><fgColor rgb="` + color + `"/><bgColor indexed="64"/></patternFill></fill>`
}

func xlsxAlignmentXML(align string) string {
	align = strings.ToLower(strings.TrimSpace(align))
	switch align {
	case "left", "center", "right":
		return `<alignment horizontal="` + align + `"/>`
	case "置中", "居中":
		return `<alignment horizontal="center"/>`
	case "靠右":
		return `<alignment horizontal="right"/>`
	}
	return ""
}

func normalizeXlsxColor(color string) string {
	color = strings.TrimSpace(strings.TrimPrefix(color, "#"))
	if len(color) == 6 {
		return "FF" + strings.ToUpper(color)
	}
	if len(color) == 8 {
		return strings.ToUpper(color)
	}
	return ""
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

func xlsxWorkbookXML(sheetName string) string {
	sheetName = sanitizeXlsxSheetName(sheetName)
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="` + html.EscapeString(sheetName) + `" sheetId="1" r:id="rId1"/></sheets>
</workbook>`
}

func sanitizeXlsxSheetName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Sheet1"
	}
	replacer := strings.NewReplacer("[", "_", "]", "_", ":", "_", "*", "_", "?", "_", "/", "_", "\\", "_")
	name = replacer.Replace(name)
	runes := []rune(name)
	if len(runes) > 31 {
		name = string(runes[:31])
	}
	if strings.TrimSpace(name) == "" {
		return "Sheet1"
	}
	return name
}

const xlsxStyles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>
<fills count="1"><fill><patternFill patternType="none"/></fill></fills>
<borders count="1"><border/></borders>
<cellStyleXfs count="1"><xf/></cellStyleXfs>
<cellXfs count="1"><xf/></cellXfs>
</styleSheet>`
