// Package adapter_registry manages the list of available CLI adapters
// and their runtime status (online/offline/degraded).
//
// This replaces the hardcoded fallback adapter list in GetConsoleState / App.jsx.
// Each adapter entry describes an installed CLI (Claude, Codex, Gemini, etc.)
// and its current connectivity state.
//
// Spec reference: AI_Console_Spec_v3_3_2 §4 — 外部 CLI adapter 規格.
// Legacy reference: TASKS_1_1.md 遺留能力 #1.
package adapter_registry

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ui_console/data/storage"
)

// Status represents the connectivity state of a CLI adapter.
type Status string

const (
	StatusOnline   Status = "online"
	StatusOffline  Status = "offline"
	StatusDegraded Status = "degraded" // partially functional (e.g. rate-limited)
)

// Adapter describes a single registered CLI adapter.
type Adapter struct {
	ID        string    `json:"id"`                 // unique key, e.g. "claude-cli"
	Name      string    `json:"name"`               // display name, e.g. "Claude"
	Icon      string    `json:"icon"`               // single-char icon for Sidebar
	Path      string    `json:"path,omitempty"`     // executable path for the local CLI
	Endpoint  string    `json:"endpoint,omitempty"` // API endpoint for local models
	Model     string    `json:"model,omitempty"`    // model ID for local models (e.g. "qwen2.5:14b")
	Kind      string    `json:"kind,omitempty"`     // cli / api / sub / main
	Status    Status    `json:"status"`             // current connectivity
	LastCheck time.Time `json:"last_check"`         // last health-check timestamp
}

// Service manages the adapter registry.
type Service struct {
	mu       sync.Mutex
	store    *storage.JSONStore[[]Adapter]
	adapters []Adapter
}

// NewService creates or loads an adapter registry from disk.
// Missing storage starts empty: adapters must be registered by real CLI
// detection/connection flow, otherwise the UI would show mock CLIs as usable.
func NewService(dataRoot string) *Service {
	svc := &Service{
		store: storage.NewJSONStore[[]Adapter](
			filepath.Join(dataRoot, "data", "preferences", "adapter_registry.json"),
		),
	}
	if loaded, err := svc.store.Load(); err == nil && loaded != nil {
		svc.adapters = loaded
	}
	return svc
}

// ListAvailable returns all registered adapters with their current status.
func (s *Service) ListAvailable() []Adapter {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Adapter, len(s.adapters))
	copy(result, s.adapters)
	return result
}

// GetStatus returns the status of a single adapter by ID.
func (s *Service) GetStatus(adapterID string) (Adapter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.adapters {
		if a.ID == adapterID {
			return a, nil
		}
	}
	return Adapter{}, fmt.Errorf("adapter_registry: adapter %q not found", adapterID)
}

// SetStatus updates the connectivity status of an adapter.
// Typically called by health-check goroutine or after a CLI call fails.
func (s *Service) SetStatus(adapterID string, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.adapters {
		if a.ID == adapterID {
			s.adapters[i].Status = status
			s.adapters[i].LastCheck = time.Now()
			return s.store.SaveRaw(s.adapters)
		}
	}
	return fmt.Errorf("adapter_registry: adapter %q not found", adapterID)
}

// Rename updates the sidebar display name for CLI/API adapters.
func (s *Service) Rename(adapterID, displayName string) (Adapter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := strings.TrimSpace(displayName)
	if name == "" {
		return Adapter{}, fmt.Errorf("adapter_registry: display name must not be empty")
	}
	for i, a := range s.adapters {
		if a.ID == adapterID {
			s.adapters[i].Name = name
			s.adapters[i].LastCheck = time.Now()
			if err := s.store.SaveRaw(s.adapters); err != nil {
				return Adapter{}, err
			}
			return s.adapters[i], nil
		}
	}
	return Adapter{}, fmt.Errorf("adapter_registry: adapter %q not found", adapterID)
}

// Reorder persists the sidebar order for CLI/API adapters.
func (s *Service) Reorder(orderIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(orderIDs) == 0 {
		return nil
	}

	byID := make(map[string]Adapter, len(s.adapters))
	for _, adapter := range s.adapters {
		byID[adapter.ID] = adapter
	}

	seen := make(map[string]bool, len(orderIDs))
	next := make([]Adapter, 0, len(s.adapters))
	for _, id := range orderIDs {
		if adapter, ok := byID[id]; ok && !seen[id] {
			next = append(next, adapter)
			seen[id] = true
		}
	}
	for _, adapter := range s.adapters {
		if !seen[adapter.ID] {
			next = append(next, adapter)
		}
	}

	s.adapters = next
	return s.store.SaveRaw(s.adapters)
}

// Register adds a new adapter to the registry. Deduplicates by ID.
func (s *Service) Register(id, name, icon string, paths ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := ""
	if len(paths) > 0 {
		path = paths[0]
	}
	for _, a := range s.adapters {
		if a.ID == id {
			if path == "" || a.Path == path {
				return nil // already registered
			}
			for i := range s.adapters {
				if s.adapters[i].ID == id {
					s.adapters[i].Path = path
					s.adapters[i].LastCheck = time.Now()
					return s.store.SaveRaw(s.adapters)
				}
			}
		}
	}
	s.adapters = append(s.adapters, Adapter{
		ID:        id,
		Name:      name,
		Icon:      icon,
		Path:      path,
		Status:    StatusOffline,
		LastCheck: time.Now(),
	})
	return s.store.SaveRaw(s.adapters)
}

// RegisterAPI adds an LLM API-backed adapter. Secret material is stored outside
// this registry; the entry only gives the sidebar a selectable provider.
func (s *Service) RegisterAPI(id, name, icon string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.adapters {
		if a.ID == id {
			s.adapters[i].Name = name
			s.adapters[i].Icon = icon
			s.adapters[i].Kind = "api"
			s.adapters[i].Status = StatusOnline
			s.adapters[i].LastCheck = time.Now()
			return s.store.SaveRaw(s.adapters)
		}
	}
	s.adapters = append(s.adapters, Adapter{
		ID:        id,
		Name:      name,
		Icon:      icon,
		Kind:      "api",
		Status:    StatusOnline,
		LastCheck: time.Now(),
	})
	return s.store.SaveRaw(s.adapters)
}

// RegisterLocal adds a local-model-backed adapter (e.g. Ollama, LM Studio).
// The endpoint is stored so SendAPIMessage can reach the local inference server.
func (s *Service) RegisterLocal(id, name, icon, endpoint, model string, paths ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := ""
	if len(paths) > 0 {
		path = paths[0]
	}
	for i, a := range s.adapters {
		if a.ID == id {
			s.adapters[i].Name = name
			s.adapters[i].Icon = icon
			s.adapters[i].Kind = "local"
			if path != "" {
				s.adapters[i].Path = path
			}
			s.adapters[i].Endpoint = endpoint
			s.adapters[i].Model = model
			s.adapters[i].Status = StatusOnline
			s.adapters[i].LastCheck = time.Now()
			return s.store.SaveRaw(s.adapters)
		}
	}
	s.adapters = append(s.adapters, Adapter{
		ID:        id,
		Name:      name,
		Icon:      icon,
		Path:      path,
		Kind:      "local",
		Endpoint:  endpoint,
		Model:     model,
		Status:    StatusOnline,
		LastCheck: time.Now(),
	})
	return s.store.SaveRaw(s.adapters)
}

// ResolveExecutable returns the executable path for an adapter.
// It first uses the persisted path, then falls back to known CLI detection.
func (s *Service) ResolveExecutable(adapterID string) (string, error) {
	s.mu.Lock()
	for i, a := range s.adapters {
		if a.ID == adapterID && a.Path != "" {
			path := a.Path
			unwrapped := unwrapLauncherPath(path, inferBinaryNameForAdapter(adapterID))
			if unwrapped != path {
				s.adapters[i].Path = unwrapped
				s.adapters[i].LastCheck = time.Now()
				_ = s.store.SaveRaw(s.adapters)
				path = unwrapped
			}
			s.mu.Unlock()
			if isExecutableFile(path) {
				return path, nil
			}
			return "", fmt.Errorf("adapter_registry: adapter %q path %q is not executable", adapterID, path)
		}
	}
	s.mu.Unlock()

	for _, cli := range knownCLIs {
		if cli.AdapterID == adapterID {
			path := findKnownCLI(cli)
			if path == "" {
				return "", fmt.Errorf("adapter_registry: adapter %q executable not found", adapterID)
			}
			s.mu.Lock()
			for i := range s.adapters {
				if s.adapters[i].ID == adapterID && s.adapters[i].Path == "" {
					s.adapters[i].Path = path
					s.adapters[i].LastCheck = time.Now()
					_ = s.store.SaveRaw(s.adapters)
					break
				}
			}
			s.mu.Unlock()
			return path, nil
		}
	}
	return "", fmt.Errorf("adapter_registry: adapter %q not found", adapterID)
}

func inferBinaryNameForAdapter(adapterID string) string {
	for _, cli := range knownCLIs {
		if cli.AdapterID == adapterID {
			return cli.BinaryName
		}
	}
	return strings.TrimSuffix(adapterID, "-cli")
}

// --- CLI auto-detection ---

// knownCLI 描述一個已知的 CLI 程式，用於自動偵測。
type knownCLI struct {
	BinaryName     string   // 可執行檔名（PATH 中搜尋）
	AdapterID      string   // 對應的 adapter ID
	Label          string   // 顯示名稱
	Icon           string   // 單字 icon
	CandidatePaths []string // PATH 之外的明確候選位置，支援 $HOME
}

// knownCLIs 是本機可掃描的已知 CLI 列表。
// 可透過 AddKnownCLI 動態擴充。
var knownCLIs = []knownCLI{
	{BinaryName: "claude", AdapterID: "claude-cli", Label: "Claude", Icon: "C"},
	{
		BinaryName: "codex",
		AdapterID:  "codex-cli",
		Label:      "Codex",
		Icon:       "◎",
		CandidatePaths: []string{
			"/Applications/Codex.app/Contents/Resources/codex",
			"$HOME/.opencode/bin/codex",
		},
	},
	{
		BinaryName: "gemini",
		AdapterID:  "gemini-cli",
		Label:      "Gemini",
		Icon:       "✦",
		CandidatePaths: []string{
			"$HOME/gemini_cli/node_modules/.bin/gemini",
		},
	},
	{
		BinaryName: "ollama",
		AdapterID:  "ollama-cli",
		Label:      "Ollama",
		Icon:       "O",
		CandidatePaths: []string{
			"/opt/homebrew/bin/ollama",
			"/usr/local/bin/ollama",
			"/Applications/Ollama.app/Contents/Resources/ollama",
		},
	},
	{BinaryName: "aider", AdapterID: "aider-cli", Label: "Aider", Icon: "A"},
	{BinaryName: "copilot", AdapterID: "copilot-cli", Label: "Copilot", Icon: "⬡"},
}

// DetectResult 描述一個偵測結果。
type DetectResult struct {
	AdapterID string `json:"adapter_id"`
	Name      string `json:"name"`
	Path      string `json:"path"` // 偵測到的完整路徑，空字串表示未安裝
	Found     bool   `json:"found"`
}

// extraSearchPaths 回傳 macOS GUI app 中 PATH 不包含的常見 CLI 安裝路徑。
// macOS .app 啟動時 PATH 通常只有 /usr/bin:/bin:/usr/sbin:/sbin，
// npm global、Homebrew、cargo 等路徑都不在裡面。
func extraSearchPaths() []string {
	home, _ := os.UserHomeDir()
	paths := []string{
		"/opt/homebrew/bin", // Apple Silicon Homebrew
		"/usr/local/bin",    // Intel Homebrew / 手動安裝
		"/usr/local/sbin",
	}
	if home != "" {
		paths = append(paths,
			filepath.Join(home, ".local", "bin"),      // pipx, uv 等
			filepath.Join(home, ".opencode", "bin"),   // Codex/OpenCode wrapper
			filepath.Join(home, ".cargo", "bin"),      // Rust / cargo install
			filepath.Join(home, ".bun", "bin"),        // bun add -g
			filepath.Join(home, "bin"),                // 使用者自訂
			filepath.Join(home, ".npm-global", "bin"), // npm config prefix 自訂
			filepath.Join(home, "Library", "pnpm"),    // pnpm setup 預設
		)
		// npm global bin: 嘗試讀取 npm prefix
		if npmPrefix := resolveNpmPrefix(); npmPrefix != "" {
			paths = append(paths, filepath.Join(npmPrefix, "bin"))
		}
		// nvm 預設路徑
		nvmDir := os.Getenv("NVM_DIR")
		if nvmDir == "" {
			nvmDir = filepath.Join(home, ".nvm")
		}
		// 掃描所有 nvm 版本的 bin
		if entries, err := os.ReadDir(filepath.Join(nvmDir, "versions", "node")); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					paths = append(paths, filepath.Join(nvmDir, "versions", "node", e.Name(), "bin"))
				}
			}
		}
		// fnm (Fast Node Manager)
		fnmDir := filepath.Join(home, ".local", "share", "fnm", "node-versions")
		if entries, err := os.ReadDir(fnmDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					paths = append(paths, filepath.Join(fnmDir, e.Name(), "installation", "bin"))
				}
			}
		}
	}
	if runtime.GOOS == "linux" {
		paths = append(paths, "/snap/bin")
	}
	return paths
}

// resolveNpmPrefix 嘗試呼叫 npm prefix -g 取得全域路徑。
func resolveNpmPrefix() string {
	// 先用完整路徑找 npm，避免 PATH 問題
	npmBin := ""
	for _, p := range []string{"/opt/homebrew/bin/npm", "/usr/local/bin/npm"} {
		if _, err := os.Stat(p); err == nil {
			npmBin = p
			break
		}
	}
	if npmBin == "" {
		// fallback: 試 PATH
		var err error
		npmBin, err = exec.LookPath("npm")
		if err != nil {
			return ""
		}
	}
	// SEC-19: 加入 5 秒 timeout 防止 npm 卡住
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, npmBin, "prefix", "-g").Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("resolveNpmPrefix: npm prefix -g timeout after 5s")
		}
		return ""
	}
	return strings.TrimSpace(string(out))
}

// findBinary 在 PATH + extraSearchPaths 中搜尋可執行檔。
// 回傳第一個找到的完整路徑，找不到則回傳空字串。
func findBinary(name string) string {
	// 1. 先試標準 PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	// 2. 掃描額外路徑
	for _, dir := range extraSearchPaths() {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			// 確認有執行權限
			if info.Mode()&0o111 != 0 {
				return candidate
			}
		}
	}
	return ""
}

func findKnownCLI(cli knownCLI) string {
	if path := findBinary(cli.BinaryName); path != "" {
		return unwrapLauncherPath(path, cli.BinaryName)
	}
	for _, candidate := range cli.CandidatePaths {
		if path := expandHome(candidate); isExecutableFile(path) {
			return unwrapLauncherPath(path, cli.BinaryName)
		}
	}
	return ""
}

func unwrapLauncherPath(path, binaryName string) string {
	if binaryName != "codex" {
		return path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return path
	}
	content := string(data)
	if !strings.Contains(content, "codex-browser") || !strings.Contains(content, "CODEX_ORIGINAL_BIN=") {
		return path
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "export CODEX_ORIGINAL_BIN=") {
			continue
		}
		value := strings.TrimPrefix(line, "export CODEX_ORIGINAL_BIN=")
		value = strings.Trim(value, `"'`)
		if isExecutableFile(value) {
			return value
		}
	}
	return path
}

func expandHome(path string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return path
	}
	if path == "$HOME" {
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

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func resolveExecutablePath(path, binaryName string) (string, error) {
	path = expandHome(strings.TrimSpace(path))
	if path == "" {
		return "", fmt.Errorf("adapter_registry: path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("adapter_registry: path %q not found: %w", path, err)
	}
	if !info.IsDir() {
		if info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("adapter_registry: path %q is not executable", path)
		}
		return unwrapLauncherPath(path, binaryName), nil
	}

	names := []string{binaryName}
	if binaryName == "" {
		names = []string{"claude", "codex", "gemini", "ollama", "aider", "copilot"}
	}
	for _, name := range names {
		for _, rel := range []string{
			name,
			filepath.Join("bin", name),
			filepath.Join("node_modules", ".bin", name),
		} {
			candidate := filepath.Join(path, rel)
			if isExecutableFile(candidate) {
				return unwrapLauncherPath(candidate, name), nil
			}
		}
	}
	return "", fmt.Errorf("adapter_registry: no executable CLI found under %q", path)
}

func inferCLIName(name, path string) string {
	cleanName := strings.TrimSpace(name)
	if cleanName != "" {
		return cleanName
	}
	lowerPath := strings.ToLower(path)
	for _, known := range []string{"claude", "codex", "gemini", "ollama", "aider", "copilot"} {
		if strings.Contains(lowerPath, known) {
			return strings.ToUpper(known[:1]) + known[1:]
		}
	}
	base := strings.TrimSpace(filepath.Base(path))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "Custom CLI"
	}
	base = strings.TrimSuffix(base, "_cli")
	base = strings.TrimSuffix(base, "-cli")
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	fields := strings.Fields(base)
	if len(fields) == 0 {
		return "Custom CLI"
	}
	for i, field := range fields {
		fields[i] = strings.ToUpper(field[:1]) + field[1:]
	}
	return strings.Join(fields, " ")
}

// ResolveCustomCLI resolves an executable CLI path without mutating the registry.
// It accepts either an executable file or a project/install directory containing
// common CLI entrypoints such as node_modules/.bin/gemini.
func ResolveCustomCLI(name, path string) (DetectResult, error) {
	cleanName := inferCLIName(name, path)
	r := DetectResult{Name: cleanName}
	binaryName := strings.ToLower(strings.Fields(cleanName)[0])
	resolvedPath, err := resolveExecutablePath(path, binaryName)
	if err != nil {
		return r, err
	}
	r.AdapterID = strings.ToLower(strings.ReplaceAll(cleanName, " ", "-")) + "-cli"
	r.Path = resolvedPath
	r.Found = true
	return r, nil
}

// AutoDetect 掃描本機 PATH + 常見安裝路徑，找出已安裝的 CLI。
// **只偵測不自動註冊**——回傳偵測結果供 UI 顯示，由使用者選擇要啟用哪些。
// 要啟用（註冊）請呼叫 EnableDetectedCLI。
func (s *Service) AutoDetect() []DetectResult {
	var results []DetectResult
	for _, cli := range knownCLIs {
		path := findKnownCLI(cli)
		if path != "" {
			results = append(results, DetectResult{
				AdapterID: cli.AdapterID,
				Name:      cli.Label,
				Path:      path,
				Found:     true,
			})
		}
	}
	return results
}

// EnableDetectedCLI 由使用者在 UI 上選擇後呼叫，將偵測到的 CLI 註冊到 registry。
func (s *Service) EnableDetectedCLI(adapterID string) error {
	for _, cli := range knownCLIs {
		if cli.AdapterID == adapterID {
			path := findKnownCLI(cli)
			if path == "" {
				return fmt.Errorf("adapter_registry: adapter %q executable not found", adapterID)
			}
			return s.Register(cli.AdapterID, cli.Label, cli.Icon, path)
		}
	}
	return fmt.Errorf("adapter_registry: unknown adapter %q", adapterID)
}

// RegisterCustomCLI 讓使用者手動註冊一個自訂 CLI adapter。
// 會驗證 path 是否存在且可執行。
func (s *Service) RegisterCustomCLI(name, path string) (DetectResult, error) {
	r, err := ResolveCustomCLI(name, path)
	if err != nil {
		return r, err
	}

	// 產生 adapter ID
	icon := strings.ToUpper(r.Name[:1])
	if err := s.Register(r.AdapterID, r.Name, icon, r.Path); err != nil {
		return r, err
	}
	r.Found = true
	return r, nil
}

// Unregister 移除一個 adapter（用於使用者手動移除）。
func (s *Service) Unregister(adapterID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.adapters {
		if a.ID == adapterID {
			s.adapters = append(s.adapters[:i], s.adapters[i+1:]...)
			return s.store.SaveRaw(s.adapters)
		}
	}
	return fmt.Errorf("adapter_registry: adapter %q not found", adapterID)
}

// persistence 由 storage.JSONStore 處理。
