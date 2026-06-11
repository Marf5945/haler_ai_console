// dianliao_bom.go — 電料 BOM skill 的後端核心。
//
// 職責三件事，全部純標準庫、零新增依賴：
//  1. 結構化讀取「電料編碼紀錄」.xlsx（依欄名定位，不寫死欄序）。
//  2. 料號查表 + 欄位對應（DB 欄名 → BOM 欄名）。
//  3. 套用固定版面範本，產生多工作表 BOM .xlsx（重用 GenerateMultiSheetXlsx）。
//
// 數量 / 註解 DB 沒有，必須由使用者輸入 → 透過 BOMItem.Qty / BOMItem.Note 帶入；
// 缺數量會列入 Result.Warnings，供「引導補足」流程追問。
package builtin

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------- 對外資料結構（可直接給 Wails binding / skill program 使用）----------

// BOMItem 是一列電料。料號由使用者輸入（廣達料號或供應商料號皆可），
// 品名/廠商料號/規格由 DB 自動帶出；數量、註解必須由使用者補足。
type BOMItem struct {
	PartNo string `json:"part_no"`        // 使用者輸入的料號（廣達料號或供應商料號）
	Qty    string `json:"qty"`            // 數量（使用者必填）
	Note   string `json:"note,omitempty"` // 註解（使用者選填）
}

// BOMSheet 對應 BOM 的一個工作表（一個電控箱 / 設備分區）。
type BOMSheet struct {
	Name     string    `json:"name"`                // 工作表名稱，例如「移載設備」「主電箱」「操作箱」
	LeadTime string    `json:"lead_time,omitempty"` // 備料天數，供「請購用加總」彙整（選填）
	Items    []BOMItem `json:"items"`
}

// BOMRequest 是產生一份 BOM 所需的完整輸入。
type BOMRequest struct {
	Machine string     `json:"machine"`         // 機台名稱（標題列，必填）
	Date    string     `json:"date"`            // 日期（標題列，必填）
	Title   string     `json:"title,omitempty"` // 抬頭，例如「SLM003 電控BOM」
	Sheets  []BOMSheet `json:"sheets"`
}

// BOMResult 回報結果。Warnings 收集「查無此料號」「未填數量」等，供前端引導補足。
type BOMResult struct {
	OutputPath string   `json:"output_path"`
	RowCount   int      `json:"row_count"`
	Warnings   []string `json:"warnings,omitempty"`
}

// ---------- DB 欄位對應（DB 欄名 → BOM 欄名）----------
// 只認欄名，不寫死欄序；DB 調整欄位順序也不會壞。

const (
	dbColPartNoQuanta   = "廣達料號"
	dbColPartNoSupplier = "供應商料號"
	dbColName           = "品名"
	dbColSpec           = "詳細規格"
	dbSheetMaterials    = "Materials"
)

type dbRecord struct {
	Name           string // 品名
	PartNoQuanta   string // 廣達料號 → BOM 料號
	PartNoSupplier string // 供應商料號 → BOM 廠商料號
	Spec           string // 詳細規格 → BOM 規格
}

// loadDianliaoDB 讀取電料編碼紀錄，建立「料號 → 紀錄」索引。
// 廣達料號與供應商料號都會建索引，使用者輸入任一種都能查到。
func loadDianliaoDB(dbPath string) (map[string]dbRecord, error) {
	grid, err := ReadXlsxSheetGrid(dbPath, dbSheetMaterials)
	if err != nil {
		return nil, err
	}
	if len(grid) < 2 {
		return nil, fmt.Errorf("dianliao_bom: DB 工作表 %q 沒有資料", dbSheetMaterials)
	}

	header := grid[0]
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}
	need := []string{dbColPartNoQuanta, dbColPartNoSupplier, dbColName, dbColSpec}
	for _, n := range need {
		if _, ok := col[n]; !ok {
			return nil, fmt.Errorf("dianliao_bom: DB 缺少欄位 %q", n)
		}
	}

	get := func(row []string, idx int) string {
		if idx >= 0 && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	index := map[string]dbRecord{}
	for _, row := range grid[1:] {
		rec := dbRecord{
			Name:           get(row, col[dbColName]),
			PartNoQuanta:   get(row, col[dbColPartNoQuanta]),
			PartNoSupplier: get(row, col[dbColPartNoSupplier]),
			Spec:           get(row, col[dbColSpec]),
		}
		if rec.PartNoQuanta != "" {
			index[normalizePartNo(rec.PartNoQuanta)] = rec
		}
		if rec.PartNoSupplier != "" {
			// 廣達料號優先；供應商料號不覆蓋已存在的廣達索引
			key := normalizePartNo(rec.PartNoSupplier)
			if _, exists := index[key]; !exists {
				index[key] = rec
			}
		}
	}
	return index, nil
}

func normalizePartNo(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// ---------- BOM 版面範本 ----------

// BOM 欄位順序：序號 | 品名 | 料號 | 廠商料號 | 數量 | 規格 | 註解
const (
	bomStyleTitle  = "title"
	bomStyleSub    = "subtitle"
	bomStyleHeader = "header"
)

func bomStyles() map[string]XlsxStyle {
	return map[string]XlsxStyle{
		bomStyleTitle:  {Bold: true},
		bomStyleSub:    {Bold: true, FontColor: "#444444"},
		bomStyleHeader: {Bold: true, FillColor: "#D9E1F2", Align: "center"},
	}
}

func bomColWidths() map[string]float64 {
	return map[string]float64{
		"A": 6, "B": 34, "C": 14, "D": 18, "E": 8, "F": 34, "G": 22,
	}
}

func cell(v interface{}) XlsxCell                 { return XlsxCell{Value: v} }
func styledCell(v interface{}, s string) XlsxCell { return XlsxCell{Value: v, Style: s} }

// BuildDianliaoBOM 是 skill 的單一進入點：查表 → 對應 → 套版面 → 產多工作表 xlsx。
func BuildDianliaoBOM(req BOMRequest, dbPath, destPath string) (BOMResult, error) {
	res := BOMResult{}
	if strings.TrimSpace(req.Machine) == "" {
		res.Warnings = append(res.Warnings, "缺少機台名稱")
	}
	if strings.TrimSpace(req.Date) == "" {
		res.Warnings = append(res.Warnings, "缺少日期")
	}
	if len(req.Sheets) == 0 {
		return res, fmt.Errorf("dianliao_bom: 沒有任何工作表資料")
	}

	db, err := loadDianliaoDB(dbPath)
	if err != nil {
		return res, err
	}

	title := req.Title
	if strings.TrimSpace(title) == "" {
		title = "電控 BOM"
	}

	var specs []XlsxSpec
	for _, sheet := range req.Sheets {
		spec := XlsxSpec{
			SheetName: sheet.Name,
			Styles:    bomStyles(),
			ColWidths: bomColWidths(),
		}

		// 標題列
		spec.Rows = append(spec.Rows, []XlsxCell{
			styledCell(req.Machine, bomStyleTitle), cell(""), cell(""),
			cell(""), cell(""), styledCell("日期:", bomStyleSub), styledCell(req.Date, bomStyleSub),
		})
		spec.Rows = append(spec.Rows, []XlsxCell{
			styledCell(title+"  /  "+sheet.Name, bomStyleSub),
		})
		// 表頭
		spec.Rows = append(spec.Rows, []XlsxCell{
			styledCell("序號", bomStyleHeader), styledCell("品名", bomStyleHeader),
			styledCell("料號", bomStyleHeader), styledCell("廠商料號", bomStyleHeader),
			styledCell("數量", bomStyleHeader), styledCell("規格", bomStyleHeader),
			styledCell("註解", bomStyleHeader),
		})

		// 明細列
		seq := 0
		for _, item := range sheet.Items {
			seq++
			res.RowCount++
			name, partQuanta, partSupplier, spec2 := "", "", "", ""
			if rec, ok := db[normalizePartNo(item.PartNo)]; ok {
				name = rec.Name
				partQuanta = rec.PartNoQuanta
				partSupplier = rec.PartNoSupplier
				spec2 = rec.Spec
			} else {
				// 查無此料號：保留使用者輸入的料號，標記警告
				partQuanta = item.PartNo
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("工作表「%s」第%d列：DB 查無料號 %q，品名/規格未自動帶入", sheet.Name, seq, item.PartNo))
			}
			if strings.TrimSpace(item.Qty) == "" {
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("工作表「%s」第%d列（%s）：未填數量", sheet.Name, seq, item.PartNo))
			}
			spec.Rows = append(spec.Rows, []XlsxCell{
				cell(seq), cell(name), cell(partQuanta), cell(partSupplier),
				cell(item.Qty), cell(spec2), cell(item.Note),
			})
		}
		specs = append(specs, spec)
	}

	// 請購用加總（lead time 彙整）
	specs = append(specs, buildPurchaseSummary(req, title))

	if err := GenerateMultiSheetXlsx(specs, destPath); err != nil {
		return res, err
	}
	res.OutputPath = destPath
	return res, nil
}

// buildPurchaseSummary 產生「請購用加總」工作表：項目（=各工作表）、數量合計、備料天數。
func buildPurchaseSummary(req BOMRequest, title string) XlsxSpec {
	spec := XlsxSpec{
		SheetName: "請購用加總",
		Styles:    bomStyles(),
		ColWidths: map[string]float64{"A": 6, "B": 28, "C": 12, "D": 14},
	}
	spec.Rows = append(spec.Rows, []XlsxCell{
		styledCell(title+"（請購用）", bomStyleTitle), cell(""), cell(""),
		styledCell(req.Date, bomStyleSub),
	})
	spec.Rows = append(spec.Rows, []XlsxCell{
		styledCell("序號", bomStyleHeader), styledCell("項目", bomStyleHeader),
		styledCell("數量合計", bomStyleHeader), styledCell("備料天數", bomStyleHeader),
	})
	for i, sheet := range req.Sheets {
		total := 0
		for _, item := range sheet.Items {
			if n, err := strconv.Atoi(strings.TrimSpace(item.Qty)); err == nil {
				total += n
			}
		}
		spec.Rows = append(spec.Rows, []XlsxCell{
			cell(i + 1), cell(sheet.Name), cell(total), cell(sheet.LeadTime),
		})
	}
	return spec
}
