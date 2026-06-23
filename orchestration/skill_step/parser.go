// parser.go 負責將使用者輸入的動作字串解析成結構化的 ActionTarget，
// 再透過評分與模糊判定管線，從已歸檔的 skill manifest 中選出最合適的 skill。
//
// 核心概念：
//   - 「動作ㄌ目標」格式：使用注音符號 ㄌ（U+310C）作為動作與目標的分隔符，
//     例如「查詢ㄌ天氣」會被解析為 Action="查詢"、Target="天氣"。
//   - 這個格式刻意選用罕見字元，避免使用者在正常輸入中誤觸分隔符。
package skill_step

import (
	"fmt"
	"strings"
	"unicode"
)

// ActionTarget 保存解析後的動作與目標，是 Router.Resolve 的輸入單元。
//
// 欄位說明：
//   - Action：動作關鍵字，例如「查詢」「轉換」「匯出」
//   - Target：操作對象，例如「天氣」「pdf」「使用者資料」
//   - Raw：原始輸入字串，保留給日誌或除錯使用，不參與路由評分
type ActionTarget struct {
	Action string // 動作關鍵字（ㄌ 左側）
	Target string // 目標物件（ㄌ 右側）
	Raw    string // 原始輸入，未經修改，供日誌記錄使用
}

// separator 是動作與目標的分隔字元，固定為注音符號 ㄌ（U+310C）。
// 選用此字元的原因：在中文輸入情境下極少被使用者主動輸入，
// 因此可以安全作為協定層的分隔符，不會與使用者內容衝突。
const separator = "ㄌ"

// ParseActionTarget 將 input 字串按 ㄌ 分割，解析出 Action 與 Target。
//
// 解析規則：
//  1. input 不得為空字串
//  2. input 必須包含 ㄌ 分隔符，否則回傳錯誤
//  3. 分割後的兩段分別經過 sanitize 清洗（去首尾空白、移除控制字元）
//
// 呼叫端注意：
//   - 若使用者輸入不符合格式，應向使用者顯示友善提示，而非直接顯示 error message
//   - 不要在 UI 層暴露 "skill_step" 等技術前綴
func ParseActionTarget(input string) (ActionTarget, error) {
	if input == "" {
		return ActionTarget{}, fmt.Errorf("skill_step: ParseActionTarget: input is empty")
	}

	// 尋找分隔符位置；沒有找到代表輸入格式不符合預期
	idx := strings.Index(input, separator)
	if idx < 0 {
		return ActionTarget{}, fmt.Errorf("skill_step: ParseActionTarget: separator ㄌ not found in %q", input)
	}

	// 分隔符左側為動作、右側為目標，各自清洗後回傳
	action := sanitize(input[:idx])
	target := sanitize(input[idx+len(separator):])

	return ActionTarget{
		Action: action,
		Target: target,
		Raw:    input, // Raw 保留原始值，不做任何修改
	}, nil
}

// sanitize 清洗輸入字串：去除首尾空白，並移除所有 Unicode 控制字元。
//
// 為什麼要移除控制字元：
//   - 控制字元（如 \r \t \x00）若進入評分管線，可能造成假比對或安全疑慮。
//   - 這裡使用 unicode.IsControl 判斷，涵蓋 U+0000–U+001F 與 U+007F–U+009F 範圍。
func sanitize(s string) string {
	s = strings.TrimSpace(s) // 先去首尾空白
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsControl(r) {
			b.WriteRune(r) // 只保留非控制字元
		}
	}
	return b.String()
}
