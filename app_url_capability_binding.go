// app_url_capability_binding.go — SEC-06 URL Capability Layer 的 Wails bindings
// 與 action-chain 接線。
//
// 管線：url_source → source_trust(allowlist) → safefetcher 決策 → audit → action。
// 所有「讀取網址內容」行為只能走 FetchURLContent / maybeHandleURLFetch，
// 不要另開 http client。
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/domain/url_source"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/executil"
	"ui_console/shared/safefetcher"
)

// ── registry 單例（記憶體 + JSONL audit）──

var (
	urlRegistryOnce sync.Once
	urlRegistry     *url_source.Registry
)

func getURLRegistry() *url_source.Registry {
	urlRegistryOnce.Do(func() {
		auditPath := filepath.Join(appDataRoot(), "url_source", "audit.jsonl")
		urlRegistry = url_source.NewRegistry(auditPath)
	})
	return urlRegistry
}

// ── Wails bindings ──

// RegisterURLOccurrence 前端回報 URL 出現（貼上 / LLM 回答渲染時）。
// 回傳 occurrence 與該 URL 的最低信任來源，供 UI 畫來源 chip。
func (a *App) RegisterURLOccurrence(rawURL, source, sessionID, traceID string) (interface{}, error) {
	src, ok := url_source.ValidSource(source)
	if !ok {
		return nil, fmt.Errorf("未知的 url_source: %q", source)
	}
	rec, err := getURLRegistry().Record(rawURL, src, sessionID, traceID, a.trustLabelFor(rawURL))
	if err != nil {
		return nil, err
	}
	lowest, _ := getURLRegistry().LowestTrustSource(rawURL)
	return frontendDTO(map[string]interface{}{
		"record":              rec,
		"lowest_trust_source": string(lowest),
		"effective_risk_tier": lowest.RiskTier(),
	}), nil
}

// GetURLProvenance 查 URL 的最低信任來源與風險層級（chip 顯示用）。
func (a *App) GetURLProvenance(rawURL string) (interface{}, error) {
	lowest, found := getURLRegistry().LowestTrustSource(rawURL)
	if !found {
		return frontendDTO(map[string]interface{}{"known": false}), nil
	}
	return frontendDTO(map[string]interface{}{
		"known":               true,
		"lowest_trust_source": string(lowest),
		"effective_risk_tier": lowest.RiskTier(),
	}), nil
}

// FetchURLContent 「讀取此網址內容」按鈕入口。userConfirmed=true 代表使用者
// 明確點擊（視為確認）。風險以該 URL 的最低信任來源計，不以呼叫端宣稱為準。
func effectiveURLSource(rawURL string, fallback url_source.Source) url_source.Source {
	src := fallback
	if lowest, found := getURLRegistry().LowestTrustSource(rawURL); found && lowest.TrustRank() < src.TrustRank() {
		src = lowest
	}
	return src
}

func (a *App) FetchURLContent(rawURL, source, sessionID, traceID string, userConfirmed bool) (interface{}, error) {
	src, ok := url_source.ValidSource(source)
	if !ok {
		return nil, fmt.Errorf("未知的 url_source: %q", source)
	}
	if _, err := getURLRegistry().Record(rawURL, src, sessionID, traceID, a.trustLabelFor(rawURL)); err != nil {
		return nil, err
	}
	// 洗白防護：取歷史最低信任來源做決策
	if lowest, found := getURLRegistry().LowestTrustSource(rawURL); found && lowest.TrustRank() < src.TrustRank() {
		src = lowest
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := safefetcher.FetchURLForLLM(ctx, safefetcher.FetchRequest{
		RawURL:        rawURL,
		Source:        src,
		Purpose:       "user_button",
		SessionID:     sessionID,
		TraceID:       traceID,
		UserConfirmed: userConfirmed,
		Allowlisted:   a.isHostAllowlisted(rawURL),
	})
	if err != nil {
		return nil, err
	}
	debugtrace.Record("url_fetch.ok", traceID, map[string]interface{}{
		"host": result.FinalHost, "source": string(src), "hash": result.ContentHash,
	})
	return frontendDTO(result), nil
}

// trustLabelFor 取 source_trust 分類標籤（失敗回空字串，不阻斷）。
func (a *App) trustLabelFor(rawURL string) string {
	defer func() { _ = recover() }() // 防 allowlistStore 未初始化（測試環境）
	ev := a.ClassifySource(rawURL, "", nil)
	return string(ev.SourceTrustLabel)
}

// isHostAllowlisted 查 source_trust 專案白名單是否 active。
func (a *App) isHostAllowlisted(rawURL string) bool {
	defer func() { _ = recover() }()
	_, host, err := url_source.HashURL(rawURL)
	if err != nil || a.allowlistStore == nil {
		return false
	}
	return a.allowlistStore.CheckStatus(host) == "active"
}

// ── action-chain 接線：使用者訊息含 URL + 讀取意圖 → 確認 → 代抓 ──

var (
	pendingURLFetchMu sync.Mutex
	pendingURLFetches = map[string]pendingURLFetch{} // sessionID → pending
)

type pendingURLFetch struct {
	RawURL    string
	Source    url_source.Source
	ExpiresAt time.Time
}

var (
	urlInTextRe   = regexp.MustCompile(`https?://[^\s'"<>）)】\]]+`)
	fetchIntentRe = regexp.MustCompile(`讀取|摘要|總結|內容|read|summar`)
	openIntentRe  = regexp.MustCompile(`開啟|打開|前往|跑到|跳到|到這|到那|位置|資料夾|open|go to|show`)
	// 確認詞：句首出現即可，容許「好啊麻煩你」「好，謝謝」等口語尾綴。
	confirmRe = regexp.MustCompile(`^\s*(好的|好啊|好喔|好|是的|是|沒問題|確認|可以|要|ok|okay|yes|yep|sure|y)([，,。.!！、~\s].*)?$`)
	// 否定詞：明確拒絕時清掉 pending，不要把「不用了」當沒事忽略。
	declineRe = regexp.MustCompile(`^\s*(不用|不要|不|別|算了|取消|no|nope|cancel|n)([，,。.!！、~\s].*)?$`)
)

// isURLOnlyQuery 判斷整段（去空白後）就是單一 URL、沒有其他文字或意圖詞。
func isURLOnlyQuery(text string) bool {
	t := strings.TrimSpace(text)
	return t != "" && urlInTextRe.FindString(t) == t
}

// isDeclineText 判斷是否為取消意圖（僅在有 pending 時呼叫）。
// 句首匹配之外，短句（≤12 字）包含明確取消詞也算——「我要取消」「幫我取消」。
// 取消是安全方向，過觸發的代價只是使用者重問一次。
func isDeclineText(lower string) bool {
	if declineRe.MatchString(lower) {
		return true
	}
	if len([]rune(lower)) <= 12 {
		for _, kw := range []string{"取消", "算了", "不用", "不要", "cancel"} {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

// ── pending 確認上下文：給 LLM judge 的「上一輪系統在等什麼」摘要 ──
// 確定性攔截（isDeclineText/confirmRe）優先且免 LLM；這層處理攔不到的情況：
// pending 已過期、或回覆語意模糊時，judge prompt 帶上下文而非讓 LLM 瞎猜。

type lastConfirmQuestion struct {
	Question string
	AskedAt  time.Time
}

var (
	lastConfirmMu sync.Mutex
	lastConfirms  = map[string]lastConfirmQuestion{} // sessionID → 最近的系統確認問句
)

func rememberConfirmQuestion(sessionID, question string) {
	lastConfirmMu.Lock()
	lastConfirms[sessionID] = lastConfirmQuestion{Question: question, AskedAt: time.Now()}
	lastConfirmMu.Unlock()
}

func clearConfirmQuestion(sessionID string) {
	lastConfirmMu.Lock()
	delete(lastConfirms, sessionID)
	lastConfirmMu.Unlock()
}

// pendingConfirmPromptContext 組 judge prompt 的 pending 摘要；問句後 10 分鐘內有效
// （比 pending 本身的 2 分鐘長：過期後使用者回「取消」，LLM 仍知道在取消什麼）。
func pendingConfirmPromptContext(sessionID string) string {
	lastConfirmMu.Lock()
	lc, ok := lastConfirms[sessionID]
	lastConfirmMu.Unlock()
	if !ok || time.Since(lc.AskedAt) > 10*time.Minute {
		return ""
	}
	return fmt.Sprintf("\n[系統提供: pending_confirm]\n上一輪系統向使用者提出確認：%q（約 %d 秒前）。\n若使用者本句是在回應此確認（同意、拒絕、取消、猶豫、追問），輸出 閒聊ㄌ<說明目前狀態的簡短回應，例如已取消或請重新提出請求>；不要路由到操作/查詢/搜尋/網路。\n[/系統提供: pending_confirm]\n", lc.Question, int(time.Since(lc.AskedAt).Seconds()))
}

type pendingResourceAction struct {
	Kind      string
	Target    string
	ExpiresAt time.Time
}

var (
	pendingResourceMu      sync.Mutex
	pendingResourceActions = map[string]pendingResourceAction{} // sessionID → pending
)

// fileTargetAskTTL 是「你要處理哪個檔案」問句的有效期。問過一次後，
// 若使用者在這段時間內又送了一句仍沒帶路徑的訊息，就放行到 CLI，
// 避免 deterministic gate 對同一句反覆攔截、永遠到不了 CLI。
const fileTargetAskTTL = 2 * time.Minute

var (
	pendingFileAskMu      sync.Mutex
	pendingFileTargetAsks = map[string]time.Time{} // sessionID → 問過「哪個檔案」的時間
)

// maybeHandleResourceGate is the deterministic resource layer in front of LLM routing.
// It handles concrete resources (URLs and local paths) before fuzzy tool selection.
func (a *App) maybeHandleResourceGate(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	text := strings.TrimSpace(userText)
	if text == "" {
		return nil, false
	}
	// need_confirm 待確認：使用者回「要/取消」優先由此接手（在 LLM 路由前），
	// 避免確認回覆被丟回三段式路由造成 skill 權限確認迴圈。
	if resp, handled := a.maybeHandlePendingSkillConfirm(text, sessionID, traceID); handled {
		return resp, true
	}
	// 電料BOM 互動收集中：每一回合（機台/電料項/要不要補）在 LLM 路由前直接接手，
	// 避免「料號 數量」這種輸入被三段式路由誤分類。
	if resp, handled := a.maybeHandlePendingDianliaoBom(text, sessionID, traceID); handled {
		return resp, true
	}
	if resp, handled := a.maybeHandlePendingResourceAction(text, sessionID, traceID); handled {
		return resp, true
	}
	if resp, handled := a.maybeHandleURLFetch(text, sessionID, traceID); handled {
		return resp, true
	}
	if resp, handled := a.maybeHandleURLOpen(text, sessionID, traceID); handled {
		return resp, true
	}
	// 純 URL 無意圖詞：先問「讀取還是開啟」，避免被 LLM 當搜尋字串丟去網路搜尋。
	if resp, handled := a.maybeHandleBareURL(text, sessionID, traceID); handled {
		return resp, true
	}
	if resp, handled := a.maybeHandleLocalPathResource(text, sessionID, traceID); handled {
		return resp, true
	}
	if asksForFileWithoutTarget(text) {
		now := time.Now()
		pendingFileAskMu.Lock()
		askedAt, asked := pendingFileTargetAsks[sessionID]
		stillPending := asked && now.Sub(askedAt) <= fileTargetAskTTL
		if stillPending {
			// 已問過且仍在效期內，但這句還是沒帶路徑 → 放行到 CLI，別卡死。
			delete(pendingFileTargetAsks, sessionID)
			pendingFileAskMu.Unlock()
			debugtrace.Record("resource_gate.file_target.passthrough", traceID, map[string]interface{}{
				"session_id": sessionID,
			})
			return nil, false
		}
		// 第一次詢問：記下時間，讓下一輪可去重。
		pendingFileTargetAsks[sessionID] = now
		pendingFileAskMu.Unlock()
		return &skill_step.CLIResponse{
			Text: "你要我處理哪個檔案？請貼本機路徑、拖入檔案，或說已引用檔案名稱。",
		}, true
	}
	return nil, false
}

func (a *App) maybeHandlePendingResourceAction(text, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	pendingResourceMu.Lock()
	pending, has := pendingResourceActions[sessionID]
	if has && time.Now().After(pending.ExpiresAt) {
		delete(pendingResourceActions, sessionID)
		has = false
	}
	pendingResourceMu.Unlock()
	if !has {
		return nil, false
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	if isDeclineText(lower) {
		pendingResourceMu.Lock()
		delete(pendingResourceActions, sessionID)
		pendingResourceMu.Unlock()
		clearConfirmQuestion(sessionID)
		return &skill_step.CLIResponse{Text: "好，已取消。"}, true
	}
	// 純 URL 待選：使用者回「讀取」走 safefetcher，回「開啟」交系統瀏覽器。
	if pending.Kind == "choose_url" {
		if fetchIntentRe.MatchString(text) || strings.Contains(text, "讀取") {
			pendingResourceMu.Lock()
			delete(pendingResourceActions, sessionID)
			pendingResourceMu.Unlock()
			clearConfirmQuestion(sessionID)
			return a.runConfirmedURLFetch(pendingURLFetch{RawURL: pending.Target, Source: url_source.SourceUserPaste}, sessionID, traceID), true
		}
		if openIntentRe.MatchString(text) || strings.Contains(text, "開啟") {
			pending.Kind = "open_url"
			clearConfirmQuestion(sessionID)
			return a.executePendingResourceAction(pending, sessionID, traceID), true
		}
		return &skill_step.CLIResponse{Text: "請回覆「讀取」或「開啟」，或回覆「取消」。"}, true
	}
	if pending.Kind == "choose_local_path" {
		if openIntentRe.MatchString(text) || strings.Contains(text, "開啟") {
			pending.Kind = "open_local_path"
		} else if fetchIntentRe.MatchString(text) || strings.Contains(text, "讀取") {
			pending.Kind = "read_local_file"
		} else {
			return &skill_step.CLIResponse{Text: "請回覆「讀取」或「開啟」，或回覆「取消」。"}, true
		}
		pendingResourceMu.Lock()
		pendingResourceActions[sessionID] = pending
		pendingResourceMu.Unlock()
		return a.confirmPendingResourceAction(pending, sessionID, traceID), true
	}
	if pending.Kind == "read_local_file" && openIntentRe.MatchString(text) {
		pending.Kind = "open_local_path"
		return a.executePendingResourceAction(pending, sessionID, traceID), true
	}
	if confirmRe.MatchString(lower) {
		pendingResourceMu.Lock()
		delete(pendingResourceActions, sessionID)
		pendingResourceMu.Unlock()
		clearConfirmQuestion(sessionID)
		return a.executePendingResourceAction(pending, sessionID, traceID), true
	}
	return nil, false
}

func (a *App) confirmPendingResourceAction(pending pendingResourceAction, sessionID, traceID string) *skill_step.CLIResponse {
	pending.ExpiresAt = time.Now().Add(5 * time.Minute)
	pendingResourceMu.Lock()
	pendingResourceActions[sessionID] = pending
	pendingResourceMu.Unlock()
	var resp *skill_step.CLIResponse
	switch pending.Kind {
	case "open_url":
		resp = &skill_step.CLIResponse{
			Text:   fmt.Sprintf("要我開啟 %s 嗎？回覆「好」確認。", pending.Target),
			Action: "開啟",
			Target: pending.Target,
			Next:   "確認",
		}
	case "open_local_path":
		resp = &skill_step.CLIResponse{
			Text:   fmt.Sprintf("要我開啟這個本機位置嗎？\n%s\n回覆「好」確認。", pending.Target),
			Action: "開啟",
			Target: pending.Target,
			Next:   "確認",
		}
	case "read_local_file":
		resp = &skill_step.CLIResponse{
			Text:   fmt.Sprintf("要我讀取這個本機檔案的內容嗎？\n%s\n回覆「好」確認；如果只是要到檔案位置，回覆「開啟」。", pending.Target),
			Action: "讀取",
			Target: pending.Target,
			Next:   "確認",
		}
	default:
		resp = &skill_step.CLIResponse{Text: "請回覆「讀取」或「開啟」，或回覆「取消」。"}
	}
	// 記住問句：LLM judge 可在攔截層失效時（過期/語意模糊）仍有上下文。
	rememberConfirmQuestion(sessionID, resp.Text)
	return resp
}

func (a *App) executePendingResourceAction(p pendingResourceAction, sessionID, traceID string) *skill_step.CLIResponse {
	switch p.Kind {
	case "open_url":
		if err := a.OpenExternalURL(p.Target); err != nil {
			return &skill_step.CLIResponse{Text: fmt.Sprintf("無法開啟該網址：%v", err)}
		}
		return &skill_step.CLIResponse{Text: "已交給系統瀏覽器開啟。", Action: "開啟", Target: p.Target, Next: "standby"}
	case "open_local_path":
		if err := openLocalPathInShell(p.Target); err != nil {
			return &skill_step.CLIResponse{Text: fmt.Sprintf("無法開啟這個本機位置：%v", err)}
		}
		return &skill_step.CLIResponse{Text: "已開啟本機位置。", Action: "開啟", Target: p.Target, Next: "standby"}
	case "read_local_file":
		return a.importAndReadLocalFile(p.Target, traceID)
	default:
		return &skill_step.CLIResponse{Text: "這個待確認動作已失效，請重新輸入。"}
	}
}

func (a *App) maybeHandleURLOpen(text, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	rawURL := urlInTextRe.FindString(text)
	if rawURL == "" || !openIntentRe.MatchString(text) || fetchIntentRe.MatchString(text) {
		return nil, false
	}
	pending := pendingResourceAction{Kind: "open_url", Target: rawURL, ExpiresAt: time.Now().Add(5 * time.Minute)}
	debugtrace.Record("resource_gate.url_open.pending", traceID, map[string]interface{}{"url": rawURL})
	return a.confirmPendingResourceAction(pending, sessionID, traceID), true
}

// maybeHandleBareURL 處理「整段就是一個 URL、沒有讀取/開啟意圖」的情況：
// 記 occurrence，建 choose_url pending，問使用者要讀取還是開啟。
func (a *App) maybeHandleBareURL(text, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	if !isURLOnlyQuery(text) {
		return nil, false
	}
	rawURL := urlInTextRe.FindString(text)
	rec, err := getURLRegistry().Record(rawURL, url_source.SourceUserPaste, sessionID, traceID, a.trustLabelFor(rawURL))
	if err != nil {
		return nil, false
	}
	pendingResourceMu.Lock()
	pendingResourceActions[sessionID] = pendingResourceAction{
		Kind: "choose_url", Target: rawURL, ExpiresAt: time.Now().Add(2 * time.Minute),
	}
	pendingResourceMu.Unlock()
	debugtrace.Record("resource_gate.bare_url.pending", traceID, map[string]interface{}{"host": rec.NormalizedHost})
	question := fmt.Sprintf("你貼了 %s。要我「讀取」它的內容，還是用瀏覽器「開啟」它？（或回覆「取消」）", rec.NormalizedHost)
	rememberConfirmQuestion(sessionID, question)
	return &skill_step.CLIResponse{
		Text: question,
		Next: "確認",
	}, true
}

func (a *App) maybeHandleLocalPathResource(text, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	path, ok := extractExistingLocalPath(text)
	if !ok {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return &skill_step.CLIResponse{Text: fmt.Sprintf("找不到這個本機路徑：%s", path)}, true
	}
	if fetchIntentRe.MatchString(text) {
		if info.IsDir() {
			pending := pendingResourceAction{Kind: "open_local_path", Target: path, ExpiresAt: time.Now().Add(5 * time.Minute)}
			return a.confirmPendingResourceAction(pending, sessionID, traceID), true
		}
		pending := pendingResourceAction{Kind: "read_local_file", Target: path, ExpiresAt: time.Now().Add(5 * time.Minute)}
		debugtrace.Record("resource_gate.local_read.pending", traceID, map[string]interface{}{"path": path})
		return a.confirmPendingResourceAction(pending, sessionID, traceID), true
	}
	if openIntentRe.MatchString(text) {
		pending := pendingResourceAction{Kind: "open_local_path", Target: path, ExpiresAt: time.Now().Add(5 * time.Minute)}
		debugtrace.Record("resource_gate.local_open.pending", traceID, map[string]interface{}{"path": path})
		return a.confirmPendingResourceAction(pending, sessionID, traceID), true
	}
	pending := pendingResourceAction{Kind: "choose_local_path", Target: path, ExpiresAt: time.Now().Add(5 * time.Minute)}
	pendingResourceMu.Lock()
	pendingResourceActions[sessionID] = pending
	pendingResourceMu.Unlock()
	return &skill_step.CLIResponse{
		Text: fmt.Sprintf("我看到這個本機路徑：\n%s\n你要我「讀取」內容，還是「開啟」位置？", path),
	}, true
}

// maybeHandleURLFetch 在 web search 之前攔截兩種情況：
//  1. session 有 pending fetch 且使用者回覆確認 → 執行抓取
//  2. 使用者訊息含 URL + 讀取意圖 → 記 occurrence、建 pending、回確認問句
//
// 由 maybeHandleWebSearch 開頭呼叫（單一掛載點，覆蓋所有路由）。
func (a *App) maybeHandleURLFetch(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	text := strings.TrimSpace(userText)

	// — 1. pending 確認 —
	pendingURLFetchMu.Lock()
	pending, has := pendingURLFetches[sessionID]
	if has && time.Now().After(pending.ExpiresAt) {
		delete(pendingURLFetches, sessionID)
		has = false
	}
	pendingURLFetchMu.Unlock()

	if has {
		lower := strings.ToLower(text)
		if isDeclineText(lower) {
			pendingURLFetchMu.Lock()
			delete(pendingURLFetches, sessionID)
			pendingURLFetchMu.Unlock()
			clearConfirmQuestion(sessionID)
			return &skill_step.CLIResponse{Text: "好，已取消。"}, true
		}
		if confirmRe.MatchString(lower) {
			pendingURLFetchMu.Lock()
			delete(pendingURLFetches, sessionID)
			pendingURLFetchMu.Unlock()
			clearConfirmQuestion(sessionID)
			return a.runConfirmedURLFetch(pending, sessionID, traceID), true
		}
	}

	// — 2. 新的「URL + 讀取意圖」—
	rawURL := urlInTextRe.FindString(text)
	if rawURL == "" || !fetchIntentRe.MatchString(text) {
		return nil, false
	}
	// 使用者親自輸入 → user_paste；歷史最低信任在 FetchURLContent 階段套用
	rec, err := getURLRegistry().Record(rawURL, url_source.SourceUserPaste, sessionID, traceID, a.trustLabelFor(rawURL))
	if err != nil {
		return nil, false // URL 解析失敗 → 交回一般路由
	}
	pendingURLFetchMu.Lock()
	pendingURLFetches[sessionID] = pendingURLFetch{
		RawURL:    rawURL,
		Source:    effectiveURLSource(rawURL, rec.Source),
		ExpiresAt: time.Now().Add(2 * time.Minute), // 縮短窗口，降低 stale「好」誤觸
	}
	pendingURLFetchMu.Unlock()
	debugtrace.Record("url_fetch.pending", traceID, map[string]interface{}{
		"host": rec.NormalizedHost, "source": string(effectiveURLSource(rawURL, rec.Source)),
	})
	question := fmt.Sprintf("要我讀取 %s 的內容嗎？回覆「好」確認，我只會抓取公開文字內容（不帶登入狀態）。", rec.NormalizedHost)
	rememberConfirmQuestion(sessionID, question)
	return &skill_step.CLIResponse{
		Text:   question,
		Action: "讀取",
		Target: rec.NormalizedHost,
		Next:   "確認",
	}, true
}

// runConfirmedURLFetch 執行已確認的抓取並組回覆。
func (a *App) runConfirmedURLFetch(p pendingURLFetch, sessionID, traceID string) *skill_step.CLIResponse {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	src := p.Source
	if lowest, found := getURLRegistry().LowestTrustSource(p.RawURL); found && lowest.TrustRank() < src.TrustRank() {
		src = lowest
	}
	result, err := safefetcher.FetchURLForLLM(ctx, safefetcher.FetchRequest{
		RawURL:        p.RawURL,
		Source:        src,
		Purpose:       "action_chain",
		SessionID:     sessionID,
		TraceID:       traceID,
		UserConfirmed: true,
		Allowlisted:   a.isHostAllowlisted(p.RawURL),
	})
	if err != nil {
		return &skill_step.CLIResponse{Text: fmt.Sprintf("無法讀取該網址：%v", err)}
	}
	debugtrace.Record("url_fetch.ok", traceID, map[string]interface{}{
		"host": result.FinalHost, "source": string(src), "hash": result.ContentHash,
	})
	header := result.Title
	if header == "" {
		header = result.FinalHost
	}
	body := result.Text
	if result.Truncated {
		body += "\n（內容過長已截斷）"
	}
	warning := ""
	if result.SourceWarning != "" {
		warning = "\n⚠ " + result.SourceWarning
	}
	return &skill_step.CLIResponse{
		Text:   fmt.Sprintf("【%s】\n%s%s", header, body, warning),
		Action: "讀取",
		Target: result.FinalHost,
		Next:   "standby",
	}
}

var windowsPathInTextRe = regexp.MustCompile(`(?i)(file://[^\s'"<>）)】\]]+|[a-z]:\\[^\r\n"<>|]+|\\\\[^\r\n"<>|]+|/[^\r\n"<>|]+)`)

func extractExistingLocalPath(text string) (string, bool) {
	raw := strings.TrimSpace(windowsPathInTextRe.FindString(text))
	if raw == "" {
		return "", false
	}
	candidates := localPathCandidates(raw)
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	if len(candidates) > 0 {
		return candidates[0], true
	}
	return "", false
}

func localPathCandidates(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "。.!！?？,，、;；\"'`")
	if strings.HasPrefix(strings.ToLower(raw), "file://") {
		if parsed, err := url.Parse(raw); err == nil {
			raw = parsed.Path
			if runtime.GOOS == "windows" {
				raw = strings.TrimPrefix(raw, "/")
				raw = strings.ReplaceAll(raw, "/", "\\")
			}
		}
	}
	raw = expandUserPath(raw)
	candidates := []string{strings.TrimSpace(raw)}
	parts := strings.Fields(raw)
	for i := len(parts) - 1; i > 0; i-- {
		candidates = append(candidates, strings.TrimSpace(strings.Join(parts[:i], " ")))
	}
	out := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.Trim(candidate, "。.!！?？,，、;；\"'`")
		if candidate == "" || seen[strings.ToLower(candidate)] {
			continue
		}
		seen[strings.ToLower(candidate)] = true
		out = append(out, candidate)
	}
	return out
}

// fileContainerTerms 是「介面/容器」詞：指的是檔案總管這類程式或資料夾介面，
// 而非「某一份要處理的檔案」。它們本身含有「檔案/file」字樣，若不先剔除會把
// 「在檔案總管中切換書籤資料夾」這種自動化請求誤判成「你要處理哪個檔案」。
var fileContainerTerms = []string{
	"檔案總管", "檔案管理員", "檔案管理器", "檔案管理",
	"file explorer", "file manager", "windows explorer", "finder",
}

func asksForFileWithoutTarget(text string) bool {
	if windowsPathInTextRe.MatchString(text) || urlInTextRe.MatchString(text) {
		return false
	}
	// 先把容器詞剔除，再判斷句中是否還有「檔案」名詞。
	stripped := text
	lowerStripped := strings.ToLower(text)
	for _, term := range fileContainerTerms {
		stripped = strings.ReplaceAll(stripped, term, "")
		lowerStripped = strings.ReplaceAll(lowerStripped, strings.ToLower(term), "")
	}
	hasFileNoun := strings.Contains(stripped, "檔案") || strings.Contains(stripped, "文件") || strings.Contains(stripped, "檔") ||
		strings.Contains(lowerStripped, "file") || strings.Contains(lowerStripped, "document")
	if !hasFileNoun {
		return false
	}
	// 意圖只看「針對某份檔案」的動作（讀取/開啟/詢問哪份）。
	// 不採用 openIntentRe，因為它含「資料夾/位置/前往」等資料夾導航詞，
	// 那些是針對資料夾而非檔案，不該觸發「你要處理哪個檔案」。
	hasFileIntent := fetchIntentRe.MatchString(stripped) ||
		strings.Contains(stripped, "開啟") || strings.Contains(stripped, "打開") ||
		strings.Contains(lowerStripped, "open") ||
		strings.Contains(stripped, "哪個") || strings.Contains(stripped, "哪一個") ||
		strings.Contains(stripped, "哪份") || strings.Contains(stripped, "什麼")
	return hasFileIntent
}

func openLocalPathInShell(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("路徑是空的")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		if info.IsDir() {
			return executil.Command("explorer.exe", path).Start()
		}
		return executil.Command("explorer.exe", "/select,"+path).Start()
	case "darwin":
		if info.IsDir() {
			return executil.Command("open", path).Start()
		}
		return executil.Command("open", "-R", path).Start()
	default:
		target := path
		if !info.IsDir() {
			target = filepath.Dir(path)
		}
		return executil.Command("xdg-open", target).Start()
	}
}

func (a *App) importAndReadLocalFile(path string, traceID string) *skill_step.CLIResponse {
	info, err := os.Stat(path)
	if err != nil {
		return &skill_step.CLIResponse{Text: fmt.Sprintf("無法讀取這個本機檔案：%v", err)}
	}
	if info.IsDir() {
		return &skill_step.CLIResponse{Text: "這是資料夾，不是檔案。你可以回覆「開啟」前往該位置，或補上要讀取的檔名。"}
	}
	result, err := a.HandleDocumentDrop(path)
	if err != nil {
		debugtrace.Record("resource_gate.local_read.import_error", traceID, map[string]interface{}{"path": path, "error": err.Error()})
		return &skill_step.CLIResponse{Text: fmt.Sprintf("無法讀取這個本機檔案：%v", err)}
	}
	content, err := a.ReadDocumentContent(result.DocID)
	if err != nil {
		return &skill_step.CLIResponse{Text: fmt.Sprintf("已匯入 %s，但讀取內容失敗：%v", result.DisplayName, err)}
	}
	content = strings.TrimSpace(content)
	if len([]rune(content)) > 4000 {
		runes := []rune(content)
		content = string(runes[:4000]) + "\n（內容過長已截斷）"
	}
	if content == "" {
		content = "（沒有可讀文字內容）"
	}
	return &skill_step.CLIResponse{
		Text:   fmt.Sprintf("【%s】\n%s", result.DisplayName, content),
		Action: "讀取",
		Target: path,
		Next:   "standby",
	}
}

// recordWebSearchResultURLs web search 結果 URL 進 registry（web_search_result 來源）。
func recordWebSearchResultURLs(urls []string, sessionID, traceID string) {
	reg := getURLRegistry()
	for _, u := range urls {
		if strings.TrimSpace(u) == "" {
			continue
		}
		_, _ = reg.Record(u, url_source.SourceWebSearchResult, sessionID, traceID, "")
	}
}
