package replan

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"ui_console/domain/risk"
	"ui_console/orchestration/dag"
)

// CAS 前置條件不符時回傳的 sentinel 錯誤。
var (
	ErrRevisionConflict   = errors.New("replan: revision conflict")           // run 已被其他寫入推進
	ErrActiveNodeConflict = errors.New("replan: active node conflict")        // 當前節點已改變
	ErrTailHashConflict   = errors.New("replan: tail hash conflict")          // planned tail 已變動
	ErrTailNotReplaceable = errors.New("replan: tail contains started nodes") // tail 含已開始節點
)

// TailPatch 描述一次原子 tail 替換，附 compare-and-swap 前置條件。
// 缺任一條件，replan LLM 回來時可能撞上 cancel / 節點已跑掉 / review 介入。
type TailPatch struct {
	ExpectedRevision     int           // 期望的 run.Revision
	ExpectedActiveNodeID string        // 期望的 run.ActiveNodeID
	ExpectedOldTailHash  string        // 期望的舊 planned tail hash
	NewNodes             []dag.DAGNode // 版本化 id 的新 tail
}

// isTailStatus 判斷狀態是否屬於「可被 replan 替換的 tail」（尚未開始）。
func isTailStatus(s dag.NodeStatus) bool {
	return s == dag.StatusPlanned || s == dag.StatusReady
}

// PlannedTail 取出 run 中尚未開始的尾段節點（planned / ready）。
func PlannedTail(run *dag.DAGRun) []dag.DAGNode {
	var tail []dag.DAGNode
	for _, n := range run.Nodes {
		if isTailStatus(n.Status) {
			tail = append(tail, n)
		}
	}
	return tail
}

// ComputeTailHash 對既有 tail 節點算 hash，作為 CAS 的 old-tail 期望值。
// 納入 id 以偵測節點集合或順序的任何變動。
func ComputeTailHash(nodes []dag.DAGNode) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, n.ID+"|"+n.Action+"|"+n.Target+"|"+strings.Join(n.Dependencies, ","))
	}
	return sha256Hex(parts...)
}

// TailSignature 對「提案 tail」算 signature 供震盪偵測。
// 刻意不納入 id（新 tail id 每次不同），只看 action+target+deps 的語意內容，
// 並排序以忽略順序差異——A↔B 來回會得到相同 signature。
func TailSignature(nodes []ProposedNode) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		deps := append([]string(nil), n.Dependencies...)
		sort.Strings(deps)
		parts = append(parts, strings.ToLower(n.Action)+"|"+n.Target+"|"+strings.Join(deps, ","))
	}
	sort.Strings(parts)
	return sha256Hex(parts...)
}

// VersionedNodeID 產生版本化節點 id（r{rev}_node_{idx}），避免重用舊 id 造成
// trace / review / history 對不上。
func VersionedNodeID(revision, idx int) string {
	return fmt.Sprintf("r%d_node_%d", revision, idx)
}

// BuildNewNodes 把模型提案轉成 DAGNode：版本化 id、狀態 planned，
// 並用「確定性分類器」覆寫 RiskClass（不採信模型自報），ModelRiskClass 僅留存參考。
func BuildNewNodes(p ReplanProposal, newRevision int) []dag.DAGNode {
	nodes := make([]dag.DAGNode, 0, len(p.ProposedTail))
	for i, pn := range p.ProposedTail {
		cls := risk.ClassifyOperation(pn.Action, targetsOf(pn))
		execType, actionCode := executableFor(pn.Action)
		nodes = append(nodes, dag.DAGNode{
			ID:             VersionedNodeID(newRevision, i+1),
			Title:          pn.Title,
			Operation:      actionCode, // 對齊 TaskPlanToNodes（Operation=ActionCode）
			Action:         pn.Action,
			ActionCode:     actionCode, // 可執行 FS code，executeToolTaskNode 據此 dispatch
			ExecutorType:   execType,   // tool_call → dispatchFSAction
			Target:         pn.Target,
			RiskClass:      string(cls),       // Go 裁定
			ModelRiskClass: pn.ModelRiskClass, // 模型自報，僅參考
			Status:         dag.StatusPlanned,
			Dependencies:   pn.Dependencies,
		})
	}
	return nodes
}

// executableFor 把動作詞映成 DAG executor 的真實 (ExecutorType, ActionCode)。
// 沿用 dispatchFSAction 認得的四個 FS code；未知動作回 ("","")，但不應發生
// （allowlist 已過濾，且非 read-only 會走 review）。
func executableFor(action string) (executorType, actionCode string) {
	switch strings.TrimSpace(action) {
	case "搜尋", "本機搜尋", "查找", "grep_search", "search", "find":
		return "tool_call", "grep_search"
	case "讀取", "查看", "read_file", "read":
		return "tool_call", "read_file"
	case "列出", "list_directory":
		return "tool_call", "list_directory"
	case "glob":
		return "tool_call", "glob"
	default:
		return "", ""
	}
}

// ApplyTailPatch 在記憶體中原子替換 run 的 planned tail，先驗 CAS 前置條件。
// 成功後 run.Revision +1、更新 UpdatedAt。呼叫端隨後以 AtomicSaveFullRun 落地。
func ApplyTailPatch(run *dag.DAGRun, patch TailPatch) error {
	// CAS 1：版本一致。
	if run.Revision != patch.ExpectedRevision {
		return fmt.Errorf("%w: have %d want %d", ErrRevisionConflict, run.Revision, patch.ExpectedRevision)
	}
	// CAS 2：當前活躍節點未變。
	if run.ActiveNodeID != patch.ExpectedActiveNodeID {
		return fmt.Errorf("%w: have %q want %q", ErrActiveNodeConflict, run.ActiveNodeID, patch.ExpectedActiveNodeID)
	}
	// CAS 3：planned tail 未被其他寫入動過。
	tail := PlannedTail(run)
	if got := ComputeTailHash(tail); got != patch.ExpectedOldTailHash {
		return fmt.Errorf("%w: have %s want %s", ErrTailHashConflict, got, patch.ExpectedOldTailHash)
	}

	// 保留所有非 tail 節點（succeeded / running / waiting_review / blocked / failed），
	// 只把尾段 planned/ready 換掉。
	kept := make([]dag.DAGNode, 0, len(run.Nodes))
	for _, n := range run.Nodes {
		if isTailStatus(n.Status) {
			continue // 丟棄舊 tail
		}
		kept = append(kept, n)
	}
	run.Nodes = append(kept, patch.NewNodes...)
	run.Revision++
	run.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}
