# 對話流程（給模型遵循）

對應 DAG `產出電料Bom` 的節點：

1. **confirm_intent**：偵測到電料 / 料號 / 「電料 BOM」→ 先問「要產生電料 BOM 嗎？」
2. **collect_fields**：逐項引導補足，缺哪項問哪項：
   - 機台名稱？
   - 日期？（預設今天）
   - 分成哪幾個電控箱 / 分區？（每區一張工作表）
   - 每區要放哪些料號，各自的「數量」（必填）與「註解」？
3. **lookup_db / fill_mapping**：組成 BOMRequest（見 programs/build_bom/build_bom.md 的 JSON 契約），呼叫後端查表填值。
4. **generate_xlsx**：輸出多工作表 BOM `.xlsx`。
5. **verify_output**：對 Warnings（查無料號 / 未填數量）回報使用者，補正後再輸出。

## 欄位對應（DB → BOM）
| BOM 欄位 | 來源 | 必填 |
|---|---|---|
| 品名 | DB `品名` | 自動 |
| 料號 | DB `廣達料號` | 自動 |
| 廠商料號 | DB `供應商料號` | 自動 |
| 規格 | DB `詳細規格` | 自動 |
| 數量 | 使用者輸入 | **是** |
| 註解 | 使用者輸入 | 否 |
