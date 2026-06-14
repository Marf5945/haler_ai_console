// memory/expand.go — deep_memory 寫入與讀回（v3.1.7）。
// 補上管線缺的兩端：摘要時把原文細節落到 deep_memory.md（防丟），
// 模型用 展開ㄌtag或關鍵字ㄌ待命 把細節撈回 context（找回）。
// index.json 維護 S-tag ↔ D-tag ↔ sentence ID 對照。
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const FileMemoryIndex = "index.json"

// MemoryIndexEntry 是一筆 S↔D 對照。
type MemoryIndexEntry struct {
	SummaryTag  string   `json:"summary_tag"`            // 例 "S-12345"
	DeepTag     string   `json:"deep_tag"`               // 例 "D-12345"
	SentenceIDs []string `json:"sentence_ids,omitempty"` // 被摘要的原始句子 ID
	CreatedAt   string   `json:"created_at"`
}

// AppendDeepMemory 將摘要前的原文細節追加到 deep_memory.md（同 AppendSummary 格式）。
// Rule 8：寫入前 redaction。
func (p *Pipeline) AppendDeepMemory(tag, content string) ([]RedactionRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cleaned, records := RedactBeforeWrite(content)
	ts := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n## %s — %s\n%s\n", tag, ts, cleaned)
	f, err := os.OpenFile(filepath.Join(p.rootDir, FileDeepMemory), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("寫入 deep_memory 失敗: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return nil, fmt.Errorf("寫入 deep_memory 失敗: %w", err)
	}
	return records, nil
}

// AppendIndexEntry 追加一筆 S↔D 對照到 index.json（read-modify-write，p.mu 保護）。
func (p *Pipeline) AppendIndexEntry(entry MemoryIndexEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	path := filepath.Join(p.rootDir, FileMemoryIndex)
	var entries []MemoryIndexEntry
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &entries) // 壞檔視為空，重建索引
	}
	entries = append(entries, entry)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// memoryTagPattern 接受 "S-123" / "D-123"，也容忍 "[S-123: ts]" 整段貼進來。
var memoryTagPattern = regexp.MustCompile(`([SD])-(\d+)`)

// NormalizeMemoryTag 從輸入抽出標準 tag；抽不出回空字串（→ 走關鍵字搜尋）。
func NormalizeMemoryTag(input string) string {
	m := memoryTagPattern.FindStringSubmatch(input)
	if m == nil {
		return ""
	}
	return m[1] + "-" + m[2]
}

// LookupByTag 依 tag 撈細節：D-tag 直接讀 deep_memory 段落；
// S-tag 先查 index 找到對應 D-tag，查無 index 時退而求其次用同號 D-tag 試讀。
func (p *Pipeline) LookupByTag(tag string) (string, error) {
	tag = NormalizeMemoryTag(tag)
	if tag == "" {
		return "", fmt.Errorf("不是合法的記憶標籤")
	}
	if strings.HasPrefix(tag, "S-") {
		deepTag := "D-" + strings.TrimPrefix(tag, "S-") // 同號 fallback
		if entries := p.loadIndex(); entries != nil {
			for _, e := range entries {
				if e.SummaryTag == tag && e.DeepTag != "" {
					deepTag = e.DeepTag
					break
				}
			}
		}
		tag = deepTag
	}
	section, err := p.readSection(FileDeepMemory, tag)
	if err != nil {
		return "", err
	}
	return section, nil
}

// SearchDeepMemory 關鍵字搜尋（大小寫不敏感），回傳最多 maxHits 段、每段截 maxBytes。
// 多關鍵字以空白分隔，全部命中才算（AND）。
func (p *Pipeline) SearchDeepMemory(query string, maxHits, maxBytes int) ([]string, error) {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return nil, fmt.Errorf("關鍵字不可為空")
	}
	data, err := os.ReadFile(filepath.Join(p.rootDir, FileDeepMemory))
	if err != nil {
		return nil, fmt.Errorf("deep_memory 尚無內容")
	}
	var hits []string
	// 新的記憶通常更相關 → 由後往前掃
	sections := strings.Split(string(data), "\n## ")
	for i := len(sections) - 1; i >= 0 && len(hits) < maxHits; i-- {
		section := strings.TrimSpace(sections[i])
		if section == "" {
			continue
		}
		lower := strings.ToLower(section)
		matched := true
		for _, term := range terms {
			if !strings.Contains(lower, term) {
				matched = false
				break
			}
		}
		if matched {
			hits = append(hits, truncateBytesRuneSafe("## "+section, maxBytes))
		}
	}
	return hits, nil
}

func (p *Pipeline) loadIndex() []MemoryIndexEntry {
	data, err := os.ReadFile(filepath.Join(p.rootDir, FileMemoryIndex))
	if err != nil {
		return nil
	}
	var entries []MemoryIndexEntry
	if json.Unmarshal(data, &entries) != nil {
		return nil
	}
	return entries
}

// readSection 讀 "## <tag> — ts" 起、到下一個 "## " 前的段落。
func (p *Pipeline) readSection(file, tag string) (string, error) {
	data, err := os.ReadFile(filepath.Join(p.rootDir, file))
	if err != nil {
		return "", fmt.Errorf("找不到記憶檔 %s", file)
	}
	for _, section := range strings.Split(string(data), "\n## ") {
		section = strings.TrimSpace(section)
		if strings.HasPrefix(section, tag+" ") || strings.HasPrefix(section, tag+" —") {
			return "## " + section, nil
		}
	}
	return "", fmt.Errorf("找不到標籤 %s 的記憶段落", tag)
}

// truncateBytesRuneSafe 以 byte 上限截斷但不切壞 UTF-8。
func truncateBytesRuneSafe(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && (s[cut]&0xC0) == 0x80 {
		cut--
	}
	return s[:cut] + "\n…（已截斷，可用 tag 展開完整段落）"
}
