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
	"sync"
)

// ──────────────────────────────────────────────
// 持久化結構（僅關鍵節點）
// ──────────────────────────────────────────────

// Schema 標記避免兩種視圖共用檔名造成誤解析（TASK 31 / Phase 0.1 修 bug）。
const (
	SchemaDagCriticalV1 = "dag_run.critical.v1"
	SchemaDagFullV1     = "dag_run.full.v1"
	suffixCritical      = ".critical.json"
	suffixFull          = ".full.json"
)

// PersistedRun 是持久化到磁碟的 DAGRun 精簡版。
type PersistedRun struct {
	Schema        string         `json:"schema"` // 固定 SchemaDagCriticalV1
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
	os.MkdirAll(dir, 0o700) // 與 SaveFullRun 權限一致

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
		Schema:        SchemaDagCriticalV1, // 標記版本，load 時據此選 struct
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

	// 拆檔名：critical 與 full 各自獨立檔，不再互相覆蓋。
	path := filepath.Join(dir, run.ID+suffixCritical)
	return os.WriteFile(path, data, 0o600)
}

// SaveFullRun persists the complete run for task progress history/debug.
func SaveFullRun(projectRoot string, run *DAGRun) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	run.Schema = SchemaDagFullV1 // 標記版本（idempotent，omitempty）
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化完整 DAGRun 失敗: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, run.ID+suffixFull), data, 0o600)
}

// AtomicSaveFullRun 以 temp file + rename 原子寫入完整 DAGRun。
// 避免 replan / cancel / executor 幾乎同時寫造成 last-write-wins。
// 同檔系統上 os.Rename 為原子操作（Windows 用 MoveFileEx 亦可覆寫）。
func AtomicSaveFullRun(projectRoot string, run *DAGRun) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	run.Schema = SchemaDagFullV1
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化完整 DAGRun 失敗: %w", err)
	}
	// temp 檔與目標同目錄，確保 rename 不跨檔系統。
	tmp, err := os.CreateTemp(dir, run.ID+".full.*.tmp")
	if err != nil {
		return fmt.Errorf("建立 temp 檔失敗: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("寫入 temp 檔失敗: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	// 原子替換：成功後舊檔被新檔取代，讀者永遠看到完整檔。
	return os.Rename(tmpName, filepath.Join(dir, run.ID+suffixFull))
}

// fullRunWriteMu 序列化完整 DAGRun 的寫入，確保「單一 writer」語意：
// replan / cancel / executor 對同一 run 的寫入不會交錯覆蓋。
var fullRunWriteMu sync.Mutex

// SaveFullRunLocked 是 task 進度的單一原子寫入口：取鎖 + temp+rename。
// 取代散落各處直接呼叫 SaveFullRun（os.WriteFile，非原子）。
func SaveFullRunLocked(projectRoot string, run *DAGRun) error {
	fullRunWriteMu.Lock()
	defer fullRunWriteMu.Unlock()
	return AtomicSaveFullRun(projectRoot, run)
}

// ──────────────────────────────────────────────
// 載入
// ──────────────────────────────────────────────

// LoadCriticalNodes 從磁碟載入 DAGRun。
// 只有關鍵節點保留完整狀態，其餘節點狀態設為 planned（需重算）。
func LoadCriticalNodes(projectRoot string, runID string) (*DAGRun, error) {
	path := filepath.Join(projectRoot, "dag_runs", runID+suffixCritical)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("載入 DAGRun 失敗: %w", err)
	}

	var persisted PersistedRun
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("解析 DAGRun 失敗: %w", err)
	}
	// schema 不符代表檔案被誤寫成別種格式，明確報錯而非默默吃掉。
	if persisted.Schema != "" && persisted.Schema != SchemaDagCriticalV1 {
		return nil, fmt.Errorf("DAGRun schema 不符: 期望 %s 得到 %s", SchemaDagCriticalV1, persisted.Schema)
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
	path := filepath.Join(projectRoot, "dag_runs", runID+suffixFull)
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffixFull) {
			continue
		}
		runID := strings.TrimSuffix(entry.Name(), suffixFull)
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
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffixCritical) {
			ids = append(ids, strings.TrimSuffix(entry.Name(), suffixCritical))
		}
	}
	return ids, nil
}

// DeletePersistedRun 刪除持久化的 DAGRun。
func DeletePersistedRun(projectRoot string, runID string) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	// critical 與 full 兩檔都嘗試刪除；缺檔忽略。loop sidecar 一併清，避免孤兒檔。
	_ = os.Remove(filepath.Join(dir, runID+suffixCritical))
	DeleteLoopStatesForRun(projectRoot, runID)
	return os.Remove(filepath.Join(dir, runID+suffixFull))
}
