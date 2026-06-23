package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"ui_console/shared/hookgene"
)

type HookGeneReviewSummary struct {
	hookgene.SafeExportSummary
	IncompleteCount int `json:"incomplete_count"`
}

func (a *App) startHookGeneRecorder() {
	if a == nil {
		return
	}
	a.hookGeneMu.Lock()
	defer a.hookGeneMu.Unlock()
	if a.hookGeneStarted {
		return
	}
	if a.hookGeneRecorder == nil {
		r, err := hookgene.NewRecorder(hookgene.HookGeneDir(appDataRoot(), "default"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: hook gene recorder init failed: %v\n", err)
			return
		}
		a.hookGeneRecorder = r
	}
	a.hookGeneRecorder.Start()
	a.hookGeneStarted = true
}

func (a *App) stopHookGeneRecorder() {
	if a == nil {
		return
	}
	a.hookGeneMu.Lock()
	r := a.hookGeneRecorder
	a.hookGeneRecorder = nil
	a.hookGeneStarted = false
	a.hookGeneMu.Unlock()
	if r != nil {
		r.Stop()
	}
}

func (a *App) hookGeneInvocationID(traceID, fallback string) string {
	base := strings.TrimSpace(traceID)
	if base == "" {
		base = strings.TrimSpace(fallback)
	}
	if base == "" {
		base = "hookgene"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func (a *App) emitHookGeneDataEntered(skillID, invocationID string) {
	a.emitHookGeneSignal(skillID, invocationID, hookgene.SignalDataEntered, false)
}

func (a *App) emitHookGeneDataProcessed(skillID, invocationID string) {
	a.emitHookGeneSignal(skillID, invocationID, hookgene.SignalDataProcessed, false)
}

func (a *App) emitHookGeneDataLeft(skillID, invocationID string, crossedBoundary bool) {
	a.emitHookGeneSignal(skillID, invocationID, hookgene.SignalDataLeft, crossedBoundary)
}

func (a *App) emitHookGenePaused(skillID, invocationID string) {
	a.emitHookGeneSignal(skillID, invocationID, hookgene.SignalPaused, false)
}

func (a *App) emitHookGeneCompleted(skillID, invocationID string) {
	a.emitHookGeneSignal(skillID, invocationID, hookgene.SignalCompleted, false)
}

func (a *App) emitHookGeneSignal(skillID, invocationID string, typ hookgene.SignalType, crossedBoundary bool) {
	if a == nil {
		return
	}
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return
	}
	invocationID = strings.TrimSpace(invocationID)
	if invocationID == "" {
		invocationID = a.hookGeneInvocationID("", skillID)
	}
	a.hookGeneMu.Lock()
	r := a.hookGeneRecorder
	a.hookGeneMu.Unlock()
	if r == nil {
		return
	}
	r.Emit(hookgene.Signal{
		SkillID:         skillID,
		InvocationID:    invocationID,
		Type:            typ,
		CrossedBoundary: crossedBoundary,
		At:              time.Now().UTC(),
	})
}

func hookGeneSkillID(skillID string) string {
	return strings.TrimSpace(skillID)
}

// GetHookGeneReviewSummary exposes the safe, summary-only hook gene state for
// review/debug panels. It never returns raw gene strings or recorder events.
func (a *App) GetHookGeneReviewSummary(skillID string) (*HookGeneReviewSummary, error) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return nil, fmt.Errorf("hook gene review: skill_id is required")
	}
	a.hookGeneMu.Lock()
	r := a.hookGeneRecorder
	a.hookGeneMu.Unlock()
	if r == nil {
		return nil, fmt.Errorf("hook gene recorder is not available")
	}
	stats, ok := r.Stats(skillID)
	if !ok {
		return nil, fmt.Errorf("hook gene stats not found for %s", skillID)
	}
	return &HookGeneReviewSummary{
		SafeExportSummary: hookgene.BuildSafeExport(&stats),
		IncompleteCount:   stats.IncompleteCount,
	}, nil
}
