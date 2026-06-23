// html_rw.go — HTML 產生與純文字抽取。
// Import：去除 HTML tag，保留純文字。
// Export：用簡單模板將純文字包成 HTML。
// 依賴：strings, html（Go 標準庫）。不引入 golang.org/x/net/html 以減少依賴。
package builtin

import (
	"fmt"
	"html"
	"os"
	"strings"
)

// ExtractHTMLText 從 HTML 字串中去除所有 tag，回傳純文字。
// 簡易實作：用狀態機跳過 < > 之間的內容，保留文字。
// 不處理 CDATA、script、style 內容（第一版足夠）。
func ExtractHTMLText(raw string) string {
	var result strings.Builder
	inTag := false
	prevSpace := false

	for _, r := range raw {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			// tag 結束後加空白，避免文字黏在一起
			if !prevSpace {
				result.WriteRune(' ')
				prevSpace = true
			}
		case !inTag:
			result.WriteRune(r)
			prevSpace = (r == ' ' || r == '\n' || r == '\t')
		}
	}

	// 清理多餘空白 + HTML entities
	text := result.String()
	text = html.UnescapeString(text) // 處理 &amp; &lt; 等
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

// ExtractHTMLTextFromFile 從 HTML 檔案抽取純文字。
func ExtractHTMLTextFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("html_rw: read %s: %w", path, err)
	}
	return ExtractHTMLText(string(data)), nil
}

// GenerateHTML 將純文字包成簡單 HTML 檔案。
// 每行文字包在 <p> 標籤中。
func GenerateHTML(content string, title string, path string) error {
	if title == "" {
		title = "Document"
	}

	var body strings.Builder
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		escaped := html.EscapeString(line)
		if strings.TrimSpace(escaped) == "" {
			body.WriteString("<p>&nbsp;</p>\n")
		} else {
			body.WriteString("<p>" + escaped + "</p>\n")
		}
	}

	doc := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-TW">
<head>
<meta charset="UTF-8">
<title>%s</title>
<style>body{font-family:system-ui,sans-serif;max-width:800px;margin:2em auto;line-height:1.6;}</style>
</head>
<body>
%s</body>
</html>`, html.EscapeString(title), body.String())

	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		return fmt.Errorf("html_rw: write %s: %w", path, err)
	}
	return nil
}
