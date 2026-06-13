package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/builtin"
	"ui_console/orchestration/skill_flow"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
)

// dianliao_bom_flow.go — 「產出電料Bom」接通用流程引擎（skill_flow）的轉接層。
//
// Step 1 重構（見 REFACTOR_SKILL_FLOW.md）：原本寫死在這裡的多輪狀態機
// （機台 → 逐項電料 → 模糊比對 → 補數量 → 修正 → review → 輸出）已泛型化搬入
// orchestration/skill_flow。本檔只剩：
//   1. 路由起點／LLM 路由前攔截 → 引擎 Handle → Outcome 轉 CLIResponse。
//   2. on_complete：executeDianliaoBom 產出 xlsx。
//   3. 既有測試沿用的純函式（多數已改為轉呼叫 skill_flow）。
// FlowDef 宣告與 Resolver 轉接在 dianliao_flow_def.go；模糊比對在 dianliao_bom_match.go。

const dianliaoBomSkillName = "產出電料Bom"

const dianliaoBomTTL = 30 * time.Minute

// dianliaoFlowStore：sessionID → 進行中流程狀態（TTL 由 skill_flow.Store 管理）。
var dianliaoFlowStore = skill_flow.NewStore(dianliaoBomTTL)

// isDianliaoBomTarget 判斷路由目標是不是「產出電料Bom」這個 skill。
func isDianliaoBomTarget(target string) bool {
	return normalizeGoProgramLookup(target) == normalizeGoProgramLookup(dianliaoBomSkillName)
}

func dianliaoAsk(q, traceID string) skill_step.CLIResponse {
	return skill_step.CLIResponse{
		Text:      setQuestionFloatingCandidates(q, traceID),
		Action:    "提問",
		Target:    q,
		Next:      actionchain.StandbyNext,
		NeedsUser: true,
	}
}

// startDianliaoBomFlow 起始一輪互動收集：先問機台。
func (a *App) startDianliaoBomFlow(sessionID, traceID string) skill_step.CLIResponse {
	st := &skill_flow.State{}
	out := a.dianliaoEngine(sessionID, traceID).Start(st)
	dianliaoFlowStore.Put(sessionID, st)
	debugtrace.Record("go.dianliaoBom.start", traceID, map[string]interface{}{"session_id": sessionID})
	return dianliaoAsk(out.Ask, traceID)
}

// maybeHandlePendingDianliaoBom 在 LLM 路由前攔截「正在收集電料BOM」的回合。
func (a *App) maybeHandlePendingDianliaoBom(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	st, ok := dianliaoFlowStore.Get(sessionID)
	if !ok {
		return nil, false
	}
	out := a.dianliaoEngine(sessionID, traceID).Handle(st, userText)
	switch {
	case out.Cancelled:
		dianliaoFlowStore.Delete(sessionID)
		clearConfirmQuestion(sessionID)
		return &skill_step.CLIResponse{Text: out.Text}, true
	case out.Complete:
		items := make([]builtin.BOMItem, 0, len(st.Items))
		for _, it := range st.Items {
			items = append(items, builtin.BOMItem{PartNo: it.Value, Qty: it.Qty, Note: it.Note})
		}
		resp := a.executeDianliaoBom(traceID, st.Values["machine"], items)
		dianliaoFlowStore.Delete(sessionID)
		return ptrResp(resp), true
	default:
		dianliaoFlowStore.Put(sessionID, st) // 保存進度並刷新 TTL
		return ptrResp(dianliaoAsk(out.Ask, traceID)), true
	}
}

// executeDianliaoBom 產出 xlsx 到 outputs，再用 surfaceSkillOutput 讓它出現在右側。
func (a *App) executeDianliaoBom(traceID, machine string, items []builtin.BOMItem) skill_step.CLIResponse {
	dbPath := a.findDianliaoReferencePath("編碼紀錄")
	if dbPath == "" {
		return skill_step.CLIResponse{
			Text:   "找不到「電料編碼紀錄」檔，請先把它載入（拖入）再回覆「輸出」。",
			Action: "流程", Target: dianliaoBomSkillName, NeedsUser: true,
		}
	}
	now := time.Now()
	req := buildDianliaoRequest(machine, items, now)
	outputDir := skillOutputsDir()
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return skill_step.CLIResponse{
			Error: err.Error(), Text: "建立輸出資料夾失敗：" + err.Error(),
			Action: "流程", Target: dianliaoBomSkillName,
		}
	}
	destPath := filepath.Join(outputDir, dianliaoOutputFileName(machine, now))

	res, err := builtin.BuildDianliaoBOM(req, dbPath, destPath)
	if err != nil {
		debugtrace.Record("go.dianliaoBom.execute_error", traceID, map[string]interface{}{"error": err.Error()})
		return skill_step.CLIResponse{
			Error: err.Error(), Text: "產出電料BOM失敗：" + err.Error(),
			Action: "流程", Target: dianliaoBomSkillName,
		}
	}
	shownName := filepath.Base(res.OutputPath)
	ref, derr := a.surfaceSkillOutput(res.OutputPath, dianliaoBomSkillName)
	if derr != nil {
		debugtrace.Record("go.dianliaoBom.surface_warn", traceID, map[string]interface{}{"error": derr.Error()})
	} else if strings.TrimSpace(ref.Name) != "" {
		shownName = ref.Name
	}
	debugtrace.Record("go.dianliaoBom.execute_ok", traceID, map[string]interface{}{
		"output": res.OutputPath, "surfaced": shownName, "rows": res.RowCount, "warnings": len(res.Warnings),
	})

	var b strings.Builder
	fmt.Fprintf(&b, "已產出電料BOM：%s（%d 列，機台 %s）。已加入右側「引用文件」。", shownName, res.RowCount, machine)
	if len(res.Warnings) > 0 {
		b.WriteString("\n提醒：" + strings.Join(res.Warnings, "；"))
	}
	return skill_step.CLIResponse{
		Text: b.String(), Action: "流程", Target: dianliaoBomSkillName,
		Next: actionchain.NormalizeNext("輸出"),
	}
}

// findDianliaoReferencePath 從近期已載入引用檔中，挑名稱含 keyword 的檔案路徑。
func (a *App) findDianliaoReferencePath(keyword string) string {
	for _, r := range a.recentReferenceFilesForRouting(12) {
		if strings.Contains(r.Name, keyword) {
			return r.Path
		}
	}
	return ""
}

// ---------- 純函式（可單元測試；多數轉呼叫 skill_flow 的泛型版）----------

func ptrResp(r skill_step.CLIResponse) *skill_step.CLIResponse { return &r }

func isDianliaoDoneText(text string) bool {
	return skill_flow.MatchWord(text, dianliaoDoneWords)
}

func isDianliaoMoreText(text string) bool {
	return skill_flow.MatchWord(text, dianliaoMoreWords)
}

func isDianliaoCancelText(text string) bool {
	return skill_flow.MatchWord(text, dianliaoCancelWords)
}

// normalizeMachineInput 連續剝掉「機台/名稱/是/為/：」等前綴與分隔，回乾淨的機台名稱。
func normalizeMachineInput(text string) string {
	s := strings.TrimSpace(text)
	prefixes := []string{"機台名稱", "機台", "名稱", "叫做", "叫", "是", "為", "：", ":", "　", " "}
	for {
		trimmed := false
		for _, p := range prefixes {
			if s != p && strings.HasPrefix(s, p) {
				s = strings.TrimSpace(strings.TrimPrefix(s, p))
				trimmed = true
				break
			}
		}
		if !trimmed {
			break
		}
	}
	return strings.TrimSpace(s)
}

// splitItemInput 從一行電料輸入切出「模糊查詢字串」與「數量」。
func splitItemInput(text string) (query, qty string) {
	return skill_flow.SplitItemInput(text)
}

// parseQtyInput 從文字抽出數量（取第一段連續數字，允許小數）。
func parseQtyInput(text string) (string, bool) {
	return skill_flow.ParseQty(text)
}

// parseBOMItemInput 保留：解析「料號 數量 [註解]」（供既有測試與單純情境）。
func parseBOMItemInput(text string) (builtin.BOMItem, bool) {
	norm := strings.NewReplacer("，", " ", ",", " ", "、", " ", "\t", " ", "　", " ").Replace(text)
	fields := strings.Fields(norm)
	if len(fields) == 0 {
		return builtin.BOMItem{}, false
	}
	item := builtin.BOMItem{PartNo: fields[0]}
	var noteParts []string
	for _, tok := range fields[1:] {
		if item.Qty == "" && isNumericToken(tok) {
			item.Qty = tok
			continue
		}
		noteParts = append(noteParts, tok)
	}
	item.Note = strings.Join(noteParts, " ")
	if strings.TrimSpace(item.PartNo) == "" {
		return builtin.BOMItem{}, false
	}
	return item, true
}

func isNumericToken(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// matchDianliaoPick 把使用者的選擇對應到候選：支援編號、料號、品名。
func matchDianliaoPick(text string, cands []builtin.DianliaoRecord) (builtin.DianliaoRecord, bool) {
	idx, ok := skill_flow.MatchPick(text, dianliaoToCandidates(cands))
	if !ok {
		return builtin.DianliaoRecord{}, false
	}
	return cands[idx], true
}

// leadingInt 取字串開頭的連續數字。
func leadingInt(s string) (int, bool) {
	return skill_flow.LeadingInt(s)
}

func dianliaoCandLabel(r builtin.DianliaoRecord) string {
	name := strings.TrimSpace(r.Name)
	pn := r.BestPartNo()
	switch {
	case name != "" && pn != "":
		return name + "（" + pn + "）"
	case name != "":
		return name
	default:
		return pn
	}
}

func dianliaoToCandidates(recs []builtin.DianliaoRecord) []skill_flow.Candidate {
	out := make([]skill_flow.Candidate, len(recs))
	for i, r := range recs {
		out[i] = skill_flow.Candidate{
			Value: r.BestPartNo(),
			Label: dianliaoCandLabel(r),
			Name:  strings.TrimSpace(r.Name),
		}
	}
	return out
}

func buildDianliaoRequest(machine string, items []builtin.BOMItem, now time.Time) builtin.BOMRequest {
	sheetName := strings.TrimSpace(machine)
	if sheetName == "" {
		sheetName = "電控BOM"
	}
	return builtin.BOMRequest{
		Machine: strings.TrimSpace(machine),
		Date:    now.Format("2006/01/02"),
		Title:   strings.TrimSpace(machine) + " 電控BOM",
		Sheets:  []builtin.BOMSheet{{Name: sheetName, Items: items}},
	}
}

func dianliaoOutputFileName(machine string, now time.Time) string {
	safe := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ' ':
			return '-'
		}
		return r
	}, strings.TrimSpace(machine))
	if safe == "" {
		safe = "未命名"
	}
	return fmt.Sprintf("電料BOM_%s_%s.xlsx", safe, now.Format("20060102-150405"))
}
