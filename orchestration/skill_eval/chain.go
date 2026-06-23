// Package skill_eval 是獨立於 DAG 的「評量層」（TASK 31）。
// 職責：評量 skill/sub 是否匹配、是否 drift、是否該產生清理/合併/新增候選。
// DAG 仍負責執行與 dependency；skill_eval 只在 DAG node 執行前後收事件做評量。
//
// 核心不變式：
//   - LLM 只吐最簡單的「動作ㄌ目標ㄌ下一步」，由 translator 轉成 struct。
//   - ㄅㄔㄇㄖ step code、ㄌ 只以結構化欄位存在，永不被當不可信自由文字送回 LLM。
package skill_eval

import "ui_console/shared/actionchain"

// EvalStep 是評量用的單步結構。Code/Requirement 由 evaluator 對 expected_chain 後補，
// LLM 不吐這兩個欄位。
type EvalStep struct {
	Action      string `json:"action"`
	Target      string `json:"target"`
	Next        string `json:"next"`                  // 已正規化（待命 等）
	Code        string `json:"code,omitempty"`        // ㄅ|ㄔ|ㄇ|ㄖ，獨立欄位
	Requirement string `json:"requirement,omitempty"` // OP|RE
	Raw         string `json:"raw,omitempty"`
}

// SkillChain 是一個 skill 的多步序列；16 步序列 = []EvalStep（非一條超長 ㄌ 字串）。
type SkillChain struct {
	SkillID string     `json:"skill_id"`
	Steps   []EvalStep `json:"steps"`
}

// FromActionChain 把 translator parse 好的 ActionChain 轉成單步 EvalStep。
// 不變式 4：Next 在此先 NormalizeNext，確保與 expected 比對時兩邊基準一致。
func FromActionChain(c actionchain.ActionChain) EvalStep {
	return EvalStep{
		Action: c.Action,
		Target: c.Target,
		Next:   actionchain.NormalizeNext(c.Next),
		Raw:    c.Raw,
	}
}

// ParseStep 是 translator 入口：raw LLM 文字 → ActionChain → EvalStep。
// 注意：必須對「raw」parse，不可對 sanitized summary（後者已把 ㄌ 轉義，會 parse 失敗）。
func ParseStep(rawLLM string) (EvalStep, error) {
	chain, err := actionchain.Parse(rawLLM)
	if err != nil {
		return EvalStep{Raw: rawLLM}, err
	}
	return FromActionChain(chain), nil
}
