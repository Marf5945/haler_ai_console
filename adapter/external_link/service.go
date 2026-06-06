// Package external_link implements the v3.3.2 P0.6 external link preview,
// validate, register, and link_type routing contract.
//
// Link type routing:
//
//	external_service  → shown in main-screen tool area
//	adapter_candidate → shown in CLI Adapter list
//	llm_provider_candidate → starts API adapter setup
//	documentation     → reference-only, NOT entered into routing / execution area
//	mcp_server        → MCP server registration（獨立路徑，localhost 不受 HTTPS 限制）
//	unsupported       → rejected; NOT registered
//
// URL rules（#47 補強 — localhost 路徑分離）：
//   - Release mode: external_service / documentation 只接受 https://
//   - MCP server registration: localhost / 127.0.0.1 走 #42 quarantine + Review Card
//     流程，不受 release mode HTTPS 限制
//   - Dev mode: localhost 可作為 external_service / documentation 類型
package external_link

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ui_console/data/storage"
)

// ──────────────────────────────────────────────
// LinkType 分類
// ──────────────────────────────────────────────

// LinkType classifies a validated external link.
type LinkType string

const (
	LinkExternalService  LinkType = "external_service"  // goes into tool area
	LinkAdapterCandidate LinkType = "adapter_candidate" // goes into CLI Adapter list
	LinkLLMProvider      LinkType = "llm_provider_candidate"
	LinkDocumentation    LinkType = "documentation" // reference only, no execution
	LinkMCPServer        LinkType = "mcp_server"    // #47: MCP server（獨立路徑）
	LinkUnsupported      LinkType = "unsupported"   // rejected
)

// ExternalLink is one registered link entry.
type ExternalLink struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	LinkType     LinkType  `json:"link_type"`
	Label        string    `json:"label"`
	RegisteredAt time.Time `json:"registered_at"`
}

// PreviewResult is returned before the user confirms registration.
type PreviewResult struct {
	URL      string   `json:"url"`
	LinkType LinkType `json:"link_type"`
	Valid    bool     `json:"valid"`
	Reason   string   `json:"reason,omitempty"`
	// #47: MCP server 走獨立路徑時，提示需要走 quarantine + Review Card
	RequiresQuarantine bool   `json:"requires_quarantine,omitempty"`
	ProviderID         string `json:"provider_id,omitempty"`
	ProviderName       string `json:"provider_name,omitempty"`
	BaseURL            string `json:"base_url,omitempty"`
	APIKeyURL          string `json:"api_key_url,omitempty"`
	DocsURL            string `json:"docs_url,omitempty"`
	MatchedBy          string `json:"matched_by,omitempty"`
}

// Service manages external link preview, validation, and registration.
type Service struct {
	mu          sync.Mutex
	store       *storage.JSONStore[[]ExternalLink]
	links       []ExternalLink
	devModeFlag bool // set true only in dev builds
}

// NewService creates (or loads) an external link service.
func NewService(dataRoot string, devMode bool) *Service {
	svc := &Service{
		store: storage.NewJSONStore[[]ExternalLink](
			filepath.Join(dataRoot, "data", "preferences", "external_link_registry.json"),
		),
		devModeFlag: devMode,
	}
	if loaded, err := svc.store.Load(); err == nil && loaded != nil {
		svc.links = loaded
	}
	return svc
}

// ──────────────────────────────────────────────
// Preview / Register
// ──────────────────────────────────────────────

// Preview validates a URL and determines its link_type without registering it.
// This must be called before offering the user a Register option.
//
// #47 路徑分離：
//   - MCP server 的 localhost 不走本服務的 register，而是走 #42 quarantine 流程。
//   - Preview 回傳 RequiresQuarantine = true 時，前端應導向 package_import 流程。
func (s *Service) Preview(url string) PreviewResult {
	// ── #47: 先檢查是否為 MCP server localhost ──
	if isMCPServerURL(url) {
		return PreviewResult{
			URL:                url,
			LinkType:           LinkMCPServer,
			Valid:              true,
			Reason:             "MCP server registration 走 quarantine + Review Card 流程",
			RequiresQuarantine: true,
		}
	}

	if err := s.validateURL(url); err != nil {
		return PreviewResult{URL: url, LinkType: LinkUnsupported, Valid: false, Reason: err.Error()}
	}
	if match, ok := DetectLLMProviderURL(url); ok {
		return PreviewResult{
			URL:          url,
			LinkType:     LinkLLMProvider,
			Valid:        true,
			Reason:       "偵測到 LLM/API Provider，確定後應進入 API 接口設定流程",
			ProviderID:   match.Provider.ID,
			ProviderName: match.Provider.Name,
			BaseURL:      match.Provider.BaseURL,
			APIKeyURL:    match.Provider.APIKeyURL,
			DocsURL:      match.Provider.DocsURL,
			MatchedBy:    match.MatchedBy,
		}
	}
	lt := classifyURL(url)
	return PreviewResult{URL: url, LinkType: lt, Valid: lt != LinkUnsupported, Reason: ""}
}

// Register adds a link to the registry after user confirmation.
// Returns an error for unsupported link types or invalid URLs.
// documentation links are accepted but flagged as reference-only.
//
// #47: MCP server localhost 不走此方法，走 package_import 的 quarantine 流程。
func (s *Service) Register(url, label string) (*ExternalLink, error) {
	// ── #47: 攔截 MCP server URL ──
	if isMCPServerURL(url) {
		return nil, fmt.Errorf("external_link: MCP server URL %q must go through quarantine + Review Card flow (package_import), not direct registration", url)
	}

	if err := s.validateURL(url); err != nil {
		return nil, err
	}
	lt := classifyURL(url)
	if match, ok := DetectLLMProviderURL(url); ok {
		lt = LinkLLMProvider
		label = strings.TrimSpace(label)
		if label == "" {
			label = match.Provider.Name
		}
	}
	if lt == LinkUnsupported {
		return nil, fmt.Errorf("external_link: URL %q is unsupported and cannot be registered", url)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Deduplicate by URL.
	for _, l := range s.links {
		if l.URL == url {
			return &l, nil
		}
	}

	link := ExternalLink{
		ID:           fmt.Sprintf("link-%d", time.Now().UnixNano()),
		URL:          url,
		LinkType:     lt,
		Label:        label,
		RegisteredAt: time.Now(),
	}
	s.links = append(s.links, link)
	if err := s.store.SaveRaw(s.links); err != nil {
		return nil, err
	}
	return &link, nil
}

// RegisterAdapterCandidate 已廢棄——adapter 註冊現在只走 adapter_registry。
// Deprecated: adapter registration is handled exclusively by adapter_registry.

// ListByType returns all registered links of a given type.
// documentation links must NOT be passed to the routing / execution area.
func (s *Service) ListByType(lt LinkType) []ExternalLink {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []ExternalLink
	for _, l := range s.links {
		if l.LinkType == lt {
			result = append(result, l)
		}
	}
	return result
}

// ──────────────────────────────────────────────
// #47 MCP Server localhost 偵測
// ──────────────────────────────────────────────

// isMCPServerURL 判斷 URL 是否為本地 MCP server。
// MCP server 通常跑在 localhost / 127.0.0.1，並包含 mcp 相關路徑。
// 即使不含 /mcp 路徑，只要是 localhost 且非 dev mode，也視為 MCP server 候選
// （因為 release mode 下 localhost external_service 本身就被阻擋）。
func isMCPServerURL(url string) bool {
	lower := strings.ToLower(strings.TrimSpace(url))
	isLocalhost := strings.HasPrefix(lower, "http://localhost") ||
		strings.HasPrefix(lower, "http://127.0.0.1") ||
		strings.HasPrefix(lower, "https://localhost") ||
		strings.HasPrefix(lower, "https://127.0.0.1")
	if !isLocalhost {
		return false
	}
	// 明確含 mcp 路徑 → 一定是 MCP server
	if strings.Contains(lower, "/mcp") || strings.Contains(lower, "mcp-server") ||
		strings.Contains(lower, "modelcontextprotocol") {
		return true
	}
	return false
}

// ──────────────────────────────────────────────
// URL validation
// ──────────────────────────────────────────────

// validateURL 驗證非 MCP server 的 URL。
// #47 路徑分離：此驗證僅適用於 external_service / documentation / adapter_candidate。
func (s *Service) validateURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("external_link: URL must not be empty")
	}
	if strings.HasPrefix(url, "https://") {
		return nil // always accepted
	}
	if strings.HasPrefix(url, "http://localhost") || strings.HasPrefix(url, "http://127.0.0.1") {
		if s.devModeFlag {
			return nil // localhost only in dev mode for external_service / documentation
		}
		return fmt.Errorf("external_link: localhost URLs are only accepted in dev mode for external_service/documentation; MCP server registration uses a separate quarantine flow")
	}
	if strings.HasPrefix(url, "http://") {
		return fmt.Errorf("external_link: plain http:// is not accepted in release mode; use https://")
	}
	return fmt.Errorf("external_link: unsupported URL scheme: %q", url)
}

// classifyURL heuristically determines the link type from the URL.
// In production this would also consult an allowlist / manifest.
func classifyURL(url string) LinkType {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "/docs") || strings.Contains(lower, "/documentation") ||
		strings.Contains(lower, "readme") || strings.Contains(lower, "/wiki"):
		return LinkDocumentation
	case strings.Contains(lower, "/adapter") || strings.Contains(lower, "/cli"):
		return LinkAdapterCandidate
	case strings.HasPrefix(lower, "https://"):
		return LinkExternalService
	default:
		return LinkUnsupported
	}
}

// persistence 由 storage.JSONStore 處理。
