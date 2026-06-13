// app_adapter.go - split out of app.go (same package, codemod).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ui_console/adapter/adapter_registry"
	"ui_console/adapter/debugtrace"
	"ui_console/adapter/remote_bridge"
	"ui_console/data/storage"
	"ui_console/internal/urlsafe"
	"ui_console/shared/eventbus"
	"ui_console/shared/taborder"
)

func listSubagentTabs(projectRoot string) []savedSubagent {
	callableDir := filepath.Join(projectRoot, "subagents", "callable")
	entries, err := os.ReadDir(callableDir)
	if err != nil {
		return nil
	}

	var saved []savedSubagent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		label := id
		var createdAt time.Time
		data, err := os.ReadFile(filepath.Join(callableDir, id, "sub_meta.json"))
		if err == nil {
			var meta subagentHaoraMeta
			if json.Unmarshal(data, &meta) == nil {
				if strings.TrimSpace(meta.Name) != "" {
					label = strings.TrimSpace(meta.Name)
				}
				if !meta.CreatedAt.IsZero() {
					createdAt = meta.CreatedAt
				}
			}
		}
		saved = append(saved, savedSubagent{id: id, label: label, createdAt: createdAt})
	}

	sort.SliceStable(saved, func(i, j int) bool {
		if !saved[i].createdAt.IsZero() && !saved[j].createdAt.IsZero() {
			return saved[i].createdAt.Before(saved[j].createdAt)
		}
		return saved[i].id < saved[j].id
	})

	return orderSavedSubagents(projectRoot, saved)
}

func orderSavedSubagents(projectRoot string, saved []savedSubagent) []savedSubagent {
	byID := make(map[string]savedSubagent, len(saved))
	for _, sub := range saved {
		byID[sub.id] = sub
	}

	seen := make(map[string]bool, len(saved))
	var ordered []savedSubagent
	orderMgr := taborder.NewManager(projectRoot)
	for _, id := range orderMgr.GetOrder().SubOrder {
		sub, ok := byID[id]
		if !ok || seen[id] {
			continue
		}
		ordered = append(ordered, sub)
		seen[id] = true
	}
	for _, sub := range saved {
		if !seen[sub.id] {
			ordered = append(ordered, sub)
			seen[sub.id] = true
		}
	}

	ids := make([]string, 0, len(ordered))
	for _, sub := range ordered {
		ids = append(ids, sub.id)
	}
	_ = orderMgr.Reorder(ids)
	return ordered
}

func resolveSubagentOrder(projectRoot string, requested []string) []string {
	callableDir := filepath.Join(projectRoot, "subagents", "callable")
	entries, err := os.ReadDir(callableDir)
	if err != nil {
		return requested
	}

	ids := make(map[string]bool)
	labels := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		ids[id] = true
		label := id
		data, err := os.ReadFile(filepath.Join(callableDir, id, "sub_meta.json"))
		if err == nil {
			var meta subagentHaoraMeta
			if json.Unmarshal(data, &meta) == nil && strings.TrimSpace(meta.Name) != "" {
				label = strings.TrimSpace(meta.Name)
			}
		}
		labels[label] = id
	}

	resolved := make([]string, 0, len(requested))
	seen := make(map[string]bool, len(requested))
	for _, item := range requested {
		id := item
		if !ids[id] {
			if mapped, ok := labels[item]; ok {
				id = mapped
			}
		}
		if ids[id] && !seen[id] {
			resolved = append(resolved, id)
			seen[id] = true
		}
	}
	return resolved
}

func (a *App) syncActionTagsToCLIAdapter(traceID string) []string {
	tags := a.collectActionTags()
	if setter, ok := a.cliAdapter.(cliActionTagSetter); ok {
		// Keep prompt candidates fresh; tools and builtin skills can change at runtime.
		setter.SetActionTags(tags)
	}
	debugtrace.Record("go.actionTags.sync", traceID, map[string]interface{}{
		"count": len(tags),
		"tags":  tags,
	})
	return tags
}

// GetNewSubagentCandidates returns new subagent candidates created by hook runs.
// Candidates are read-only proposals — they do NOT replace or disable existing subagents.
func (a *App) GetNewSubagentCandidates() (interface{}, error) {
	candidates, err := a.candidateService.ListCandidates()
	return frontendDTO(candidates), err
}

func localAdapterReconnectHint(adapter adapter_registry.Adapter, cause string) string {
	name := strings.TrimSpace(adapter.Name)
	if name == "" {
		name = "本機模型"
	}
	cause = strings.TrimSpace(cause)
	if cause != "" {
		cause = "\n原始錯誤：" + cause
	}
	return fmt.Sprintf("本機模型「%s」連線受阻。系統已嘗試自動喚醒一次；如果仍然失敗，請點一下或雙擊左側的本機模型卡片來喚醒連線，確認狀態燈變綠後再送出訊息。%s", name, cause)
}

func (a *App) loadLLMAPIAdapterConfig(adapterID string) (llmAPIAdapterConfig, error) {
	ref := "llm_provider:" + adapterID + ":config"
	if a.secretStore != nil {
		if raw, err := a.secretStore.Load(ref); err == nil && strings.TrimSpace(raw) != "" {
			var cfg llmAPIAdapterConfig
			if err := json.Unmarshal([]byte(raw), &cfg); err == nil {
				cfg.ProviderID = strings.TrimSpace(cfg.ProviderID)
				cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
				cfg.Model = strings.TrimSpace(cfg.Model)
				cfg.Name = strings.TrimSpace(cfg.Name)
				return fillLLMAPIConfigDefaults(adapterID, cfg), nil
			}
		}
	}
	// For local adapters, pull endpoint + model from the adapter registry as fallback.
	if a.adapterRegistry != nil {
		if adapterInfo, adapterErr := a.adapterRegistry.GetStatus(adapterID); adapterErr == nil && adapterInfo.Kind == "local" && adapterInfo.Endpoint != "" {
			providerID := "ollama"
			if strings.Contains(adapterInfo.Endpoint, ":1234") {
				providerID = "lmstudio"
			}
			cfg := fillLLMAPIConfigDefaults(adapterID, llmAPIAdapterConfig{
				ProviderID: providerID,
				Name:       adapterInfo.Name,
				BaseURL:    adapterInfo.Endpoint,
				Model:      adapterInfo.Model, // e.g. "qwen2.5:14b"
			})
			return cfg, nil
		}
	}
	cfg := fillLLMAPIConfigDefaults(adapterID, llmAPIAdapterConfig{})
	if cfg.ProviderID == "" || cfg.BaseURL == "" {
		return cfg, fmt.Errorf("API adapter config not found for %s", adapterID)
	}
	return cfg, nil
}

// RegisterLLMAPIAdapter 第一版只建立 sidebar API adapter 入口；API key/baseURL/model
// 的實際呼叫流程會接在下一階段，名稱修改沿用現有 rename 彈窗。
func (a *App) RegisterLLMAPIAdapter(providerID, providerName, baseURL, model, apiKey string) (interface{}, error) {
	providerID = strings.TrimSpace(providerID)
	providerName = strings.TrimSpace(providerName)
	if providerID == "" {
		providerID = "generic-api"
	}
	if providerName == "" {
		providerName = "LLM API"
	}
	// SEC-03: 驗證 baseURL 防止 SSRF
	needConfirm, host, urlErr := urlsafe.ValidateLLMBaseURL(providerID, baseURL)
	if urlErr != nil {
		return nil, fmt.Errorf("baseURL 驗證失敗 (%s): %w", baseURL, urlErr)
	}
	if needConfirm {
		// 私有 IP 需前端確認，回傳特殊 DTO 讓前端彈確認框
		return frontendDTO(map[string]interface{}{
			"need_confirm":  true,
			"confirm_type":  "private_network",
			"hostname":      host,
			"original_url":  strings.TrimSpace(baseURL),
			"provider_id":   providerID,
			"provider_name": providerName,
			"model":         strings.TrimSpace(model),
		}), nil
	}

	return a.registerLLMAPIAdapterInternal(providerID, providerName, baseURL, model, apiKey)
}

// ConfirmRegisterLLMAPIAdapter SEC-03: 使用者已確認私有網路連線後呼叫。
func (a *App) ConfirmRegisterLLMAPIAdapter(providerID, providerName, baseURL, model, apiKey string) (interface{}, error) {
	providerID = strings.TrimSpace(providerID)
	providerName = strings.TrimSpace(providerName)
	if providerID == "" {
		providerID = "generic-api"
	}
	if providerName == "" {
		providerName = "LLM API"
	}
	log.Printf("ConfirmRegisterLLMAPIAdapter: user confirmed private network %s", baseURL)
	return a.registerLLMAPIAdapterInternal(providerID, providerName, baseURL, model, apiKey)
}

// registerLLMAPIAdapterInternal 實際建立 adapter 的內部函式。
func (a *App) registerLLMAPIAdapterInternal(providerID, providerName, baseURL, model, apiKey string) (interface{}, error) {
	adapterID := fmt.Sprintf("llm-api-%s-%d", providerID, time.Now().UnixMilli())
	secretRef := "llm_provider:" + adapterID + ":api_key"
	if strings.TrimSpace(apiKey) != "" {
		if err := a.secretStore.Store(secretRef, strings.TrimSpace(apiKey)); err != nil {
			return nil, err
		}
	}
	configRef := "llm_provider:" + adapterID + ":config"
	config := llmAPIAdapterConfig{
		ProviderID: providerID,
		Name:       providerName,
		BaseURL:    strings.TrimSpace(baseURL),
		Model:      strings.TrimSpace(model),
	}
	if configRaw, err := json.Marshal(config); err == nil {
		if err := a.secretStore.Store(configRef, string(configRaw)); err != nil {
			return nil, err
		}
	}
	icon := "A"
	if runes := []rune(providerName); len(runes) > 0 {
		icon = strings.ToUpper(string(runes[0]))
	}
	if err := a.adapterRegistry.RegisterAPI(adapterID, providerName, icon); err != nil {
		return nil, err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
			"kind":       "api",
		})
	}
	return frontendDTO(map[string]string{
		"adapter_id":  adapterID,
		"name":        providerName,
		"kind":        "api",
		"base_url":    strings.TrimSpace(baseURL),
		"model":       strings.TrimSpace(model),
		"api_key_ref": secretRef,
		"config_ref":  configRef,
	}), nil
}

// RenameAdapter 更新左側 Adapter 顯示名稱。API adapter 註冊後也走這個命名流程。
func (a *App) RenameAdapter(adapterID, displayName string) (interface{}, error) {
	adapter, err := a.adapterRegistry.Rename(adapterID, displayName)
	if err == nil && a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
		})
	}
	return frontendDTO(adapter), err
}

// UnregisterAdapter 移除一個已註冊的 adapter。
func (a *App) UnregisterAdapter(adapterID string) error {
	return a.adapterRegistry.Unregister(adapterID)
}

// WakeLocalAdapter is a user-triggered local-model wake-up action. It only
// starts known local runtimes from fixed discovery paths; arbitrary adapter
// paths are never executed.
func (a *App) WakeLocalAdapter(adapterID string) (interface{}, error) {
	if a.adapterRegistry == nil {
		return nil, fmt.Errorf("adapter registry is not available")
	}
	adapter, err := a.adapterRegistry.GetStatus(adapterID)
	if err != nil {
		return nil, err
	}
	if adapter.Kind != "local" {
		return nil, fmt.Errorf("adapter %s is not a local model", adapterID)
	}
	if strings.Contains(adapter.Endpoint, ":11434") || strings.Contains(strings.ToLower(adapter.ID+" "+adapter.Name), "ollama") {
		result, err := a.wakeOllamaAdapter(adapter)
		return frontendDTO(result), err
	}
	if strings.Contains(adapter.Endpoint, ":1234") {
		if pingOpenAIModelsEndpoint(adapter.Endpoint, 800*time.Millisecond) {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusOnline)
			return frontendDTO(map[string]string{"status": "online", "message": "LM Studio 已在線"}), nil
		}
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return nil, fmt.Errorf("LM Studio 尚未啟動；請先在 LM Studio 啟動 local server")
	}
	return nil, fmt.Errorf("unknown local adapter endpoint: %s", adapter.Endpoint)
}

// wakeOllamaAdapter — registry path：包 wakeOllamaDaemon 並在前後更新 adapter 狀態。
func (a *App) wakeOllamaAdapter(adapter adapter_registry.Adapter) (map[string]string, error) {
	baseURL := ollamaBaseURL(adapter.Endpoint)
	modelDir := resolveOllamaModelDir(adapter.Path)
	// 先 ping 一次拿快速 path 的訊息差異（已上線 vs 剛喚醒）。
	online := pingOllamaTags(baseURL, 800*time.Millisecond)
	if online {
		a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusOnline)
		return map[string]string{"status": "online", "message": "Ollama 已在線"}, nil
	}
	if err := wakeOllamaDaemon(baseURL, modelDir); err != nil {
		a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusDegraded)
		return nil, err
	}
	a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusOnline)
	return map[string]string{"status": "online", "message": "Ollama 已喚醒"}, nil
}

// sanitizeAdapterID replaces characters that are invalid in adapter IDs.
func sanitizeAdapterID(s string) string {
	r := strings.NewReplacer(":", "-", "/", "-", " ", "-")
	return strings.ToLower(r.Replace(s))
}

// ReorderAdapters updates the persisted sidebar order for CLI/API adapters.
func (a *App) ReorderAdapters(orderJSON string) error {
	var orderIDs []string
	if err := json.Unmarshal([]byte(orderJSON), &orderIDs); err != nil {
		return err
	}
	if err := a.adapterRegistry.Reorder(orderIDs); err != nil {
		return err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"reason": "reordered",
		})
	}
	return nil
}

// ListAvailableAdapters returns CLI adapters plus callable sub tabs for the sidebar.
func (a *App) ListAvailableAdapters() interface{} {
	items := make([]map[string]interface{}, 0)
	for _, adapter := range a.adapterRegistry.ListAvailable() {
		if !shouldExposeAdapter(adapter) {
			continue
		}
		kind := strings.TrimSpace(adapter.Kind)
		if kind == "" {
			kind = "cli"
		}
		items = append(items, map[string]interface{}{
			"id":         adapter.ID,
			"name":       adapter.Name,
			"icon":       adapter.Icon,
			"path":       adapter.Path,
			"endpoint":   adapter.Endpoint,
			"model":      adapter.Model,
			"status":     adapter.Status,
			"last_check": adapter.LastCheck,
			"kind":       kind,
		})
	}

	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	for _, sub := range listSubagentTabs(projectRoot) {
		items = append(items, map[string]interface{}{
			"id":     sub.id,
			"name":   sub.label,
			"icon":   "⊕",
			"status": "offline",
			"kind":   "sub",
		})
	}
	return frontendDTO(items)
}

func shouldExposeAdapter(adapter adapter_registry.Adapter) bool {
	kind := strings.TrimSpace(adapter.Kind)
	if kind != "local" {
		return true
	}
	identity := strings.ToLower(adapter.ID + " " + adapter.Name + " " + adapter.Endpoint)
	if strings.Contains(identity, "ollama") && !isOllamaGenerativeModelID(adapter.Model) {
		return false
	}
	return true
}

func (a *App) refreshAdapterRuntimeHealth() {
	if a.adapterRegistry == nil {
		return
	}
	a.adapterHealthMu.Lock()
	defer a.adapterHealthMu.Unlock()
	for _, adapter := range a.adapterRegistry.ListAvailable() {
		kind := strings.TrimSpace(adapter.Kind)
		if kind != "" && kind != "cli" {
			continue
		}
		a.checkAdapterRuntimeHealth(adapter)
	}
}

func (a *App) checkAdapterRuntimeHealth(adapter adapter_registry.Adapter) {
	// 背景健康檢查只碰 CLI metadata，不呼叫模型，避免開機耗 token 或卡 UI。
	cliPath, err := a.adapterRegistry.ResolveExecutable(adapter.ID)
	if err != nil {
		a.setAdapterRuntimeStatusWithMessage(adapter.ID, adapter_registry.StatusOffline, err.Error())
		return
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := runCLIHealthProbe(checkCtx, cliPath); err != nil {
		status := adapter_registry.StatusDegraded
		message := err.Error()
		if errors.Is(checkCtx.Err(), context.DeadlineExceeded) {
			message = "CLI health check timed out after 3s"
		}
		a.setAdapterRuntimeStatusWithMessage(adapter.ID, status, message)
		return
	}
	a.setAdapterRuntimeStatusWithMessage(adapter.ID, adapter_registry.StatusOnline, "")
}

func (a *App) setAdapterRuntimeStatusWithMessage(adapterID string, status adapter_registry.Status, message string) {
	if adapterID == "" || a.adapterRegistry == nil {
		return
	}
	if err := a.adapterRegistry.SetStatus(adapterID, status); err != nil {
		return
	}
	if a.eventBus != nil {
		payload := map[string]string{
			"adapter_id": adapterID,
			"status":     string(status),
		}
		if strings.TrimSpace(message) != "" {
			payload["message"] = strings.TrimSpace(message)
		}
		a.eventBus.Emit(eventbus.EventAdapterStatusChanged, payload)
	}
	if strings.TrimSpace(message) != "" {
		debugtrace.Record("go.adapter.health", "", map[string]interface{}{
			"adapter_id": adapterID,
			"status":     string(status),
			"message":    message,
		})
	}
}

// GetAdapterStatus returns the status of a specific adapter by ID.
func (a *App) GetAdapterStatus(adapterID string) (interface{}, error) {
	status, err := a.adapterRegistry.GetStatus(adapterID)
	return frontendDTO(status), err
}

// SetAdapterStatus updates the connectivity status of an adapter.
func (a *App) SetAdapterStatus(adapterID string, status string) error {
	err := a.adapterRegistry.SetStatus(adapterID, adapter_registry.Status(status))
	if err == nil {
		a.eventBus.Emit(eventbus.EventAdapterStatusChanged, map[string]string{
			"adapter_id": adapterID, "status": status,
		})
	}
	return err
}

func (a *App) setAdapterRuntimeStatus(adapterID string, status adapter_registry.Status) {
	if adapterID == "" || a.adapterRegistry == nil {
		return
	}
	if err := a.adapterRegistry.SetStatus(adapterID, status); err != nil {
		return
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterStatusChanged, map[string]string{
			"adapter_id": adapterID,
			"status":     string(status),
		})
	}
}

// ListRemoteBridgeInboundAdapters 讓前端/測試知道哪些平台已有半雙向接口。
func (a *App) ListRemoteBridgeInboundAdapters() interface{} {
	return frontendDTO(remote_bridge.ListInboundAdapterStatuses())
}
