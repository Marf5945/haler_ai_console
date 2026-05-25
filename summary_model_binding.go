package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
		out, err := exec.CommandContext(ctx, path, "list").Output()
		if err == nil {
			return parseOllamaListOutput(string(out))
		}
	}
	if models := scanOllamaModelLibrary(os.Getenv("OLLAMA_MODELS")); len(models) > 0 {
		return models
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	if models := scanOllamaModelLibrary(filepath.Join(home, "ollama")); len(models) > 0 {
		return models
	}
	return scanOllamaModelLibrary(filepath.Join(home, ".ollama", "models"))
}

func resolveOllamaExecutable() string {
	if path, err := exec.LookPath("ollama"); err == nil {
		return path
	}
	for _, path := range []string{
		"/opt/homebrew/bin/ollama",
		"/usr/local/bin/ollama",
		"/Applications/Ollama.app/Contents/Resources/ollama",
	} {
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return path
		}
	}
	return ""
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
		options = append(options, SummaryModelOption{
			Provider: "ollama",
			ID:       id,
			Label:    "Ollama - " + id,
			Endpoint: "http://localhost:11434",
		})
	}
	return options
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
		id := modelName + ":" + tag
		if namespace != "library" {
			id = namespace + "/" + id
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
	client := http.Client{Timeout: 800 * time.Millisecond}
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
