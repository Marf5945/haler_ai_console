// report.go — sub 覆蓋率 / 清理 / 合併候選的「純邏輯門檻」（TASK 31 / Phase 3）。
// 注意：真正的命中資料需等 subagent executor 獨立後才可信（見 TASKS Phase 3 整合說明）；
// 此檔只提供可單元測試的判準，不自行抓資料。
package skill_eval

import "strings"

// 門檻常數（與 implementation review 一致）。
const (
	NewSubagentUncoveredThreshold = 0.50 // uncovered_ratio 超過 → new subagent candidate
	SubCleanupCoveredThreshold    = 0.80 // 被其他 sub 覆蓋率超過 → 提醒清理
	MergeRESimilarityThreshold    = 0.80 // RE 必要步驟相似度達標才算合併候選
	SubIdleDaysForCleanup         = 30   // 連續未獨立命中天數
)

// UncoveredRatio = 未被既有 sub 覆蓋的步驟數 / 全部步驟數。
func UncoveredRatio(uncoveredSteps, totalSteps int) float64 {
	if totalSteps <= 0 {
		return 0
	}
	return float64(uncoveredSteps) / float64(totalSteps)
}

// IsNewSubagentCandidate 超過一半未覆蓋 → 建議新增 subagent。
func IsNewSubagentCandidate(uncoveredRatio float64) bool {
	return uncoveredRatio > NewSubagentUncoveredThreshold
}

// ShouldSuggestCleanup：keep_forever != true 且 30 天未獨立命中 且 被其他 sub 覆蓋率 > 80%。
func ShouldSuggestCleanup(keepForever bool, daysSinceIndependentHit int, coveredByOtherRatio float64) bool {
	if keepForever {
		return false
	}
	return daysSinceIndependentHit >= SubIdleDaysForCleanup &&
		coveredByOtherRatio > SubCleanupCoveredThreshold
}

// SkillSummary 是合併判定所需的最小資訊（只用三個核心條件）。
type SkillSummary struct {
	ActionTag string
	Purpose   string   // purpose/domain
	Domain    string
	RESteps   []string // RE 必要步驟的簽章（如 "查詢|天氣"）
}

// IsMergeCandidate：action_tag 相同 + purpose/domain 相同 + RE 步驟相似度 >= 0.80。
// 權限/code stream/使用結果不作為主條件（僅供 review 顯示）。
func IsMergeCandidate(a, b SkillSummary) bool {
	if !eqTrim(a.ActionTag, b.ActionTag) {
		return false
	}
	if !eqTrim(a.Purpose, b.Purpose) || !eqTrim(a.Domain, b.Domain) {
		return false
	}
	return REStepSimilarity(a.RESteps, b.RESteps) >= MergeRESimilarityThreshold
}

// REStepSimilarity 以 Jaccard（交集/聯集）計算兩組 RE 步驟簽章的相似度 [0,1]。
func REStepSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	set := map[string]bool{}
	for _, x := range a {
		set[strings.TrimSpace(x)] = true
	}
	inter := 0
	bset := map[string]bool{}
	for _, y := range b {
		y = strings.TrimSpace(y)
		bset[y] = true
		if set[y] {
			inter++
		}
	}
	union := len(set)
	for y := range bset {
		if !set[y] {
			union++
		}
	}
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

func eqTrim(a, b string) bool { return strings.TrimSpace(a) == strings.TrimSpace(b) }
