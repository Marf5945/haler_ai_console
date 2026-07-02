// app_model.go - split out of app.go (same package, codemod).
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/adapter/adapter_registry"
	"ui_console/adapter/wa3_media"
	"ui_console/data/conversation"
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
	return quotaSwitchModelMessage(adapterID), true
}

// quotaSwitchModelMessage 產生「請切換模型」提示文字（不含偵測，純組字串），
// 供 quotaSwitchModelNotice 與 routing 層 quota fast-fail 共用。
func quotaSwitchModelMessage(adapterID string) string {
	name := strings.TrimSpace(adapterID)
	if name == "" {
		name = "目前的模型"
	}
	return "⚠️ " + name + " 配額已用盡或被限流，暫時無法回應。請在上方模型選單切換到其他模型後重試。"
}

// buildLocalModelPrompt produces a guided prompt for small local models.
// Unlike the ㄌ protocol on the API path, it asks for three short lines and
// supplies per-word action meanings plus a worked example for each action.
func buildLocalModelPrompt(systemPrompt string, actionTags []string, userText string) string {
	tagList := strings.Join(conversation.PromptActionTags(actionTags), "\u3001")
	return fmt.Sprintf(
		"%s\n"+
			"\u56de\u7b54\u898f\u5247\uff1a\u53ea\u7528\u4e09\u884c\u56de\u7b54\uff0c\u6bcf\u884c\u4e00\u500b\u6b04\u4f4d\uff0c\u4e0d\u8981\u5beb\u6b04\u4f4d\u540d\u7a31\uff08\u4e0d\u8981\u5beb \u52d5\u4f5c:\u3001\u5167\u5bb9:\u3001\u4e0b\u4e00\u6b65:\uff09\uff0c\u4e0d\u8981\u52a0\u5f15\u865f\u3001JSON \u6216\u689d\u5217\u3002\n"+
			"\u7b2c\u4e00\u884c=\u52d5\u4f5c\uff0c\u5f9e\u5019\u9078\u4e2d\u6311\u4e00\u500b\uff0c\u5fc5\u9808\u539f\u6587\u7167\u6284\u5019\u9078\u8a5e\uff0c\u4e0d\u53ef\u7ffb\u8b6f\u52d5\u4f5c\u8a5e\uff0c\u4e0d\u53ef\u6539\u6210\u7c21\u9ad4\u5b57\uff1b\u7279\u5225\u662f\u5fc5\u9808\u5beb \u8f38\u51fa\uff0c\u4e0d\u8981\u5beb \u8f93\u51fa\uff1a%s\n"+
			"\u7b2c\u4e8c\u884c=\u8981\u986f\u793a\u7d66\u4f7f\u7528\u8005\u7684\u5167\u5bb9\u6216\u52d5\u4f5c\u76ee\u6a19\uff0c\u4fdd\u7559\u6307\u5b9a\u8a9e\u8a00\u6216\u4f7f\u7528\u8005\u8a9e\u8a00\uff0c\u4e0d\u8981\u7ffb\u6210\u4e2d\u6587\uff0c\u4e0d\u8981\u52a0 \u5167\u5bb9:\u3002\n"+
			"English role: the second line must be English. Korean role: the second line must be Korean Hangul, not English, Japanese, or Chinese, except short technical tokens such as CSV/API/UI.\n"+
			"\u7b2c\u4e09\u884c=\u4e0b\u4e00\u6b65\uff0c\u53ea\u80fd\u539f\u6587\u5beb \u5f85\u547d\u3001\u8f38\u51fa\u3001\u9078\u9805 \u5176\u4e2d\u4e4b\u4e00\uff0c\u4e0d\u53ef\u7ffb\u8b6f\u3002\n"+
			"\n"+
			"\u6bcf\u500b\u52d5\u4f5c\u8a5e\u7684\u610f\u7fa9\uff1a\n"+
			"\u8f38\u51fa=\u5df2\u77e5\u7b54\u6848\u3001\u4e00\u822c\u804a\u5929\u3001\u5bd2\u669e\u3001\u60c5\u7dd2\u56de\u61c9\uff0c\u76f4\u63a5\u628a\u7b54\u6848\u5beb\u5728\u7b2c\u4e8c\u884c\u3002\n"+
			"\u641c\u5c0b=\u9700\u8981\u7cfb\u7d71\u5e6b\u5fd9\u67e5\u672c\u6a5f\u8cc7\u6599\uff0c\u7b2c\u4e8c\u884c\u5beb\u8981\u67e5\u7684\u95dc\u9375\u5b57\u6216\u76ee\u6a19\u3002\n"+
			"\u641c\u5c0b\u512a\u5148\uff1a\u4f7f\u7528\u8005\u8aaa\u67e5\u3001\u641c\u5c0b\u3001\u627e\u8cc7\u6599\u3001\u904b\u52e2\uff0c\u4e14\u6c92\u6709\u660e\u78ba\u8aaa\u4e0a\u7db2\u3001\u7db2\u8def\u3001\u6700\u65b0\u65b0\u805e\u3001\u5916\u90e8\u7db2\u7ad9\u6642\uff0c\u7b2c\u4e00\u884c\u9078 \u641c\u5c0b\u3002\n"+
			"\u7db2\u8def=\u9700\u8981\u4e0a\u7db2\u67e5\u6700\u65b0\u65b0\u805e\u6216\u5916\u90e8\u7db2\u7ad9\u8cc7\u8a0a\uff0c\u7b2c\u4e8c\u884c\u5beb\u67e5\u8a62\u8a5e\u3002\n"+
			"\u8b80\u53d6=\u53d6\u5f97\u67d0\u500b\u6a94\u6848\u6216\u7db2\u5740\u7684\u5167\u5bb9\u4e26\u56de\u5831\uff0c\u7b2c\u4e8c\u884c\u5beb\u76ee\u6a19\u3002\n"+
			"\u5217\u51fa=\u5217\u51fa\u7cfb\u7d71\u6e05\u55ae\uff08\u6a94\u6848\u3001\u6280\u80fd\u3001\u6392\u7a0b\u7b49\uff09\uff0c\u7b2c\u4e8c\u884c\u5beb\u8981\u5217\u4ec0\u9ebc\u3002\n"+
			"\u958b\u555f=\u7528\u5916\u90e8\u61c9\u7528\u7a0b\u5f0f\u5448\u73fe\uff0c\u7b2c\u4e8c\u884c\u5beb\u8981\u958b\u555f\u7684\u76ee\u6a19\u3002\n"+
			"\u5beb\u5165=\u65b0\u589e\u6216\u4fee\u6539\u6a94\u6848\u5167\u5bb9\uff0c\u7b2c\u4e8c\u884c\u5beb\u6a94\u540d\u6216\u5167\u5bb9\u3002\n"+
			"\u5132\u5b58=\u4fdd\u5b58\u5df2\u5b58\u5728\u7684\u5167\u5bb9\uff0c\u7b2c\u4e8c\u884c\u5beb\u8981\u5b58\u4ec0\u9ebc\u3002\n"+
			"\u532f\u5165=\u628a\u5916\u90e8\u8cc7\u6e90\u52a0\u5165\u7cfb\u7d71\uff0c\u7b2c\u4e8c\u884c\u5beb\u4f86\u6e90\u3002\n"+
			"\u532f\u51fa=\u7522\u751f\u6a94\u6848\uff0c\u7b2c\u4e8c\u884c\u5beb\u8981\u532f\u51fa\u7684\u5167\u5bb9\u3002\n"+
			"\u7a0b\u5f0f=\u9700\u8981\u88fd\u4f5c\u4e00\u652f\u7368\u7acb\u7a0b\u5f0f\uff0c\u7b2c\u4e8c\u884c\u5beb\u7a0b\u5f0f\u7528\u9014\u3002\n"+
			"\u6d41\u7a0b=\u7528\u65e2\u6709 skill \u8655\u7406\u8cc7\u6599\uff0c\u7b2c\u4e8c\u884c\u5beb\u8981\u8655\u7406\u7684\u5167\u5bb9\u3002\n"+
			"git=\u7248\u63a7\u64cd\u4f5c\uff0c\u7b2c\u4e8c\u884c\u5beb\u8981\u505a\u7684\u7248\u63a7\u52d5\u4f5c\u3002\n"+
			"\u6392\u7a0b=\u5b9a\u6642\u6216\u63d0\u9192\uff0c\u7b2c\u4e8c\u884c\u5beb\u6642\u9593\u8207\u5167\u5bb9\u3002\n"+
			"\u5c55\u958b=\u53d6\u56de\u5c0d\u8a71\u4e2d [S-NNN] \u6458\u8981\u88ab\u58d3\u7e2e\u7684\u7d30\u7bc0\uff0c\u7b2c\u4e8c\u884c\u5beb\u6a19\u7c64\u6216\u95dc\u9375\u5b57\u3002\n"+
			"\u64cd\u4f5c=\u53ea\u4ee3\u8868\u57f7\u884c\u6216\u91cd\u73fe\u5df2\u4fdd\u5b58\u7684\u87a2\u5e55 replay \u64cd\u4f5c\uff1b\u6c92\u6709\u660e\u78ba\u300c\u91cd\u73fe\uff0f\u56de\u653e\uff0f\u7167\u505a\uff0f\u57f7\u884c\u5df2\u4fdd\u5b58\u64cd\u4f5c\u300d\u7684\u610f\u601d\u6642\u4e0d\u8981\u9078 \u64cd\u4f5c\u3002\n"+
			"\u63d0\u554f=\u53ea\u6709\u7f3a\u5c11\u5fc5\u8981\u8cc7\u8a0a\u6642\u624d\u88dc\u554f\uff0c\u7b2c\u4e8c\u884c\u5beb\u554f\u984c\u6587\u5b57\u3002\n"+
			"\u9078\u9805=\u53ea\u6709\u4f7f\u7528\u8005\u660e\u78ba\u8981\u6c42\u9078\u64c7\u6642\u624d\u986f\u793a\uff1b\u7b2c\u4e8c\u884c\u7528\u4f7f\u7528\u8005\u8a9e\u8a00\u548c\u4f7f\u7528\u8005\u63d0\u4f9b\u7684\u503c\u5217\u51fa\u9078\u9805\uff0c\u7528\u9813\u865f\u6216\u9017\u865f\u5206\u9694\uff0c\u4f8b\u5982 6\u670821\u65e5\u30016\u670822\u65e5\u30016\u670823\u65e5\u3002\u4e0d\u8981\u7ffb\u8b6f\u9078\u9805\uff0c\u4e0d\u8981\u6539\u6210\u7bc4\u4f8b\u5167\u5bb9\uff0c\u4e0d\u8981\u4f7f\u7528\u6ce8\u97f3\u7b26\u865f\uff0c\u4e0d\u8981\u4f7f\u7528 \u3124 \u6216 \u310c\uff0c\u4e0d\u8981\u81ea\u5df1\u52a0\u7121\u95dc\u5b57\u5143\u3002\n"+
			"\u7a0b\u5f0f\u3001\u6d41\u7a0b\u7684\u7b2c\u4e09\u884c\u901a\u5e38\u5beb \u8f38\u51fa\uff1b\u5176\u9918\u901a\u5e38\u5beb \u5f85\u547d\u3002\n"+
			"\u8de8\u8a9e\u8a00\u6ce8\u610f\uff1a\u7b2c\u4e00\u884c\u8207\u7b2c\u4e09\u884c\u662f\u7cfb\u7d71\u6b04\u4f4d\uff0c\u4fdd\u6301\u4e2d\u6587\u52d5\u4f5c\u8a5e\uff1b\u7b2c\u4e8c\u884c\u662f\u7d66\u4f7f\u7528\u8005\u6216\u5de5\u5177\u7684\u76ee\u6a19\uff0c\u4e0d\u8981\u6a21\u4eff\u7bc4\u4f8b\u6539\u6210\u4e2d\u6587\u3002\n"+
			"\u7db2\u8def\u641c\u5c0b\u82e5\u662f\u67e5\u52d5\u7269\uff0c\u7b2c\u4e8c\u884c\u8981\u5beb\u6210\u52d5\u7269\u7269\u7a2e\u6216\u52d5\u7269\u4f8b\u5b50\u7684\u641c\u5c0b\u8a5e\uff0c\u4f8b\u5982 nocturnal animal species examples\uff0c\u4e0d\u8981\u7559\u4e0b\u5bb9\u6613\u8b8a\u6210\u96fb\u5f71\u7247\u540d\u7684\u6a21\u7cca\u8a5e\u3002\n"+
			"\n"+
			"\u7bc4\u4f8b\uff08\u6bcf\u500b\u7bc4\u4f8b\u90fd\u662f\u4e09\u884c\uff09\uff1a\n"+
			"\u8f38\u51fa\n\u4f60\u597d\u554a\uff0c\u6709\u4ec0\u9ebc\u6211\u53ef\u4ee5\u5e6b\u5fd9\u7684\uff1f\n\u5f85\u547d\n\n"+
			"\u641c\u5c0b\n\u4eca\u65e5\u5c04\u624b\u5ea7\u904b\u52e2\n\u5f85\u547d\n\n"+
			"\u7db2\u8def\n\u591c\u884c\u6027\u52d5\u7269\u7269\u7a2e\n\u5f85\u547d\n\n"+
			"\u8b80\u53d6\n<local document path>\n\u5f85\u547d\n\n"+
			"\u5217\u51fa\n\u76ee\u524d\u53ef\u7528\u7684\u6392\u7a0b\n\u5f85\u547d\n\n"+
			"\u5beb\u5165\n\u6703\u8b70\u8a18\u9304.md\n\u5f85\u547d\n\n"+
			"\u532f\u51fa\n\u672c\u6708\u92b7\u552e\u5831\u8868\n\u5f85\u547d\n\n"+
			"\u7a0b\u5f0f\n\u628a CSV \u8f49\u6210\u7d71\u8a08\u5716\u7684\u7a0b\u5f0f\n\u8f38\u51fa\n\n"+
			"\u6d41\u7a0b\n\u7528\u96fb\u6599BOM\u6280\u80fd\u6574\u7406\u9019\u4efd\u6e05\u55ae\n\u8f38\u51fa\n\n"+
			"\u63d0\u554f\n\u4f60\u60f3\u67e5\u54ea\u4e00\u5929\u7684\u904b\u52e2\uff1f\n\u5f85\u547d\n\n"+
			"\u9078\u9805\n6\u670821\u65e5\u30016\u670822\u65e5\u30016\u670823\u65e5\n\u9078\u9805\n\n"+
			"\u9078\u9805\n6\uc6d4 21\uc77c\u30016\uc6d4 22\uc77c\u30016\uc6d4 23\uc77c\n\u9078\u9805\n\n"+
			"\u9078\u9805\nJune 21\u3001June 22\u3001June 23\n\u9078\u9805\n\n"+
			"\u7db2\u8def\n\u591c\u884c\u6027\u306e\u52d5\u7269\n\u5f85\u547d\n\n"+
			"\u641c\u5c0b\n\uc624\ub298 \uc0ac\uc218\uc790\ub9ac \uc6b4\uc138\n\u5f85\u547d\n\n"+
			"\u7a0b\u5f0f\nCSV\u3092\u30b0\u30e9\u30d5\u306b\u5909\u63db\u3059\u308b\u30d7\u30ed\u30b0\u30e9\u30e0\n\u8f38\u51fa\n\n"+
			"\u7a0b\u5f0f\nCSV\ub97c \uadf8\ub798\ud504\ub85c \ubcc0\ud658\ud558\ub294 \ud504\ub85c\uadf8\ub7a8\n\u8f38\u51fa\n\n"+
			"\u7a0b\u5f0f\nMake a program that charts animal counts from CSV\n\u8f38\u51fa\n\n"+
			"\nQ: %s\n",
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
func (a *App) DetectModelPollution(path string) (*wa3_media.PollutionReport, error) {
	report, err := a.wa3Media.DetectMediaPollution(path)
	if err == nil && report != nil && report.IsPollutionRisk {
		a.eventBus.Emit("wa3:pollution_detected", map[string]interface{}{"path": path, "score": report.WeightedTotal})
	}
	return report, err
}
