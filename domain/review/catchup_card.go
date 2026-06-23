// catchup_card.go — 多個 missed high-risk job 的合併審查卡。
package review

import (
	"fmt"
	"sync"
	"ui_console/domain/risk"
)

// CatchUpItem 代表一個被擋下的 catch-up job。
type CatchUpItem struct {
	JobID     string         `json:"job_id"`
	JobName   string         `json:"job_name"`
	RiskClass risk.RiskClass `json:"risk_class"`
	Decision  string         `json:"decision"` // "pending" / "execute" / "skip"
}

// CatchUpCard 是多個 missed high-risk job 的合併審查卡。
type CatchUpCard struct {
	Card  Card          `json:"card"`
	Items []CatchUpItem `json:"items"`
	mu    sync.Mutex
}

// NewCatchUpCard 建立一張 catch-up 合併卡。
func NewCatchUpCard(items []CatchUpItem) *CatchUpCard {
	// 找出所有 item 中最高的風險等級
	maxRisk := risk.Low
	for _, item := range items {
		maxRisk = risk.Max(maxRisk, item.RiskClass)
	}

	card := NewCard(CardParams{
		RiskClass:     maxRisk,
		Operation:     "scheduler_catchup",
		Target:        fmt.Sprintf("%d 個排程任務需要補執行決策", len(items)),
		Reason:        "App 關閉期間有排程任務錯過執行時間",
		AcceptLabel:   "全部補跑",
		RejectLabel:   "全部跳過",
		AcceptEffect:  "依個別選擇補跑或跳過",
		RejectEffect:  "所有錯過的任務皆跳過",
		RollbackAvail: true,
		SourceType:    "scheduler_catchup",
	})

	return &CatchUpCard{Card: card, Items: items}
}

// ResolveItem 設定單一 item 的決策。decision: "execute" 或 "skip"。
func (c *CatchUpCard) ResolveItem(jobID, decision string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.Items {
		if c.Items[i].JobID == jobID {
			if decision != "execute" && decision != "skip" {
				return fmt.Errorf("catchup_card: 無效決策 %q，需為 execute 或 skip", decision)
			}
			c.Items[i].Decision = decision
			return nil
		}
	}
	return fmt.Errorf("catchup_card: 找不到 job %q", jobID)
}

// AllResolved 回傳是否所有 item 都已決策。
func (c *CatchUpCard) AllResolved() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range c.Items {
		if item.Decision == "pending" || item.Decision == "" {
			return false
		}
	}
	return true
}

// ItemsToExecute 回傳決策為 execute 的 job ID 清單。
func (c *CatchUpCard) ItemsToExecute() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ids []string
	for _, item := range c.Items {
		if item.Decision == "execute" {
			ids = append(ids, item.JobID)
		}
	}
	return ids
}
