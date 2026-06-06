package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type NativeReferenceFileDragResult struct {
	Status           string `json:"status"`
	SourcePath       string `json:"source_path"`
	LandedPath       string `json:"landed_path"`
	Platform         string `json:"platform"`
	FallbackRequired bool   `json:"fallback_required"`
	Message          string `json:"message"`
	DisplayName      string `json:"display_name"`
	DropTargetKind   string `json:"drop_target_kind"`
	DropTargetDir    string `json:"drop_target_dir"`
}

func (a *App) NativeDragExportReferenceFile(sourcePath string) (*NativeReferenceFileDragResult, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("reference: source path is empty")
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("reference: folders are not supported")
	}

	dragResult := startNativeFileDrag(sourcePath)
	out := &NativeReferenceFileDragResult{
		Status:           dragResult.Status,
		SourcePath:       sourcePath,
		LandedPath:       dragResult.LandedPath,
		Platform:         runtime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		DisplayName:      filepath.Base(sourcePath),
		DropTargetKind:   dragResult.DropTargetKind,
		DropTargetDir:    dragResult.DropTargetDir,
	}
	if dragResult.Status == nativeDragStatusSuccess && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "reference:native_completed", out)
	}
	return out, nil
}

func (a *App) FinalizeNativeReferenceFileExport(action, sourcePath, landedPath string) error {
	switch action {
	case "remove":
		// SEC-W02 同類修補（2026-05-24）：sourcePath 來自前端，刪之前先驗 file（非目錄）。
		if sourcePath == "" {
			return fmt.Errorf("reference: source path is empty")
		}
		info, err := os.Stat(sourcePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // 已不存在，視為成功
			}
			return fmt.Errorf("reference remove: stat: %w", err)
		}
		if info.IsDir() {
			return fmt.Errorf("reference remove: source is a directory, refused")
		}
		if err := os.Remove(sourcePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reference remove: remove: %w", err)
		}
	case "copy":
		// Finder already received the promised file. Keep both copies.
	case "cancel":
		if landedPath != "" {
			// SEC-W02 同類修補（2026-05-24）：landedPath 來自前端，
			// reference 是單檔 → 用 os.Remove 而非 RemoveAll，且驗 basename 與 source 一致。
			info, err := os.Stat(landedPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("reference cancel: stat landed: %w", err)
			}
			if info.IsDir() {
				return fmt.Errorf("reference cancel: landed is a directory, refused")
			}
			if sourcePath != "" && filepath.Base(landedPath) != filepath.Base(sourcePath) {
				return fmt.Errorf("reference cancel: basename mismatch (landed=%q, source=%q)",
					filepath.Base(landedPath), filepath.Base(sourcePath))
			}
			if err := os.Remove(landedPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("reference cancel: remove landed: %w", err)
			}
		}
	default:
		return fmt.Errorf("unknown reference export action: %s", action)
	}
	return nil
}
