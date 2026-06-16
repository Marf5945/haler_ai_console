// scheduler_notify.go — 排程通知（3.1.10 Phase D）。
//
// 把排程「到點要開始做事 / 完成結果」透過 remote_bridge 推到使用者綁定的
// 通訊軟體。未綁定任何 active 通道時自動安靜略過（no-op），不視為錯誤。
package main

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/adapter/remote_bridge"
	"ui_console/shared/scheduler"
)

// schedulerNotifier 實作 scheduler.SchedulerNotifier。
type schedulerNotifier struct {
	app *App
}

// NotifyJobFired：到點提醒「系統要開始做事」。
func (n schedulerNotifier) NotifyJobFired(job *scheduler.Job) {
	if n.app == nil || job == nil {
		return
	}
	title := schedulerJobTitle(job)
	n.app.dispatchSchedulerNotice(remote_bridge.NotifyProgressUpdate, "排程開始："+title, schedulerActionSummary(job))
}

// NotifyJobResult：完成/失敗結果通知（Phase E/F 接上真正執行後使用）。
func (n schedulerNotifier) NotifyJobResult(job *scheduler.Job, ok bool, summary string) {
	if n.app == nil || job == nil {
		return
	}
	title := schedulerJobTitle(job)
	body := strings.TrimSpace(summary)
	if ok {
		if body == "" {
			body = "已完成。"
		}
		n.app.dispatchSchedulerNotice(remote_bridge.NotifyResultSummary, "排程完成："+title, body)
		return
	}
	n.app.dispatchSchedulerNotice(remote_bridge.NotifySystemWarning, "排程失敗："+title, body)
}

// dispatchSchedulerNotice 透過 remote_bridge 發通知；無 active 通道則安靜略過。
func (a *App) dispatchSchedulerNotice(kind remote_bridge.NotificationType, title, details string) {
	if a == nil || a.remoteBridge == nil {
		return
	}
	dispatcher := remote_bridge.NewDispatcher(a.remoteBridge)
	if _, err := dispatcher.Dispatch(remote_bridge.DispatchRequest{
		Type:    kind,
		Title:   title,
		Details: details,
	}); err != nil {
		// 「no active channel」= 使用者沒綁通訊軟體，屬正常情況，只記 trace 不報錯。
		debugtrace.Record("go.scheduler.notify.skip", "", map[string]interface{}{
			"title": title,
			"err":   err.Error(),
		})
	}
}

func (a *App) startSchedulerHeartbeat(ctx context.Context, title string) func() {
	stop := make(chan struct{})
	var once sync.Once
	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-timer.C:
				a.dispatchSchedulerNotice(remote_bridge.NotifyHeartbeat, "排程進行中："+strings.TrimSpace(title), "長任務仍在執行中。")
				timer.Reset(30 * time.Second)
			}
		}
	}()
	return func() {
		once.Do(func() { close(stop) })
	}
}

func schedulerJobTitle(job *scheduler.Job) string {
	if job == nil {
		return "排程任務"
	}
	if t := strings.TrimSpace(job.Name); t != "" {
		return t
	}
	return "排程任務"
}

// schedulerActionSummary 從 ActionPayload 取出一句摘要供通知顯示。
// payload 形如 {"event_name":..,"data":{"title":..,"action":..,"summary":..}}。
func schedulerActionSummary(job *scheduler.Job) string {
	if job == nil {
		return ""
	}
	var p struct {
		Data struct {
			Summary string `json:"summary"`
			Action  string `json:"action"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(job.ActionPayload), &p); err == nil {
		if s := strings.TrimSpace(p.Data.Summary); s != "" {
			return s
		}
		if a := strings.TrimSpace(p.Data.Action); a != "" {
			return a
		}
	}
	return ""
}
