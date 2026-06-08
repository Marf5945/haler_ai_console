// Package replan 實作 Bounded Replan v1：在 safety-first 架構上，讓 agent 在
// 「low-risk 且不改變目標」前提下自動改寫尚未開始的後續節點（tail），
// 連續無進展達上限即停下交人決定。
//
// 設計原則（對應 BOUNDED_REPLAN_SPEC）：
//   - LLM 只「提案」新 tail；Go 才「裁決」silent / review / stop。
//   - 硬天花板編譯在 Go：read-only allowlist + classifier 否決 + GoalContract 不變。
//   - 零第三方依賴：僅 stdlib + 內部 dag / risk / audit_log。
//
// 檔案分責：
//
//	types.go    列舉、read-only allowlist、全域 deny、提案型別。
//	contract.go GoalContract 行為（同目標/輸出驗證；資料型別在 dag）。
//	hash.go     sha256 共用 helper。
//	policy.go   Gate：硬規則裁決。
//	patch.go    TailPatch + CAS + tail hash + 原子套用。
//	counter.go  連續無進展 + 總閘 + 震盪偵測。
//	audit.go    safe summary + hash 的 append-only 稽核。
package replan
