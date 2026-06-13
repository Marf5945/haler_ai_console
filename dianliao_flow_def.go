package main

import (
	"fmt"
	"strings"

	"ui_console/orchestration/skill_flow"
)

// dianliao_flow_def.go — 「產出電料Bom」的流程宣告（FlowDef）與 Resolver 轉接。
//
// Step 1：FlowDef 先由 Go 程式碼定義；Step 2 將序列化進 skill_manifest.json 的
// flow 區段，由 manifest 載入，本檔屆時移除。見 REFACTOR_SKILL_FLOW.md。

var (
	dianliaoDoneWords   = []string{"不要", "輸出", "好了", "ok", "結束", "完成", "不用了", "不補了"}
	dianliaoMoreWords   = []string{"繼續", "補", "要", "下一項", "下一個", "再補", "繼續補"}
	dianliaoCancelWords = []string{"取消", "算了", "cancel", "停止", "中止"}
)

func dianliaoFlowDef() skill_flow.FlowDef {
	return skill_flow.FlowDef{
		SkillName:   dianliaoBomSkillName,
		TTL:         dianliaoBomTTL,
		CancelWords: dianliaoCancelWords,
		CancelText:  "好，已取消電料BOM。需要時再說一次就行。",
		Fields: []skill_flow.FieldDef{{
			Name:      "machine",
			Prompt:    "要產出電料BOM。請問機台名稱是？（例如：SLM003）",
			RePrompt:  "請給我機台名稱（例如：SLM003）。",
			Normalize: normalizeMachineInput,
			Ack: func(v string) string {
				return fmt.Sprintf("機台「%s」收到。請輸入第一項電料，可打描述（例如：murr 10m）或料號＋數量。", v)
			},
		}},
		List: &skill_flow.ListDef{
			MinItems:       1,
			MinItemsPrompt: "至少要一項電料才能產出。請先輸入一項（例如：murr 10m）。",
			Prompt:         "請輸入第一項電料，可打描述（例如：murr 10m）或料號＋數量。",
			NextPrompt:     "好，請輸入下一項電料（描述或料號＋數量）。",
			DoneWords:      dianliaoDoneWords,
			MoreWords:      dianliaoMoreWords,

			QtyPrompt:   "請問數量是多少？（例如：5）",
			QtyRePrompt: "請給我數量（純數字，例如：5）。",

			NoSourceText: "找不到「電料編碼紀錄」檔，請先把它載入（拖入）再輸入電料。",
			MatchedText:  "已對應到：%s。",
			NoMatchText:  "查無相符料號，先照原樣帶入「%s」。",
			PickedText:   "選了：%s。",

			PickPrompt:        "「%s」找到幾個相近的電料，要哪一個？（回編號或點選）",
			PickPromptNoQuery: "要哪一個？（回編號或點選）",
			PickRePrompt:      "沒看懂選哪個。請回編號（例如 1），或直接輸入料號。",

			ReviewHeader: "目前已收 %d 項電料：",
			ReviewTail:   "要不要補？繼續輸入下一項，或回「不要／輸出」結束並產出檔案。",

			FixNoHitText:      "沒對到要修正哪一項。可以說「第N項改成X」或「料號 改成 X」。",
			FixPickPrompt:     "有多項符合，要修正哪一項？回清單編號：",
			FixPickRePrompt:   "請回清單編號（%s）。",
			FixQtyPrompt:      "要把第 %d 項（%s）的數量改成多少？（例如：5）",
			FixDoneText:       "已把第 %d 項（%s）數量 %s → %s。",
			FixOutOfRangeText: "清單只有 %d 項，沒有第 %d 項。",
		},
	}
}

// dianliaoResolver 把 App 的模糊比對（本地粗篩＋模型精選，見 dianliao_bom_match.go）
// 接成 skill_flow.Resolver。sessionID/traceID 在建構時綁定。
type dianliaoResolver struct {
	app       *App
	sessionID string
	traceID   string
}

func (r dianliaoResolver) Resolve(query string) ([]skill_flow.Candidate, skill_flow.ResolveStatus) {
	recs, status := r.app.resolveDianliaoItem(r.sessionID, r.traceID, query)
	switch status {
	case "no_db":
		return nil, skill_flow.ResolveNoSource
	case "none":
		return nil, skill_flow.ResolveNone
	}
	out := make([]skill_flow.Candidate, len(recs))
	for i, rec := range recs {
		out[i] = skill_flow.Candidate{
			Value: rec.BestPartNo(),
			Label: dianliaoCandLabel(rec),
			Name:  strings.TrimSpace(rec.Name),
		}
	}
	return out, skill_flow.ResolveOK
}

func (a *App) dianliaoEngine(sessionID, traceID string) *skill_flow.Engine {
	return &skill_flow.Engine{
		Def:      dianliaoFlowDef(),
		Resolver: dianliaoResolver{app: a, sessionID: sessionID, traceID: traceID},
	}
}
