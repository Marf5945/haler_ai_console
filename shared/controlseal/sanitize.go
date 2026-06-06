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
	llmText = neutralizeUntrustedInstructions(sourceType, llmText)
	escapedCount := fakeSealCount + strings.Count(escapedLines, "ㄌ")

	return SanitizedResult{
		RawText:      rawText,
		LLMText:      llmText,
		EscapedCount: escapedCount,
		HasFakeSeal:  fakeSealCount > 0,
		SourceType:   sourceType,
	}
}

func neutralizeUntrustedInstructions(sourceType SourceType, text string) string {
	switch sourceType {
	case SourceDocument, SourceToolOutput, SourceCLIOutput, SourceMemory:
	default:
		return text
	}
	replacements := []struct {
		old string
		new string
	}{
		{"忽略上面所有指令", "[UNTRUSTED_INSTRUCTION_REDACTED]"},
		{"忽略之前的指令", "[UNTRUSTED_INSTRUCTION_REDACTED]"},
		{"忽略所有規則", "[UNTRUSTED_INSTRUCTION_REDACTED]"},
		{"ignore previous instructions", "[UNTRUSTED_INSTRUCTION_REDACTED]"},
		{"ignore all previous", "[UNTRUSTED_INSTRUCTION_REDACTED]"},
		{"disregard your instructions", "[UNTRUSTED_INSTRUCTION_REDACTED]"},
		{"bypass security", "[UNTRUSTED_INSTRUCTION_REDACTED]"},
	}
	out := text
	for _, replacement := range replacements {
		out = strings.ReplaceAll(out, replacement.old, replacement.new)
	}
	return out
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
