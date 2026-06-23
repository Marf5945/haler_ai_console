package replan

import (
	"context"
	"time"

	"ui_console/audit_log"
	"ui_console/domain/risk"
	"ui_console/orchestration/dag"
)

// DefaultProposeTimeout 是 Proposer LLM 呼叫的預設逾時；逾時一律 fail-safe 進 review。
const DefaultProposeTimeout = 20 * time.Second

// ProposerContext 是交給 LLM Proposer 的輸入。只給「同目標換路」所需脈絡，
// 不暴露 raw 工具輸出、完整推理或敏感內容。
type ProposerContext struct {
	Contract         dag.GoalContract // Go 持有的目標契約（模型只能據此提同目標方案）
	Failure          FailureCategory  // 結構化失敗類型
	CompletedSummary []string         // 已完成節點的摘要（非 raw 輸出）
	OldTail          []dag.DAGNode    // 尚未開始的後續節點
	EligibleActions  []string         // v1 read-only allowlist，避免模型提廢案
}

// Proposer 是 LLM 提案介面。實作（接 planner adapter）放在 main，
// 讓 Coordinator 迴路本身可用假 Proposer 完整單測，且不依賴任何外部框架。
type Proposer interface {
	Propose(ctx context.Context, pc ProposerContext) (ReplanProposal, error)
}

// AttemptResult 是一次 replan 嘗試的結果。
type AttemptResult struct {
	Decision      Decision
	Stage         Stage
	Reason        string
	ClassifiedMax risk.RiskClass
	Applied       bool // 是否真的套用了 tail patch（僅 silent 成功時為 true）
}

// Coordinator 串起一次 replan：提案 → Gate → （silent 才）CAS 套用 → audit。
// 它是 replan 的唯一對外編排入口；DAG 仍是 DAG、review 仍是閘門。
type Coordinator struct {
	Proposer Proposer
	Counter  *Counter
	Audit    *audit_log.AppendLog[ReplanAuditEntry]
	Timeout  time.Duration
	Critic   Critic // 可選；nil 時不啟用。只在 borderline 觸發，只能把 silent 收緊成 review。
}

// NewCoordinator 建立 Coordinator，timeout 為 0 時用預設值。
func NewCoordinator(p Proposer, c *Counter, audit *audit_log.AppendLog[ReplanAuditEntry], timeout time.Duration) *Coordinator {
	if timeout <= 0 {
		timeout = DefaultProposeTimeout
	}
	return &Coordinator{Proposer: p, Counter: c, Audit: audit, Timeout: timeout}
}

// Attempt 執行一次 replan 嘗試並回傳裁決結果。
// 任何錯誤 / 逾時 / CAS 衝突一律 fail-safe：不套用、進 review，並落 audit。
func (co *Coordinator) Attempt(run *dag.DAGRun, failure FailureCategory, triggerNodeRisk risk.RiskClass) AttemptResult {
	contract := contractOf(run)
	oldTail := PlannedTail(run)

	pc := ProposerContext{
		Contract:         contract,
		Failure:          failure,
		CompletedSummary: completedSummaries(run),
		OldTail:          oldTail,
		EligibleActions:  EligibleReplanActions(),
	}

	// 1) 取提案（帶逾時）。
	ctx, cancel := context.WithTimeout(context.Background(), co.Timeout)
	defer cancel()
	proposal, err := co.Proposer.Propose(ctx, pc)
	if err != nil {
		return co.failSafe(run, contract, failure, "proposer error: "+err.Error())
	}

	// 2) 震盪偵測（純讀）。
	sig := TailSignature(proposal.ProposedTail)
	oscillating := co.Counter != nil && co.Counter.IsOscillating(sig)

	// 3) Gate 裁決。
	gate := Gate(GateInput{
		Contract:        contract,
		Proposal:        proposal,
		Run:             run,
		TriggerNodeRisk: triggerNodeRisk,
		Failure:         failure,
		Counter:         co.Counter,
		Oscillating:     oscillating,
	})

	oldHash := ComputeTailHash(oldTail)

	// 3.5) Critic（可選，只在 borderline 觸發）：silent + borderline 才問，且只能收緊。
	if gate.Decision == DecisionSilent && co.Critic != nil && isBorderline(co.Counter, oscillating) {
		cctx, ccancel := context.WithTimeout(context.Background(), co.Timeout)
		verdict, cerr := co.Critic.Review(cctx, pc, proposal)
		ccancel()
		if cerr != nil || verdict.Concern {
			reason := "critic concern: " + verdict.Note
			if cerr != nil {
				reason = "critic error: " + cerr.Error()
			}
			gate = GateResult{Decision: DecisionReview, Stage: StageSilentNotice, Reason: reason, ClassifiedMax: gate.ClassifiedMax}
		}
	}

	// 4) 非 silent：直接落 audit 後回（不消耗計數）。
	if gate.Decision != DecisionSilent {
		co.writeAudit(run, contract, failure, gate, proposal, oldHash, "", false)
		return AttemptResult{Decision: gate.Decision, Stage: gate.Stage, Reason: gate.Reason, ClassifiedMax: gate.ClassifiedMax}
	}

	// 5) silent：建新節點 + CAS 原子套用。
	newNodes := BuildNewNodes(proposal, run.Revision+1)
	patch := TailPatch{
		ExpectedRevision:     run.Revision,
		ExpectedActiveNodeID: run.ActiveNodeID,
		ExpectedOldTailHash:  oldHash,
		NewNodes:             newNodes,
	}
	if applyErr := ApplyTailPatch(run, patch); applyErr != nil {
		// CAS 衝突（期間被 cancel / 節點已跑掉 / review 介入）→ fail-safe。
		return co.failSafe(run, contract, failure, "apply conflict: "+applyErr.Error())
	}

	// 6) 套用成功：計數 + audit（silent）。
	if co.Counter != nil {
		co.Counter.RecordReplan(sig)
	}
	newHash := ComputeTailHash(newNodes)
	co.writeAudit(run, contract, failure, gate, proposal, oldHash, newHash, true)
	return AttemptResult{
		Decision:      DecisionSilent,
		Stage:         gate.Stage,
		Reason:        gate.Reason,
		ClassifiedMax: gate.ClassifiedMax,
		Applied:       true,
	}
}

// failSafe 是所有「不可套用」狀況的統一出口：進 review、落 audit、不套用。
func (co *Coordinator) failSafe(run *dag.DAGRun, contract dag.GoalContract, failure FailureCategory, reason string) AttemptResult {
	gate := GateResult{Decision: DecisionReview, Stage: StageSilentNotice, Reason: reason}
	co.writeAudit(run, contract, failure, gate, ReplanProposal{Reason: reason}, ComputeTailHash(PlannedTail(run)), "", false)
	return AttemptResult{Decision: DecisionReview, Stage: StageSilentNotice, Reason: reason}
}

// writeAudit 落一筆稽核（含 silent）。Audit 為 nil 時略過（測試方便）。
func (co *Coordinator) writeAudit(run *dag.DAGRun, contract dag.GoalContract, failure FailureCategory, gate GateResult, proposal ReplanProposal, oldHash, newHash string, silent bool) {
	if co.Audit == nil {
		return
	}
	consec, total := 0, 0
	if co.Counter != nil {
		consec, total = co.Counter.ConsecutiveNoProgress, co.Counter.RunTotal
	}
	_ = AppendAuditEntry(co.Audit, ReplanAuditEntry{
		RunID:                 run.ID,
		TriggerReason:         firstNonEmpty(proposal.Reason, string(failure)),
		Failure:               failure,
		Decision:              gate.Decision,
		OldTailHash:           oldHash,
		NewTailHash:           newHash,
		ClassifiedMaxRisk:     string(gate.ClassifiedMax),
		Silent:                silent,
		ScopeReplan:           gate.ScopeReplan,
		ConsecutiveNoProgress: consec,
		RunTotal:              total,
		GoalHash:              contract.Hash(),
	})
}

// contractOf 從 run 取出 GoalContract 值（nil → 零值 legacy，Gate 會保守進 review）。
func contractOf(run *dag.DAGRun) dag.GoalContract {
	if run != nil && run.GoalContract != nil {
		return *run.GoalContract
	}
	return dag.GoalContract{}
}

// completedSummaries 收集已成功節點的非空摘要（非 raw 輸出）。
func completedSummaries(run *dag.DAGRun) []string {
	var out []string
	for _, n := range run.Nodes {
		if n.Status == dag.StatusSucceeded && n.ResultSummary != "" {
			out = append(out, n.ResultSummary)
		}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// Critic 是可選的第二意見（同模型或小模型）。只能讓 Coordinator 更保守，
// 不能放寬。預設不啟用（Coordinator.Critic == nil），避免在熱路徑多一次 LLM 呼叫。
type Critic interface {
	Review(ctx context.Context, pc ProposerContext, proposal ReplanProposal) (CriticVerdict, error)
}

// CriticVerdict 是 Critic 的判斷；Concern=true → Coordinator 改走 review。
type CriticVerdict struct {
	Concern bool
	Note    string
}

// isBorderline 以 Go 可算的訊號判定是否為「邊界情況」（才值得問 Critic）：
// 震盪，或連續無進展已到第 3 次以上。不靠模型判斷。
func isBorderline(c *Counter, oscillating bool) bool {
	return oscillating || (c != nil && c.ConsecutiveNoProgress >= 3)
}
