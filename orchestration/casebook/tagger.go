// tagger.go — 分層判定 + 標籤指派。
//
// 兩層：
//  1. predicate（machine）：EvalPredicate 驗得了 → 直接定 verdict + 核心 tag。
//  2. model（fallback）：predicate 驗不了時，呼叫注入的 ModelJudge 自評。
//     ModelJudge 是介面，本包不綁任何模型，保持零依賴、可單測。
//
// tag 字典策略：核心 tag 由規則指派（可重現、可統計）；模型只能「額外補」
// free-text tag，補的 tag 不覆蓋核心判定。
package casebook

import (
	"errors"
	"strings"
)

// ModelJudge 是模型自評層的注入點。caller 用既有 critic / summary adapter 實作。
// 回傳 verdict 與「額外」free-text tags（核心 tag 仍由規則決定）。
// 實作不可 panic；無結論時回 (VerdictPartial, nil, nil)。
type ModelJudge interface {
	Judge(goal, expected, actual string) (verdict Verdict, extraTags []string, err error)
}

// Signals 是規則層需要的執行訊號（由 DAGNode 終局狀態映射而來）。
type Signals struct {
	NodeStatus      string // 對應 dag.NodeStatus："succeeded"/"failed"/"blocked"/"cancelled"...
	FailureCategory string // dag.DAGNode.FailureCategory（若有）
	ErrorText       string // dag.DAGNode.Error
	TimedOut        bool   // 輪數/預算耗盡（task_loop 可判定）
}

// Classify 走分層判定，回傳最終 verdict 與正規化後的 tags（核心在前）。
// judge 可為 nil（純規則模式）；規則定不了且 judge==nil 時回 partial/unknown。
func Classify(goal, expected, predicate string, in PredicateInput, sig Signals, judge ModelJudge) (Verdict, []string) {
	// ── 規則層：先看硬訊號，再看 predicate ──
	if v, tag, decided := classifyBySignals(sig); decided {
		return v, normalizeTags([]string{tag})
	}

	if pass, ok := EvalPredicate(predicate, in); ok {
		// 執行報錯時 fail-closed：不讓 predicate 把報錯節點升級成 pass。
		if pass && !in.HadError {
			return VerdictPass, normalizeTags([]string{TagSuccess})
		}
		// 有產出但不達標 / 報錯：predicate 不過 → wrong_output
		return VerdictFail, normalizeTags([]string{TagWrongOutput})
	}

	// predicate 驗不了但執行已報錯（且硬訊號未命中分類）→ 直接 fail，不浪費模型呼叫。
	if in.HadError {
		return VerdictFail, normalizeTags([]string{TagWrongOutput})
	}

	// ── 模型層：predicate 驗不了，退模型自評 ──
	if judge != nil {
		if v, extra, err := safeJudge(judge, goal, expected, in.Actual); err == nil {
			tags := []string{verdictCoreTag(v)}
			tags = append(tags, extra...) // 模型只補，不覆蓋核心
			return v, normalizeTags(tags)
		}
	}

	// fail-open：判不了不擋路，但留痕
	return VerdictPartial, normalizeTags([]string{TagUnknown})
}

// classifyBySignals 從硬執行訊號直接定案（最可靠，優先）。
func classifyBySignals(sig Signals) (Verdict, string, bool) {
	switch strings.ToLower(strings.TrimSpace(sig.NodeStatus)) {
	case "blocked", "waiting_review":
		return VerdictFail, TagBlocked, true
	case "cancelled":
		return VerdictFail, TagCancelled, true
	case "succeeded":
		// 成功狀態仍交給 predicate 做二次確認（避免「假完成」）；不在此定案。
	}
	if sig.TimedOut {
		return VerdictFail, TagTimeout, true
	}
	// FailureCategory / Error 文字命中已知失敗型態。
	if tag, ok := failureTextTag(sig.FailureCategory + " " + sig.ErrorText); ok {
		return VerdictFail, tag, true
	}
	return "", "", false
}

// failureTextTag 用關鍵字把錯誤文字映到核心 tag（規則優先、可擴充）。
func failureTextTag(text string) (string, bool) {
	t := strings.ToLower(text)
	switch {
	case t == "" || strings.TrimSpace(t) == "":
		return "", false
	case containsAny(t, "timeout", "deadline", "逾時", "timed out"):
		return TagTimeout, true
	case containsAny(t, "http 4", "http 5", "status 4", "status 5", "connection refused",
		"connection reset", "dns", "econnrefused", "api error", "rate limit", "429", "500", "502", "503"):
		return TagAPIError, true
	case containsAny(t, "missing", "not found", "no such", "缺", "找不到", "未產出"):
		return TagMissingStep, true
	}
	return "", false
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// verdictCoreTag：模型給 verdict 後對應的核心 tag。
func verdictCoreTag(v Verdict) string {
	switch v {
	case VerdictPass:
		return TagSuccess
	case VerdictFail:
		return TagWrongOutput // 模型判失敗但無更細訊號時的保守歸群
	default:
		return TagUnknown
	}
}

// safeJudge 包一層，吸收實作的 panic，保證 fail-open。
func safeJudge(judge ModelJudge, goal, expected, actual string) (v Verdict, extra []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			v, extra, err = VerdictPartial, nil, errJudgePanic
		}
	}()
	return judge.Judge(goal, expected, actual)
}

var errJudgePanic = errors.New("casebook: model judge panicked") // safeJudge fail-open sentinel
