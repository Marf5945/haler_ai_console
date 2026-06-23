// Package toolresolver maps action-chain tags to a concrete tool.
package toolresolver

import "strings"

type ResolutionStatus string

const (
	StatusSelected  ResolutionStatus = "selected"
	StatusAmbiguous ResolutionStatus = "ambiguous"
	StatusNotFound  ResolutionStatus = "not_found"
)

// Tool is the minimal registry view needed for action tag resolution.
type Tool struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	ActionTags []string `json:"action_tags"`
	RiskClass  string   `json:"risk_class,omitempty"`
}

// Context captures recent user/tool state for low-friction disambiguation.
type Context struct {
	RecentToolID  string `json:"recent_tool_id,omitempty"`
	RecentChannel string `json:"recent_channel,omitempty"`
	ActiveToolID  string `json:"active_tool_id,omitempty"`
}

// ToolResolution tells the Controller whether to run or ask for clarification.
type ToolResolution struct {
	Status     ResolutionStatus `json:"status"`
	Tool       Tool             `json:"tool,omitempty"`
	Candidates []Tool           `json:"candidates,omitempty"`
	Confidence float64          `json:"confidence"`
	Reason     string           `json:"reason,omitempty"`
}

// ResolveTool chooses the best matching tool for an action tag and target.
func ResolveTool(actionTag, target string, context Context, tools []Tool) ToolResolution {
	matches := matchingTools(actionTag, tools)
	if len(matches) == 0 {
		return ToolResolution{Status: StatusNotFound, Reason: "no tool registered this action tag"}
	}
	if len(matches) == 1 {
		return ToolResolution{Status: StatusSelected, Tool: matches[0], Confidence: 1}
	}

	contextMatches := contextualMatches(context, matches)
	if len(contextMatches) == 1 {
		return ToolResolution{
			Status:     StatusSelected,
			Tool:       contextMatches[0],
			Candidates: matches,
			Confidence: 0.9,
			Reason:     "recent context selected the tool",
		}
	}

	return ToolResolution{
		Status:     StatusAmbiguous,
		Candidates: matches,
		Confidence: 0.4,
		Reason:     "multiple tools share this action tag; ask the user",
	}
}

func matchingTools(actionTag string, tools []Tool) []Tool {
	actionTag = strings.TrimSpace(actionTag)
	var matches []Tool
	for _, tool := range tools {
		for _, tag := range tool.ActionTags {
			if strings.TrimSpace(tag) == actionTag {
				matches = append(matches, tool)
				break
			}
		}
	}
	return matches
}

func contextualMatches(context Context, tools []Tool) []Tool {
	var matches []Tool
	for _, tool := range tools {
		if context.ActiveToolID != "" && tool.ID == context.ActiveToolID {
			matches = append(matches, tool)
			continue
		}
		if context.RecentToolID != "" && tool.ID == context.RecentToolID {
			matches = append(matches, tool)
			continue
		}
		channel := strings.ToLower(strings.TrimSpace(context.RecentChannel))
		if channel != "" && (strings.Contains(strings.ToLower(tool.ID), channel) ||
			strings.Contains(strings.ToLower(tool.Name), channel) ||
			strings.Contains(strings.ToLower(tool.Kind), channel)) {
			matches = append(matches, tool)
		}
	}
	return matches
}
