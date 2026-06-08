// Package localsearch searches only AI Console managed local data.
package localsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	DefaultLimit           = 5
	defaultMaxFileSize     = 1024 * 1024
	defaultMaxFiles        = 2000
	defaultMaxBytesScanned = 8 * 1024 * 1024
	defaultCacheTTL        = 60 * time.Second
	maxSnippetRunes        = 100
	maxResponseRunes       = 3600
)

var (
	ErrEmptyQuery        = errors.New("localsearch: empty query")
	ErrOutsideSearchRoot = errors.New("localsearch: outside search root")
	errStopSearch        = errors.New("localsearch: stop search")
	secretPatterns       = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9]{20,}`),
		regexp.MustCompile(`(?i)\bsk-ant-[A-Za-z0-9\-]{20,}`),
		regexp.MustCompile(`(?i)\bsk-or-v1-[A-Za-z0-9]{20,}`),
		regexp.MustCompile(`\bgh[ps]_[A-Za-z0-9]{36,}`),
		regexp.MustCompile(`\bAIza[A-Za-z0-9\-_]{35}`),
		regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`),
	}
	scopeDirectivePattern = regexp.MustCompile(`(?i)\bscope\s*=\s*[\w,\-]+`)
)

var rootIndexCache = struct {
	sync.Mutex
	entries map[string]cachedRootIndex
}{entries: make(map[string]cachedRootIndex)}

type SearchRequest struct {
	Query string   `json:"query"`
	Scope []string `json:"scope,omitempty"`
	Limit int      `json:"limit,omitempty"`
}

type SearchOutcome struct {
	Results      []SearchResult `json:"results"`
	Incomplete   bool           `json:"incomplete,omitempty"`
	Reason       string         `json:"reason,omitempty"`
	FilesScanned int            `json:"files_scanned,omitempty"`
	BytesScanned int64          `json:"bytes_scanned,omitempty"`
}

type SearchResult struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Path    string `json:"path,omitempty"`
	Score   int    `json:"score"`
}

type Root struct {
	Path   string
	Source string
}

type Item struct {
	Source  string
	Title   string
	Path    string
	Content string
}

type Service struct {
	roots           []Root
	items           []Item
	maxFileSize     int64
	maxFiles        int
	maxBytesScanned int64
	cacheTTL        time.Duration
}

func NewService(roots []Root, items []Item) *Service {
	return &Service{
		roots:           roots,
		items:           items,
		maxFileSize:     defaultMaxFileSize,
		maxFiles:        defaultMaxFiles,
		maxBytesScanned: defaultMaxBytesScanned,
		cacheTTL:        defaultCacheTTL,
	}
}

func (s *Service) Search(req SearchRequest) ([]SearchResult, error) {
	outcome, err := s.SearchWithContext(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return outcome.Results, nil
}

func (s *Service) SearchWithContext(ctx context.Context, req SearchRequest) (SearchOutcome, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return SearchOutcome{}, ErrEmptyQuery
	}
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	scope := normalizeScope(req.Scope)
	var candidates []SearchResult
	for _, item := range s.items {
		if ctx.Err() != nil {
			return SearchOutcome{Results: candidates, Incomplete: true, Reason: "timeout"}, nil
		}
		if !scopeAllows(scope, item.Source) {
			continue
		}
		if result, ok := matchItem(req.Query, item); ok {
			candidates = append(candidates, result)
		}
	}
	fileOutcome, err := s.searchRoots(ctx, req.Query, scope)
	if err != nil {
		return SearchOutcome{}, err
	}
	candidates = append(candidates, fileOutcome.Results...)
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Title < candidates[j].Title
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	fileOutcome.Results = candidates
	return fileOutcome, nil
}

func (s *Service) searchRoots(ctx context.Context, query string, scope map[string]bool) (SearchOutcome, error) {
	outcome := SearchOutcome{}
	for _, root := range s.roots {
		cleanRoot, err := secureRoot(root.Path)
		if err != nil {
			continue
		}
		source := strings.TrimSpace(root.Source)
		if source == "" {
			source = sourceFromPath(cleanRoot)
		}
		if !scopeAllows(scope, source) {
			continue
		}
		index, err := s.indexRoot(ctx, cleanRoot, source)
		if err != nil {
			return SearchOutcome{}, err
		}
		outcome.FilesScanned += index.filesScanned
		outcome.BytesScanned += index.bytesScanned
		if index.incomplete {
			outcome.Incomplete = true
			outcome.Reason = index.reason
		}
		for _, item := range index.items {
			if ctx.Err() != nil {
				outcome.Incomplete = true
				outcome.Reason = "timeout"
				return outcome, nil
			}
			if !scopeAllows(scope, item.Source) {
				continue
			}
			if result, ok := matchItem(query, item); ok {
				outcome.Results = append(outcome.Results, result)
			}
		}
	}
	return outcome, nil
}

type cachedRootIndex struct {
	items        []Item
	expiresAt    time.Time
	incomplete   bool
	reason       string
	filesScanned int
	bytesScanned int64
}

func (s *Service) indexRoot(ctx context.Context, cleanRoot, source string) (cachedRootIndex, error) {
	key := fmt.Sprintf("%s|%s|%d|%d|%d", cleanRoot, source, s.maxFileSize, s.maxFiles, s.maxBytesScanned)
	now := time.Now()
	rootIndexCache.Lock()
	if cached, ok := rootIndexCache.entries[key]; ok && now.Before(cached.expiresAt) {
		rootIndexCache.Unlock()
		return cached, nil
	}
	rootIndexCache.Unlock()

	index := cachedRootIndex{expiresAt: now.Add(s.cacheTTL)}
	err := filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if ctx.Err() != nil {
			index.incomplete = true
			index.reason = "timeout"
			return errStopSearch
		}
		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) && path != cleanRoot {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 || info.Size() > s.maxFileSize || !isTextSearchable(path) {
			return nil
		}
		if index.filesScanned >= s.maxFiles || index.bytesScanned+info.Size() > s.maxBytesScanned {
			// Stop scanning before a large local corpus can block the chat UI.
			index.incomplete = true
			index.reason = "scan limit"
			return errStopSearch
		}
		if !isInside(cleanRoot, path) {
			// Keep searches inside AI Console data roots.
			return ErrOutsideSearchRoot
		}
		data, err := os.ReadFile(path)
		index.filesScanned++
		index.bytesScanned += info.Size()
		if err != nil || !utf8.Valid(data) {
			return nil
		}
		index.items = append(index.items, Item{
			Source:  sourceFromPathWithFallback(path, source),
			Title:   filepath.Base(path),
			Path:    path,
			Content: string(data),
		})
		return nil
	})
	if errors.Is(err, errStopSearch) {
		err = nil
	}
	if err != nil {
		return cachedRootIndex{}, err
	}
	rootIndexCache.Lock()
	rootIndexCache.entries[key] = index
	rootIndexCache.Unlock()
	return index, nil
}

func matchItem(query string, item Item) (SearchResult, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return SearchResult{}, false
	}
	haystack := strings.ToLower(item.Title + "\n" + item.Path + "\n" + item.Content)
	if !strings.Contains(haystack, q) {
		return SearchResult{}, false
	}
	score := 10
	titleLower := strings.ToLower(item.Title)
	pathLower := strings.ToLower(item.Path)
	if strings.Contains(titleLower, q) {
		// Prefer title/path hits because users often remember filenames.
		score += 45
	}
	if strings.Contains(pathLower, q) {
		score += 25
	}
	if item.Source == "tool" || item.Source == "skill" {
		score += 15
	}
	snippet := buildSnippet(item.Content, query)
	if snippet == "" {
		snippet = item.Title
	}
	// Redact before rendering snippets into chat bubbles.
	snippet = Redact(snippet)
	return SearchResult{
		Source:  item.Source,
		Title:   item.Title,
		Snippet: snippet,
		Path:    item.Path,
		Score:   score,
	}, true
}

func FormatChatResponse(req SearchRequest, results []SearchResult) string {
	return FormatSearchOutcome(req, SearchOutcome{Results: results})
}

func FormatSearchOutcome(req SearchRequest, outcome SearchOutcome) string {
	query := strings.TrimSpace(req.Query)
	if len(outcome.Results) == 0 {
		out := fmt.Sprintf("本機搜尋「%s」沒有找到結果。\n可以縮短關鍵字，或指定範圍：記憶、文件、紀錄、trace、工具。", query)
		return appendIncompleteNotice(out, outcome)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("本機搜尋「%s」有找到 %d 筆：", query, len(outcome.Results)))
	for i, result := range outcome.Results {
		b.WriteString(fmt.Sprintf("\n\n%d. ", i+1))
		if result.Snippet != "" {
			b.WriteString("內容：")
			b.WriteString(result.Snippet)
		}
	}
	out := appendIncompleteNotice(b.String(), outcome)
	if utf8.RuneCountInString(out) > maxResponseRunes {
		runes := []rune(out)
		out = string(runes[:maxResponseRunes]) + "\n\n結果過長，已截斷。"
	}
	return out
}

func appendIncompleteNotice(out string, outcome SearchOutcome) string {
	if !outcome.Incomplete {
		return out
	}
	reason := strings.TrimSpace(outcome.Reason)
	if reason == "" {
		reason = "已達搜尋保護上限"
	}
	return out + "\n\n提示：搜尋可能不完整（" + reason + "）。"
}

func EmptyQueryMessage() string {
	return "想搜尋什麼？請輸入關鍵字，也可以指定範圍：記憶、文件、紀錄、trace、工具。"
}

func ParseUserQuery(text string) (SearchRequest, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return SearchRequest{}, false
	}
	for _, prefix := range []string{"搜尋", "查找", "查詢"} {
		if target, ok := cutCommand(trimmed, prefix); ok {
			return requestFromTarget(target, true), true
		}
	}
	for _, prefix := range []string{"search", "find", "query"} {
		if target, ok := cutColonCommand(trimmed, prefix); ok {
			return requestFromTarget(target, true), true
		}
	}
	return SearchRequest{}, false
}

func IsSearchAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "本機搜尋", "搜尋", "查找", "查詢", "search", "find", "query":
		return true
	default:
		return false
	}
}

func RequestFromAction(action, target string) (SearchRequest, bool) {
	if !IsSearchAction(action) {
		return SearchRequest{}, false
	}
	return requestFromTarget(target, true), true
}

func Redact(text string) string {
	out := text
	for _, pattern := range secretPatterns {
		out = pattern.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}

func requestFromTarget(target string, allowAll bool) SearchRequest {
	scopes := scopesFromText(target)
	query := stripScopeWords(target)
	if query == "" {
		query = target
	}
	if allowAll && len(scopes) == 0 {
		scopes = []string{"all"}
	}
	return SearchRequest{Query: query, Scope: scopes, Limit: DefaultLimit}
}

func cutCommand(text, prefix string) (string, bool) {
	if text == prefix {
		return "", true
	}
	if strings.HasPrefix(text, prefix+" ") || strings.HasPrefix(text, prefix+"：") || strings.HasPrefix(text, prefix+":") {
		return strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(text, prefix), " ：:")), true
	}
	return "", false
}

func cutColonCommand(text, prefix string) (string, bool) {
	lower := strings.ToLower(text)
	marker := prefix + ":"
	if !strings.HasPrefix(lower, marker) {
		return "", false
	}
	return strings.TrimSpace(text[len(marker):]), true
}

func scopesFromText(text string) []string {
	lower := strings.ToLower(text)
	var scopes []string
	add := func(scope string) {
		for _, existing := range scopes {
			if existing == scope {
				return
			}
		}
		scopes = append(scopes, scope)
	}
	if strings.Contains(lower, "記憶") || strings.Contains(lower, "memory") || strings.Contains(lower, "聊天") {
		add("memory")
	}
	if strings.Contains(lower, "文件") || strings.Contains(lower, "document") || strings.Contains(lower, "檔案") {
		add("document")
	}
	if strings.Contains(lower, "紀錄") || strings.Contains(lower, "記錄") || strings.Contains(lower, "trace") || strings.Contains(lower, "監視") {
		add("trace")
	}
	if strings.Contains(lower, "工具") || strings.Contains(lower, "tool") {
		add("tool")
	}
	if strings.Contains(lower, "skill") || strings.Contains(lower, "技能") {
		add("skill")
	}
	return scopes
}

func stripScopeWords(text string) string {
	text = scopeDirectivePattern.ReplaceAllString(text, "")
	replacer := strings.NewReplacer(
		"記憶", "", "文件", "", "紀錄", "", "記錄", "", "trace", "", "Trace", "", "TRACE", "",
		"監視", "", "工具", "", "技能", "", "document", "", "Document", "", "memory", "", "Memory", "",
		"tool", "", "Tool", "", "skill", "", "Skill", "",
		// NOTE: only strip multi-char locatives ("…中的"/"…裡的"). Never strip bare
		// "中"/"裡": that corrupts real terms like 台中 / 中壢 (target→query).
		"裡的", "", "中的", "",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func normalizeScope(scopes []string) map[string]bool {
	normalized := make(map[string]bool)
	if len(scopes) == 0 {
		normalized["all"] = true
		return normalized
	}
	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		if scope == "" {
			continue
		}
		normalized[scope] = true
	}
	if len(normalized) == 0 {
		normalized["all"] = true
	}
	return normalized
}

func scopeAllows(scopes map[string]bool, source string) bool {
	source = strings.ToLower(source)
	if scopes["all"] {
		// Trace/debug 為雜訊；memory/talk（對話）當前已是模型 context，預設不搜，
		// 避免搜尋命中使用者自己剛打的訊息（自我回音）。明確指定「記憶/對話」才搜。
		return source != "trace" && source != "debug" && source != "memory" && source != "talk"
	}
	if scopes[source] {
		return true
	}
	if source == "talk" && scopes["memory"] {
		return true
	}
	if source == "debug" && scopes["trace"] {
		return true
	}
	return false
}

func buildSnippet(content, query string) string {
	content = strings.TrimSpace(compactJSON(content))
	if content == "" {
		return ""
	}
	lower := strings.ToLower(content)
	q := strings.ToLower(strings.TrimSpace(query))
	idx := strings.Index(lower, q)
	runes := []rune(content)
	if idx < 0 {
		if len(runes) > maxSnippetRunes {
			return string(runes[:maxSnippetRunes]) + "..."
		}
		return content
	}
	prefixRunes := utf8.RuneCountInString(content[:idx])
	queryRunes := utf8.RuneCountInString(query)
	start := prefixRunes
	if queryRunes < maxSnippetRunes {
		start = prefixRunes - ((maxSnippetRunes - queryRunes) / 2)
	}
	if start < 0 {
		start = 0
	}
	end := start + maxSnippetRunes
	if end > len(runes) {
		end = len(runes)
	}
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func compactJSON(content string) string {
	var value interface{}
	if json.Unmarshal([]byte(content), &value) != nil {
		return content
	}
	data, err := json.Marshal(value)
	if err != nil {
		return content
	}
	return string(data)
}

func secureRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", os.ErrNotExist
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return abs, nil
	}
	return resolved, nil
}

func isInside(root, path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, abs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func shouldSkipDir(name string) bool {
	switch name {
	// TASK 31 / Phase 0.2：skill_eval 評量資料絕不可進入 LLM-facing 搜尋語料（不變式 2）。
	case "secrets", "cache", "tmp", "node_modules", ".git", "cli-workspaces", "skill_eval":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func isTextSearchable(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".json", ".jsonl", ".log", ".csv", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func sourceFromPathWithFallback(path, fallback string) string {
	source := sourceFromPath(path)
	if source == "file" {
		return fallback
	}
	return source
}

func sourceFromPath(path string) string {
	lower := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lower, "/memory/") || strings.Contains(lower, "talk_full"):
		return "memory"
	case strings.Contains(lower, "/documents/") || strings.Contains(lower, "/references/"):
		return "document"
	case strings.Contains(lower, "/debug/") || strings.Contains(lower, "trace"):
		return "trace"
	case strings.Contains(lower, "/tools/"):
		return "tool"
	case strings.Contains(lower, "/skills/"):
		return "skill"
	default:
		return "file"
	}
}

func sourceLabel(source string) string {
	switch source {
	case "memory", "talk":
		return "記憶"
	case "document":
		return "文件"
	case "trace", "debug":
		return "紀錄"
	case "tool":
		return "工具"
	case "skill":
		return "技能"
	default:
		return "本機"
	}
}
