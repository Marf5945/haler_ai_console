// scheduler_wake_test.go — Phase G cron→StartCalendarInterval 轉換測試（跨平台）。
package main

import (
	"testing"

	"ui_console/shared/scheduler"
)

func TestCronToCalendarIntervals_Daily(t *testing.T) {
	expr, err := scheduler.ParseCron("0 9 * * *")
	if err != nil {
		t.Fatal(err)
	}
	ivs := cronToCalendarIntervals(expr)
	if len(ivs) != 1 {
		t.Fatalf("daily 應為 1 筆, got %d", len(ivs))
	}
	if ivs[0].Minute == nil || *ivs[0].Minute != 0 {
		t.Errorf("minute 應為 0")
	}
	if ivs[0].Hour == nil || *ivs[0].Hour != 9 {
		t.Errorf("hour 應為 9")
	}
	if ivs[0].Weekday != nil || ivs[0].Day != nil {
		t.Errorf("daily 不該指定 weekday/day")
	}
}

func TestCronToCalendarIntervals_Weekly(t *testing.T) {
	expr, err := scheduler.ParseCron("30 6 * * 1")
	if err != nil {
		t.Fatal(err)
	}
	ivs := cronToCalendarIntervals(expr)
	if len(ivs) != 1 {
		t.Fatalf("weekly 應為 1 筆, got %d", len(ivs))
	}
	if ivs[0].Weekday == nil || *ivs[0].Weekday != 1 {
		t.Errorf("weekday 應為 1")
	}
	if ivs[0].Hour == nil || *ivs[0].Hour != 6 {
		t.Errorf("hour 應為 6")
	}
}

func TestCronToCalendarIntervals_HourlyOmitsHour(t *testing.T) {
	expr, err := scheduler.ParseCron("@hourly")
	if err != nil {
		t.Fatal(err)
	}
	ivs := cronToCalendarIntervals(expr)
	if len(ivs) != 1 {
		t.Fatalf("hourly 應為 1 筆, got %d", len(ivs))
	}
	if ivs[0].Minute == nil || *ivs[0].Minute != 0 {
		t.Errorf("minute 應為 0")
	}
	if ivs[0].Hour != nil {
		t.Errorf("hourly 應省略 Hour（每小時）")
	}
}

func TestCronToCalendarIntervals_YearlyKeepsMonth(t *testing.T) {
	expr, err := scheduler.ParseCron("@yearly")
	if err != nil {
		t.Fatal(err)
	}
	ivs := cronToCalendarIntervals(expr)
	if len(ivs) != 1 {
		t.Fatalf("yearly 應為 1 筆, got %d", len(ivs))
	}
	if ivs[0].Month == nil || *ivs[0].Month != 1 {
		t.Fatalf("yearly 應指定 Month=1, got %+v", ivs[0])
	}
	if ivs[0].Day == nil || *ivs[0].Day != 1 {
		t.Fatalf("yearly 應指定 Day=1, got %+v", ivs[0])
	}
}
