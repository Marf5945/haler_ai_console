// routing_validator.go
//
// 收尾版 Step 1：集中式「確定性硬閘」（Validator/Gate）。
//
// 撞點 3 裁定——不做語意重寫：網路/操作升格、引用文件覆寫等語意修正留在
// normalizeToolRoutingDecision；validator 只守硬邊界。完整 pipeline 為：
//
//	parse → normalize(語意) → validate(硬閘) → execute
//
// 撞點 4 裁定——降級友善：SkillID 白名單與 candidate 比對所需的 context 由
// Step 4/5 才接上；routingValidationContext 為空時，這些分支 pass-through，
// 不影響 Step 1~3 單獨上線。
//
// Step 1 守備範圍（本檔已完成）：
//   - action / next 只接受白名單，未知 action → 退回 need_tool（不臆測）
//   - target 截到單行、長度上限 routingTargetMaxRunes、拒 JSON/工具呼叫語法
//   - 網路/搜尋 query 送出前剝除 [已補充背景]…[/已補充背景] 框架標記
//     （撞點 1/2 收尾：再也不讓對話背景的框架文字混進出境 query）
//   - canonicalMissingSlot：把自由字串的缺漏槽位收斂成封閉 enum，供 Step 3
//     的「同一 slot 重問即停」能真正比對得上（否則「地點/地區/城市」會漏接）
package main

import (
	"strings"

	"ui_console/shared/actionchain"
)

// routingTargetMaxRunes 是單一 action-chain target 的長度上限（rune）。
const routingTargetMaxRunes = 240

// routingValidationContext 攜帶硬閘判斷所需的確定性資料。
// Step 1 全部留空亦可正常運作（降級友善）；Step 4/5 再把候選/目錄接上。
type routingValidationContext struct {
	CandidateSkillIDs map[string]struct{} // 本輪能力初篩候選（Step 4 注入）
	ArchivedSkillIDs  map[string]struct{} // ListArchived()/builtin 全集（Step 5 注入）
	AlreadyRepicked   bool                // 已重篩過一次，避免無限重篩（Step 5）
}

// routingValidationVerdict 區分硬閘裁決。
type routingValidationVerdict int

const (
	routingValidationPass   routingValidationVerdict = iota // 通過，可執行
	routingValidationRepick                                 // SkillID 不在候選 → 觸發一次能力再初篩（Step 5）
)

// whitelistedRoutingActions 是 judge 允許輸出的 action（已 NormalizeAction）。
var whitelistedRoutingActions = map[string]struct{}{
	"聊天": {}, "操作": {}, "程式": {}, "流程": {},
	"查詢": {}, "搜尋": {}, "網路": {}, "提問": {},
}

// whitelistedRoutingNext 是允許的 next 槽位。
var whitelistedRoutingNext = map[string]struct{}{
	actionchain.StandbyNext:  {}, // 待命
	actionchain.QuestionNext: {}, // 提問
	"輸出":                     {}, "操作": {}, "文件": {},
}

func isWhitelistedRoutingAction(action string) bool {
	_, ok := whitelistedRoutingActions[strings.TrimSpace(action)]
	return ok
}

// validateRoutingDecision 是 Step 1 的硬閘入口。回傳清理後的 decision 與裁決。
// 只處理 action 類決策；聊天/need_tool 直接放行（語意已在上游決定）。
func (a *App) validateRoutingDecision(decision toolRoutingDecision, vctx routingValidationContext) (toolRoutingDecision, routingValidationVerdict) {
	if decision.Kind != toolRoutingDecisionAction {
		return decision, routingValidationPass
	}
	action := actionchain.NormalizeAction(strings.TrimSpace(decision.Action))
	decision.Action = action

	// 1) action 白名單：未知 action → 退回 need_tool 交回上層，不臆測一個工具。
	if !isWhitelistedRoutingAction(action) {
		return toolRoutingDecision{Kind: toolRoutingDecisionNeedTool, Raw: decision.Raw}, routingValidationPass
	}

	// 2) target 硬邊界：剝框架標記 → 收單行 → 拒結構化語法 → 長度上限。
	cleanTarget := sanitizeRoutingTarget(decision.Target)
	if cleanTarget == "" {
		// 清理後為空，無法安全執行 → 退回提問，請使用者補一句。
		return toolRoutingDecision{
			Kind:   toolRoutingDecisionAction,
			Action: "提問",
			Target: "可以再說清楚一點你想做什麼嗎？",
			Next:   actionchain.StandbyNext,
			Raw:    decision.Raw,
		}, routingValidationPass
	}
	decision.Target = cleanTarget
	if action == "流程" {
		target, title, summary := parseSchedulerRoutingTargetMetadata(decision.Target)
		decision.Target = target
		if strings.TrimSpace(decision.Title) == "" {
			decision.Title = title
		}
		if strings.TrimSpace(decision.Summary) == "" {
			decision.Summary = summary
		}
	}

	// 3) next 白名單：非法/空 next 收斂成該 action 的預設 next。
	decision.Next = normalizeRoutingNext(action, decision.Next)

	// 4) SkillID 白名單（降級友善：候選/目錄為空 → pass-through，Step 5 啟用）。
	if action == "流程" {
		if a.validateSkillIDWhitelist(&decision, vctx) == routingValidationRepick {
			return decision, routingValidationRepick
		}
	}
	return decision, routingValidationPass
}

func parseSchedulerRoutingTargetMetadata(target string) (string, string, string) {
	raw := strings.TrimSpace(target)
	if raw == "" {
		return "", "", ""
	}
	parts := strings.Split(raw, ";")
	skillID := strings.TrimSpace(parts[0])
	var title string
	var summary string
	for _, part := range parts[1:] {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = sanitizeRoutingMetadataValue(value, 120)
		switch key {
		case "title", "標題", "name", "名稱":
			title = value
		case "summary", "摘要", "description", "說明":
			summary = value
		}
	}
	return skillID, title, summary
}

func sanitizeRoutingMetadataValue(value string, maxRunes int) string {
	s := firstNonEmptyRoutingLine(stripBackgroundFraming(value))
	s = strings.Trim(s, ` "'「」『』`)
	if looksLikeStructuredPayload(s) {
		return ""
	}
	if r := []rune(s); maxRunes > 0 && len(r) > maxRunes {
		s = strings.TrimSpace(string(r[:maxRunes]))
	}
	return s
}

// sanitizeRoutingTarget 把 judge 給的 target 清成可安全出境/路由的單行字串：
//  1. 剝除 [已補充背景]…[/已補充背景] 框架標記（撞點 1/2）
//  2. 取第一條非空白行（拒多行）
//  3. 偵測 JSON / 工具呼叫語法 → 視為無效（回空字串，交由上層提問）
//  4. 截斷到 routingTargetMaxRunes
func sanitizeRoutingTarget(target string) string {
	s := stripBackgroundFraming(target)
	s = firstNonEmptyRoutingLine(s)
	if s == "" {
		return ""
	}
	if looksLikeStructuredPayload(s) {
		return ""
	}
	if r := []rune(s); len(r) > routingTargetMaxRunes {
		s = strings.TrimSpace(string(r[:routingTargetMaxRunes]))
	}
	return s
}

// stripBackgroundFraming 移除 formatToolBackgroundContext 產生的框架區塊，
// 連同標記之間的內容一起拿掉；背景只該進 judge 推理 context，不進出境 query。
func stripBackgroundFraming(s string) string {
	const open = "[已補充背景]"
	const close = "[/已補充背景]"
	for {
		i := strings.Index(s, open)
		if i < 0 {
			break
		}
		j := strings.Index(s[i:], close)
		if j < 0 {
			// 只有開標記沒有閉標記：從開標記處截掉其後全部，保守處理。
			s = s[:i]
			break
		}
		s = s[:i] + s[i+j+len(close):]
	}
	// 殘留的單邊標記也一併清掉。
	s = strings.ReplaceAll(s, open, "")
	s = strings.ReplaceAll(s, close, "")
	return strings.TrimSpace(s)
}

func firstNonEmptyRoutingLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// looksLikeStructuredPayload 偵測 JSON 物件/陣列、程式碼圍欄、工具呼叫語法，
// 這些都不該出現在一個搜尋關鍵字或候選名稱裡。
func looksLikeStructuredPayload(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return false
	}
	if (strings.HasPrefix(t, "{") && strings.Contains(t, "\"")) ||
		(strings.HasPrefix(t, "[") && strings.Contains(t, "\"")) {
		return true
	}
	lower := strings.ToLower(t)
	return strings.Contains(t, "```") ||
		strings.Contains(lower, "function_call") ||
		strings.Contains(lower, "tool_call") ||
		strings.Contains(lower, "<tool") ||
		strings.Contains(lower, "</tool")
}

// defaultRoutingNext 是各 action 在 next 缺漏/非法時的安全預設。
var defaultRoutingNext = map[string]string{
	"網路": actionchain.StandbyNext,
	"操作": actionchain.StandbyNext,
	"提問": actionchain.StandbyNext,
	"搜尋": "文件",
	"查詢": "操作",
	"程式": "輸出",
	"流程": "輸出",
}

func normalizeRoutingNext(action, next string) string {
	n := actionchain.NormalizeNext(strings.TrimSpace(next))
	if _, ok := whitelistedRoutingNext[n]; ok {
		return n
	}
	if def, ok := defaultRoutingNext[action]; ok {
		return def
	}
	return actionchain.StandbyNext
}

// validateSkillIDWhitelist 在 action=流程 時驗證 target(SkillID)。Step 5：
//
//	合法＝在本輪候選 ∩ 在目錄 ∩ lifecycle 可執行。
//	在目錄且可執行但不在本輪候選（初篩 recall miss）＝回 Repick，由
//	  validateRoutingDecisionWithRepick 放寬候選重驗一次。
//	幻覺 SkillID / 不可執行 / 重篩後仍不在候選 ＝ 絕不執行，降級為提問。
//
// 降級友善：候選與目錄都空（Step 1~4 單獨上線）→ pass-through。
func (a *App) validateSkillIDWhitelist(decision *toolRoutingDecision, vctx routingValidationContext) routingValidationVerdict {
	if len(vctx.CandidateSkillIDs) == 0 && len(vctx.ArchivedSkillIDs) == 0 {
		return routingValidationPass // 降級：context 未接上
	}
	skillID := strings.TrimSpace(decision.Target)
	_, inCatalog := vctx.ArchivedSkillIDs[skillID]
	_, inCandidate := vctx.CandidateSkillIDs[skillID]
	executable := a != nil && a.skillRouter != nil && a.skillRouter.SkillExecutable(skillID)

	if inCandidate && inCatalog && executable {
		return routingValidationPass
	}
	if inCatalog && executable && !vctx.AlreadyRepicked {
		return routingValidationRepick // recall miss → 放寬重驗一次
	}
	// 安全背刺：非目錄 / 不可執行 / 重篩後仍不在候選 → 不執行，問清楚。
	*decision = toolRoutingDecision{
		Kind:   toolRoutingDecisionAction,
		Action: "提問",
		Target: "我不太確定要用哪個功能，可以再說一下你想做什麼嗎？",
		Next:   actionchain.StandbyNext,
		Raw:    decision.Raw,
	}
	return routingValidationPass
}

// validateRoutingDecisionWithRepick 包住 validateRoutingDecision，處理 Step 5 的
// 「candidate miss 最多重篩一次」：第一次 Repick 後，把候選放寬成「目錄可執行全集」
// 並標記 AlreadyRepicked 重驗一次；仍不合法就降級提問。呼叫端只需這一個入口。
func (a *App) validateRoutingDecisionWithRepick(decision toolRoutingDecision, lookup toolRoutingLookupContext) toolRoutingDecision {
	out, verdict := a.validateRoutingDecision(decision, a.routingValidationContextForTurn(lookup))
	if verdict != routingValidationRepick {
		return out
	}
	vctx2 := a.routingValidationContextForTurn(lookup)
	vctx2.AlreadyRepicked = true
	if vctx2.CandidateSkillIDs == nil {
		vctx2.CandidateSkillIDs = map[string]struct{}{}
	}
	for id := range vctx2.ArchivedSkillIDs {
		vctx2.CandidateSkillIDs[id] = struct{}{}
	}
	out, _ = a.validateRoutingDecision(out, vctx2)
	return out
}

// canonicalMissingSlot 把自由文字的缺漏槽位收斂成封閉 enum。
//
// 補充建議 1：Step 3 的「同一 slot 重問即停」必須比對得上，但二次 judge 會用
// 自然語言自由生成缺漏描述（「地點/地區/城市/你在哪」）。若直接用原字串比對，
// 去重會靜默失效、上限形同虛設。一律先過此函式正規化後再存入 AskedSlots。
func canonicalMissingSlot(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "other"
	}
	switch {
	case containsAny(s, []string{"location", "city", "where"}) ||
		containsAny(raw, []string{"地點", "地區", "城市", "哪裡", "哪個地方", "縣市"}):
		return "location"
	case containsAny(s, []string{"date", "birth", "when", "time"}) ||
		containsAny(raw, []string{"日期", "生日", "出生", "時間", "幾號", "哪一天"}):
		return "date"
	case containsAny(s, []string{"zodiac", "horoscope", "sign"}) ||
		containsAny(raw, []string{"星座", "上升", "太陽星座", "月亮星座"}):
		return "zodiac"
	case containsAny(s, []string{"scope", "category", "range"}) ||
		containsAny(raw, []string{"範圍", "類別", "分類", "哪一類", "種類"}):
		return "scope"
	case containsAny(s, []string{"file", "folder", "path"}) ||
		containsAny(raw, []string{"檔案", "資料夾", "路徑", "哪個檔"}):
		return "file"
	case containsAny(s, []string{"operation", "replay", "record"}) ||
		containsAny(raw, []string{"操作", "回放", "重現", "錄製"}):
		return "operation"
	default:
		return "other"
	}
}

// routingValidationContextForTurn 組裝本輪硬閘 context。
// Step 1：回空 context（降級友善，硬閘只跑 action/next/target 邊界）。
// Step 4/5 會在此把「能力初篩候選」與「skill 目錄全集」接上，啟用 SkillID 白名單。
func (a *App) routingValidationContextForTurn(lookup toolRoutingLookupContext) routingValidationContext {
	vctx := routingValidationContext{}
	if a != nil && a.skillRouter != nil {
		// Step 5：本輪候選 + 目錄全集，啟用 SkillID 白名單。
		vctx.CandidateSkillIDs = a.routingCandidateSkillIDs(lookup.Terms)
		vctx.ArchivedSkillIDs = a.skillRouter.ArchivedAndBuiltinSkillIDs()
	}
	return vctx
}
