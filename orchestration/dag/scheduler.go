// dag/scheduler.go — DAG Scheduler 核心（§19）。
// 管理任務執行圖，每個節點代表一個操作步驟。
// 11 種 NodeStatus，持久化策略：僅關鍵節點。
package dag

import (
	"fmt"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// 節點狀態定義（§19.1 共 11 種）
// ──────────────────────────────────────────────

// NodeStatus 定義 DAG 節點的執行狀態。
type NodeStatus string

const (
	StatusPlanned       NodeStatus = "planned"
	StatusReady         NodeStatus = "ready"
	StatusRunning       NodeStatus = "running"
	StatusPaused        NodeStatus = "paused"
	StatusWaitingUser   NodeStatus = "waiting_user"
	StatusWaitingReview NodeStatus = "waiting_review"
	StatusBlocked       NodeStatus = "blocked"
	StatusSucceeded     NodeStatus = "succeeded"
	StatusFailed        NodeStatus = "failed"
	StatusCancelled     NodeStatus = "cancelled"
	StatusSkipped       NodeStatus = "skipped"
)

// IsCriticalStatus 判斷是否為需要持久化的關鍵狀態。
// 僅 waiting_review / blocked / failed 需要持久化。
func IsCriticalStatus(s NodeStatus) bool {
	return s == StatusWaitingReview || s == StatusBlocked || s == StatusFailed
}

// IsTerminal 判斷是否為終端狀態（不會再變更）。
func IsTerminal(s NodeStatus) bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusCancelled || s == StatusSkipped
}

// ──────────────────────────────────────────────
// DAG 結構
// ──────────────────────────────────────────────

// DAGNode 是 DAG 中的單一節點。
type DAGNode struct {
	ID             string     `json:"id"`
	Title          string     `json:"title,omitempty"`
	Operation      string     `json:"operation"`
	Action         string     `json:"action,omitempty"`
	ActionCode     string     `json:"action_code,omitempty"`
	Target         string     `json:"target,omitempty"`
	ExecutorType   string     `json:"executor_type,omitempty"`
	RiskClass      string     `json:"risk_class"`
	ModelRiskClass string     `json:"model_risk_class,omitempty"`
	Status         NodeStatus `json:"status"`
	Dependencies   []string   `json:"dependencies"` // 前置節點 ID
	ParallelRoot   bool       `json:"parallel_root,omitempty"` // TASK 31：合法平行起始節點
	BlockReason    string     `json:"block_reason"` // blocked 時的原因
	Error          string     `json:"error"`        // failed 時的錯誤訊息
	StartedAt      string     `json:"started_at"`
	CompletedAt    string     `json:"completed_at"`
	ReviewID       string     `json:"review_id,omitempty"`
	ResultSummary  string     `json:"result_summary,omitempty"`
	OutputRef      string     `json:"output_ref,omitempty"`
	TraceHash      string     `json:"trace_hash,omitempty"`
	RetryCount     int        `json:"retry_count"`
	MaxRetries     int        `json:"max_retries"`
	ApprovedBy     string     `json:"approved_by,omitempty"`
	ApprovedAt     string     `json:"approved_at,omitempty"`
	AppSessionID   string     `json:"app_session_id,omitempty"`
}

// PlannerMetadata captures the model output used to create a run.
type PlannerMetadata struct {
	NormalizedPlan        *TaskPlan `json:"normalized_plan,omitempty"`
	RawModelPlan          string    `json:"raw_model_plan,omitempty"`
	RawModelPlanTruncated bool      `json:"raw_model_plan_truncated,omitempty"`
	RepairAttemptCount    int       `json:"repair_attempt_count"`
	PlannerAdapterID      string    `json:"planner_adapter_id,omitempty"`
	PlannerModelID        string    `json:"planner_model_id,omitempty"`
	ValidationWarnings    []string  `json:"validation_warnings,omitempty"`
}

// DAGRun 是一次 DAG 執行記錄。
type DAGRun struct {
	ID              string          `json:"id"`
	Status          string          `json:"status"` // running, completed, blocked, failed
	Title           string          `json:"title,omitempty"`
	Nodes           []DAGNode       `json:"nodes"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
	GuardHash       string          `json:"guard_hash"` // Resume Guard 建立時的 hash
	HookRunID       string          `json:"hook_run_id,omitempty"`
	OutlineID       string          `json:"outline_id,omitempty"`
	ActiveNodeID    string          `json:"active_node_id,omitempty"`
	ActiveTraceID   string          `json:"active_trace_id,omitempty"`
	InterruptReason string          `json:"interrupt_reason,omitempty"`
	Planner         PlannerMetadata `json:"planner,omitempty"`
	Schema          string          `json:"schema,omitempty"` // TASK 31：持久化格式標記（dag_run.full.v1）
}

// ──────────────────────────────────────────────
// Scheduler 核心
// ──────────────────────────────────────────────

// Scheduler 管理 DAG 執行。
type Scheduler struct {
	mu          sync.Mutex
	projectRoot string
	runs        map[string]*DAGRun // 記憶體快取
}

// NewScheduler 建立 DAG 排程器。
func NewScheduler(projectRoot string) *Scheduler {
	return &Scheduler{
		projectRoot: projectRoot,
		runs:        make(map[string]*DAGRun),
	}
}

// ──────────────────────────────────────────────
// DAGRun 生命週期
// ──────────────────────────────────────────────

// CreateRun 建立新的 DAG 執行記錄。
func (s *Scheduler) CreateRun(nodes []DAGNode, guardHash string) (*DAGRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(nodes) == 0 {
		return nil, fmt.Errorf("DAG 至少需要一個節點")
	}

	now := time.Now()
	run := &DAGRun{
		ID:        fmt.Sprintf("dag-%d", now.UnixNano()),
		Status:    "running",
		Nodes:     nodes,
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
		GuardHash: guardHash,
	}

	// 初始化節點狀態
	for i := range run.Nodes {
		if run.Nodes[i].Status == "" {
			run.Nodes[i].Status = StatusPlanned
		}
	}

	// 標記無前置依賴的節點為 ready
	s.markReadyNodes(run)

	s.runs[run.ID] = run

	// 持久化關鍵節點
	SaveCriticalNodes(s.projectRoot, run)

	return run, nil
}

// GetRun 取得指定 DAGRun。
func (s *Scheduler) GetRun(runID string) (*DAGRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[runID]
	if !ok {
		// 嘗試從磁碟載入
		loaded, err := LoadCriticalNodes(s.projectRoot, runID)
		if err != nil {
			return nil, fmt.Errorf("DAGRun 不存在: %s", runID)
		}
		s.runs[runID] = loaded
		return loaded, nil
	}
	return run, nil
}

// ListRuns 列出所有 DAGRun。
func (s *Scheduler) ListRuns() []*DAGRun {
	s.mu.Lock()
	defer s.mu.Unlock()

	runs := make([]*DAGRun, 0, len(s.runs))
	for _, run := range s.runs {
		runs = append(runs, run)
	}
	return runs
}

// ──────────────────────────────────────────────
// 節點狀態更新
// ──────────────────────────────────────────────

// UpdateNodeStatus 更新指定節點的狀態。
func (s *Scheduler) UpdateNodeStatus(runID, nodeID string, status NodeStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[runID]
	if !ok {
		return fmt.Errorf("DAGRun 不存在: %s", runID)
	}

	found := false
	for i := range run.Nodes {
		if run.Nodes[i].ID == nodeID {
			now := time.Now().Format(time.RFC3339)

			// 記錄時間戳
			if status == StatusRunning && run.Nodes[i].StartedAt == "" {
				run.Nodes[i].StartedAt = now
			}
			if IsTerminal(status) {
				run.Nodes[i].CompletedAt = now
			}

			run.Nodes[i].Status = status
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("節點不存在: %s (run: %s)", nodeID, runID)
	}

	// 更新時間
	run.UpdatedAt = time.Now().Format(time.RFC3339)

	// 推進：標記新的 ready 節點
	s.markReadyNodes(run)

	// 更新 run 整體狀態
	s.updateRunStatus(run)

	// 持久化關鍵節點
	SaveCriticalNodes(s.projectRoot, run)

	return nil
}

// SetNodeBlocked 將節點標記為 blocked 並記錄原因。
func (s *Scheduler) SetNodeBlocked(runID, nodeID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[runID]
	if !ok {
		return fmt.Errorf("DAGRun 不存在: %s", runID)
	}

	for i := range run.Nodes {
		if run.Nodes[i].ID == nodeID {
			run.Nodes[i].Status = StatusBlocked
			run.Nodes[i].BlockReason = reason
			break
		}
	}

	run.UpdatedAt = time.Now().Format(time.RFC3339)
	s.updateRunStatus(run)
	SaveCriticalNodes(s.projectRoot, run)
	return nil
}

// SetNodeFailed 將節點標記為 failed 並記錄錯誤。
func (s *Scheduler) SetNodeFailed(runID, nodeID, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[runID]
	if !ok {
		return fmt.Errorf("DAGRun 不存在: %s", runID)
	}

	now := time.Now().Format(time.RFC3339)
	for i := range run.Nodes {
		if run.Nodes[i].ID == nodeID {
			run.Nodes[i].Status = StatusFailed
			run.Nodes[i].Error = errMsg
			run.Nodes[i].CompletedAt = now
			break
		}
	}

	run.UpdatedAt = now
	s.updateRunStatus(run)
	SaveCriticalNodes(s.projectRoot, run)
	return nil
}

// ──────────────────────────────────────────────
// 內部推進邏輯
// ──────────────────────────────────────────────

// markReadyNodes 將所有前置依賴已完成的 planned 節點標記為 ready。
func (s *Scheduler) markReadyNodes(run *DAGRun) {
	completedNodes := make(map[string]bool)
	for _, node := range run.Nodes {
		if node.Status == StatusSucceeded || node.Status == StatusSkipped {
			completedNodes[node.ID] = true
		}
	}

	for i := range run.Nodes {
		if run.Nodes[i].Status != StatusPlanned {
			continue
		}
		allDepsComplete := true
		for _, dep := range run.Nodes[i].Dependencies {
			if !completedNodes[dep] {
				allDepsComplete = false
				break
			}
		}
		if allDepsComplete {
			run.Nodes[i].Status = StatusReady
		}
	}
}

// updateRunStatus 根據所有節點狀態更新 run 整體狀態。
func (s *Scheduler) updateRunStatus(run *DAGRun) {
	hasBlocked := false
	hasFailed := false
	hasRunning := false
	allTerminal := true

	for _, node := range run.Nodes {
		if !IsTerminal(node.Status) {
			allTerminal = false
		}
		switch node.Status {
		case StatusBlocked:
			hasBlocked = true
		case StatusFailed:
			hasFailed = true
		case StatusRunning, StatusReady, StatusPlanned:
			hasRunning = true
		}
	}

	if allTerminal {
		if hasFailed {
			run.Status = "failed"
		} else {
			run.Status = "completed"
		}
	} else if hasBlocked {
		run.Status = "blocked"
	} else if hasRunning {
		run.Status = "running"
	}
}
