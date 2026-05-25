// scheduler_binding.go — Wails 前端綁定方法，暴露排程器 CRUD 給前端。
//
// 所有方法為 App 的 exported method，Wails 會自動產生 TypeScript 綁定檔。
// 前端透過 wailsjs/go/main/App.XXX() 呼叫這些方法。
package main

import (
	"context"
	"fmt"

	"ui_console/shared/scheduler"
)

// --------------------------------------------------------------------------
// CreateScheduledJob — 建立排程任務
// --------------------------------------------------------------------------

// CreateScheduledJob 建立一個新的排程任務。
//
// 參數：
//   - name:          任務顯示名稱
//   - cronExpr:      Cron 表達式（五欄位格式或 @hourly 等快捷字）
//   - actionType:    動作類型（"event" / "skill" / "callback"）
//   - actionPayload: 動作的 JSON 酬載
//
// 回傳新建立的 Job 或驗證錯誤（如 cron 表達式不合法）。
func (a *App) CreateScheduledJob(name, cronExpr, actionType, actionPayload string) (*scheduler.Job, error) {
	if a.schedulerService == nil {
		return nil, fmt.Errorf("scheduler service 尚未初始化")
	}
	// TODO: riskClass 待接上 risk.ClassifyOperation；projectID 待前端傳入
	return a.schedulerService.CreateJob(name, cronExpr, scheduler.ActionType(actionType), actionPayload, "medium", "")
}

// --------------------------------------------------------------------------
// CancelScheduledJob — 取消正在執行的排程任務
// --------------------------------------------------------------------------

// CancelScheduledJob 取消正在執行中的指定排程任務。
func (a *App) CancelScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.CancelJob(id)
}

// --------------------------------------------------------------------------
// ListScheduledJobs — 列出所有排程任務
// --------------------------------------------------------------------------

// ListScheduledJobs 回傳所有排程任務的副本清單。
func (a *App) ListScheduledJobs() []scheduler.Job {
	if a.schedulerService == nil {
		return []scheduler.Job{}
	}
	return a.schedulerService.ListJobs()
}

// --------------------------------------------------------------------------
// DeleteScheduledJob — 刪除排程任務
// --------------------------------------------------------------------------

// DeleteScheduledJob 刪除指定 ID 的排程任務。
func (a *App) DeleteScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.DeleteJob(id)
}

// --------------------------------------------------------------------------
// PauseScheduledJob — 暫停排程任務
// --------------------------------------------------------------------------

// PauseScheduledJob 暫停指定 ID 的排程任務（Enabled = false）。
func (a *App) PauseScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.PauseJob(id)
}

// --------------------------------------------------------------------------
// ResumeScheduledJob — 恢復排程任務
// --------------------------------------------------------------------------

// ResumeScheduledJob 恢復指定 ID 的排程任務（Enabled = true），並重新計算 NextFire。
func (a *App) ResumeScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.ResumeJob(id)
}

// --------------------------------------------------------------------------
// GetJobExecutionHistory — 查詢執行歷史
// --------------------------------------------------------------------------

// GetJobExecutionHistory 查詢指定任務的執行歷史紀錄。
// limit 為回傳的最大筆數，依時間由新到舊排列。
func (a *App) GetJobExecutionHistory(jobID string, limit int) ([]scheduler.JobExecution, error) {
	if a.schedulerService == nil {
		return nil, fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.GetJobHistory(jobID, limit)
}

// --------------------------------------------------------------------------
// RegisterSchedulerCallback — 註冊 Go 側 callback placeholder
// --------------------------------------------------------------------------

// RegisterSchedulerCallback 註冊一個具名 callback placeholder。
// 實際產品功能可由 Go 側模組用同名 callback 覆蓋此 placeholder。
func (a *App) RegisterSchedulerCallback(name string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	if name == "" {
		return fmt.Errorf("callback name 不可為空")
	}
	a.schedulerService.Callbacks().Register(name, func(ctx context.Context, args string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})
	return nil
}
