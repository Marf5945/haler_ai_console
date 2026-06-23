package go_program

import (
	"encoding/json"
	"fmt"
	"strings"
)

type AuthoringDraft struct {
	Files        map[string]string `json:"files"`
	Input        json.RawMessage   `json:"input"`
	InputSchema  ObjectSchema      `json:"input_schema"`
	OutputSchema ObjectSchema      `json:"output_schema"`
	Purpose      string            `json:"purpose"`
}

func ParseAuthoringDraft(text string) (AuthoringDraft, error) {
	raw := extractJSONObject(text)
	var draft AuthoringDraft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return AuthoringDraft{}, fmt.Errorf("go_program: parse authoring draft: %w", err)
	}
	if len(draft.Files) == 0 {
		return AuthoringDraft{}, fmt.Errorf("go_program: authoring draft requires files")
	}
	if len(draft.Input) == 0 {
		draft.Input = json.RawMessage(`{"input":{}}`)
	}
	if len(draft.InputSchema.Required) == 0 {
		draft.InputSchema.Required = []string{"input"}
	}
	if len(draft.OutputSchema.Required) == 0 {
		draft.OutputSchema.Required = []string{"result"}
	}
	return draft, nil
}

func extractJSONObject(text string) string {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 {
			lines = lines[1 : len(lines)-1]
			trimmed = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end >= start {
		return trimmed[start : end+1]
	}
	return trimmed
}
