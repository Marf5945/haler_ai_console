package replan

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// sha256Hex 回傳輸入字串的 sha256 十六進位摘要。
// tail hash、tail signature、goal hash、audit hash 共用此 helper。
func sha256Hex(parts ...string) string {
	h := sha256.New()
	// 用 \x1f（unit separator）分隔，避免欄位邊界被內容偽造。
	h.Write([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(h.Sum(nil))
}
