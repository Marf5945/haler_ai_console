package skill_flow

import (
	"regexp"
	"strconv"
	"strings"
)

// fix.go — 「修正已收項目」語意解析（自 dianliao_bom_fix.go 泛型化搬入）。
// 任何宣告 List 的 flow skill 自動獲得：第N項/值/模糊定位 → 改數量。

var (
	// 「改成5」「為 4項」「應該是10」→ 抓新數量（取最後一個符合的）。
	fixQtyRe = regexp.MustCompile(`(?:改成|改為|改到|應該是|應該為|為|是)\s*([0-9]+(?:\.[0-9]+)?)`)
	// 「第2項」「第 3 個」→ 直接指定清單編號。
	fixIdxRe = regexp.MustCompile(`第\s*([0-9]+)\s*(?:項|個|筆|列)?`)
	// 尾端孤立數字（含單位）當數量的 fallback：「人機改4」「第2項 5」。
	fixLastNumRe = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*(?:項|個|條|顆|組|件|支|pcs|PCS)?\s*$`)
)

var fixIntentWords = []string{
	"修正", "更正", "修改", "改成", "改為", "改到", "改一下",
	"數量錯", "打錯", "輸入錯", "弄錯", "錯了", "改",
}

// HasFixIntent 判斷是不是「修正已收項目」的語意。
func HasFixIntent(text string) bool {
	for _, w := range fixIntentWords {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

// ParseFix 從修正語句切出（目標描述, 指定清單編號, 新數量）。
// idx=0 代表沒講「第N項」；qty 可能為空（之後補問）。
func ParseFix(text string) (target string, idx int, qty string) {
	s := strings.TrimSpace(text)

	if m := fixIdxRe.FindStringSubmatchIndex(s); m != nil {
		idx, _ = strconv.Atoi(s[m[2]:m[3]])
		s = s[:m[0]] + " " + s[m[1]:]
	}
	if ms := fixQtyRe.FindAllStringSubmatchIndex(s, -1); len(ms) > 0 {
		m := ms[len(ms)-1]
		qty = s[m[2]:m[3]]
		s = s[:m[0]] + " " + s[m[1]:]
	} else {
		// fallback：尾端孤立數字（≤4 位，避免把料號當數量）。
		rest := strings.TrimSpace(s)
		if m := fixLastNumRe.FindStringSubmatchIndex(rest); m != nil && m[3]-m[2] <= 4 {
			head := strings.TrimSpace(rest[:m[0]])
			if idx > 0 || StripFixFiller(head) != "" {
				qty = rest[m[2]:m[3]]
				s = head
			}
		}
	}
	return StripFixFiller(s), idx, qty
}

// StripFixFiller 去掉修正語句裡的口語填充詞，留下目標描述。
func StripFixFiller(s string) string {
	fillers := []string{
		"幫我", "請", "麻煩", "把", "將", "的數量", "數量",
		"修正", "更正", "修改", "改一下", "改成", "改為", "改到", "改",
		"打錯", "輸入錯", "弄錯", "錯了", "應該",
		"項", "個", "筆", "顆", "組", "了", "的",
		"，", ",", "。", "、", "！", "!", "？", "?",
	}
	for _, f := range fillers {
		s = strings.ReplaceAll(s, f, " ")
	}
	return strings.Join(strings.Fields(s), " ")
}
