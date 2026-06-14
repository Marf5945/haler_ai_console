// scheduler_exec.go — 排程執行：把 Phase C 的故障切換順序真正接到技能執行（3.1.10 Phase E 後端骨幹）。
//
// 流程：scheduler 到點 → schedulerSkillExecutor.ExecuteSkill →
// 本檔依 adapter 登錄表排出備援順序（CLI → 本地 → API 墊底），逐一嘗試；
// 遇到 quota/限流/斷線/逾時等「可重試」失敗就換下一個後端，全部失敗才回報錯誤。
package main

import (
	"fmt"
	"strings"
	"time"

	"ui_console/adapter/adapter_registry"
)

// schedulerAttemptResult 為單一 adapter 的嘗試結果。
type schedulerAttemptResult struct {
	ok        bool  // 是否成功
	retriable bool  // 失敗是否可換下一個後端重試
	err       error // 失敗原因
}

// schedulerFailoverCandidates 從 adapter 登錄表建出排程執行備援候選。
// 順序沿用使用者在登錄表的排序（Order = 索引）；分類見 classifySchedulerAdapterKind。
func (a *App) schedulerFailoverCandidates() []failoverCandidate {
	if a == nil || a.adapterRegistry == nil {
		return nil
	}
	adapters := a.adapterRegistry.ListAvailable()
	out := make([]failoverCandidate, 0, len(adapters))
	for i, ad := range adapters {
		kind, ok := classifySchedulerAdapterKind(ad)
		if !ok {
			continue // 跳過 main / sub 等非模型後端
		}
		out = append(out, failoverCandidate{
			AdapterID: ad.ID,
			Kind:      kind,
			Order:     i,
			Available: ad.Status != adapter_registry.StatusOffline,
		})
	}
	return out
}

// classifySchedulerAdapterKind 把登錄表 adapter 分類為 failoverKind。
// 第二回傳值為 false 代表不納入排程後援（main / sub / 未知）。
func classifySchedulerAdapterKind(ad adapter_registry.Adapter) (failoverKind, bool) {
	switch strings.ToLower(strings.TrimSpace(ad.Kind)) {
	case "api":
		return failoverAPI, true
	case "cli":
		// 有 endpoint（本地模型 server）或 Ollama 執行檔 → 視為本地模型，優先於雲端 API。
		if strings.TrimSpace(ad.Endpoint) != "" || adapter_registry.IsOllamaExecutablePath(ad.Path) {
			return failoverLocal, true
		}
		return failoverCLI, true
	default:
		return failoverCLI, false
	}
}

// runSchedulerWithFailover 依備援順序逐一嘗試 attempt；retriable 失敗換下一個，
// 成功回傳該 adapterID；非可重試失敗立即停止；全部用盡回傳最後一個錯誤。
func runSchedulerWithFailover(cands []failoverCandidate, attempt func(adapterID string) schedulerAttemptResult) (string, error) {
	ordered := orderSchedulerFailover(cands)
	if len(ordered) == 0 {
		return "", fmt.Errorf("scheduler: 沒有可用的執行後端")
	}
	var lastErr error
	for _, c := range ordered {
		res := attempt(c.AdapterID)
		if res.ok {
			return c.AdapterID, nil
		}
		lastErr = res.err
		if !res.retriable {
			if res.err == nil {
				return "", fmt.Errorf("scheduler: 後端 %s 執行失敗", c.AdapterID)
			}
			return "", res.err // 非可重試（如技能邏輯/路由問題）→ 不再換後端
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("scheduler: 所有後端皆無法執行")
	}
	return "", lastErr
}

// runSchedulerSkillOnce 用單一 adapter 跑一次 skill，並把失敗分類成是否可換後端重試。
func (a *App) runSchedulerSkillOnce(adapterID, sessionID, actionTarget string) schedulerAttemptResult {
	decision, err := a.ExecuteSkillMessage(adapterID, sessionID, actionTarget, fmt.Sprintf("scheduler-skill-%d", time.Now().UnixNano()))
	if err != nil {
		return schedulerAttemptResult{ok: false, retriable: isSchedulerRetriableFailure(err.Error()), err: err}
	}
	if decision == nil || !decision.Executed {
		reason := "skill 未執行"
		if decision != nil && strings.TrimSpace(decision.Decision) != "" {
			reason = decision.Decision
		}
		return schedulerAttemptResult{
			ok:        false,
			retriable: isSchedulerRetriableFailure(reason),
			err:       fmt.Errorf("scheduler skill executor: skill %q 未執行 (decision=%s)", actionTarget, reason),
		}
	}
	return schedulerAttemptResult{ok: true}
}

// isSchedulerRetriableFailure 判斷失敗是否該換下一個後端：配額/限流/斷線/逾時。
func isSchedulerRetriableFailure(msg string) bool {
	if isQuotaExhaustedError(msg) {
		return true
	}
	low := strings.ToLower(msg)
	for _, k := range []string{
		"timeout", "timed out", "deadline exceeded",
		"connection", "disconnect", "econnrefused", "refused",
		"unavailable", "no capacity", "broken pipe", "eof",
	} {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}
