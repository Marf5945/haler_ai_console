package skill_eval

import (
	"testing"
	"time"

	"ui_console/orchestration/skill_step"
	"ui_console/shared/riskgrant"
)

// V-31-14：執行決策——auto / 需確認 / 候選 / review / 無 skill。
func TestDecideExecution(t *testing.T) {
	enabledAuto := &skill_step.Lifecycle{Status: skill_step.LifecycleEnabled, AutoExecute: true}
	pending := &skill_step.Lifecycle{Status: skill_step.LifecyclePending, AutoExecute: false}

	cases := []struct {
		status skill_step.ResolveStatus
		lc     *skill_step.Lifecycle
		grant  bool
		want   ExecDecision
	}{
		{skill_step.StatusAutoSelected, enabledAuto, false, ExecAuto},
		{skill_step.StatusAutoSelected, pending, false, ExecNeedConfirm}, // pending 第一次要確認
		{skill_step.StatusAutoSelected, pending, true, ExecAuto},         // 允許一次後免確認
		{skill_step.StatusNeedsCLI, nil, false, ExecCandidate},
		{skill_step.StatusNeedsReview, nil, false, ExecReview},
		{skill_step.StatusRejected, nil, false, ExecNoSkill},
	}
	for i, c := range cases {
		if got := DecideExecution(c.status, c.lc, c.grant); got != c.want {
			t.Errorf("case %d: want %s got %s", i, c.want, got)
		}
	}
}

// V-31-17：允許一次走 riskgrant；同 session 再呼叫免再問。
func TestGrantOnceRoundtrip(t *testing.T) {
	store := riskgrant.NewStoreWithClock(5*time.Minute, time.Now)
	m := &skill_step.SkillManifest{
		SkillID: "weather.lookup",
		Tags:    skill_step.SkillTags{ActionTag: []string{"查詢"}, RiskTag: []string{"medium"}},
	}
	if HasSkillGrant(store, m) {
		t.Fatal("初始不應有授權")
	}
	GrantOnce(store, m)
	if !HasSkillGrant(store, m) {
		t.Fatal("允許一次後同 session 應命中授權")
	}
}

// V-31-15 / V-31-16：pending draft——通過驗證 → pending（可見/可候選/不自動執行）。
func TestBuildPendingDraft(t *testing.T) {
	good := &skill_step.ExpectedChain{Steps: []skill_step.ExpectedStep{
		{Action: "查詢", Target: "天氣", Code: "ㄔ", Requirement: "RE"},
	}}
	m, problems := BuildPendingDraft("user.weather", "查天氣", []string{"查詢"}, []string{"天氣"}, good)
	if len(problems) != 0 {
		t.Fatalf("good draft should pass: %v", problems)
	}
	if m.Lifecycle.Status != skill_step.LifecyclePending || !m.Lifecycle.VisibleInToolbar {
		t.Fatalf("pending 應可見: %+v", m.Lifecycle)
	}
	if m.Lifecycle.AutoExecute {
		t.Fatal("pending 不可自動執行")
	}

	// 非法草稿 → draft_candidate（不顯示為工具）
	bad := &skill_step.ExpectedChain{Steps: []skill_step.ExpectedStep{
		{Action: "輸出", Target: "x", Code: "X", Requirement: "RE"},
	}}
	m2, problems2 := BuildPendingDraft("user.bad", "壞草稿", nil, nil, bad)
	if len(problems2) == 0 || m2.Lifecycle.Status != skill_step.LifecycleDraftCandidate {
		t.Fatalf("bad draft 應降為 draft_candidate, problems=%v status=%s", problems2, m2.Lifecycle.Status)
	}
	if m2.Lifecycle.VisibleInToolbar {
		t.Fatal("draft_candidate 不應顯示為工具")
	}
}
