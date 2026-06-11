package localsearch

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui_console/builtin"
)

func TestSearchFindsContentAndPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "projects", "default", "documents")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(path, "note.md")
	if err := os.WriteFile(file, []byte("User: 今天討論本機搜尋功能"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService([]Root{{Path: root, Source: "document"}}, nil) // 用預設會搜的本機檔案來源
	results, err := svc.Search(SearchRequest{Query: "本機搜尋", Limit: 3})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 || results[0].Path == "" || !strings.Contains(results[0].Snippet, "本機搜尋") {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestSearchRedactsSnippet(t *testing.T) {
	svc := NewService(nil, []Item{{
		Source:  "document", // 預設會搜的本機檔案來源（記憶搜尋另有專測）
		Title:   "secret",
		Content: "token " + "sk-" + "abcdefghijklmnopqrstuvwxyz123456" + " should not show",
	}})
	results, err := svc.Search(SearchRequest{Query: "token"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || strings.Contains(results[0].Snippet, "sk-"+"abcdefghijklmnopqrstuvwxyz") {
		t.Fatalf("snippet not redacted: %#v", results)
	}
}

func TestFormatSearchOutcomeOnlyShowsContentSnippets(t *testing.T) {
	out := FormatSearchOutcome(SearchRequest{Query: "試試看"}, SearchOutcome{Results: []SearchResult{
		{
			Source:  "document",
			Title:   "試試看.txt",
			Path:    "/Users/tester/Library/Application Support/ai-console/data/references/files/試試看.txt",
			Snippet: "試試看，很好玩的文件",
		},
		{
			Source:  "memory",
			Title:   "talk_full.md",
			Path:    "/Users/tester/Library/Application Support/ai-console/data/projects/default/memory/talk_full.md",
			Snippet: "## [2026-05-24 01:00:12] user\n你能找到試試看文檔嗎？",
		},
	}})

	for _, forbidden := range []string{"路徑：", "[文件]", "[記憶]", "試試看.txt", "talk_full.md", "/Users/tester"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("formatted result leaked location metadata %q: %s", forbidden, out)
		}
	}
	if !strings.Contains(out, "有找到 2 筆") || !strings.Contains(out, "內容：試試看，很好玩的文件") {
		t.Fatalf("formatted result should say found and show content snippets: %s", out)
	}
}

func TestParseUserQueryAcceptsExplicitLocalCommands(t *testing.T) {
	req, ok := ParseUserQuery("查詢 記憶 裡的天氣")
	if !ok || req.Query != "天氣" || len(req.Scope) == 0 || req.Scope[0] != "memory" {
		t.Fatalf("local query not parsed: %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("查詢 API key")
	if !ok || req.Query != "API key" {
		t.Fatalf("查詢 should no longer require scope: %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("搜尋：天氣設定")
	if !ok || req.Query != "天氣設定" {
		t.Fatalf("full-width colon query not parsed: %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("search: API key")
	if !ok || req.Query != "API key" {
		t.Fatalf("english colon query not parsed: %#v ok=%v", req, ok)
	}
	if _, ok := ParseUserQuery("search API key"); ok {
		t.Fatal("english command without colon should not intercept normal prose")
	}
	if _, ok := ParseUserQuery("How do I search in VS Code for d:/?"); ok {
		t.Fatal("normal english question should not trigger direct local search")
	}
}

func TestRequestFromActionSupportsActionChainLocalSearch(t *testing.T) {
	req, ok := RequestFromAction("本機搜尋", "記憶 API key")
	if !ok || req.Query != "API key" || len(req.Scope) == 0 || req.Scope[0] != "memory" {
		t.Fatalf("action-chain local search not parsed: %#v ok=%v", req, ok)
	}
	req, ok = RequestFromAction("search", "API key scope=memory,document")
	if !ok || req.Query != "API key" || len(req.Scope) != 2 {
		t.Fatalf("scope directive should not leak into query: %#v ok=%v", req, ok)
	}
	if _, ok := RequestFromAction("聊天", "本機搜尋 API key"); ok {
		t.Fatal("non-search action should not route to local search")
	}
}

func TestFileTypeClassifierKeepsGenericFilesAllAndSpecificTypesScoped(t *testing.T) {
	req, ok := ParseUserQuery("搜尋 剛剛拉進來的檔案")
	if !ok || req.Query != "剛剛拉進來的檔案" || len(req.Scope) != 1 || req.Scope[0] != "all" {
		t.Fatalf("generic 檔案 should search all scopes: %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("搜尋 文件 客戶合約")
	if !ok || req.Query != "客戶合約" || len(req.Scope) != 1 || req.Scope[0] != "document" {
		t.Fatalf("文件 should scope to document and strip classifier word: %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("搜尋 圖片 介面截圖")
	if !ok || req.Query != "介面截圖" || len(req.Scope) != 1 || req.Scope[0] != "image" {
		t.Fatalf("圖片 should scope to image: %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("搜尋 影片 教學")
	if !ok || req.Query != "教學" || len(req.Scope) != 1 || req.Scope[0] != "video" {
		t.Fatalf("影片 should scope to video: %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("搜尋 照片和影片")
	if !ok || req.Query != "照片和影片" || len(req.Scope) != 2 || req.Scope[0] != "image" || req.Scope[1] != "video" {
		t.Fatalf("mixed media words should keep both scopes: %#v ok=%v", req, ok)
	}
}

func TestDefaultSearchExcludesTraceAndMemoryUnlessExplicit(t *testing.T) {
	svc := NewService(nil, []Item{
		{Source: "trace", Title: "trace stdout", Content: "needle from current stdout"},
		{Source: "memory", Title: "memory note", Content: "needle from user memory"},
		{Source: "document", Title: "doc", Content: "needle in a document"},
		{Source: "image", Title: "image", Content: "needle in an image filename"},
		{Source: "video", Title: "video", Content: "needle in a video filename"},
	})
	// 預設 all：排除 trace 與 memory（對話已是 context，避免自我回音），只回本機檔案。
	results, err := svc.Search(SearchRequest{Query: "needle", Scope: []string{"all"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("default all scope should skip trace+memory and keep file categories: %#v", results)
	}

	// 明確 memory：才回對話。
	results, err = svc.Search(SearchRequest{Query: "needle", Scope: []string{"memory"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search memory: %v", err)
	}
	if len(results) != 1 || results[0].Source != "memory" {
		t.Fatalf("explicit memory scope should include memory: %#v", results)
	}

	// 明確 trace：才回 trace。
	results, err = svc.Search(SearchRequest{Query: "needle", Scope: []string{"trace"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search trace: %v", err)
	}
	if len(results) != 1 || results[0].Source != "trace" {
		t.Fatalf("explicit trace scope should include trace: %#v", results)
	}

	results, err = svc.Search(SearchRequest{Query: "needle", Scope: []string{"image"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search image: %v", err)
	}
	if len(results) != 1 || results[0].Source != "image" {
		t.Fatalf("explicit image scope should include only image: %#v", results)
	}

	results, err = svc.Search(SearchRequest{Query: "needle", Scope: []string{"video"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search video: %v", err)
	}
	if len(results) != 1 || results[0].Source != "video" {
		t.Fatalf("explicit video scope should include only video: %#v", results)
	}
}

func TestSearchSkipsUnsafeDirectories(t *testing.T) {
	root := t.TempDir()
	secretDir := filepath.Join(root, "data", "secrets")
	if err := os.MkdirAll(secretDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("needle"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService([]Root{{Path: root, Source: "file"}}, nil)
	results, err := svc.Search(SearchRequest{Query: "needle"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("secret directory should be skipped: %#v", results)
	}
}

func TestSearchWithContextMarksCanceledScanIncomplete(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("needle"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := NewService([]Root{{Path: root, Source: "document"}}, nil)
	outcome, err := svc.SearchWithContext(ctx, SearchRequest{Query: "needle"})
	if err != nil {
		t.Fatalf("SearchWithContext: %v", err)
	}
	if !outcome.Incomplete || outcome.Reason != "timeout" {
		t.Fatalf("expected incomplete timeout outcome: %#v", outcome)
	}
}

func TestSearchWithContextMarksScanLimitIncomplete(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("needle"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService([]Root{{Path: root, Source: "document"}}, nil)
	svc.maxFiles = 1
	outcome, err := svc.SearchWithContext(context.Background(), SearchRequest{Query: "needle", Limit: 10})
	if err != nil {
		t.Fatalf("SearchWithContext: %v", err)
	}
	if !outcome.Incomplete || outcome.Reason != "scan limit" {
		t.Fatalf("expected scan limit outcome: %#v", outcome)
	}
	if !strings.Contains(FormatSearchOutcome(SearchRequest{Query: "needle"}, outcome), "搜尋可能不完整") {
		t.Fatal("incomplete outcome should be visible in chat response")
	}
}

func TestSearchSkipsNonUTF8Files(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bad.txt"), []byte{0xff, 0xfe, 0xfd}, 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService([]Root{{Path: root, Source: "document"}}, nil)
	results, err := svc.Search(SearchRequest{Query: "needle"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("non-UTF-8 file should be skipped: %#v", results)
	}
}

func TestSearchIndexesAllSupportedDocumentFormats(t *testing.T) {
	cases := []struct {
		name   string
		ext    string
		create func(string, string) error
	}{
		{"txt", ".txt", func(path, text string) error { return os.WriteFile(path, []byte(text), 0o600) }},
		{"md", ".md", func(path, text string) error { return os.WriteFile(path, []byte("# 標題\n"+text), 0o600) }},
		{"csv", ".csv", func(path, text string) error { return os.WriteFile(path, []byte("name,value\nrow,"+text+"\n"), 0o600) }},
		{"tsv", ".tsv", func(path, text string) error {
			return os.WriteFile(path, []byte("name\tvalue\nrow\t"+text+"\n"), 0o600)
		}},
		{"json", ".json", func(path, text string) error { return os.WriteFile(path, []byte(`{"content":"`+text+`"}`), 0o600) }},
		{"html", ".html", func(path, text string) error {
			return os.WriteFile(path, []byte("<html><body><p>"+text+"</p></body></html>"), 0o600)
		}},
		{"htm", ".htm", func(path, text string) error {
			return os.WriteFile(path, []byte("<html><body><p>"+text+"</p></body></html>"), 0o600)
		}},
		{"docx", ".docx", func(path, text string) error { return builtin.GenerateDocx(text, path) }},
		{"xlsx", ".xlsx", func(path, text string) error { return builtin.GenerateXlsx("欄位\n"+text, path) }},
		{"pptx", ".pptx", func(path, text string) error { return builtin.GeneratePptx(path, text) }},
		{"odt", ".odt", func(path, text string) error { return builtin.GenerateOdt(text, path) }},
		{"ods", ".ods", func(path, text string) error { return builtin.GenerateOds(text, path) }},
		{"odp", ".odp", func(path, text string) error { return builtin.GenerateOdp(text, path) }},
		{"epub", ".epub", createTestEpub},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			needle := "共同索引驗收-" + tc.name
			path := filepath.Join(root, "sample"+tc.ext)
			if err := tc.create(path, needle); err != nil {
				t.Fatalf("create %s: %v", tc.ext, err)
			}
			wantPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				t.Fatalf("EvalSymlinks: %v", err)
			}
			svc := NewService([]Root{{Path: root, Source: "document"}}, nil)
			results, err := svc.Search(SearchRequest{Query: needle, Scope: []string{"document"}, Limit: 3})
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if len(results) == 0 || results[0].Path != wantPath || !strings.Contains(results[0].Snippet, needle) {
				t.Fatalf("%s should be indexed and searchable, got %#v", tc.ext, results)
			}
		})
	}
}

func TestSearchMatchesRecentReferencePhraseByTerms(t *testing.T) {
	svc := NewService(nil, []Item{{
		Source:   "document",
		Title:    "最近引用文件: demo.xlsx",
		Content:  "demo.xlsx\n副檔名 xlsx",
		Keywords: "引用文件 已載入 檔案 本機資料 最近 最新 剛剛 剛才 拉進來 拉進來的 拖進來",
	}})
	results, err := svc.Search(SearchRequest{Query: "看到 剛剛 拉進來的 檔案", Scope: []string{"document"}, Limit: 3})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("recent reference phrase should match metadata item: %#v", results)
	}
}

func TestSearchIndexesImageAndVideoMetadata(t *testing.T) {
	root := t.TempDir()
	imageDir := filepath.Join(root, "data", "references", "files")
	videoDir := filepath.Join(root, "data", "videos")
	if err := os.MkdirAll(imageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(videoDir, 0o700); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(imageDir, "介面截圖.png")
	videoPath := filepath.Join(videoDir, "教學影片.mp4")
	if err := os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G'}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(videoPath, []byte{0x00, 0x00, 0x00, 0x18}, 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService([]Root{
		{Path: imageDir, Source: "document"},
		{Path: videoDir, Source: "video"},
	}, nil)

	results, err := svc.Search(SearchRequest{Query: "圖片 介面截圖", Scope: []string{"image"}, Limit: 5})
	if err != nil {
		t.Fatalf("Search image: %v", err)
	}
	if len(results) != 1 || results[0].Source != "image" || !strings.Contains(results[0].Title, "介面截圖") {
		t.Fatalf("image metadata should be indexed as image: %#v", results)
	}

	results, err = svc.Search(SearchRequest{Query: "影片 教學", Scope: []string{"video"}, Limit: 5})
	if err != nil {
		t.Fatalf("Search video: %v", err)
	}
	if len(results) != 1 || results[0].Source != "video" || !strings.Contains(results[0].Title, "教學影片") {
		t.Fatalf("video metadata should be indexed as video: %#v", results)
	}
}

func createTestEpub(path, text string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	files := map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
</package>`,
		"OEBPS/chapter1.xhtml": `<html xmlns="http://www.w3.org/1999/xhtml"><body><p>` + text + `</p></body></html>`,
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func TestBuildSnippetCentersAroundQueryWithinOneHundredRunes(t *testing.T) {
	content := strings.Repeat("前", 80) + "試試看" + strings.Repeat("後", 80)
	snippet := buildSnippet(content, "試試看")
	if got := len([]rune(strings.Trim(snippet, "."))); got != maxSnippetRunes {
		t.Fatalf("snippet should be %d runes before ellipsis, got %d: %q", maxSnippetRunes, got, snippet)
	}
	if !strings.Contains(snippet, "試試看") {
		t.Fatalf("snippet should include query: %q", snippet)
	}
	if strings.Count(snippet, "前") < 40 || strings.Count(snippet, "後") < 40 {
		t.Fatalf("snippet should stay centered around query: %q", snippet)
	}
}
