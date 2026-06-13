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

func TestServiceCreatePauseResumeDeletePersistsJobs(t *testing.T) {
	svc := newTestService(t)
	job, err := svc.CreateJob("daily", "@daily", ActionCallback, `{"callback_name":"noop"}`)
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}
	if job.ID == "" || job.NextFire == "" || !job.Enabled {
		t.Fatalf("created job missing expected fields: %#v", job)
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
