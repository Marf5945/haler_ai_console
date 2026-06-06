// task_progress_fs.go — Phase A: Runtime FS Read Actions for DAG executor
//
// 本檔提供 4 個 read-only file system action 給 DAG 的 tool_call executor 使用：
//   - list_directory
//   - read_file
//   - glob
//   - grep_search
//
// 設計原則（依使用者 v4.3 spec §1 雙軌 + §6 純 stdlib）：
//   1. 純 stdlib，不引入第三方套件
//   2. 路徑邊界：ProjectRoot + data/references/files cache（v1 範圍）
//   3. 截斷時回傳結構化 metadata（truncated/limit_bytes/reason）
//   4. next_offset 預留欄位但不啟用（防 LLM token 水車）
//   5. risk = low（read-only、邊界內）
//
// 不抽 helper 出 package main、不新增跨層 import；所有實作 private。
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"ui_console/data/storage"
	"ui_console/shared/controlseal"
)

// ──────────────────────────────────────────────────────────────────────
// Constants — 截斷上限（依 spec §2）
// ──────────────────────────────────────────────────────────────────────

const (
	fsMaxListEntries       = 200       // list_directory 最多回 200 筆
	fsMaxListEntryBytes    = 200       // 每筆 entry 描述最多 200 byte
	fsMaxReadDefaultBytes  = 64 * 1024 // read_file 預設 64 KB
	fsMaxReadTextBytes     = 128 * 1024 // .md/.txt 提高到 128 KB
	fsMaxGlobMatches       = 100        // glob 最多回 100 個匹配
	fsMaxGrepMatches       = 50         // grep_search 最多回 50 筆匹配行
	fsMaxGrepLineBytes     = 200        // 每行 grep 結果最多 200 byte
	fsGrepLineSkipBytes    = 1000       // 單行超過 1000 byte 直接跳過該行
	fsTruncReasonOverflow  = "context_overflow_guard"
	fsTruncReasonLineLimit = "line_limit_reached"
)

// ──────────────────────────────────────────────────────────────────────
// Sensitive paths blacklist — 即使在邊界內也拒絕
// ──────────────────────────────────────────────────────────────────────
//
// 設計（spec §5）：以 path component 比對，case-insensitive。任一 path 段
// 命中即拒絕。跨平台覆蓋 macOS 與 Windows 常見敏感目錄/檔案。
var fsSensitivePathComponents = []string{
	".ssh",
	".git",
	".env",
	".aws",
	".gnupg",
	".docker",
	"keychains",
	"cookies",
	"credentials",
	"token",
	"id_rsa",
	"id_ed25519",
}

// fsTextExtensions 列出視為「純文字」可放寬到 128 KB 的副檔名。
var fsTextExtensions = map[string]bool{
	".md":    true,
	".txt":   true,
	".markdown": true,
}

// ──────────────────────────────────────────────────────────────────────
// 入口 dispatcher
// ──────────────────────────────────────────────────────────────────────

// fsActionCodes 為 executeToolTaskNode 提供白名單判斷。
var fsActionCodes = map[string]bool{
	"list_directory": true,
	"read_file":      true,
	"glob":           true,
	"grep_search":    true,
}

// isFSActionCode reports whether the given action_code dispatches to FS read.
func isFSActionCode(code string) bool {
	return fsActionCodes[code]
}

// dispatchFSAction routes to the right action implementation. Returns a JSON
// string (DAG result_summary 寫進 node.Result) so 截斷 metadata 自然帶過去。
func (a *App) dispatchFSAction(actionCode, target string) (string, error) {
	root := a.fsAllowedRoots()
	switch actionCode {
	case "list_directory":
		return fsListDirectory(root, target)
	case "read_file":
		return fsReadFile(root, target)
	case "glob":
		return fsGlob(root, target)
	case "grep_search":
		return fsGrepSearch(root, target)
	default:
		return "", fmt.Errorf("fs: unsupported action_code %q", actionCode)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Allowed roots
// ──────────────────────────────────────────────────────────────────────

// fsAllowedRoots 回傳 v1 允許讀取的根目錄清單（spec §4 範圍）。
// v1 只支援 ProjectRoot + references files cache；ReferenceLibrary Original
// Path 在 spec/code 尚未落實，待 v2 由 reference upload 流程註冊後加入。
//
// 注意（2026-05-28 修正）：references files cache 實際存放在 appDataRoot() 之下
// （見 app.go 的 "referenceDir" 與 app_document.go:184）——是 app-level，不是
// project-level。我先前誤接到 projectRoot 之下導致 grep_search 找不到使用者拖入的
// 檔案，fallback 到 projectRoot 後反而 grep 到 memory/talk_full.md。
func (a *App) fsAllowedRoots() []string {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	return []string{
		projectRoot,
		filepath.Join(appDataRoot(), "data", "references", "files"),
	}
}

// ──────────────────────────────────────────────────────────────────────
// Path boundary check (spec §5)
// ──────────────────────────────────────────────────────────────────────

// fsCheckPath 對使用者提供的 path（可能是相對路徑、含 ..、含 symlink）做完整邊界檢查。
// 通過後回傳 resolved 絕對路徑；不通過直接 error。
//
// 步驟：
//   1. trim + filepath.Clean
//   2. 若為相對路徑，與第一個 allowed root 接合
//   3. 取絕對路徑
//   4. EvalSymlinks 解 symlink（檔案不存在不視為錯誤，僅在 read_file 那邊 stat 處理）
//   5. 路徑必須位於任一 allowed root 之下
//   6. 任一 path component 不得命中 sensitive 黑名單
func fsCheckPath(allowedRoots []string, target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("fs: empty path")
	}

	cleaned := filepath.Clean(target)

	// 相對路徑：用第一個 allowed root 當 base
	if !filepath.IsAbs(cleaned) {
		if len(allowedRoots) == 0 {
			return "", fmt.Errorf("fs: no allowed root configured")
		}
		cleaned = filepath.Join(allowedRoots[0], cleaned)
	}

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("fs: resolve abs: %w", err)
	}

	// EvalSymlinks：若 path 不存在，回傳原 abs（讓 stat 階段給出明確錯誤）
	resolved := abs
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		resolved = real
	}

	// 必須位於某個 allowed root 下
	if !fsWithinAnyRoot(resolved, allowedRoots) {
		return "", fmt.Errorf("fs: path outside allowed roots: %s", target)
	}

	// 敏感 path component 黑名單檢查
	if comp := fsHitsSensitiveComponent(resolved); comp != "" {
		return "", fmt.Errorf("fs: path contains sensitive component %q (blocked unconditionally)", comp)
	}

	return resolved, nil
}

// fsWithinAnyRoot 檢查 resolved 是否在任一 allowed root 之內。
// 注意：root 自己也需 EvalSymlinks，否則使用者 home 是 symlink 時會誤判。
func fsWithinAnyRoot(resolved string, roots []string) bool {
	for _, root := range roots {
		realRoot := root
		if r, err := filepath.EvalSymlinks(root); err == nil {
			realRoot = r
		}
		rel, err := filepath.Rel(realRoot, resolved)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

// fsHitsSensitiveComponent 對 path 拆每一段，case-insensitive 比對黑名單。
// 命中即回傳該段名稱；沒命中回傳 ""。
func fsHitsSensitiveComponent(path string) string {
	sep := string(os.PathSeparator)
	parts := strings.Split(path, sep)
	if runtime.GOOS == "windows" {
		// Windows 同時切 / 與 \
		parts = strings.FieldsFunc(path, func(r rune) bool { return r == '\\' || r == '/' })
	}
	for _, p := range parts {
		if p == "" {
			continue
		}
		lp := strings.ToLower(p)
		for _, bad := range fsSensitivePathComponents {
			if lp == bad || strings.Contains(lp, bad) {
				return p
			}
		}
	}
	return ""
}

// ──────────────────────────────────────────────────────────────────────
// list_directory
// ──────────────────────────────────────────────────────────────────────

type fsListEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

type fsListResult struct {
	Path      string        `json:"path"`
	Entries   []fsListEntry `json:"entries"`
	Total     int           `json:"total"`
	Truncated bool          `json:"truncated,omitempty"`
	Reason    string        `json:"reason,omitempty"`
}

func fsListDirectory(allowedRoots []string, target string) (string, error) {
	resolved, err := fsCheckPath(allowedRoots, target)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("fs list_directory: stat: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("fs list_directory: not a directory: %s", target)
	}

	dirEntries, err := os.ReadDir(resolved)
	if err != nil {
		return "", fmt.Errorf("fs list_directory: read: %w", err)
	}

	out := fsListResult{Path: target, Total: len(dirEntries)}
	for i, e := range dirEntries {
		if i >= fsMaxListEntries {
			out.Truncated = true
			out.Reason = fsTruncReasonOverflow
			break
		}
		// SEC-W29.8（2026-05-27）：檔名可能含 ㄌ / 偽 seal，sanitize 後再回 LLM。
		name := controlseal.SanitizeForLLM(controlseal.SourceToolOutput, e.Name()).LLMText
		if len(name) > fsMaxListEntryBytes {
			name = name[:fsMaxListEntryBytes] + "..."
		}
		entry := fsListEntry{Name: name, IsDir: e.IsDir()}
		if !e.IsDir() {
			if fi, err := e.Info(); err == nil {
				entry.Size = fi.Size()
			}
		}
		out.Entries = append(out.Entries, entry)
	}
	return marshalJSON(out)
}

// ──────────────────────────────────────────────────────────────────────
// read_file
// ──────────────────────────────────────────────────────────────────────

type fsReadResult struct {
	Path            string `json:"path"`
	BytesRead       int    `json:"bytes_read"`
	TotalSize       int64  `json:"total_size"`
	LimitBytes      int    `json:"limit_bytes"`
	Content         string `json:"content"`
	Truncated       bool   `json:"truncated,omitempty"`
	Reason          string `json:"reason,omitempty"`
	NextOffset      int64  `json:"next_offset,omitempty"`       // 預留，v1 不啟用
	OffsetSupported bool   `json:"offset_supported"`            // v1 = false
}

func fsReadFile(allowedRoots []string, target string) (string, error) {
	resolved, err := fsCheckPath(allowedRoots, target)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("fs read_file: stat: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("fs read_file: path is a directory: %s", target)
	}

	// 純文字副檔名放寬 128KB；其他預設 64KB
	limit := fsMaxReadDefaultBytes
	if fsTextExtensions[strings.ToLower(filepath.Ext(resolved))] {
		limit = fsMaxReadTextBytes
	}

	f, err := os.Open(resolved)
	if err != nil {
		return "", fmt.Errorf("fs read_file: open: %w", err)
	}
	defer f.Close()

	buf, err := io.ReadAll(io.LimitReader(f, int64(limit)+1)) // +1 用來偵測是否超出
	if err != nil {
		return "", fmt.Errorf("fs read_file: read: %w", err)
	}

	out := fsReadResult{
		Path:            target,
		TotalSize:       info.Size(),
		LimitBytes:      limit,
		OffsetSupported: false,
	}
	// SEC-W29.8（2026-05-27）：檔案內容是外部不可信來源，sanitize 後再放進 LLM context。
	// 截斷後再 sanitize；先 sanitize 再截斷會在 escape marker 中間截開。
	var rawContent string
	if len(buf) > limit {
		out.Truncated = true
		out.Reason = fsTruncReasonOverflow
		rawContent = string(buf[:limit])
		out.BytesRead = limit
		out.NextOffset = int64(limit) // 預留欄位，v1 client 不應依賴
	} else {
		rawContent = string(buf)
		out.BytesRead = len(buf)
	}
	sanitized := controlseal.SanitizeForLLM(controlseal.SourceDocument, rawContent)
	out.Content = sanitized.LLMText
	return marshalJSON(out)
}

// ──────────────────────────────────────────────────────────────────────
// glob
// ──────────────────────────────────────────────────────────────────────

type fsGlobResult struct {
	Pattern   string   `json:"pattern"`
	Matches   []string `json:"matches"`
	Total     int      `json:"total"`
	Truncated bool     `json:"truncated,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}

// fsGlob 純 stdlib 實作。支援以下 pattern 子集（spec §6）：
//   - 單層 glob：*.md / data/*.txt
//   - 遞迴：**/*.md / **/*.txt / **/*.{md,txt}
//
// 不支援完整 shell glob。
func fsGlob(allowedRoots []string, pattern string) (string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", fmt.Errorf("fs glob: empty pattern")
	}

	// 找一個 walk 起點：pattern 前 ** 之前的固定部分
	root := allowedRoots[0]
	subRoot, rest := splitGlobRoot(pattern)
	walkRoot := filepath.Join(root, subRoot)
	if _, err := fsCheckPath(allowedRoots, walkRoot); err != nil {
		return "", err
	}

	// 把 {md,txt} 展開成多 pattern
	expanded := expandBracePattern(rest)

	out := fsGlobResult{Pattern: pattern}
	seen := map[string]bool{}

	walkErr := filepath.WalkDir(walkRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 單一目錄失敗不終止整體
		}
		if d.IsDir() {
			// 遇到 sensitive component 整個 dir 跳過
			if fsHitsSensitiveComponent(p) != "" {
				return filepath.SkipDir
			}
			return nil
		}
		if fsHitsSensitiveComponent(p) != "" {
			return nil
		}
		// 計算相對於 walkRoot 的相對路徑，套各 expanded pattern
		rel, err := filepath.Rel(walkRoot, p)
		if err != nil {
			return nil
		}
		matched := false
		for _, pat := range expanded {
			if globMatch(pat, rel) {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
		if seen[p] {
			return nil
		}
		seen[p] = true
		if len(out.Matches) >= fsMaxGlobMatches {
			out.Truncated = true
			out.Reason = fsTruncReasonOverflow
			return filepath.SkipAll
		}
		// SEC-W29.8：path 中的檔名可能含 ㄌ，sanitize 後再回 LLM。
		out.Matches = append(out.Matches, controlseal.SanitizeForLLM(controlseal.SourceToolOutput, p).LLMText)
		return nil
	})
	if walkErr != nil && walkErr != filepath.SkipAll {
		return "", fmt.Errorf("fs glob: walk: %w", walkErr)
	}
	out.Total = len(out.Matches)
	return marshalJSON(out)
}

// splitGlobRoot 從 pattern 中拆出「不含 wildcard 的前綴」當 walk 起點。
//   "data/refs/**/*.md" → ("data/refs", "**/*.md")
//   "**/*.md"           → ("", "**/*.md")
//   "*.md"              → ("", "*.md")
func splitGlobRoot(pattern string) (string, string) {
	parts := strings.Split(pattern, "/")
	for i, p := range parts {
		if strings.ContainsAny(p, "*?[{") {
			return strings.Join(parts[:i], "/"), strings.Join(parts[i:], "/")
		}
	}
	return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1]
}

// expandBracePattern 把 "**/*.{md,txt}" 展開為 ["**/*.md", "**/*.txt"]。
// 只支援單一 {...} 區塊（spec §6 子集）。
func expandBracePattern(pattern string) []string {
	open := strings.Index(pattern, "{")
	close := strings.Index(pattern, "}")
	if open < 0 || close < 0 || close < open {
		return []string{pattern}
	}
	prefix := pattern[:open]
	suffix := pattern[close+1:]
	alts := strings.Split(pattern[open+1:close], ",")
	out := make([]string, 0, len(alts))
	for _, alt := range alts {
		out = append(out, prefix+strings.TrimSpace(alt)+suffix)
	}
	return out
}

// globMatch 支援 ** 遞迴與 * / ? 單層。
// 把 ** 換成 internal sentinel 再轉 regexp 處理。
func globMatch(pattern, name string) bool {
	// 把 ** 換成佔位符，避免被當成兩個 *
	const dblStarPlaceholder = "\x00DOUBLESTAR\x00"
	p := strings.ReplaceAll(pattern, "**", dblStarPlaceholder)

	var re strings.Builder
	re.WriteString("^")
	for _, r := range p {
		switch r {
		case '*':
			re.WriteString("[^/]*")
		case '?':
			re.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			re.WriteString("\\")
			re.WriteRune(r)
		default:
			re.WriteRune(r)
		}
	}
	re.WriteString("$")
	pat := strings.ReplaceAll(re.String(), dblStarPlaceholder, ".*")
	matched, err := regexp.MatchString(pat, name)
	if err != nil {
		return false
	}
	return matched
}

// ──────────────────────────────────────────────────────────────────────
// grep_search
// ──────────────────────────────────────────────────────────────────────

type fsGrepHit struct {
	File    string `json:"file"`
	LineNo  int    `json:"line_no"`
	Line    string `json:"line"`
	Truncated bool `json:"line_truncated,omitempty"`
}

type fsGrepResult struct {
	Pattern   string      `json:"pattern"`
	Hits      []fsGrepHit `json:"hits"`
	Total     int         `json:"total"`
	Truncated bool        `json:"truncated,omitempty"`
	Reason    string      `json:"reason,omitempty"`
}

// fsGrepSearch 在第一個 allowed root 下遞迴掃描純文字檔，套 regexp pattern。
// 為避免一次性把整個 cache 掃完，第一版固定走第一個 allowed root 的 references files。
func fsGrepSearch(allowedRoots []string, pattern string) (string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", fmt.Errorf("fs grep_search: empty pattern")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("fs grep_search: bad regexp: %w", err)
	}
	// 預設掃 references files cache（v1）；若不存在則退回 projectRoot
	searchRoot := allowedRoots[len(allowedRoots)-1]
	if _, err := os.Stat(searchRoot); err != nil {
		searchRoot = allowedRoots[0]
	}
	if _, err := fsCheckPath(allowedRoots, searchRoot); err != nil {
		return "", err
	}

	out := fsGrepResult{Pattern: pattern}
	walkErr := filepath.WalkDir(searchRoot, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() {
			if fsHitsSensitiveComponent(p) != "" {
				return filepath.SkipDir
			}
			return nil
		}
		if fsHitsSensitiveComponent(p) != "" {
			return nil
		}
		// 只掃純文字
		if !fsTextExtensions[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			// 單行超過 1000 byte 直接跳過（spec §2）
			if len(line) > fsGrepLineSkipBytes {
				continue
			}
			if !re.MatchString(line) {
				continue
			}
			if len(out.Hits) >= fsMaxGrepMatches {
				out.Truncated = true
				out.Reason = fsTruncReasonLineLimit
				return filepath.SkipAll
			}
			// SEC-W29.8：grep 行內容來自外部檔案，sanitize 後再回 LLM。
			// 截斷再 sanitize（同 read_file 處理順序）。
			rawLine := line
			truncated := false
			if len(rawLine) > fsMaxGrepLineBytes {
				rawLine = rawLine[:fsMaxGrepLineBytes]
				truncated = true
			}
			hit := fsGrepHit{
				File:      controlseal.SanitizeForLLM(controlseal.SourceToolOutput, p).LLMText,
				LineNo:    lineNo,
				Line:      controlseal.SanitizeForLLM(controlseal.SourceDocument, rawLine).LLMText,
				Truncated: truncated,
			}
			out.Hits = append(out.Hits, hit)
		}
		return nil
	})
	if walkErr != nil && walkErr != filepath.SkipAll {
		return "", fmt.Errorf("fs grep_search: walk: %w", walkErr)
	}
	out.Total = len(out.Hits)
	return marshalJSON(out)
}

// ──────────────────────────────────────────────────────────────────────
// shared
// ──────────────────────────────────────────────────────────────────────

func marshalJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("fs: marshal result: %w", err)
	}
	return string(b), nil
}
