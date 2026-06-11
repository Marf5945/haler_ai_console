package skill_flow

import (
	"strings"
	"testing"
	"time"
)

// stubResolver 依 query 回固定結果，模擬「本地粗篩＋模型精選」。
type stubResolver struct {
	byQuery map[string][]Candidate
	status  ResolveStatus
}

func (r stubResolver) Resolve(query string) ([]Candidate, ResolveStatus) {
	if r.status != "" && r.status != ResolveOK {
		return nil, r.status
	}
	for k, cs := range r.byQuery {
		if strings.Contains(query, k) {
			return cs, ResolveOK
		}
	}
	return nil, ResolveNone
}

func testDef() FlowDef {
	return FlowDef{
		SkillName:   "測試流程",
		CancelWords: []string{"取消", "cancel"},
		CancelText:  "已取消。",
		Fields: []FieldDef{{
			Name:     "machine",
			Prompt:   "請問機台名稱是？",
			RePrompt: "請給我機台名稱。",
			Ack: func(v string) string {
				return "機台「" + v + "」收到。請輸入第一項。"
			},
		}},
		List: &ListDef{
			MinItems:          1,
			MinItemsPrompt:    "至少要一項。",
			Prompt:            "請輸入第一項。",
			NextPrompt:        "請輸入下一項。",
			DoneWords:         []string{"不要", "輸出"},
			MoreWords:         []string{"繼續", "補"},
			QtyPrompt:         "請問數量是多少？",
			QtyRePrompt:       "請給我數量。",
			NoSourceText:      "找不到資料來源。",
			MatchedText:       "已對應到：%s。",
			NoMatchText:       "查無相符，先照原樣帶入「%s」。",
			PickedText:        "選了：%s。",
			PickPrompt:        "「%s」找到幾個相近的，要哪一個？",
			PickPromptNoQuery: "要哪一個？",
			PickRePrompt:      "沒看懂選哪個。",
			ReviewHeader:      "目前已收 %d 項：",
			ReviewTail:        "要不要補？",
			FixNoHitText:      "沒對到要修正哪一項。",
			FixPickPrompt:     "有多項符合，要修正哪一項？回清單編號：",
			FixPickRePrompt:   "請回清單編號（%s）。",
			FixQtyPrompt:      "要把第 %d 項（%s）的數量改成多少？",
			FixDoneText:       "已把第 %d 項（%s）數量 %s → %s。",
			FixOutOfRangeText: "清單只有 %d 項，沒有第 %d 項。",
		},
	}
}

func newTestEngine() (*Engine, *State) {
	hmi := Candidate{Value: "80160001", Label: "威綸.10.1\" 人機（80160001）", Name: "威綸.10.1\" 人機"}
	res := stubResolver{byQuery: map[string][]Candidate{
		"人機":   {hmi},
		"murr": {{Value: "Q-001", Label: "Murr 接線（Q-001）", Name: "Murr 接線"}, {Value: "Q-002", Label: "Murr 電源（Q-002）", Name: "Murr 電源"}},
	}}
	e := &Engine{Def: testDef(), Resolver: res}
	return e, &State{}
}

func TestEngineHappyPath(t *testing.T) {
	e, st := newTestEngine()

	out := e.Start(st)
	if out.Ask != "請問機台名稱是？" {
		t.Fatalf("start ask = %q", out.Ask)
	}
	out = e.Handle(st, "aaa-122")
	if !strings.Contains(out.Ask, "機台「aaa-122」收到") {
		t.Fatalf("machine ack = %q", out.Ask)
	}
	// 「人機 21」一行帶數量 → 單筆自動帶入 → review
	out = e.Handle(st, "人機 21")
	if !strings.Contains(out.Ask, "已對應到") || !strings.Contains(out.Ask, "1. 80160001 ×21") {
		t.Fatalf("collect = %q", out.Ask)
	}
	// 「不要」→ 完成
	out = e.Handle(st, "不要")
	if !out.Complete {
		t.Fatalf("done: %+v", out)
	}
	if st.Values["machine"] != "aaa-122" || len(st.Items) != 1 || st.Items[0].Qty != "21" {
		t.Fatalf("state: %+v", st)
	}
}

func TestEngineQtyFollowUp(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	out := e.Handle(st, "人機") // 沒帶數量
	if !strings.Contains(out.Ask, "請問數量是多少？") {
		t.Fatalf("ask qty = %q", out.Ask)
	}
	out = e.Handle(st, "5")
	if !strings.Contains(out.Ask, "1. 80160001 ×5") {
		t.Fatalf("review = %q", out.Ask)
	}
}

func TestEnginePick(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	out := e.Handle(st, "murr 3") // 兩筆候選 → 反問
	if st.Phase != PhasePick || !strings.Contains(out.Ask, "#1. ") {
		t.Fatalf("pick ask = %q phase=%s", out.Ask, st.Phase)
	}
	out = e.Handle(st, "亂講")
	if !strings.Contains(out.Ask, "沒看懂選哪個") {
		t.Fatalf("re-ask = %q", out.Ask)
	}
	out = e.Handle(st, "2")
	if !strings.Contains(out.Ask, "選了：Murr 電源") || !strings.Contains(out.Ask, "1. Q-002 ×3") {
		t.Fatalf("picked = %q", out.Ask)
	}
}

func TestEngineNoMatchKeepRaw(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	out := e.Handle(st, "不存在的料 2")
	if !strings.Contains(out.Ask, "查無相符") || !strings.Contains(out.Ask, "1. 不存在的料 ×2") {
		t.Fatalf("keep raw = %q", out.Ask)
	}
}

func TestEngineFixSingle(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	e.Handle(st, "人機 21")
	out := e.Handle(st, "幫我修正 人機 為 4項")
	if !strings.Contains(out.Ask, "已把第 1 項（80160001）數量 21 → 4") {
		t.Fatalf("fix = %q", out.Ask)
	}
	if st.Items[0].Qty != "4" {
		t.Fatalf("qty = %q", st.Items[0].Qty)
	}
}

func TestEngineFixDuplicateNeedsPick(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	e.Handle(st, "人機 21")
	e.Handle(st, "人機 1") // 第二筆同料號
	out := e.Handle(st, "幫我修正 人機 為 4項")
	if st.Phase != PhaseFixPick || !strings.Contains(out.Ask, "有多項符合") {
		t.Fatalf("fix pick ask = %q phase=%s", out.Ask, st.Phase)
	}
	out = e.Handle(st, "2")
	if !strings.Contains(out.Ask, "已把第 2 項（80160001）數量 1 → 4") {
		t.Fatalf("fix applied = %q", out.Ask)
	}
	if st.Items[0].Qty != "21" || st.Items[1].Qty != "4" {
		t.Fatalf("items: %+v", st.Items)
	}
}

func TestEngineFixMissingQty(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	e.Handle(st, "人機 21")
	out := e.Handle(st, "人機數量錯了")
	if !strings.Contains(out.Ask, "改成多少") {
		t.Fatalf("ask new qty = %q", out.Ask)
	}
	out = e.Handle(st, "7")
	if !strings.Contains(out.Ask, "21 → 7") || st.Items[0].Qty != "7" {
		t.Fatalf("fix = %q items=%+v", out.Ask, st.Items)
	}
}

func TestEngineFixIndexOutOfRange(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	e.Handle(st, "人機 21")
	out := e.Handle(st, "第5項改成2")
	if !strings.Contains(out.Ask, "沒有第 5 項") {
		t.Fatalf("out of range = %q", out.Ask)
	}
}

func TestEngineCancelAndMinItems(t *testing.T) {
	e, st := newTestEngine()
	e.Start(st)
	e.Handle(st, "M1")
	out := e.Handle(st, "輸出") // 還沒收任何項
	if !strings.Contains(out.Ask, "至少要一項") {
		t.Fatalf("min items = %q", out.Ask)
	}
	out = e.Handle(st, "取消")
	if !out.Cancelled || out.Text != "已取消。" {
		t.Fatalf("cancel: %+v", out)
	}
}

func TestEngineNoSource(t *testing.T) {
	e, st := newTestEngine()
	e.Resolver = stubResolver{status: ResolveNoSource}
	e.Start(st)
	e.Handle(st, "M1")
	out := e.Handle(st, "人機 21")
	if out.Ask != "找不到資料來源。" {
		t.Fatalf("no source = %q", out.Ask)
	}
}

func TestStoreTTL(t *testing.T) {
	s := NewStore(10 * time.Millisecond)
	s.Put("a", &State{})
	if _, ok := s.Get("a"); !ok {
		t.Fatal("should exist")
	}
	time.Sleep(15 * time.Millisecond)
	if _, ok := s.Get("a"); ok {
		t.Fatal("should expire")
	}
	s.Put("b", &State{})
	s.Delete("b")
	if _, ok := s.Get("b"); ok {
		t.Fatal("should be deleted")
	}
}
