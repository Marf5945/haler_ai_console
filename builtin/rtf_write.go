// rtf_write.go — 簡單 RTF 富文字產生（僅 export）。
// RTF 格式是純文字標記語言，比 .doc 輕很多，TextEdit/Word 都能開。
// Import 不做：RTF 解析太複雜，不值得第一版投入。
// 依賴：純字串組裝，零依賴。
package builtin

import (
	"fmt"
	"os"
	"strings"
)

// GenerateRTF 將純文字產生成 RTF 檔案。
// 每行文字 = 一個段落（\par）。支援中文（Unicode escape）。
func GenerateRTF(content string, path string) error {
	var body strings.Builder

	// RTF header：指定 UTF-8 字集 + 預設字型
	body.WriteString(`{\rtf1\ansi\deff0`)
	body.WriteString(`{\fonttbl{\f0 Helvetica;}}`)
	body.WriteString(`\f0\fs24 `) // 12pt Helvetica

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// RTF 中非 ASCII 字元要用 \uN? 格式 escape
		body.WriteString(rtfEscapeLine(line))
		if i < len(lines)-1 {
			body.WriteString(`\par `) // 段落分隔
		}
	}

	body.WriteString(`}`)

	if err := os.WriteFile(path, []byte(body.String()), 0o600); err != nil {
		return fmt.Errorf("rtf_write: write %s: %w", path, err)
	}
	return nil
}

// rtfEscapeLine 將一行文字轉成 RTF 安全格式。
// ASCII 直接輸出；非 ASCII 用 \uN? escape（N = Unicode code point）。
// RTF 特殊字元 { } \ 需要加反斜線。
func rtfEscapeLine(line string) string {
	var b strings.Builder
	for _, r := range line {
		switch {
		case r == '{' || r == '}' || r == '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		case r < 128:
			// ASCII 直接輸出
			b.WriteRune(r)
		default:
			// 非 ASCII：\uN?（? 是 fallback 字元，通常用 ?）
			fmt.Fprintf(&b, `\u%d?`, r)
		}
	}
	return b.String()
}
