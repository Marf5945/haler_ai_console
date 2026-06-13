package replan

import "fmt"

// ActivitySummary 產生給使用者看的一句活動摘要（status rail 用）。
// 由 failure + stage 組成，不暴露 raw 工具輸出或模型推理；
// 進入 adjusting 階段後附帶 (n/5) 計數，呼應「越接近上限越明顯」。
func ActivitySummary(failure FailureCategory, stage Stage, consecutive int) string {
	var base string
	switch failure {
	case FailureNoResults:
		base = "正在調整搜尋路線：原條件沒有命中，改用相關線索重查。"
	case FailurePathNotFound:
		base = "正在調整路線：原路徑找不到，改往其他位置查找。"
	case FailureTruncated:
		base = "正在調整路線：結果不完整，改用更精確的方式取得。"
	case FailureAmbiguous:
		base = "正在調整路線：結果不夠明確，換個策略再試。"
	default:
		base = "正在調整路線。"
	}
	if stage == StageAdjusting {
		return fmt.Sprintf("%s（%d/%d）", base, consecutive, MaxConsecutiveNoProgress)
	}
	return base
}
