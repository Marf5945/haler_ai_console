// memory/talk_full.go — talk_full.md 專用操作（§18.1）。
// talk_full 是 append-only 的完整對話記錄。
// 輪轉策略：500KB 固定大小，到達閾值自動歸檔。
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ──────────────────────────────────────────────
// Talk Index 結構
// ──────────────────────────────────────────────

// TalkIndexEntry 是 talk_index.json 中的索引條目。
type TalkIndexEntry struct {
	Timestamp string `json:"timestamp"`
	Role      string `json:"role"`
	Preview   string `json:"preview"` // 前 50 字元摘要
	Offset    int64  `json:"offset"`  // 在 talk_full.md 中的 byte offset
}

// ──────────────────────────────────────────────
// 讀取操作
// ──────────────────────────────────────────────

// ReadTalkFull 讀取 talk_full.md 的完整內容。
// 僅供本地除錯用，不應送入 LLM context。
func ReadTalkFull(projectRoot string) (string, error) {
	path := filepath.Join(projectRoot, "memory", FileTalkFull)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("讀取 talk_full 失敗: %w", err)
	}
	return string(data), nil
}

// GetTalkFullSize 回傳 talk_full.md 的檔案大小（bytes）。
func GetTalkFullSize(projectRoot string) int64 {
	return fileSize(filepath.Join(projectRoot, "memory", FileTalkFull))
}

// ──────────────────────────────────────────────
// 搜尋操作
// ──────────────────────────────────────────────

// SearchTalkFull 在 talk_full.md 中搜尋關鍵字。
// 回傳包含關鍵字的行（最多 maxResults 筆）。
// 用於本地記憶檢索，不走 LLM。
func SearchTalkFull(projectRoot string, keyword string, maxResults int) []string {
	path := filepath.Join(projectRoot, "memory", FileTalkFull)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	if maxResults <= 0 {
		maxResults = 20
	}

	lines := strings.Split(string(data), "\n")
	var results []string
	lowerKeyword := strings.ToLower(keyword)

	for _, line := range lines {
		if len(results) >= maxResults {
			break
		}
		if strings.Contains(strings.ToLower(line), lowerKeyword) {
			results = append(results, strings.TrimSpace(line))
		}
	}

	return results
}

// ──────────────────────────────────────────────
// 歸檔清單
// ──────────────────────────────────────────────

// ListTalkArchives 列出所有歸檔的 talk_full 檔案。
func ListTalkArchives(projectRoot string) ([]string, error) {
	archiveDir := filepath.Join(projectRoot, "memory", "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var archives []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "talk_full_") {
			archives = append(archives, entry.Name())
		}
	}
	return archives, nil
}
