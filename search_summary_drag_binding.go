package main

// 搜尋摘要小卡「拖出備份」binding。
//
// 與 reference_file_drag_binding.go 同一套機制：前端拖曳成功後開共用
// DragActionModal（這裡只給「複製／取消」兩個 action，不給 remove——
// 搜尋摘要是暫存內容、未入庫，沒有「從庫裡移除」的語意）。
//
// 語意：
//   copy   = 保留 Finder 那份（兩份都在）
//   cancel = 刪掉剛剛拖出的落地檔
//
// 安全性比 reference 更嚴：cancel 時除了驗 basename，再比對內容 sha256，
// 確保刪的是「自己這次拖出的檔」，而不是落地資料夾裡剛好同名的既有檔。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// searchSummaryExportTempRoot 是所有搜尋摘要暫存檔的根目錄名（位於 os.TempDir() 底下）。
const searchSummaryExportTempRoot = "ai-console-search-summary"

// NativeSearchSummaryDragResult 欄位刻意對齊既有 reference/skill 匯出，
// 讓前端能沿用同一個 DragActionModal 與事件處理流程。
type NativeSearchSummaryDragResult struct {
	Status           string `json:"status"`
	TempPath         string `json:"temp_path"`   // 暫存 .md 路徑（拖曳來源；finalize 用來清暫存）
	LandedPath       string `json:"landed_path"` // Finder 落地路徑（cancel 要刪的目標）
	Platform         string `json:"platform"`
	FallbackRequired bool   `json:"fallback_required"`
	Message          string `json:"message"`
	DisplayName      string `json:"display_name"`
	Filename         string `json:"filename"`
	ArtifactID       string `json:"artifact_id"`
	Checksum         string `json:"checksum"` // 暫存 md 的 sha256（hex）；cancel 時比對
	DropTargetKind   string `json:"drop_target_kind"`
	DropTargetDir    string `json:"drop_target_dir"`
}

// NativeDragExportSearchSummary 把一段搜尋摘要 markdown 落成暫存 .md，
// 再以原生拖曳交給 Finder。回傳含 checksum 與暫存路徑，供 finalize 收尾。
func (a *App) NativeDragExportSearchSummary(title, filename, markdown, artifactID string) (*NativeSearchSummaryDragResult, error) {
	if strings.TrimSpace(markdown) == "" {
		return nil, fmt.Errorf("search summary export: markdown is empty")
	}
	safeName := safeSearchSummaryFileName(filename, title)

	// 每次匯出獨立暫存父層（奈秒時間戳），避免並發互相覆蓋；
	// 真正落地的檔名只用乾淨的 safeName，使用者一眼看得懂。
	parentDir := filepath.Join(os.TempDir(), searchSummaryExportTempRoot, fmt.Sprintf("export-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return nil, fmt.Errorf("search summary export: 建立暫存目錄失敗: %w", err)
	}
	tempPath := filepath.Join(parentDir, safeName)

	data := []byte(markdown)
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		_ = os.RemoveAll(parentDir)
		return nil, fmt.Errorf("search summary export: 寫入暫存檔失敗: %w", err)
	}
	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	dragResult := startNativeFileDrag(tempPath)
	out := &NativeSearchSummaryDragResult{
		Status:           dragResult.Status,
		TempPath:         tempPath,
		LandedPath:       dragResult.LandedPath,
		Platform:         runtime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		DisplayName:      firstNonEmpty(strings.TrimSpace(title), safeName),
		Filename:         safeName,
		ArtifactID:       strings.TrimSpace(artifactID),
		Checksum:         checksum,
		DropTargetKind:   dragResult.DropTargetKind,
		DropTargetDir:    dragResult.DropTargetDir,
	}

	// 拖曳沒成功就立刻清暫存；成功才留著等 finalize（copy 留、cancel 刪落地）。
	if dragResult.Status != nativeDragStatusSuccess {
		_ = os.RemoveAll(parentDir)
	} else if a != nil && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "searchsummary:native_completed", out)
	}
	return out, nil
}

// FinalizeNativeSearchSummaryExport 收尾搜尋摘要拖曳。
// copy = 保留落地檔；cancel = 安全刪除落地檔。不論哪種，最後都清掉暫存父層。
func (a *App) FinalizeNativeSearchSummaryExport(action, tempPath, landedPath, checksum string) error {
	action = strings.TrimSpace(action)
	switch action {
	case "copy", "":
		// 保留 Finder 那份，不動落地檔。
	case "cancel":
		if err := removeLandedSearchSummary(landedPath, tempPath, checksum); err != nil {
			return err
		}
	default:
		return fmt.Errorf("search summary export: unknown action %q", action)
	}
	// 暫存只是拖曳來源，無論結果都該清掉（僅限我們自己的暫存根底下）。
	if dir := searchSummaryTempParent(tempPath); dir != "" {
		_ = os.RemoveAll(dir)
	}
	return nil
}

// removeLandedSearchSummary 安全刪除 cancel 的落地檔：
//
//	空路徑或已不存在 → 視為成功（idempotent）
//	目錄            → 拒絕
//	basename 不一致  → 拒絕
//	checksum 不符    → 拒絕（被改過，或撞名到別的既有檔）
func removeLandedSearchSummary(landedPath, tempPath, checksum string) error {
	landedPath = strings.TrimSpace(landedPath)
	if landedPath == "" {
		return nil
	}
	info, err := os.Stat(landedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("search summary cancel: stat landed: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("search summary cancel: landed is a directory, refused")
	}
	// basename 必須和當初拖出的暫存檔一致。
	if base := strings.TrimSpace(filepath.Base(tempPath)); base != "" && base != "." {
		if filepath.Base(landedPath) != base {
			return fmt.Errorf("search summary cancel: basename mismatch (landed=%q, temp=%q)",
				filepath.Base(landedPath), base)
		}
	}
	// 內容雜湊必須對得上，確保刪的是自己這次產生的檔。
	if want := strings.TrimSpace(checksum); want != "" {
		got, err := fileSHA256Hex(landedPath)
		if err != nil {
			return fmt.Errorf("search summary cancel: 讀落地檔失敗: %w", err)
		}
		if !strings.EqualFold(got, want) {
			return fmt.Errorf("search summary cancel: checksum mismatch, refused")
		}
	}
	if err := os.Remove(landedPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("search summary cancel: remove landed: %w", err)
	}
	return nil
}

// searchSummaryTempParent 回傳暫存檔的父層；僅在它確實落在我們的暫存根底下時
// 才回傳，避免 tempPath 被竄改後誤刪到外部目錄。
func searchSummaryTempParent(tempPath string) string {
	tempPath = strings.TrimSpace(tempPath)
	if tempPath == "" {
		return ""
	}
	parent := filepath.Dir(tempPath)
	root := filepath.Join(os.TempDir(), searchSummaryExportTempRoot)
	rel, err := filepath.Rel(root, parent)
	if err != nil || rel == "." || rel == "" || strings.HasPrefix(rel, "..") {
		return "" // 不在我們的暫存根底下 → 安全起見不刪。
	}
	return parent
}

// fileSHA256Hex 算檔案內容的 sha256（hex 小寫）。
func fileSHA256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// safeSearchSummaryFileName 把前端給的檔名/標題收斂成安全的單段 .md 檔名：
// 只取最後一段杜絕路徑穿越、清掉 ..、換掉檔名禁用字元、確保 .md 副檔名。
func safeSearchSummaryFileName(filename, fallbackTitle string) string {
	name := strings.TrimSpace(filename)
	if name == "" {
		name = strings.TrimSpace(fallbackTitle)
	}
	// 只取最後一段路徑，杜絕 ../ 穿越。
	name = filepath.Base(filepath.FromSlash(name))
	name = strings.ReplaceAll(name, "..", "")
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = fmt.Sprintf("search-summary-%d", time.Now().Unix())
	}
	// 換掉跨平台檔名禁用字元（保留中文、英數、- _ .）。
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	name = b.String()
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		name += ".md"
	}
	return name
}
