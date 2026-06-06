package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
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
	req := websearch.SearchRequest{Query: strings.TrimSpace(query), Limit: limit}
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
	req, ok := websearch.ParseUserQuery(userText)
	if !ok {
		return nil, false
	}
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: "網路", Target: req.Query, Next: actionchain.StandbyNext}
	if handled, resp := a.maybeAskForToolReadiness(sessionID, decision, traceID); handled {
		return &resp, true
	}
	req.Query = a.targetWithBackground(sessionID, req.Query)
	resp := a.executeWebSearch(req, traceID)
	return &resp, true
}

func (a *App) executeWebSearch(req websearch.SearchRequest, traceID string) skill_step.CLIResponse {
	debugtrace.Record("web_search.enter", traceID, map[string]interface{}{
		"query": req.Query,
		"limit": req.Limit,
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
	return skill_step.CLIResponse{Text: websearch.FormatSearchOutcome(req, outcome)}
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
