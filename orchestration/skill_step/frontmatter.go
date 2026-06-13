// frontmatter.go — 解析外部 CLI（Claude Code / Codex / Gemini）skill 的 SKILL.md
// 開頭 YAML frontmatter。刻意「零外部依賴」：只用標準庫處理 SKILL.md 常見的
// 扁平「key: value」子集（含 inline 陣列 [a, b] 與「- item」區塊清單）。
//
// 設計原則（呼應本套件的最小信任策略）：
//   - 只「讀取與擷取」，不執行、不寫檔。
//   - 解析失敗一律回傳 (zero, false)，呼叫端退回既有行為，不讓匯入流程崩潰。
//   - 不嘗試完整 YAML 規格（巢狀 map、錨點、多行字串等）；遇到不認得的格式就略過該行。
package skill_step

import "strings"

// SkillFrontmatter 是從 SKILL.md frontmatter 擷取出的中繼資料。
// Fields 保留所有解析到的原始 key→value（值為字串；清單已正規化為逗號分隔），
// 方便日後擴充而不必改動函式簽名。
type SkillFrontmatter struct {
	Name         string            // frontmatter 的 name 欄位
	Description  string            // frontmatter 的 description 欄位
	AllowedTools []string          // allowed-tools，逗號 / inline 陣列 / 區塊清單皆可
	Fields       map[string]string // 所有解析到的鍵值（key 一律小寫）
}

// ParseSkillFrontmatter 擷取 content 開頭、以兩條「---」夾住的 YAML frontmatter 區塊。
// 找到合法區塊時回傳 (frontmatter, true)；否則回傳 (zero, false)。
func ParseSkillFrontmatter(content []byte) (SkillFrontmatter, bool) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimPrefix(text, "\ufeff") // 去除 UTF-8 BOM
	lines := strings.Split(text, "\n")

	// 略過開頭空行，接著必須是一條單獨的「---」作為起始分隔線。
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || strings.TrimSpace(lines[i]) != "---" {
		return SkillFrontmatter{}, false
	}
	i++ // 跳過起始分隔線

	// 收集區塊內容，直到遇到結束分隔線「---」或「...」。
	var block []string
	closed := false
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "---" || t == "..." {
			closed = true
			break
		}
		block = append(block, lines[i])
	}
	if !closed {
		// 只有起始分隔線、沒有結束分隔線 → 視為不合法，退回既有行為。
		return SkillFrontmatter{}, false
	}

	fm := SkillFrontmatter{Fields: map[string]string{}}
	for j := 0; j < len(block); j++ {
		line := strings.TrimSpace(block[j])
		if line == "" || strings.HasPrefix(line, "#") {
			continue // 空行與註解
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue // 不是 key: value 形式，略過
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}

		if val == "" {
			// 值為空 → 嘗試收集後續「- item」區塊清單。
			var items []string
			for j+1 < len(block) {
				next := strings.TrimSpace(block[j+1])
				if next == "-" || strings.HasPrefix(next, "- ") {
					items = append(items, unquoteFM(strings.TrimSpace(strings.TrimPrefix(next, "-"))))
					j++
					continue
				}
				break
			}
			fm.Fields[key] = strings.Join(nonEmptyFM(items), ", ")
			continue
		}

		val = unquoteFM(val)
		// inline 陣列：[a, b, c]
		if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			inner := val[1 : len(val)-1]
			var items []string
			for _, p := range strings.Split(inner, ",") {
				items = append(items, unquoteFM(strings.TrimSpace(p)))
			}
			val = strings.Join(nonEmptyFM(items), ", ")
		}
		fm.Fields[key] = val
	}

	fm.Name = fm.Fields["name"]
	fm.Description = fm.Fields["description"]
	fm.AllowedTools = splitListFM(fm.Fields["allowed-tools"])
	return fm, true
}

// unquoteFM 去除字串外圍成對的單／雙引號。
func unquoteFM(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// nonEmptyFM 回傳去除空白項後的新切片（每項皆 trim）。
func nonEmptyFM(in []string) []string {
	var out []string
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// splitListFM 將逗號分隔字串切成清單；空字串回傳 nil。
func splitListFM(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return nonEmptyFM(strings.Split(s, ","))
}

// appendUniqueFM 將 v 加入 list（若尚未存在），回傳更新後的切片。
func appendUniqueFM(list []string, v string) []string {
	for _, e := range list {
		if e == v {
			return list
		}
	}
	return append(list, v)
}
