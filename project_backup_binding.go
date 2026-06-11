// project_backup_binding.go — 單一專案加密備份／還原的 Wails binding。
//
// 使用情境：跨裝置接續對話。匯出一個上鎖的 .aicbak 檔，
// 放任何雲端／通訊軟體傳輸都不怕被讀；另一台裝置匯入後接著工作。
//
// 前端流程建議：
//  1. SelectProjectBackupExportDirectory() 選位置
//  2. ExportProjectBackupHandler(projectID, destDir, password, redact)
//  3. 另一台：SelectProjectBackupFile() → InspectProjectBackupHandler() 預覽
//  4. ImportProjectBackupHandler(path, password, mode)
//     mode: "fail_if_exists"（預設）→ 回 status:"conflict" 時讓使用者選
//     "overwrite" 或 "copy"
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ui_console/data/backup"
)

// SelectProjectBackupExportDirectory 讓使用者選備份檔存放位置。
func (a *App) SelectProjectBackupExportDirectory() (string, error) {
	home, _ := os.UserHomeDir()
	startDir := filepath.Join(home, "Desktop")
	if a.ctx == nil {
		return startDir, nil
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "選擇備份檔存放位置",
		DefaultDirectory: startDir,
	})
}

// SelectProjectBackupFile 讓使用者挑選要匯入的備份檔。
func (a *App) SelectProjectBackupFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app context 尚未就緒")
	}
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "選擇 AI Console 備份檔",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "AI Console 備份 (*.aicbak)", Pattern: "*.aicbak"},
		},
	})
}

// ExportProjectBackupHandler 匯出備份。
// encrypt 對應 UI「加密備份檔」勾選框：
//   - false（預設）→ 不問密碼、直接存檔（password 參數被忽略），
//     簡單保存對話不會有忘記密碼的問題。
//   - true → 必須提供至少 8 字元密碼，產出加密＋防竄改的備份。
//
// projectID 留空時匯出目前的 default 專案。
func (a *App) ExportProjectBackupHandler(projectID, destDir, password string, redact bool, encrypt bool) (map[string]interface{}, error) {
	if strings.TrimSpace(projectID) == "" {
		projectID = "default"
	}
	if strings.TrimSpace(destDir) == "" {
		return nil, fmt.Errorf("請先選擇存放位置")
	}
	if !encrypt {
		password = "" // 未勾選加密：忽略密碼，走明文模式
	} else if strings.TrimSpace(password) == "" {
		return map[string]interface{}{"status": "weak_password", "message": "已勾選加密，請設定至少 8 字元的密碼"}, nil
	}
	filename := fmt.Sprintf("%s-%s.aicbak", projectID, time.Now().Format("20060102-150405"))
	destPath := filepath.Join(destDir, filename)

	res, err := backup.ExportProject(appDataRoot(), projectID, destPath, password, redact)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"status":         "ok",
		"bundle_path":    res.BundlePath,
		"file_count":     res.FileCount,
		"encrypted":      res.Encrypted,
		"redacted":       res.Redacted,
		"redaction_hits": res.RedactionHits,
		"size_bytes":     res.SizeBytes,
	}, nil
}

// IsProjectBackupEncryptedHandler 匯入第一步：判斷選的備份檔要不要問密碼。
// 加密檔 → 前端跳密碼欄；明文檔 → 直接進預覽／匯入。
func (a *App) IsProjectBackupEncryptedHandler(bundlePath string) (map[string]interface{}, error) {
	encrypted, err := backup.IsBundleEncrypted(bundlePath)
	if err != nil {
		return backupErrorResult(err)
	}
	return map[string]interface{}{"status": "ok", "encrypted": encrypted}, nil
}

// InspectProjectBackupHandler 匯入前預覽：驗證密碼並回傳備份描述，不落地檔案。
func (a *App) InspectProjectBackupHandler(bundlePath, password string) (map[string]interface{}, error) {
	m, err := backup.InspectBundle(bundlePath, password)
	if err != nil {
		return backupErrorResult(err)
	}
	return map[string]interface{}{
		"status":     "ok",
		"project_id": m.ProjectID,
		"created_at": m.CreatedAt,
		"redacted":   m.Redacted,
		"file_count": m.FileCount,
	}, nil
}

// ImportProjectBackupHandler 匯入備份。mode 見檔頭說明。
func (a *App) ImportProjectBackupHandler(bundlePath, password, mode string) (map[string]interface{}, error) {
	importMode := backup.ImportMode(strings.TrimSpace(mode))
	switch importMode {
	case backup.ModeOverwrite, backup.ModeCopy:
	default:
		importMode = backup.ModeFailIfExists
	}
	res, err := backup.ImportProject(appDataRoot(), bundlePath, password, importMode)
	if err != nil {
		return backupErrorResult(err)
	}
	return map[string]interface{}{
		"status":      "ok",
		"project_id":  res.ProjectID,
		"restored_as": res.RestoredAs,
		"file_count":  res.FileCount,
		"redacted":    res.Redacted,
		"created_at":  res.CreatedAt,
	}, nil
}

// backupErrorResult 把可預期的錯誤轉成前端好處理的 status，
// 其餘錯誤照常回傳 error。
func backupErrorResult(err error) (map[string]interface{}, error) {
	switch {
	case errors.Is(err, backup.ErrProjectExists):
		return map[string]interface{}{"status": "conflict", "message": "同名專案已存在，請選擇覆蓋或另存"}, nil
	case errors.Is(err, backup.ErrDecryptFailed):
		return map[string]interface{}{"status": "bad_password", "message": "密碼錯誤，或備份檔已被竄改"}, nil
	case errors.Is(err, backup.ErrBadFormat):
		return map[string]interface{}{"status": "bad_file", "message": "不是 AI Console 備份檔，或檔案已損壞"}, nil
	case errors.Is(err, backup.ErrPasswordTooShort):
		return map[string]interface{}{"status": "weak_password", "message": err.Error()}, nil
	default:
		return nil, err
	}
}
