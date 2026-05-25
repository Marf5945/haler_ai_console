// sub_export_binding.go — §31 Sub Export/Import + Tab Order Wails 綁定。
//
// 前端呼叫路徑：
//
//	ExportSubHandler(subID, displayName, mode, destDir, toolsJSON) → 匯出/複製
//	ImportSubHandler(exportDir) → 匯入安裝（回傳衝突清單）
//	ResolveImportToolConflict(conflictJSON) → 解決工具衝突
//	GetTabOrder() → 取得 tab 排序狀態
//	AppendTabOrder(systemCode) → 新增 sub 到排序末尾
//	RemoveTabOrder(systemCode) → 移除 sub 排序
//	ReorderTabs(newOrderJSON) → 批次更新 sub 排序
//
// v4.0 — 對應 Spec §31。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"ui_console/data/storage"
	"ui_console/data/subexport"
	"ui_console/shared/taborder"
)

// ──────────────────────────────────────────────
// 懶載入 taborder Manager（避免 App struct 過大）
// ──────────────────────────────────────────────

func (a *App) getTabOrderManager() *taborder.Manager {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	return taborder.NewManager(projectRoot)
}

// ──────────────────────────────────────────────
// Export 綁定
// ──────────────────────────────────────────────

// ExportSubHandler 匯出 sub（移除或複製）。
// mode: "export_remove" 或 "export_copy"
// toolsJSON: ManifestTool 陣列的 JSON 字串（可為 "[]"）
// validSubID SEC-14: subID 只允許英數、底線、連字號。
var validSubID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (a *App) ExportSubHandler(subID, displayName, mode, destDir, toolsJSON string) (*subexport.ExportResult, error) {
	// SEC-14: binding 層 regex 快速拒絕非法 subID
	if !validSubID.MatchString(subID) {
		return nil, fmt.Errorf("subID 格式不合法: %q", subID)
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")

	// 解析工具清單
	var tools []subexport.ToolRef
	if toolsJSON != "" && toolsJSON != "[]" {
		if err := json.Unmarshal([]byte(toolsJSON), &tools); err != nil {
			return nil, fmt.Errorf("解析工具清單失敗: %w", err)
		}
	}

	exportMode := subexport.ExportCopy
	if mode == "export_remove" {
		exportMode = subexport.ExportRemove
	}

	result, err := subexport.PackExport(subexport.ExportOptions{
		ProjectRoot:    projectRoot,
		SubID:          subID,
		DisplayName:    displayName,
		Mode:           exportMode,
		DestDir:        destDir,
		ConnectedTools: tools,
	})
	if err != nil {
		return nil, err
	}

	// 寫入 delegation_log
	subexport.WriteDelegationLog(projectRoot, string(exportMode), subID, result.NewSystemCode)

	// 移除模式：刪除原始 sub + 更新 tab order
	if exportMode == subexport.ExportRemove {
		if err := subexport.RemoveSubFromSystem(projectRoot, subID); err != nil {
			log.Printf("[EXPORT] 移除原始 sub 失敗: %v", err)
		}
		mgr := a.getTabOrderManager()
		if err := mgr.Remove(subID); err != nil {
			log.Printf("[EXPORT] 移除 tab order 失敗: %v", err)
		}
	}

	log.Printf("[EXPORT] 匯出完成: mode=%s sub=%s new_code=%s", mode, subID, result.NewSystemCode)
	return result, nil
}

// ──────────────────────────────────────────────
// Import 綁定
// ──────────────────────────────────────────────

// ImportSubResult 匯入結果（前端用）。
type ImportSubResult struct {
	NewSystemCode  string                   `json:"new_system_code"`
	SubDir         string                   `json:"sub_dir"`
	ToolConflicts  []subexport.ToolConflict `json:"tool_conflicts"`
	InstalledTools []string                 `json:"installed_tools"`
}

type SubPackagePreview struct {
	ExportDir        string                   `json:"export_dir"`
	DisplayName      string                   `json:"display_name"`
	SourceSystemCode string                   `json:"source_system_code"`
	ToolCount        int                      `json:"tool_count"`
	Tools            []subexport.ManifestTool `json:"tools"`
}

type NativeSubDragExportResult struct {
	Status           string `json:"status"`
	ExportDir        string `json:"export_dir"`
	LandedPath       string `json:"landed_path"`
	Platform         string `json:"platform"`
	FallbackRequired bool   `json:"fallback_required"`
	Message          string `json:"message"`
	SubID            string `json:"sub_id"`
	DisplayName      string `json:"display_name"`
	NewSystemCode    string `json:"new_system_code"`
}

type ConflictResolutionRequest struct {
	Strategy  string                   `json:"strategy"`
	Conflicts []subexport.ToolConflict `json:"conflicts"`
}

// NativeDragExportSubHandler 先匯出到暫存區，再交給平台原生拖曳層。
func (a *App) NativeDragExportSubHandler(subID, displayName, mode, toolsJSON string) (*NativeSubDragExportResult, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	tools, err := parseSubExportTools(toolsJSON)
	if err != nil {
		return nil, err
	}

	tempRoot := filepath.Join(os.TempDir(), "ai-console-export")
	if err := os.MkdirAll(tempRoot, 0o700); err != nil { // SEC-15: 限制暫存目錄權限
		return nil, fmt.Errorf("建立暫存匯出目錄失敗: %w", err)
	}

	// 原生拖曳前不可刪原 sub，所以暫存打包永遠用 copy。
	result, err := subexport.PackExport(subexport.ExportOptions{
		ProjectRoot:    projectRoot,
		SubID:          subID,
		DisplayName:    displayName,
		Mode:           subexport.ExportCopy,
		DestDir:        tempRoot,
		ConnectedTools: tools,
	})
	if err != nil {
		return nil, err
	}

	dragResult := startNativeFileDrag(result.ExportDir)
	out := &NativeSubDragExportResult{
		Status:           dragResult.Status,
		ExportDir:        result.ExportDir,
		LandedPath:       dragResult.LandedPath,
		Platform:         runtime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		SubID:            subID,
		DisplayName:      displayName,
		NewSystemCode:    result.NewSystemCode,
	}

	if dragResult.Status != nativeDragStatusSuccess {
		_ = os.RemoveAll(result.ExportDir)
	} else if a.ctx != nil {
		// Native drag starts outside React's normal drag lifecycle; emit a
		// completion event so the UI can show the final action dialog reliably.
		wailsruntime.EventsEmit(a.ctx, "subexport:native_completed", out)
	}

	return out, nil
}

func (a *App) FinalizeNativeSubExport(action, subID, tempExportDir, landedPath, newSystemCode string) error {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	switch action {
	case "remove":
		subexport.WriteDelegationLog(projectRoot, string(subexport.ExportRemove), subID, newSystemCode)
		if err := removeSubAfterExport(a, projectRoot, subID); err != nil {
			return err
		}
	case "copy":
		subexport.WriteDelegationLog(projectRoot, string(subexport.ExportCopy), subID, newSystemCode)
	case "cancel":
		// SEC-W08：把 newSystemCode 傳下去做 manifest 交叉驗證
		if err := removeLandedSubExport(landedPath, newSystemCode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("未知的 native sub 匯出動作: %s", action)
	}
	if tempExportDir != "" {
		_ = os.RemoveAll(tempExportDir)
	}
	return nil
}

func (a *App) GetSubExportDesktopDirectory() (string, error) {
	return defaultSubExportDirectory()
}

func (a *App) GetSubExportFallbackDirectory() (string, error) {
	if runtime.GOOS == "darwin" {
		if dir, err := frontFinderDirectory(); err == nil && dir != "" {
			return dir, nil
		}
	}
	return defaultSubExportDirectory()
}

func defaultSubExportDirectory() (string, error) {
	home, _ := os.UserHomeDir()
	startDir := filepath.Join(home, "Desktop")
	return startDir, nil
}

func frontFinderDirectory() (string, error) {
	script := `tell application "Finder"
if (count of Finder windows) > 0 then
POSIX path of (target of front Finder window as alias)
else
POSIX path of (desktop as alias)
end if
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *App) SelectSubExportDirectory() (string, error) {
	startDir, err := defaultSubExportDirectory()
	if err != nil {
		return "", err
	}
	if a.ctx == nil {
		return startDir, nil
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "選擇 Sub 匯出位置",
		DefaultDirectory: startDir,
	})
}

func (a *App) PreviewSubPackage(exportDir string) (*SubPackagePreview, error) {
	exportDir = normalizePackageDropPath(exportDir)
	manifest, err := subexport.LoadManifest(exportDir)
	if err != nil {
		return nil, err
	}
	return &SubPackagePreview{
		ExportDir:        exportDir,
		DisplayName:      manifest.DisplayName,
		SourceSystemCode: manifest.SourceSystemCode,
		ToolCount:        len(manifest.Files.Tools),
		Tools:            manifest.Files.Tools,
	}, nil
}

// ImportSubHandler 匯入 sub 匯出包。
// 回傳工具衝突清單（前端顯示對話框讓使用者決定）。
func (a *App) ImportSubHandler(exportDir string) (*ImportSubResult, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	exportDir = normalizePackageDropPath(exportDir)

	result, err := subexport.ImportSub(exportDir, projectRoot)
	if err != nil {
		return nil, err
	}

	// 新增到 tab order
	mgr := a.getTabOrderManager()
	if err := mgr.Append(result.NewSystemCode); err != nil {
		log.Printf("[IMPORT] 新增 tab order 失敗: %v", err)
	}

	log.Printf("[IMPORT] 匯入完成: new_code=%s conflicts=%d installed=%d",
		result.NewSystemCode, len(result.ToolConflicts), len(result.InstalledTools))

	return &ImportSubResult{
		NewSystemCode:  result.NewSystemCode,
		SubDir:         result.SubDir,
		ToolConflicts:  result.ToolConflicts,
		InstalledTools: result.InstalledTools,
	}, nil
}

func normalizePackageDropPath(path string) string {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return filepath.Dir(path)
	}
	return path
}

func parseSubExportTools(toolsJSON string) ([]subexport.ToolRef, error) {
	var tools []subexport.ToolRef
	if toolsJSON != "" && toolsJSON != "[]" {
		if err := json.Unmarshal([]byte(toolsJSON), &tools); err != nil {
			return nil, fmt.Errorf("解析工具清單失敗: %w", err)
		}
	}
	return tools, nil
}

func exportModeFromString(mode string) subexport.ExportMode {
	if mode == "export_remove" {
		return subexport.ExportRemove
	}
	return subexport.ExportCopy
}

func removeSubAfterExport(a *App, projectRoot, subID string) error {
	if err := subexport.RemoveSubFromSystem(projectRoot, subID); err != nil {
		log.Printf("[EXPORT] 移除原始 sub 失敗: %v", err)
		return err
	}
	mgr := a.getTabOrderManager()
	if err := mgr.Remove(subID); err != nil {
		log.Printf("[EXPORT] 移除 tab order 失敗: %v", err)
		return err
	}
	return nil
}

// removeLandedSubExport 在使用者取消 native drag 後清理 landed sub export folder。
//
// SEC-W08（2026-05-24）：landedPath 來自前端，必須在 RemoveAll 前做 5 條驗證
// 才能確認這真的是這次匯出的 sub folder，而不是被誘導刪掉的任意目錄。
// 驗證項目：
//   1. landedPath 是目錄（不是檔案）
//   2. basename == expectedSystemCode（與 caller 傳入比對）或 basename 含 "_SUB_"
//   3. install_manifest.json 存在且可解析
//   4. manifest.ExportType == "sub_handler"
//   5. manifest.SourceSystemCode == expectedSystemCode
//
// 不抽跨檔 helper、不動 data/subexport/*、不動 orchestration/dag/*。
func removeLandedSubExport(path, expectedSystemCode string) error {
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 已不存在，視為成功
		}
		return fmt.Errorf("sub cancel: stat landed: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("sub cancel: landed path is not a directory: %s", path)
	}
	base := filepath.Base(path)
	if expectedSystemCode != "" && base != expectedSystemCode {
		return fmt.Errorf("sub cancel: basename mismatch (landed=%q, expected=%q)", base, expectedSystemCode)
	}
	if !strings.Contains(base, "_SUB_") {
		return fmt.Errorf("sub cancel: basename %q does not contain _SUB_ marker", base)
	}
	m, err := subexport.LoadManifest(path)
	if err != nil {
		return fmt.Errorf("sub cancel: load manifest: %w", err)
	}
	if m.ExportType != "sub_handler" {
		return fmt.Errorf("sub cancel: manifest export_type=%q, expected %q", m.ExportType, "sub_handler")
	}
	if expectedSystemCode != "" && m.SourceSystemCode != expectedSystemCode {
		return fmt.Errorf("sub cancel: manifest source_system_code=%q, expected %q",
			m.SourceSystemCode, expectedSystemCode)
	}
	return os.RemoveAll(path)
}

// ResolveImportToolConflict 解決單一工具衝突（覆蓋現有版本）。
func (a *App) ResolveImportToolConflict(conflictJSON string) error {
	var conflict subexport.ToolConflict
	if err := json.Unmarshal([]byte(conflictJSON), &conflict); err != nil {
		return fmt.Errorf("解析衝突資料失敗: %w", err)
	}
	return subexport.ResolveToolConflict(conflict)
}

func (a *App) ResolveImportToolConflicts(requestJSON string) error {
	var req ConflictResolutionRequest
	if err := json.Unmarshal([]byte(requestJSON), &req); err != nil {
		return fmt.Errorf("解析衝突策略失敗: %w", err)
	}
	if req.Strategy != "overwrite_all" {
		return nil
	}
	for _, conflict := range req.Conflicts {
		if err := subexport.ResolveToolConflict(conflict); err != nil {
			return err
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// Tab Order 綁定
// ──────────────────────────────────────────────

// GetTabOrder 取得當前 tab 排序狀態。
func (a *App) GetTabOrder() taborder.TabOrder {
	mgr := a.getTabOrderManager()
	return mgr.GetOrder()
}

// AppendTabOrder 將新 sub 附加到 tab 排序末尾。
func (a *App) AppendTabOrder(systemCode string) error {
	mgr := a.getTabOrderManager()
	return mgr.Append(systemCode)
}

// RemoveTabOrder 從 tab 排序中移除指定 sub。
func (a *App) RemoveTabOrder(systemCode string) error {
	mgr := a.getTabOrderManager()
	return mgr.Remove(systemCode)
}

// ReorderTabs 批次更新 sub 排序（前端拖曳重排後呼叫）。
// newOrderJSON: string 陣列的 JSON（system codes 依序排列）。
func (a *App) ReorderTabs(newOrderJSON string) error {
	var newOrder []string
	if err := json.Unmarshal([]byte(newOrderJSON), &newOrder); err != nil {
		return fmt.Errorf("解析排序資料失敗: %w", err)
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	newOrder = resolveSubagentOrder(projectRoot, newOrder)
	mgr := a.getTabOrderManager()
	return mgr.Reorder(newOrder)
}
