# build_bom — 查表 → 填值 → 產 xlsx

此 program 不外呼任何 CLI，直接呼叫 App 內建 Go 函式。

進入點：`builtin.BuildDianliaoBOM(req builtin.BOMRequest, dbPath, destPath string) (builtin.BOMResult, error)`

對應 DAG 三步：
1. **查表 lookup_db** → `loadDianliaoDB(dbPath)`：讀電料編碼紀錄 `Materials`，依欄名建索引（廣達料號 + 供應商料號）。
2. **填值 fill_mapping** → 依料號帶出 品名/廠商料號/規格；數量、註解用使用者輸入。
3. **產 xlsx generate_xlsx** → `GenerateMultiSheetXlsx`：每個電控箱一張工作表 + 「請購用加總」。

DAG 定義：見同目錄 `dag_產出電料Bom.json`（title = 產出電料Bom）。

## 建議的 Wails binding（加在 app.go）
```go
func (a *App) GenerateDianliaoBOM(reqJSON, dbPath, destPath string) (builtin.BOMResult, error) {
    var req builtin.BOMRequest
    if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
        return builtin.BOMResult{}, err
    }
    return builtin.BuildDianliaoBOM(req, dbPath, destPath)
}
```

## BOMRequest JSON 契約
```json
{
  "machine": "SLM003 移載設備",
  "date": "2026-03-27",
  "title": "SLM003 電控BOM",
  "sheets": [
    { "name": "副電箱", "lead_time": "30",
      "items": [
        { "part_no": "80151002", "qty": "1", "note": "副電箱總電壓" },
        { "part_no": "2618000000", "qty": "10", "note": "油壓馬達" }
      ] }
  ]
}
```
`part_no` 可填廣達料號或供應商料號；`qty` 必填；`note` 選填。
回傳 `BOMResult.Warnings` 非空時（查無料號 / 未填數量），先回報使用者再決定是否重出。
