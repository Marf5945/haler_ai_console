package scheduler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ui_console/shared/eventbus"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(ServiceConfig{
		DataRoot:  t.TempDir(),
		EventBus:  eventbus.New(nil),
		SkillExec: &mockSkillExecutor{},
	})
}

type mockSchedulerNotifier struct {
	fired int
}

func (m *mockSchedulerNotifier) NotifyJobFired(job *Job) {
	m.fired++
}

func (m *mockSchedulerNotifier) NotifyJobResult(job *Job, ok bool, summary string) {}

func TestServiceCreatePauseResumeDeletePersistsJobs(t *testing.T) {
	svc := newTestService(t)
	job, err := svc.CreateJob("daily", "@daily", ActionCallback, `{"callback_name":"noop"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	if job.ID == "" || job.NextFire == "" || !job.Enabled {
		t.Fatalf("created job missing expected fields: %#v", job)
	}
	if job.ScheduleNo != 1 {
		t.Fatalf("created job schedule no = %d, want 1", job.ScheduleNo)
	}

	if _, err := svc.CreateJob("bad", "@daily", ActionType("bad"), `{}`); err == nil {
		t.Fatalf("expected invalid ActionType error")
	}

	if err := svc.PauseJob(job.ID); err != nil {
		t.Fatalf("PauseJob returned error: %v", err)
	}
	if jobs := svc.ListJobs(); len(jobs) != 1 || jobs[0].Enabled {
		t.Fatalf("PauseJob did not disable job: %#v", jobs)
	}

	if err := svc.ResumeJob(job.ID); err != nil {
		t.Fatalf("ResumeJob returned error: %v", err)
	}
	if jobs := svc.ListJobs(); len(jobs) != 1 || !jobs[0].Enabled {
		t.Fatalf("ResumeJob did not enable job: %#v", jobs)
	}

	jobsPath := filepath.Join(filepath.Dir(filepath.Dir(svc.store.jobs.Path())), "scheduler", "jobs.json")
	if _, err := os.Stat(jobsPath); err != nil {
		t.Fatalf("jobs.json was not persisted at %s: %v", jobsPath, err)
	}

	if err := svc.DeleteJob(job.ID); err != nil {
		t.Fatalf("DeleteJob returned error: %v", err)
	}
	if jobs := svc.ListJobs(); len(jobs) != 0 {
		t.Fatalf("DeleteJob left jobs behind: %#v", jobs)
	}
}

func TestServiceScheduleNumberIsStableAndNotReused(t *testing.T) {
	svc := newTestService(t)
	first, err := svc.CreateJob("first", "@daily", ActionEvent, `{"event_name":"first"}`)
	if err != nil {
		t.Fatalf("CreateJob first returned error: %v", err)
	}
	second, err := svc.CreateJob("second", "@daily", ActionEvent, `{"event_name":"second"}`)
	if err != nil {
		t.Fatalf("CreateJob second returned error: %v", err)
	}
	if first.ScheduleNo != 1 || second.ScheduleNo != 2 {
		t.Fatalf("unexpected schedule numbers: first=%d second=%d", first.ScheduleNo, second.ScheduleNo)
	}
	if err := svc.DeleteJob(first.ID); err != nil {
		t.Fatalf("DeleteJob returned error: %v", err)
	}
	third, err := svc.CreateJob("third", "@daily", ActionEvent, `{"event_name":"third"}`)
	if err != nil {
		t.Fatalf("CreateJob third returned error: %v", err)
	}
	if third.ScheduleNo != 3 {
		t.Fatalf("schedule number was reused: got %d, want 3", third.ScheduleNo)
	}
}

func TestServiceUpdateJobRecomputesScheduleAndPayloadHash(t *testing.T) {
	svc := newTestService(t)
	job, err := svc.CreateJob("daily", "@daily", ActionEvent, `{"event_name":"old"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	oldHash := job.PayloadHash

	updated, err := svc.UpdateJob(job.ID, "weekly review", "0 9 * * 1", ActionEvent, `{"event_name":"new"}`)
	if err != nil {
		t.Fatalf("UpdateJob returned error: %v", err)
	}
	if updated.Name != "weekly review" || updated.CronExpr != "0 9 * * 1" {
		t.Fatalf("UpdateJob did not update visible fields: %#v", updated)
	}
	if updated.PayloadHash == "" || updated.PayloadHash == oldHash {
		t.Fatalf("UpdateJob did not refresh payload hash: old=%q updated=%q", oldHash, updated.PayloadHash)
	}
	if updated.NextFire == "" {
		t.Fatalf("UpdateJob did not recalculate NextFire: %#v", updated)
	}

	jobs := svc.ListJobs()
	if len(jobs) != 1 || jobs[0].Name != updated.Name || jobs[0].PayloadHash != updated.PayloadHash {
		t.Fatalf("updated job was not persisted in memory: %#v", jobs)
	}
}

func TestServiceExecutesDueJobAndRecordsHistory(t *testing.T) {
	svc := newTestService(t)
	done := make(chan struct{}, 1)
	svc.Callbacks().Register("done", func(ctx context.Context, args string) error {
		done <- struct{}{}
		return nil
	})
	job, err := svc.CreateJob("every minute", "* * * * *", ActionCallback, `{"callback_name":"done"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	now := time.Now().Truncate(time.Minute)
	job.NextFire = now.Add(-time.Minute).Format(time.RFC3339)

	svc.tick(context.Background(), now)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("due job was not executed")
	}

	history := waitForHistory(t, svc, job.ID, 1)
	if len(history) != 1 || history[0].Status != ExecStatusSuccess {
		t.Fatalf("unexpected history: %#v", history)
	}
	if job.LastFired == "" || job.NextFire == "" {
		t.Fatalf("job fire times were not updated: %#v", job)
	}
}

func TestServiceNotifyOnFireGatesRemoteNotifier(t *testing.T) {
	notifier := &mockSchedulerNotifier{}
	svc := NewService(ServiceConfig{
		DataRoot:  t.TempDir(),
		EventBus:  eventbus.New(nil),
		SkillExec: &mockSkillExecutor{},
		Notifier:  notifier,
	})
	calls := 0
	svc.Callbacks().Register("noop", func(ctx context.Context, args string) error {
		calls++
		return nil
	})
	job, err := svc.CreateJob("notify gate", "@daily", ActionCallback, `{"callback_name":"noop"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	if !job.NotifyOnFire {
		t.Fatalf("new job NotifyOnFire = false, want true")
	}

	svc.executeJob(context.Background(), job, time.Now())
	if notifier.fired != 1 {
		t.Fatalf("NotifyJobFired calls = %d, want 1", notifier.fired)
	}

	job.NotifyOnFire = false
	svc.executeJob(context.Background(), job, time.Now().Add(time.Minute))
	if notifier.fired != 1 {
		t.Fatalf("NotifyJobFired was called despite NotifyOnFire=false: %d", notifier.fired)
	}
	if calls != 2 {
		t.Fatalf("callback executions = %d, want 2", calls)
	}
}

func TestServiceBindJobSkillInSubPersistsSkillMode(t *testing.T) {
	svc := newTestService(t)
	job, err := svc.CreateJob("daily forecast", "@daily", ActionEvent, `{"event_name":"scheduler:reminder"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	payload := `{"action_target":"查詢ㄌ星座預告","session_id":"sub-20260614-120000"}`
	if err := svc.BindJobSkillInSub(job.ID, "sched.1", "sub-20260614-120000", payload); err != nil {
		t.Fatalf("BindJobSkillInSub returned error: %v", err)
	}

	jobs := svc.ListJobs()
	if len(jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(jobs))
	}
	got := jobs[0]
	if got.ActionType != ActionSkill || got.ActionPayload != payload {
		t.Fatalf("job was not converted to skill mode: %#v", got)
	}
	if got.SkillID != "sched.1" || got.SourceSubID != "sub-20260614-120000" || !got.AutoRunSkill {
		t.Fatalf("skill binding fields not persisted: %#v", got)
	}
	if got.PayloadHash == "" || got.FlowHash == "" {
		t.Fatalf("binding did not refresh hashes: %#v", got)
	}
}

func TestServiceSkipsOverlappingJob(t *testing.T) {
	svc := newTestService(t)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	svc.Callbacks().Register("block", func(ctx context.Context, args string) error {
		started <- struct{}{}
		<-release
		return nil
	})
	job, err := svc.CreateJob("blocking", "* * * * *", ActionCallback, `{"callback_name":"block"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	now := time.Now().Truncate(time.Minute)
	job.NextFire = now.Add(-time.Minute).Format(time.RFC3339)

	svc.tick(context.Background(), now)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("blocking job did not start")
	}

	svc.tick(context.Background(), now.Add(time.Minute))
	history, err := svc.GetJobHistory(job.ID, 10)
	if err != nil {
		t.Fatalf("GetJobHistory returned error: %v", err)
	}
	foundSkipped := false
	for _, entry := range history {
		if entry.Status == ExecStatusSkipped {
			foundSkipped = true
		}
	}
	if !foundSkipped {
		t.Fatalf("expected skipped history entry, got %#v", history)
	}

	close(release)
	svc.Stop()
}

func TestServiceRetriesOnceAndResetsFailuresOnSuccess(t *testing.T) {
	svc := newTestService(t)
	attempts := 0
	svc.Callbacks().Register("flaky", func(ctx context.Context, args string) error {
		attempts++
		if attempts == 1 {
			return errors.New("first failure")
		}
		return nil
	})
	job, err := svc.CreateJob("flaky", "@hourly", ActionCallback, `{"callback_name":"flaky"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	firedAt := time.Now().Truncate(time.Minute)

	svc.executeJob(context.Background(), job, firedAt)

	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if job.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures = %d, want 0", job.ConsecutiveFailures)
	}
	history, err := svc.GetJobHistory(job.ID, 10)
	if err != nil {
		t.Fatalf("GetJobHistory returned error: %v", err)
	}
	if len(history) != 1 || history[0].Status != ExecStatusSuccess || !history[0].Retried {
		t.Fatalf("unexpected retry history: %#v", history)
	}
}

func TestServiceCatchUpOnlyRunsPreviouslyFiredJobs(t *testing.T) {
	svc := newTestService(t)
	calls := 0
	svc.Callbacks().Register("catchup", func(ctx context.Context, args string) error {
		calls++
		return nil
	})
	job, err := svc.CreateJob("catchup", "@hourly", ActionCallback, `{"callback_name":"catchup"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	job.NextFire = time.Now().Add(-time.Hour).Format(time.RFC3339)

	svc.catchUp(context.Background())
	if calls != 0 {
		t.Fatalf("catchUp ran never-fired job")
	}

	job.LastFired = time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	svc.catchUp(context.Background())
	if calls != 1 {
		t.Fatalf("catchUp calls = %d, want 1", calls)
	}
}

func waitForHistory(t *testing.T, svc *Service, jobID string, min int) []JobExecution {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		history, err := svc.GetJobHistory(jobID, 10)
		if err != nil {
			t.Fatalf("GetJobHistory returned error: %v", err)
		}
		if len(history) >= min {
			return history
		}
		time.Sleep(10 * time.Millisecond)
	}
	history, _ := svc.GetJobHistory(jobID, 10)
	return history
}
