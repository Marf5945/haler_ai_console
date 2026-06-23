package hookgene

// SafeExportSummary 是可匯出的精簡摘要（§3.1.5.18.7 / H-15）。
// 刻意只含統計摘要：不含完整 gene 序列、candidate action 細節、內部 skill 結構，
// 也不含任何資料內容 / 路徑 / credential。
type SafeExportSummary struct {
	SkillID         string  `json:"skill_id"`
	CompleteSamples int     `json:"complete_samples"`
	BloatRatio      float64 `json:"bloat_ratio"`
	HookComplexity  float64 `json:"hook_complexity"`
	ReviewSuggested bool    `json:"review_suggested"`
}

// BuildSafeExport 由 SkillStats 投影出可安全匯出的 summary。
func BuildSafeExport(s *SkillStats) SafeExportSummary {
	return SafeExportSummary{
		SkillID:         s.SkillID,
		CompleteSamples: s.CompleteCount(),
		BloatRatio:      s.BloatRatio(),
		HookComplexity:  s.HookComplexity,
		ReviewSuggested: s.ShouldPromptReview(),
	}
}
