package actionchain

import "strings"

// ──────────────────────────────────────────────
// 兩種 parser mode（防止 LLM 用 target 夾帶第二層命令）
//   single proposal mode：只一行「動作ㄌ目標」或「找不到」（replan 下一步建議）。
//   chain plan mode：多行，一行一步「動作ㄌ內容ㄌ下一步」（expected_chain / 偏離評估 / cache）。
// 共同硬規則：每行只 parse 一次行級結構；target 原封不動，絕不二次 dispatch。
// ──────────────────────────────────────────────

// 長度上限（rule 4：防 prompt injection / 整篇網頁塞進一步）。
const (
	MaxTargetRunes = 500  // 單步 target 上限（rune 計）
	MaxChainSteps  = 16   // 整條 chain 步數上限
	MaxOutputBytes = 8192 // 整體輸出 bytes 上限
)

// NoProposalToken 是「找不到」單步信號（replan 無法提出新方式）。
const NoProposalToken = "找不到"

// StepError 標記某一行的解析問題（供呼叫端決定 review）。
type StepError struct {
	Line   int    `json:"line"`
	Raw    string `json:"raw"`
	Reason string `json:"reason"`
}

// ParseSingleResult 是 single proposal mode 的結果。
type ParseSingleResult struct {
	Chain      ActionChain // 解析出的 動作ㄌ目標（NoProposal=true 時無意義）
	NoProposal bool        // 模型回「找不到」
	Err        *StepError  // 格式問題 → 呼叫端應走 review
}

// ParseSingleLine 解析 single proposal mode：只取第一個非空行。
//   - 「找不到」→ NoProposal=true。
//   - 必須剛好 2 段（動作ㄌ目標）；3 段或壞結構 → Err（呼叫端 review）。
//   - target 超長 → Err。
func ParseSingleLine(output string) ParseSingleResult {
	if len(output) > MaxOutputBytes {
		return ParseSingleResult{Err: &StepError{Reason: "output exceeds byte limit"}}
	}
	line := firstNonEmptyLine(output)
	if line == "" {
		return ParseSingleResult{Err: &StepError{Reason: "empty output"}}
	}
	if strings.TrimSpace(line) == NoProposalToken {
		return ParseSingleResult{NoProposal: true}
	}
	chain, err := Parse(line)
	if err != nil {
		return ParseSingleResult{Err: &StepError{Raw: line, Reason: err.Error()}}
	}
	// 對齊系統真實格式：接受 動作ㄌ目標 或 動作ㄌ目標ㄌ下一步（取 動作+目標，下一步僅參考）。
	if len([]rune(chain.Target)) > MaxTargetRunes {
		return ParseSingleResult{Err: &StepError{Raw: line, Reason: "target exceeds length limit"}}
	}
	return ParseSingleResult{Chain: chain}
}

// ParseChainResult 是 chain plan mode 的結果。
type ParseChainResult struct {
	Steps  []ActionChain // 逐行解析出的步驟（一行一步）
	Errors []StepError   // 壞行（呼叫端可據此決定 review；不影響其他行）
}

// ParseChainLines 解析 chain plan mode：換行分隔，一行一步。
// 每行各自 Parse；壞行記入 Errors 但不影響其他行（每行獨立、只 parse 一次）。
func ParseChainLines(output string) ParseChainResult {
	res := ParseChainResult{}
	if len(output) > MaxOutputBytes {
		res.Errors = append(res.Errors, StepError{Reason: "output exceeds byte limit"})
		return res
	}
	lineNo := 0
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lineNo++
		if len(res.Steps) >= MaxChainSteps {
			res.Errors = append(res.Errors, StepError{Line: lineNo, Raw: line, Reason: "exceeds max chain steps"})
			break
		}
		chain, err := Parse(line)
		if err != nil {
			res.Errors = append(res.Errors, StepError{Line: lineNo, Raw: line, Reason: err.Error()})
			continue
		}
		if len([]rune(chain.Target)) > MaxTargetRunes {
			res.Errors = append(res.Errors, StepError{Line: lineNo, Raw: line, Reason: "target exceeds length limit"})
			continue
		}
		res.Steps = append(res.Steps, chain)
	}
	return res
}
