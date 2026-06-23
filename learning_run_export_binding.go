package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"ui_console/adapter/visual_learning"
)

// 讓「錄製（操作示範 / learning run）」擁有與 skill 相同的生命週期：
// 可打包拖到桌面、拖回來被辨識安裝、拖出去時跳出 移除／複製／取消。
// 結構刻意比照 go_program_authoring_catalog_binding.go 的匯出/匯入流程。

const learningRunExportManifestFile = "learning_run_export_manifest.json"
const learningRunExportType = "learning_run"

type LearningRunExportManifest struct {
	ExportType string `json:"export_type"`
	ExportedAt string `json:"exported_at"`
	RunID      string `json:"run_id"`
	Tag        string `json:"tag,omitempty"`
	Title      string `json:"title,omitempty"`
	StepCount  int    `json:"step_count,omitempty"`
}

// LearningRunExportPreview 給前端的安裝預覽（拖回安裝時用）。
type LearningRunExportPreview struct {
	ExportType string `json:"export_type"`
	RunID      string `json:"run_id"`
	Tag        string `json:"tag,omitempty"`
	Title      string `json:"title,omitempty"`
	StepCount  int    `json:"step_count,omitempty"`
}

type NativeLearningRunDragExportResult struct {
	Status           string `json:"status"`
	ExportDir        string `json:"export_dir"`
	LandedPath       string `json:"landed_path"`
	Platform         string `json:"platform"`
	FallbackRequired bool   `json:"fallback_required"`
	Message          string `json:"message"`
	RunID            string `json:"run_id"`
	Title            string `json:"title"`
	DropTargetKind   string `json:"drop_target_kind"`
	DropTargetDir    string `json:"drop_target_dir"`
}

// NativeDragExportLearningRun 把一筆錄製打包，並啟動原生拖曳到桌面。
func (a *App) NativeDragExportLearningRun(runID string) (*NativeLearningRunDragExportResult, error) {
	if a == nil || a.learningService == nil {
		return nil, fmt.Errorf("learning run export: 錄製服務尚未就緒")
	}
	run, err := a.learningService.GetRun(strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	tempRoot := filepath.Join(os.TempDir(), "ai-console-learning-run-export")
	if err := os.MkdirAll(tempRoot, 0o700); err != nil {
		return nil, fmt.Errorf("建立錄製匯出暫存目錄失敗: %w", err)
	}
	// 唯一暫存父層放奈秒時間戳，落地資料夾只用錄製標題命名（與 skill 匯出一致）。
	parentDir := filepath.Join(tempRoot, fmt.Sprintf("export-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return nil, fmt.Errorf("建立錄製匯出暫存目錄失敗: %w", err)
	}
	exportDir, err := packLearningRunExport(parentDir, a.learningService.RunDir(run.ID), run)
	if err != nil {
		_ = os.RemoveAll(parentDir)
		return nil, err
	}
	dragResult := startNativeFileDrag(exportDir)
	out := &NativeLearningRunDragExportResult{
		Status:           dragResult.Status,
		ExportDir:        exportDir,
		LandedPath:       dragResult.LandedPath,
		Platform:         runtime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		RunID:            run.ID,
		Title:            firstNonEmpty(run.Title, run.Name, run.Tag, run.ID),
		DropTargetKind:   dragResult.DropTargetKind,
		DropTargetDir:    dragResult.DropTargetDir,
	}
	if dragResult.Status != nativeDragStatusSuccess {
		_ = os.RemoveAll(parentDir)
	} else if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "learningrun:native_completed", out)
	}
	return out, nil
}

// FinalizeNativeLearningRunExport 處理拖出成功後的 移除／複製／取消。
//   - cancel：刪掉剛剛拖到桌面的資料夾
//   - remove：複製保留在桌面，並把本機這筆錄製刪除
//   - copy / ""：保留本機錄製
func (a *App) FinalizeNativeLearningRunExport(action, runID, tempExportDir, landedPath string) error {
	if a == nil || a.learningService == nil {
		return fmt.Errorf("learning run export: 錄製服務尚未就緒")
	}
	action = strings.TrimSpace(action)
	if strings.TrimSpace(landedPath) != "" {
		if err := validateLearningRunExport(landedPath, runID); err != nil {
			return err
		}
	}
	switch action {
	case "cancel":
		if strings.TrimSpace(landedPath) != "" {
			if err := os.RemoveAll(landedPath); err != nil {
				return fmt.Errorf("移除錄製匯出資料夾失敗: %w", err)
			}
		}
	case "remove":
		if strings.TrimSpace(landedPath) == "" {
			return fmt.Errorf("learning run export: remove 前必須先成功落地")
		}
		if err := a.learningService.DeleteRun(strings.TrimSpace(runID)); err != nil {
			return err
		}
		if a.eventBus != nil {
			a.eventBus.Emit("learningrun:catalog_updated", nil)
		}
	case "copy", "":
		// 保留本機錄製，不動。
	default:
		return fmt.Errorf("learning run export: 未知動作 %q", action)
	}
	if strings.TrimSpace(tempExportDir) != "" {
		_ = os.RemoveAll(filepath.Dir(tempExportDir))
	}
	return nil
}

// DeleteLearningRun 直接刪除一筆錄製（不必走拖曳流程）。
func (a *App) DeleteLearningRun(runID string) error {
	if a == nil || a.learningService == nil {
		return fmt.Errorf("learning run export: 錄製服務尚未就緒")
	}
	if err := a.learningService.DeleteRun(strings.TrimSpace(runID)); err != nil {
		return err
	}
	if a.eventBus != nil {
		a.eventBus.Emit("learningrun:catalog_updated", nil)
	}
	return nil
}

// PreviewLearningRunExport 驗證並讀取匯出包，給前端做安裝預覽。
func (a *App) PreviewLearningRunExport(path string) (*LearningRunExportPreview, error) {
	if err := validateLearningRunExport(path, ""); err != nil {
		return nil, err
	}
	var manifest LearningRunExportManifest
	if err := readJSONFile(filepath.Join(filepath.Clean(path), learningRunExportManifestFile), &manifest); err != nil {
		return nil, err
	}
	return &LearningRunExportPreview{
		ExportType: manifest.ExportType,
		RunID:      manifest.RunID,
		Tag:        manifest.Tag,
		Title:      manifest.Title,
		StepCount:  manifest.StepCount,
	}, nil
}

// ImportLearningRunExport 把匯出資料夾安裝回錄製清單（同 run id 覆蓋＝restore 語意）。
func (a *App) ImportLearningRunExport(path string) (*LearningRunExportPreview, error) {
	if a == nil || a.learningService == nil {
		return nil, fmt.Errorf("learning run export: 錄製服務尚未就緒")
	}
	if err := validateLearningRunExport(path, ""); err != nil {
		return nil, err
	}
	clean := filepath.Clean(path)
	var manifest LearningRunExportManifest
	if err := readJSONFile(filepath.Join(clean, learningRunExportManifestFile), &manifest); err != nil {
		return nil, err
	}
	src := filepath.Join(clean, "run")
	if _, err := os.Stat(filepath.Join(src, "run.json")); err != nil {
		return nil, fmt.Errorf("learning run export: 匯出包缺少 run/run.json，無法安裝")
	}
	runID, err := a.learningService.ImportRunDir(src)
	if err != nil {
		return nil, err
	}
	if a.eventBus != nil {
		a.eventBus.Emit("learningrun:catalog_updated", nil)
	}
	return &LearningRunExportPreview{
		ExportType: manifest.ExportType,
		RunID:      runID,
		Tag:        manifest.Tag,
		Title:      manifest.Title,
		StepCount:  manifest.StepCount,
	}, nil
}

func packLearningRunExport(parentDir, runDir string, run *visual_learning.LearningRun) (string, error) {
	if run == nil {
		return "", fmt.Errorf("learning run export: 缺少錄製資料")
	}
	title := firstNonEmpty(run.Title, run.Name, run.Tag, run.ID)
	exportDir := filepath.Join(parentDir, safeSkillFolderName(title))
	if err := copySubExportDirectory(runDir, filepath.Join(exportDir, "run")); err != nil {
		return "", err
	}
	manifest := LearningRunExportManifest{
		ExportType: learningRunExportType,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		RunID:      run.ID,
		Tag:        run.Tag,
		Title:      title,
		StepCount:  run.StepCount,
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(exportDir, learningRunExportManifestFile), data, 0o600); err != nil {
		return "", err
	}
	readme := "# " + title + "\n\n" +
		"這是 AI Console 的錄製（操作示範）匯出資料夾。\n\n" +
		"- run/: run.json 與加密操作 trace\n" +
		"- " + learningRunExportManifestFile + ": 匯出索引\n"
	if err := os.WriteFile(filepath.Join(exportDir, "README.md"), []byte(readme), 0o600); err != nil {
		return "", err
	}
	return exportDir, nil
}

func validateLearningRunExport(path, expectedRunID string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("learning run export: 路徑為空")
	}
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("learning run export: 不是資料夾")
	}
	var manifest LearningRunExportManifest
	if err := readJSONFile(filepath.Join(clean, learningRunExportManifestFile), &manifest); err != nil {
		return err
	}
	if manifest.ExportType != learningRunExportType {
		return fmt.Errorf("learning run export: 匯出類型不符")
	}
	if expectedRunID != "" && manifest.RunID != strings.TrimSpace(expectedRunID) {
		return fmt.Errorf("learning run export: run id 不符")
	}
	return nil
}
