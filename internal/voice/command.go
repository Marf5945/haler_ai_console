package voice

import "strings"

type CommandRoute struct {
	Matched    bool   `json:"matched"`
	Action     string `json:"action"`
	Transcript string `json:"transcript"`
	Reason     string `json:"reason"`
}

func RouteCommand(text string, enabled bool) CommandRoute {
	normalized := normalizeCommandText(text)
	route := CommandRoute{Transcript: strings.TrimSpace(text)}
	if !enabled {
		route.Reason = "command_mode_disabled"
		return route
	}
	if normalized == "" {
		route.Reason = "empty"
		return route
	}

	switch {
	case exactAny(normalized, "停止", "停下", "暫停", "暂停", "取消", "stop", "cancel", "pause"):
		route.Matched = true
		route.Action = "stop_active_job"
	case exactAny(normalized, "繼續", "继续", "恢復", "恢复", "continue", "resume"):
		route.Matched = true
		route.Action = "resume_active_job"
	case strings.Contains(normalized, "不要改檔") ||
		strings.Contains(normalized, "不要改档") ||
		strings.Contains(normalized, "只讀") ||
		strings.Contains(normalized, "只读") ||
		strings.Contains(normalized, "read only"):
		route.Matched = true
		route.Action = "append_readonly_constraint"
	default:
		route.Reason = "not_in_allowlist"
	}
	return route
}

func normalizeCommandText(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	replacer := strings.NewReplacer(
		"。", "",
		"，", "",
		",", "",
		".", "",
		"！", "",
		"!", "",
		" ", "",
		"\n", "",
		"\t", "",
	)
	return replacer.Replace(text)
}

func exactAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == normalizeCommandText(candidate) {
			return true
		}
	}
	return false
}
