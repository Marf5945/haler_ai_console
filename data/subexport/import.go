// subexport/import.go — Sub 匯入安裝（§31.5）。
// 讀取 install_manifest.json，產生新碼，複製檔案，回報工具衝突。
package subexport

import (
	"fmt"
	"os"
	"path/filepath"
)

// ──────────────────────────────────────────────
// 匯入結果
// ──────────────────────────────────────────────

// ToolConflict 工具衝突記錄。
type ToolConflict struct {
	OriginalID string `json:"original_id"` // 工具 ID
	Type       string `json:"type"`        // skill, mcp, app
	ExportPath string `json:"export_path"` // 匯出資料夾內路徑
	SystemPath string `json:"system_path"` // 系統內已存在路徑
}

// ImportResult 匯入結果。
type ImportResult struct {
	NewSystemCode string         `json:"new_system_code"` // 新產生的系統碼
	SubDir        string         `json:"sub_dir"`         // 安裝的 sub 目錄
	ToolConflicts []ToolConflict `json:"tool_conflicts"`  // 需要使用者決定的工具衝突
	InstalledTools []string      `json:"installed_tools"`  // 已自動安裝的工具
}

// ──────────────────────────────────────────────
// 匯入流程
// ──────────────────────────────────────────────

// ImportSub 從匯出資料夾匯入 sub。
// 步驟:
// 1. 讀取 manifest
// 2. 產生新碼
// 3. 建立 sub 目錄
// 4. 複製 memory/dag/tool_history
// 5. 檢查工具衝突（不自動覆蓋，回傳衝突清單）
func ImportSub(exportDir, projectRoot string) (*ImportResult, error) {
	// 讀取 manifest
	manifest, err := LoadManifest(exportDir)
	if err != nil {
		return nil, fmt.Errorf("讀取匯出 manifest 失敗: %w", err)
	}

	// 產生新碼（匯入時一律重新產碼）
	newCode, err := GenerateSystemCode(manifest.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("產生新碼失敗: %w", err)
	}

	// 建立 sub 目錄
	subBase := filepath.Join(projectRoot, "subagents", "callable", newCode)
	for _, d := range []string{"memory", "dag", "tool_history"} {
		if err := os.MkdirAll(filepath.Join(subBase, d), 0755); err != nil {
			return nil, fmt.Errorf("建立目錄失敗: %w", err)
		}
	}

	// 複製 memory/
	if err := copyDirContents(filepath.Join(exportDir, "memory"), filepath.Join(subBase, "memory")); err != nil {
		return nil, fmt.Errorf("複製 memory 失敗: %w", err)
	}

	// 複製 dag/
	if err := copyDirContents(filepath.Join(exportDir, "dag"), filepath.Join(subBase, "dag")); err != nil {
		return nil, fmt.Errorf("複製 dag 失敗: %w", err)
	}

	// 複製 tool_history/
	if err := copyDirContents(filepath.Join(exportDir, "tool_history"), filepath.Join(subBase, "tool_history")); err != nil {
		return nil, fmt.Errorf("複製 tool_history 失敗: %w", err)
	}

	// 檢查工具衝突
	var conflicts []ToolConflict
	var installed []string

	for _, tool := range manifest.Files.Tools {
		systemToolPath := resolveToolSystemPath(projectRoot, tool.Type, tool.OriginalID)
		exportToolPath := filepath.Join(exportDir, tool.Path)

		if _, err := os.Stat(systemToolPath); err == nil {
			// 工具已存在 → 記錄衝突，等使用者決定
			conflicts = append(conflicts, ToolConflict{
				OriginalID: tool.OriginalID,
				Type:       tool.Type,
				ExportPath: exportToolPath,
				SystemPath: systemToolPath,
			})
		} else {
			// 工具不存在 → 自動安裝
			if err := os.MkdirAll(systemToolPath, 0755); err != nil {
				return nil, fmt.Errorf("建立工具目錄失敗: %w", err)
			}
			if err := copyDirContents(exportToolPath, systemToolPath); err != nil {
				return nil, fmt.Errorf("安裝工具 %s 失敗: %w", tool.OriginalID, err)
			}
			installed = append(installed, tool.OriginalID)
		}
	}

	// 寫入 delegation_log
	if err := WriteDelegationLog(projectRoot, "import_install", newCode, newCode); err != nil {
		// 非致命錯誤，記錄但不中斷
		fmt.Fprintf(os.Stderr, "警告: 寫入 delegation_log 失敗: %v\n", err)
	}

	return &ImportResult{
		NewSystemCode:  newCode,
		SubDir:         subBase,
		ToolConflicts:  conflicts,
		InstalledTools: installed,
	}, nil
}

// ResolveToolConflict 解決單一工具衝突（覆蓋現有工具）。
func ResolveToolConflict(conflict ToolConflict) error {
	// 刪除現有後複製匯出的版本
	if err := os.RemoveAll(conflict.SystemPath); err != nil {
		return fmt.Errorf("刪除現有工具失敗: %w", err)
	}
	if err := os.MkdirAll(conflict.SystemPath, 0755); err != nil {
		return fmt.Errorf("建立工具目錄失敗: %w", err)
	}
	return copyDirContents(conflict.ExportPath, conflict.SystemPath)
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

// resolveToolSystemPath 依工具類型解析系統內路徑。
func resolveToolSystemPath(projectRoot, toolType, originalID string) string {
	// 依照 §17 存放結構: data/projects/{project}/tools/{type}s/{id}/
	// 這裡假設 projectRoot 即為 project 目錄
	return filepath.Join(projectRoot, "tools", toolType+"s", originalID)
}
