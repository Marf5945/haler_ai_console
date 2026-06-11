package skill_flow

import (
	"fmt"
	"strconv"
	"strings"
)

// engine.go — 通用狀態機：欄位收集 → 清單迴圈（lookup/數量/修正/review）→ 完成。
// 行為與原 dianliao_bom_flow.go 等價；所有文案來自 FlowDef，引擎不含 skill 專屬內容。

type Engine struct {
	Def      FlowDef
	Resolver Resolver
}

// Start 起始一輪流程：問第一個欄位（沒有欄位就直接進清單）。
func (e *Engine) Start(st *State) Outcome {
	if st.Values == nil {
		st.Values = map[string]string{}
	}
	st.Phase = PhaseField
	st.FieldIdx = 0
	if len(e.Def.Fields) == 0 {
		return e.enterList(st, "")
	}
	return Outcome{Ask: e.Def.Fields[0].Prompt}
}

// Handle 處理進行中流程的一回合輸入。
func (e *Engine) Handle(st *State, text string) Outcome {
	t := strings.TrimSpace(text)
	if MatchWord(t, e.Def.CancelWords) {
		return Outcome{Cancelled: true, Text: e.Def.CancelText}
	}
	switch st.Phase {
	case PhaseField:
		return e.handleField(st, t)
	case PhaseCollect:
		return e.handleCollect(st, t)
	case PhasePick:
		return e.handlePick(st, t)
	case PhaseQty:
		return e.handleQty(st, t)
	case PhaseFixPick:
		return e.handleFixPick(st, t)
	}
	return Outcome{Cancelled: true, Text: e.Def.CancelText}
}

func (e *Engine) handleField(st *State, text string) Outcome {
	if st.FieldIdx < 0 || st.FieldIdx >= len(e.Def.Fields) {
		return e.enterList(st, "")
	}
	f := e.Def.Fields[st.FieldIdx]
	v := text
	if f.Normalize != nil {
		v = f.Normalize(v)
	}
	if strings.TrimSpace(v) == "" {
		return Outcome{Ask: f.RePrompt}
	}
	st.Values[f.Name] = v
	st.FieldIdx++
	ack := ""
	if f.Ack != nil {
		ack = f.Ack(v)
	}
	if st.FieldIdx < len(e.Def.Fields) {
		if ack == "" {
			ack = e.Def.Fields[st.FieldIdx].Prompt
		}
		return Outcome{Ask: ack}
	}
	return e.enterList(st, ack)
}

func (e *Engine) enterList(st *State, ack string) Outcome {
	if e.Def.List == nil {
		return Outcome{Complete: true}
	}
	st.Phase = PhaseCollect
	if ack == "" {
		ack = e.Def.List.Prompt
	}
	return Outcome{Ask: ack}
}

func (e *Engine) handleCollect(st *State, text string) Outcome {
	l := e.Def.List
	if MatchWord(text, l.DoneWords) {
		if len(st.Items) < l.MinItems {
			return Outcome{Ask: l.MinItemsPrompt}
		}
		return Outcome{Complete: true}
	}
	if MatchWord(text, l.MoreWords) {
		return Outcome{Ask: l.NextPrompt}
	}
	if len(st.Items) > 0 && HasFixIntent(text) {
		return e.handleFix(st, text)
	}

	query, qty := SplitItemInput(text)
	if strings.TrimSpace(query) == "" {
		return Outcome{Ask: l.NextPrompt}
	}
	cands, status := e.Resolver.Resolve(query)
	switch status {
	case ResolveNoSource:
		return Outcome{Ask: l.NoSourceText}
	case ResolveNone:
		// 查無相符：照原樣帶入（輸出時由 on_complete 端提醒）。
		st.PendingValue, st.PendingLabel, st.PendingQty = query, query, qty
		return e.afterResolve(st, fmt.Sprintf(l.NoMatchText, query))
	default: // ResolveOK
		if len(cands) == 1 {
			c := cands[0]
			st.PendingValue, st.PendingLabel, st.PendingQty = c.Value, c.Label, qty
			return e.afterResolve(st, fmt.Sprintf(l.MatchedText, c.Label))
		}
		st.Candidates = cands
		st.PendingQty = qty
		st.Phase = PhasePick
		return Outcome{Ask: buildPickAsk(fmt.Sprintf(l.PickPrompt, query), cands)}
	}
}

// afterResolve 在項目值決定後：缺數量就問數量，否則加入清單並回 review。
func (e *Engine) afterResolve(st *State, prefix string) Outcome {
	l := e.Def.List
	if strings.TrimSpace(st.PendingQty) == "" {
		st.Phase = PhaseQty
		return Outcome{Ask: prefix + l.QtyPrompt}
	}
	e.appendPending(st)
	st.Phase = PhaseCollect
	return Outcome{Ask: prefix + "\n" + e.reviewPrompt(st)}
}

func (e *Engine) handlePick(st *State, text string) Outcome {
	l := e.Def.List
	idx, ok := MatchPick(text, st.Candidates)
	if !ok {
		return Outcome{Ask: l.PickRePrompt + "\n" + buildPickAsk(l.PickPromptNoQuery, st.Candidates)}
	}
	c := st.Candidates[idx]
	st.PendingValue, st.PendingLabel = c.Value, c.Label
	st.Candidates = nil
	return e.afterResolve(st, fmt.Sprintf(l.PickedText, c.Label))
}

func (e *Engine) handleQty(st *State, text string) Outcome {
	l := e.Def.List
	qty, ok := ParseQty(text)
	if !ok {
		return Outcome{Ask: l.QtyRePrompt}
	}
	if st.FixIndex >= 1 && st.FixIndex <= len(st.Items) {
		return e.applyFix(st, st.FixIndex, qty)
	}
	st.PendingQty = qty
	e.appendPending(st)
	st.Phase = PhaseCollect
	return Outcome{Ask: e.reviewPrompt(st)}
}

// appendPending 把暫存的待加項加入 Items，並清掉暫存欄位。
func (e *Engine) appendPending(st *State) {
	st.Items = append(st.Items, Item{
		Value: st.PendingValue,
		Label: st.PendingLabel,
		Qty:   st.PendingQty,
	})
	st.PendingValue, st.PendingLabel, st.PendingQty = "", "", ""
	st.Candidates = nil
}

// ---------- 修正（fix） ----------

func (e *Engine) handleFix(st *State, text string) Outcome {
	l := e.Def.List
	target, idx, qty := ParseFix(text)

	var hits []int
	switch {
	case idx >= 1 && idx <= len(st.Items):
		hits = []int{idx}
	case idx != 0:
		return Outcome{Ask: fmt.Sprintf(l.FixOutOfRangeText, len(st.Items), idx) + "\n" + e.reviewPrompt(st)}
	default:
		hits = e.matchFixItems(st, target)
	}

	switch len(hits) {
	case 0:
		return Outcome{Ask: l.FixNoHitText + "\n" + e.reviewPrompt(st)}
	case 1:
		if qty == "" {
			st.FixIndex = hits[0]
			st.Phase = PhaseQty
			return Outcome{Ask: fmt.Sprintf(l.FixQtyPrompt, hits[0], st.Items[hits[0]-1].Value)}
		}
		return e.applyFix(st, hits[0], qty)
	default:
		st.FixIndexes = hits
		st.FixQty = qty
		st.Phase = PhaseFixPick
		var b strings.Builder
		b.WriteString(l.FixPickPrompt + "\n")
		for _, h := range hits {
			it := st.Items[h-1]
			q := it.Qty
			if q == "" {
				q = "(未填數量)"
			}
			fmt.Fprintf(&b, "%d. %s ×%s\n", h, it.Value, q)
		}
		return Outcome{Ask: strings.TrimRight(b.String(), "\n")}
	}
}

// matchFixItems 在已收清單中定位要修正的項目（1-based）。
// 先比 Value 子字串；沒中再用 Resolver 模糊解析找值對回清單。
func (e *Engine) matchFixItems(st *State, target string) []int {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	low := strings.ToLower(target)
	var hits []int
	for i, it := range st.Items {
		v := strings.ToLower(strings.TrimSpace(it.Value))
		if v != "" && (strings.Contains(low, v) || strings.Contains(v, low)) {
			hits = append(hits, i+1)
		}
	}
	if len(hits) > 0 {
		return hits
	}
	cands, status := e.Resolver.Resolve(target)
	if status != ResolveOK {
		return nil
	}
	want := map[string]bool{}
	for _, c := range cands {
		if v := strings.ToLower(strings.TrimSpace(c.Value)); v != "" {
			want[v] = true
		}
	}
	for i, it := range st.Items {
		if want[strings.ToLower(strings.TrimSpace(it.Value))] {
			hits = append(hits, i+1)
		}
	}
	return hits
}

func (e *Engine) handleFixPick(st *State, text string) Outcome {
	l := e.Def.List
	n, ok := LeadingInt(text)
	if !ok || !containsInt(st.FixIndexes, n) {
		opts := make([]string, 0, len(st.FixIndexes))
		for _, h := range st.FixIndexes {
			opts = append(opts, strconv.Itoa(h))
		}
		return Outcome{Ask: fmt.Sprintf(l.FixPickRePrompt, strings.Join(opts, " 或 "))}
	}
	if st.FixQty == "" {
		st.FixIndex = n
		st.FixIndexes = nil
		st.Phase = PhaseQty
		return Outcome{Ask: fmt.Sprintf(l.FixQtyPrompt, n, st.Items[n-1].Value)}
	}
	return e.applyFix(st, n, st.FixQty)
}

// applyFix 套用修正：更新數量、清掉 fix 暫存、回 review 清單。
func (e *Engine) applyFix(st *State, idx int, qty string) Outcome {
	l := e.Def.List
	old := st.Items[idx-1].Qty
	if old == "" {
		old = "(未填)"
	}
	st.Items[idx-1].Qty = qty
	st.FixIndex, st.FixIndexes, st.FixQty = 0, nil, ""
	st.Phase = PhaseCollect
	return Outcome{Ask: fmt.Sprintf(l.FixDoneText, idx, st.Items[idx-1].Value, old, qty) + "\n" + e.reviewPrompt(st)}
}

// ---------- 共用組字 ----------

func (e *Engine) reviewPrompt(st *State) string {
	l := e.Def.List
	var b strings.Builder
	fmt.Fprintf(&b, l.ReviewHeader, len(st.Items))
	b.WriteString("\n")
	for i, it := range st.Items {
		qty := it.Qty
		if qty == "" {
			qty = "(未填數量)"
		}
		note := ""
		if strings.TrimSpace(it.Note) != "" {
			note = "　" + it.Note
		}
		fmt.Fprintf(&b, "%d. %s ×%s%s\n", i+1, it.Value, qty, note)
	}
	b.WriteString(l.ReviewTail)
	return b.String()
}

// buildPickAsk 組「選項反問」：問題#1. label=value#2. label=value…
// （# 格式與前端 floating candidates 協議一致。）
func buildPickAsk(question string, cands []Candidate) string {
	var b strings.Builder
	b.WriteString(question)
	for i, c := range cands {
		fmt.Fprintf(&b, "#%d. %s=%s", i+1, c.Label, c.Value)
	}
	return b.String()
}
