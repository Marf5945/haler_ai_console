package main

import (
	"fmt"
	"strconv"
	"strings"

	"ui_console/adapter/debugtrace"
	"ui_console/builtin"
)

// dianliao_bom_match.go — 電料模糊解析：本地粗篩 → 模型精選 → fallback。
//
// 回傳 (resolved, status)：
//   status="no_db"：找不到電料編碼紀錄檔。
//   status="none" ：DB 有讀到但找不到相符。
//   status="ok"   ：resolved 至少一筆；len==1 表示可自動帶入，>1 表示要列選項。
func (a *App) resolveDianliaoItem(sessionID, traceID, query string) ([]builtin.DianliaoRecord, string) {
	dbPath := a.findDianliaoReferencePath("編碼紀錄")
	if dbPath == "" {
		return nil, "no_db"
	}
	records, err := builtin.LoadDianliaoRecords(dbPath)
	if err != nil {
		debugtrace.Record("go.dianliaoBom.db_load_error", traceID, map[string]interface{}{"error": err.Error()})
		return nil, "no_db"
	}
	scored := builtin.SearchDianliaoLocal(records, query, 8)
	if len(scored) == 0 {
		return nil, "none"
	}
	cands := make([]builtin.DianliaoRecord, len(scored))
	for i, s := range scored {
		cands[i] = s.Record
	}

	picks, ok := a.modelRankDianliao(sessionID, traceID, query, cands)
	if !ok {
		// 模型不可用（額度/逾時等）→ 退回純本地：top1 明顯領先就自動，否則列前三。
		if len(scored) == 1 || scored[0].Score > scored[1].Score {
			return []builtin.DianliaoRecord{scored[0].Record}, "ok"
		}
		return topRecords(cands, 3), "ok"
	}
	if len(picks) == 0 {
		return nil, "none"
	}
	var out []builtin.DianliaoRecord
	for _, p := range picks {
		if p >= 1 && p <= len(cands) {
			out = append(out, cands[p-1])
		}
		if len(out) >= 3 {
			break
		}
	}
	if len(out) == 0 {
		return nil, "none"
	}
	return out, "ok"
}

func topRecords(cands []builtin.DianliaoRecord, n int) []builtin.DianliaoRecord {
	if n > len(cands) {
		n = len(cands)
	}
	out := make([]builtin.DianliaoRecord, n)
	copy(out, cands[:n])
	return out
}

// modelRankDianliao 讓模型從候選中挑出最可能的編號。
// 回 (picks, ok)：ok=false 代表模型呼叫失敗（呼叫端應 fallback 本地）。
func (a *App) modelRankDianliao(sessionID, traceID, query string, cands []builtin.DianliaoRecord) ([]int, bool) {
	prompt := buildDianliaoRankPrompt(query, cands)
	out, err := a.callRawModel(a.defaultSkillExecutionAdapterID(), sessionID, prompt, dianliaoTrace(traceID))
	if err != nil {
		debugtrace.Record("go.dianliaoBom.rank_model_error", traceID, map[string]interface{}{"error": err.Error()})
		return nil, false
	}
	picks := parseModelPickNumbers(out, len(cands))
	debugtrace.Record("go.dianliaoBom.rank", traceID, map[string]interface{}{
		"query": query, "candidates": len(cands), "picks": picks,
	})
	return picks, true
}

func dianliaoTrace(traceID string) string { return traceID + ":dianliao-rank" }

func buildDianliaoRankPrompt(query string, cands []builtin.DianliaoRecord) string {
	var b strings.Builder
	b.WriteString("你是電料料號比對助手。使用者要找一項電料，請從候選清單挑出最可能符合的。\n")
	b.WriteString("使用者描述：" + strings.TrimSpace(query) + "\n")
	b.WriteString("候選（編號. 品名 | 廣達料號 | 規格）：\n")
	for i, c := range cands {
		fmt.Fprintf(&b, "%d. %s | %s | %s\n", i+1,
			oneLine(c.Name), oneLine(c.BestPartNo()), oneLine(c.Spec))
	}
	b.WriteString("只輸出最可能的編號，依可能性高到低用半形逗號分隔（例如 2,5）；完全沒有相符輸出 0。除了數字與逗號外不要任何文字。")
	return b.String()
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// parseModelPickNumbers 從模型輸出抽出 [1,max] 範圍內的編號，去重、保序。
// 只含 0 或沒有有效編號 → 回空 slice（代表「沒有相符」）。
func parseModelPickNumbers(out string, max int) []int {
	var nums []int
	seen := map[int]bool{}
	cur := strings.Builder{}
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		n, err := strconv.Atoi(cur.String())
		cur.Reset()
		if err != nil {
			return
		}
		if n >= 1 && n <= max && !seen[n] {
			seen[n] = true
			nums = append(nums, n)
		}
	}
	for _, r := range out {
		if r >= '0' && r <= '9' {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return nums
}
