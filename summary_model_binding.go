package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ui_console/internal/urlsafe"
	"ui_console/shared/executil"
	"ui_console/shared/settings"
)

type SummaryModelOption struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Label    string `json:"label"`
	Endpoint string `json:"endpoint"`
}

type SummaryModelScanResult struct {
	Options []SummaryModelOption `json:"options"`
	Message string               `json:"message"`
}

func (a *App) GetSummaryModelSettings() settings.SummaryModelSettings {
	return a.settingsService.SummaryModelSettings()
}

func (a *App) SaveSummaryModelSettings(next settings.SummaryModelSettings) settings.SummaryModelSettings {
	return a.settingsService.SaveSummaryModelSettings(next)
}

func (a *App) ScanLocalSummaryModels() SummaryModelScanResult {
	options := append(scanOllamaModels(), scanLMStudioModels()...)
	if len(options) == 0 {
		return SummaryModelScanResult{Options: options, Message: "未偵測到本機模型"}
	}
	return SummaryModelScanResult{Options: options}
}

func scanOllamaModels() []SummaryModelOption {
	if path := resolveOllamaExecutable(); path != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
		defer cancel()
		out, err := executil.CommandContext(ctx, path, "list").Output()
		if err == nil {
			return parseOllamaListOutput(string(out))
		}
	}
	// CLI 不在（或掛了）但 daemon 在跑 → 直接問 /api/tags，比檔案掃描準。
	if models := scanOllamaModelsViaAPI("http://localhost:11434"); len(models) > 0 {
		return models
	}
	if models := scanOllamaModelLibrary(os.Getenv("OLLAMA_MODELS")); len(models) > 0 {
		return models
	}
	if home, _ := os.UserHomeDir(); home != "" {
		if models := scanOllamaModelLibrary(filepath.Join(home, "ollama")); len(models) > 0 {
			return models
		}
		if models := scanOllamaModelLibrary(filepath.Join(home, ".ollama", "models")); len(models) > 0 {
			return models
		}
	}
	// Linux systemd 服務模式：daemon 以 ollama 使用者跑，模型庫在它的家目錄。
	return scanOllamaModelLibrary(systemOllamaModelDir())
}

func resolveOllamaExecutable() string {
	if path, err := exec.LookPath("ollama"); err == nil {
		return path
	}
	// Windows：exe 沒有 unix 執行位，只驗存在即可；預設裝在 %LOCALAPPDATA%\Programs\Ollama。
	if runtime.GOOS == "windows" {
		var candidates []string
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			candidates = append(candidates, filepath.Join(localAppData, "Programs", "Ollama", "ollama.exe"))
		}
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			candidates = append(candidates, filepath.Join(programFiles, "Ollama", "ollama.exe"))
		}
		for _, path := range candidates {
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				return path
			}
		}
		return ""
	}
	// macOS + Linux 常見安裝位置（Linux：官方 install.sh → /usr/local/bin、
	// deb/rpm → /usr/bin、snap → /snap/bin、使用者自裝 → ~/.local/bin）。
	candidates := []string{
		"/opt/homebrew/bin/ollama",
		"/usr/local/bin/ollama",
		"/usr/bin/ollama",
		"/snap/bin/ollama",
		"/Applications/Ollama.app/Contents/Resources/ollama",
	}
	if home, _ := os.UserHomeDir(); home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", "ollama"))
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return path
		}
	}
	return ""
}

// scanOllamaModelsViaAPI — CLI 不可用時的 HTTP fallback：GET /api/tags。
// 沿用 scanLMStudioModels 的 SEC-05 / SEC-W09 防護（PolicyLocalLLM + 1MB 上限）。
func scanOllamaModelsViaAPI(baseURL string) []SummaryModelOption {
	client := urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "model_scan", 800*time.Millisecond)
	resp, err := client.Get(baseURL + "/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil
	}
	options := make([]SummaryModelOption, 0, len(payload.Models))
	for _, m := range payload.Models {
		id := strings.TrimSpace(m.Name)
		if id == "" || !isOllamaGenerativeModelID(id) {
			continue
		}
		options = append(options, SummaryModelOption{
			Provider: "ollama",
			ID:       id,
			Label:    "Ollama - " + id,
			Endpoint: "http://localhost:11434",
		})
	}
	return options
}

func parseOllamaListOutput(out string) []SummaryModelOption {
	lines := strings.Split(out, "\n")
	options := make([]SummaryModelOption, 0)
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		id := fields[0]
		if !isOllamaGenerativeModelID(id) {
			continue
		}
		options = append(options, SummaryModelOption{
			Provider: "ollama",
			ID:       id,
			Label:    "Ollama - " + id,
			Endpoint: "http://localhost:11434",
		})
	}
	return options
}

func isOllamaGenerativeModelID(id string) bool {
	normalized := strings.ToLower(strings.TrimSpace(id))
	if normalized == "" {
		return false
	}
	name := normalized
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		name = name[slash+1:]
	}
	if colon := strings.Index(name, ":"); colon >= 0 {
		name = name[:colon]
	}
	nonGenerativeMarkers := []string{
		"embed",
		"embedding",
		"bge-",
		"e5-",
		"gte-",
		"jina-clip",
		"jina-embeddings",
		"minilm",
		"rerank",
		"ranker",
		"sentence-transformers",
		"text-embedding",
	}
	for _, marker := range nonGenerativeMarkers {
		if strings.Contains(name, marker) {
			return false
		}
	}
	return true
}

func isOllamaModelLibrary(path string) bool {
	path = expandUserPath(path)
	if path == "" {
		return false
	}
	for _, name := range []string{"blobs", "manifests"} {
		info, err := os.Stat(filepath.Join(path, name))
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

func scanOllamaModelLibrary(path string) []SummaryModelOption {
	path = expandUserPath(path)
	if !isOllamaModelLibrary(path) {
		return nil
	}
	root := filepath.Join(path, "manifests")
	options := make([]SummaryModelOption, 0)
	_ = filepath.WalkDir(root, func(candidate string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, candidate)
		if err != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 {
			return nil
		}
		tag := parts[len(parts)-1]
		modelName := parts[len(parts)-2]
		namespace := parts[len(parts)-3]
		registry := strings.Join(parts[:len(parts)-3], "/")
		prefix := namespace
		if registry != "" && registry != "registry.ollama.ai" {
			prefix = registry + "/" + namespace
		}
		id := modelName + ":" + tag
		if prefix != "library" {
			id = prefix + "/" + id
		}
		if !isOllamaGenerativeModelID(id) {
			return nil
		}
		options = append(options, SummaryModelOption{
			Provider: "ollama",
			ID:       id,
			Label:    "Ollama - " + id,
			Endpoint: "http://localhost:11434",
		})
		return nil
	})
	return options
}

func expandUserPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return path
	}
	if path == "$HOME" || path == "~" {
		return home
	}
	if strings.HasPrefix(path, "$HOME/") {
		return filepath.Join(home, strings.TrimPrefix(path, "$HOME/"))
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

// §29.3 DismissSummarization — 使用者點「稍後」，啟動 5000 字 cooldown。
func (a *App) DismissSummarization() {
	if a.cliAdapter != nil {
		if adapter, ok := a.cliAdapter.(interface{ DismissSummarization() }); ok {
			adapter.DismissSummarization()
		}
	}
}

func scanLMStudioModels() []SummaryModelOption {
	// SEC-05 2a: 本機 model 掃描走 PolicyLocalLLM。
	client := urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "model_scan", 800*time.Millisecond)
	resp, err := client.Get("http://localhost:1234/v1/models")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	// SEC-W09（2026-05-24）：限制 ollama tags 回應 1 MB，避免惡意/錯誤 server 灌爆記憶體。
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil
	}
	options := make([]SummaryModelOption, 0, len(payload.Data))
	for _, model := range payload.Data {
		if strings.TrimSpace(model.ID) == "" {
			continue
		}
		options = append(options, SummaryModelOption{
			Provider: "lmstudio",
			ID:       model.ID,
			Label:    "LM Studio - " + model.ID,
			Endpoint: "http://localhost:1234/v1",
		})
	}
	return options
}
