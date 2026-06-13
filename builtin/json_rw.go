// json_rw.go — JSON 讀寫。
// Import：遞迴抽取所有字串值，以換行分隔。
// Export：將純文字包成 JSON 字串或美化格式化。
// 依賴：encoding/json（Go 標準庫）。
package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ExtractJSONText 讀取 JSON 檔案，遞迴抽取所有字串值。
// 回傳以 \n 分隔的純文字。適合把結構化 JSON 轉成可搜尋的文字。
func ExtractJSONText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("json_rw: read %s: %w", path, err)
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("json_rw: parse %s: %w", path, err)
	}

	var texts []string
	extractStrings(raw, &texts)
	return strings.Join(texts, "\n"), nil
}

// extractStrings 遞迴走訪 JSON 結構，收集所有字串值。
func extractStrings(v interface{}, out *[]string) {
	switch val := v.(type) {
	case string:
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			*out = append(*out, trimmed)
		}
	case []interface{}:
		for _, item := range val {
			extractStrings(item, out)
		}
	case map[string]interface{}:
		for _, item := range val {
			extractStrings(item, out)
		}
		// 數字、bool、nil：略過
	}
}

// WriteJSONPretty 將純文字以美化 JSON 格式寫入檔案。
// 文字被包成 {"content": "..."} 結構。
func WriteJSONPretty(content string, path string) error {
	wrapper := map[string]string{"content": content}
	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return fmt.Errorf("json_rw: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("json_rw: write %s: %w", path, err)
	}
	return nil
}
