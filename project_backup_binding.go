// project_backup_binding.go — 單一專案加密備份／還原的 Wails binding。
//
// ⚠️ 暫時 stub 版本（2026-06-12）：
// data/backup 套件原始碼被 .gitignore 的 /data/backup/ 規則擋住,
// 從未進入版控,目前只存在於 Mac 開發機上。
// 還原步驟（在 Mac 上執行）：
//  1. 編輯 .gitignore,移除或修正 /data/backup/ 規則
//  2. git add -f data/backup && git commit && git push
//  3. git checkout <stub之前的版本> -- project_backup_binding.go 還原本檔
//     （或從 git log 找回原始 binding 內容）
// 在那之前,備份/還原功能會明確回報「暫不可用」,其餘功能不受影響。
//
// 原始使用情境：跨裝置接續對話。匯出一個上鎖的 .aicbak 檔，
// 放任何雲端／通訊軟體傳輸都不怕被讀；另一台裝置匯入後接著工作。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// errBackupUnavailable 是備份功能暫時停用期間所有 handler 的統一回覆。
var errBackupUnavailable = fmt.Errorf("備份功能暫時不可用：data/backup 模組尚未隨此版本提供")

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

// ExportProjectBackupHandler 匯出備份。（stub:暫不可用）
func (a *App) ExportProjectBackupHandler(projectID, destDir, password string, redact bool, encrypt bool) (map[string]interface{}, error) {
	return nil, errBackupUnavailable
}

// IsProjectBackupEncryptedHandler 匯入第一步：判斷備份檔要不要問密碼。（stub:暫不可用）
func (a *App) IsProjectBackupEncryptedHandler(bundlePath string) (map[string]interface{}, error) {
	return nil, errBackupUnavailable
}

// InspectProjectBackupHandler 匯入前預覽。（stub:暫不可用）
func (a *App) InspectProjectBackupHandler(bundlePath, password string) (map[string]interface{}, error) {
	return nil, errBackupUnavailable
}

// ImportProjectBackupHandler 匯入備份。（stub:暫不可用）
func (a *App) ImportProjectBackupHandler(bundlePath, password, mode string) (map[string]interface{}, error) {
	return nil, errBackupUnavailable
}
