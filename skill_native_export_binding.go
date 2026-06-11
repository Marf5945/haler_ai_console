package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"ui_console/orchestration/skill_step"
)

type NativeSkillDragExportResult struct {
	Status           string `json:"status"`
	ExportDir        string `json:"export_dir"`
	LandedPath       string `json:"landed_path"`
	Platform         string `json:"platform"`
	FallbackRequired bool   `json:"fallback_required"`
	Message          string `json:"message"`
	SkillID          string `json:"skill_id"`
	DisplayName      string `json:"display_name"`
	DropTargetKind   string `json:"drop_target_kind"`
	DropTargetDir    string `json:"drop_target_dir"`
}

func (a *App) NativeDragExportSkill(skillID string) (*NativeSkillDragExportResult, error) {
	manifest, skillDir, err := a.findArchivedSkillForNativeExport(skillID)
	if err != nil {
		return nil, err
	}
	tempRoot := filepath.Join(os.TempDir(), "ai-console-skill-export")
	if err := os.MkdirAll(tempRoot, 0o700); err != nil {
		return nil, fmt.Errorf("建立 skill 匯出暫存目錄失敗: %w", err)
	}
	// 每次匯出放進一個唯一的暫存父層（含奈秒時間戳），但真正被拖出去、
	// 落在桌面的資料夾只用 skill 的顯示名稱命名，使用者一眼就看得懂是哪個 skill。
	parentDir := filepath.Join(tempRoot, fmt.Sprintf("export-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return nil, fmt.Errorf("建立 skill 匯出暫存目錄失敗: %w", err)
	}
	displayName := firstNonEmpty(manifest.DisplayName, manifest.SkillID)
	exportDir := filepath.Join(parentDir, safeSkillFolderName(displayName))
	if _, err := skill_step.ExportSkill(skillDir, exportDir); err != nil {
		_ = os.RemoveAll(parentDir)
		return nil, err
	}
	dragResult := startNativeFileDrag(exportDir)
	out := &NativeSkillDragExportResult{
		Status:           dragResult.Status,
		ExportDir:        exportDir,
		LandedPath:       dragResult.LandedPath,
		Platform:         runtime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		SkillID:          manifest.SkillID,
		DisplayName:      firstNonEmpty(manifest.DisplayName, manifest.SkillID),
		DropTargetKind:   dragResult.DropTargetKind,
		DropTargetDir:    dragResult.DropTargetDir,
	}
	if dragResult.Status != nativeDragStatusSuccess {
		_ = os.RemoveAll(parentDir)
	} else if a != nil && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "skill:native_completed", out)
	}
	return out, nil
}

func (a *App) FinalizeNativeSkillExport(action, skillID, tempExportDir, landedPath string) error {
	action = strings.TrimSpace(action)
	if strings.TrimSpace(landedPath) != "" {
		if err := validateLandedSkillExport(landedPath, skillID); err != nil {
			return err
		}
	}
	switch action {
	case "cancel":
		if strings.TrimSpace(landedPath) != "" {
			if err := os.RemoveAll(landedPath); err != nil {
				return fmt.Errorf("移除 skill 匯出資料夾失敗: %w", err)
			}
		}
	case "remove":
		_, skillDir, err := a.findArchivedSkillForNativeExport(skillID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(landedPath) == "" {
			return fmt.Errorf("skill export: landed path is required before removing archived skill")
		}
		if err := os.RemoveAll(skillDir); err != nil {
			return fmt.Errorf("移除已歸檔 skill 失敗: %w", err)
		}
		if a != nil && a.toolsService != nil {
			a.toolsService.RemoveTool("skill:" + strings.TrimSpace(skillID))
		}
		if a != nil && a.eventBus != nil {
			a.eventBus.Emit("tools:list_changed", nil)
		}
	case "copy", "":
		// Keep the archived skill installed.
	default:
		return fmt.Errorf("skill export: unknown action %q", action)
	}
	if strings.TrimSpace(tempExportDir) != "" {
		_ = os.RemoveAll(tempExportDir)
	}
	return nil
}

func (a *App) findArchivedSkillForNativeExport(skillID string) (*skill_step.SkillManifest, string, error) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return nil, "", fmt.Errorf("skill export: skill_id is required")
	}
	if filepath.Base(skillID) != skillID || strings.Contains(skillID, "..") {
		return nil, "", fmt.Errorf("skill export: invalid skill_id %q", skillID)
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return nil, "", err
	}
	for i := range manifests {
		if manifests[i].SkillID != skillID {
			continue
		}
		skillDir := filepath.Join(appDataRoot(), "data", "skills", skillID)
		if _, statErr := os.Stat(filepath.Join(skillDir, "skill_manifest.json")); statErr != nil {
			return nil, "", fmt.Errorf("skill export: archived skill missing manifest: %w", statErr)
		}
		return &manifests[i], skillDir, nil
	}
	return nil, "", fmt.Errorf("skill export: archived skill %q not found", skillID)
}

func validateLandedSkillExport(landedPath, expectedSkillID string) error {
	expectedSkillID = strings.TrimSpace(expectedSkillID)
	info, err := os.Stat(landedPath)
	if err != nil {
		return fmt.Errorf("skill export: stat landed: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill export: landed path is not a directory: %s", landedPath)
	}
	manifest, err := skill_step.LoadManifest(filepath.Join(landedPath, "skill_manifest.json"))
	if err != nil {
		return fmt.Errorf("skill export: landed manifest invalid: %w", err)
	}
	if manifest.SkillID != expectedSkillID {
		return fmt.Errorf("skill export: landed skill mismatch (landed=%q, expected=%q)", manifest.SkillID, expectedSkillID)
	}
	if _, err := os.Stat(filepath.Join(landedPath, "export_manifest.json")); err != nil {
		return fmt.Errorf("skill export: landed export manifest missing: %w", err)
	}
	return nil
}

// safeSkillFolderName 把 skill 顯示名稱轉成可當資料夾名稱的字串：
// 保留中文等非 ASCII 文字，只替換掉路徑分隔字元、控制字元，
// 以及 Windows 不允許出現在檔名的字元。
// 這樣拖到桌面的資料夾會直接叫「產出電料Bom」，而不是一串隨機編號。
func safeSkillFolderName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "skill"
	}
	const illegal = `<>:"/\|?*`
	var b strings.Builder
	for _, r := range value {
		switch {
		case r < 0x20:
			b.WriteRune('-')
		case strings.ContainsRune(illegal, r):
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(strings.TrimSpace(b.String()), ".-")
	if out == "" {
		return "skill"
	}
	if runes := []rune(out); len(runes) > 80 {
		out = string(runes[:80])
	}
	return out
}
