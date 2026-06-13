package url_source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashURL_Normalization(t *testing.T) {
	h1, host, err := HashURL("HTTPS://Example.COM/path?q=1#frag")
	if err != nil {
		t.Fatal(err)
	}
	h2, _, _ := HashURL("https://example.com/path?q=1")
	if h1 != h2 {
		t.Errorf("正規化後 hash 應相同: %s vs %s", h1, h2)
	}
	if host != "example.com" {
		t.Errorf("host=%s, want example.com", host)
	}
}

// 洗白防護：LLM 提過的 URL，使用者再貼仍以最低信任為準。
func TestLowestTrustSource(t *testing.T) {
	r := NewRegistry("")
	u := "https://example.com/page"
	if _, err := r.Record(u, SourceLLMExtracted, "s1", "t1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Record(u, SourceUserPaste, "s1", "t2", ""); err != nil {
		t.Fatal(err)
	}
	src, ok := r.LowestTrustSource(u)
	if !ok || src != SourceLLMExtracted {
		t.Errorf("最低信任應為 llm_extracted，得到 %s (ok=%v)", src, ok)
	}
}

func TestRiskTier(t *testing.T) {
	if SourceLLMExtracted.RiskTier() != "high" || SourceUserPaste.RiskTier() != "low" {
		t.Error("risk tier 分層錯誤")
	}
}

// audit JSONL 不可含完整 URL（query 可能有 token）。
func TestAuditOmitsFullURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	r := NewRegistry(path)
	secret := "https://example.com/cb?token=SUPERSECRET"
	if _, err := r.Record(secret, SourceUserPaste, "s1", "t1", ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "SUPERSECRET") {
		t.Error("audit 檔不應含完整 URL / token")
	}
	if !strings.Contains(string(data), "example.com") {
		t.Error("audit 檔應含 normalized host")
	}
}

func TestRecordsForSession(t *testing.T) {
	r := NewRegistry("")
	_, _ = r.Record("https://a.com/1", SourceUserPaste, "s1", "", "")
	_, _ = r.Record("https://a.com/2", SourceUserPaste, "s2", "", "")
	_, _ = r.Record("https://a.com/3", SourceUserPaste, "s1", "", "")
	got := r.RecordsForSession("s1", 10)
	if len(got) != 2 {
		t.Errorf("s1 應有 2 筆，得到 %d", len(got))
	}
}
