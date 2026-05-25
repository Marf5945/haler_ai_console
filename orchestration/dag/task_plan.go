package dag

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"ui_console/domain/risk"
)

const RawPlanLimit = 64 * 1024

var actionCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type TaskPlan struct {
	Title string         `json:"title"`
	Nodes []TaskPlanNode `json:"nodes"`
}

type TaskPlanNode struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	ExecutorType string   `json:"executor_type"`
	ActionCode   string   `json:"action_code"`
	Action       string   `json:"action"`
	Target       string   `json:"target"`
	RiskClass    string   `json:"risk_class"`
	Dependencies []string `json:"dependencies"`
}

type NormalizeResult struct {
	Plan     TaskPlan
	Warnings []string
}

func ExtractJSONPlan(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func NormalizePlan(raw string) (NormalizeResult, error) {
	var plan TaskPlan
	if err := json.Unmarshal([]byte(ExtractJSONPlan(raw)), &plan); err != nil {
		return NormalizeResult{}, fmt.Errorf("plan JSON 不合法: %w", err)
	}
	return ValidateAndNormalizePlan(plan)
}

func ValidateAndNormalizePlan(plan TaskPlan) (NormalizeResult, error) {
	plan.Title = strings.TrimSpace(plan.Title)
	if plan.Title == "" {
		return NormalizeResult{}, fmt.Errorf("plan.title is required")
	}
	if len(plan.Nodes) == 0 {
		return NormalizeResult{}, fmt.Errorf("plan.nodes must not be empty")
	}
	seen := map[string]bool{}
	warnings := []string{}
	for i := range plan.Nodes {
		n := &plan.Nodes[i]
		if strings.TrimSpace(n.ID) == "" {
			n.ID = fmt.Sprintf("node_%d", i+1)
		}
		if seen[n.ID] {
			return NormalizeResult{}, fmt.Errorf("duplicate node id: %s", n.ID)
		}
		seen[n.ID] = true
		n.Title = strings.TrimSpace(n.Title)
		n.ExecutorType = strings.TrimSpace(n.ExecutorType)
		n.ActionCode = strings.TrimSpace(n.ActionCode)
		n.Action = strings.TrimSpace(n.Action)
		n.Target = strings.TrimSpace(n.Target)
		n.RiskClass = strings.TrimSpace(n.RiskClass)
		if n.Title == "" || n.ExecutorType == "" || n.ActionCode == "" || n.Action == "" || n.Target == "" || n.RiskClass == "" {
			return NormalizeResult{}, fmt.Errorf("node %s missing required fields", n.ID)
		}
		if !actionCodePattern.MatchString(n.ActionCode) {
			return NormalizeResult{}, fmt.Errorf("node %s action_code must be snake_case", n.ID)
		}
		switch n.ExecutorType {
		case "cli_task", "tool_call", "subagent_call":
		default:
			return NormalizeResult{}, fmt.Errorf("node %s unsupported executor_type: %s", n.ID, n.ExecutorType)
		}
		modelRisk := risk.RiskClass(n.RiskClass)
		if !validRisk(modelRisk) {
			return NormalizeResult{}, fmt.Errorf("node %s unsupported risk_class: %s", n.ID, n.RiskClass)
		}
		classified := risk.ClassifyOperation(n.ActionCode, []string{n.Action, n.Target})
		finalRisk := risk.Max(modelRisk, classified)
		if string(finalRisk) != n.RiskClass {
			warnings = append(warnings, fmt.Sprintf("node %s risk raised from %s to %s", n.ID, n.RiskClass, finalRisk))
			n.RiskClass = string(finalRisk)
		}
		if !strings.Contains(strings.ToLower(n.Title), strings.ReplaceAll(n.ActionCode, "_", " ")) {
			warnings = append(warnings, fmt.Sprintf("node %s title/action_code should be reviewed", n.ID))
		}
	}
	for _, n := range plan.Nodes {
		for _, dep := range n.Dependencies {
			if !seen[dep] {
				return NormalizeResult{}, fmt.Errorf("node %s dependency not found: %s", n.ID, dep)
			}
		}
	}
	if hasCycle(plan.Nodes) {
		return NormalizeResult{}, fmt.Errorf("plan dependencies contain a cycle")
	}
	return NormalizeResult{Plan: plan, Warnings: warnings}, nil
}

func validRisk(c risk.RiskClass) bool {
	switch c {
	case risk.Low, risk.Medium, risk.HighNonDestructive, risk.UserOwnedAssetDestructive,
		risk.SubagentLifecycleRemoval, risk.SecurityBoundaryRewrite, risk.CriticalRuntimeAction:
		return true
	default:
		return false
	}
}

func TaskPlanToNodes(plan TaskPlan) []DAGNode {
	nodes := make([]DAGNode, 0, len(plan.Nodes))
	for _, n := range plan.Nodes {
		nodes = append(nodes, DAGNode{
			ID:           n.ID,
			Title:        n.Title,
			Operation:    n.ActionCode,
			Action:       n.Action,
			ActionCode:   n.ActionCode,
			Target:       n.Target,
			ExecutorType: n.ExecutorType,
			RiskClass:    n.RiskClass,
			Status:       StatusPlanned,
			Dependencies: append([]string(nil), n.Dependencies...),
		})
	}
	return nodes
}

func TruncateRawPlan(raw string) (string, bool) {
	if len([]byte(raw)) <= RawPlanLimit {
		return raw, false
	}
	return string([]byte(raw)[:RawPlanLimit]), true
}

func hasCycle(nodes []TaskPlanNode) bool {
	graph := map[string][]string{}
	for _, n := range nodes {
		graph[n.ID] = append([]string(nil), n.Dependencies...)
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		for _, dep := range graph[id] {
			if visit(dep) {
				return true
			}
		}
		visiting[id] = false
		visited[id] = true
		return false
	}
	for _, n := range nodes {
		if visit(n.ID) {
			return true
		}
	}
	return false
}
