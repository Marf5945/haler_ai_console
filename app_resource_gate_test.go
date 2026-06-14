package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui_console/domain/url_source"
)

func resetResourceGateForTest(t *testing.T) {
	t.Helper()
	pendingURLFetchMu.Lock()
	pendingURLFetches = map[string]pendingURLFetch{}
	pendingURLFetchMu.Unlock()
	pendingResourceMu.Lock()
	pendingResourceActions = map[string]pendingResourceAction{}
	pendingResourceMu.Unlock()
	urlRegistryOnce.Do(func() {})
	urlRegistry = url_source.NewRegistry(filepath.Join(t.TempDir(), "audit.jsonl"))
}

func TestResourceGateRoutesURLReadBeforeWebSearch(t *testing.T) {
	resetResourceGateForTest(t)
	app := &App{}
	resp, handled := app.maybeHandleResourceGate("讀取 https://example.com 的內容", "session-url-read", "trace-url-read")
	if !handled {
		t.Fatal("URL read should be handled by resource gate")
	}
	if !strings.Contains(resp.Text, "要我讀取 example.com 的內容嗎") {
		t.Fatalf("unexpected response: %q", resp.Text)
	}
}

func TestURLFetchPendingUsesLowestTrustSource(t *testing.T) {
	resetResourceGateForTest(t)
	app := &App{}
	rawURL := "https://example.com/llm-suggested"
	if _, err := getURLRegistry().Record(rawURL, url_source.SourceLLMExtracted, "session-llm", "trace-llm", ""); err != nil {
		t.Fatal(err)
	}

	_, handled := app.maybeHandleResourceGate("read "+rawURL, "session-url-read", "trace-url-read")
	if !handled {
		t.Fatal("URL read should be handled by resource gate")
	}
	pendingURLFetchMu.Lock()
	pending, ok := pendingURLFetches["session-url-read"]
	pendingURLFetchMu.Unlock()
	if !ok {
		t.Fatal("expected pending URL fetch")
	}
	if pending.Source != url_source.SourceLLMExtracted {
		t.Fatalf("pending source = %q, want %q", pending.Source, url_source.SourceLLMExtracted)
	}
}

func TestResourceGateAsksBeforeOpeningLocalURL(t *testing.T) {
	resetResourceGateForTest(t)
	app := &App{}
	resp, handled := app.maybeHandleResourceGate("開啟 http://127.0.0.1:48765", "session-url-open", "trace-url-open")
	if !handled {
		t.Fatal("local URL open should be handled by resource gate")
	}
	if !strings.Contains(resp.Text, "要我開啟 http://127.0.0.1:48765 嗎") {
		t.Fatalf("unexpected response: %q", resp.Text)
	}
}

func TestResourceGateAsksWhichFileWhenMissingTarget(t *testing.T) {
	resetResourceGateForTest(t)
	app := &App{}
	resp, handled := app.maybeHandleResourceGate("讀取檔案", "session-file-missing", "trace-file-missing")
	if !handled {
		t.Fatal("missing file target should be handled by resource gate")
	}
	if !strings.Contains(resp.Text, "哪個檔案") {
		t.Fatalf("unexpected response: %q", resp.Text)
	}
}

func TestResourceGateAsksBeforeReadingLocalPath(t *testing.T) {
	resetResourceGateForTest(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := &App{}
	resp, handled := app.maybeHandleResourceGate("讀取 "+path+" 的內容", "session-file-read", "trace-file-read")
	if !handled {
		t.Fatal("local path read should be handled by resource gate")
	}
	if !strings.Contains(resp.Text, "要我讀取這個本機檔案的內容嗎") {
		t.Fatalf("unexpected response: %q", resp.Text)
	}
}
