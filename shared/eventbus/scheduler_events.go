// scheduler_events.go — 排程器相關事件常數。
// 獨立檔案避免修改 eventbus.go 主檔，降低合併衝突風險。
package eventbus

const (
	// EventSchedulerJobFired 在排程任務成功觸發執行後發送。
	EventSchedulerJobFired = "scheduler:job_fired"

	// EventSchedulerJobSkipped 在排程任務因重疊執行而被跳過時發送。
	EventSchedulerJobSkipped = "scheduler:job_skipped"

	// EventSchedulerJobError 在排程任務執行失敗（含重試）後發送。
	EventSchedulerJobError = "scheduler:job_error"

	// EventSchedulerJobMissedCatchUp 在啟動時偵測到錯過的排程並補執行時發送。
	EventSchedulerJobMissedCatchUp = "scheduler:job_missed_catchup"

	// EventSchedulerJobCancelled 在排程任務被使用者取消時發送。
	EventSchedulerJobCancelled = "scheduler:job_cancelled"

	// EventSchedulerActionRequested 當 LLM 輸出排程相關 action chain 時發送。
	// Controller 解析後由 app.go 接手做實際排程操作。
	EventSchedulerActionRequested = "scheduler:action_requested"
)
