package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/domain/controlled_trust"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/eventbus"
)

type LearningDigestPrepareResult struct {
	HasUpdates            bool     `json:"has_updates"`
	Reason                string   `json:"reason"`
	SkillID               string   `json:"skill_id,omitempty"`
	Summary               string   `json:"summary"`
	ItemsAdded            int      `json:"items_added"`
	PendingCandidateCount int      `json:"pending_candidate_count"`
	RecentRunCount        int      `json:"recent_run_count"`
	HookReviewCount       int      `json:"hook_review_count"`
	Problems              []string `json:"problems,omitempty"`
	GeneratedAt           string   `json:"generated_at"`
}

// PrepareLearningDigest is the idle-time ecosystem organizer. It creates a new
// pending skill draft plus a Pending Digest item for review. It never enables,
// modifies, promotes, or executes an existing skill.
func (a *App) PrepareLearningDigest(reason string) (*LearningDigestPrepareResult, error) {
	if a == nil {
		return nil, fmt.Errorf("learning digest: app is nil")
	}
	if a.digestService == nil {
		return nil, fmt.Errorf("learning digest: pending digest service 尚未初始化")
	}
	if a.skillArchive == nil {
		return nil, fmt.Errorf("learning digest: skill archive 尚未初始化")
	}

	now := time.Now()
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "idle"
	}
	signals := a.collectLearningDigestSignals()
	skillID := fmt.Sprintf("ecosystem.learning.%d", now.UnixNano())
	title := "生態系整理建議 " + now.Format("2006-01-02 15:04")
	summary := learningDigestSummary(signals)

	chain := &skill_step.ExpectedChain{
		Schema:   "skill_chain.v1",
		MaxSteps: 6,
		Steps: []skill_step.ExpectedStep{{
			Action:      "整理",
			Target:      "生態系學習摘要",
			Next:        actionchain.StandbyNext,
			Code:        "ㄅ",
			Requirement: "OP",
		}},
	}
	draft, problems := a.BuildSkillDraft(
		skillID,
		title,
		[]string{"整理", "檢視"},
		[]string{"生態系", "學習", "skill"},
		chain,
	)
	draft.Description = summary
	draft.Routing.ActionPatterns = []string{"整理ㄌ生態系學習摘要"}
	draft.Routing.TargetAliases = []string{"生態系整理", "學習整理", "skill 建議"}
	saved, err := a.skillArchive.SavePendingDraft(draft)
	if err != nil {
		return nil, err
	}
	if err := writeLearningDigestReadme(saved.SkillID, title, summary, reason, signals); err != nil {
		debugtrace.Record("go.learning_digest.readme_error", "", map[string]interface{}{
			"skill_id": saved.SkillID,
			"error":    err.Error(),
		})
	}

	latest, err := a.digestService.LoadLatest()
	if err != nil {
		return nil, err
	}
	items := append([]controlled_trust.DigestItem{}, latest.Items...)
	items = append(items, controlled_trust.DigestItem{
		ID:           "learning-digest-" + saved.SkillID,
		SourceType:   "learning_ecosystem_skill",
		SourceID:     saved.SkillID,
		BackendGroup: controlled_trust.DigestHighValueCandidate,
		Category:     "high_value",
		Title:        title,
		RiskLevel:    "medium",
		Summary:      summary,
		Confidence:   0.72,
		AgeDays:      0,
	})
	digest, err := a.digestService.Generate(items)
	if err != nil {
		return nil, err
	}
	if result, archiveErr := a.digestService.AutoArchiveIfOverLimit(digest); archiveErr == nil && result != nil && result.ArchivedCount > 0 && a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventDigestAutoArchived, result)
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{"reason": "learning_digest"})
		a.eventBus.Emit(eventbus.EventDigestUpdated, map[string]string{
			"reason":   reason,
			"skill_id": saved.SkillID,
		})
	}
	debugtrace.Record("go.learning_digest.prepared", "", map[string]interface{}{
		"reason":                  reason,
		"skill_id":                saved.SkillID,
		"pending_candidate_count": signals.PendingCandidateCount,
		"recent_run_count":        signals.RecentRunCount,
		"hook_review_count":       signals.HookReviewCount,
		"problems":                problems,
	})
	return &LearningDigestPrepareResult{
		HasUpdates:            true,
		Reason:                reason,
		SkillID:               saved.SkillID,
		Summary:               summary,
		ItemsAdded:            1,
		PendingCandidateCount: signals.PendingCandidateCount,
		RecentRunCount:        signals.RecentRunCount,
		HookReviewCount:       signals.HookReviewCount,
		Problems:              problems,
		GeneratedAt:           now.Format(time.RFC3339),
	}, nil
}

type learningDigestSignals struct {
	PendingCandidateCount int
	RecentRunCount        int
	HookReviewCount       int
	HookSkillIDs          []string
}

func (a *App) collectLearningDigestSignals() learningDigestSignals {
	var s learningDigestSignals
	if a.pendingCandidateMgr != nil {
		s.PendingCandidateCount = a.pendingCandidateMgr.ActiveCount()
	}
	if a.learningService != nil {
		if catalog, err := a.learningService.ListReplayCatalog(5); err == nil {
			s.RecentRunCount = len(catalog)
		}
	}
	if a.skillArchive != nil {
		if skills, err := a.skillArchive.ListArchived(); err == nil {
			for _, skill := range skills {
				summary, err := a.GetHookGeneReviewSummary(skill.SkillID)
				if err != nil || summary == nil || !summary.ReviewSuggested {
					continue
				}
				s.HookReviewCount++
				if len(s.HookSkillIDs) < 3 {
					s.HookSkillIDs = append(s.HookSkillIDs, skill.SkillID)
				}
			}
		}
	}
	return s
}

func learningDigestSummary(s learningDigestSignals) string {
	parts := []string{
		fmt.Sprintf("目前有 %d 個視覺學習待審候選、%d 筆近期示範錄製。", s.PendingCandidateCount, s.RecentRunCount),
	}
	if s.HookReviewCount > 0 {
		parts = append(parts, fmt.Sprintf("hook-gene 建議檢視 %d 個可能偏肥大的 skill。", s.HookReviewCount))
	} else {
		parts = append(parts, "hook-gene 尚未達到需要修正的肥大提示門檻。")
	}
	parts = append(parts, "建議先在 Review Panel 檢視候選，只新增可審核 skill 草稿，不啟用、不覆寫既有 skill。")
	return strings.Join(parts, " ")
}

func writeLearningDigestReadme(skillID, title, summary, reason string, s learningDigestSignals) error {
	if strings.TrimSpace(skillID) == "" {
		return nil
	}
	path := filepath.Join(appDataRoot(), "data", "skills", skillID, "README.md")
	hookLine := "尚無 hook-gene 肥大提示。"
	if len(s.HookSkillIDs) > 0 {
		hookLine = "優先檢視：" + strings.Join(s.HookSkillIDs, "、")
	}
	body := fmt.Sprintf("# %s\n\n%s\n\n- 觸發原因：%s\n- 視覺學習待審候選：%d\n- 近期示範錄製：%d\n- hook-gene 建議：%s\n\n此草稿只供審核與整理建議，不會自動執行，也不會修改既有 skill。\n",
		title, summary, reason, s.PendingCandidateCount, s.RecentRunCount, hookLine)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o600)
}
