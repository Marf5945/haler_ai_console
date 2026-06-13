package skill_eval

import "testing"

// V-31-18：覆蓋率與 new subagent candidate 門檻。
func TestUncoveredAndCandidate(t *testing.T) {
	if r := UncoveredRatio(6, 10); r != 0.6 {
		t.Fatalf("ratio want 0.6 got %v", r)
	}
	if !IsNewSubagentCandidate(0.6) {
		t.Fatal("0.6 應為 new subagent candidate")
	}
	if IsNewSubagentCandidate(0.5) {
		t.Fatal("0.5 不應超過門檻")
	}
}

// V-31-19：清理提醒門檻。
func TestShouldSuggestCleanup(t *testing.T) {
	if !ShouldSuggestCleanup(false, 31, 0.85) {
		t.Fatal("符合條件應提醒清理")
	}
	if ShouldSuggestCleanup(true, 99, 0.99) {
		t.Fatal("keep_forever 不應提醒")
	}
	if ShouldSuggestCleanup(false, 10, 0.85) {
		t.Fatal("未滿 30 天不應提醒")
	}
}

// V-31-20：合併候選——三核心條件 + RE 相似度。
func TestMergeCandidate(t *testing.T) {
	a := SkillSummary{ActionTag: "查詢", Purpose: "lookup", Domain: "天氣",
		RESteps: []string{"查詢|天氣", "整理|資料"}}
	b := SkillSummary{ActionTag: "查詢", Purpose: "lookup", Domain: "天氣",
		RESteps: []string{"查詢|天氣", "整理|資料"}}
	if !IsMergeCandidate(a, b) {
		t.Fatal("完全相同應為合併候選")
	}
	c := b
	c.ActionTag = "轉換" // action 不同 → 非候選
	if IsMergeCandidate(a, c) {
		t.Fatal("action_tag 不同不應為候選")
	}
}
