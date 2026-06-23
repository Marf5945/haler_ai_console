// throttler.go 實作 DAGThrottler：150ms timer-based 節流閥。
//
// #I-804: 攔截 Node 傳來的高頻 dag:* 狀態更新，
// 確保最多每 150ms 才向 Wails 前端 emit 一次，避免 React 渲染卡頓。
//
// 運作原理：
//  1. Enqueue(event, payload) 被呼叫時，覆寫該 event 的最新 payload
//  2. 如果 150ms 計時器尚未啟動，則啟動一個
//  3. 計時器觸發時，將所有累積的事件一次性 flush 給 emitter
//  4. 若 flush 後無新事件進入，計時器停止（不空轉）
package cli_manager

import (
	"sync"
	"time"
)

// throttleInterval 是最小 emit 間隔。
const throttleInterval = 150 * time.Millisecond

// Emitter 是事件發送介面，由 eventbus.Bus 實作。
type Emitter interface {
	Emit(eventName string, payload interface{})
}

// DAGThrottler 節流高頻 DAG 事件。
// 每個 event name 只保留最新的 payload，定時 flush。
type DAGThrottler struct {
	mu      sync.Mutex
	emitter Emitter
	pending map[string]interface{} // event name → 最新 payload
	timer   *time.Timer
	running bool
}

// NewDAGThrottler 建立節流閥。emitter 通常是 eventbus.Bus。
func NewDAGThrottler(emitter Emitter) *DAGThrottler {
	return &DAGThrottler{
		emitter: emitter,
		pending: make(map[string]interface{}),
	}
}

// Enqueue 將事件放入節流佇列。
// 同一 event name 的舊 payload 會被新的覆蓋（只保留最新狀態）。
func (t *DAGThrottler) Enqueue(eventName string, payload interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pending[eventName] = payload

	// 如果計時器尚未啟動，啟動一個
	if !t.running {
		t.running = true
		t.timer = time.AfterFunc(throttleInterval, t.flush)
	}
}

// flush 將所有累積的事件一次性發送給前端。
// 在計時器觸發時於獨立 goroutine 中執行。
func (t *DAGThrottler) flush() {
	t.mu.Lock()

	// 取出所有 pending 事件
	batch := t.pending
	t.pending = make(map[string]interface{})
	t.running = false
	t.timer = nil

	t.mu.Unlock()

	// 在鎖外逐一 emit，避免持鎖呼叫外部函式
	for name, payload := range batch {
		t.emitter.Emit(name, payload)
	}
}

// Stop 停止節流閥，取消待發送的計時器。
// 應在 app 關閉時呼叫。
func (t *DAGThrottler) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	t.running = false
	t.pending = make(map[string]interface{})
}
