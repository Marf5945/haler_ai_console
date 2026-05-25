// subexport/codegen.go — 系統識別碼產生（§31.1）。
// 格式: [display_name]_SUB_[12碼隨機]
// 字元集排除 I 和 O，避免與 1 和 0 混淆。
package subexport

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// 常數定義
// ──────────────────────────────────────────────

// charset 排除 I 和 O 的字元集（34 字元）
const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ0123456789"

// codeLength 隨機碼長度
const codeLength = 12

// ──────────────────────────────────────────────
// 碼產生函式
// ──────────────────────────────────────────────

// GenerateRandomCode 產生 12 碼隨機識別碼（使用 crypto/rand）。
func GenerateRandomCode() (string, error) {
	result := make([]byte, codeLength)
	max := big.NewInt(int64(len(charset)))

	for i := 0; i < codeLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("產生隨機碼失敗: %w", err)
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

// GenerateSystemCode 產生完整的系統識別碼。
// 格式: [displayName]_SUB_[12碼]
func GenerateSystemCode(displayName string) (string, error) {
	code, err := GenerateRandomCode()
	if err != nil {
		return "", err
	}
	return formatSystemCode(displayName, code), nil
}

// GenerateSystemCodeAt 產生指定時間的系統識別碼（供測試使用）。
func GenerateSystemCodeAt(displayName string, t time.Time) (string, error) {
	code, err := GenerateRandomCode()
	if err != nil {
		return "", err
	}
	_ = t
	return formatSystemCode(displayName, code), nil
}

func formatSystemCode(displayName string, code string) string {
	return fmt.Sprintf("%s_SUB_%s", sanitizeSystemCodeName(displayName), code)
}

func sanitizeSystemCodeName(displayName string) string {
	name := strings.TrimSpace(displayName)
	if name == "" {
		return "sub"
	}
	replacer := strings.NewReplacer(
		"/", "／",
		"\\", "＼",
		":", "：",
		string(rune(0)), "",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return "sub"
	}
	return name
}
