package statusrail

import (
	"strings"
	"unicode"
)

const RejectTemplate = "我這邊只能聊天，請至中央對話區修改"

var blockedTerms = []string{
	"摘要", "進度", "狀態", "做到哪", "剩幾個", "完成度",
	"dag", "review", "binding", "memory", "subagent", "lineage",
	"runtime", "診斷", "log", "stack", "crash",
	"傳給中央", "幫我執行", "叫main", "寫入記憶", "更新工作流",
}

func IsBlocked(input string) bool {
	raw := strings.TrimSpace(input)
	cleaned := cleanInput(raw)
	normalized := normalizeInput(cleaned)
	for _, candidate := range []string{raw, cleaned, normalized} {
		lowered := strings.ToLower(candidate)
		for _, term := range blockedTerms {
			if strings.Contains(lowered, term) {
				return true
			}
		}
	}
	return false
}

func cleanInput(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, input)
}

func normalizeInput(input string) string {
	replacer := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"：", ":",
		"／", "/",
		"　", "",
	)
	return strings.ToLower(replacer.Replace(input))
}
