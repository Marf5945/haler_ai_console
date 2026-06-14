package main

import (
	"testing"

	"ui_console/shared/scheduler"
)

func TestSchedulerBootstrapSpecFromEventPayload(t *testing.T) {
	job := &scheduler.Job{
		ID:   "job-1",
		Name: "每日星座",
		ActionPayload: `{
			"event_name":"scheduler:reminder",
			"data":{"title":"星座預告","action":"整理今天的星座預告","summary":"每天整理今日星座資料"}
		}`,
	}
	spec := schedulerBootstrapSpecFromJob(job)
	if spec.Title != "星座預告" || spec.Action != "整理今天的星座預告" || spec.Summary != "每天整理今日星座資料" {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	if got := schedulerActionTarget(spec); got != "查詢ㄌ整理今天的星座預告" {
		t.Fatalf("action target = %q", got)
	}
}

func TestSchedulerSkillIDUsesStableScheduleNumber(t *testing.T) {
	job := &scheduler.Job{ID: "abc/def", ScheduleNo: 12}
	if got := schedulerSkillID(job); got != "sched.12" {
		t.Fatalf("skill id = %q, want sched.12", got)
	}
}
