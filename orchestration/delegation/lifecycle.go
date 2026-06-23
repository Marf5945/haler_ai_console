// delegation/lifecycle.go — Sub 生命週期管理。
// 計算月度使用分數，標記低頻 sub 為清除候選。
package delegation

import (
	"sort"
	"time"
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// SubUsageStats 單一 sub 的使用統計與分數。
type SubUsageStats struct {
	SubID            string    `json:"sub_id"`
	Name             string    `json:"name"`
	UsageCount       int       `json:"usage_count"`       // 使用次數
	LastUsed         time.Time `json:"last_used"`         // 最後使用時間
	Score            float64   `json:"score"`             // 綜合使用分數
	CleanupCandidate bool      `json:"cleanup_candidate"` // 是否為清除候選
}

// CleanupResult 清除決策結果。
type CleanupResult struct {
	Retain     []string `json:"retain"`     // 保留的 sub ID 清單
	Candidates []string `json:"candidates"` // 建議清除的 sub ID 清單
}

// ──────────────────────────────────────────────
// 使用分數計算
// ──────────────────────────────────────────────

// CalculateUsageScores 依 usageLog 計算每個 sub 的使用統計與分數。
// Score = UsageCount（簡單頻率分數，可依需求擴充衰減）。
func CalculateUsageScores(entries []SubEntry, usageLog []ActionRecord) []SubUsageStats {
	// 建立使用次數與最後使用時間的統計表
	type stat struct {
		count    int
		lastUsed time.Time
	}
	stats := make(map[string]*stat, len(entries))
	for _, e := range entries {
		stats[e.ID] = &stat{}
	}

	for _, rec := range usageLog {
		if rec.Type != "sub_delegated" {
			continue
		}
		s, ok := stats[rec.ToolID]
		if !ok {
			continue
		}
		s.count++

		// 解析時間戳，更新最後使用時間
		if t, err := time.Parse(time.RFC3339, rec.Timestamp); err == nil {
			if t.After(s.lastUsed) {
				s.lastUsed = t
			}
		}
	}

	// 組合 SubUsageStats
	result := make([]SubUsageStats, 0, len(entries))
	for _, e := range entries {
		s := stats[e.ID]

		// 若 registry 有 LastUsed 且統計中無使用記錄，使用 registry 的時間
		lastUsed := s.lastUsed
		if lastUsed.IsZero() && e.LastUsed != "" {
			if t, err := time.Parse(time.RFC3339, e.LastUsed); err == nil {
				lastUsed = t
			}
		}

		result = append(result, SubUsageStats{
			SubID:      e.ID,
			Name:       e.Name,
			UsageCount: s.count,
			LastUsed:   lastUsed,
			Score:      float64(s.count),
		})
	}

	return result
}

// ──────────────────────────────────────────────
// 清除決策
// ──────────────────────────────────────────────

// DetermineCleanup 依使用分數決定保留或清除候選。
// 排序後依百分位切分：
//   - 前 25%（高頻）→ 保留
//   - 25–75%（中頻）→ 清除候選
//   - 75–90%（低頻但罕見）→ 保留（稀有但獨特）
//   - 90–100%（極低頻）→ 清除候選
func DetermineCleanup(stats []SubUsageStats) CleanupResult {
	if len(stats) == 0 {
		return CleanupResult{}
	}

	// 依分數由高到低排序
	sorted := make([]SubUsageStats, len(stats))
	copy(sorted, stats)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	n := len(sorted)
	// 計算各百分位邊界索引（ceiling）
	top25 := percentileIndex(n, 25) // 前 25%
	top75 := percentileIndex(n, 75) // 前 75%
	top90 := percentileIndex(n, 90) // 前 90%

	var retain, candidates []string

	for i, s := range sorted {
		switch {
		case i < top25:
			// 高頻（前 25%）→ 保留
			retain = append(retain, s.SubID)
		case i < top75:
			// 中頻（25–75%）→ 清除候選
			candidates = append(candidates, s.SubID)
		case i < top90:
			// 低頻但罕見（75–90%）→ 保留
			retain = append(retain, s.SubID)
		default:
			// 極低頻（90–100%）→ 清除候選
			candidates = append(candidates, s.SubID)
		}
	}

	return CleanupResult{
		Retain:     retain,
		Candidates: candidates,
	}
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

// percentileIndex 計算 n 個元素中第 pct 百分位的邊界索引（不含）。
func percentileIndex(n int, pct int) int {
	idx := n * pct / 100
	if idx > n {
		idx = n
	}
	return idx
}
