// storage/path_hardening.go — 路徑安全驗證（§17.4）。
// 所有路徑操作必須通過此模組的驗證，防止路徑穿越、symlink 攻擊、zip-slip 等。
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ──────────────────────────────────────────────
// 路徑驗證（§17.4 核心）
// ──────────────────────────────────────────────

// ValidatePath 驗證路徑是否安全。
// 若 boundary 非空，驗證路徑是否在 boundary 目錄內（防止路徑穿越）。
// 若 boundary 為空，僅驗證路徑不含危險字元。
func ValidatePath(path, boundary string) error {
	if path == "" {
		return fmt.Errorf("路徑不能為空")
	}

	// 拒絕含有路徑穿越字元的路徑
	if strings.Contains(path, "..") {
		return fmt.Errorf("路徑不能包含 '..'")
	}

	// 拒絕含有 null byte 的路徑
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("路徑不能包含 null byte")
	}

	// 若有 boundary，執行邊界檢查
	if boundary != "" {
		return validateBoundary(path, boundary)
	}

	return nil
}

// validateBoundary 驗證路徑在 boundary 目錄範圍內。
func validateBoundary(path, boundary string) error {
	// 正規化兩個路徑
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("無法解析路徑: %w", err)
	}

	absBoundary, err := filepath.Abs(boundary)
	if err != nil {
		return fmt.Errorf("無法解析邊界路徑: %w", err)
	}

	// 嘗試解析 symlink（若路徑存在）
	realPath, err := resolveIfExists(absPath)
	if err != nil {
		return fmt.Errorf("無法解析真實路徑: %w", err)
	}

	realBoundary, err := resolveIfExists(absBoundary)
	if err != nil {
		return fmt.Errorf("無法解析邊界真實路徑: %w", err)
	}

	// 確保路徑在邊界內
	if !strings.HasPrefix(realPath+string(os.PathSeparator), realBoundary+string(os.PathSeparator)) &&
		realPath != realBoundary {
		return fmt.Errorf("路徑 %s 超出允許範圍 %s", realPath, realBoundary)
	}

	return nil
}

// resolveIfExists 若路徑存在則解析 symlink，否則遞迴解析最近的存在祖先。
// 這確保 /var/folders/... 在 macOS 上一律被解析為 /private/var/folders/...，
// 即使路徑末端的檔案尚未建立。
func resolveIfExists(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 檔案不存在 → 解析父目錄，再接回 basename
			parent := filepath.Dir(path)
			base := filepath.Base(path)
			if parent == path {
				// 遞迴到根目錄仍不存在，回傳 Clean 路徑
				return filepath.Clean(path), nil
			}
			resolvedParent, err2 := resolveIfExists(parent)
			if err2 != nil {
				return filepath.Clean(path), nil
			}
			return filepath.Join(resolvedParent, base), nil
		}
		return "", err
	}
	return resolved, nil
}

// ──────────────────────────────────────────────
// Symlink / Hardlink / Device File 拒絕
// ──────────────────────────────────────────────

// RejectSpecialFile 拒絕 symlink、device file 等非一般檔案。
// 用於匯入檔案時的安全檢查。
func RejectSpecialFile(path string) error {
	info, err := os.Lstat(path) // Lstat 不跟隨 symlink
	if err != nil {
		return fmt.Errorf("無法取得檔案資訊: %w", err)
	}

	mode := info.Mode()

	// 拒絕 symlink
	if mode&os.ModeSymlink != 0 {
		return fmt.Errorf("拒絕 symlink: %s", path)
	}

	// 拒絕 device file
	if mode&os.ModeDevice != 0 {
		return fmt.Errorf("拒絕 device file: %s", path)
	}

	// 拒絕 named pipe
	if mode&os.ModeNamedPipe != 0 {
		return fmt.Errorf("拒絕 named pipe: %s", path)
	}

	// 拒絕 socket
	if mode&os.ModeSocket != 0 {
		return fmt.Errorf("拒絕 socket: %s", path)
	}

	return nil
}

// ──────────────────────────────────────────────
// Zip-slip 保護
// ──────────────────────────────────────────────

// ValidateZipEntry 驗證 zip 內的檔案路徑是否安全（防止 zip-slip 攻擊）。
// destDir 為解壓縮目標目錄，entryName 為 zip 內的路徑。
func ValidateZipEntry(destDir, entryName string) (string, error) {
	// 清理路徑
	cleanName := filepath.Clean(entryName)

	// 拒絕絕對路徑
	if filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("zip entry 不能是絕對路徑: %s", entryName)
	}
	if strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, `\`) {
		return "", fmt.Errorf("zip entry 不能是絕對路徑: %s", entryName)
	}

	// 拒絕路徑穿越
	if strings.HasPrefix(cleanName, "..") || strings.Contains(cleanName, string(os.PathSeparator)+"..") {
		return "", fmt.Errorf("zip entry 包含路徑穿越: %s", entryName)
	}

	// 組合完整路徑
	fullPath := filepath.Join(destDir, cleanName)

	// 再次驗證結果路徑在目標目錄內
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absFull, absDest+string(os.PathSeparator)) && absFull != absDest {
		return "", fmt.Errorf("zip-slip 偵測: %s 逃逸至 %s 外", entryName, destDir)
	}

	return fullPath, nil
}

// ──────────────────────────────────────────────
// Atomic Write 輔助函式
// ──────────────────────────────────────────────

// AtomicWriteFile 以原子方式寫入檔案。
// 先寫入暫存檔，再 rename 覆蓋目標，確保斷電時不會產生半寫入狀態。
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// 產生隨機暫存檔名
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("無法產生隨機檔名: %w", err)
	}
	tmpName := filepath.Join(dir, ".tmp_"+hex.EncodeToString(randBytes))

	// 寫入暫存檔
	if err := os.WriteFile(tmpName, data, perm); err != nil {
		return fmt.Errorf("寫入暫存檔失敗: %w", err)
	}

	// Rename（原子操作）
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName) // 清理暫存檔
		return fmt.Errorf("rename 失敗: %w", err)
	}

	return nil
}

// AtomicAppendLine 以原子方式追加一行到檔案（用於 append-only log）。
// 注意：在同一進程內的並行追加需外部同步。
func AtomicAppendLine(path string, line []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("開啟 append-only 檔案失敗: %w", err)
	}
	defer f.Close()

	// 確保以換行結尾
	data := line
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("追加寫入失敗: %w", err)
	}

	return nil
}

// SafeCopy 安全複製檔案，驗證來源與目標路徑。
func SafeCopy(src, dst, boundary string) error {
	// 驗證來源路徑
	if err := ValidatePath(src, boundary); err != nil {
		return fmt.Errorf("來源路徑驗證失敗: %w", err)
	}
	// 驗證目標路徑
	if err := ValidatePath(dst, boundary); err != nil {
		return fmt.Errorf("目標路徑驗證失敗: %w", err)
	}
	// 拒絕特殊檔案
	if err := RejectSpecialFile(src); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("開啟來源檔案失敗: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("建立目標檔案失敗: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("複製失敗: %w", err)
	}

	return nil
}
