// app_web_chain_eval.go — Web Chain drift 沉澱（2.5.5.11 Phase 2）。
// 把每輪「期望 vs 實際」不符的 drift 累積成一筆 skill_eval EventRecord，
// 寫進獨立 JSONL（localsearch 掃描範圍外，不會被當語料餵回 LLM）。
// web chain 現實優先、永不阻擋，故全部記 RiskLow / Blocked=false。
package main

import (
	"ui_console/orchestration/skill_eval"
	"ui_console/shared/actionchain"
)

// webChainEvalAppend 是寫入 sink，預設落 skill_eval store；測試可覆寫攔截。
var webChainEvalAppend = func(rec skill_eval.EventRecord) error {
	return skill_eval.NewStore(appDataRoot(), "default").AppendEvent(rec)
}

// buildWebChainDriftEvent 把一輪 action 不符組成 skill_eval.DriftEvent。
// expected* 為期望步（占位 target 也照填，比對端已知不嚴格比 target）；actual* 為實際決策。
func buildWebChainDriftEvent(expectedAction, expectedTarget, expectedNext, actualAction, actualTarget, actualNext, reason string) skill_eval.DriftEvent {
	return skill_eval.DriftEvent{
		Kind: skill_eval.DriftActionMismatch,
		Risk: skill_eval.RiskLow, // 現實優先：只記錄不阻擋
		Expected: skill_eval.EvalStep{
			Action: expectedAction,
			Target: expectedTarget,
			Next:   actionchain.NormalizeNext(expectedNext),
		},
		Actual: skill_eval.EvalStep{
			Action: actualAction,
			Target: actualTarget,
			Next:   actionchain.NormalizeNext(actualNext),
		},
		Reason:  reason,
		Blocked: false,
	}
}

// recordWebChainDrifts 一條鏈收尾時把累積的 drift 寫成單筆事件；無 drift 不落檔。
func recordWebChainDrifts(sessionID string, drifts []skill_eval.DriftEvent) {
	if len(drifts) == 0 {
		return
	}
	_ = webChainEvalAppend(skill_eval.EventRecord{
		SessionID: sessionID,
		Drifts:    drifts,
		Note:      "web_chain", // 來源標記，月報/查詢時可篩
	})
}

// ── Phase 3：每次鏈執行落一筆 run（升格分母 + 月報）──

// webChainRunAppend 是 run 寫入 sink，預設落 skill_eval store；測試可覆寫。
var webChainRunAppend = func(rec skill_eval.WebChainRun) error {
	return skill_eval.NewStore(appDataRoot(), "default").AppendWebChainRun(rec)
}

// recordWebChainRun 記錄一次鏈執行的簽章與 drift 數。單步（無續跑）不記，省雜訊。
func recordWebChainRun(sessionID string, executedActions []string, driftCount int) {
	if len(executedActions) < 2 {
		return // 沒真的續跑成多步，不是「鏈」
	}
	_ = webChainRunAppend(skill_eval.WebChainRun{
		SessionID:  sessionID,
		Signature:  skill_eval.WebChainSignature(executedActions),
		Steps:      len(executedActions),
		DriftCount: driftCount,
	})
}

// WebChainSkillCandidates 回傳目前達升格門檻的複合鏈簽章（drift 歸零且累積夠多乾淨跑）。
// 供前端/工程師查詢用：只回「候選訊號」，是否真的建 skill 交使用者確認（沿用 candidate 卡流程）。
func (a *App) WebChainSkillCandidates() ([]skill_eval.WebChainSigStats, error) {
	runs, err := skill_eval.NewStore(appDataRoot(), "default").LoadWebChainRuns()
	if err != nil {
		return nil, err
	}
	return skill_eval.WebChainSkillCandidates(skill_eval.SummarizeWebChainRuns(runs)), nil
}

// WebChainMonthlyStats 回傳所有複合鏈簽章的聚合統計，供月報納入 web_chain 區塊。
func (a *App) WebChainMonthlyStats() ([]skill_eval.WebChainSigStats, error) {
	runs, err := skill_eval.NewStore(appDataRoot(), "default").LoadWebChainRuns()
	if err != nil {
		return nil, err
	}
	return skill_eval.SummarizeWebChainRuns(runs), nil
}
