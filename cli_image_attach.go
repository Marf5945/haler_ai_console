// cli_image_attach.go —「多模態 CLI 直送圖」的落地與引用。
//
// 背景：CLI 大腦（claude / gemini）不像 API 路徑能用 image_url 內嵌 base64，它們
// 透過「prompt 內引用檔案路徑」讀圖。所以本檔負責：
//  1. 把本回合 base64 圖片寫進「隔離暫存資料夾」（0700/0600，session 專屬）。
//  2. 依各 CLI 的語法產生 prompt 後綴：
//     - claude  → 要求它讀絕對路徑（Claude Code 預設有檔案讀取能力）。
//     - gemini  → 用 @<絕對路徑> 內聯語法。
//  3. 回傳 cleanup，由呼叫端 defer 在 CLI 跑完後即刪——把「落地」嚴格侷限在這一輪。
//
// 安全：暫存根目錄在 os.TempDir() 下、session 命名空間隔離；cleanup 一定執行
// （呼叫端 defer），避免圖片殘留。codex 等沒有可靠讀圖路徑介面的 CLI 不在此處理，
// 由呼叫端退回「視覺器官」轉文字。
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cliVisionTempRoot 是所有 CLI 直送圖暫存檔的根目錄名（位於 os.TempDir() 底下）。
const cliVisionTempRoot = "ai-console-vision-cli"

// imageExtForMIME 由 MIME 推導副檔名，讓 CLI 能正確辨識圖片格式。
func imageExtForMIME(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/bmp":
		return ".bmp"
	default:
		return ".img"
	}
}

// cliAdapterSupportsImagePath 回報該 CLI 是否有「prompt 引用檔案路徑讀圖」的介面。
// 只有確定支援的 CLI 才回 true；其餘（如 codex）一律 false → 呼叫端走視覺器官。
func cliAdapterSupportsImagePath(adapterID string) bool {
	switch strings.ToLower(strings.TrimSpace(adapterID)) {
	case "claude-cli", "gemini-cli":
		return true
	default:
		return false
	}
}

// cliImagePromptSuffix 依 CLI 產生引用已落地圖片的 prompt 後綴。
// paths 為絕對路徑。不支援的 adapter 回 ("", false)。
func cliImagePromptSuffix(adapterID string, paths []string) (string, bool) {
	if len(paths) == 0 {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(adapterID)) {
	case "claude-cli":
		var b strings.Builder
		b.WriteString("\n\n[附圖] 使用者附了以下圖片檔案，請讀取並結合問題回答（路徑為本機絕對路徑）：")
		for _, p := range paths {
			b.WriteString("\n")
			b.WriteString(p)
		}
		return b.String(), true
	case "gemini-cli":
		// Gemini CLI 用 @<path> 內聯檔案內容（含圖片）。
		parts := make([]string, 0, len(paths))
		for _, p := range paths {
			parts = append(parts, "@"+p)
		}
		return "\n\n請參考使用者附上的圖片：" + strings.Join(parts, " "), true
	default:
		return "", false
	}
}

// stageCLIImagesIfSupported 在 adapter 支援路徑讀圖時，把圖片落地並回傳：
//   - suffix：要附到 userText 的 prompt 後綴（引用落地路徑）。
//   - cleanup：呼叫端必須 defer 執行，刪除整個 session 暫存目錄。
//   - ok：是否成功走「直送圖」。false 表示 adapter 不支援或落地失敗，
//     呼叫端應退回視覺器官。
//
// cleanup 永遠非 nil（false 時為 no-op），呼叫端可無條件 defer。
func stageCLIImagesIfSupported(adapterID, sessionID string, imgs []composerImage) (suffix string, cleanup func(), ok bool) {
	noop := func() {}
	if len(imgs) == 0 || !cliAdapterSupportsImagePath(adapterID) {
		return "", noop, false
	}
	dir := filepath.Join(os.TempDir(), cliVisionTempRoot, fmt.Sprintf("%s-%d", sanitizeSessionDir(sessionID), time.Now().UnixNano()))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", noop, false
	}
	cleanupDir := func() { _ = os.RemoveAll(dir) }

	paths := make([]string, 0, len(imgs))
	for i, img := range imgs {
		raw, err := base64.StdEncoding.DecodeString(img.DataB64)
		if err != nil || len(raw) == 0 {
			cleanupDir()
			return "", noop, false
		}
		name := fmt.Sprintf("img-%02d%s", i+1, imageExtForMIME(img.MIME))
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, raw, 0o600); err != nil {
			cleanupDir()
			return "", noop, false
		}
		paths = append(paths, p)
	}

	sfx, supported := cliImagePromptSuffix(adapterID, paths)
	if !supported {
		cleanupDir()
		return "", noop, false
	}
	return sfx, cleanupDir, true
}

// sanitizeSessionDir 把 sessionID 變成安全的目錄片段（避免路徑穿越 / 奇異字元）。
func sanitizeSessionDir(sessionID string) string {
	s := strings.TrimSpace(sessionID)
	if s == "" {
		return "sess"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
