package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"ui_console/data/storage"
	"ui_console/orchestration/go_program"
)

const goProgramAuthoringRunFile = "authoring_run.json"
const goProgramExportManifestFile = "go_program_export_manifest.json"

type GoProgramAuthoringCatalogItem struct {
	RunID             string `json:"run_id"`
	ProgramID         string `json:"program_id"`
	ProgramName       string `json:"program_name"`
	Status            string `json:"status"`
	Purpose           string `json:"purpose,omitempty"`
	WorkspaceDir      string `json:"workspace_dir"`
	AttemptCount      int    `json:"attempt_count"`
	LatestAttemptHash string `json:"latest_attempt_hash,omitempty"`
	PendingSkillID    string `json:"pending_skill_id,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	Message           string `json:"message,omitempty"`
}

type GoProgramAuthoringDetail struct {
	GoProgramAuthoringCatalogItem
	AuthoringPrompt string                     `json:"authoring_prompt,omitempty"`
	ControlSteps    []string                   `json:"control_steps,omitempty"`
	Manifest        *go_program.Manifest       `json:"manifest,omitempty"`
	Attempts        []go_program.AttemptRecord `json:"attempts,omitempty"`
	Exportable      bool                       `json:"exportable"`
}

type GoProgramAuthoringRunMeta struct {
	RunID             string `json:"run_id"`
	ProgramID         string `json:"program_id"`
	ProgramName       string `json:"program_name"`
	Status            string `json:"status"`
	Purpose           string `json:"purpose,omitempty"`
	WorkspaceDir      string `json:"workspace_dir"`
	AttemptCount      int    `json:"attempt_count"`
	LatestAttemptHash string `json:"latest_attempt_hash,omitempty"`
	PendingSkillID    string `json:"pending_skill_id,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	Message           string `json:"message,omitempty"`
}

type NativeGoProgramDragExportResult struct {
	Status           string `json:"status"`
	ExportDir        string `json:"export_dir"`
	LandedPath       string `json:"landed_path"`
	Platform         string `json:"platform"`
	FallbackRequired bool   `json:"fallback_required"`
	Message          string `json:"message"`
	RunID            string `json:"run_id"`
	ProgramID        string `json:"program_id"`
	ProgramName      string `json:"program_name"`
	DropTargetKind   string `json:"drop_target_kind"`
	DropTargetDir    string `json:"drop_target_dir"`
}

type GoProgramAuthoringExportManifest struct {
	ExportType     string `json:"export_type"`
	ExportedAt     string `json:"exported_at"`
	RunID          string `json:"run_id"`
	ProgramID      string `json:"program_id"`
	ProgramName    string `json:"program_name"`
	Status         string `json:"status"`
	AttemptCount   int    `json:"attempt_count"`
	PendingSkillID string `json:"pending_skill_id,omitempty"`
	WorkspaceDir   string `json:"workspace_dir,omitempty"`
}

func (a *App) ListGoProgramAuthoringCatalog(limit int) ([]GoProgramAuthoringCatalogItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	root := goProgramAuthoringRoot()
	entries, err := scanGoProgramAuthoringDetails(root)
	if err != nil {
		return nil, err
	}
	// 已長出 skill 的小程式統一以 skill（✦）顯示，這裡濾掉，避免同一能力在工具列出現兩組。
	if archived := a.archivedSkillIDSet(); len(archived) > 0 {
		kept := entries[:0]
		for _, entry := range entries {
			if entry.PendingSkillID != "" && archived[entry.PendingSkillID] {
				continue
			}
			kept = append(kept, entry)
		}
		entries = kept
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt > entries[j].UpdatedAt
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	items := make([]GoProgramAuthoringCatalogItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entry.GoProgramAuthoringCatalogItem)
	}
	return items, nil
}

func (a *App) GetGoProgramAuthoringDetail(runID string) (*GoProgramAuthoringDetail, error) {
	detail, err := findGoProgramAuthoringDetail(strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (a *App) NativeDragExportGoProgramAuthoring(runID string) (*NativeGoProgramDragExportResult, error) {
	detail, err := findGoProgramAuthoringDetail(strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	tempRoot := filepath.Join(os.TempDir(), "ai-console-go-program-export")
	if err := os.MkdirAll(tempRoot, 0o700); err != nil {
		return nil, fmt.Errorf("建立小程式匯出暫存目錄失敗: %w", err)
	}
	exportDir, err := packGoProgramAuthoringExport(tempRoot, detail)
	if err != nil {
		return nil, err
	}
	dragResult := startNativeFileDrag(exportDir)
	out := &NativeGoProgramDragExportResult{
		Status:           dragResult.Status,
		ExportDir:        exportDir,
		LandedPath:       dragResult.LandedPath,
		Platform:         runtime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		RunID:            detail.RunID,
		ProgramID:        detail.ProgramID,
		ProgramName:      detail.ProgramName,
		DropTargetKind:   dragResult.DropTargetKind,
		DropTargetDir:    dragResult.DropTargetDir,
	}
	if dragResult.Status != nativeDragStatusSuccess {
		_ = os.RemoveAll(exportDir)
	} else if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "goprogram:native_completed", out)
	}
	return out, nil
}

func (a *App) FinalizeNativeGoProgramAuthoringExport(action, runID, tempExportDir, landedPath string) error {
	action = strings.TrimSpace(action)
	if strings.TrimSpace(landedPath) != "" {
		if err := validateGoProgramAuthoringExport(landedPath, runID); err != nil {
			return err
		}
	}
	if action == "cancel" && strings.TrimSpace(landedPath) != "" {
		if err := os.RemoveAll(landedPath); err != nil {
			return fmt.Errorf("移除小程式匯出資料夾失敗: %w", err)
		}
	}
	if action == "remove" {
		detail, err := findGoProgramAuthoringDetail(strings.TrimSpace(runID))
		if err != nil {
			return err
		}
		if strings.TrimSpace(detail.WorkspaceDir) != "" {
			if err := os.RemoveAll(detail.WorkspaceDir); err != nil {
				return fmt.Errorf("移除小程式製作紀錄失敗: %w", err)
			}
		}
	}
	if strings.TrimSpace(tempExportDir) != "" {
		_ = os.RemoveAll(tempExportDir)
	}
	return nil
}

func (a *App) writeGoProgramAuthoringRun(result *GoProgramAuthoringResult) error {
	if result == nil || strings.TrimSpace(result.WorkspaceDir) == "" {
		return nil
	}
	detail, _ := buildGoProgramAuthoringDetail(result.WorkspaceDir)
	meta := GoProgramAuthoringRunMeta{
		RunID:        runIDForGoProgramWorkspace(result.ProgramID, result.WorkspaceDir),
		ProgramID:    result.ProgramID,
		ProgramName:  result.ProgramName,
		Status:       firstNonEmpty(result.Status, "ready"),
		WorkspaceDir: result.WorkspaceDir,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
		Message:      result.Message,
	}
	if result.Manifest != nil {
		meta.Purpose = result.Manifest.Purpose
	}
	if detail != nil {
		meta.CreatedAt = firstNonEmpty(detail.CreatedAt, meta.CreatedAt)
		meta.AttemptCount = detail.AttemptCount
		meta.LatestAttemptHash = detail.LatestAttemptHash
	}
	if len(result.Attempts) > 0 {
		meta.AttemptCount = len(result.Attempts)
		meta.LatestAttemptHash = result.Attempts[len(result.Attempts)-1].Hash
	}
	meta.PendingSkillID = result.PendingSkillID
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(result.WorkspaceDir, goProgramAuthoringRunFile), data, 0o600); err != nil {
		return err
	}
	if a != nil && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "goprogram:authoring_updated", meta)
	}
	return nil
}

// ---------------------------------------------------------------------------
// #44 後續：拖入 go-program「匯出資料夾」的辨識與安裝（restore / 分享回灌）。
// 匯出包結構：go_program_export_manifest.json + README.md + workspace/ + source/。
// 安裝作法：把 workspace/（內含 program_manifest.json / authoring_run.json…）複製進
// catalog run 目錄，使其出現在「工具 > 自動流程」。
// ---------------------------------------------------------------------------

// GoProgramExportPreview 是拖入匯出資料夾時回傳給前端的辨識結果。
type GoProgramExportPreview struct {
	ExportType     string `json:"export_type"`
	ProgramID      string `json:"program_id"`
	ProgramName    string `json:"program_name"`
	RunID          string `json:"run_id"`
	Status         string `json:"status"`
	PendingSkillID string `json:"pending_skill_id,omitempty"`
}

// PreviewGoProgramExport 檢查 path 是否為合法的 go_program_authoring 匯出資料夾。
// 是 → 回傳預覽；否 → 回傳 error（呼叫端據此判斷「不是這個型別」，再去試別種）。
func (a *App) PreviewGoProgramExport(path string) (*GoProgramExportPreview, error) {
	if err := validateGoProgramAuthoringExport(path, ""); err != nil {
		return nil, err
	}
	var manifest GoProgramAuthoringExportManifest
	if err := readJSONFile(filepath.Join(filepath.Clean(path), goProgramExportManifestFile), &manifest); err != nil {
		return nil, err
	}
	return &GoProgramExportPreview{
		ExportType:     manifest.ExportType,
		ProgramID:      manifest.ProgramID,
		ProgramName:    manifest.ProgramName,
		RunID:          manifest.RunID,
		Status:         manifest.Status,
		PendingSkillID: manifest.PendingSkillID,
	}, nil
}

// ImportGoProgramExport 把匯出資料夾安裝進 catalog，使其顯示於「工具 > 自動流程」。
// 同名 programID 採覆蓋（restore 語意）。回傳安裝後的 catalog 項目。
func (a *App) ImportGoProgramExport(path string) (*GoProgramAuthoringCatalogItem, error) {
	if err := validateGoProgramAuthoringExport(path, ""); err != nil {
		return nil, err
	}
	clean := filepath.Clean(path)
	var manifest GoProgramAuthoringExportManifest
	if err := readJSONFile(filepath.Join(clean, goProgramExportManifestFile), &manifest); err != nil {
		return nil, err
	}
	src := filepath.Join(clean, "workspace")
	if _, err := os.Stat(filepath.Join(src, "program_manifest.json")); err != nil {
		return nil, fmt.Errorf("go program export: 匯出包缺少 workspace/program_manifest.json，無法安裝")
	}
	programID := firstNonEmpty(manifest.ProgramID, normalizeGoProgramID(manifest.ProgramName), "go-program")
	dest := filepath.Join(goProgramAuthoringRoot(), programID)
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return nil, err
	}
	_ = os.RemoveAll(dest) // 同名覆蓋
	if err := copySubExportDirectory(src, dest); err != nil {
		return nil, fmt.Errorf("go program export: 複製失敗: %w", err)
	}
	detail, err := buildGoProgramAuthoringDetail(dest)
	if err != nil {
		return nil, fmt.Errorf("go program export: 安裝後讀取失敗: %w", err)
	}
	if a.eventBus != nil {
		a.eventBus.Emit("goprogram:authoring_updated", nil)
	}
	item := detail.GoProgramAuthoringCatalogItem
	return &item, nil
}

func goProgramAuthoringRoot() string {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	return filepath.Join(projectRoot, "data", "go_program_authoring")
}

func scanGoProgramAuthoringDetails(root string) ([]GoProgramAuthoringDetail, error) {
	var details []GoProgramAuthoringDetail
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return details, nil
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() || path == root {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, "program_manifest.json")); err != nil {
			return nil
		}
		detail, err := buildGoProgramAuthoringDetail(path)
		if err == nil && detail != nil {
			details = append(details, *detail)
		}
		return filepath.SkipDir
	})
	return details, err
}

func findGoProgramAuthoringDetail(runID string) (*GoProgramAuthoringDetail, error) {
	if runID == "" {
		return nil, fmt.Errorf("go program authoring: run id is required")
	}
	details, err := scanGoProgramAuthoringDetails(goProgramAuthoringRoot())
	if err != nil {
		return nil, err
	}
	for i := range details {
		if details[i].RunID == runID || details[i].ProgramID == runID {
			return &details[i], nil
		}
	}
	return nil, fmt.Errorf("找不到小程式製作流程: %s", runID)
}

func buildGoProgramAuthoringDetail(workspace string) (*GoProgramAuthoringDetail, error) {
	var manifest go_program.Manifest
	if err := readJSONFile(filepath.Join(workspace, "program_manifest.json"), &manifest); err != nil {
		return nil, err
	}
	attempts := readGoProgramAttempts(filepath.Join(workspace, "attempts"))
	updated := newestMTime(workspace)
	created := inferGoProgramWorkspaceCreatedAt(workspace)
	status := "ready"
	if len(attempts) > 0 {
		status = "authoring"
	}
	var meta GoProgramAuthoringRunMeta
	if err := readJSONFile(filepath.Join(workspace, goProgramAuthoringRunFile), &meta); err == nil {
		status = firstNonEmpty(meta.Status, status)
	}
	steps := readLines(filepath.Join(workspace, "control_steps.txt"))
	prompt := readText(filepath.Join(workspace, "authoring_prompt.txt"))
	programID := firstNonEmpty(meta.ProgramID, manifest.ProgramID, filepath.Base(filepath.Dir(workspace)))
	programName := firstNonEmpty(meta.ProgramName, manifest.DisplayName, programID)
	item := GoProgramAuthoringCatalogItem{
		RunID:        firstNonEmpty(meta.RunID, runIDForGoProgramWorkspace(programID, workspace)),
		ProgramID:    programID,
		ProgramName:  programName,
		Status:       status,
		Purpose:      firstNonEmpty(meta.Purpose, manifest.Purpose),
		WorkspaceDir: workspace,
		AttemptCount: len(attempts),
		CreatedAt:    firstNonEmpty(meta.CreatedAt, created.Format(time.RFC3339)),
		UpdatedAt:    firstNonEmpty(meta.UpdatedAt, updated.Format(time.RFC3339)),
		Message:      meta.Message,
	}
	if len(attempts) > 0 {
		item.LatestAttemptHash = attempts[len(attempts)-1].Hash
	}
	if meta.LatestAttemptHash != "" {
		item.LatestAttemptHash = meta.LatestAttemptHash
	}
	item.PendingSkillID = meta.PendingSkillID
	return &GoProgramAuthoringDetail{
		GoProgramAuthoringCatalogItem: item,
		AuthoringPrompt:               prompt,
		ControlSteps:                  steps,
		Manifest:                      &manifest,
		Attempts:                      attempts,
		Exportable:                    true,
	}, nil
}

func readGoProgramAttempts(root string) []go_program.AttemptRecord {
	var attempts []go_program.AttemptRecord
	matches, _ := filepath.Glob(filepath.Join(root, "attempt-*", "attempt.json"))
	sort.Strings(matches)
	for _, path := range matches {
		var rec go_program.AttemptRecord
		if err := readJSONFile(path, &rec); err == nil {
			attempts = append(attempts, rec)
		}
	}
	sort.Slice(attempts, func(i, j int) bool { return attempts[i].Attempt < attempts[j].Attempt })
	return attempts
}

func packGoProgramAuthoringExport(tempRoot string, detail *GoProgramAuthoringDetail) (string, error) {
	if detail == nil {
		return "", fmt.Errorf("go program export: missing detail")
	}
	base := detail.ProgramID
	if base == "" {
		base = normalizeGoProgramID(detail.ProgramName)
	}
	exportDir := filepath.Join(tempRoot, base+"-authoring")
	if _, err := os.Stat(exportDir); err == nil {
		exportDir = filepath.Join(tempRoot, fmt.Sprintf("%s-authoring-%d", base, time.Now().UnixNano()))
	}
	if err := os.MkdirAll(exportDir, 0o700); err != nil {
		return "", err
	}
	if err := copySubExportDirectory(detail.WorkspaceDir, filepath.Join(exportDir, "workspace")); err != nil {
		return "", err
	}
	if len(detail.Attempts) > 0 {
		best := filepath.Join(detail.WorkspaceDir, "attempts", fmt.Sprintf("attempt-%d", detail.Attempts[len(detail.Attempts)-1].Attempt))
		if err := copySubExportDirectory(best, filepath.Join(exportDir, "source")); err != nil {
			return "", err
		}
	} else {
		source := filepath.Join(detail.WorkspaceDir, "source")
		if _, err := os.Stat(source); err == nil {
			if err := copySubExportDirectory(source, filepath.Join(exportDir, "source")); err != nil {
				return "", err
			}
		}
	}
	manifest := GoProgramAuthoringExportManifest{
		ExportType:     "go_program_authoring",
		ExportedAt:     time.Now().UTC().Format(time.RFC3339),
		RunID:          detail.RunID,
		ProgramID:      detail.ProgramID,
		ProgramName:    detail.ProgramName,
		Status:         detail.Status,
		AttemptCount:   detail.AttemptCount,
		PendingSkillID: detail.PendingSkillID,
		WorkspaceDir:   detail.WorkspaceDir,
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(exportDir, goProgramExportManifestFile), data, 0o600); err != nil {
		return "", err
	}
	readme := "# " + detail.ProgramName + "\n\n" +
		"這是 AI Console 的 Go Program Authoring 匯出資料夾。\n\n" +
		"- workspace/: 原始製作流程、prompt、control steps、attempts\n" +
		"- source/: 最新可檢視 Go 原始碼\n" +
		"- " + goProgramExportManifestFile + ": 匯出索引\n"
	if err := os.WriteFile(filepath.Join(exportDir, "README.md"), []byte(readme), 0o600); err != nil {
		return "", err
	}
	return exportDir, nil
}

func validateGoProgramAuthoringExport(path, expectedRunID string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("go program export: empty path")
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("go program export: not a directory")
	}
	var manifest GoProgramAuthoringExportManifest
	if err := readJSONFile(filepath.Join(clean, goProgramExportManifestFile), &manifest); err != nil {
		return err
	}
	if manifest.ExportType != "go_program_authoring" {
		return fmt.Errorf("go program export: invalid export type")
	}
	if expectedRunID != "" && manifest.RunID != expectedRunID {
		return fmt.Errorf("go program export: run id mismatch")
	}
	return nil
}

func readJSONFile(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func readLines(path string) []string {
	text := strings.TrimSpace(readText(path))
	if text == "" {
		return nil
	}
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func newestMTime(root string) time.Time {
	latest := time.Now().UTC()
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := entry.Info(); err == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest
}

func inferGoProgramWorkspaceCreatedAt(workspace string) time.Time {
	name := filepath.Base(workspace)
	if t, err := time.ParseInLocation("20060102-150405", name, time.Local); err == nil {
		return t.UTC()
	}
	if info, err := os.Stat(workspace); err == nil {
		return info.ModTime().UTC()
	}
	return time.Now().UTC()
}

func runIDForGoProgramWorkspace(programID, workspace string) string {
	return firstNonEmpty(programID, "program") + "_" + filepath.Base(workspace)
}
