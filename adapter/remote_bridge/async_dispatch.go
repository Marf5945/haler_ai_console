// remote_bridge/async_dispatch.go — 非同步 Dispatch（§12A.5B）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ Wails binding 不阻塞 HTTP POST，使用背景 goroutine 逐段送出。│
// │                                                             │
// │ 流程：                                                      │
// │  1. DispatchAsync 立即回傳 dispatch_id（不等 HTTP）         │
// │  2. 背景 goroutine 呼叫 shrinkAndSplit 拆段                │
// │  3. 逐段呼叫 SendWebhook（每段 timeout 8-10 秒）           │
// │  4. 每段完成後 emit remote_bridge:dispatch_progress         │
// │  5. 全部完成後 emit remote_bridge:dispatch_result           │
// └─────────────────────────────────────────────────────────────┘
package remote_bridge

import (
	"fmt"
	"time"

	"ui_console/shared/eventbus"
)

// ──────────────────────────────────────────────
// 事件名稱常數
// ──────────────────────────────────────────────

const (
	EventDispatchProgress = "remote_bridge:dispatch_progress"
	EventDispatchResult   = "remote_bridge:dispatch_result"
)

// ──────────────────────────────────────────────
// 請求 / 結果結構
// ──────────────────────────────────────────────

// AsyncDispatchRequest 非同步分發請求。
type AsyncDispatchRequest struct {
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
	Priority  string `json:"priority,omitempty"` // "normal" | "high"
}

// SegmentResult 單段發送結果。
type SegmentResult struct {
	PartIndex  int    `json:"part_index"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error,omitempty"`
}

// DispatchProgressEvent 逐段進度事件 payload。
type DispatchProgressEvent struct {
	DispatchID string `json:"dispatch_id"`
	PartIndex  int    `json:"part_index"`
	TotalParts int    `json:"total_parts"`
	Status     string `json:"status"` // "sending" | "sent" | "failed"
}

// DispatchResultEvent 全部完成事件 payload。
type DispatchResultEvent struct {
	DispatchID     string          `json:"dispatch_id"`
	OverallStatus  string          `json:"overall_status"` // "success" | "partial_fail" | "failed"
	SegmentResults []SegmentResult `json:"segment_results"`
}

// ──────────────────────────────────────────────
// AsyncDispatcher
// ──────────────────────────────────────────────

// AsyncDispatcher 負責非阻塞分發。
type AsyncDispatcher struct {
	service  *Service
	eventBus *eventbus.Bus
}

// NewAsyncDispatcher 建立非同步分發器。
func NewAsyncDispatcher(service *Service, bus *eventbus.Bus) *AsyncDispatcher {
	return &AsyncDispatcher{service: service, eventBus: bus}
}

// DispatchAsync 非阻塞分發，立即回傳 dispatch_id。
func (ad *AsyncDispatcher) DispatchAsync(req AsyncDispatchRequest) string {
	dispatchID := fmt.Sprintf("disp_%d", time.Now().UnixMilli())

	// 取得通道資訊。若指定 channel_id，測試傳送會送到該通道。
	active := ad.service.GetActiveChannel()
	if req.ChannelID != "" {
		active = ad.service.GetChannelByID(req.ChannelID)
	}
	if active == nil {
		// 無啟用通道 → 立即回報失敗
		errText := "no active channel"
		if req.ChannelID != "" {
			errText = "channel not found or unavailable"
		}
		ad.eventBus.Emit(EventDispatchResult, DispatchResultEvent{
			DispatchID:    dispatchID,
			OverallStatus: "failed",
			SegmentResults: []SegmentResult{{
				PartIndex: 0,
				Error:     errText,
			}},
		})
		return dispatchID
	}

	// 背景 goroutine 執行分發
	go ad.dispatchInBackground(dispatchID, active, req.Content)

	return dispatchID
}

// RetrySegment 重試指定段（手動重試單段）。
func (ad *AsyncDispatcher) RetrySegment(dispatchID string, partIndex int, content string) error {
	active := ad.service.GetActiveChannel()
	if active == nil {
		return fmt.Errorf("no active channel")
	}

	maxLen := channelMaxLength(active.Channel)
	parts := (&Dispatcher{service: ad.service}).shrinkAndSplit(content, maxLen)

	if partIndex >= len(parts) {
		return fmt.Errorf("part_index %d out of range (total %d)", partIndex, len(parts))
	}

	// 建構 webhook request
	dispatcher := &Dispatcher{service: ad.service}
	webhookReq := dispatcher.buildWebhookRequest(active, parts[partIndex], partIndex, len(parts))
	webhookReq.TimeoutSeconds = 10

	resp, err := SendWebhook(webhookReq)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.ResponseBody)
	}

	// 回報重試成功
	ad.eventBus.Emit(EventDispatchProgress, DispatchProgressEvent{
		DispatchID: dispatchID,
		PartIndex:  partIndex,
		TotalParts: -1, // 重試不知 total
		Status:     "sent",
	})

	return nil
}

// dispatchInBackground 背景逐段發送。
func (ad *AsyncDispatcher) dispatchInBackground(dispatchID string, channel *ChannelBinding, content string) {
	dispatcher := &Dispatcher{service: ad.service}
	maxLen := channelMaxLength(channel.Channel)
	parts := dispatcher.shrinkAndSplit(content, maxLen)
	totalParts := len(parts)

	var results []SegmentResult
	failCount := 0

	for i, part := range parts {
		// 通知前端「正在送出第 i 段」
		ad.eventBus.Emit(EventDispatchProgress, DispatchProgressEvent{
			DispatchID: dispatchID,
			PartIndex:  i,
			TotalParts: totalParts,
			Status:     "sending",
		})

		// 建構並送出
		webhookReq := dispatcher.buildWebhookRequest(channel, part, i, totalParts)
		webhookReq.TimeoutSeconds = 9 // 8-10 秒範圍

		resp, err := SendWebhook(webhookReq)

		segResult := SegmentResult{PartIndex: i}
		if err != nil {
			segResult.Error = err.Error()
			failCount++
			ad.eventBus.Emit(EventDispatchProgress, DispatchProgressEvent{
				DispatchID: dispatchID,
				PartIndex:  i,
				TotalParts: totalParts,
				Status:     "failed",
			})
		} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			segResult.StatusCode = resp.StatusCode
			segResult.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			failCount++
			ad.eventBus.Emit(EventDispatchProgress, DispatchProgressEvent{
				DispatchID: dispatchID,
				PartIndex:  i,
				TotalParts: totalParts,
				Status:     "failed",
			})
		} else {
			segResult.StatusCode = resp.StatusCode
			ad.eventBus.Emit(EventDispatchProgress, DispatchProgressEvent{
				DispatchID: dispatchID,
				PartIndex:  i,
				TotalParts: totalParts,
				Status:     "sent",
			})
		}
		results = append(results, segResult)
	}

	// 決定整體狀態
	var overall string
	switch {
	case failCount == 0:
		overall = "success"
	case failCount == totalParts:
		overall = "failed"
	default:
		overall = "partial_fail"
	}

	ad.eventBus.Emit(EventDispatchResult, DispatchResultEvent{
		DispatchID:     dispatchID,
		OverallStatus:  overall,
		SegmentResults: results,
	})
}
