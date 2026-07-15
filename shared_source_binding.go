package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

// ──────────────────────────────────────────────
// 右欄「共用資料夾」內容清單（只讀第一層，絕不遞迴）
// ──────────────────────────────────────────────

// SharedSourceFile 右欄共用資料夾底下顯示的單一檔案。
type SharedSourceFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Ext     string `json:"ext"`
	ModTime string `json:"mod_time"`
	Size    int64  `json:"size"`
}

// SharedSourceListing 一個已連結 shared_source 的第一層內容。
type SharedSourceListing struct {
	LinkID    string             `json:"link_id"`
	Label     string             `json:"label"`
	Path      string             `json:"path"`
	IsDir     bool               `json:"is_dir"`
	Files     []SharedSourceFile `json:"files"`
	Truncated bool               `json:"truncated,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// sharedSourceListMaxFiles 右欄清單上限；超過就截斷並標 Truncated，
// 避免使用者連到超大資料夾時右欄爆版。
const sharedSourceListMaxFiles = 60

// ListSharedSourceFiles 列出每個已連結共用資料夾「第一層」的可搜尋文件。
// 設計約束：只 os.ReadDir 一層——子資料夾、隱藏檔一律略過，與
// localsearch.Root.TopLevelOnly 的搜尋範圍保持一致。
func (a *App) ListSharedSourceFiles() []SharedSourceListing {
	if a.linkService == nil {
		return []SharedSourceListing{}
	}
	links := a.linkService.ListByType(external_link.LinkSharedSource)
	out := make([]SharedSourceListing, 0, len(links))
	for _, link := range links {
		listing := SharedSourceListing{LinkID: link.ID, Label: link.Label, Path: link.URL}
		info, err := os.Stat(link.URL)
		if err != nil {
			listing.Error = "路徑無法讀取"
			out = append(out, listing)
			continue
		}
		if !info.IsDir() {
			// 單一文件的 shared source：自己就是唯一一筆。
			listing.Files = []SharedSourceFile{newSharedSourceFile(link.URL, info)}
			out = append(out, listing)
			continue
		}
		listing.IsDir = true
		entries, err := os.ReadDir(link.URL)
		if err != nil {
			listing.Error = "資料夾無法讀取"
			out = append(out, listing)
			continue
		}
		type fileWithTime struct {
			file SharedSourceFile
			mod  time.Time
		}
		var files []fileWithTime
		for _, entry := range entries {
			// 只讀這一層：子資料夾與隱藏檔直接略過。
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			path := filepath.Join(link.URL, entry.Name())
			if !builtin.IsSearchableFormat(path) {
				continue
			}
			fi, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, fileWithTime{file: newSharedSourceFile(path, fi), mod: fi.ModTime()})
		}
		sort.SliceStable(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
		if len(files) > sharedSourceListMaxFiles {
			files = files[:sharedSourceListMaxFiles]
			listing.Truncated = true
		}
		listing.Files = make([]SharedSourceFile, 0, len(files))
		for _, f := range files {
			listing.Files = append(listing.Files, f.file)
		}
		out = append(out, listing)
	}
	return out
}

func newSharedSourceFile(path string, info os.FileInfo) SharedSourceFile {
	return SharedSourceFile{
		Name:    filepath.Base(path),
		Path:    path,
		Ext:     strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		ModTime: info.ModTime().Format(time.RFC3339),
		Size:    info.Size(),
	}
}
