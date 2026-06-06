// subexport/export.go — Sub 匯出打包（§31.3 移除/複製）。
// 將 sub 的 memory/dag/tool_history/tools 打包至匯出資料夾，
// 產生 install_manifest.json + README_INSTALL.md。
package subexport

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SEC-15: 統一匯出目錄權限
const exportDirPerm = 0o700

// ──────────────────────────────────────────────
// 匯出選項
// ──────────────────────────────────────────────

// ExportMode 匯出模式。
type ExportMode string

const (
	ExportRemove ExportMode = "export_remove" // 移除：匯出後刪除原始 sub
	ExportCopy   ExportMode = "export_copy"   // 複製：匯出但保留原始 sub
)

// ExportOptions 匯出所需參數。
type ExportOptions struct {
	ProjectRoot string     // 專案根目錄
	SubID       string     // 原始 sub 的系統碼
	DisplayName string     // 顯示名稱
	Mode        ExportMode // 匯出模式
	DestDir     string     // 匯出目的地（桌面、資料夾等）

	// 連接的工具清單（type + 系統內路徑 + original_id）
	ConnectedTools []ToolRef
}

// ToolRef 工具參考（匯出時需要的資訊）。
type ToolRef struct {
	Type       string // skill, mcp, app
	SystemPath string // 系統內完整路徑
	OriginalID string // 工具 ID
}

// ExportResult 匯出結果。
type ExportResult struct {
	ExportDir       string // 匯出資料夾完整路徑
	NewSystemCode   string // 匯出用的新系統碼
	ManifestPath    string // install_manifest.json 路徑
	DelegationLogOp string // 寫入 delegation_log 的操作類型
}

// ──────────────────────────────────────────────
// 匯出流程
// ──────────────────────────────────────────────

// PackExport 執行匯出打包。
// 1. 產生新系統碼
// 2. 建立匯出資料夾
// 3. 複製 memory/dag/tool_history
// 4. 複製工具定義
// 5. 產生 manifest + README
func PackExport(opts ExportOptions) (*ExportResult, error) {
	// 產生新碼
	newCode, err := GenerateSystemCode(opts.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("產生匯出碼失敗: %w", err)
	}

	// 匯出資料夾名稱：sub 名稱 + sub 代號，不額外加時間戳或 action suffix。
	exportDir := filepath.Join(opts.DestDir, newCode)

	// 建立匯出目錄結構
	subDirs := []string{"memory", "dag", "tool_history", "tools"}
	for _, d := range subDirs {
		if err := os.MkdirAll(filepath.Join(exportDir, d), exportDirPerm); err != nil {
			return nil, fmt.Errorf("建立匯出目錄失敗: %w", err)
		}
	}

	// SEC-14: data 層路徑邊界檢查
	safeSubID := filepath.Base(opts.SubID)
	if safeSubID == "." || safeSubID == ".." {
		return nil, fmt.Errorf("subID 路徑不合法: %q", opts.SubID)
	}
	subBase := filepath.Join(opts.ProjectRoot, "subagents", "callable", safeSubID)
	callableRoot := filepath.Join(opts.ProjectRoot, "subagents", "callable")
	if rel, err := filepath.Rel(callableRoot, subBase); err != nil || rel != safeSubID {
		return nil, fmt.Errorf("subID 路徑不合法: %q", opts.SubID)
	}

	// 複製 memory/
	if err := copyDirContents(filepath.Join(subBase, "memory"), filepath.Join(exportDir, "memory")); err != nil {
		return nil, fmt.Errorf("複製 memory 失敗: %w", err)
	}

	// 複製 dag/
	if err := copyDirContents(filepath.Join(subBase, "dag"), filepath.Join(exportDir, "dag")); err != nil {
		return nil, fmt.Errorf("複製 dag 失敗: %w", err)
	}

	// 複製 tool_history/
	if err := copyDirContents(filepath.Join(subBase, "tool_history"), filepath.Join(exportDir, "tool_history")); err != nil {
		return nil, fmt.Errorf("複製 tool_history 失敗: %w", err)
	}

	// 複製工具定義（保留原始資料夾結構）
	var manifestTools []ManifestTool
	for _, t := range opts.ConnectedTools {
		relPath := fmt.Sprintf("tools/%ss/%s/", t.Type, t.OriginalID)
		destToolDir := filepath.Join(exportDir, relPath)
		if err := os.MkdirAll(destToolDir, exportDirPerm); err != nil {
			return nil, fmt.Errorf("建立工具目錄失敗: %w", err)
		}
		if err := copyDirContents(t.SystemPath, destToolDir); err != nil {
			return nil, fmt.Errorf("複製工具 %s 失敗: %w", t.OriginalID, err)
		}
		manifestTools = append(manifestTools, ManifestTool{
			Type:       t.Type,
			Path:       relPath,
			OriginalID: t.OriginalID,
		})
	}

	// 產生 manifest
	manifest := NewInstallManifest(opts.DisplayName, newCode, manifestTools)
	if err := SaveManifest(exportDir, manifest); err != nil {
		return nil, fmt.Errorf("寫入 manifest 失敗: %w", err)
	}

	// 產生 README
	if err := SaveReadme(exportDir, manifest); err != nil {
		return nil, fmt.Errorf("寫入 README 失敗: %w", err)
	}

	return &ExportResult{
		ExportDir:       exportDir,
		NewSystemCode:   newCode,
		ManifestPath:    filepath.Join(exportDir, "install_manifest.json"),
		DelegationLogOp: string(opts.Mode),
	}, nil
}

// RemoveSubFromSystem 移除模式下刪除原始 sub（由上層呼叫）。
// 刪除 subagents/callable/[subID]/ 整個目錄。
func RemoveSubFromSystem(projectRoot, subID string) error {
	// SEC-14: 防止路徑穿越，確認在 callable/ 下
	safeID := filepath.Base(subID)
	if safeID == "." || safeID == ".." {
		return fmt.Errorf("subID 路徑不合法: %q", subID)
	}
	subDir := filepath.Join(projectRoot, "subagents", "callable", safeID)
	callableRoot := filepath.Join(projectRoot, "subagents", "callable")
	if rel, err := filepath.Rel(callableRoot, subDir); err != nil || rel != safeID {
		return fmt.Errorf("subID 路徑不合法: %q", subID)
	}
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		return fmt.Errorf("sub 目錄不存在: %s", safeID)
	}
	return os.RemoveAll(subDir)
}

// WriteDelegationLog 寫入 delegation_log.jsonl 記錄。
func WriteDelegationLog(projectRoot string, op string, subID string, newCode string) error {
	logPath := filepath.Join(projectRoot, "main", "delegation_log.jsonl")
	if err := os.MkdirAll(filepath.Dir(logPath), exportDirPerm); err != nil {
		return err
	}

	entry := map[string]string{
		"op":         op,
		"sub_id":     subID,
		"new_code":   newCode,
		"created_at": time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

// ──────────────────────────────────────────────
// 檔案複製輔助
// ──────────────────────────────────────────────

// copyDirContents 遞迴複製目錄內容（來源不存在時跳過）。
func copyDirContents(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 來源不存在，跳過
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s 不是目錄", src)
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 計算相對路徑
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if shouldSkipPortableExportPath(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		dstPath := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(dstPath, exportDirPerm)
		}

		return copyFile(path, dstPath)
	})
}

func shouldSkipPortableExportPath(rel string) bool {
	if rel == "." || rel == "" {
		return false
	}
	normalized := filepath.ToSlash(strings.ToLower(rel))
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		switch part {
		case ".git", ".env", ".env.local", ".env.production", "credentials", "secrets", "secret",
			"tokens", "api_keys", "apikeys", "private", "private_keys", "uploads", "uploaded_files",
			"attachments", "reference_files", "node_modules", "cache", ".cache", "tmp", "temp":
			return true
		}
		if strings.Contains(part, "api_key") ||
			strings.Contains(part, "apikey") ||
			strings.Contains(part, "secret") ||
			strings.Contains(part, "token") ||
			strings.Contains(part, "password") ||
			strings.Contains(part, "credential") ||
			strings.Contains(part, "private_key") {
			return true
		}
	}
	ext := filepath.Ext(normalized)
	switch ext {
	case ".key", ".pem", ".p12", ".pfx":
		return true
	}
	return false
}

// copyFile 複製單一檔案。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
