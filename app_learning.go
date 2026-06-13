// app_learning.go - split out of app.go (same package, codemod).
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/adapter/visual_learning"
	"ui_console/data/storage"
	"ui_console/domain/controlled_trust"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/eventbus"
)

func visualLearningModelBasePath(cwd string) string {
	programRoot := appProgramRoot(cwd)
	candidates := []string{
		filepath.Join(cwd, "assets", "models", "yolox_button_s"),
		filepath.Join(programRoot, "assets", "models", "yolox_button_s"),
		filepath.Clean(filepath.Join(programRoot, "..", "..", "assets", "models", "yolox_button_s")),
		filepath.Join(cwd, "assets", "models", "yolox_nano"),
		filepath.Join(programRoot, "assets", "models", "yolox_nano"),
		filepath.Clean(filepath.Join(programRoot, "..", "..", "assets", "models", "yolox_nano")),
	}
	if resourceRoot := appResourceRoot(); resourceRoot != "" {
		candidates = append(candidates,
			filepath.Join(resourceRoot, "assets", "models", "yolox_button_s"),
			filepath.Join(resourceRoot, "assets", "models", "yolox_nano"),
		)
	}
	for _, candidate := range candidates {
		if visualLearningModelExists(candidate) {
			return candidate
		}
	}
	return candidates[0]
}

func visualLearningModelExists(basePath string) bool {
	if info, err := os.Stat(basePath + ".mlmodelc"); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(basePath + ".onnx"); err == nil && !info.IsDir() {
		return true
	}
	return false
}

// GetPendingDigest returns the most recently generated pending digest.
// Digest is local-only computation — no LLM calls are made.
// #I-1001: 載入後檢查 300 筆上限，超過時自動封存並透過 EventBus 通知前端。
func (a *App) GetPendingDigest() (interface{}, error) {
	digest, err := a.digestService.LoadLatest()
	if err != nil {
		return nil, err
	}
	if digest.ID == "digest-empty" {
		// If no weekly digest exists yet, compute an empty digest immediately.
		digest, err = a.digestService.Generate([]controlled_trust.DigestItem{})
		if err != nil {
			return nil, err
		}
	}
	// #I-1001: 300 筆上限自動封存
	result, archiveErr := a.digestService.AutoArchiveIfOverLimit(digest)
	if archiveErr == nil && result != nil && result.ArchivedCount > 0 {
		a.eventBus.Emit(eventbus.EventDigestAutoArchived, result)
	}
	return frontendDTO(digest), nil
}

func (a *App) buildLearningReplayPromptContext(traceID string) string {
	if a == nil || a.learningService == nil {
		return ""
	}
	catalog, catalogErr := a.learningService.ListReplayCatalog(1)
	if catalogErr != nil || len(catalog) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n[系統提供: screen_control_demo 內建控制]\n")
	b.WriteString("已有保存的螢幕操作，但此處不提供 catalog 或步驟明細以節省 token。\n")
	b.WriteString("若使用者要求查詢、重現、回放、照錄製內容操作，先輸出「查詢ㄌ<自然語言關鍵詞>ㄌ操作」。\n")
	b.WriteString("系統會用關鍵詞搜尋候選操作，再把少量候選交給你選；不要直接輸出 demo tag、replay tag 或 [[控制:...]]。\n")
	b.WriteString("[/系統提供]\n")
	debugtrace.Record("go.learningReplay.prompt_context", traceID, map[string]interface{}{
		"catalog_count": len(catalog),
		"mode":          "thin_lookup_hint",
	})
	return b.String()
}

func compactLearningReplayStepForPrompt(step visual_learning.LearningReplayStep) string {
	action := strings.TrimSpace(step.Action)
	if action == "" {
		action = "click"
	}
	target := firstNonEmpty(
		step.WindowTitle,
		step.Label,
		step.Role,
		step.CSSSelector,
		step.Tag,
		"unknown target",
	)
	method := "DOM selector/viewport coordinate"
	if step.Source == "native" || step.CoordinateSpace == "screen" {
		method = "native screen coordinate"
	}
	process := strings.TrimSpace(step.WindowProcess)
	if process == "" {
		process = strings.TrimSpace(step.Tag)
	}
	process = filepath.Base(process)
	if process != "." && process != string(filepath.Separator) && process != "" {
		return fmt.Sprintf("%s via %s at (%d,%d), target=%q, process=%s", action, method, step.X, step.Y, target, process)
	}
	return fmt.Sprintf("%s via %s at (%d,%d), target=%q", action, method, step.X, step.Y, target)
}

func buildLearningMetadataPrompt(plan *visual_learning.LearningReplayPlan) string {
	var b strings.Builder
	b.WriteString("你要替一段使用者示範的螢幕操作命名。只輸出一個 JSON object，不要 Markdown，不要解釋。\n")
	b.WriteString("JSON schema: {\"title\":\"自然語言操作名稱\",\"summary\":\"一句話摘要\",\"keywords\":[\"關鍵詞\"],\"operation_tag\":\"短分類詞\"}\n")
	b.WriteString("規則：title 要讓使用者看得懂；operation_tag 用 1-3 個可搜尋詞，例如 chatgpt、line、chrome-chatgpt，不要用 demo 編號，不要用 op- 前綴；keywords 放 3-8 個詞。\n")
	if plan == nil {
		b.WriteString("步驟：none\n")
		return b.String()
	}
	fmt.Fprintf(&b, "既有 fallback title=%q summary=%q risk=%s steps=%d\n", strings.TrimSpace(plan.Title), strings.TrimSpace(plan.RunSummary), learningRiskLevel(plan.Risk), len(plan.Steps))
	for i, step := range plan.Steps {
		if i >= learningPromptMaxSteps {
			fmt.Fprintf(&b, "...另有 %d 個步驟省略。\n", len(plan.Steps)-learningPromptMaxSteps)
			break
		}
		fmt.Fprintf(&b, "%d. %s\n", i+1, compactLearningReplayStepForPrompt(step))
	}
	return b.String()
}

func parseLearningMetadataResponse(text string) (visual_learning.LearningRunMetadataUpdate, error) {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return visual_learning.LearningRunMetadataUpdate{}, fmt.Errorf("learning metadata: empty LLM response")
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return visual_learning.LearningRunMetadataUpdate{}, fmt.Errorf("learning metadata: response did not contain JSON object")
	}
	var parsed learningMetadataLLMResponse
	if err := json.Unmarshal([]byte(raw[start:end+1]), &parsed); err != nil {
		return visual_learning.LearningRunMetadataUpdate{}, fmt.Errorf("learning metadata: parse JSON: %w", err)
	}
	update := visual_learning.LearningRunMetadataUpdate{
		Title:        strings.TrimSpace(parsed.Title),
		Summary:      strings.TrimSpace(parsed.Summary),
		Keywords:     parsed.Keywords,
		OperationTag: strings.TrimSpace(parsed.OperationTag),
	}
	if update.Title == "" && update.Summary == "" && len(update.Keywords) == 0 && update.OperationTag == "" {
		return visual_learning.LearningRunMetadataUpdate{}, fmt.Errorf("learning metadata: JSON had no usable metadata")
	}
	return update, nil
}

func learningRiskLevel(risk *visual_learning.OperationRisk) string {
	if risk == nil {
		return ""
	}
	return strings.TrimSpace(risk.Level)
}

func (a *App) buildLearningOperationPromptContext(userText, traceID string) string {
	if a == nil || a.learningService == nil {
		return ""
	}
	if strings.Contains(userText, "[系統提供: operation_candidates]") {
		return ""
	}
	query, listOnly, ok := learningOperationQueryFromText(userText)
	if !ok {
		return ""
	}
	debugtrace.Record("go.learningOperation.prompt_context", traceID, map[string]interface{}{
		"query":     query,
		"list_only": listOnly,
		"mode":      "thin_lookup_hint",
	})
	var b strings.Builder
	b.WriteString("\n\n[系統提供: saved_operations 已保存螢幕操作]\n")
	b.WriteString("使用者可能正在詢問或要求已保存的螢幕操作；這不是本機文件搜尋。\n")
	if query != "" {
		fmt.Fprintf(&b, "operation_query=%q\n", query)
	}
	b.WriteString("不要猜 demo tag，也不要直接執行。請先輸出「查詢ㄌ<關鍵詞>ㄌ操作」，讓系統搜尋少量候選。\n")
	b.WriteString("若只是列出全部操作，關鍵詞可留空：查詢ㄌㄌ操作。\n")
	b.WriteString("系統回傳候選後，若使用者意圖是重現/回放/執行，再從候選中選自然關鍵詞輸出「操作ㄌ<關鍵詞>ㄌ待命」。\n")
	b.WriteString("[/系統提供]\n")
	return b.String()
}

func isLearningOperationCatalogText(text string) bool {
	_, _, ok := learningOperationQueryFromText(text)
	return ok
}

func learningOperationQueryFromText(text string) (query string, listOnly bool, ok bool) {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return "", false, false
	}
	lower := strings.ToLower(raw)
	hasOperationWord := strings.Contains(raw, "操作") || strings.Contains(raw, "操做") || strings.Contains(lower, "operation")
	hasReplayWord := containsAny(raw, []string{"重現", "復現", "回放", "重播", "錄製", "錄影", "示範", "剛剛做", "剛剛的", "做了什麼", "做了甚麼"})
	asksSavedTag := strings.Contains(lower, "tag") && containsAny(raw, []string{"儲存", "保存", "已保存", "已儲存", "畫面", "錄影", "錄製", "示範", "紀錄", "記錄", "操作"})
	asksCatalog := containsAny(raw, []string{"有哪些", "列出", "清單", "查看", "看看", "知道"}) && (hasOperationWord || containsAny(raw, []string{"錄影", "錄製", "示範", "紀錄", "記錄"}) || strings.Contains(lower, "tag"))
	if !hasOperationWord && !hasReplayWord && !asksSavedTag && !asksCatalog {
		return "", false, false
	}
	query = normalizeLearningOperationQueryText(raw)
	if asksSavedTag || asksCatalog {
		return query, query == "", true
	}
	if query == "" {
		return "", true, true
	}
	return query, false, true
}

func normalizeLearningOperationQueryText(text string) string {
	replacer := strings.NewReplacer(
		"操作", " ", "操做", " ", "operation", " ",
		"相關", " ", "關於", " ", "有關", " ",
		"已保存", " ", "已儲存", " ", "保存", " ", "儲存", " ",
		"錄影紀錄", " ", "錄製紀錄", " ", "示範紀錄", " ", "紀錄", " ", "記錄", " ",
		"示範", " ", "流程", " ", "畫面", " ", "tag", " ",
		"重現", " ", "復現", " ", "回放", " ", "重播", " ", "錄製", " ", "錄影", " ",
		"幫我", " ", "請", " ", "查詢", " ", "搜尋", " ", "查找", " ", "尋找", " ", "找", " ",
		"列出", " ", "查看", " ", "看看", " ", "知道", " ",
		"執行", " ", "回放", " ", "重播", " ", "開啟", " ", "打開", " ",
		"有哪些", " ", "什麼", " ", "甚麼", " ", "樣", " ", "的", " ",
		"，", " ", "。", " ", "！", " ", "？", " ", "、", " ",
		",", " ", ".", " ", "!", " ", "?", " ", ":", " ", ";", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ",
		"\"", " ", "'", " ", "`", " ", "<", " ", ">", " ", "|", " ", "/", " ", "\\", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(text)), " ")
}

func isLearningOperationExecutionRequest(text string) bool {
	return containsAny(text, []string{
		"重現", "回放", "重播", "照做", "照著做", "照錄製", "照示範", "執行", "開始",
		"replay", "rerun", "repeat", "execute", "run",
	})
}

// StartLearningMode 啟動視覺學習錄製模式（使用者必須明確啟動，禁止背景錄製）。
func (a *App) StartLearningMode(activeWindowHash string) (interface{}, error) {
	run, err := a.learningService.StartDemonstration(activeWindowHash)
	if err != nil {
		return nil, err
	}
	if a.nativeInput != nil {
		if err := a.nativeInput.Start(func(event visual_learning.NativeClickEvent) {
			eventType := visual_learning.MouseEventClick
			if event.ClickCount >= 2 {
				eventType = visual_learning.MouseEventDoubleClick
			}
			trace := visual_learning.MouseEventTrace{
				Timestamp:       event.Timestamp,
				EventType:       eventType,
				X:               event.X,
				Y:               event.Y,
				Button:          event.Button,
				Source:          "native",
				CoordinateSpace: "screen",
				TargetRegionID:  fmt.Sprintf("native-click-%d-%d-%d", event.Timestamp.UnixNano(), event.X, event.Y),
				TargetLabel:     strings.TrimSpace(event.WindowTitle),
				TargetRole:      "native-window",
				TargetTag:       filepath.Base(event.WindowProcess),
				Viewport: &visual_learning.EventViewport{
					Width:       event.ScreenWidth,
					Height:      event.ScreenHeight,
					DeviceScale: 1,
				},
				WindowTitle:   event.WindowTitle,
				WindowProcess: event.WindowProcess,
				WindowHandle:  event.WindowHandle,
				WindowRect:    event.WindowRect,
			}
			trace.WindowsAnchor = a.recordedNativeWindowsAnchor(event, trace.Viewport)
			if err := a.learningService.RecordEvent(trace); err != nil {
				log.Printf("visual learning native click ignored: %v", err)
			}
		}); err != nil {
			log.Printf("visual learning native recorder degraded: %v", err)
			a.eventBus.Emit("visual_learning:native_recorder_degraded", map[string]string{"error": err.Error()})
		}
	}
	a.eventBus.Emit("visual_learning:recording_started", map[string]string{"run_id": run.ID})
	return frontendDTO(run), nil
}

// StopLearningMode 停止錄製模式。
func (a *App) StopLearningMode() (interface{}, error) {
	if a.nativeInput != nil {
		if err := a.nativeInput.Stop(); err != nil {
			log.Printf("visual learning native recorder stop: %v", err)
		}
	}
	run, err := a.learningService.StopDemonstration()
	if err != nil {
		return nil, err
	}
	a.eventBus.Emit("visual_learning:recording_stopped", map[string]string{"run_id": run.ID})
	return frontendDTO(run), nil
}

// IsLearningModeActive 是否正在錄製。
func (a *App) IsLearningModeActive() bool {
	return a.learningService.IsRecording()
}

// GetActiveLearningRun 回傳目前錄製中的 run（無錄製回傳 nil）。
func (a *App) GetActiveLearningRun() interface{} {
	return frontendDTO(a.learningService.ActiveRun())
}

// RecordLearningMouseEvent records one explicit user demonstration click.
func (a *App) RecordLearningMouseEvent(payload string) error {
	var event visual_learning.MouseEventTrace
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("visual learning event: invalid payload: %w", err)
	}
	if event.EventType == "" {
		event.EventType = visual_learning.MouseEventClick
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Button == "" {
		event.Button = "left"
	}
	if strings.TrimSpace(event.TargetRegionID) == "" {
		event.TargetRegionID = fmt.Sprintf("dom-%d-%d", event.X, event.Y)
	}
	if event.WindowsAnchor == nil {
		event.WindowsAnchor = visual_learning.RecordedClickWindowsAnchor(event.X, event.Y, event.TargetRect, event.Viewport)
	}
	if err := a.learningService.RecordEvent(event); err != nil {
		return err
	}
	a.eventBus.Emit("visual_learning:event_recorded", map[string]interface{}{
		"event_type": event.EventType,
		"label":      event.TargetLabel,
		"x":          event.X,
		"y":          event.Y,
	})
	return nil
}

func (a *App) recordedNativeWindowsAnchor(event visual_learning.NativeClickEvent, fallbackViewport *visual_learning.EventViewport) *visual_learning.WindowsClickAnchorResult {
	if a == nil || a.nativeInput == nil || event.WindowHandle == 0 {
		return visual_learning.RecordedClickWindowsAnchor(event.X, event.Y, nil, fallbackViewport)
	}
	capture, err := a.nativeInput.CaptureWindow(event.WindowHandle)
	if err != nil || capture.Width <= 0 || capture.Height <= 0 || len(capture.ImageData) == 0 {
		return visual_learning.RecordedClickWindowsAnchor(event.X, event.Y, nil, fallbackViewport)
	}
	// 螢幕座標是 point，截圖是 pixel（Retina 為 2x）；anchor 一律存在截圖
	// pixel 空間（anchor.ImageWidth/Height 記錄該空間大小供回放縮放）。
	scale := capture.PixelScale()
	localX := int(math.Round(float64(event.X-capture.WindowRect.X) * scale))
	localY := int(math.Round(float64(event.Y-capture.WindowRect.Y) * scale))
	detection := a.yoloDetector.Detect(capture.ImageData, capture.Width, capture.Height)
	shape := a.opencvPipeline.Propose(capture.ImageData, capture.Width, capture.Height)
	anchor, err := visual_learning.ResolveWindowsClickAnchor(
		capture.ImageData,
		capture.Width,
		capture.Height,
		localX,
		localY,
		detection,
		shape,
		visual_learning.WindowsClickAnchorOptions{NearMissPx: 12, ShapeFallbackRadius: 48, ManualBoxSize: 28, CropPadding: 12},
	)
	if err != nil {
		return visual_learning.RecordedClickWindowsAnchor(event.X, event.Y, nil, fallbackViewport)
	}
	return &anchor
}

// GetLastLearningReplayPlan returns a safe plan-only replay of the last stopped
// demonstration. It does not execute clicks or keyboard input.
func (a *App) GetLastLearningReplayPlan() (interface{}, error) {
	plan, err := a.learningService.LastReplayPlan()
	if err != nil {
		return nil, err
	}
	return frontendDTO(plan), nil
}

// ListLearningReplayCatalog returns compact metadata for recent demos so an LLM
// can choose the right tag before asking the app to replay it.
func (a *App) ListLearningReplayCatalog(limit int) (interface{}, error) {
	items, err := a.learningService.ListReplayCatalog(limit)
	if err != nil {
		return nil, err
	}
	return frontendDTO(items), nil
}

// SearchLearningOperations resolves natural-language operation keywords.
func (a *App) SearchLearningOperations(query string, limit int) (interface{}, error) {
	items, err := a.learningService.SearchOperations(query, limit)
	if err != nil {
		return nil, err
	}
	return frontendDTO(items), nil
}

// GetLearningReplayPlan returns a safe plan-only replay by demo tag or run ID.
func (a *App) GetLearningReplayPlan(tagOrRunID string) (interface{}, error) {
	plan, err := a.learningService.ReplayPlanByTag(tagOrRunID)
	if err != nil {
		return nil, err
	}
	return frontendDTO(plan), nil
}

// UpdateLearningRunMetadata writes LLM-generated title/summary back to run.json.
func (a *App) UpdateLearningRunMetadata(payload string) (interface{}, error) {
	var update visual_learning.LearningRunMetadataUpdate
	if err := json.Unmarshal([]byte(payload), &update); err != nil {
		return nil, fmt.Errorf("learning metadata: invalid payload: %w", err)
	}
	run, err := a.learningService.UpdateRunMetadata(update)
	if err != nil {
		return nil, err
	}
	return frontendDTO(run), nil
}

// GenerateLearningRunMetadata asks the selected CLI to name a stopped demo and
// writes the result back to run.json. This is an internal operation-index step,
// not a normal user chat turn.
func (a *App) GenerateLearningRunMetadata(adapterID string, sessionID string, runID string, traceID string) (interface{}, error) {
	if a.learningService == nil {
		return nil, fmt.Errorf("learning metadata: service is not available")
	}
	if a.cliAdapter == nil {
		return nil, fmt.Errorf("learning metadata: CLI adapter is not available")
	}
	if strings.TrimSpace(adapterID) == "" {
		return nil, fmt.Errorf("learning metadata: adapter is required")
	}
	cliPath := ""
	if a.adapterRegistry != nil {
		if resolved, err := a.adapterRegistry.ResolveExecutable(adapterID); err == nil {
			cliPath = resolved
		}
	}
	if strings.TrimSpace(cliPath) == "" {
		return nil, fmt.Errorf("learning metadata: CLI executable is not available for %s", adapterID)
	}
	if err := a.ensureSidecarRunning(); err != nil {
		return nil, err
	}
	plan, err := a.learningService.ReplayPlanByTag(runID)
	if err != nil {
		return nil, err
	}
	prompt := buildLearningMetadataPrompt(plan)
	debugtrace.Record("go.learningMetadata.generate.enter", traceID, map[string]interface{}{
		"adapter_id": adapterID,
		"run_id":     runID,
		"steps":      len(plan.Steps),
	})
	resp, err := a.cliAdapter.SendMessage(skill_step.CLIMessageOptions{
		AdapterID:      adapterID,
		CLIPath:        cliPath,
		SessionID:      sessionID,
		UserText:       prompt,
		ContinuityKey:  conversationContinuityKey("learning-metadata", sessionID),
		TraceID:        traceID,
		SkipContinuity: true,
	})
	if err != nil {
		debugtrace.Record("go.learningMetadata.generate.error", traceID, map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	if strings.TrimSpace(resp.Error) != "" {
		debugtrace.Record("go.learningMetadata.generate.error", traceID, map[string]interface{}{"error": resp.Error})
		return nil, fmt.Errorf("learning metadata: %s", resp.Error)
	}
	debugtrace.Record("go.learningMetadata.generate.raw", traceID, map[string]interface{}{
		"text": resp.Text,
	})
	update, err := parseLearningMetadataResponse(resp.Text)
	if err != nil {
		debugtrace.Record("go.learningMetadata.generate.error", traceID, map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	update.RunID = plan.RunID
	update.Tag = plan.Tag
	update.MetadataSource = "llm"
	run, err := a.learningService.UpdateRunMetadata(update)
	if err != nil {
		debugtrace.Record("go.learningMetadata.generate.error", traceID, map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	debugtrace.Record("go.learningMetadata.generate.updated", traceID, map[string]interface{}{
		"run_id":        run.ID,
		"tag":           run.Tag,
		"operation_tag": run.OperationTag,
		"title":         run.Title,
		"keywords":      run.Keywords,
	})
	return frontendDTO(run), nil
}

// ExecuteNativeLearningReplayStep executes one native screen-coordinate replay
// step after the frontend has shown the user a confirmation prompt.
func (a *App) ExecuteNativeLearningReplayStep(payload string) (interface{}, error) {
	if a.nativeInput == nil {
		return nil, fmt.Errorf("native replay executor is not available")
	}
	var step visual_learning.LearningReplayStep
	if err := json.Unmarshal([]byte(payload), &step); err != nil {
		return nil, fmt.Errorf("native replay step: invalid payload: %w", err)
	}
	invocationID := a.hookGeneInvocationID("", "learning-replay")
	const replayGeneSkillID = "learning_replay"
	a.emitHookGeneDataEntered(replayGeneSkillID, invocationID)
	defer a.emitHookGeneCompleted(replayGeneSkillID, invocationID)
	// 錄到的 window handle 可能已失效（macOS 的 CGWindowID 在目標視窗關閉後
	// 即作廢）；先以 handle/process/title 比對目前桌面，必要時重新找回視窗。
	resolvedWin, resolvedOK := a.nativeInput.ResolveWindow(step.WindowHandle, step.WindowProcess, step.WindowTitle)
	if resolvedOK {
		step.WindowHandle = resolvedWin.Handle
	}
	if relocated, ok := a.relocateNativeReplayStep(step); ok {
		a.emitHookGeneDataProcessed(replayGeneSkillID, invocationID)
		if relocated.NeedsConfirmation && canAutoConfirmLowRiskReplay(step, relocated) {
			relocated.NeedsConfirmation = false
			relocated.OK = true
			relocated.Reason = "low-risk replay auto-confirmed for non-dangerous click"
		}
		if relocated.NeedsConfirmation {
			previewStep := step
			previewStep.X = relocated.ExecutionPoint.X
			previewStep.Y = relocated.ExecutionPoint.Y
			preview := visual_learning.NativeReplayResult{OK: false}
			previewError := relocated.Reason
			if relocated.Method != "capture_guard" {
				preview = a.nativeInput.MoveCursorOnly(previewStep)
				if preview.Error != "" {
					previewError = strings.TrimSpace(previewError + "; preview move failed: " + preview.Error)
				}
			}
			a.emitHookGenePaused(replayGeneSkillID, invocationID)
			return frontendDTO(visual_learning.NativeReplayResult{
				OK:                   false,
				Skipped:              true,
				NeedsConfirmation:    true,
				Method:               "visual_relocation",
				Index:                step.Index,
				Label:                step.Label,
				X:                    relocated.ExecutionPoint.X,
				Y:                    relocated.ExecutionPoint.Y,
				OriginalX:            step.X,
				OriginalY:            step.Y,
				Error:                previewError,
				WindowTitle:          step.WindowTitle,
				WindowProcess:        step.WindowProcess,
				ForegroundOK:         preview.OK,
				RelocationMethod:     relocated.Method,
				RelocationConfidence: relocated.Confidence,
				RelocationReason:     relocated.Reason,
				DebugImagePath:       relocated.DebugImagePath,
				DebugInfoPath:        debugInfoPathForImage(relocated.DebugImagePath),
			}), nil
		}
		step.X = relocated.ExecutionPoint.X
		step.Y = relocated.ExecutionPoint.Y
		result := a.nativeInput.Click(step)
		a.emitHookGeneDataLeft(replayGeneSkillID, invocationID, true)
		result.OriginalX = relocated.OriginalPoint.X
		result.OriginalY = relocated.OriginalPoint.Y
		result.Relocated = true
		result.RelocationMethod = relocated.Method
		result.RelocationConfidence = relocated.Confidence
		result.RelocationReason = relocated.Reason
		result.DebugImagePath = relocated.DebugImagePath
		result.DebugInfoPath = debugInfoPathForImage(relocated.DebugImagePath)
		return frontendDTO(result), nil
	}
	// 無視覺重定位可用：至少把錄製座標重映射到視窗目前的位置/大小，
	// 避免視窗移動後盲點舊座標。
	if resolvedOK {
		step = adjustNativeStepToWindowRect(step, resolvedWin.Rect)
	}
	result := a.nativeInput.Click(step)
	a.emitHookGeneDataLeft(replayGeneSkillID, invocationID, true)
	return frontendDTO(result), nil
}

func (a *App) relocateNativeReplayStep(step visual_learning.LearningReplayStep) (visual_learning.AnchorRelocationResult, bool) {
	if a == nil || a.nativeInput == nil || step.WindowsAnchor == nil || step.WindowHandle == 0 {
		return visual_learning.AnchorRelocationResult{}, false
	}
	// anchor 座標存在「錄製當下截圖」的 pixel 空間；優先用 anchor 自記的影像
	// 尺寸。拿 point 空間的 WindowRect 來當分母在 Retina 上會差 2 倍，導致
	// 重定位整個錯位（舊 trace 沒有 image_width 時仍退回 WindowRect）。
	recordedWidth := step.WindowsAnchor.ImageWidth
	recordedHeight := step.WindowsAnchor.ImageHeight
	if recordedWidth <= 0 || recordedHeight <= 0 {
		recordedWidth = step.WindowRect.W
		recordedHeight = step.WindowRect.H
	}
	if recordedWidth <= 0 || recordedHeight <= 0 {
		if step.Viewport != nil {
			recordedWidth = step.Viewport.Width
			recordedHeight = step.Viewport.Height
		}
	}
	if recordedWidth <= 0 || recordedHeight <= 0 {
		return visual_learning.AnchorRelocationResult{}, false
	}
	capture, err := a.nativeInput.CaptureWindow(step.WindowHandle)
	if err != nil || capture.Width <= 0 || capture.Height <= 0 || len(capture.ImageData) == 0 {
		return visual_learning.AnchorRelocationResult{}, false
	}
	if isSuspiciousReplayCapture(capture) {
		relocated := visual_learning.AnchorRelocationResult{
			NeedsConfirmation: true,
			Method:            "capture_guard",
			Reason:            fmt.Sprintf("captured target window is suspiciously small (%dx%d); replay stopped before native click because the stored window handle may point to a browser chrome/strip instead of page content", capture.Width, capture.Height),
			OriginalPoint:     visual_learning.PixelPoint{X: step.X, Y: step.Y},
			ExecutionPoint:    visual_learning.PixelPoint{X: step.X, Y: step.Y},
		}
		if debugPath := a.saveReplayRelocationDebugOverlay(step, capture, relocated); debugPath != "" {
			relocated.DebugImagePath = debugPath
		}
		return relocated, true
	}
	detection := a.yoloDetector.Detect(capture.ImageData, capture.Width, capture.Height)
	shape := a.opencvPipeline.Propose(capture.ImageData, capture.Width, capture.Height)
	relocated := visual_learning.ResolveAnchorRelocation(
		step.WindowsAnchor,
		recordedWidth,
		recordedHeight,
		capture.Width,
		capture.Height,
		detection,
		shape,
		visual_learning.AnchorRelocationOptions{ConfidenceThreshold: 0.5, CurrentImageData: capture.ImageData},
	)
	if relocated.OK || relocated.NeedsConfirmation || relocated.Confidence > 0 {
		if debugPath := a.saveReplayRelocationDebugOverlay(step, capture, relocated); debugPath != "" {
			relocated.DebugImagePath = debugPath
		}
		windowPoint := relocated.ExecutionPoint
		captureScale := capture.PixelScale()
		relocated.ExecutionPoint = visual_learning.PixelPoint{
			X: capture.WindowRect.X + int(math.Round(float64(windowPoint.X)/captureScale)),
			Y: capture.WindowRect.Y + int(math.Round(float64(windowPoint.Y)/captureScale)),
		}
	}
	relocated.OriginalPoint = visual_learning.PixelPoint{X: step.X, Y: step.Y}
	if relocated.OK || relocated.NeedsConfirmation {
		return relocated, true
	}
	fallback, ok := fallbackScaledWindowAnchorRelocation(step, capture, recordedWidth, recordedHeight, relocated.Reason)
	if ok {
		fallback.Candidates = relocated.Candidates
		if debugPath := a.saveReplayRelocationDebugOverlay(step, capture, fallback); debugPath != "" {
			fallback.DebugImagePath = debugPath
		}
	}
	return fallback, ok
}

func isSuspiciousReplayCapture(capture visual_learning.WindowCapture) bool {
	return capture.Width < 240 || capture.Height < 160
}

func fallbackScaledWindowAnchorRelocation(step visual_learning.LearningReplayStep, capture visual_learning.WindowCapture, recordedWidth, recordedHeight int, baseReason string) (visual_learning.AnchorRelocationResult, bool) {
	anchor := step.WindowsAnchor
	if anchor == nil || !anchor.OK || recordedWidth <= 0 || recordedHeight <= 0 || capture.Width <= 0 || capture.Height <= 0 {
		return visual_learning.AnchorRelocationResult{}, false
	}
	if strings.EqualFold(anchor.DetectorBackend, "recorded") {
		return visual_learning.AnchorRelocationResult{}, false
	}
	recordedPoint := anchor.ExecutionPoint
	if recordedPoint.X == 0 && recordedPoint.Y == 0 {
		recordedPoint = anchor.Click
	}
	if recordedPoint.X < 0 || recordedPoint.Y < 0 || recordedPoint.X > recordedWidth || recordedPoint.Y > recordedHeight {
		return visual_learning.AnchorRelocationResult{}, false
	}
	scaledX := int(math.Round(float64(recordedPoint.X) / float64(recordedWidth) * float64(capture.Width)))
	scaledY := int(math.Round(float64(recordedPoint.Y) / float64(recordedHeight) * float64(capture.Height)))
	windowPoint := clampReplayWindowPoint(visual_learning.PixelPoint{X: scaledX, Y: scaledY}, capture.Width, capture.Height)
	scaledBox := scaleReplayAnchorBBox(anchor.AnchorBBox, recordedWidth, recordedHeight, capture.Width, capture.Height)
	reason := strings.TrimSpace(baseReason)
	if reason == "" {
		reason = "YOLO/OpenCV did not produce a relocation candidate"
	}
	captureScale := capture.PixelScale()
	return visual_learning.AnchorRelocationResult{
		OK:            true,
		Method:        "scaled_anchor_relocation",
		Reason:        reason + "; using recorded visual anchor ratio inside the current resized window",
		Confidence:    0.45,
		OriginalPoint: visual_learning.PixelPoint{X: step.X, Y: step.Y},
		ExecutionPoint: visual_learning.PixelPoint{
			X: capture.WindowRect.X + int(math.Round(float64(windowPoint.X)/captureScale)),
			Y: capture.WindowRect.Y + int(math.Round(float64(windowPoint.Y)/captureScale)),
		},
		AnchorBBox: scaledBox,
	}, true
}

func scaleReplayAnchorBBox(box visual_learning.PixelBBox, recordedWidth, recordedHeight, currentWidth, currentHeight int) visual_learning.PixelBBox {
	if box.W <= 0 || box.H <= 0 || recordedWidth <= 0 || recordedHeight <= 0 || currentWidth <= 0 || currentHeight <= 0 {
		return visual_learning.PixelBBox{}
	}
	x := int(math.Round(float64(box.X) / float64(recordedWidth) * float64(currentWidth)))
	y := int(math.Round(float64(box.Y) / float64(recordedHeight) * float64(currentHeight)))
	w := int(math.Round(float64(box.W) / float64(recordedWidth) * float64(currentWidth)))
	h := int(math.Round(float64(box.H) / float64(recordedHeight) * float64(currentHeight)))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= currentWidth {
		x = currentWidth - 1
	}
	if y >= currentHeight {
		y = currentHeight - 1
	}
	if x+w > currentWidth {
		w = currentWidth - x
	}
	if y+h > currentHeight {
		h = currentHeight - y
	}
	return visual_learning.PixelBBox{X: x, Y: y, W: w, H: h}
}

func clampReplayWindowPoint(point visual_learning.PixelPoint, width, height int) visual_learning.PixelPoint {
	if width <= 0 || height <= 0 {
		return visual_learning.PixelPoint{}
	}
	if point.X < 0 {
		point.X = 0
	}
	if point.Y < 0 {
		point.Y = 0
	}
	if point.X >= width {
		point.X = width - 1
	}
	if point.Y >= height {
		point.Y = height - 1
	}
	return point
}

func (a *App) saveReplayRelocationDebugOverlay(step visual_learning.LearningReplayStep, capture visual_learning.WindowCapture, relocated visual_learning.AnchorRelocationResult) string {
	debugDir := filepath.Join(storage.ProjectRoot(appDataRoot(), "default"), "data", "visual_learning", "replay_debug")
	name := fmt.Sprintf("%s-step-%02d.png", time.Now().Format("20060102-150405.000"), step.Index)
	path := filepath.Join(debugDir, name)
	overlayResult := relocationResultForWindowOverlay(relocated, capture)
	if err := visual_learning.SaveAnchorRelocationDebugOverlay(path, capture.ImageData, capture.Width, capture.Height, overlayResult); err != nil {
		log.Printf("visual relocation debug overlay failed: %v", err)
		return ""
	}
	if err := saveReplayRelocationDebugInfo(debugInfoPathForImage(path), step, capture, relocated); err != nil {
		log.Printf("visual relocation debug info failed: %v", err)
	}
	log.Printf("visual relocation debug overlay: %s", path)
	return path
}

func relocationResultForWindowOverlay(result visual_learning.AnchorRelocationResult, capture visual_learning.WindowCapture) visual_learning.AnchorRelocationResult {
	if capture.Width <= 0 || capture.Height <= 0 {
		return result
	}
	// 螢幕座標（point）落在視窗範圍內時，轉成截圖（pixel）座標再畫 overlay。
	scale := capture.PixelScale()
	winW := int(math.Round(float64(capture.Width) / scale))
	winH := int(math.Round(float64(capture.Height) / scale))
	if result.ExecutionPoint.X >= capture.WindowRect.X && result.ExecutionPoint.X < capture.WindowRect.X+winW &&
		result.ExecutionPoint.Y >= capture.WindowRect.Y && result.ExecutionPoint.Y < capture.WindowRect.Y+winH {
		result.ExecutionPoint = visual_learning.PixelPoint{
			X: int(math.Round(float64(result.ExecutionPoint.X-capture.WindowRect.X) * scale)),
			Y: int(math.Round(float64(result.ExecutionPoint.Y-capture.WindowRect.Y) * scale)),
		}
	}
	return result
}

func saveReplayRelocationDebugInfo(path string, step visual_learning.LearningReplayStep, capture visual_learning.WindowCapture, relocated visual_learning.AnchorRelocationResult) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"step_index":         step.Index,
		"method":             relocated.Method,
		"reason":             relocated.Reason,
		"confidence":         relocated.Confidence,
		"needs_confirmation": relocated.NeedsConfirmation,
		"ok":                 relocated.OK,
		"original_point":     relocated.OriginalPoint,
		"execution_point":    relocated.ExecutionPoint,
		"anchor_bbox":        relocated.AnchorBBox,
		"candidate_count":    len(relocated.Candidates),
		"candidates":         relocated.Candidates,
		"capture": map[string]interface{}{
			"width":          capture.Width,
			"height":         capture.Height,
			"window_rect":    capture.WindowRect,
			"window_title":   capture.WindowTitle,
			"window_process": capture.WindowProcess,
			"suspicious":     isSuspiciousReplayCapture(capture),
		},
		"recorded": map[string]interface{}{
			"x":              step.X,
			"y":              step.Y,
			"window_title":   step.WindowTitle,
			"window_process": step.WindowProcess,
			"window_handle":  step.WindowHandle,
			"window_rect":    step.WindowRect,
			"viewport":       step.Viewport,
			"anchor":         step.WindowsAnchor,
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func canAutoConfirmLowRiskReplay(step visual_learning.LearningReplayStep, relocated visual_learning.AnchorRelocationResult) bool {
	if relocated.Confidence < 0.5 {
		return false
	}
	processPath := strings.ReplaceAll(strings.TrimSpace(step.WindowProcess), "\\", "/")
	process := strings.ToLower(filepath.Base(processPath))
	switch process {
	case "chrome.exe", "msedge.exe", "firefox.exe", "brave.exe", "opera.exe", "vivaldi.exe":
	// macOS 的 process 是 app 名稱而非 .exe（CGWindowList 的 kCGWindowOwnerName）。
	case "google chrome", "google chrome beta", "microsoft edge", "firefox", "safari", "brave browser", "opera", "vivaldi", "arc":
	case "explorer.exe":
	default:
		return false
	}
	text := strings.ToLower(strings.Join([]string{
		step.WindowTitle,
		step.Label,
		step.Role,
		step.Tag,
		step.Summary,
		relocated.Reason,
	}, " "))
	for _, word := range []string{
		"download", "downloads", "save as", "settings", "system settings", "chrome settings",
		"delete", "remove", "rename", "properties", "recycle bin", "format", "eject",
		"submit", "pay", "payment", "purchase", "checkout", "transfer",
		"下載", "另存", "設定", "系統設定", "刪除", "移除", "重新命名", "內容", "資源回收桶", "格式化", "退出",
		"送出", "提交", "付款", "購買", "結帳", "轉帳",
	} {
		if strings.Contains(text, word) {
			return false
		}
	}
	return true
}

// ResolveWindowsClickAnchor creates a compact Windows visual anchor for one
// recorded click. It runs YOLOX/OpenCV locally, chooses a bbox by Windows
// learning-mode rules, and returns only the small crop/structure needed for
// later CLI candidate matching. OCR is deliberately optional and not invoked
// here, so replay does not depend on text recognition.
func (a *App) ResolveWindowsClickAnchor(imageData []byte, width, height, clickX, clickY int, optionsJSON string) (visual_learning.WindowsClickAnchorResult, error) {
	var opts visual_learning.WindowsClickAnchorOptions
	if strings.TrimSpace(optionsJSON) != "" {
		if err := json.Unmarshal([]byte(optionsJSON), &opts); err != nil {
			return visual_learning.WindowsClickAnchorResult{}, fmt.Errorf("windows click anchor: invalid options JSON: %w", err)
		}
	}
	detection := a.yoloDetector.Detect(imageData, width, height)
	shape := a.opencvPipeline.Propose(imageData, width, height)
	return visual_learning.ResolveWindowsClickAnchor(imageData, width, height, clickX, clickY, detection, shape, opts)
}

// ExportVisualLearning 執行安全匯出（僅允許項目通過，禁止項目被阻擋）。
func (a *App) ExportVisualLearning(sectionsJSON string) (interface{}, error) {
	var sections []string
	if err := json.Unmarshal([]byte(sectionsJSON), &sections); err != nil {
		return nil, fmt.Errorf("invalid sections: %w", err)
	}
	manifest, err := a.vlSafeExporter.Export(sections)
	return frontendDTO(manifest), err
}
