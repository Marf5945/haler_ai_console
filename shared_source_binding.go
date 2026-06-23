package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ui_console/adapter/external_link"
	"ui_console/builtin"
)

type sharedSourceInfo struct {
	Path   string
	Label  string
	Kind   string
	Reason string
}

func previewSharedSourcePath(raw string) (external_link.PreviewResult, bool) {
	info, handled, err := classifySharedSourcePath(raw)
	if !handled {
		return external_link.PreviewResult{}, false
	}
	if err != nil {
		return external_link.PreviewResult{
			URL:      expandUserPath(raw),
			LinkType: external_link.LinkUnsupported,
			Valid:    false,
			Reason:   err.Error(),
		}, true
	}
	return external_link.PreviewResult{
		URL:      info.Path,
		LinkType: external_link.LinkSharedSource,
		Valid:    true,
		Reason:   info.Reason,
	}, true
}

func registerSharedSourcePath(raw, label string, service *external_link.Service) (*external_link.ExternalLink, bool, error) {
	info, handled, err := classifySharedSourcePath(raw)
	if !handled {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	if strings.TrimSpace(label) == "" || label == raw {
		label = info.Label
	}
	link, err := service.RegisterSharedSource(info.Path, label)
	return link, true, err
}

func classifySharedSourcePath(raw string) (sharedSourceInfo, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !looksLikeSharedSourcePath(raw) {
		return sharedSourceInfo{}, false, nil
	}
	path := expandUserPath(raw)
	info, err := os.Stat(path)
	if err != nil {
		return sharedSourceInfo{}, true, fmt.Errorf("shared source: path is not readable: %w", err)
	}

	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	label := filepath.Base(path)
	if label == "." || label == string(os.PathSeparator) || strings.TrimSpace(label) == "" {
		label = path
	}

	if info.IsDir() {
		kind := "shared_folder"
		reason := "偵測到共用資料夾；會以文件範圍搜尋其中可讀的 md / Excel / Office / CSV / JSON 等文件。"
		if isObsidianVault(path) {
			kind = "obsidian_vault"
			reason = "偵測到 Obsidian vault；會以文件範圍搜尋其中的 Markdown 與附件文字。"
		}
		return sharedSourceInfo{Path: path, Label: label, Kind: kind, Reason: reason}, true, nil
	}

	if !builtin.IsSearchableFormat(path) {
		return sharedSourceInfo{}, true, fmt.Errorf("shared source: unsupported document format %q", filepath.Ext(path))
	}
	reason := "偵測到共用文檔；會以文件範圍搜尋這份文件內容。"
	if strings.EqualFold(filepath.Ext(path), ".xlsx") {
		reason = "偵測到 Excel 共用文檔；會抽取工作表文字並納入文件搜尋。"
	}
	return sharedSourceInfo{Path: path, Label: label, Kind: "shared_document", Reason: reason}, true, nil
}

func looksLikeSharedSourcePath(path string) bool {
	path = strings.TrimSpace(path)
	return strings.HasPrefix(path, "/") ||
		strings.HasPrefix(path, "~/") ||
		strings.HasPrefix(path, "$HOME/") ||
		strings.HasPrefix(path, `\\`) ||
		len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
}

func isObsidianVault(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".obsidian"))
	return err == nil && info.IsDir()
}
