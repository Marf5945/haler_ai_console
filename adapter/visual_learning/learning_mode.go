// Package visual_learning implements Visual Capture Replay Learning Mode (v3.1.0).
// Users must explicitly activate Learning Mode — it must NEVER record in the background.
// LLM / OCR processing is deferred (not required during demonstration).
package visual_learning

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LearningRunStatus tracks the lifecycle of a recording session.
type LearningRunStatus string

const (
	LearningRunActive    LearningRunStatus = "active"
	LearningRunStopped   LearningRunStatus = "stopped"
	LearningRunProcessed LearningRunStatus = "processed"
)

// MouseEventType classifies pointer interactions.
type MouseEventType string

const (
	MouseEventClick       MouseEventType = "click"
	MouseEventRightClick  MouseEventType = "right_click"
	MouseEventDoubleClick MouseEventType = "double_click"
	MouseEventDrag        MouseEventType = "drag"
	MouseEventScroll      MouseEventType = "scroll"
)

// LearningRun is the top-level record for one user-initiated demonstration session.
// Corresponds to schema #53 in TASKS_1_2.md.
type LearningRun struct {
	ID               string            `json:"id"`
	ActiveWindowHash string            `json:"active_window_hash"`
	Tag              string            `json:"tag,omitempty"`
	Title            string            `json:"title,omitempty"`
	Name             string            `json:"name,omitempty"`
	Summary          string            `json:"summary,omitempty"`
	Keywords         []string          `json:"keywords,omitempty"`
	Risk             *OperationRisk    `json:"risk,omitempty"`
	OperationTag     string            `json:"operation_tag,omitempty"`
	StepCount        int               `json:"step_count,omitempty"`
	MetadataSource   string            `json:"metadata_source,omitempty"`
	StartedAt        time.Time         `json:"started_at"`
	StoppedAt        *time.Time        `json:"stopped_at,omitempty"`
	Status           LearningRunStatus `json:"status"`
	TracePath        string            `json:"trace_path"`
}

// MouseEventTrace records a single pointer event during a demonstration.
// Corresponds to schema #53 in TASKS_1_2.md.
// LLM / OCR processing must happen AFTER demonstration, not here.
type MouseEventTrace struct {
	Timestamp       time.Time                 `json:"timestamp"`
	EventType       MouseEventType            `json:"event_type"`
	X               int                       `json:"x"`
	Y               int                       `json:"y"`
	Button          string                    `json:"button"`
	Source          string                    `json:"source,omitempty"`
	CoordinateSpace string                    `json:"coordinate_space,omitempty"`
	TargetRegionID  string                    `json:"target_region_id"`
	TargetLabel     string                    `json:"target_label,omitempty"`
	TargetRole      string                    `json:"target_role,omitempty"`
	TargetTag       string                    `json:"target_tag,omitempty"`
	CSSSelector     string                    `json:"css_selector,omitempty"`
	TargetRect      *EventRect                `json:"target_rect,omitempty"`
	Viewport        *EventViewport            `json:"viewport,omitempty"`
	WindowsAnchor   *WindowsClickAnchorResult `json:"windows_anchor,omitempty"`
	WindowTitle     string                    `json:"window_title,omitempty"`
	WindowProcess   string                    `json:"window_process,omitempty"`
	WindowHandle    uintptr                   `json:"window_handle,omitempty"`
	WindowRect      PixelBBox                 `json:"window_rect,omitempty"`
	BeforeHash      string                    `json:"before_hash"`
	AfterHash       string                    `json:"after_hash"`
}

// LearningReplayPlan is a safe, plan-only interpretation of the last stopped
// demonstration. It is intentionally not an executor.
type LearningReplayPlan struct {
	RunID          string               `json:"run_id"`
	Tag            string               `json:"tag,omitempty"`
	OperationTag   string               `json:"operation_tag,omitempty"`
	Title          string               `json:"title,omitempty"`
	RunName        string               `json:"run_name,omitempty"`
	RunSummary     string               `json:"run_summary,omitempty"`
	Keywords       []string             `json:"keywords,omitempty"`
	Risk           *OperationRisk       `json:"risk,omitempty"`
	MetadataSource string               `json:"metadata_source,omitempty"`
	Mode           string               `json:"mode"`
	StepCount      int                  `json:"step_count"`
	Steps          []LearningReplayStep `json:"steps"`
	Note           string               `json:"note"`
}

// LearningRunMetadataUpdate is written after an LLM has named/summarized a demo.
type LearningRunMetadataUpdate struct {
	RunID          string   `json:"run_id,omitempty"`
	Tag            string   `json:"tag,omitempty"`
	OperationTag   string   `json:"operation_tag,omitempty"`
	Title          string   `json:"title,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Keywords       []string `json:"keywords,omitempty"`
	MetadataSource string   `json:"metadata_source,omitempty"`
}

// LearningRunCatalogItem is a compact index entry shown to the LLM.
type LearningRunCatalogItem struct {
	RunID          string         `json:"run_id"`
	Tag            string         `json:"tag"`
	OperationTag   string         `json:"operation_tag,omitempty"`
	Title          string         `json:"title"`
	Summary        string         `json:"summary"`
	Keywords       []string       `json:"keywords,omitempty"`
	Risk           *OperationRisk `json:"risk,omitempty"`
	MetadataSource string         `json:"metadata_source"`
	StepCount      int            `json:"step_count"`
	StartedAt      time.Time      `json:"started_at"`
	StoppedAt      *time.Time     `json:"stopped_at,omitempty"`
}

// OperationRisk is computed by code, not by the LLM.
type OperationRisk struct {
	Level   string   `json:"level"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
}

// OperationSearchResult maps a natural-language query to a saved demo tag.
type OperationSearchResult struct {
	RunID        string         `json:"run_id"`
	Tag          string         `json:"tag"`
	OperationTag string         `json:"operation_tag,omitempty"`
	Title        string         `json:"title"`
	Summary      string         `json:"summary"`
	Keywords     []string       `json:"keywords,omitempty"`
	Risk         *OperationRisk `json:"risk,omitempty"`
	Score        float64        `json:"score"`
	Reasons      []string       `json:"reasons,omitempty"`
}

// LearningReplayStep is one recorded user action translated into replay terms.
type LearningReplayStep struct {
	Index           int                       `json:"index"`
	Action          string                    `json:"action"`
	X               int                       `json:"x"`
	Y               int                       `json:"y"`
	Button          string                    `json:"button"`
	Source          string                    `json:"source,omitempty"`
	CoordinateSpace string                    `json:"coordinate_space,omitempty"`
	Label           string                    `json:"label,omitempty"`
	Role            string                    `json:"role,omitempty"`
	Tag             string                    `json:"tag,omitempty"`
	CSSSelector     string                    `json:"css_selector,omitempty"`
	TargetRect      *EventRect                `json:"target_rect,omitempty"`
	Viewport        *EventViewport            `json:"viewport,omitempty"`
	WindowsAnchor   *WindowsClickAnchorResult `json:"windows_anchor,omitempty"`
	WindowTitle     string                    `json:"window_title,omitempty"`
	WindowProcess   string                    `json:"window_process,omitempty"`
	WindowHandle    uintptr                   `json:"window_handle,omitempty"`
	WindowRect      PixelBBox                 `json:"window_rect,omitempty"`
	Summary         string                    `json:"summary"`
}

// EventRect is the browser viewport rectangle for the element hit by the user.
type EventRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// EventViewport records the visual coordinate space used by X/Y.
type EventViewport struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	DeviceScale float64 `json:"device_scale"`
}

// RecordedClickWindowsAnchor creates a compact anchor when recording a click.
// It uses the DOM element rect when available; otherwise it preserves the click
// as a small manual box. Screenshot-based YOLO/OpenCV anchoring is handled by
// ResolveWindowsClickAnchor and can be attached later when a screenshot exists.
func RecordedClickWindowsAnchor(clickX, clickY int, rect *EventRect, viewport *EventViewport) *WindowsClickAnchorResult {
	width, height := 1, 1
	if viewport != nil {
		if viewport.Width > 0 {
			width = viewport.Width
		}
		if viewport.Height > 0 {
			height = viewport.Height
		}
	}
	click := clampPoint(PixelPoint{X: clickX, Y: clickY}, width, height)
	result := &WindowsClickAnchorResult{
		Platform:        "windows",
		OK:              true,
		Click:           click,
		ImageWidth:      width,
		ImageHeight:     height,
		OCRStatus:       "not_used",
		OCRNote:         "OCR is optional and not used for recorded click anchors.",
		DetectorBackend: "recorded",
	}
	if rect != nil && rect.Width > 0 && rect.Height > 0 {
		box := clampBBox(PixelBBox{
			X: int(math.Round(rect.X)),
			Y: int(math.Round(rect.Y)),
			W: int(math.Round(rect.Width)),
			H: int(math.Round(rect.Height)),
		}, width, height)
		result.Mode = "dom_rect_anchor"
		result.Reason = "in-app DOM element rect captured during learning; YOLO/OpenCV screenshot resolver was not required"
		result.ExecutionPoint = PixelPoint{X: box.X + box.W/2, Y: box.Y + box.H/2}
		result.ExecutionHint = "click_bbox_center"
		result.AnchorBBox = box
		result.CropBBox = box
		return result
	}
	box := manualClickBox(click, width, height, 28)
	result.Mode = "manual_click_box"
	result.Reason = "no element rectangle was available during learning; preserving the click as a small manual anchor"
	result.ExecutionPoint = click
	result.ExecutionHint = "fast_click_original_point"
	result.AnchorBBox = box
	result.CropBBox = box
	result.DetectorDegraded = true
	result.NeedsReview = true
	return result
}

// LearningService manages Learning Mode activation and trace recording.
type LearningService struct {
	mu          sync.Mutex
	learnDir    string
	activeRun   *LearningRun
	isRecording bool
}

// NewLearningService creates a service whose data is stored under:
//
//	<projectRoot>/data/projects/<project>/visual_learning/learning_runs/
func NewLearningService(projectRoot string) *LearningService {
	return &LearningService{
		learnDir: filepath.Join(projectRoot, "data", "visual_learning", "learning_runs"),
	}
}

// LearnDir returns the root directory where learning runs (recordings) are stored.
func (s *LearningService) LearnDir() string {
	return s.learnDir
}

// RunDir returns the directory for a specific recording run id.
func (s *LearningService) RunDir(runID string) string {
	return filepath.Join(s.learnDir, runID)
}

// GetRun loads a single recording's metadata (run.json) by id.
func (s *LearningService) GetRun(runID string) (*LearningRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	runID = strings.TrimSpace(runID)
	if runID == "" || filepath.Base(runID) != runID || strings.Contains(runID, "..") {
		return nil, fmt.Errorf("learning run: invalid run id %q", runID)
	}
	data, err := os.ReadFile(filepath.Join(s.learnDir, runID, "run.json"))
	if err != nil {
		return nil, fmt.Errorf("learning run: load run: %w", err)
	}
	var run LearningRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("learning run: parse run: %w", err)
	}
	return &run, nil
}

// DeleteRun removes a recording's directory. Refuses to delete the active run.
func (s *LearningService) DeleteRun(runID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	runID = strings.TrimSpace(runID)
	if runID == "" || filepath.Base(runID) != runID || strings.Contains(runID, "..") {
		return fmt.Errorf("learning run: invalid run id %q", runID)
	}
	if s.activeRun != nil && s.activeRun.ID == runID {
		return fmt.Errorf("learning run: 不能刪除正在錄製中的紀錄")
	}
	dir := filepath.Join(s.learnDir, runID)
	if _, err := os.Stat(filepath.Join(dir, "run.json")); err != nil {
		return fmt.Errorf("learning run: 找不到紀錄 %q: %w", runID, err)
	}
	return os.RemoveAll(dir)
}

// ImportRunDir installs a recording folder (run.json + trace) copied to learnDir/<runID>,
// and rewrites trace_path to point at the new local location. Returns the installed run id.
func (s *LearningService) ImportRunDir(srcRunDir string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(srcRunDir, "run.json"))
	if err != nil {
		return "", fmt.Errorf("learning run: import 缺少 run.json: %w", err)
	}
	var run LearningRun
	if err := json.Unmarshal(data, &run); err != nil {
		return "", fmt.Errorf("learning run: import 解析 run.json 失敗: %w", err)
	}
	runID := strings.TrimSpace(run.ID)
	if runID == "" || filepath.Base(runID) != runID || strings.Contains(runID, "..") {
		return "", fmt.Errorf("learning run: import run id 非法 %q", runID)
	}
	destDir := filepath.Join(s.learnDir, runID)
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", err
	}
	// 複製 trace（若有）。
	srcTrace := filepath.Join(srcRunDir, "encrypted_learning_trace.jsonl")
	destTrace := filepath.Join(destDir, "encrypted_learning_trace.jsonl")
	if traceBytes, terr := os.ReadFile(srcTrace); terr == nil {
		if werr := os.WriteFile(destTrace, traceBytes, 0o600); werr != nil {
			return "", werr
		}
	}
	// trace_path 重寫成本機新位置，跨安裝/機器才指得到。
	run.TracePath = destTrace
	out, _ := json.MarshalIndent(&run, "", "  ")
	if err := os.WriteFile(filepath.Join(destDir, "run.json"), out, 0o600); err != nil {
		return "", err
	}
	return runID, nil
}

// IsRecording returns whether Learning Mode is currently active.
func (s *LearningService) IsRecording() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRecording
}

// StartDemonstration activates Learning Mode.
// Only one run may be active at a time; user must call StopDemonstration before starting again.
func (s *LearningService) StartDemonstration(activeWindowHash string) (*LearningRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRecording {
		return nil, fmt.Errorf("learning mode: already recording run %q; stop it first", s.activeRun.ID)
	}

	id := fmt.Sprintf("learn-%d", time.Now().UnixNano())
	tracePath := filepath.Join(s.learnDir, id, "encrypted_learning_trace.jsonl")
	run := &LearningRun{
		ID:               id,
		ActiveWindowHash: activeWindowHash,
		Tag:              learningRunTag(id),
		MetadataSource:   "pending_llm",
		StartedAt:        time.Now(),
		Status:           LearningRunActive,
		TracePath:        tracePath,
	}
	if err := os.MkdirAll(filepath.Dir(tracePath), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(tracePath, nil, 0o600); err != nil {
		return nil, err
	}
	if err := s.saveRunLocked(run); err != nil {
		return nil, err
	}
	s.activeRun = run
	s.isRecording = true
	return run, nil
}

// StopDemonstration deactivates Learning Mode and finalises the run.
func (s *LearningService) StopDemonstration() (*LearningRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRecording {
		return nil, fmt.Errorf("learning mode: no active recording")
	}
	now := time.Now()
	s.activeRun.StoppedAt = &now
	s.activeRun.Status = LearningRunStopped
	if steps, err := s.replayStepsFromTraceLocked(s.activeRun.TracePath); err == nil {
		s.activeRun.StepCount = len(steps)
		if strings.TrimSpace(s.activeRun.Title) == "" {
			s.activeRun.Title = buildLearningRunTitle(s.activeRun, steps)
		}
		if strings.TrimSpace(s.activeRun.Name) == "" {
			s.activeRun.Name = s.activeRun.Title
		}
		if strings.TrimSpace(s.activeRun.Summary) == "" {
			s.activeRun.Summary = buildLearningRunSummary(steps)
		}
		if strings.TrimSpace(s.activeRun.MetadataSource) == "" || s.activeRun.MetadataSource == "pending_llm" {
			s.activeRun.MetadataSource = "fallback"
		}
		s.enrichOperationMetadataLocked(s.activeRun, steps)
	}
	if err := s.saveRunLocked(s.activeRun); err != nil {
		return nil, err
	}
	run := s.activeRun
	s.activeRun = nil
	s.isRecording = false
	return run, nil
}

// RecordEvent appends a mouse event trace to the active run's trace file.
// Returns an error if Learning Mode is not active — background recording is FORBIDDEN.
func (s *LearningService) RecordEvent(event MouseEventTrace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRecording {
		return fmt.Errorf("learning mode: recording not active; cannot record event")
	}
	if err := os.MkdirAll(filepath.Dir(s.activeRun.TracePath), 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.activeRun.TracePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// ActiveRun returns a copy of the currently active run, or nil.
func (s *LearningService) ActiveRun() *LearningRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeRun == nil {
		return nil
	}
	copy := *s.activeRun
	return &copy
}

// LastReplayPlan reads the most recent stopped demonstration and returns a
// plan-only replay summary. The caller decides whether/how to execute it later.
func (s *LearningService) LastReplayPlan() (*LearningReplayPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, err := s.lastStoppedRunLocked()
	if err != nil {
		return nil, err
	}
	return s.replayPlanForRunLocked(run)
}

// ReplayPlanByTag reads a stopped demonstration by tag or run ID.
func (s *LearningService) ReplayPlanByTag(tagOrRunID string) (*LearningReplayPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, err := s.stoppedRunByTagLocked(tagOrRunID)
	if err != nil {
		return nil, err
	}
	return s.replayPlanForRunLocked(run)
}

// ListReplayCatalog returns compact metadata for recent stopped demos.
func (s *LearningService) ListReplayCatalog(limit int) ([]LearningRunCatalogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runs, err := s.stoppedRunsLocked()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(runs) {
		limit = len(runs)
	}
	items := make([]LearningRunCatalogItem, 0, limit)
	for _, run := range runs[:limit] {
		steps, _ := s.replayStepsFromTraceLocked(run.TracePath)
		s.enrichOperationMetadataLocked(run, steps)
		title := firstNonEmptyString(run.Title, run.Name)
		if title == "" {
			title = buildLearningRunTitle(run, steps)
		}
		summary := strings.TrimSpace(run.Summary)
		if summary == "" {
			summary = buildLearningRunSummary(steps)
		}
		stepCount := run.StepCount
		if stepCount == 0 {
			stepCount = len(steps)
		}
		items = append(items, LearningRunCatalogItem{
			RunID:          run.ID,
			Tag:            firstNonEmptyString(run.Tag, learningRunTag(run.ID)),
			OperationTag:   run.OperationTag,
			Title:          title,
			Summary:        summary,
			Keywords:       operationKeywords(run, steps),
			Risk:           operationRisk(run, steps),
			MetadataSource: firstNonEmptyString(run.MetadataSource, "fallback"),
			StepCount:      stepCount,
			StartedAt:      run.StartedAt,
			StoppedAt:      run.StoppedAt,
		})
	}
	return items, nil
}

// SearchOperations resolves natural language keywords to saved operation demos.
func (s *LearningService) SearchOperations(query string, limit int) ([]OperationSearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens := queryTokens(query)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("operation search: query is required")
	}
	runs, err := s.stoppedRunsLocked()
	if err != nil {
		return nil, err
	}
	results := make([]OperationSearchResult, 0, len(runs))
	for _, run := range runs {
		steps, _ := s.replayStepsFromTraceLocked(run.TracePath)
		s.enrichOperationMetadataLocked(run, steps)
		score, reasons := scoreOperationRun(run, steps, tokens)
		if score <= 0 {
			continue
		}
		results = append(results, OperationSearchResult{
			RunID:        run.ID,
			Tag:          firstNonEmptyString(run.Tag, learningRunTag(run.ID)),
			OperationTag: run.OperationTag,
			Title:        firstNonEmptyString(run.Title, run.Name, buildLearningRunTitle(run, steps)),
			Summary:      firstNonEmptyString(run.Summary, buildLearningRunSummary(steps)),
			Keywords:     operationKeywords(run, steps),
			Risk:         operationRisk(run, steps),
			Score:        score,
			Reasons:      reasons,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].RunID > results[j].RunID
		}
		return results[i].Score > results[j].Score
	})
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}
	return results, nil
}

// UpdateRunMetadata writes LLM-generated title/summary back to run.json.
func (s *LearningService) UpdateRunMetadata(update LearningRunMetadataUpdate) (*LearningRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := firstNonEmptyString(update.RunID, update.Tag)
	run, err := s.stoppedRunByTagLocked(key)
	if err != nil {
		return nil, err
	}
	if value := strings.TrimSpace(update.Tag); value != "" {
		run.Tag = value
	} else if strings.TrimSpace(run.Tag) == "" {
		run.Tag = learningRunTag(run.ID)
	}
	if value := strings.TrimSpace(update.OperationTag); value != "" {
		run.OperationTag = normalizeOperationTag(value)
	}
	if value := strings.TrimSpace(update.Title); value != "" {
		run.Title = value
		run.Name = value
	}
	if value := strings.TrimSpace(update.Summary); value != "" {
		run.Summary = value
	}
	if len(update.Keywords) > 0 {
		run.Keywords = cleanKeywords(update.Keywords)
	}
	steps, _ := s.replayStepsFromTraceLocked(run.TracePath)
	s.enrichOperationMetadataLocked(run, steps)
	run.MetadataSource = firstNonEmptyString(update.MetadataSource, "llm")
	if err := s.saveRunLocked(run); err != nil {
		return nil, err
	}
	copy := *run
	return &copy, nil
}

func (s *LearningService) replayPlanForRunLocked(run *LearningRun) (*LearningReplayPlan, error) {
	steps, err := s.replayStepsFromTraceLocked(run.TracePath)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("learning replay: last demonstration has no recorded steps")
	}
	s.enrichOperationMetadataLocked(run, steps)
	runName := strings.TrimSpace(run.Name)
	if runName == "" {
		runName = buildLearningRunTitle(run, steps)
	}
	runSummary := strings.TrimSpace(run.Summary)
	if runSummary == "" {
		runSummary = buildLearningRunSummary(steps)
	}

	return &LearningReplayPlan{
		RunID:          run.ID,
		Tag:            firstNonEmptyString(run.Tag, learningRunTag(run.ID)),
		OperationTag:   run.OperationTag,
		Title:          firstNonEmptyString(run.Title, runName),
		RunName:        runName,
		RunSummary:     runSummary,
		Keywords:       operationKeywords(run, steps),
		Risk:           operationRisk(run, steps),
		MetadataSource: firstNonEmptyString(run.MetadataSource, "fallback"),
		Mode:           "plan_only",
		StepCount:      len(steps),
		Steps:          steps,
		Note:           "Plan only: execution requires user confirmation.",
	}, nil
}

// --- persistence ---

func (s *LearningService) saveRunLocked(run *LearningRun) error {
	dir := filepath.Join(s.learnDir, run.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "run.json"), data, 0o600)
}

func (s *LearningService) lastStoppedRunLocked() (*LearningRun, error) {
	runs, err := s.stoppedRunsLocked()
	if err != nil {
		return nil, err
	}
	return runs[0], nil
}

func (s *LearningService) stoppedRunByTagLocked(tagOrRunID string) (*LearningRun, error) {
	key := strings.TrimSpace(tagOrRunID)
	if key == "" {
		return nil, fmt.Errorf("learning replay: tag or run id is required")
	}
	runs, err := s.stoppedRunsLocked()
	if err != nil {
		return nil, err
	}
	for _, run := range runs {
		if run.ID == key || run.Tag == key || learningRunTag(run.ID) == key {
			return run, nil
		}
	}
	return nil, fmt.Errorf("learning replay: demo %q not found", key)
}

func (s *LearningService) stoppedRunsLocked() ([]*LearningRun, error) {
	entries, err := os.ReadDir(s.learnDir)
	if err != nil {
		return nil, fmt.Errorf("learning replay: no demonstrations found: %w", err)
	}
	var runs []*LearningRun
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.learnDir, entry.Name(), "run.json"))
		if err != nil {
			continue
		}
		var run LearningRun
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		if run.Status == LearningRunStopped && run.TracePath != "" {
			runs = append(runs, &run)
		}
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("learning replay: no stopped demonstration found")
	}
	sort.Slice(runs, func(i, j int) bool {
		ti := runs[i].StartedAt
		tj := runs[j].StartedAt
		if runs[i].StoppedAt != nil {
			ti = *runs[i].StoppedAt
		}
		if runs[j].StoppedAt != nil {
			tj = *runs[j].StoppedAt
		}
		return ti.After(tj)
	})
	return runs, nil
}

func (s *LearningService) replayStepsFromTraceLocked(tracePath string) ([]LearningReplayStep, error) {
	data, err := os.ReadFile(tracePath)
	if err != nil {
		return nil, fmt.Errorf("learning replay: read trace: %w", err)
	}
	var steps []LearningReplayStep
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event MouseEventTrace
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.EventType == "" {
			event.EventType = MouseEventClick
		}
		if event.Button == "" {
			event.Button = "left"
		}
		step := LearningReplayStep{
			Index:           len(steps) + 1,
			Action:          string(event.EventType),
			X:               event.X,
			Y:               event.Y,
			Button:          event.Button,
			Source:          event.Source,
			CoordinateSpace: event.CoordinateSpace,
			Label:           event.TargetLabel,
			Role:            event.TargetRole,
			Tag:             event.TargetTag,
			CSSSelector:     event.CSSSelector,
			TargetRect:      event.TargetRect,
			Viewport:        event.Viewport,
			WindowsAnchor:   event.WindowsAnchor,
			WindowTitle:     event.WindowTitle,
			WindowProcess:   event.WindowProcess,
			WindowHandle:    event.WindowHandle,
			WindowRect:      event.WindowRect,
		}
		step.Summary = replayStepSummary(step)
		if prev := len(steps) - 1; prev >= 0 && shouldMergeDoubleClickStep(steps[prev], step) {
			step.Index = steps[prev].Index
			steps[prev] = step
			continue
		}
		steps = append(steps, step)
	}
	return steps, nil
}

// shouldMergeDoubleClickStep reports whether a double_click event is the second
// half of the immediately preceding single click (same window, same spot), in
// which case the single click must be replaced instead of replayed twice.
func shouldMergeDoubleClickStep(prev, next LearningReplayStep) bool {
	if next.Action != string(MouseEventDoubleClick) || prev.Action != string(MouseEventClick) {
		return false
	}
	if prev.Source != next.Source || prev.Button != next.Button {
		return false
	}
	if prev.WindowHandle != next.WindowHandle {
		return false
	}
	dx := prev.X - next.X
	dy := prev.Y - next.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= 8 && dy <= 8
}

func learningRunTag(runID string) string {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return "demo-unknown"
	}
	trimmed = strings.TrimPrefix(trimmed, "learn-")
	if len(trimmed) > 8 {
		trimmed = trimmed[len(trimmed)-8:]
	}
	return "demo-" + trimmed
}

func fallbackOperationTag(run *LearningRun, steps []LearningReplayStep) string {
	for _, token := range operationKeywords(run, steps) {
		if !genericOperationKeyword(token) {
			return normalizeOperationTag(token)
		}
	}
	if run != nil {
		return strings.TrimPrefix(learningRunTag(run.ID), "demo-")
	}
	return "unknown"
}

func normalizeOperationTag(value string) string {
	tokens := queryTokens(value)
	if len(tokens) == 0 {
		return ""
	}
	filtered := make([]string, 0, 3)
	for _, token := range tokens {
		if genericOperationKeyword(token) && len(filtered) > 0 {
			continue
		}
		filtered = append(filtered, token)
		if len(filtered) >= 3 {
			break
		}
	}
	if len(filtered) == 0 {
		filtered = tokens[:1]
	}
	return strings.Join(filtered, "-")
}

func genericOperationKeyword(token string) bool {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "", "to", "from", "click", "button", "native", "window", "screen", "browser", "google", "chrome", "exe", "操作", "操做", "開啟", "關閉", "視窗", "按鈕", "點擊":
		return true
	default:
		return false
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildLearningRunTitle(run *LearningRun, steps []LearningReplayStep) string {
	if len(steps) == 0 {
		return "empty demo"
	}
	first := replayStepTarget(steps[0])
	last := replayStepTarget(steps[len(steps)-1])
	if first == last || last == "" {
		return fmt.Sprintf("click %s", first)
	}
	return fmt.Sprintf("%s to %s", first, last)
}

func buildLearningRunSummary(steps []LearningReplayStep) string {
	if len(steps) == 0 {
		return "No replayable click steps were recorded."
	}
	nativeCount := 0
	domCount := 0
	targets := make([]string, 0, 3)
	seenTargets := map[string]bool{}
	for _, step := range steps {
		if step.Source == "native" || step.CoordinateSpace == "screen" {
			nativeCount++
		} else {
			domCount++
		}
		target := replayStepTarget(step)
		if target != "" && !seenTargets[target] && len(targets) < 3 {
			seenTargets[target] = true
			targets = append(targets, target)
		}
	}
	parts := []string{fmt.Sprintf("%d click steps", len(steps))}
	if nativeCount > 0 {
		parts = append(parts, fmt.Sprintf("%d native window steps", nativeCount))
	}
	if domCount > 0 {
		parts = append(parts, fmt.Sprintf("%d in-app DOM steps", domCount))
	}
	if len(targets) > 0 {
		parts = append(parts, "targets: "+strings.Join(targets, ", "))
	}
	return strings.Join(parts, "; ") + "."
}

func replayStepTarget(step LearningReplayStep) string {
	target := strings.TrimSpace(step.WindowTitle)
	if target == "" {
		target = strings.TrimSpace(step.Label)
	}
	if target == "" {
		target = strings.TrimSpace(step.Role)
	}
	if target == "" {
		target = strings.TrimSpace(step.CSSSelector)
	}
	if target == "" {
		target = strings.TrimSpace(step.Tag)
	}
	if target == "" {
		target = fmt.Sprintf("coordinate (%d, %d)", step.X, step.Y)
	}
	return target
}

func (s *LearningService) enrichOperationMetadataLocked(run *LearningRun, steps []LearningReplayStep) {
	if run == nil {
		return
	}
	if strings.TrimSpace(run.Tag) == "" {
		run.Tag = learningRunTag(run.ID)
	}
	// Keep replay lookup (demo-*) separate from the operation lookup tag that
	// LLMs/search use, e.g. "chatgpt" or "line".
	if strings.TrimSpace(run.OperationTag) == "" {
		run.OperationTag = fallbackOperationTag(run, steps)
	} else {
		run.OperationTag = normalizeOperationTag(run.OperationTag)
	}
	if len(run.Keywords) == 0 {
		run.Keywords = operationKeywords(run, steps)
	} else {
		run.Keywords = cleanKeywords(run.Keywords)
	}
	run.Risk = operationRisk(run, steps)
}

func operationKeywords(run *LearningRun, steps []LearningReplayStep) []string {
	keywords := []string{}
	if run != nil {
		keywords = append(keywords, run.Keywords...)
		keywords = append(keywords, run.Title, run.Name, run.Summary, run.Tag, run.OperationTag)
	}
	for _, step := range steps {
		keywords = append(keywords,
			step.Label,
			step.Role,
			step.Tag,
			step.WindowTitle,
			fileBaseKeyword(step.WindowProcess),
		)
	}
	return cleanKeywords(keywords)
}

func cleanKeywords(values []string) []string {
	seen := map[string]bool{}
	keywords := []string{}
	for _, value := range values {
		for _, token := range queryTokens(value) {
			if len(token) < 2 || seen[token] {
				continue
			}
			seen[token] = true
			keywords = append(keywords, token)
			if len(keywords) >= 24 {
				return keywords
			}
		}
	}
	return keywords
}

func operationRisk(run *LearningRun, steps []LearningReplayStep) *OperationRisk {
	score := 0
	reasons := []string{}
	nativeCount := 0
	manualAnchorCount := 0
	for _, step := range steps {
		if step.Source == "native" || step.CoordinateSpace == "screen" {
			nativeCount++
		}
		if step.WindowsAnchor != nil && step.WindowsAnchor.NeedsReview {
			manualAnchorCount++
		}
	}
	if nativeCount > 0 {
		score += 35
		reasons = append(reasons, "external_window_native_click")
	}
	if manualAnchorCount > 0 {
		score += 25
		reasons = append(reasons, "manual_anchor_needs_review")
	}
	if len(steps) >= 6 {
		score += 15
		reasons = append(reasons, "multi_step_operation")
	}
	riskText := firstNonEmptyString(run.Title, run.Name, run.Summary)
	hasDanger := operationTextHasDanger(riskText) || operationTextHasHighImpactKeywords(riskText)
	if hasDanger {
		score += 35
		reasons = append(reasons, "dangerous_action_keywords")
	}
	level := "low"
	if hasDanger && score >= 70 {
		level = "high"
	} else if score >= 35 {
		level = "medium"
	}
	return &OperationRisk{Level: level, Score: score, Reasons: reasons}
}

func operationTextHasDanger(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range []string{"delete", "remove", "pay", "purchase", "submit", "刪除", "移除", "付款", "購買", "送出"} {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

func operationTextHasHighImpactKeywords(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range []string{
		"delete", "remove", "pay", "purchase", "submit", "checkout", "transfer",
		"system settings", "settings", "registry", "admin", "administrator",
		"刪除", "移除", "付款", "購買", "送出", "提交", "轉帳", "系統設定", "設定", "登錄檔", "系統管理員",
	} {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

func scoreOperationRun(run *LearningRun, steps []LearningReplayStep, tokens []string) (float64, []string) {
	title := strings.ToLower(firstNonEmptyString(run.Title, run.Name))
	summary := strings.ToLower(run.Summary)
	keywords := operationKeywords(run, steps)
	keywordSet := map[string]bool{}
	for _, keyword := range keywords {
		keywordSet[strings.ToLower(keyword)] = true
	}
	var score float64
	reasons := []string{}
	for _, token := range tokens {
		t := strings.ToLower(token)
		switch {
		case t == "":
			continue
		case title == t || strings.Contains(title, t):
			score += 4
			reasons = append(reasons, "title:"+t)
		case keywordSet[t]:
			score += 3
			reasons = append(reasons, "keyword:"+t)
		case strings.Contains(summary, t):
			score += 2
			reasons = append(reasons, "summary:"+t)
		default:
			for _, step := range steps {
				if strings.Contains(strings.ToLower(replayStepTarget(step)), t) {
					score += 1.5
					reasons = append(reasons, "step:"+t)
					break
				}
			}
		}
	}
	if score > 0 && len(tokens) > 0 {
		score = score / float64(len(tokens))
	}
	return score, reasons
}

func queryTokens(value string) []string {
	text := strings.ToLower(strings.TrimSpace(value))
	if text == "" {
		return nil
	}
	replacer := strings.NewReplacer(
		"ㄌ", " ", "\"", " ", "'", " ", "「", " ", "」", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ",
		",", " ", ".", " ", "，", " ", "。", " ", ":", " ", "：", " ",
		"/", " ", "\\", " ", "|", " ", "-", " ", "_", " ",
	)
	text = replacer.Replace(text)
	raw := strings.Fields(text)
	tokens := make([]string, 0, len(raw))
	stop := map[string]bool{
		"操作": true, "查詢": true, "搜尋": true, "待命": true, "輸出": true,
		"demo": true, "google": true, "chrome": false,
	}
	for _, token := range raw {
		token = strings.TrimSpace(token)
		if token == "" || stop[token] {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func fileBaseKeyword(path string) string {
	base := strings.TrimSpace(filepath.Base(path))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return base
}

func replayStepSummary(step LearningReplayStep) string {
	action := strings.TrimSpace(step.Action)
	if action == "" {
		action = "click"
	}
	return fmt.Sprintf("%s %s at (%d, %d)", action, replayStepTarget(step), step.X, step.Y)
}
