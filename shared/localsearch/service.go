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

	"ui_console/builtin"
)

const (
	DefaultLimit           = 5
	defaultMaxFileSize     = 16 * 1024 * 1024
	defaultMaxFiles        = 2000
	defaultMaxBytesScanned = 32 * 1024 * 1024
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
	// AuxTerms 是「用法/做法」這類問法詞：不計入比對門檻、也不能單獨讓 item
	// 成立，只在主要關鍵詞已命中時微幅加分，協助排序。
	AuxTerms []string `json:"aux_terms,omitempty"`
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
	// Keywords 只參與比對，不會出現在回傳給使用者的 Snippet。
	// 用來塞同義詞／別名（如「剛剛 拖進來 已載入」）讓查詢命中，
	// 但不污染顯示內容。
	Keywords string
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
		if result, ok := matchItem(req.Query, req.AuxTerms, item); ok {
			candidates = append(candidates, result)
		}
	}
	fileOutcome, err := s.searchRoots(ctx, req.Query, req.AuxTerms, scope)
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

func (s *Service) searchRoots(ctx context.Context, query string, auxTerms []string, scope map[string]bool) (SearchOutcome, error) {
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
		if !scopeAllows(scope, source) && !scopeMayAllowFileCategory(scope) {
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
			if result, ok := matchItem(query, auxTerms, item); ok {
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
		if info.Mode()&os.ModeSymlink != 0 || info.Size() > s.maxFileSize || !isIndexableFile(path) {
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
		index.filesScanned++
		index.bytesScanned += info.Size()
		source := sourceForFile(path, source)
		content, err := searchableContentForFile(path, source, info)
		if err != nil || strings.TrimSpace(content) == "" || !utf8.ValidString(content) {
			return nil
		}
		index.items = append(index.items, Item{
			Source:  source,
			Title:   filepath.Base(path),
			Path:    path,
			Content: content,
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

// titleTermHit 判斷查詢中是否有「夠長」的詞（≥2 runes，已過濾停用詞）直接出現在
// 標題。標題命中是高訊號，可作為 need 門檻之外的後備命中條件，避免描述性雜訊詞
// 把整筆結果擋掉。單一詞查詢由 queryMatchScore 的整串比對處理，這裡只補多詞情境。
func titleTermHit(q, titleLower string) bool {
	if titleLower == "" {
		return false
	}
	terms := queryTerms(q)
	if len(terms) == 0 {
		// 單詞查詢：queryTerms 回 nil，直接比對整串。
		return utf8.RuneCountInString(q) >= 2 && strings.Contains(titleLower, q)
	}
	for _, term := range terms {
		if utf8.RuneCountInString(term) >= 2 && strings.Contains(titleLower, term) {
			return true
		}
	}
	return false
}

func matchItem(query string, auxTerms []string, item Item) (SearchResult, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return SearchResult{}, false
	}
	haystack := strings.ToLower(item.Title + "\n" + item.Path + "\n" + item.Content + "\n" + item.Keywords)
	titleLower := strings.ToLower(item.Title)
	queryScore, matched := queryMatchScore(q, haystack, auxTerms)
	if !matched && titleTermHit(q, titleLower) {
		// 後備命中：只要有「夠長」的查詢詞直接命中標題，就視為命中，避免 judge
		// 附加的描述性雜訊詞（如「使用方式」）把 need 門檻卡死——明明標題就是該
		// skill／檔名，卻因為多了一個沒命中的詞而搜不到。
		queryScore = 30 + auxBonus(haystack, auxTerms)
		matched = true
	}
	if !matched {
		return SearchResult{}, false
	}
	score := 10 + queryScore
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

func queryMatchScore(q, haystack string, auxTerms []string) (int, bool) {
	if strings.Contains(haystack, q) {
		return 60 + auxBonus(haystack, auxTerms), true
	}
	terms := queryTerms(q)
	if len(terms) == 0 {
		return 0, false
	}
	hits := 0
	for _, term := range terms {
		if strings.Contains(haystack, term) {
			hits++
		}
	}
	need := len(terms)
	if need > 2 {
		need = 2
	}
	if hits < need {
		return 0, false
	}
	return hits*12 + auxBonus(haystack, auxTerms), true
}

// auxBonus 計算輔助詞（用法/做法等）的加權。只有在主要關鍵詞已成立、呼叫到
// 這裡時才會疊加：每命中一個 +4，總和上限 12。因此輔助詞永遠無法單獨讓一筆
// item 成立，只能在已命中的結果之間微調排序。
func auxBonus(haystack string, auxTerms []string) int {
	bonus := 0
	for _, aux := range auxTerms {
		aux = strings.ToLower(strings.TrimSpace(aux))
		if aux == "" {
			continue
		}
		if strings.Contains(haystack, aux) {
			bonus += 4
			if bonus >= 12 {
				return 12
			}
		}
	}
	return bonus
}

// auxHowToWords 是「用法/做法」這類問法詞。它們不該當主要搜尋關鍵詞（會稀釋
// 比對、卡住 need 門檻），但使用者原句若提到，仍可作輔助詞協助排序。
var auxHowToWords = []string{
	"要怎麼用", "怎麼用", "怎樣用", "如何用",
	"怎麼做", "如何做", "怎麼操作", "如何操作",
}

// AuxTermsFromText 從原始輸入挑出出現過的問法詞，回傳去重後的輔助搜尋詞，
// 供呼叫端塞進 SearchRequest.AuxTerms。
func AuxTermsFromText(text string) []string {
	lower := strings.ToLower(text)
	if lower == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, w := range auxHowToWords {
		if seen[w] {
			continue
		}
		if strings.Contains(lower, w) {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

func queryTerms(q string) []string {
	raw := strings.Fields(q)
	if len(raw) <= 1 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, term := range raw {
		term = strings.Trim(strings.ToLower(term), "。.!！?？,，、:：;；\"'`[]{}()（）")
		term = strings.TrimSuffix(term, "的")
		if term == "" || isQueryStopWord(term) || seen[term] {
			continue
		}
		seen[term] = true
		out = append(out, term)
	}
	return out
}

func isQueryStopWord(term string) bool {
	switch term {
	case "我", "你", "他", "她", "它", "我們", "你們", "看到", "看得到", "能", "可以", "有", "嗎", "嗎?", "嗎？", "了", "一下":
		return true
	default:
		return false
	}
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
		// skill 結果改以「名稱＋功能摘要」呈現，而不是回傳內部檔內容/雜湊。
		if result.Source == "skill" && strings.TrimSpace(result.Title) != "" {
			b.WriteString("技能：")
			b.WriteString(result.Title)
			if snip := strings.TrimSpace(result.Snippet); snip != "" && snip != result.Title {
				b.WriteString("\n   摘要：")
				b.WriteString(snip)
			}
			continue
		}
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
	if query == "" || isTrivialConnectorQuery(query) {
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
	if containsAny(lower, "文件", "文檔", "document", "documents", "docx", "xlsx", "pptx", "odt", "ods", "odp", "epub", "csv", "tsv", "md", "txt") {
		add("document")
	}
	if containsAny(lower, "圖片", "照片", "相片", "圖像", "影像", "image", "images", "photo", "photos", "picture", "pictures", "png", "jpg", "jpeg", "webp", "gif") {
		add("image")
	}
	if containsAny(lower, "影片", "視頻", "video", "videos", "movie", "movies", "mp4", "mov", "m4v", "webm", "mkv", "avi", "wmv") {
		add("video")
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
		"記憶", "", "文件", "", "文檔", "", "圖片", "", "照片", "", "相片", "", "圖像", "", "影像", "", "影片", "", "視頻", "",
		"紀錄", "", "記錄", "", "trace", "", "Trace", "", "TRACE", "",
		"監視", "", "工具", "", "技能", "", "document", "", "Document", "", "memory", "", "Memory", "",
		"image", "", "Image", "", "photo", "", "Photo", "", "picture", "", "Picture", "", "video", "", "Video", "",
		"tool", "", "Tool", "", "skill", "", "Skill", "",
		// NOTE: only strip multi-char locatives ("…中的"/"…裡的"). Never strip bare
		// "中"/"裡": that corrupts real terms like 台中 / 中壢 (target→query).
		"裡的", "", "中的", "",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func isTrivialConnectorQuery(query string) bool {
	trimmed := strings.TrimSpace(query)
	switch trimmed {
	case "", "和", "與", "跟", "及", "或", "and", "or", "/", "&", "+", "and/or":
		return true
	default:
		return false
	}
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

func scopeMayAllowFileCategory(scopes map[string]bool) bool {
	return scopes["image"] || scopes["video"] || scopes["document"]
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
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

func isIndexableFile(path string) bool {
	return builtin.IsSearchableFormat(path) || fileMediaSource(path) != ""
}

func searchableContentForFile(path, source string, info os.FileInfo) (string, error) {
	if builtin.IsSearchableFormat(path) {
		return builtin.ExtractSearchableText(path)
	}
	if source == "image" || source == "video" {
		return mediaMetadataContent(path, source, info), nil
	}
	return "", fmt.Errorf("localsearch: unsupported index format %q", filepath.Ext(path))
}

func mediaMetadataContent(path, source string, info os.FileInfo) string {
	name := filepath.Base(path)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	labels := "媒體 檔案 資料 本機資料"
	if source == "image" {
		labels = "圖片 照片 相片 圖像 影像 image photo picture " + labels
	} else if source == "video" {
		labels = "影片 視頻 video movie " + labels
	}
	modified := ""
	size := ""
	if info != nil {
		modified = info.ModTime().Format(time.RFC3339)
		size = fmt.Sprintf("%d", info.Size())
	}
	return strings.Join([]string{
		name,
		labels,
		"副檔名 " + ext,
		"修改時間 " + modified,
		"大小 " + size,
	}, "\n")
}

func sourceForFile(path, fallback string) string {
	if source := fileMediaSource(path); source != "" {
		return source
	}
	return sourceFromPathWithFallback(path, fallback)
}

func fileMediaSource(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if isImageExt(ext) {
		return "image"
	}
	if isVideoExt(ext) {
		return "video"
	}
	return ""
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tif", ".tiff", ".heic", ".heif":
		return true
	default:
		return false
	}
}

func isVideoExt(ext string) bool {
	switch ext {
	case ".mp4", ".mov", ".m4v", ".webm", ".mkv", ".avi", ".wmv", ".flv", ".mpg", ".mpeg", ".3gp", ".ogv":
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
	case strings.Contains(lower, "/images/"):
		return "image"
	case strings.Contains(lower, "/videos/"):
		return "video"
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
	case "image":
		return "圖片"
	case "video":
		return "影片"
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
