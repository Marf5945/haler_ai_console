package hookgene

import "time"

// 統計門檻（MVP 寫死，§3.1.5.18.4 / §3.1.5.18.8；日後可改 per-project config）。
const (
	StatsWindow        = 14 * 24 * time.Hour // 14 天窗
	MinCompleteSamples = 7                   // 至少 7 次完整樣本才評估
	BloatPromptRatio   = 0.80                // >= 80% 樣本肥大才提示 review
)

// sample 是一次「完整」invocation 的精簡統計（只進 14 天窗，不存任何資料內容）。
type sample struct {
	At      time.Time `json:"at"`
	Bloated bool      `json:"bloated"`
}

// SkillStats 保存單一 skill 的 14 天樣本、incomplete 計數與第 8 桶 hook_complexity。
type SkillStats struct {
	SkillID         string   `json:"skill_id"`
	Samples         []sample `json:"samples"`          // 只含完整樣本（進 80% 分母）
	IncompleteCount int      `json:"incomplete_count"` // 不完整 invocation 計數（不入分母）
	HookComplexity  float64  `json:"hook_complexity"`  // 第 8 桶：結構性、不隨時間衰減
}

// prune 移除 14 天窗外的樣本。
func (s *SkillStats) prune(now time.Time) {
	cutoff := now.Add(-StatsWindow)
	out := make([]sample, 0, len(s.Samples))
	for _, sm := range s.Samples {
		if sm.At.After(cutoff) {
			out = append(out, sm)
		}
	}
	s.Samples = out
}

// CompleteCount 回傳目前窗內完整樣本數。
func (s *SkillStats) CompleteCount() int { return len(s.Samples) }

func (s *SkillStats) bloatedCount() int {
	n := 0
	for _, sm := range s.Samples {
		if sm.Bloated {
			n++
		}
	}
	return n
}

// BloatRatio 回傳窗內肥大樣本比例（0 樣本時為 0）。
func (s *SkillStats) BloatRatio() float64 {
	if len(s.Samples) == 0 {
		return 0
	}
	return float64(s.bloatedCount()) / float64(len(s.Samples))
}

// ShouldPromptReview：完整樣本 >= 7 且肥大比例 >= 80%（§3.1.5.18.4）。
// 平常不打擾；此判定僅供 debug/review/learning mode 顯示提示用。
func (s *SkillStats) ShouldPromptReview() bool {
	return s.CompleteCount() >= MinCompleteSamples && s.BloatRatio() >= BloatPromptRatio
}

// recomputeComplexity 在每次「完整 gene 觀測」後重算 hook_complexity。
// hook_complexity 是結構性指標：以目前窗內肥大比例表示，隨 gene 觀測更新、不隨時間自行衰減
// （§3.1.5.18.5）。只有 gene 修飾/拆分/突變/重新審核改變結構時才會變動。
func (s *SkillStats) recomputeComplexity() {
	s.HookComplexity = s.BloatRatio()
}

// RecorderState 是 recorder_state.json 的內容：衍生狀態，可由 events replay 重建。
type RecorderState struct {
	LastEventID string                 `json:"last_event_id"` // 提交點：最後成功套用的 event
	LastHash    string                 `json:"last_hash"`     // hash chain 尾端
	Skills      map[string]*SkillStats `json:"skills"`
	UpdatedAt   time.Time              `json:"updated_at"`
}
