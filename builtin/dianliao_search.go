// dianliao_search.go — 電料編碼紀錄的「可搜尋載入 + 本地粗篩」。
//
// 用途：使用者輸入模糊描述（例如「murr 10m」）時，先用本地 token 比對從 DB
// 粗篩出最像的數筆，再交給模型精選/排序（模型那一步在 app 層）。
// 全部純標準庫、純函式，方便單元測試。
package builtin

import (
	"fmt"
	"sort"
	"strings"
)

// DianliaoRecord 是可對外搜尋的一筆電料編碼紀錄。
type DianliaoRecord struct {
	Name           string // 品名
	PartNoQuanta   string // 廣達料號
	PartNoSupplier string // 供應商料號
	Spec           string // 詳細規格
}

// BestPartNo 回填入 BOM 用的料號：優先廣達料號，否則供應商料號。
func (r DianliaoRecord) BestPartNo() string {
	if strings.TrimSpace(r.PartNoQuanta) != "" {
		return strings.TrimSpace(r.PartNoQuanta)
	}
	return strings.TrimSpace(r.PartNoSupplier)
}

func (r DianliaoRecord) searchText() string {
	return strings.ToLower(strings.Join(
		[]string{r.Name, r.PartNoQuanta, r.PartNoSupplier, r.Spec}, " "))
}

// LoadDianliaoRecords 讀整份電料編碼紀錄，回所有可搜尋的列（依欄名定位）。
func LoadDianliaoRecords(dbPath string) ([]DianliaoRecord, error) {
	grid, err := ReadXlsxSheetGrid(dbPath, dbSheetMaterials)
	if err != nil {
		return nil, err
	}
	if len(grid) < 2 {
		return nil, fmt.Errorf("dianliao: DB 工作表 %q 沒有資料", dbSheetMaterials)
	}
	col := map[string]int{}
	for i, h := range grid[0] {
		col[strings.TrimSpace(h)] = i
	}
	get := func(row []string, name string) string {
		idx, ok := col[name]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	var out []DianliaoRecord
	for _, row := range grid[1:] {
		rec := DianliaoRecord{
			Name:           get(row, dbColName),
			PartNoQuanta:   get(row, dbColPartNoQuanta),
			PartNoSupplier: get(row, dbColPartNoSupplier),
			Spec:           get(row, dbColSpec),
		}
		if rec.Name == "" && rec.PartNoQuanta == "" && rec.PartNoSupplier == "" {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

// ScoredDianliaoRecord 是帶分數的搜尋結果。
type ScoredDianliaoRecord struct {
	Record DianliaoRecord
	Score  int
}

// SearchDianliaoLocal 用 token 比對做本地粗篩，回分數>0 的前 topN 筆（高到低）。
// 料號完全相符給大加分，確保使用者打的就是料號時排在最前。
func SearchDianliaoLocal(records []DianliaoRecord, query string, topN int) []ScoredDianliaoRecord {
	tokens := TokenizeDianliaoQuery(query)
	if len(tokens) == 0 {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var scored []ScoredDianliaoRecord
	for _, rec := range records {
		text := rec.searchText()
		score := 0
		for _, t := range tokens {
			if strings.Contains(text, t) {
				score++
			}
		}
		if q != "" && (strings.ToLower(rec.PartNoQuanta) == q || strings.ToLower(rec.PartNoSupplier) == q) {
			score += 100
		}
		if score > 0 {
			scored = append(scored, ScoredDianliaoRecord{Record: rec, Score: score})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if topN > 0 && len(scored) > topN {
		scored = scored[:topN]
	}
	return scored
}

// TokenizeDianliaoQuery 把查詢切成小寫 token：以空白與常見符號分隔、去重、長度≥2。
func TokenizeDianliaoQuery(query string) []string {
	repl := strings.NewReplacer(
		"，", " ", ",", " ", "、", " ", "/", " ", "\\", " ",
		"．", " ", ".", " ", "-", " ", "_", " ",
		"(", " ", ")", " ", "（", " ", "）", " ", "　", " ", "\t", " ", "\n", " ")
	fields := strings.Fields(strings.ToLower(repl.Replace(query)))
	var out []string
	seen := map[string]bool{}
	for _, f := range fields {
		if len([]rune(f)) < 2 || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}
