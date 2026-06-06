// conversation/model_list.go — 動態模型清單 + 可用性驗證（§29.4）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 從已設定的 adapter / API profile + 本地模型動態組合清單。    │
// │ 不硬編碼任何模型名稱（禁止字串常量如 "GPT-4o-mini"）。     │
// │                                                             │
// │ 模型消失時：                                                │
// │  • 禁止靜默 fallback                                       │
// │  • 禁止 crash                                              │
// │  • 橫幅改為「原本的整理 LLM 無法使用，請重新選擇」         │
// └─────────────────────────────────────────────────────────────┘
package conversation

// ──────────────────────────────────────────────
// ModelOption 結構
// ──────────────────────────────────────────────

// ModelOption 單一可用模型的描述。
type ModelOption struct {
	ModelID     string `json:"model_id"`
	DisplayName string `json:"display_name"`
	Source      string `json:"source"`    // "local_ollama" | "local_lmstudio" | "cloud_adapter"
	Available   bool   `json:"available"`
}

// ──────────────────────────────────────────────
// 模型可用性驗證
// ──────────────────────────────────────────────

// ValidateModelAvailability 檢查模型是否仍可用。
// 此函式不硬編碼任何模型，僅從給定清單中查找。
func ValidateModelAvailability(modelID string, availableModels []ModelOption) bool {
	for _, m := range availableModels {
		if m.ModelID == modelID && m.Available {
			return true
		}
	}
	return false
}
