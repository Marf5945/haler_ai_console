package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/data/memory"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/eventbus"
	"ui_console/shared/websearch"
)

const (
	webSearchProviderRef = "web_search:provider"
	webSearchAPIKeyRef   = "web_search:%s:api_key"
	webSearchCXRef       = "web_search:%s:cx"
)

// GetWebSearchConfig returns only non-secret search configuration state.
func (a *App) GetWebSearchConfig() interface{} {
	return frontendDTO(a.webSearchConfigPublic())
}

// SaveWebSearchConfig stores selected provider credentials in the OS-backed
// encrypted credential store. Windows uses DPAPI for the master key; macOS uses
// Keychain. The API key value is never returned by GetWebSearchConfig.
func (a *App) SaveWebSearchConfig(providerID, apiKey, cx string) (interface{}, error) {
	providerID = websearch.NormalizeProviderID(providerID)
	if len(websearch.RequiredFields(providerID)) == 0 {
		return nil, fmt.Errorf("unknown web search provider: %s", providerID)
	}
	if a.secretStore == nil {
		return nil, fmt.Errorf("credential store is not available")
	}
	apiKey = strings.TrimSpace(apiKey)
	cx = strings.TrimSpace(cx)
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if providerID == websearch.ProviderGoogleCSE && cx == "" {
		return nil, fmt.Errorf("Google Custom Search Engine ID (cx) is required")
	}
	if err := a.secretStore.Store(webSearchProviderRef, providerID); err != nil {
		return nil, err
	}
	if err := a.secretStore.Store(fmt.Sprintf(webSearchAPIKeyRef, providerID), apiKey); err != nil {
		return nil, err
	}
	if providerID == websearch.ProviderGoogleCSE {
		if err := a.secretStore.Store(fmt.Sprintf(webSearchCXRef, providerID), cx); err != nil {
			return nil, err
		}
	}
	return frontendDTO(a.webSearchConfigPublic()), nil
}

// ClearWebSearchConfig removes all web-search provider credentials.
func (a *App) ClearWebSearchConfig() (interface{}, error) {
	if a.secretStore == nil {
		return nil, fmt.Errorf("credential store is not available")
	}
	_ = a.secretStore.Delete(webSearchProviderRef)
	for _, option := range websearch.ProviderOptions() {
		_ = a.secretStore.Delete(fmt.Sprintf(webSearchAPIKeyRef, option.ID))
		_ = a.secretStore.Delete(fmt.Sprintf(webSearchCXRef, option.ID))
	}
	return frontendDTO(a.webSearchConfigPublic()), nil
}

// SearchWeb is the Wails-facing network search binding.
func (a *App) SearchWeb(query string, limit int) (interface{}, error) {
	// SEC-15: 直接呼叫入口同樣過出境檢查；命中回 need_confirm，
	// 前端確認後改呼叫 SearchWebConfirmed 送遮蔽版。
	masked, records := memory.RedactBeforeWrite(strings.TrimSpace(query))
	if len(records) > 0 {
		return map[string]interface{}{
			"status":       "need_confirm",
			"masked_query": masked,
			"hits":         describeEgressHits(records),
		}, nil
	}
	return a.searchWebDirect(strings.TrimSpace(query), limit)
}

// SearchWebConfirmed 前端確認後用遮蔽版查詢續行（仍再過一次檢查，防呆）。
func (a *App) SearchWebConfirmed(maskedQuery string, limit int) (interface{}, error) {
	cleaned, _ := memory.RedactBeforeWrite(strings.TrimSpace(maskedQuery))
	return a.searchWebDirect(cleaned, limit)
}

func (a *App) searchWebDirect(query string, limit int) (interface{}, error) {
	req := websearch.SearchRequest{Query: query, Limit: limit}
	req = a.applyWebSearchAllowlist(req)
	cfg, err := a.loadWebSearchProviderConfig()
	if err != nil {
		a.emitWebSearchConfigRequired()
		return nil, err
	}
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, 12*time.Second)
	defer cancel()
	outcome, err := websearch.NewService().Search(ctx, req, cfg)
	if errors.Is(err, websearch.ErrNoResults) {
		return frontendDTO(outcome), nil
	}
	return frontendDTO(outcome), err
}

func (a *App) maybeHandleWebSearch(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	// SEC-06: URL 讀取（含 pending 確認）優先於搜尋——單一掛載點覆蓋所有路由。
	if resp, handled := a.maybeHandleURLFetch(userText, sessionID, traceID); handled {
		return resp, true
	}
	// SEC-15: 出境閘門的 pending 確認（「好」→ 送遮蔽版、「取消」→ 放棄）。
	if resp, handled := a.maybeResumePendingSearchEgress(userText, sessionID, traceID); handled {
		return resp, true
	}
	req, ok := websearch.ParseUserQuery(userText)
	if !ok {
		return nil, false
	}
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "網路", Target: req.Query, Next: actionchain.StandbyNext}
	if handled, resp := a.maybeAskForToolReadiness(sessionID, decision, userText, traceID); handled {
		return &resp, true
	}
	req.Query = a.targetWithBackground(sessionID, req.Query)
	// SEC-15: 查詢（含背景）出境前過機密檢查，命中先問。
	if resp, gated := a.gateSearchEgress(req, sessionID, traceID); gated {
		return resp, true
	}
	resp := a.executeWebSearch(req, traceID)
	return &resp, true
}

func (a *App) executeWebSearch(req websearch.SearchRequest, traceID string) skill_step.CLIResponse {
	req = a.applyWebSearchAllowlist(req)
	a.pushActionStatus("網路", req.Query) // status rail：正在用網路搜尋「…」…
	debugtrace.Record("web_search.enter", traceID, map[string]interface{}{
		"query":                 req.Query,
		"limit":                 req.Limit,
		"include_domains_count": len(req.IncludeDomains),
	})
	cfg, cfgErr := a.loadWebSearchProviderConfig()
	if cfgErr != nil {
		a.emitWebSearchConfigRequired()
		debugtrace.Record("web_search.config_missing", traceID, map[string]interface{}{
			"error": cfgErr.Error(),
		})
		return skill_step.CLIResponse{
			Text:   websearch.MissingConfigMessage(),
			Action: "web_search",
			Target: req.Query,
			Next:   "standby",
		}
	}
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, 12*time.Second)
	defer cancel()

	outcome, err := websearch.NewService().Search(ctx, req, cfg)
	if err != nil {
		if errors.Is(err, websearch.ErrEmptyQuery) {
			return skill_step.CLIResponse{Text: websearch.EmptyQueryMessage()}
		}
		if errors.Is(err, websearch.ErrNoResults) {
			return skill_step.CLIResponse{Text: websearch.FormatSearchOutcome(req, outcome)}
		}
		debugtrace.Record("web_search.error", traceID, map[string]interface{}{
			"error": err.Error(),
		})
		return skill_step.CLIResponse{Error: err.Error()}
	}
	debugtrace.Record("web_search.results", traceID, map[string]interface{}{
		"count":    len(outcome.Results),
		"provider": outcome.ProviderID,
	})
	// SEC-06: 搜尋結果 URL 進 provenance registry（web_search_result 來源）。
	urls := make([]string, 0, len(outcome.Results))
	for _, r := range outcome.Results {
		urls = append(urls, r.URL)
	}
	recordWebSearchResultURLs(urls, "", traceID)
	// 搜尋＋模型摘要：先嘗試用目前對話模型整理成帶來源標號的摘要；
	// 失敗、未設定可用模型或回應為空時，回退成原始結果清單（不讓使用者看到錯誤）。
	if summary, ok := a.summarizeWebSearchOutcome(req, outcome, traceID); ok {
		return skill_step.CLIResponse{Text: summary}
	}
	return skill_step.CLIResponse{Text: websearch.FormatSearchOutcome(req, outcome)}
}

func (a *App) applyWebSearchAllowlist(req websearch.SearchRequest) websearch.SearchRequest {
	domains := a.webSearchAllowlistDomains()
	if len(domains) == 0 {
		return req
	}
	seen := map[string]bool{}
	merged := make([]string, 0, len(req.IncludeDomains)+len(domains))
	for _, domain := range append(req.IncludeDomains, domains...) {
		domain = strings.TrimSpace(strings.ToLower(domain))
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		merged = append(merged, domain)
	}
	sort.Strings(merged)
	req.IncludeDomains = merged
	return req
}

func (a *App) webSearchAllowlistDomains() []string {
	if a == nil || a.allowlistStore == nil {
		return nil
	}
	entries, err := a.allowlistStore.ListActive()
	if err != nil || len(entries) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var domains []string
	for _, entry := range entries {
		host := strings.TrimSpace(strings.ToLower(entry.CanonicalHostname))
		if host == "" || seen[host] || !webSearchAllowlistEntryApplies(entry.AllowedFor, entry.NotAllowedFor) {
			continue
		}
		seen[host] = true
		domains = append(domains, host)
	}
	sort.Strings(domains)
	return domains
}

func webSearchAllowlistEntryApplies(allowedFor, notAllowedFor []string) bool {
	for _, item := range notAllowedFor {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "web_search", "search_web", "web", "internet":
			return false
		}
	}
	if len(allowedFor) == 0 {
		return true
	}
	for _, item := range allowedFor {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "web_search", "search_web", "web", "internet", "lookup", "research", "research_reference", "rag_ranking":
			return true
		}
	}
	return false
}

func (a *App) webSearchConfigPublic() websearch.ConfigPublic {
	state := websearch.ConfigPublic{
		Options:     websearch.ProviderOptions(),
		StorageMode: a.credentialStorageMode(),
	}
	cfg, err := a.loadWebSearchProviderConfig()
	if err != nil {
		state.Missing = missingWebSearchFields(a.currentWebSearchProviderID())
		return state
	}
	state.Configured = true
	state.ProviderID = cfg.ProviderID
	state.Provider = websearch.ProviderName(cfg.ProviderID)
	return state
}

func (a *App) loadWebSearchProviderConfig() (websearch.ProviderConfig, error) {
	if a.secretStore == nil {
		return websearch.ProviderConfig{}, fmt.Errorf("credential store is not available")
	}
	providerID, err := a.secretStore.Load(webSearchProviderRef)
	if err != nil || strings.TrimSpace(providerID) == "" {
		return websearch.ProviderConfig{}, websearch.ErrProviderMissing
	}
	providerID = websearch.NormalizeProviderID(providerID)
	cfg := websearch.ProviderConfig{ProviderID: providerID}
	apiKey, err := a.secretStore.Load(fmt.Sprintf(webSearchAPIKeyRef, providerID))
	if err != nil || strings.TrimSpace(apiKey) == "" {
		return websearch.ProviderConfig{}, websearch.ErrCredentialMissing
	}
	cfg.APIKey = apiKey
	if providerID == websearch.ProviderGoogleCSE {
		cx, err := a.secretStore.Load(fmt.Sprintf(webSearchCXRef, providerID))
		if err != nil || strings.TrimSpace(cx) == "" {
			return websearch.ProviderConfig{}, websearch.ErrCredentialMissing
		}
		cfg.CX = cx
	}
	return cfg, nil
}

func (a *App) credentialStorageMode() string {
	if a == nil || a.credentialStore == nil {
		return "credential_store_unavailable"
	}
	providerID := strings.TrimSpace(a.credentialStore.ProviderID())
	if providerID == "" {
		return "credential_store_unavailable"
	}
	return providerID + "_credential_store"
}

func (a *App) currentWebSearchProviderID() string {
	if a == nil || a.secretStore == nil {
		return ""
	}
	providerID, err := a.secretStore.Load(webSearchProviderRef)
	if err != nil {
		return ""
	}
	return websearch.NormalizeProviderID(providerID)
}

func missingWebSearchFields(providerID string) []string {
	fields := websearch.RequiredFields(providerID)
	if len(fields) == 0 {
		return []string{"provider", "api_key"}
	}
	return fields
}

func (a *App) emitWebSearchConfigRequired() {
	if a == nil || a.eventBus == nil {
		return
	}
	a.eventBus.Emit(eventbus.EventWebSearchConfigRequired, a.webSearchConfigPublic())
}
