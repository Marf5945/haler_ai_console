package replan

import "strings"

// ClassifyFailure 把執行層的工具輸出 / 錯誤轉成結構化 FailureCategory。
// replan policy 只吃這個 enum，不靠自然語言猜測；判定基於 FS 層真實訊號。
//
// 順序由「最敏感」到「最一般」：sensitive/outside 會被 Gate 全域 deny（一律 review），
// 因此優先判定，避免被較一般的類型蓋過。
func ClassifyFailure(action, output string, err error) FailureCategory {
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	text := strings.ToLower(output + " " + errText)

	// 1) 敏感路徑 / 安全邊界（path_guard 系統目錄、路徑穿越、credential 阻擋）→ 全域 deny。
	if containsAny(text, "拒絕系統目錄", "系統目錄", "拒絕含 ..", "system director", "permission denied",
		"credential", "auth_cache", "cookies", "blacklist", "黑名單") {
		return FailureSensitivePath
	}
	// 2) 超出範圍 → 全域 deny。
	if containsAny(text, "超出範圍", "out of scope", "outside scope", "not in scope") {
		return FailureOutsideScope
	}
	// 3) 路徑/檔案不存在。
	if containsAny(text, "檔案不存在", "不存在", "找不到", "no such file", "not exist", "does not exist", "not found") {
		return FailurePathNotFound
	}
	// 4) 結果被截斷 / 過大 / 資訊不足。
	if containsAny(text, "截斷", "truncat", "too large", "size_guard", "exceed", "limit") {
		return FailureTruncated
	}
	// 5) 結果模糊 / 多重命中，需換策略。
	if containsAny(text, "模糊", "歧義", "ambiguous", "multiple match", "多個") {
		return FailureAmbiguous
	}
	// 6) 無命中：成功呼叫但結果為空，或明確「no results」訊號。
	if err == nil && (strings.TrimSpace(output) == "" ||
		containsAny(text, "no match", "no result", "0 result", "無命中", "沒有結果", "empty")) {
		return FailureNoResults
	}

	// 預設：仍有 error 但無法精確歸類 → ambiguous（Gate 後續 allowlist/classifier 仍會把關）。
	if err != nil {
		return FailureAmbiguous
	}
	return FailureNoResults
}

// IsReplanTrigger 回報該失敗類型是否值得觸發一次 replan 嘗試。
// 空字串（未分類）不觸發。
func IsReplanTrigger(c FailureCategory) bool {
	return c != ""
}

// containsAny 回報 text 是否含任一子字串。
func containsAny(text string, subs ...string) bool {
	for _, s := range subs {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}

// ClassifyResult 判定一個「成功（err==nil）」的節點結果是否其實是軟失敗。
// 與 ClassifyFailure 不同：這裡預設回空字串（視為真正成功，不觸發 replan），
// 只有命中明確失敗訊號才回類別。對應「LLM 用自然語言報失敗、程式接手」的分工：
// 偵測失敗只把事情往 Gate 送，永遠不放行任何動作。
func ClassifyResult(text string) FailureCategory {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return "" // 空輸出由執行層另計；這裡只看「有講話但說失敗」
	}
	// 敏感 / 範圍（最優先；→ Gate 全域 deny）。
	if containsAny(t, "allowed workspace", "outside the allowed", "拒絕系統目錄", "權限不足",
		"permission denied", "credential", "存取被拒") {
		return FailureSensitivePath
	}
	if containsAny(t, "out of scope", "outside scope", "超出範圍", "不在允許") {
		return FailureOutsideScope
	}
	// 以下類別可能通往 silent re-route：只在「短答」時才判定，
	// 避免 LLM 長篇道歉/說明文字被誤判成失敗（搭配 prompt 要求短答）。
	// 注意：長度守門「只」套用在這裡，安全訊號（上面）不論長短一律抓。
	if len([]rune(t)) > MaxSoftFailureRunes {
		return ""
	}
	// 路徑 / 檔案不存在或無法存取。
	if containsAny(t, "不存在", "找不到檔", "no such file", "not exist", "does not exist",
		"cannot be accessed", "cannot access", "無法存取", "無法讀取", "file not found") {
		return FailurePathNotFound
	}
	// 無命中 / 查無結果。
	if containsAny(t, "沒有找到", "沒找到", "找不到", "查無", "沒有相關", "沒有結果",
		"no results", "no matching", "no match", "not found", "無命中") {
		return FailureNoResults
	}
	return "" // 視為真正成功
}

// MaxSoftFailureRunes 是 not-found 類別「短答」的長度上限（rune 計）。
// 超過視為 LLM 在長篇說明，不判定為失敗——把判定限縮在明確、簡短的訊號。
const MaxSoftFailureRunes = 40
