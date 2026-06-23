// Package scheduler 提供排程任務的持久化儲存功能。
package scheduler

import (
	"path/filepath"
	"sort"

	"ui_console/data/storage"
)

// maxHistoryEntries 定義歷史紀錄的最大保留筆數。
// 當歷史紀錄超過此數量時，會自動截斷最舊的紀錄，僅保留最新的部分。
const maxHistoryEntries = 500

// Store 封裝排程任務與執行歷史的 JSON 檔案儲存。
// 內部使用兩個 JSONStore 實例分別管理任務清單與執行歷史。
type Store struct {
	// jobs 儲存所有排程任務的定義
	jobs *storage.JSONStore[[]Job]
	// history 儲存所有任務的執行歷史紀錄
	history *storage.JSONStore[[]JobExecution]
}

// NewStore 建立新的 Store 實例。
// dataRoot 為資料根目錄，任務檔案會存放於 <dataRoot>/data/scheduler/jobs.json，
// 歷史紀錄則存放於 <dataRoot>/data/scheduler/history.json。
func NewStore(dataRoot string) *Store {
	dir := filepath.Join(dataRoot, "data", "scheduler")
	return &Store{
		jobs:    storage.NewJSONStore[[]Job](filepath.Join(dir, "jobs.json")),
		history: storage.NewJSONStore[[]JobExecution](filepath.Join(dir, "history.json")),
	}
}

// LoadJobs 從磁碟載入所有排程任務。
// 若檔案尚未建立或為空，將回傳空切片與 nil 錯誤。
func (s *Store) LoadJobs() ([]Job, error) {
	jobs, err := s.jobs.Load()
	if err != nil {
		return nil, err
	}
	if jobs == nil {
		return []Job{}, nil
	}
	return jobs, nil
}

// SaveJobs 將排程任務清單寫入磁碟。
// 此操作會完整覆寫現有的任務檔案。
func (s *Store) SaveJobs(jobs []Job) error {
	return s.jobs.Save(jobs)
}

// AppendHistory 將一筆新的執行紀錄附加至歷史檔案。
// 流程：載入現有歷史 → 附加新紀錄 → 若超過上限則截斷最舊的部分 → 寫回磁碟。
// 截斷策略：僅保留最新的 maxHistoryEntries 筆紀錄。
func (s *Store) AppendHistory(exec JobExecution) error {
	// 載入現有的歷史紀錄
	entries, err := s.history.Load()
	if err != nil {
		return err
	}

	// 附加新的執行紀錄
	entries = append(entries, exec)

	// 若超過上限，截斷最舊的紀錄，僅保留最新的 maxHistoryEntries 筆
	if len(entries) > maxHistoryEntries {
		entries = entries[len(entries)-maxHistoryEntries:]
	}

	// 寫回磁碟
	return s.history.Save(entries)
}

// GetHistory 查詢指定任務的執行歷史紀錄。
// jobID 為欲查詢的任務識別碼，limit 為回傳的最大筆數。
// 回傳結果依時間由新到舊排列（最新的紀錄在前）。
func (s *Store) GetHistory(jobID string, limit int) ([]JobExecution, error) {
	// 載入所有歷史紀錄
	all, err := s.history.Load()
	if err != nil {
		return nil, err
	}

	// 篩選出符合指定 jobID 的紀錄
	var filtered []JobExecution
	for _, entry := range all {
		if entry.JobID == jobID {
			filtered = append(filtered, entry)
		}
	}

	// 依觸發時間由新到舊排序（FiredAt 為 RFC3339 字串，字典序即時間序）
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].FiredAt > filtered[j].FiredAt
	})

	// 依 limit 限制回傳筆數
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}
