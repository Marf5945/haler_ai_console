// Package hookgene 是 AI Console 的「行動基因 / Recorder」獨立評分系統（§3.1.5.18–19）。
//
// 定位
//   - 獨立系統：把每個 skill invocation 的執行「行為」轉成固定 16 格的行動基因
//     （hook gene），用來偵測 skill 肥大，並驅動使用者主動觸發的突變學習。
//   - 純 stdlib、零外部相依、leaf package：core layer 不得 import 本套件，
//     本套件也不 import 任何 core / 對話 / LLM 套件，確保架構守門（leaf）。
//
// 分層原則（重要，維護時勿混用）
//   - hook gene = ㄅ/ㄖ/ㄔ/ㄇ，僅內部評分用，永不進入 LLM 回覆文字流。
//   - 因此與 action-chain 的 ㄌ 分隔符、§31 控制印章字元集「分層、互不衝突」。
//
// 兩層架構
//   - Recorder：平常 always-on，後台低成本、無語意、無 LLM；只記錄不突變。
//   - Learning Mode：只在使用者按鈕觸發，才分析 / 突變 / 生成 candidate（不在本套件啟動，
//     由上層 UI 呼叫 mutation.go / guard.go 的 API）。
//
// 獨立資料夾結構（由 HookGeneDir 解析，見 paths.go）
//
//	data/projects/[project]/hook_gene/
//	├── recorder_events.jsonl            ← append-only 事件（best-effort、hash chain）
//	├── recorder_state.json              ← 目前 14 天統計 / 肥大 summary（temp+rename）
//	├── recorder_rotation_manifest.json  ← 各 rotated 檔的 tail hash 鏈
//	└── recorder_events.<ts>.jsonl       ← rotate 後的舊事件
//
// 維護指引
//   - 門檻（16 格 / ㄅ>=13 / 14 天 / 7 次 / 80%）為 MVP 寫死常數，集中在 gene.go / state.go，
//     日後要做 per-project config 從這兩處抽出即可。
//   - 加減 pressure bucket 時，請同步 complexity.go 的 PressureBucketCount 與幾何平均指數。
//   - Recorder 的所有檔案寫入都在「單一 writer goroutine」內完成；外部只透過 Emit 送訊號，
//     Emit 永不阻塞（queue 滿即丟棄並標 incomplete），詳見 recorder.go。
package hookgene
