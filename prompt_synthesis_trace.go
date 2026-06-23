package main

import "ui_console/adapter/debugtrace"

const promptSynthesisTracePreviewBytes = 16000

func recordPromptSynthesisTrace(node, traceID, prompt string, fields map[string]interface{}) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	preview := truncateRunes(prompt, promptSynthesisTracePreviewBytes)
	fields["prompt_len"] = len([]rune(prompt))
	fields["prompt_preview"] = preview
	fields["prompt_compacted"] = len(prompt) > len(preview)
	debugtrace.Record(node, traceID, fields)
}
