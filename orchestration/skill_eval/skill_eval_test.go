package skill_eval

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui_console/orchestration/skill_step"
	"ui_console/shared/controlseal"
)

// V-31-08：低階 drift（injection / executor），不需 expected。
func TestEvaluateLowLevel(t *testing.T) {
	actual := EvalStep{Action: "查詢", Target: "天氣", Next: "待命"}

	// fake seal → suspected_injection（high, blocked）
	inj := EvaluateLowLevel(actual, controlseal.SanitizedResult{HasFakeSeal: true}, "", "")
	if len(inj) != 1 || inj[0].Kind != DriftSuspectedInjection || !inj[0].Blocked {
		t.Fatalf("expected blocked suspected_injection, got %+v", inj)
	}
	// executor 不符 → executor_mismatch（low, 不 block）
	ex := EvaluateLowLevel(actual, controlseal.SanitizedResult{}, "tool_call", "subagent_call")
	if len(ex) != 1 || ex[0].Kind != DriftExecutorMismatch || ex[0].Blocked {
		t.Fatalf("expected non-blocking executor_mismatch, got %+v", ex)
	}
	// 一致 executor + 無 injection → 無 drift
	if got := EvaluateLowLevel(actual, controlseal.SanitizedResult{}, "tool_call", "tool_call"); len(got) != 0 {
		t.Fatalf("expected no drift, got %+v", got)
	}
}

// V-31-10：NormalizeNext 雙邊，「等待指令」vs「待命」不應假 mismatch。
func TestNormalizeNextBothSides(t *testing.T) {
	exp := &skill_step.ExpectedChain{Steps: []skill_step.ExpectedStep{
		{Action: "查詢", Target: "天氣", Next: "待命", Code: "ㄔ", Requirement: "RE"},
	}}
	actual := []EvalStep{{Action: "查詢", Target: "天氣", Next: "等待指令"}}
	res := EvaluateAgainstExpected(actual, exp)
	if len(res.Drifts) != 0 {
		t.Fatalf("normalized next should match, got drifts %+v", res.Drifts)
	}
}

// V-31-11：ㄇ 替代 OP step → 無 drift，score 恰 -0.1。
func TestMuReplacesOptional(t *testing.T) {
	exp := &skill_step.ExpectedChain{Steps: []skill_step.ExpectedStep{
		{Action: "查詢", Target: "天氣", Next: "待命", Code: "ㄔ", Requirement: "OP"},
	}}
	actual := []EvalStep{{Action: "提問", Target: "哪個城市", Next: "待命"}} // 模型改用詢問（ㄇ）
	res := EvaluateAgainstExpected(actual, exp)
	if len(res.Drifts) != 0 {
		t.Fatalf("mu-substitution should not drift, got %+v", res.Drifts)
	}
	if res.Score != 1.0+MuOptionalPenalty {
		t.Fatalf("expected score %.2f, got %.2f", 1.0+MuOptionalPenalty, res.Score)
	}
}

// V-31-12：超過 16 步只給低風險提示，不阻擋。
func TestOver16Hint(t *testing.T) {
	steps := make([]skill_step.ExpectedStep, 17)
	for i := range steps {
		steps[i] = skill_step.ExpectedStep{Action: "查詢", Target: "x", Code: "ㄔ", Requirement: "OP"}
	}
	res := EvaluateAgainstExpected(nil, &skill_step.ExpectedChain{Steps: steps})
	if len(res.Hints) == 0 {
		t.Fatal("expected over-16 hint")
	}
}

// V-31-13：非法草稿（非法 code / reserved tag）被擋。
func TestValidateDraft(t *testing.T) {
	good := &skill_step.ExpectedChain{Steps: []skill_step.ExpectedStep{
		{Action: "查詢", Target: "天氣", Code: "ㄔ", Requirement: "RE"},
	}}
	if p := ValidateDraft(good); len(p) != 0 {
		t.Fatalf("good draft should pass, got %v", p)
	}
	bad := &skill_step.ExpectedChain{Steps: []skill_step.ExpectedStep{
		{Action: "輸出", Target: "x", Code: "X", Requirement: "RE"}, // reserved tag + 非法 code
	}}
	if p := ValidateDraft(bad); len(p) < 2 {
		t.Fatalf("bad draft should report >=2 problems, got %v", p)
	}
}

// V-31-07：store 寫入含 schema，且目錄路徑為 skill_eval（接 shouldSkipDir 排除）。
func TestStoreAppend(t *testing.T) {
	tmp := t.TempDir()
	st := NewStore(tmp, "default")
	if !strings.HasSuffix(st.Dir(), filepath.Join("skill_eval")) {
		t.Fatalf("store dir should end with skill_eval, got %s", st.Dir())
	}
	if err := st.AppendEvent(EventRecord{SkillID: "weather.lookup"}); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	f, err := os.Open(filepath.Join(st.Dir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("open jsonl: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Scan()
	var rec EventRecord
	if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
		t.Fatalf("bad jsonl line: %v", err)
	}
	if rec.Schema != SchemaEventV1 {
		t.Fatalf("expected schema %s, got %s", SchemaEventV1, rec.Schema)
	}
}
