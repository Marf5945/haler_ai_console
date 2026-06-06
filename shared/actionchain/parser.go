// Package actionchain parses the LLM-only ㄌ action-chain protocol.
package actionchain

import (
	"errors"
	"fmt"
	"strings"
)

const Separator = "ㄌ"
const StandbyNext = "待命"
const QuestionNext = "提問"

var separatorAliases = []string{"ㄎ"}

var (
	ErrStructure        = errors.New("actionchain: invalid structure")
	ErrUnknownActionTag = errors.New("actionchain: unknown action tag")
)

// ReservedTags are Controller-owned UI/render actions.
var ReservedTags = []string{"聊天", "輸出", "澄清", "重試", "提問", "問題", "詢問", "選項", "複製", "貼上"}

var reservedTagSet = map[string]bool{
	"聊天": true,
	"輸出": true,
	"澄清": true,
	"重試": true,
	"提問": true,
	"問題": true,
	"詢問": true,
	"選項": true,
	"複製": true,
	"貼上": true,
}

// ActionChain is the structured form of 動作ㄌ目標ㄌ下一步.
type ActionChain struct {
	Action string `json:"action"`
	Target string `json:"target"`
	Next   string `json:"next,omitempty"`
	Raw    string `json:"raw"`
}

// BuiltInDecision describes Controller-owned actions before tool routing.
type BuiltInDecision struct {
	Handled     bool        `json:"handled"`
	DisplayText string      `json:"display_text,omitempty"`
	Terminal    bool        `json:"terminal"`
	Chain       ActionChain `json:"chain"`
}

// ActionRegistry answers whether a parsed action tag is currently available.
type ActionRegistry interface {
	HasActionTag(tag string) bool
}

// StaticRegistry is a small in-memory registry used by tests and adapters.
type StaticRegistry map[string]bool

// NewStaticRegistry creates a registry with reserved and dynamic tags.
func NewStaticRegistry(dynamicTags ...string) StaticRegistry {
	registry := StaticRegistry{}
	for _, tag := range ReservedTags {
		registry[tag] = true
	}
	for _, tag := range dynamicTags {
		tag = strings.TrimSpace(tag)
		// Built-ins stay Controller-owned; tools with the same tag need explicit routing.
		if tag != "" && !IsReservedTag(tag) {
			registry[tag] = true
		}
	}
	return registry
}

// HasActionTag implements ActionRegistry.
func (r StaticRegistry) HasActionTag(tag string) bool {
	return r[strings.TrimSpace(tag)]
}

// ValidationCode describes how the Controller should handle a parsed chain.
type ValidationCode string

const (
	ValidationOK        ValidationCode = "ok"
	ValidationUnknown   ValidationCode = "unknown_action_tag"
	ValidationMalformed ValidationCode = "structure_error"
)

// ValidationResult carries tag validation status without losing the chain.
type ValidationResult struct {
	Code   ValidationCode `json:"code"`
	Chain  ActionChain    `json:"chain"`
	Err    error          `json:"-"`
	Reason string         `json:"reason,omitempty"`
}

// Parse splits LLM direct output into an ActionChain.
func Parse(output string) (ActionChain, error) {
	raw := strings.TrimSpace(output)
	raw = strings.TrimPrefix(raw, "ㄌㄤㄤ")
	raw = normalizeSeparatorAliases(raw)
	segments := strings.Split(raw, Separator)
	if len(segments) < 2 || len(segments) > 3 {
		return ActionChain{Raw: output}, fmt.Errorf("%w: expected 2 or 3 segments", ErrStructure)
	}

	chain := ActionChain{
		Action: NormalizeAction(strings.TrimSpace(segments[0])),
		Target: strings.TrimSpace(segments[1]),
		Raw:    raw,
	}
	if len(segments) == 3 {
		chain.Next = NormalizeNext(strings.TrimSpace(segments[2]))
	}
	if err := ValidateStructure(chain); err != nil {
		return chain, err
	}
	return chain, nil
}

// IsReservedTag reports whether a tag is owned by the Controller.
func IsReservedTag(tag string) bool {
	return reservedTagSet[strings.TrimSpace(tag)] || reservedTagSet[NormalizeAction(tag)]
}

// NormalizeAction folds legacy and user-facing aliases into Controller-owned tags.
func NormalizeAction(action string) string {
	switch strings.TrimSpace(action) {
	case "閒聊":
		return "聊天"
	case "問題", "詢問":
		return "提問"
	case "操做":
		return "操作"
	default:
		return strings.TrimSpace(action)
	}
}

func normalizeSeparatorAliases(raw string) string {
	if strings.Contains(raw, Separator) {
		return raw
	}
	for _, alias := range separatorAliases {
		if strings.Count(raw, alias) > 0 && strings.Count(raw, alias) <= 2 {
			return strings.ReplaceAll(raw, alias, Separator)
		}
	}
	return raw
}

// NormalizeNext keeps the protocol terse even if an older prompt says "等待指令".
func NormalizeNext(next string) string {
	next = strings.TrimSpace(next)
	firstLine := firstNonEmptyLine(next)
	if firstLine != "" {
		next = firstLine
	}
	switch next {
	case "等待", "等待指令", "等候指令":
		return StandbyNext
	default:
		return next
	}
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// IsStandbyNext marks an action-chain as complete; the app does not route more work.
func IsStandbyNext(next string) bool {
	return NormalizeNext(next) == StandbyNext
}

// IsQuestionNext means the action needs more background before execution.
func IsQuestionNext(next string) bool {
	return NormalizeNext(next) == QuestionNext
}

// ResolveBuiltIn renders Controller-owned action tags before any tool dispatch.
func ResolveBuiltIn(chain ActionChain) BuiltInDecision {
	chain.Next = NormalizeNext(chain.Next)
	decision := BuiltInDecision{Chain: chain, Terminal: IsStandbyNext(chain.Next)}
	switch chain.Action {
	case "聊天", "輸出", "澄清", "提問", "選項":
		decision.Handled = true
		decision.DisplayText = chain.Target
	case "重試":
		decision.Handled = true
		decision.DisplayText = RetryPrompt()
	case "複製":
		decision.Handled = true
		decision.DisplayText = "已複製"
	case "貼上":
		decision.Handled = true
		decision.DisplayText = "已讀取剪貼簿"
	case "版控":
		decision.Handled = true
		decision.DisplayText = chain.Target // 實際結果由 App 層覆寫
	case "本機搜尋", "搜尋", "查找", "查詢", "search", "find", "query":
		decision.Handled = true
		decision.DisplayText = chain.Target // 實際搜尋由 App 層覆寫，結果不回灌 LLM
	default:
		return BuiltInDecision{Chain: chain}
	}
	return decision
}

// ValidateStructure enforces the protocol shape before tool routing.
func ValidateStructure(chain ActionChain) error {
	if chain.Action == "" || chain.Target == "" {
		return fmt.Errorf("%w: action and target are required", ErrStructure)
	}
	if strings.Count(chain.Raw, Separator) > 2 {
		return fmt.Errorf("%w: too many separators", ErrStructure)
	}
	return nil
}

// ValidateActionTag reports unknown tags so the Controller can show Review Card.
func ValidateActionTag(action string, registry ActionRegistry) ValidationResult {
	action = NormalizeAction(action)
	if action == "" {
		return ValidationResult{Code: ValidationMalformed, Err: ErrStructure, Reason: "empty action"}
	}
	if registry == nil || !registry.HasActionTag(action) {
		return ValidationResult{
			Code:   ValidationUnknown,
			Err:    ErrUnknownActionTag,
			Reason: fmt.Sprintf("action tag %q is not registered", action),
		}
	}
	return ValidationResult{Code: ValidationOK}
}

// RetryPrompt is the short repair prompt used after structure errors.
func RetryPrompt() string {
	return "你的輸出格式錯誤，請只用 動作ㄌ目標ㄌ下一步 重新輸出。"
}
