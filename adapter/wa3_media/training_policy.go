// wa3_media/training_policy.go — §9A.14 訓練資料政策 + §11 LLM Context 整合。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 訓練資料政策決定媒體是否可安全進入模型訓練語料。            │
// │                                                             │
// │ 判定規則（§9A.14）：                                        │
// │  exact_original         → training_safe = true              │
// │  wa3_app_processed      → 依 app trust + 操作指紋判定       │
// │  platform_processed_copy → training_safe = false            │
// │  model_pollution_risk   → training_safe = false + 過濾標記  │
// │  其他                   → training_safe = false             │
// │                                                             │
// │ §11 LLM Context 整合：                                      │
// │  model_pollution_risk 的媒體必須被 LLM Context Governance   │
// │  過濾，不得進入任何模型的上下文或訓練管線。                 │
// └─────────────────────────────────────────────────────────────┘
package wa3_media

// ──────────────────────────────────────────────
// 訓練資料政策判定
// ──────────────────────────────────────────────

// EvaluateTrainingEligibility 根據驗證狀態判定訓練資料資格。
func EvaluateTrainingEligibility(info *WA3MediaInfo) TrainingEligibility {
	switch info.Status {
	case StatusExactOriginal:
		return TrainingEligibility{
			TrainingSafe:   true,
			Reason:         "byte hash 完全匹配原始檔案",
			FilterRequired: false,
		}

	case StatusWA3AppProcessed:
		// 需檢查開發者簽章的有效性
		if info.DeveloperSignature != nil {
			return TrainingEligibility{
				TrainingSafe:   true,
				Reason:         "WA3-aware app 已簽署操作指紋，app trust 有效",
				FilterRequired: false,
			}
		}
		// 無簽章 → 僅作為 low-trust hint
		return TrainingEligibility{
			TrainingSafe:   false,
			Reason:         "WA3-aware app 操作指紋未簽署，無法確認訓練安全性",
			FilterRequired: false,
		}

	case StatusPlatformProcessed:
		return TrainingEligibility{
			TrainingSafe:   false,
			Reason:         "平台處理版本，不得視為訓練安全原檔（§9A.8）",
			FilterRequired: false,
		}

	case StatusModelPollutionRisk:
		return TrainingEligibility{
			TrainingSafe:   false,
			Reason:         "偵測到模型污染風險，必須從訓練語料與 LLM Context 中過濾（§9A.9）",
			FilterRequired: true, // 需要 §11 LLM Context Governance 過濾
		}

	case StatusUnauthorizedCopy:
		return TrainingEligibility{
			TrainingSafe:   false,
			Reason:         "未授權複製，無延伸權",
			FilterRequired: false,
		}

	case StatusContentModified:
		return TrainingEligibility{
			TrainingSafe:   false,
			Reason:         "內容已被非 WA3-aware 修改",
			FilterRequired: false,
		}

	default: // StatusUnverified
		return TrainingEligibility{
			TrainingSafe:   false,
			Reason:         "無可用 WA3 驗證證據",
			FilterRequired: false,
		}
	}
}

// ──────────────────────────────────────────────
// §11 LLM Context Governance 整合接口
// ──────────────────────────────────────────────

// ShouldFilterFromLLMContext 判斷媒體是否應從 LLM Context 中過濾。
// 與 §11 LLM Context Governance 模組整合使用。
func ShouldFilterFromLLMContext(info *WA3MediaInfo) bool {
	if info == nil {
		return false
	}
	// model_pollution_risk 媒體必須過濾
	if info.Status == StatusModelPollutionRisk {
		return true
	}
	// 有 pollution report 且超過閾值也過濾
	if info.Pollution != nil && info.Pollution.IsPollutionRisk {
		return true
	}
	return false
}

// MarkTrainingEligibility 計算並填入 WA3MediaInfo 的 Training 欄位。
func MarkTrainingEligibility(info *WA3MediaInfo) {
	result := EvaluateTrainingEligibility(info)
	info.Training = &result
}
