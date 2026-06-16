// scheduler_autorun_test.go — Phase F 純函式測試。
package main

import (
	"strings"
	"testing"

	"ui_console/shared/scheduler"
)

func TestSchedulerRunPrompt_InjectsDate(t *testing.T) {
	spec := schedulerBootstrapSpec{Title: "星座運勢", Action: "整理今天的星座運勢", Summary: "每天整理"}
	got := schedulerRunPrompt(spec, "20260615")
	if !strings.Contains(got, "20260615") {
		t.Fatalf("prompt 應含當日日期: %q", got)
	}
	if !strings.Contains(got, "整理今天的星座運勢") {
		t.Fatalf("prompt 應含動作內容: %q", got)
	}
}

func TestSchedulerShouldSkipSameDay_UnchangedFlow(t *testing.T) {
	job := scheduler.Job{
		Name:           "每分鐘測試",
		ActionPayload:  `{"event_name":"scheduler:reminder","data":{"action":"輸出今天日期"}}`,
		SkillID:        "sched.9",
		LastOutputDate: "20260616",
	}
	job.FlowHash = scheduler.ExpectedFlowHash(job)

	if schedulerJobFlowChanged(job) {
		t.Fatalf("unchanged job should not need flow rebuild")
	}
	if !schedulerShouldSkipSameDay(job, "20260616", schedulerJobFlowChanged(job)) {
		t.Fatalf("unchanged job with today's output should skip")
	}
}

func TestSchedulerShouldSkipSameDay_FlowChangedByPayload(t *testing.T) {
	job := scheduler.Job{
		Name:           "每分鐘測試",
		ActionPayload:  `{"event_name":"scheduler:reminder","data":{"action":"輸出今天日期"}}`,
		SkillID:        "sched.9",
		LastOutputDate: "20260616",
	}
	job.FlowHash = scheduler.ExpectedFlowHash(job)
	job.ActionPayload = `{"event_name":"scheduler:reminder","data":{"action":"輸出今天日期和F5改流程測試訊息"}}`

	flowChanged := schedulerJobFlowChanged(job)
	if !flowChanged {
		t.Fatalf("payload change should need flow rebuild")
	}
	if schedulerShouldSkipSameDay(job, "20260616", flowChanged) {
		t.Fatalf("flow-changed job must not be skipped by same-day dedupe")
	}
}
