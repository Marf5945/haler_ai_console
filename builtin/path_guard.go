// path_guard.go — 路徑安全守衛。
// 雙區模型：Zone A (project 內) 自由讀寫；Zone B (外部) 需使用者確認。
// 拒絕已知危險路徑，防止 .. 穿越和 symlink 逃逸。
package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PathGuard 管理檔案路徑的安全驗證。
type PathGuard struct {
	projectRoot string // project 資料夾的絕對路徑（Zone A 的根）
}

// NewPathGuard 建立路徑守衛。projectRoot 必須是絕對路徑。
func NewPathGuard(projectRoot string) *PathGuard {
	return &PathGuard{projectRoot: projectRoot}
}

// IsProjectInternal 判斷 path 是否在 project 資料夾內（Zone A）。
// 用 filepath.Abs + strings.HasPrefix 判定，先解析 symlink。
func (g *PathGuard) IsProjectInternal(path string) bool {
	resolved, err := resolveAndAbs(path)
	if err != nil {
		return false // 解析失敗視為外部
	}
	root, err := resolveAndAbs(g.projectRoot)
	if err != nil {
		return false
	}
	return strings.HasPrefix(resolved, root+string(filepath.Separator)) || resolved == root
}

// ValidateImportPath 驗證匯入來源路徑是否安全。
// 檢查：路徑存在、非目錄、非系統目錄、無 .. 穿越。
func (g *PathGuard) ValidateImportPath(path string) error {
	// 基本檢查：不能為空
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path_guard: 路徑不可為空")
	}

	// 拒絕 .. 穿越（raw 字串層級檢查，在 Abs 之前）
	if containsDotDot(path) {
		return fmt.Errorf("path_guard: 拒絕含 .. 的路徑: %s", path)
	}

	// 解析為絕對路徑 + 解析 symlink
	resolved, err := resolveAndAbs(path)
	if err != nil {
		return fmt.Errorf("path_guard: 無法解析路徑: %w", err)
	}

	// 檢查檔案存在且不是目錄
	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("path_guard: 檔案不存在: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path_guard: 預期是檔案但收到目錄: %s", resolved)
	}

	// 拒絕系統目錄內的檔案
	if isSystemPath(resolved) {
		return fmt.Errorf("path_guard: 拒絕系統目錄內的檔案: %s", resolved)
	}

	return nil
}

// ValidateExportPath 驗證匯出目標路徑是否安全。
// Zone A 內直接通過；Zone B 僅驗證路徑安全性（呼叫端需另行觸發使用者確認）。
func (g *PathGuard) ValidateExportPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path_guard: 匯出路徑不可為空")
	}
	if containsDotDot(path) {
		return fmt.Errorf("path_guard: 拒絕含 .. 的匯出路徑: %s", path)
	}

	resolved, err := resolveAndAbs(path)
	if err != nil {
		return fmt.Errorf("path_guard: 無法解析匯出路徑: %w", err)
	}
	if isSystemPath(resolved) {
		return fmt.Errorf("path_guard: 拒絕匯出到系統目錄: %s", resolved)
	}
	return nil
}

// NeedsConfirmation 判斷匯出到此路徑是否需要使用者確認（Zone B）。
func (g *PathGuard) NeedsConfirmation(path string) bool {
	return !g.IsProjectInternal(path)
}

// --- 內部工具函式 ---

// resolveAndAbs 先解析 symlink 再轉絕對路徑，防止 symlink 逃逸。
// 檔案不存在時，改解析父目錄的 symlink + 接上檔名，確保一致性。
func resolveAndAbs(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// 檔案不存在時，嘗試解析父目錄（通常存在）+ 接上檔名
		dir := filepath.Dir(abs)
		base := filepath.Base(abs)
		resolvedDir, dirErr := filepath.EvalSymlinks(dir)
		if dirErr != nil {
			return abs, nil // 父目錄也不存在，直接回傳 abs
		}
		return filepath.Join(resolvedDir, base), nil
	}
	return resolved, nil
}

// containsDotDot 檢查路徑字串是否包含 .. 元件。
func containsDotDot(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

// isSystemPath 檢查路徑是否在已知系統目錄內。
// 注意：/private/var/folders/ 是 macOS 使用者暫存目錄，不應被阻擋。
func isSystemPath(resolved string) bool {
	if runtime.GOOS == "darwin" {
		// 先排除合法的使用者目錄（macOS temp 在 /private/var/folders/）
		if strings.HasPrefix(resolved, "/private/var/folders/") {
			return false
		}
		prefixes := []string{
			"/System/",
			"/Library/",
			"/usr/",
			"/bin/",
			"/sbin/",
			"/private/var/",
		}
		for _, p := range prefixes {
			if strings.HasPrefix(resolved, p) {
				return true
			}
		}
	}
	if runtime.GOOS == "windows" {
		// Windows 路徑大小寫不敏感,統一轉小寫比對。
		// 從環境變數取系統目錄(處理非 C: 安裝),env 缺失時退回常見預設值。
		prefixes := []string{
			envOrDefault("SystemRoot", `C:\Windows`),            // 系統核心
			envOrDefault("ProgramFiles", `C:\Program Files`),    // 已安裝程式
			os.Getenv("ProgramFiles(x86)"),                      // 32 位元程式(可能不存在)
			envOrDefault("ProgramData", `C:\ProgramData`),       // 全系統設定資料
		}
		lowerResolved := strings.ToLower(resolved)
		for _, p := range prefixes {
			if p == "" {
				continue
			}
			lp := strings.ToLower(p)
			if lowerResolved == lp || strings.HasPrefix(lowerResolved, lp+`\`) {
				return true
			}
		}
	}
	// Linux 可依需求擴充
	return false
}

// envOrDefault 讀取環境變數,空值時回傳預設。
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
