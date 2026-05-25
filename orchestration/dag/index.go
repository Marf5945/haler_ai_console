// dag/index.go — DAG Runs Index（§19.4）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 輕量級 index 檔案，避免每次歷史查詢都做 O(n) readdir。     │
// │                                                             │
// │ 路徑：data/projects/[project]/dag/dag_runs_index.json       │
// │ 格式：{ "runs": [ DAGRunSummary, ... ] }                   │
// │                                                             │
// │ 操作：                                                      │
// │  • AppendRunIndex — CreateRun 時 append                    │
// │  • UpdateRunIndex — UpdateNodeStatus 時更新                │
// │  • ListDAGRuns — 從 index 讀取（不 readdir）              │
// │  • Index 損壞時，可從 readdir 重建                         │
// └─────────────────────────────────────────────────────────────┘
package dag

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// ──────────────────────────────────────────────
// DAGRunSummary 結構
// ──────────────────────────────────────────────

// DAGRunSummary index 中的單筆 entry。
type DAGRunSummary struct {
	RunID           string `json:"run_id"`
	Status          string `json:"status"`            // succeeded|failed|running|blocked|cancelled
	StartedAt       string `json:"started_at"`
	EndedAt         string `json:"ended_at,omitempty"`
	DurationMs      int64  `json:"duration_ms,omitempty"`
	NodeCount       int    `json:"node_count"`
	FailedNodeCount int    `json:"failed_node_count"`
	SubID           string `json:"sub_id,omitempty"`
	ErrorSummary    string `json:"error_summary,omitempty"` // 前 120 字元
}

// indexData index 檔案的頂層結構。
type indexData struct {
	Runs []DAGRunSummary `json:"runs"`
}

// ──────────────────────────────────────────────
// 路徑
// ──────────────────────────────────────────────

func indexPath(projectRoot string) string {
	return filepath.Join(projectRoot, "dag", "dag_runs_index.json")
}

// ──────────────────────────────────────────────
// 讀寫
// ──────────────────────────────────────────────

func loadIndex(projectRoot string) indexData {
	data := indexData{Runs: []DAGRunSummary{}}
	raw, err := os.ReadFile(indexPath(projectRoot))
	if err != nil {
		return data
	}
	_ = json.Unmarshal(raw, &data)
	if data.Runs == nil {
		data.Runs = []DAGRunSummary{}
	}
	return data
}

func saveIndex(projectRoot string, data indexData) error {
	dir := filepath.Dir(indexPath(projectRoot))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath(projectRoot), raw, 0600)
}

// ──────────────────────────────────────────────
// 公開 API
// ──────────────────────────────────────────────

// AppendRunIndex 新增一筆 run entry 到 index。
func AppendRunIndex(projectRoot string, summary DAGRunSummary) error {
	data := loadIndex(projectRoot)
	data.Runs = append(data.Runs, summary)
	return saveIndex(projectRoot, data)
}

// UpdateRunIndex 更新指定 run 的 status / ended_at / duration / error。
func UpdateRunIndex(projectRoot string, runID string, status string, endedAt string, durationMs int64, failedNodeCount int, errorSummary string) error {
	data := loadIndex(projectRoot)

	for i := range data.Runs {
		if data.Runs[i].RunID == runID {
			data.Runs[i].Status = status
			if endedAt != "" {
				data.Runs[i].EndedAt = endedAt
			}
			if durationMs > 0 {
				data.Runs[i].DurationMs = durationMs
			}
			data.Runs[i].FailedNodeCount = failedNodeCount
			// 截斷 error_summary 至 120 字元
			if len([]rune(errorSummary)) > 120 {
				data.Runs[i].ErrorSummary = string([]rune(errorSummary)[:120])
			} else {
				data.Runs[i].ErrorSummary = errorSummary
			}
			break
		}
	}

	return saveIndex(projectRoot, data)
}

// ListDAGRuns 從 index 讀取 run 清單。
// 按 started_at 降序排列，支援 limit 和 statusFilter。
func ListDAGRuns(projectRoot string, limit int, statusFilter string) []DAGRunSummary {
	data := loadIndex(projectRoot)

	// 過濾（確保回傳空 slice 而非 nil）
	filtered := []DAGRunSummary{}
	for _, r := range data.Runs {
		if statusFilter != "" && r.Status != statusFilter {
			continue
		}
		filtered = append(filtered, r)
	}

	// 按 started_at 降序排列（最新在前）
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartedAt > filtered[j].StartedAt
	})

	// Limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered
}
