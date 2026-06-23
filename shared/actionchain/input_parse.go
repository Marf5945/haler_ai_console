// input_parse.go — 「輸入」動作的專用寬鬆 parser（v3.1.6 M1）。
// 一般動作維持 Parse 的嚴格三槽切法；只有 輸入 用 first/last separator，
// 讓 target（通常是 JSON 參數）可以安全內含 ㄌ，且不把多命令注入救回來。
package actionchain

import (
	"fmt"
	"strings"
)

// InputAction 是回填 PendingInput 參數的保留動作名。
const InputAction = "輸入"

// ParseInputLine 解析 輸入ㄌ<target 可含ㄌ>ㄌ待命。
// 規則：action = 第一個 ㄌ 前；next = 最後一個 ㄌ 後；target = 中間全部。
// 非 輸入 開頭一律拒絕——其他動作不得走寬鬆切法。
func ParseInputLine(output string) (ActionChain, error) {
	raw := strings.TrimSpace(output)
	first := strings.Index(raw, Separator)
	last := strings.LastIndex(raw, Separator)
	if first < 0 || last == first {
		return ActionChain{Raw: output}, fmt.Errorf("%w: input line needs leading and trailing separators", ErrStructure)
	}
	action := NormalizeAction(strings.TrimSpace(raw[:first]))
	if action != InputAction {
		return ActionChain{Raw: output}, fmt.Errorf("%w: lenient parse is reserved for %s", ErrStructure, InputAction)
	}
	target := strings.TrimSpace(raw[first+len(Separator) : last])
	if target == "" {
		return ActionChain{Raw: output}, fmt.Errorf("%w: empty input target", ErrStructure)
	}
	return ActionChain{
		Action: action,
		Target: target,
		Next:   NormalizeNext(strings.TrimSpace(raw[last+len(Separator):])),
		Raw:    raw,
	}, nil
}

// IsInputLine 快速判斷一行是否是 輸入 控制行（不驗證完整結構）。
func IsInputLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	first := strings.Index(trimmed, Separator)
	if first < 0 {
		return false
	}
	return NormalizeAction(strings.TrimSpace(trimmed[:first])) == InputAction
}
