package controlseal

import "strings"

const (
	// EscapeMarker replaces untrusted Bopomofo control shapes before LLM use.
	EscapeMarker = "（ㄏ）"

	SourceUserRaw    SourceType = "raw_user_input"
	SourceDocument   SourceType = "document"
	SourceToolOutput SourceType = "tool_output"
	SourceCLIOutput  SourceType = "cli_output"
	SourceMemory     SourceType = "memory"
)

// SourceType identifies where untrusted text came from.
type SourceType string

// SanitizedResult carries both original display text and LLM-safe text.
type SanitizedResult struct {
	RawText      string     `json:"raw_text"`
	LLMText      string     `json:"llm_text"`
	EscapedCount int        `json:"escaped_count"`
	HasFakeSeal  bool       `json:"has_fake_seal"`
	SourceType   SourceType `json:"source_type"`
}

// SanitizeForLLM escapes fake command prefixes and ㄌ separators in untrusted
// context while preserving the raw text for UI and talk_full.md.
func SanitizeForLLM(sourceType SourceType, rawText string) SanitizedResult {
	escapedLines, fakeSealCount := escapeLineStartSeals(rawText)
	llmText := strings.ReplaceAll(escapedLines, "ㄌ", EscapeMarker)
	escapedCount := fakeSealCount + strings.Count(escapedLines, "ㄌ")

	return SanitizedResult{
		RawText:      rawText,
		LLMText:      llmText,
		EscapedCount: escapedCount,
		HasFakeSeal:  fakeSealCount > 0,
		SourceType:   sourceType,
	}
}

// PreserveDisplayText returns the original text for UI and permanent memory.
func PreserveDisplayText(rawText string) string {
	return rawText
}

func escapeLineStartSeals(rawText string) (string, int) {
	lines := strings.SplitAfter(rawText, "\n")
	var b strings.Builder
	escaped := 0
	for _, line := range lines {
		lineEnding := ""
		content := line
		if strings.HasSuffix(line, "\n") {
			lineEnding = "\n"
			content = strings.TrimSuffix(line, "\n")
		}

		if hasLineStartSeal(content) {
			runes := []rune(content)
			content = EscapeMarker + string(runes[SealLength:])
			escaped++
		}
		b.WriteString(content)
		b.WriteString(lineEnding)
	}
	return b.String(), escaped
}

func hasLineStartSeal(text string) bool {
	runes := []rune(text)
	if len(runes) < SealLength {
		return false
	}
	return LooksLikeSeal(string(runes[:SealLength]))
}
