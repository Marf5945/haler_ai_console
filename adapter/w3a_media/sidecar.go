// w3a_media/sidecar.go — §9A.13 Sidecar .w3a.json 讀寫與匯出。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ Sidecar 是 W3A 的持久驗證層：                               │
// │                                                             │
// │  image.png      → image.png.w3a.json                        │
// │  audio.wav      → audio.wav.w3a.json                        │
// │                                                             │
// │ 雙重儲存策略：                                              │
// │  1. 檔案 metadata（如格式支援）→ 便利層                     │
// │  2. sidecar .w3a.json          → 持久層                     │
// │                                                             │
// │ 容錯規則：                                                  │
// │  - metadata 被清除但 sidecar 存在 → 仍可驗證                │
// │  - 兩者都不存在 → unverified                                │
// │  - 匯出時必須同時複製 sidecar，否則警告                     │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ──────────────────────────────────────────────
// Sidecar 路徑
// ──────────────────────────────────────────────

// SidecarPath 回傳媒體檔案對應的 sidecar 路徑。
func SidecarPath(mediaPath string) string {
	return mediaPath + ".w3a.json"
}

// ──────────────────────────────────────────────
// 讀取 Sidecar
// ──────────────────────────────────────────────

// ReadSidecar 讀取 .w3a.json sidecar 檔案。
// 若 sidecar 不存在，回傳 nil 而非 error。
func ReadSidecar(mediaPath string) (*W3AMediaInfo, error) {
	path := SidecarPath(mediaPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sidecar: %w", err)
	}

	var info W3AMediaInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse sidecar: %w", err)
	}
	return &info, nil
}

// ──────────────────────────────────────────────
// 寫入 Sidecar
// ──────────────────────────────────────────────

// WriteSidecar 寫入 .w3a.json sidecar 檔案。
//
// SEC-W07 例外（2026-05-24）：保持 0644。
// W3A sidecar 是「durable verification artifact」（§9A.13），跟著媒體檔流轉，
// 必須讓其他 W3A-aware app / verifier / 跨帳號流程可讀；改 0o600 會破壞 interop 契約。
func WriteSidecar(info *W3AMediaInfo, mediaPath string) error {
	path := SidecarPath(mediaPath)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sidecar: %w", err)
	}
	return os.WriteFile(path, data, 0644) // SEC-W07 例外：見上方註解
}

// ──────────────────────────────────────────────
// 存在性檢查
// ──────────────────────────────────────────────

// HasSidecar 檢查 sidecar 是否存在。
func HasSidecar(mediaPath string) bool {
	_, err := os.Stat(SidecarPath(mediaPath))
	return err == nil
}

// ──────────────────────────────────────────────
// 匯出（媒體 + sidecar 一起複製）
// ──────────────────────────────────────────────

// ExportWithSidecar 複製媒體檔案到目標路徑，同時複製 sidecar。
// 若 sidecar 不存在，僅複製媒體檔案並回傳警告 error。
func ExportWithSidecar(srcPath, destPath string) error {
	// 複製媒體檔案
	if err := copyFile(srcPath, destPath); err != nil {
		return fmt.Errorf("copy media: %w", err)
	}

	// 複製 sidecar
	srcSidecar := SidecarPath(srcPath)
	destSidecar := SidecarPath(destPath)

	if _, err := os.Stat(srcSidecar); os.IsNotExist(err) {
		return fmt.Errorf("warning: sidecar not found, media exported without W3A verification data")
	}

	if err := copyFile(srcSidecar, destSidecar); err != nil {
		return fmt.Errorf("copy sidecar: %w", err)
	}

	return nil
}

// copyFile 複製單一檔案。
func copyFile(src, dst string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()

	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstF.Close()

	if _, err := io.Copy(dstF, srcF); err != nil {
		return err
	}

	// 保留原始權限
	info, err := os.Stat(src)
	if err == nil {
		os.Chmod(dst, info.Mode())
	}
	return nil
}
