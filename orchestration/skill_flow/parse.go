package skill_flow

import (
	"strconv"
	"strings"
)

// parse.go — 引擎共用的純解析函式（自 dianliao_bom_flow.go 泛型化搬入）。

// MatchWord 判斷輸入是否等於控制詞之一（trim＋小寫後精確比對）。
func MatchWord(text string, words []string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	for _, w := range words {
		if t == strings.ToLower(w) {
			return true
		}
	}
	return false
}

// SplitItemInput 從一行清單輸入切出「查詢字串」與「數量」。
// 規則：最後一個 token 是純整數 → 當數量，其餘為查詢；否則整行為查詢、數量留空。
func SplitItemInput(text string) (query, qty string) {
	norm := strings.NewReplacer("，", " ", ",", " ", "、", " ", "\t", " ", "　", " ").Replace(text)
	fields := strings.Fields(norm)
	if len(fields) == 0 {
		return "", ""
	}
	last := fields[len(fields)-1]
	if _, err := strconv.Atoi(last); err == nil {
		return strings.Join(fields[:len(fields)-1], " "), last
	}
	return strings.Join(fields, " "), ""
}

// ParseQty 從文字抽出數量（取第一段連續數字，允許小數）。
func ParseQty(text string) (string, bool) {
	s := strings.TrimSpace(text)
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r == '.' && b.Len() > 0) {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			break
		}
	}
	out := strings.TrimRight(b.String(), ".")
	if out == "" {
		return "", false
	}
	return out, true
}

// LeadingInt 取字串開頭的連續數字。
func LeadingInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	if b.Len() == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(b.String())
	if err != nil {
		return 0, false
	}
	return n, true
}

// MatchPick 把使用者的選擇對應到候選：支援編號、Value（料號）、Name（品名）。
// 回 0-based index。
func MatchPick(text string, cands []Candidate) (int, bool) {
	s := strings.TrimSpace(text)
	if s == "" || len(cands) == 0 {
		return 0, false
	}
	if n, ok := LeadingInt(s); ok && n >= 1 && n <= len(cands) {
		return n - 1, true
	}
	low := strings.ToLower(s)
	for i, c := range cands {
		if v := strings.ToLower(strings.TrimSpace(c.Value)); v != "" && (low == v || strings.Contains(low, v)) {
			return i, true
		}
	}
	for i, c := range cands {
		if nm := strings.ToLower(strings.TrimSpace(c.Name)); nm != "" && (strings.Contains(low, nm) || strings.Contains(nm, low)) {
			return i, true
		}
	}
	return 0, false
}

func containsInt(xs []int, n int) bool {
	for _, x := range xs {
		if x == n {
			return true
		}
	}
	return false
}
