package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"ui_console/adapter/debugtrace"
	"ui_console/data/storage"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/eventbus"
	"ui_console/shared/scheduler"
)

type SchedulerSkillBootstrapResult struct {
	JobID         string                    `json:"job_id"`
	SkillID       string                    `json:"skill_id"`
	SourceSubID   string                    `json:"source_sub_id"`
	ActionTarget  string                    `json:"action_target"`
	ActionPayload string                    `json:"action_payload"`
	Manifest      *skill_step.SkillManifest `json:"manifest,omitempty"`
	Problems      []string                  `json:"problems,omitempty"`
}

type schedulerBootstrapSpec struct {
	Title   string
	Action  string
	Summary string
}

type schedulerEventPayloadForBootstrap struct {
	EventName string `json:"event_name"`
	Data      struct {
		Title   string `json:"title"`
		Action  string `json:"action"`
		Summary string `json:"summary"`
	} `json:"data"`
}

var schedulerSkillIDUnsafe = regexp.MustCompile(`[^a-z0-9._-]+`)

// BootstrapScheduledSkill creates/reuses a visible sub, saves a pending parameterized
// skill draft, and converts the scheduler job from an event reminder into a skill job.
func (a *App) BootstrapScheduledSkill(jobID string) (*SchedulerSkillBootstrapResult, error) {
	if a == nil || a.schedulerService == nil {
		return nil, fmt.Errorf("scheduler service 尚未初始化")
	}
	if a.skillArchive == nil {
		return nil, fmt.Errorf("skill archive 尚未初始化")
	}
	job, err := a.schedulerJobByID(jobID)
	if err != nil {
		return nil, err
	}
	spec := schedulerBootstrapSpecFromJob(job)
	sub, err := a.ensureSchedulerBootstrapSub(job, spec)
	if err != nil {
		return nil, err
	}
	actionTarget := schedulerActionTarget(spec)
	actionPayload, err := json.Marshal(map[string]string{
		"action_target": actionTarget,
		"session_id":    sub.ID,
	})
	if err != nil {
		return nil, err
	}
	skillID := schedulerSkillID(job)
	chain := &skill_step.ExpectedChain{
		Schema:   "skill_chain.v1",
		MaxSteps: 8,
		Steps: []skill_step.ExpectedStep{{
			Action:      "查詢",
			Target:      spec.Action,
			Next:        actionchain.StandbyNext,
			Code:        "ㄔ",
			Requirement: "OP",
		}},
	}
	draft, problems := a.BuildSkillDraft(skillID, spec.Title, []string{"查詢"}, []string{spec.Action, "排程"}, chain)
	draft.Description = spec.Summary
	if draft.Routing.ActionPatterns == nil {
		draft.Routing.ActionPatterns = []string{actionTarget}
	}
	if draft.Routing.TargetAliases == nil {
		draft.Routing.TargetAliases = []string{spec.Action, spec.Title}
	}
	saved, err := a.skillArchive.SavePendingDraft(draft)
	if err != nil {
		return nil, err
	}
	if err := a.AppendTalkEntryForAgent(sub.ID, "user", "建立排程自動 skill："+spec.Summary); err != nil {
		return nil, err
	}
	if err := a.AppendTalkEntryForAgent(sub.ID, "assistant", "已建立可審核的排程 skill 草稿："+skillID+"。第一次啟用後，後續排程會在此 sub 自動執行。"); err != nil {
		return nil, err
	}
	if err := a.schedulerService.BindJobSkillInSub(job.ID, skillID, sub.ID, string(actionPayload)); err != nil {
		return nil, err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{"reason": "scheduler_skill_bootstrap"})
		a.eventBus.Emit("scheduler:skill_bootstrapped", map[string]string{
			"job_id":        job.ID,
			"skill_id":      skillID,
			"source_sub_id": sub.ID,
		})
	}
	debugtrace.Record("go.scheduler.skill_bootstrap", "", map[string]interface{}{
		"job_id":        job.ID,
		"skill_id":      skillID,
		"source_sub_id": sub.ID,
		"problems":      problems,
	})
	return &SchedulerSkillBootstrapResult{
		JobID:         job.ID,
		SkillID:       skillID,
		SourceSubID:   sub.ID,
		ActionTarget:  actionTarget,
		ActionPayload: string(actionPayload),
		Manifest:      saved,
		Problems:      problems,
	}, nil
}

func (a *App) schedulerJobByID(jobID string) (*scheduler.Job, error) {
	id := strings.TrimSpace(jobID)
	if id == "" {
		return nil, fmt.Errorf("scheduler: job id 不可為空")
	}
	for _, job := range a.schedulerService.ListJobs() {
		if job.ID == id {
			copy := job
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("scheduler: 找不到任務 %q", id)
}

func (a *App) ensureSchedulerBootstrapSub(job *scheduler.Job, spec schedulerBootstrapSpec) (*CreatedSubagent, error) {
	if job != nil && strings.TrimSpace(job.SourceSubID) != "" {
		root := storage.ProjectRoot(appDataRoot(), "default")
		for _, sub := range listSubagentTabs(root) {
			if sub.id == job.SourceSubID {
				return &CreatedSubagent{ID: sub.id, Name: sub.label}, nil
			}
		}
	}
	name := "排程：" + spec.Title
	created, err := a.CreateSubagent(name)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func schedulerBootstrapSpecFromJob(job *scheduler.Job) schedulerBootstrapSpec {
	title := strings.TrimSpace(job.Name)
	spec := schedulerBootstrapSpec{Title: title, Action: title, Summary: title}
	var payload schedulerEventPayloadForBootstrap
	if err := json.Unmarshal([]byte(job.ActionPayload), &payload); err == nil {
		if strings.TrimSpace(payload.Data.Title) != "" {
			spec.Title = strings.TrimSpace(payload.Data.Title)
		}
		if strings.TrimSpace(payload.Data.Action) != "" {
			spec.Action = strings.TrimSpace(payload.Data.Action)
		}
		if strings.TrimSpace(payload.Data.Summary) != "" {
			spec.Summary = strings.TrimSpace(payload.Data.Summary)
		}
	}
	if strings.TrimSpace(spec.Title) == "" {
		spec.Title = "排程任務"
	}
	if strings.TrimSpace(spec.Action) == "" {
		spec.Action = spec.Title
	}
	if strings.TrimSpace(spec.Summary) == "" {
		spec.Summary = "在排程時間執行：" + spec.Action
	}
	return spec
}

func schedulerActionTarget(spec schedulerBootstrapSpec) string {
	target := strings.TrimSpace(spec.Action)
	if target == "" {
		target = strings.TrimSpace(spec.Title)
	}
	if target == "" {
		target = "排程任務"
	}
	return "查詢ㄌ" + target
}

func schedulerSkillID(job *scheduler.Job) string {
	no := "job"
	if job != nil && job.ScheduleNo > 0 {
		no = fmt.Sprintf("%d", job.ScheduleNo)
	} else if job != nil && strings.TrimSpace(job.ID) != "" {
		no = strings.TrimSpace(job.ID)
	}
	base := strings.ToLower(no)
	base = schedulerSkillIDUnsafe.ReplaceAllString(base, "-")
	base = strings.Trim(base, ".-_")
	if base == "" {
		base = "job"
	}
	return "sched." + base
}
