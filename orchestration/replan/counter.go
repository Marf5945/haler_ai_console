package replan

import "ui_console/orchestration/dag"

// 計數上限（編譯在 Go，不可由資料放寬）。
const (
	MaxConsecutiveNoProgress = 5 // 連續無進展上限，驅動升級與停下
	MaxRunTotal              = 8 // 整趟總 replan 硬上限（慢性震盪後門防護）
	OscillationPenalty       = 2 // A↔B 來回視為無進展，加速升級
)

// Counter 追蹤一個 DAG run 的 replan 預算。
// 「最多 5 次」算的是「連續沒有任何節點 succeeded 的 replan 次數」；
// 只要有節點成功且帶非空 ResultSummary 即歸零。另設整趟總閘。
type Counter struct {
	ConsecutiveNoProgress int             `json:"consecutive_no_progress"`
	RunTotal              int             `json:"run_total"`
	signatures            map[string]bool // 已套用過的 tail signature，用於震盪偵測
}

// NewCounter 建立計數器。
func NewCounter() *Counter {
	return &Counter{signatures: map[string]bool{}}
}

// IsOscillating 回報該 tail signature 是否先前已套用過（A↔B 來回）。
// 純讀取，供 Gate 的 GateInput.Oscillating 使用。
func (c *Counter) IsOscillating(signature string) bool {
	if c.signatures == nil {
		return false
	}
	return c.signatures[signature]
}

// RecordReplan 記錄一次「已套用的 silent replan」。
// 只有 silent 路徑會呼叫；走 review / stop 的不消耗預算。
// 若 signature 雷同（震盪）則加倍計入，加速逼近上限。
func (c *Counter) RecordReplan(signature string) {
	if c.signatures == nil {
		c.signatures = map[string]bool{}
	}
	c.RunTotal++
	if c.signatures[signature] {
		c.ConsecutiveNoProgress += OscillationPenalty
	} else {
		c.ConsecutiveNoProgress++
		c.signatures[signature] = true
	}
}

// RecordProgress 在節點完成時呼叫。只有 succeeded 且 ResultSummary 非空才歸零；
// skipped / failed / 空結果都不歸零，避免模型靠跳過或空結果洗掉計數。
func (c *Counter) RecordProgress(node dag.DAGNode) {
	if node.Status == dag.StatusSucceeded && node.ResultSummary != "" {
		c.ConsecutiveNoProgress = 0
	}
}

// ShouldStop 回報是否已達任一硬上限，必須停下交人。
func (c *Counter) ShouldStop() bool {
	return c.ConsecutiveNoProgress >= MaxConsecutiveNoProgress || c.RunTotal >= MaxRunTotal
}
