// app_model.go - split out of app.go (same package, codemod).
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ui_console/adapter/adapter_registry"
	"ui_console/adapter/w3a_media"
	"ui_console/data/conversation"
	"ui_console/data/memory"
	"ui_console/internal/urlsafe"
	"ui_console/internal/voice"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/eventbus"
	"ui_console/shared/executil"
)

func (a *App) InstallVoiceBaseModel() (voice.State, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// Model download is user-triggered and can take a while on slower networks.
	downloadCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	return a.voiceService.InstallBaseModel(downloadCtx, a.currentPanelLanguage())
}

func (a *App) RemoveVoiceBaseModel() (voice.State, error) {
	return a.voiceService.RemoveManagedModel(a.currentPanelLanguage())
}

func (a *App) PrepareCloudTTSText(text string) voice.CloudTTSEgressPreview {
	return prepareCloudTTSEgressPreview(text)
}

func prepareCloudTTSEgressPreview(text string) voice.CloudTTSEgressPreview {
	masked, records := memory.RedactBeforeWrite(text)
	types := map[string]bool{}
	for _, r := range records {
		if r.Type != "" {
			types[r.Type] = true
		}
	}
	hitTypes := make([]string, 0, len(types))
	for typ := range types {
		hitTypes = append(hitTypes, typ)
	}
	sort.Strings(hitTypes)
	return voice.CloudTTSEgressPreview{
		Allowed:              len(records) == 0,
		RequiresConfirmation: len(records) > 0,
		MaskedText:           masked,
		HitCount:             len(records),
		HitTypes:             hitTypes,
	}
}

// isQuotaExhaustedError 判斷字串是否為配額／限流類錯誤。
func isQuotaExhaustedError(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	return containsAny(strings.ToLower(s), quotaMarkers)
}

// quotaSwitchModelNotice 偵測回應或錯誤是否為配額／限流；是的話回傳一段提示
// 使用者切換模型的訊息，讓 UI 立即顯示可行動指引，而不是丟原始錯誤或一直重試。
func quotaSwitchModelNotice(adapterID string, resp *skill_step.CLIResponse, err error) (string, bool) {
	var parts []string
	if err != nil {
		parts = append(parts, err.Error())
	}
	if resp != nil {
		parts = append(parts, resp.Error, resp.Text)
	}
	if !isQuotaExhaustedError(strings.Join(parts, "\n")) {
		return "", false
	}
	name := strings.TrimSpace(adapterID)
	if name == "" {
		name = "目前的模型"
	}
	return "⚠️ " + name + " 配額已用盡或被限流，暫時無法回應。請在上方模型選單切換到其他模型後重試。", true
}

// buildLocalModelPrompt produces a dead-simple prompt for small local models.
// Instead of the ㄌ protocol, it asks for three short lines.
func buildLocalModelPrompt(systemPrompt string, actionTags []string, userText string) string {
	tagList := strings.Join(conversation.PromptActionTags(actionTags), "、")
	return fmt.Sprintf(
		"%s\n回答規則：用三行回答，每行一個欄位，但不要寫欄位名稱（不要寫 動作:、內容:、下一步:）。\n第一行寫動作（從候選中選：%s）\n動作定義：已知答案、一般聊天、寒暄、情緒回應用 輸出；需要系統查資料用 搜尋；需要製作獨立程式用 程式；要用既有 skill 處理資料用 流程；操作=只代表執行或重現已保存的螢幕 replay 操作，不是一般的處理/回答；讀取=取得內容並回報；開啟=用外部應用程式呈現；寫入=新增或修改檔案；匯出=產生檔案；提問=只有缺少必要資訊時才補問；選項=只有使用者明確要求選擇時才顯示選項卡。\n第二行直接寫要顯示給使用者的內容，不要加 內容:；若第一行是選項且沒有問題文字，必須用 ㄤ 開頭，例如 ㄤ紅色ㄤ綠色ㄤ藍色；若有問題文字，寫 問題ㄤ選項一ㄤ選項二。\n第三行只寫 待命、輸出、選項 其中之一；程式、流程通常寫 輸出，其他通常寫 待命。\n沒有明確重現、回放、照做、執行已保存操作的意思時，不要選 操作。\n範例：\n輸出\n你好啊\n待命\n\nQ: %s\n",
		systemPrompt, tagList, userText,
	)
}

func stripLocalModelFieldLabel(line string) string {
	text := strings.TrimSpace(line)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	for _, label := range []string{
		"\u52d5\u4f5c",       // 動作
		"\u5167\u5bb9",       // 內容
		"\u4e0b\u4e00\u6b65", // 下一步
		"action",
		"content",
		"next",
	} {
		if strings.HasPrefix(lower, label) {
			rest := strings.TrimLeft(strings.TrimSpace(text[len(label):]), ":：")
			rest = strings.TrimSpace(rest)
			if rest != "" {
				return rest
			}
		}
	}
	return text
}

func isOllamaPromptCLI(adapterID, cliPath string) bool {
	id := strings.ToLower(strings.TrimSpace(adapterID))
	if id == "ollama-cli" {
		return true
	}
	return adapter_registry.IsOllamaExecutablePath(cliPath)
}

// ScanLocalModels detects locally running Ollama / LM Studio models.
// Returns detection results without auto-registering — the user picks which to enable.
func (a *App) ScanLocalModels() interface{} {
	var results []LocalModelDetectResult

	// Scan Ollama
	ollamaModels := scanOllamaModels()
	for _, m := range ollamaModels {
		results = append(results, LocalModelDetectResult{
			AdapterID: "local-ollama-" + sanitizeAdapterID(m.ID),
			Name:      "Ollama - " + m.ID,
			ModelID:   m.ID,
			Provider:  "ollama",
			Endpoint:  "http://localhost:11434/v1",
			Found:     true,
		})
	}

	// Scan LM Studio
	lmsModels := scanLMStudioModels()
	for _, m := range lmsModels {
		results = append(results, LocalModelDetectResult{
			AdapterID: "local-lmstudio-" + sanitizeAdapterID(m.ID),
			Name:      "LM Studio - " + m.ID,
			ModelID:   m.ID,
			Provider:  "lmstudio",
			Endpoint:  "http://localhost:1234/v1",
			Found:     true,
		})
	}

	return frontendDTO(results)
}

// EnableLocalModel registers a detected local model into the adapter list.
func (a *App) EnableLocalModel(adapterID, name, modelID, provider, endpoint string) error {
	if strings.EqualFold(strings.TrimSpace(provider), "ollama") && !isOllamaGenerativeModelID(modelID) {
		return fmt.Errorf("ollama: model %q is not a chat/generative model", modelID)
	}
	icon := "◉"
	if provider == "lmstudio" {
		icon = "◈"
	}
	if err := a.adapterRegistry.RegisterLocal(adapterID, name, icon, endpoint, modelID); err != nil {
		return err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
			"kind":       "local",
		})
	}
	return nil
}

// wakeOllamaDaemon 是「拉起本機 ollama serve」的純邏輯，不做 adapter status 更新。
// caller：
//   - wakeOllamaAdapter（registry path）：包這層、額外更新 adapter status
//   - WakeOllamaDaemon Wails binding（modal path）：直接呼叫、無 status 概念
//
// 參數：
//   - baseURL：要 ping 的 endpoint，例 "http://localhost:11434"；空字串 → 預設
//   - modelDirHint：要塞 OLLAMA_MODELS 的目錄；空字串 → 不設 env
//
// 行為：
//   - 第一輪 ping 過 → 立刻 nil
//   - 沒過 → 找 binary、spawn `ollama serve`、再 ping 30×200ms = 6 秒
//   - 仍 ping 不到 → error
func wakeOllamaDaemon(baseURL, modelDirHint string) error {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if pingOllamaTags(baseURL, 800*time.Millisecond) {
		return nil
	}
	ollamaPath := resolveOllamaExecutable()
	if ollamaPath == "" {
		return fmt.Errorf("找不到 Ollama CLI，請安裝 Ollama 或加入 /opt/homebrew/bin/ollama")
	}
	cmd := executil.Command(ollamaPath, "serve")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Env = os.Environ()
	if modelDirHint != "" {
		cmd.Env = append(cmd.Env, "OLLAMA_MODELS="+modelDirHint)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // detach
	for i := 0; i < 30; i++ {
		if pingOllamaTags(baseURL, 300*time.Millisecond) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("Ollama 已嘗試啟動，但 API 尚未回應")
}

func resolveOllamaModelDir(adapterPath string) string {
	for _, candidate := range []string{
		adapterPath,
		os.Getenv("OLLAMA_MODELS"),
		userOllamaModelDir(),
		defaultOllamaModelDir(),
	} {
		candidate = expandUserPath(candidate)
		if isOllamaModelLibrary(candidate) {
			return candidate
		}
	}
	return ""
}

func userOllamaModelDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "ollama")
}

func defaultOllamaModelDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".ollama", "models")
}

func ollamaBaseURL(endpoint string) string {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(base, "/v1") {
		base = strings.TrimSuffix(base, "/v1")
	}
	if base == "" {
		base = "http://localhost:11434"
	}
	return base
}

func pingOllamaTags(baseURL string, timeout time.Duration) bool {
	// SEC-05 2a: 本機 model 偵測走 PolicyLocalLLM（允許 loopback、擋 LAN/metadata）。
	client := urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "model_ping", timeout)
	resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func pingOpenAIModelsEndpoint(endpoint string, timeout time.Duration) bool {
	// SEC-05 2a: 本機 model 偵測走 PolicyLocalLLM（允許 loopback、擋 LAN/metadata）。
	client := urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "model_ping", timeout)
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(base, "/v1") {
		base += "/models"
	} else {
		base += "/v1/models"
	}
	resp, err := client.Get(base)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// DetectModelPollution 對媒體執行模型污染偵測。
func (a *App) DetectModelPollution(path string) (*w3a_media.PollutionReport, error) {
	report, err := a.w3aMedia.DetectMediaPollution(path)
	if err == nil && report.IsPollutionRisk {
		a.eventBus.Emit("w3a:pollution_detected", map[string]interface{}{"path": path, "score": report.WeightedTotal})
	}
	return report, err
}
