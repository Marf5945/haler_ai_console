// visual_learning/model_hashes_embed.go — 模型 hash manifest 嵌入（§14.6.6）。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.6
//
// 安全性設計：
//   manifest 使用 //go:embed 嵌入二進位，攻擊者無法同時替換模型和 manifest。
//   如果需要更新模型，必須重新編譯 app（manifest 跟著更新）。
//
// 更新流程：
//   1. 放入新的 .mlmodelc 或 .onnx 模型
//   2. 用 model_verify.go 的演算法計算 SHA256
//   3. 更新 model_hashes.json
//   4. 重新編譯 — 新 hash 自動嵌入
package visual_learning

import _ "embed"

// embeddedModelHashes 是受信任的模型 hash manifest。
// 由 //go:embed 在編譯時嵌入，無法在 runtime 被篡改。
// LoadModel() 使用此 manifest 驗證模型完整性。
//
//go:embed model_hashes.json
var embeddedModelHashes []byte
