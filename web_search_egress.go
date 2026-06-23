// web_search_egress.go — SEC-15：搜尋字串「出境前」機密檢查。
//
// 背景：搜尋查詢（含 targetWithBackground 併入的對話背景）會送到第三方
// 搜尋服務（Tavily / Google / Brave）。若其中含 API 金鑰或高熵機密，
// 等於把機密寄到外部伺服器。
//
// 行為（使用者選擇的「命中先問」模式）：
//  1. 送出前過 memory.RedactBeforeWrite（與寫入記憶同一套遮蔽引擎）。
//  2. 沒命中 → 照常搜尋，零延遲零打擾。
//  3. 命中 → 暫停搜尋，告知偵測到什麼類型的機密，詢問是否用
//     「遮蔽後版本」送出。確認 → 送遮蔽版；取消 → 放棄。
//     原始未遮蔽字串永遠不出境，也不存進 pending。
//
// pending 與確認詞沿用 URL fetch 的同一套（confirmRe / isDeclineText /
// rememberConfirmQuestion），使用者體驗一致。
package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/data/memory"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/websearch"
)

// searchEgressPendingTTL pending 確認有效時間（與 URL fetch 一致）。
const searchEgressPendingTTL = 2 * time.Minute

type pendingSearchEgress struct {
	MaskedQuery string // 只存遮蔽後版本；原文不落地
	Limit       int
	ExpiresAt   time.Time
}

var (
	pendingSearchEgressMu sync.Mutex
	pendingSearchEgresses = map[string]pendingSearchEgress{} // sessionID → pending
)

func storePendingSearchEgress(sessionID, maskedQuery string, limit int) {
	pendingSearchEgressMu.Lock()
	pendingSearchEgresses[sessionID] = pendingSearchEgress{
		MaskedQuery: maskedQuery,
		Limit:       limit,
		ExpiresAt:   time.Now().Add(searchEgressPendingTTL),
	}
	pendingSearchEgressMu.Unlock()
}

// loadPendingSearchEgress 讀取未過期的 pending；過期者順手清除。
func loadPendingSearchEgress(sessionID string) (pendingSearchEgress, bool) {
	pendingSearchEgressMu.Lock()
	defer pendingSearchEgressMu.Unlock()
	p, ok := pendingSearchEgresses[sessionID]
	if !ok {
		return pendingSearchEgress{}, false
	}
	if time.Now().After(p.ExpiresAt) {
		delete(pendingSearchEgresses, sessionID)
		return pendingSearchEgress{}, false
	}
	return p, true
}

func clearPendingSearchEgress(sessionID string) {
	pendingSearchEgressMu.Lock()
	delete(pendingSearchEgresses, sessionID)
	pendingSearchEgressMu.Unlock()
}

// describeEgressHits 把遮蔽記錄整理成人話，例如「OpenAI 金鑰 ×1、疑似機密字串 ×2」。
// 只描述類型與數量，不洩漏任何原始值。
func describeEgressHits(records []memory.RedactionRecord) string {
	counts := map[string]int{}
	for _, r := range records {
		label := r.Type
		switch r.Type {
		case "key_value":
			label = "key=value 機密"
		case "entropy+context", "entropy_suspect":
			label = "疑似機密字串"
		default:
			label = r.Type + " 金鑰"
		}
		counts[label]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s ×%d", k, counts[k]))
	}
	return strings.Join(parts, "、")
}

// gateSearchEgress 出境閘門。回傳 (回應, true) 表示本輪被攔下改為詢問。
func (a *App) gateSearchEgress(req websearch.SearchRequest, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	masked, records := memory.RedactBeforeWrite(req.Query)
	if len(records) == 0 {
		return nil, false
	}
	storePendingSearchEgress(sessionID, masked, req.Limit)
	question := fmt.Sprintf(
		"⚠️ 搜尋內容偵測到敏感資料（%s）。已先擋下，沒有任何東西送出。\n要改用「遮蔽後的版本」搜尋嗎？回覆「好」送出遮蔽版，回覆「取消」放棄這次搜尋。",
		describeEgressHits(records))
	rememberConfirmQuestion(sessionID, question)
	// audit：只記命中數量與類型，不記查詢內容。
	debugtrace.Record("web_search.egress_blocked", traceID, map[string]interface{}{
		"hits": len(records),
	})
	return &skill_step.CLIResponse{Text: question, Action: "web_search", Next: "standby"}, true
}

// maybeResumePendingSearchEgress 處理上一輪閘門詢問的回覆。
// 確認 → 用遮蔽版搜尋；取消 → 放棄；其他文字 → 不攔截，照常路由
// （pending 留到過期，與 URL fetch 行為一致）。
func (a *App) maybeResumePendingSearchEgress(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	p, ok := loadPendingSearchEgress(sessionID)
	if !ok {
		return nil, false
	}
	lower := strings.ToLower(strings.TrimSpace(userText))
	if confirmRe.MatchString(lower) {
		clearPendingSearchEgress(sessionID)
		clearConfirmQuestion(sessionID)
		resp := a.executeWebSearch(websearch.SearchRequest{Query: p.MaskedQuery, Limit: p.Limit}, traceID)
		return &resp, true
	}
	if isDeclineText(lower) {
		clearPendingSearchEgress(sessionID)
		clearConfirmQuestion(sessionID)
		return &skill_step.CLIResponse{Text: "已取消這次搜尋，敏感資料沒有送出。", Next: "standby"}, true
	}
	return nil, false
}
