// scheduler_wake.go — 排程背景喚醒（3.1.10 Phase G）跨平台核心。
//
// 規格：關閉 app 後若使用者選「低耗能背景」，需能「從睡眠喚醒」到點執行。
// 作法：寫一個「使用者層級」LaunchAgent（~/Library/LaunchAgents，免 sudo），
// 用 StartCalendarInterval 在排程時間喚醒機器並重新啟動 app；啟動後既有 ticker/catchUp
// 會把到期 job 跑掉（autorun）。注意：launchd 可從「睡眠」喚醒，但**無法從「關機」喚醒**
// （軟體限制）→ UI 必須提示「請勿關機」。
package main

import "ui_console/shared/scheduler"

// calendarInterval 對應 launchd StartCalendarInterval 的一筆設定；nil 欄位代表「每個」。
type calendarInterval struct {
	Minute  *int
	Hour    *int
	Day     *int
	Month   *int
	Weekday *int
}

// maxWakeIntervals 限制 plist 內 interval 數量，避免 cron 展開爆量。
const maxWakeIntervals = 60

func intPtr(v int) *int { return &v }

// isFullRange 判斷一個 cron 欄位是否等於「整段範圍」（即 *，視為 wildcard）。
func isFullRange(vals []int, lo, hi int) bool {
	if len(vals) != hi-lo+1 {
		return false
	}
	seen := make(map[int]bool, len(vals))
	for _, v := range vals {
		seen[v] = true
	}
	for v := lo; v <= hi; v++ {
		if !seen[v] {
			return false
		}
	}
	return true
}

// cronToCalendarIntervals 把一個已解析 cron 轉成 launchd StartCalendarInterval 清單。
// wildcard 欄位 → 省略對應 key（= 每個）；分鐘為 wildcard 時用整點(0)代表，避免每分鐘喚醒。
func cronToCalendarIntervals(expr *scheduler.CronExpr) []calendarInterval {
	if expr == nil {
		return nil
	}
	mins := expr.Minute
	if isFullRange(mins, 0, 59) {
		mins = []int{0}
	}
	hours := expr.Hour
	hourWild := isFullRange(hours, 0, 23)
	wdays := expr.Weekday
	wdayWild := isFullRange(wdays, 0, 6)
	days := expr.Day
	dayWild := isFullRange(days, 1, 31)
	months := expr.Month
	monthWild := isFullRange(months, 1, 12)

	hourList := hours
	if hourWild {
		hourList = []int{-1} // -1 = 省略 Hour（每小時）
	}
	monthList := months
	if monthWild {
		monthList = []int{-1} // -1 = 省略 Month（每月）
	}

	var out []calendarInterval
	for _, m := range mins {
		for _, h := range hourList {
			for _, mon := range monthList {
				base := calendarInterval{Minute: intPtr(m)}
				if h >= 0 {
					base.Hour = intPtr(h)
				}
				if mon >= 0 {
					base.Month = intPtr(mon)
				}
				switch {
				case !wdayWild:
					for _, wd := range wdays {
						iv := base
						iv.Weekday = intPtr(wd)
						out = append(out, iv)
					}
				case !dayWild:
					for _, d := range days {
						iv := base
						iv.Day = intPtr(d)
						out = append(out, iv)
					}
				default:
					out = append(out, base)
				}
			}
		}
	}
	return out
}

// schedulerWakeIntervals 收集所有「啟用中」job 的喚醒時間（去重、設上限）。
func (a *App) schedulerWakeIntervals() []calendarInterval {
	if a == nil || a.schedulerService == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []calendarInterval
	for _, job := range a.schedulerService.ListJobs() {
		if !job.Enabled {
			continue
		}
		expr, err := scheduler.ParseCron(job.CronExpr)
		if err != nil {
			continue
		}
		for _, iv := range cronToCalendarIntervals(expr) {
			k := iv.key()
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, iv)
			if len(out) >= maxWakeIntervals {
				return out
			}
		}
	}
	return out
}

func ptrStr(p *int) string {
	if p == nil {
		return "-"
	}
	return itoaSmall(*p)
}

func itoaSmall(v int) string {
	// 小整數轉字串（避免引入 strconv 於本檔；範圍 -1..59）。
	if v < 0 {
		return "-1"
	}
	const digits = "0123456789"
	if v < 10 {
		return string(digits[v])
	}
	return string(digits[v/10]) + string(digits[v%10])
}

func (iv calendarInterval) key() string {
	return "m" + ptrStr(iv.Minute) + "h" + ptrStr(iv.Hour) + "d" + ptrStr(iv.Day) + "mo" + ptrStr(iv.Month) + "w" + ptrStr(iv.Weekday)
}
