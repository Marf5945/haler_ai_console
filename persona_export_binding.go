package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type NativePersonaDragExportResult struct {
	Status           string      `json:"status"`
	ExportPath       string      `json:"export_path"`
	LandedPath       string      `json:"landed_path"`
	Platform         string      `json:"platform"`
	FallbackRequired bool        `json:"fallback_required"`
	Message          string      `json:"message"`
	PersonaID        string      `json:"persona_id"`
	DisplayName      string      `json:"display_name"`
	DropTargetKind   string      `json:"drop_target_kind"`
	DropTargetDir    string      `json:"drop_target_dir"`
	State            interface{} `json:"state,omitempty"`
}

func (a *App) SelectPersonaExportDirectory() (string, error) {
	home, _ := os.UserHomeDir()
	startDir := filepath.Join(home, "Desktop")
	if a.ctx == nil {
		return startDir, nil
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "選擇人格匯出位置",
		DefaultDirectory: startDir,
	})
}

func (a *App) ExportPersonaHandler(personaID, mode, destDir string) (interface{}, error) {
	// Remove mode exports first; system data changes only after the JSON is safe.
	exportPath, err := a.settingsService.ExportPersona(personaID, destDir)
	if err != nil {
		return nil, err
	}

	removed := false
	state := a.settingsService.State()
	if mode == "export_remove" {
		next, err := a.settingsService.RemovePersona(personaID)
		if err != nil {
			return nil, err
		}
		state = next
		removed = true
	}

	return map[string]interface{}{
		"persona_id":  personaID,
		"export_path": exportPath,
		"removed":     removed,
		"state":       state,
	}, nil
}

func (a *App) NativeDragExportPersonaHandler(personaID, mode string) (*NativePersonaDragExportResult, error) {
	tempRoot := filepath.Join(os.TempDir(), "ai-console-persona-export")
	if err := os.MkdirAll(tempRoot, 0o700); err != nil {
		return nil, fmt.Errorf("建立暫存人格匯出目錄失敗: %w", err)
	}

	exportPath, err := a.settingsService.ExportPersona(personaID, tempRoot)
	if err != nil {
		return nil, err
	}

	displayName := personaID
	state := a.settingsService.State()
	for _, persona := range state.Personas {
		if persona.ID == personaID {
			displayName = persona.Name
			break
		}
	}

	dragResult := startNativeFileDrag(exportPath)
	out := &NativePersonaDragExportResult{
		Status:           dragResult.Status,
		ExportPath:       exportPath,
		LandedPath:       dragResult.LandedPath,
		Platform:         runtime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		PersonaID:        personaID,
		DisplayName:      displayName,
		DropTargetKind:   dragResult.DropTargetKind,
		DropTargetDir:    dragResult.DropTargetDir,
	}
	if dragResult.Status != nativeDragStatusSuccess {
		_ = os.Remove(exportPath)
	}
	return out, nil
}

func (a *App) FinalizeNativePersonaExport(action, personaID, tempExportPath, landedPath string) (interface{}, error) {
	state := a.settingsService.State()
	removed := false

	switch action {
	case "remove":
		next, err := a.settingsService.RemovePersona(personaID)
		if err != nil {
			return nil, err
		}
		state = next
		removed = true
	case "copy":
		// Finder already received the promised file. Keep it and only clean temp.
	case "cancel":
		if landedPath != "" {
			// SEC-W02（2026-05-24）：刪除 landed persona 匯出檔之前先做 6 條驗證，
			// 避免被前端任意路徑誘導刪掉非 persona 的檔案。inline 寫，不抽 helper。
			info, err := os.Stat(landedPath)
			if err != nil {
				return nil, fmt.Errorf("persona cancel: stat landed: %w", err)
			}
			if info.IsDir() {
				return nil, fmt.Errorf("persona cancel: landed path is a directory, refused")
			}
			if filepath.Ext(landedPath) != ".json" {
				return nil, fmt.Errorf("persona cancel: landed extension must be .json")
			}
			if tempExportPath != "" && filepath.Base(landedPath) != filepath.Base(tempExportPath) {
				return nil, fmt.Errorf("persona cancel: basename mismatch (landed=%q, temp=%q)",
					filepath.Base(landedPath), filepath.Base(tempExportPath))
			}
			data, err := os.ReadFile(landedPath)
			if err != nil {
				return nil, fmt.Errorf("persona cancel: read landed: %w", err)
			}
			var probe struct {
				Schema string `json:"schema"`
				ID     string `json:"id"`
			}
			if err := json.Unmarshal(data, &probe); err != nil {
				return nil, fmt.Errorf("persona cancel: parse landed JSON: %w", err)
			}
			if probe.Schema != "ai-console.persona.v1" {
				return nil, fmt.Errorf("persona cancel: schema mismatch (got %q)", probe.Schema)
			}
			if probe.ID != personaID {
				return nil, fmt.Errorf("persona cancel: personaID mismatch (file=%q, req=%q)", probe.ID, personaID)
			}
			// 通過全部 6 條驗證 → os.Remove（單檔，不要 RemoveAll）
			if err := os.Remove(landedPath); err != nil {
				return nil, fmt.Errorf("persona cancel: remove landed: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("未知的 native 人格匯出動作: %s", action)
	}

	if tempExportPath != "" {
		_ = os.Remove(tempExportPath)
	}

	return map[string]interface{}{
		"persona_id":  personaID,
		"export_path": landedPath,
		"removed":     removed,
		"state":       state,
	}, nil
}
