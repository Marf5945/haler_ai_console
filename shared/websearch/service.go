// Package websearch provides provider-backed web search for AI Console.
package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ProviderTavily  = "tavily"
	ProviderBrave   = "brave"
	ProviderExa     = "exa"
	ProviderSerpAPI = "serpapi"

	DefaultLimit     = 5
	maxLimit         = 8
	defaultTimeout   = 12 * time.Second
	maxResponseBytes = 2 * 1024 * 1024
	maxSnippetRunes  = 260
	maxOutputRunes   = 5200
)

var (
	ErrEmptyQuery        = errors.New("websearch: empty query")
	ErrNoResults         = errors.New("websearch: no results")
	ErrProviderMissing   = errors.New("websearch: provider is not configured")
	ErrCredentialMissing = errors.New("websearch: credential is missing")
	tagPattern           = regexp.MustCompile(`<[^>]+>`)
)

type ProviderOption struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Fields            []string `json:"fields"`
	DocsURL           string   `json:"docs_url"`
	FreeTierHint      string   `json:"free_tier_hint,omitempty"`
	BestFor           string   `json:"best_for,omitempty"`
	SetupGuide        []string `json:"setup_guide,omitempty"`
	UsageHint         string   `json:"usage_hint,omitempty"`
	APIKeyPlaceholder string   `json:"api_key_placeholder,omitempty"`
	PreferredOrder    int      `json:"preferred_order,omitempty"`
}

type ProviderConfig struct {
	ProviderID string `json:"provider_id"`
	APIKey     string `json:"-"`
	CX         string `json:"-"`
	Locale     string `json:"-"`
}

type ConfigPublic struct {
	Configured  bool             `json:"configured"`
	ProviderID  string           `json:"provider_id,omitempty"`
	Provider    string           `json:"provider,omitempty"`
	Missing     []string         `json:"missing,omitempty"`
	Options     []ProviderOption `json:"options"`
	StorageMode string           `json:"storage_mode"`
}

type SearchRequest struct {
	Query    string `json:"query"`
	Limit    int    `json:"limit,omitempty"`
	UILocale string `json:"ui_locale,omitempty"`
}

type SearchOutcome struct {
	Query      string         `json:"query"`
	Provider   string         `json:"provider"`
	ProviderID string         `json:"provider_id"`
	Results    []SearchResult `json:"results"`
	NoResults  bool           `json:"no_results,omitempty"`
}

type SearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet,omitempty"`
	Source  string  `json:"source,omitempty"`
	Score   float64 `json:"score,omitempty"`
}

type Service struct {
	client *http.Client
}

func NewService() *Service {
	return NewServiceWithClient(nil)
}

func NewServiceWithClient(client *http.Client) *Service {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Service{client: client}
}

func ProviderOptions() []ProviderOption {
	return []ProviderOption{
		{
			ID:                ProviderTavily,
			Name:              "Tavily",
			Fields:            []string{"api_key"},
			DocsURL:           "https://docs.tavily.com/documentation/api-reference/endpoint/search",
			FreeTierHint:      "AI 研究，含免費額度",
			BestFor:           "AI 助理、研究整理、文件搜尋",
			SetupGuide:        []string{"在 Tavily 建立 API key。", "把 API key 貼到下方欄位。", "按下儲存後即可開始網路搜尋。"},
			UsageHint:         "例如：網路搜尋 OpenAI API 最新文件",
			APIKeyPlaceholder: "tvly-...",
			PreferredOrder:    1,
		},
		{
			ID:                ProviderExa,
			Name:              "Exa",
			Fields:            []string{"api_key"},
			DocsURL:           "https://exa.ai/docs/reference/search",
			FreeTierHint:      "研究文檔，20k/月免費",
			BestFor:           "技術文件、文章、公司資料、研究型搜尋",
			SetupGuide:        []string{"到 Exa dashboard 產生 API key。", "把 API key 貼到下方欄位。", "按下儲存後即可開始網路搜尋。"},
			UsageHint:         "例如：網路搜尋 最新 LLM 論文",
			APIKeyPlaceholder: "Exa API key",
			PreferredOrder:    2,
		},
		{
			ID:                ProviderSerpAPI,
			Name:              "SerpApi",
			Fields:            []string{"api_key"},
			DocsURL:           "https://serpapi.com/search-api",
			FreeTierHint:      "Google 結果，250/月免費",
			BestFor:           "想拿 Google 風格搜尋結果資料",
			SetupGuide:        []string{"到 SerpApi 產生 private API key。", "把 API key 貼到下方欄位。", "按下儲存後即可開始網路搜尋。"},
			UsageHint:         "例如：網路搜尋 台北 天氣",
			APIKeyPlaceholder: "SerpApi API key",
			PreferredOrder:    3,
		},
		{
			ID:                ProviderBrave,
			Name:              "Brave",
			Fields:            []string{"api_key"},
			DocsURL:           "https://api-dashboard.search.brave.com/documentation/guides/authentication",
			FreeTierHint:      "一般網頁，需 API token",
			BestFor:           "通用網頁搜尋與一般資訊查找",
			SetupGuide:        []string{"在 Brave Search API 建立 subscription token。", "把 token 貼到下方欄位。", "按下儲存後即可開始網路搜尋。"},
			UsageHint:         "例如：網路搜尋 AI agent search API",
			APIKeyPlaceholder: "Brave subscription token",
			PreferredOrder:    4,
		},
	}
}

func ProviderName(providerID string) string {
	for _, option := range ProviderOptions() {
		if option.ID == providerID {
			return option.Name
		}
	}
	return strings.TrimSpace(providerID)
}

func RequiredFields(providerID string) []string {
	for _, option := range ProviderOptions() {
		if option.ID == providerID {
			return option.Fields
		}
	}
	return nil
}

func NormalizeProviderID(providerID string) string {
	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case ProviderTavily, "tavily_search":
		return ProviderTavily
	case ProviderBrave, "brave_search":
		return ProviderBrave
	case ProviderExa, "exa_search":
		return ProviderExa
	case ProviderSerpAPI, "serp_api":
		return ProviderSerpAPI
	default:
		return strings.ToLower(strings.TrimSpace(providerID))
	}
}

func (s *Service) Search(ctx context.Context, req SearchRequest, cfg ProviderConfig) (SearchOutcome, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return SearchOutcome{}, ErrEmptyQuery
	}
	limit := normalizedLimit(req.Limit)
	cfg.ProviderID = NormalizeProviderID(cfg.ProviderID)
	var outcome SearchOutcome
	var err error
	switch cfg.ProviderID {
	case ProviderTavily:
		outcome, err = s.searchTavily(ctx, req.Query, limit, cfg)
	case ProviderBrave:
		outcome, err = s.searchBrave(ctx, req.Query, limit, cfg)
	case ProviderExa:
		outcome, err = s.searchExa(ctx, req.Query, limit, cfg)
	case ProviderSerpAPI:
		outcome, err = s.searchSerpAPI(ctx, req.Query, limit, cfg)
	default:
		return SearchOutcome{}, ErrProviderMissing
	}
	if err == nil {
		rankByLocalePreference(req.UILocale, outcome.Results)
		// 依查詢語言加權：繁中→台灣(.tw/gov.tw)，英文→美/英(gov/edu/ac.uk)；清單外仍保留在後當備援。
		rankByAuthority(req.Query, outcome.Results)
	}
	return outcome, err
}

// queryLanguage 粗判查詢語言：含中日韓表意字 → zh，其餘視為 en。
func queryLanguage(q string) string {
	for _, r := range q {
		if r >= 0x4E00 && r <= 0x9FFF {
			return "zh"
		}
	}
	return "en"
}

// authorityTierFor 依語言回傳權威層級（越小越優先）。
// 繁中查詢以台灣為主；英文查詢以美國/英國官方為主。
func authorityTierFor(lang, rawURL string) int {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return 3
	}
	host := strings.ToLower(u.Hostname())
	if lang == "zh" {
		switch {
		case strings.HasSuffix(host, ".gov.tw") || strings.HasSuffix(host, ".edu.tw"):
			return 0
		case strings.HasSuffix(host, ".tw") || strings.Contains(host, ".tw."):
			return 1
		default:
			return 2
		}
	}
	// 英文：美/英官方、學術。
	switch {
	case strings.HasSuffix(host, ".gov") || strings.HasSuffix(host, ".mil") ||
		strings.HasSuffix(host, ".edu") || strings.HasSuffix(host, ".gov.uk") ||
		strings.HasSuffix(host, ".ac.uk"):
		return 0
	case strings.HasSuffix(host, ".us") || strings.HasSuffix(host, ".uk"):
		return 1
	default:
		return 2
	}
}

// rankByAuthority 穩定排序：依查詢語言把對應地區的權威來源排到最前，其餘維持原序。
func rankByAuthority(query string, results []SearchResult) {
	lang := queryLanguage(query)
	sort.SliceStable(results, func(i, j int) bool {
		return authorityTierFor(lang, results[i].URL) < authorityTierFor(lang, results[j].URL)
	})
}

func rankByLocalePreference(locale string, results []SearchResult) {
	if localePreferenceFamily(locale) == "" || len(results) < 2 {
		return
	}
	sort.SliceStable(results, func(i, j int) bool {
		left := resultMatchesLocale(locale, results[i])
		right := resultMatchesLocale(locale, results[j])
		if left == right {
			return false
		}
		return left && !right
	})
}

func localePreferenceFamily(locale string) string {
	value := strings.ToLower(strings.TrimSpace(locale))
	switch {
	case strings.HasPrefix(value, "zh"):
		return "zh"
	case strings.HasPrefix(value, "ja"):
		return "ja"
	case strings.HasPrefix(value, "en"):
		return "en"
	default:
		return ""
	}
}

func resultMatchesLocale(locale string, result SearchResult) bool {
	text := strings.TrimSpace(result.Title + "\n" + result.Snippet)
	switch localePreferenceFamily(locale) {
	case "zh":
		return containsHan(text)
	case "ja":
		return containsJapanese(text)
	case "en":
		return mostlyASCII(text)
	default:
		return false
	}
}

func containsHan(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

func containsJapanese(text string) bool {
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
			return true
		}
	}
	return false
}

func mostlyASCII(text string) bool {
	ascii := 0
	printable := 0
	for _, r := range text {
		if r <= 32 {
			continue
		}
		printable++
		if r <= 0x7F {
			ascii++
		}
	}
	return printable > 0 && ascii*100/printable >= 80
}

func applyLanguageHeaders(httpReq *http.Request, locale string) {
	if value := acceptLanguageValue(locale); value != "" {
		httpReq.Header.Set("Accept-Language", value)
	}
}

func acceptLanguageValue(locale string) string {
	switch localePreferenceFamily(locale) {
	case "zh":
		return "zh-TW,zh-Hant;q=0.95,zh;q=0.9,en;q=0.6"
	case "ja":
		return "ja-JP,ja;q=0.95,en;q=0.6"
	case "en":
		return "en-US,en;q=0.9"
	default:
		return ""
	}
}

func serpHL(locale string) string {
	switch localePreferenceFamily(locale) {
	case "zh":
		return "zh-tw"
	case "ja":
		return "ja"
	case "en":
		return "en"
	default:
		return ""
	}
}

func serpGL(locale string) string {
	switch localePreferenceFamily(locale) {
	case "zh":
		return "tw"
	case "ja":
		return "jp"
	case "en":
		return "us"
	default:
		return ""
	}
}

func (s *Service) searchTavily(ctx context.Context, query string, limit int, cfg ProviderConfig) (SearchOutcome, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return SearchOutcome{}, ErrCredentialMissing
	}
	body, err := json.Marshal(map[string]interface{}{
		"query":          query,
		"search_depth":   "basic",
		"max_results":    limit,
		"include_answer": false,
	})
	if err != nil {
		return SearchOutcome{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return SearchOutcome{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "AI-Console-WebSearch/1.0")
	applyLanguageHeaders(httpReq, cfg.Locale)
	raw, err := s.doJSON(httpReq)
	if err != nil {
		return SearchOutcome{}, err
	}
	var payload struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return SearchOutcome{}, fmt.Errorf("websearch: decode tavily response: %w", err)
	}
	var results []SearchResult
	for _, item := range payload.Results {
		results = appendResult(results, SearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Content,
			Score:   item.Score,
		}, limit)
	}
	return outcomeOrNoResults(query, ProviderTavily, ProviderName(ProviderTavily), results)
}

func (s *Service) searchBrave(ctx context.Context, query string, limit int, cfg ProviderConfig) (SearchOutcome, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return SearchOutcome{}, ErrCredentialMissing
	}
	u, _ := url.Parse("https://api.search.brave.com/res/v1/web/search")
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return SearchOutcome{}, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Subscription-Token", strings.TrimSpace(cfg.APIKey))
	httpReq.Header.Set("User-Agent", "AI-Console-WebSearch/1.0")
	applyLanguageHeaders(httpReq, cfg.Locale)
	raw, err := s.doJSON(httpReq)
	if err != nil {
		return SearchOutcome{}, err
	}
	var payload struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
				Profile     struct {
					Name string `json:"name"`
				} `json:"profile"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return SearchOutcome{}, fmt.Errorf("websearch: decode brave response: %w", err)
	}
	var results []SearchResult
	for _, item := range payload.Web.Results {
		results = appendResult(results, SearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Description,
			Source:  item.Profile.Name,
		}, limit)
	}
	return outcomeOrNoResults(query, ProviderBrave, ProviderName(ProviderBrave), results)
}

func (s *Service) searchExa(ctx context.Context, query string, limit int, cfg ProviderConfig) (SearchOutcome, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return SearchOutcome{}, ErrCredentialMissing
	}
	body, err := json.Marshal(map[string]interface{}{
		"query":      query,
		"numResults": limit,
		"contents": map[string]bool{
			"highlights": true,
			"text":       true,
		},
	})
	if err != nil {
		return SearchOutcome{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.exa.ai/search", bytes.NewReader(body))
	if err != nil {
		return SearchOutcome{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "AI-Console-WebSearch/1.0")
	httpReq.Header.Set("x-api-key", strings.TrimSpace(cfg.APIKey))
	applyLanguageHeaders(httpReq, cfg.Locale)
	raw, err := s.doJSON(httpReq)
	if err != nil {
		return SearchOutcome{}, err
	}
	var payload struct {
		Results []struct {
			Title      string   `json:"title"`
			URL        string   `json:"url"`
			Text       string   `json:"text"`
			Summary    string   `json:"summary"`
			Highlights []string `json:"highlights"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return SearchOutcome{}, fmt.Errorf("websearch: decode exa response: %w", err)
	}
	var results []SearchResult
	for _, item := range payload.Results {
		results = appendResult(results, SearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: firstNonEmpty(strings.Join(item.Highlights, " "), item.Summary, item.Text),
		}, limit)
	}
	return outcomeOrNoResults(query, ProviderExa, ProviderName(ProviderExa), results)
}

func (s *Service) searchSerpAPI(ctx context.Context, query string, limit int, cfg ProviderConfig) (SearchOutcome, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return SearchOutcome{}, ErrCredentialMissing
	}
	u, _ := url.Parse("https://serpapi.com/search.json")
	q := u.Query()
	q.Set("engine", "google")
	q.Set("q", query)
	q.Set("api_key", strings.TrimSpace(cfg.APIKey))
	q.Set("output", "json")
	if hl := serpHL(cfg.Locale); hl != "" {
		q.Set("hl", hl)
	}
	if gl := serpGL(cfg.Locale); gl != "" {
		q.Set("gl", gl)
	}
	u.RawQuery = q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return SearchOutcome{}, err
	}
	httpReq.Header.Set("User-Agent", "AI-Console-WebSearch/1.0")
	applyLanguageHeaders(httpReq, cfg.Locale)
	raw, err := s.doJSON(httpReq)
	if err != nil {
		return SearchOutcome{}, err
	}
	var payload struct {
		Error          string `json:"error"`
		OrganicResults []struct {
			Title         string `json:"title"`
			Link          string `json:"link"`
			Snippet       string `json:"snippet"`
			DisplayedLink string `json:"displayed_link"`
		} `json:"organic_results"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return SearchOutcome{}, fmt.Errorf("websearch: decode serpapi response: %w", err)
	}
	if strings.TrimSpace(payload.Error) != "" {
		return SearchOutcome{}, fmt.Errorf("websearch: serpapi error: %s", strings.TrimSpace(payload.Error))
	}
	var results []SearchResult
	for _, item := range payload.OrganicResults {
		results = appendResult(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
			Source:  item.DisplayedLink,
		}, limit)
	}
	return outcomeOrNoResults(query, ProviderSerpAPI, ProviderName(ProviderSerpAPI), results)
}

func (s *Service) doJSON(req *http.Request) ([]byte, error) {
	res, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("websearch: request failed: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := cleanText(string(raw))
		if msg != "" {
			msg = ": " + truncateRunes(msg, 180)
		}
		return nil, fmt.Errorf("websearch: provider HTTP %d%s", res.StatusCode, msg)
	}
	return raw, nil
}

func FormatSearchOutcome(req SearchRequest, outcome SearchOutcome) string {
	query := strings.TrimSpace(firstNonEmpty(outcome.Query, req.Query))
	provider := firstNonEmpty(outcome.Provider, ProviderName(outcome.ProviderID), "Web Search")
	if len(outcome.Results) == 0 {
		return fmt.Sprintf("Web search (%s) found no usable results for %q.", provider, query)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Web search (%s) found %d result(s) for %q:", provider, len(outcome.Results), query)
	for i, result := range outcome.Results {
		fmt.Fprintf(&b, "\n\n%d. %s", i+1, cleanText(result.Title))
		if result.Snippet != "" {
			fmt.Fprintf(&b, "\n%s", truncateRunes(cleanText(result.Snippet), maxSnippetRunes))
		}
		if result.URL != "" {
			fmt.Fprintf(&b, "\n%s", result.URL)
		}
	}
	out := b.String()
	if utf8.RuneCountInString(out) > maxOutputRunes {
		out = string([]rune(out)[:maxOutputRunes]) + "\n\nResults truncated."
	}
	return out
}

func EmptyQueryMessage() string {
	return "Please provide a web search query."
}

func MissingConfigMessage() string {
	return "Web search is not configured yet. Choose Tavily, Exa, SerpApi, or Brave and enter the API key."
}

func ParseUserQuery(text string) (SearchRequest, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return SearchRequest{}, false
	}
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"web_search", "search_web", "web search", "search web"} {
		if target, ok := cutFoldedCommand(trimmed, lower, prefix); ok {
			return SearchRequest{Query: target, Limit: DefaultLimit}, true
		}
	}
	for _, prefix := range []string{
		"\u7db2\u8def\u641c\u5c0b",
		"\u7db2\u8def\u67e5\u8a62",
		"\u641c\u5c0b\u7db2\u8def",
		"\u67e5\u8a62\u7db2\u8def",
		"\u67e5\u7db2\u8def",
		"\u4e0a\u7db2\u67e5",
		"\u4e0a\u7db2\u67e5\u8a62",
		"\u7db2\u8def",
	} {
		if target, ok := cutCommand(trimmed, prefix); ok {
			return SearchRequest{Query: target, Limit: DefaultLimit}, true
		}
	}
	return SearchRequest{}, false
}

func RequestFromAction(action, target string) (SearchRequest, bool) {
	if !IsSearchAction(action) {
		return SearchRequest{}, false
	}
	return SearchRequest{Query: strings.TrimSpace(target), Limit: DefaultLimit}, true
}

func IsSearchAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "web_search", "search_web", "web search", "search web",
		"\u7db2\u8def",
		"\u7db2\u8def\u641c\u5c0b",
		"\u7db2\u8def\u67e5\u8a62",
		"\u641c\u5c0b\u7db2\u8def",
		"\u67e5\u8a62\u7db2\u8def",
		"\u67e5\u7db2\u8def",
		"\u4e0a\u7db2\u67e5",
		"\u4e0a\u7db2\u67e5\u8a62":
		return true
	default:
		return false
	}
}

func appendResult(results []SearchResult, result SearchResult, limit int) []SearchResult {
	if len(results) >= limit {
		return results
	}
	result.Title = cleanText(result.Title)
	result.Snippet = truncateRunes(cleanText(result.Snippet), maxSnippetRunes)
	result.URL = strings.TrimSpace(result.URL)
	if result.Title == "" && result.Snippet == "" {
		return results
	}
	if result.Title == "" {
		result.Title = firstSentence(result.Snippet)
	}
	if !isHTTPURL(result.URL) {
		result.URL = ""
	}
	if result.Source == "" {
		result.Source = hostOf(result.URL)
	}
	for _, existing := range results {
		if existing.URL != "" && existing.URL == result.URL {
			return results
		}
	}
	return append(results, result)
}

func outcomeOrNoResults(query, providerID, provider string, results []SearchResult) (SearchOutcome, error) {
	outcome := SearchOutcome{Query: query, ProviderID: providerID, Provider: provider, Results: results}
	if len(results) == 0 {
		outcome.NoResults = true
		return outcome, ErrNoResults
	}
	return outcome, nil
}

func normalizedLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func cutCommand(text, prefix string) (string, bool) {
	if text == prefix {
		return "", true
	}
	if utf8.RuneCountInString(prefix) > 2 && strings.HasPrefix(text, prefix) {
		target := strings.TrimSpace(strings.TrimPrefix(text, prefix))
		if target != "" {
			return target, true
		}
	}
	for _, sep := range []string{" ", "\uff1a", ":", "\uff0c", ",", "\u310c"} {
		if strings.HasPrefix(text, prefix+sep) {
			return strings.TrimSpace(strings.TrimPrefix(text, prefix+sep)), true
		}
	}
	return "", false
}

func cutFoldedCommand(original, folded, prefix string) (string, bool) {
	if folded == prefix {
		return "", true
	}
	for _, sep := range []string{" ", "\uff1a", ":", "\uff0c", ",", "\u310c"} {
		needle := prefix + sep
		if strings.HasPrefix(folded, needle) {
			return strings.TrimSpace(original[len(needle):]), true
		}
	}
	return "", false
}

func cleanText(text string) string {
	text = tagPattern.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	return strings.Join(strings.Fields(text), " ")
}

func firstSentence(text string) string {
	text = cleanText(text)
	if text == "" {
		return ""
	}
	for _, sep := range []string{". ", "\u3002", "\uff1f", "?", "\uff01", "!"} {
		if idx := strings.Index(text, sep); idx > 0 {
			return strings.TrimSpace(text[:idx+len(strings.TrimSpace(sep))])
		}
	}
	return truncateRunes(text, 80)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func truncateRunes(text string, max int) string {
	if max <= 0 || utf8.RuneCountInString(text) <= max {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:max])) + "..."
}

func isHTTPURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func hostOf(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return u.Hostname()
}
