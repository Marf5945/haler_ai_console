// app_web_chain_test.go — Web Chain 純函式測試（2.5.5.11）。
package main

import (
	"strings"
	"testing"

	"ui_console/orchestration/skill_eval"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
)

// 續跑判斷：只有 網路ㄌ…ㄌ網路 才續跑。
func TestWebChainShouldContinue(t *testing.T) {
	cases := []struct {
		action, next string
		want         bool
	}{
		{"網路", "網路", true},
		{"網路", "待命", false},
		{"網路", "輸出", false},
		{"網路", "依天氣建議穿搭", false}, // 自由文字意圖不放行
		{"搜尋", "網路", false},
		{"網路", "等待指令", false}, // alias normalize 後是待命
	}
	for _, c := range cases {
		if got := webChainShouldContinue(c.action, c.next); got != c.want {
			t.Fatalf("webChainShouldContinue(%q,%q)=%v want %v", c.action, c.next, got, c.want)
		}
	}
}

// 期望鏈解析：單行無期望；多行取全部步驟。
func TestWebChainExpectedFromRaw(t *testing.T) {
	if got := webChainExpectedFromRaw("網路ㄌ台北 天氣ㄌ待命"); got != nil {
		t.Fatalf("單行不應有期望鏈，got %+v", got)
	}
	raw := "網路ㄌ台北 明天 天氣預報ㄌ網路\n網路ㄌ<依結果>ㄌ待命"
	steps := webChainExpectedFromRaw(raw)
	if len(steps) != 2 {
		t.Fatalf("期望 2 步，got %d", len(steps))
	}
	if steps[1].Target != "<依結果>" || steps[1].Next != actionchain.StandbyNext {
		t.Fatalf("第二步解析錯誤：%+v", steps[1])
	}
}

// drift 比對：action 相符無 drift；不符、超步有 reason；target 占位不比對。
func TestWebChainDriftReason(t *testing.T) {
	expected := webChainExpectedFromRaw("網路ㄌ台北 天氣ㄌ網路\n網路ㄌ<依結果>ㄌ待命")
	if r := webChainDriftReason(expected, 1, "網路"); r != "" {
		t.Fatalf("action 相符不應有 drift，got %q", r)
	}
	if r := webChainDriftReason(expected, 1, "聊天"); r == "" {
		t.Fatal("action 不符應有 drift reason")
	}
	if r := webChainDriftReason(expected, 5, "網路"); r == "" {
		t.Fatal("超出期望鏈步數應有 drift reason")
	}
	if r := webChainDriftReason(nil, 1, "網路"); r != "" {
		t.Fatalf("無期望鏈不報 drift，got %q", r)
	}
}

// 觀察截斷：防整篇網頁塞爆回灌。
func TestWebChainTruncateRunes(t *testing.T) {
	long := strings.Repeat("天", 5000)
	out := webChainTruncateRunes(long, 100)
	if len([]rune(out)) > 110 || !strings.HasSuffix(out, "…(截斷)") {
		t.Fatalf("截斷失敗，len=%d", len([]rune(out)))
	}
	if webChainTruncateRunes("短", 100) != "短" {
		t.Fatal("未超長不應截斷")
	}
}

// 回灌 prompt：含觀察區塊、原始問題與封閉格式。
func TestBuildWebChainFollowupPrompt(t *testing.T) {
	p := buildWebChainFollowupPrompt("台北明天天氣穿搭", "台北 天氣預報", "降雨機率70%")
	for _, want := range []string{"[觀察]", "[/觀察]", "Q=", "待命|網路", "閒聊ㄌ"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt 缺 %q：%s", want, p)
		}
	}
}

// 不符續跑條件時，原回應原封不動（含 nil stepCall 不得被呼叫）。
func TestMaybeContinueWebChainNoop(t *testing.T) {
	app := &App{}
	first := skill_step.CLIResponse{Text: "結果", Action: "網路", Target: "台北 天氣", Next: actionchain.StandbyNext}
	got := app.maybeContinueWebChain(nil, "s1", "t1", "台北天氣", "網路ㄌ台北 天氣ㄌ待命", first)
	if got.Text != first.Text || got.Next != first.Next {
		t.Fatalf("next=待命 不應續跑：%+v", got)
	}
}

// Phase 2：drift 事件結構正確（low risk、不阻擋、kind=action_mismatch）。
func TestBuildWebChainDriftEvent(t *testing.T) {
	ev := buildWebChainDriftEvent("網路", "<依結果>", "待命", "聊天", "直接回答", "待命", "action 不符")
	if ev.Kind != skill_eval.DriftActionMismatch {
		t.Fatalf("kind 應為 action_mismatch，got %s", ev.Kind)
	}
	if ev.Risk != skill_eval.RiskLow || ev.Blocked {
		t.Fatal("web chain drift 應 low risk 且不阻擋")
	}
	if ev.Expected.Action != "網路" || ev.Actual.Action != "聊天" {
		t.Fatalf("期望/實際 action 填錯：%+v", ev)
	}
}

// Phase 2：有 drift 才寫一筆事件；無 drift 不落檔。sink 注入攔截，不碰檔案系統。
func TestRecordWebChainDrifts(t *testing.T) {
	orig := webChainEvalAppend
	defer func() { webChainEvalAppend = orig }()

	var got []skill_eval.EventRecord
	webChainEvalAppend = func(rec skill_eval.EventRecord) error {
		got = append(got, rec)
		return nil
	}

	recordWebChainDrifts("s1", nil) // 無 drift
	if len(got) != 0 {
		t.Fatalf("無 drift 不應寫入，got %d", len(got))
	}

	recordWebChainDrifts("s1", []skill_eval.DriftEvent{
		buildWebChainDriftEvent("網路", "<依結果>", "待命", "聊天", "x", "待命", "不符"),
	})
	if len(got) != 1 {
		t.Fatalf("應寫入 1 筆，got %d", len(got))
	}
	if got[0].SessionID != "s1" || got[0].Note != "web_chain" || len(got[0].Drifts) != 1 {
		t.Fatalf("事件內容錯誤：%+v", got[0])
	}
}

// Phase 3：多步鏈才記 run；單步不記（省雜訊）。sink 注入攔截。
func TestRecordWebChainRun(t *testing.T) {
	orig := webChainRunAppend
	defer func() { webChainRunAppend = orig }()

	var got []skill_eval.WebChainRun
	webChainRunAppend = func(rec skill_eval.WebChainRun) error {
		got = append(got, rec)
		return nil
	}

	recordWebChainRun("s1", []string{"網路"}, 0) // 單步不記
	if len(got) != 0 {
		t.Fatalf("單步不應記 run，got %d", len(got))
	}

	recordWebChainRun("s1", []string{"網路", "網路"}, 1)
	if len(got) != 1 {
		t.Fatalf("多步應記 1 筆，got %d", len(got))
	}
	if got[0].Signature != "網路>網路" || got[0].Steps != 2 || got[0].DriftCount != 1 {
		t.Fatalf("run 內容錯誤：%+v", got[0])
	}
}

// 環境旗標：AI_CONSOLE_WEB_CHAIN=0 整體關閉，預設開。
func TestWebChainEnabledFlag(t *testing.T) {
	t.Setenv("AI_CONSOLE_WEB_CHAIN", "")
	if !webChainEnabled() {
		t.Fatal("預設應啟用")
	}
	t.Setenv("AI_CONSOLE_WEB_CHAIN", "0")
	if webChainEnabled() {
		t.Fatal("=0 應關閉")
	}
}
