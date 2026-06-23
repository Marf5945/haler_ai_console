# 產出電料Bom（skill_id: dianliao.bom）

當使用者提到某個電料 / 料號，或要求「產生電料 BOM」時觸發。
本 skill 會先確認是否輸出 BOM，引導補足必要欄位，最後輸出多工作表 `.xlsx`（電料 BOM）。

- **資料庫**：`電料編碼紀錄_*.xlsx`（工作表 `Materials`），後端依欄名定位。
- **輸出**：每個電控箱 / 分區一張工作表 + 「請購用加總」（含備料天數 lead time）。
- **後端進入點**：`builtin.BuildDianliaoBOM(req, dbPath, destPath)`。
- **DAG**：`programs/build_bom/dag_產出電料Bom.json`（title = 產出電料Bom）。

## 必補欄位（引導補足）
機台名稱、日期、每筆電料的「數量」。缺少時後端回傳 warning，請逐項追問後再輸出。

## 欄位對應
品名←DB品名、料號←DB廣達料號、廠商料號←DB供應商料號、規格←DB詳細規格；數量/註解由使用者輸入。

詳見 `cli_md/usage/usage.md` 與 `programs/build_bom/build_bom.md`。
