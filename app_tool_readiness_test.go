package main

import (
	"strings"
	"testing"

	"ui_console/shared/actionchain"
)

func newToolReadinessTestApp() *App {
	return &App{
		pendingToolQuestions: make(map[string]pendingToolQuestion),
		clarRoots:            make(map[string]*clarificationRoot),
	}
}

func TestActionChainQuestionNext(t *testing.T) {
	if !actionchain.IsQuestionNext("提問") {
		t.Fatal("提問 should be a next-state")
	}
	if actionchain.IsQuestionNext("待命") {
		t.Fatal("待命 should not be a question next-state")
	}
}

// 回歸釘子（收尾版）：移除關鍵字表後，網路+待命不再自動問地點。
// 「星座預報」這類查詢交由 judge 決定要不要澄清，不再被 isContextSensitiveWebQuery 誤判。
func TestNoKeywordDrivenClarification(t *testing.T) {
	app := newToolReadinessTestApp()
	for _, target := range []string{"今天會下雨嗎", "星座預報", "最新新聞"} {
		decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "網路", Target: target, Next: actionchain.StandbyNext}
		if handled, _ := app.maybeAskForToolReadiness("s1", decision, target, "trace-test"); handled {
			t.Fatalf("網路+待命 不應因關鍵字自動澄清: %q", target)
		}
	}
}

// judge 標記 next=提問 時仍會問（次要路徑保留），並記錄 pending 供補答重跑。
func TestQuestionNextStillAsks(t *testing.T) {
	app := newToolReadinessTestApp()
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "讀取", Target: "那個檔案", Next: actionchain.QuestionNext}
	handled, resp := app.maybeAskForToolReadiness("s1", decision, "讀取那個檔案", "trace-test")
	if !handled {
		t.Fatal("next=提問 應該要問")
	}
	if resp.Action != "讀取" || resp.Next != actionchain.QuestionNext {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if _, ok := app.pendingToolQuestions["s1"]; !ok {
		t.Fatal("pending question 未記錄")
	}
}

// 補答後 consume 回乾淨 re-judge 文字（根問題 + 答案），且 consume 後刪 pending（反劫持）。
func TestConsumeReturnsCleanRejudgeText(t *testing.T) {
	app := newToolReadinessTestApp()
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "讀取", Target: "今天會下雨嗎", Next: actionchain.QuestionNext}
	if handled, _ := app.maybeAskForToolReadiness("s1", decision, "今天會下雨嗎", "trace-test"); !handled {
		t.Fatal("expected initial question")
	}
	rejudge, ok := app.consumePendingToolAnswer("s1", "台北", "trace-test")
	if !ok {
		t.Fatal("expected clarification consumed")
	}
	if !strings.Contains(rejudge, "今天會下雨嗎") || !strings.Contains(rejudge, "台北") {
		t.Fatalf("rejudge text 應含根問題+答案, got %q", rejudge)
	}
	if strings.Contains(rejudge, "[已補充背景]") {
		t.Fatalf("rejudge text 不應含框架標記, got %q", rejudge)
	}
	if _, ok := app.pendingToolQuestions["s1"]; ok {
		t.Fatal("pending 應在 consume 後刪除（反劫持）")
	}
}

// Step 3：同一原始問題最多 2 次澄清，第 3 次走耗盡分支（max_rounds）。
func TestClarificationRoundCap(t *testing.T) {
	app := newToolReadinessTestApp()
	if ok, _ := app.storeClarification("s1", "網路", "天氣", "天氣", "你想查哪個地點？"); !ok {
		t.Fatal("round 1 應允許")
	}
	if ok, _ := app.storeClarification("s1", "網路", "天氣", "天氣", "你要查哪一天？"); !ok {
		t.Fatal("round 2 應允許")
	}
	if ok, reason := app.storeClarification("s1", "網路", "天氣", "天氣", "還要哪些資訊？"); ok || reason != "max_rounds" {
		t.Fatalf("round 3 應被擋, ok=%v reason=%q", ok, reason)
	}
}

// Step 3 + 補充建議 1：同一 canonical slot 重問（即使措辭不同）即停。
func TestClarificationSameSlotStops(t *testing.T) {
	app := newToolReadinessTestApp()
	if ok, _ := app.storeClarification("s1", "網路", "天氣", "天氣", "你想查哪個地點？"); !ok {
		t.Fatal("round 1 應允許")
	}
	if ok, reason := app.storeClarification("s1", "網路", "天氣", "天氣", "你在哪個城市？"); ok || reason != "same_slot" {
		t.Fatalf("同 slot（不同措辭）應停, ok=%v reason=%q", ok, reason)
	}
}

// 補充建議 1：canonical slot 正規化把不同措辭歸到同一槽（去重的基礎）。
func TestCanonicalMissingSlot(t *testing.T) {
	for _, q := range []string{"你想查哪個地點？", "你在哪個城市？", "哪裡？"} {
		if got := canonicalMissingSlot(q); got != "location" {
			t.Fatalf("%q → %q, want location", q, got)
		}
	}
	if got := canonicalMissingSlot("你的星座是？"); got != "zodiac" {
		t.Fatalf("zodiac slot = %q", got)
	}
}

func TestQueryActionIsNotPromotedToWebSearch(t *testing.T) {
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "查詢", Target: "今天會下雨嗎", Next: actionchain.StandbyNext}
	normalized := normalizeToolRoutingDecision(decision, "今天會下雨嗎", toolRoutingLookupContext{Query: "今天會下雨嗎"})
	if normalized.Action != "查詢" {
		t.Fatalf("查詢 should stay stored-data query, got %#v", normalized)
	}
}
