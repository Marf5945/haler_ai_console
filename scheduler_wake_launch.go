// scheduler_wake_launch.go — Phase G UX 精修：背景喚醒以「隱藏視窗最小背景模式」啟動，
// 並用單一實例鎖避免每次喚醒堆積行程／重複執行到期 job。
package main

import (
	"github.com/wailsapp/wails/v2/pkg/options"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// schedulerWakeFlag 由 LaunchAgent 啟動 app 時帶入，代表「這是排程喚醒，不要跳視窗」。
const schedulerWakeFlag = "--scheduled-wake"

func hasScheduledWakeFlag(args []string) bool {
	for _, a := range args {
		if a == schedulerWakeFlag {
			return true
		}
	}
	return false
}

// onSecondInstanceLaunch 處理「已有實例在跑時，又被啟動一次」：
//   - 一般重複啟動（使用者再開 app）→ 把主視窗帶到前景。
//   - 排程喚醒（--scheduled-wake）→ 維持背景，不打擾；既有 ticker 會在機器喚醒後跑到期 job，
//     第二實例由 Wails 單一實例鎖自動退出，故不會重複執行或堆積行程。
func (a *App) onSecondInstanceLaunch(data options.SecondInstanceData) {
	if hasScheduledWakeFlag(data.Args) {
		return
	}
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.Show(a.ctx)
}
