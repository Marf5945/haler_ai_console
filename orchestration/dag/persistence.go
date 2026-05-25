// dag/persistence.go — DAG 持久化（僅關鍵節點策略）。
// 只持久化 waiting_review / blocked / failed 狀態的節點。
// succeeded / running 等狀態在 restart 後需重算。
// 整個 DAGRun metadata 會保存，但只有關鍵節點的詳細狀態持久化。
package dag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ──────────────────────────────────────────────
// 持久化結構（僅關鍵節點）
// ──────────────────────────────────────────────

// PersistedRun 是持久化到磁碟的 DAGRun 精簡版。
type PersistedRun struct {
	ID            string         `json:"id"`
	Status        string         `json:"status"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	GuardHash     string         `json:"guard_hash"`
	TotalNodes    int            `json:"total_nodes"`    // 節點總數（用於 UI 顯示進度）
	CriticalNodes []DAGNode      `json:"critical_nodes"` // 僅關鍵狀態節點
	NodeSummary   map[string]int `json:"node_summary"`   // 各狀態計數
}

// ──────────────────────────────────────────────
// 儲存
// ──────────────────────────────────────────────

// SaveCriticalNodes 持久化 DAGRun 的關鍵節點。
func SaveCriticalNodes(projectRoot string, run *DAGRun) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	os.MkdirAll(dir, 0755)

	// 篩選關鍵節點
	var criticalNodes []DAGNode
	summary := make(map[string]int)
	for _, node := range run.Nodes {
		summary[string(node.Status)]++
		if IsCriticalStatus(node.Status) {
			criticalNodes = append(criticalNodes, node)
		}
	}

	persisted := PersistedRun{
		ID:            run.ID,
		Status:        run.Status,
		CreatedAt:     run.CreatedAt,
		UpdatedAt:     run.UpdatedAt,
		GuardHash:     run.GuardHash,
		TotalNodes:    len(run.Nodes),
		CriticalNodes: criticalNodes,
		NodeSummary:   summary,
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 DAGRun 失敗: %w", err)
	}

	path := filepath.Join(dir, run.ID+".json")
	return os.WriteFile(path, data, 0o600)
}

// SaveFullRun persists the complete run for task progress history/debug.
func SaveFullRun(projectRoot string, run *DAGRun) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化完整 DAGRun 失敗: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, run.ID+".json"), data, 0o600)
}

// ──────────────────────────────────────────────
// 載入
// ──────────────────────────────────────────────

// LoadCriticalNodes 從磁碟載入 DAGRun。
// 只有關鍵節點保留完整狀態，其餘節點狀態設為 planned（需重算）。
func LoadCriticalNodes(projectRoot string, runID string) (*DAGRun, error) {
	path := filepath.Join(projectRoot, "dag_runs", runID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("載入 DAGRun 失敗: %w", err)
	}

	var persisted PersistedRun
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("解析 DAGRun 失敗: %w", err)
	}

	// 重建 DAGRun（僅包含關鍵節點，其餘需由呼叫端重新提供）
	run := &DAGRun{
		ID:        persisted.ID,
		Status:    persisted.Status,
		CreatedAt: persisted.CreatedAt,
		UpdatedAt: persisted.UpdatedAt,
		GuardHash: persisted.GuardHash,
		Nodes:     persisted.CriticalNodes,
	}

	return run, nil
}

// LoadFullRun loads a complete run written by SaveFullRun.
func LoadFullRun(projectRoot string, runID string) (*DAGRun, error) {
	path := filepath.Join(projectRoot, "dag_runs", runID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("載入完整 DAGRun 失敗: %w", err)
	}
	var run DAGRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("解析完整 DAGRun 失敗: %w", err)
	}
	return &run, nil
}

// ListFullRuns returns complete run files, ignoring old guard sidecars.
func ListFullRuns(projectRoot string) []*DAGRun {
	dir := filepath.Join(projectRoot, "dag_runs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []*DAGRun{}
	}
	runs := []*DAGRun{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || strings.HasSuffix(entry.Name(), "_guard.json") {
			continue
		}
		runID := strings.TrimSuffix(entry.Name(), ".json")
		run, err := LoadFullRun(projectRoot, runID)
		if err == nil && run.ID != "" {
			runs = append(runs, run)
		}
	}
	return runs
}

// ──────────────────────────────────────────────
// 清單
// ──────────────────────────────────────────────

// ListPersistedRuns 列出所有持久化的 DAGRun ID。
func ListPersistedRuns(projectRoot string) ([]string, error) {
	dir := filepath.Join(projectRoot, "dag_runs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			id := strings.TrimSuffix(entry.Name(), ".json")
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// DeletePersistedRun 刪除持久化的 DAGRun。
func DeletePersistedRun(projectRoot string, runID string) error {
	path := filepath.Join(projectRoot, "dag_runs", runID+".json")
	return os.Remove(path)
}
