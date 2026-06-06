package safefetcher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ui_console/domain/url_source"
	"ui_console/internal/urlsafe"
)

func TestDecideFetch(t *testing.T) {
	tests := []struct {
		src         url_source.Source
		allowlisted bool
		want        Decision
	}{
		{url_source.SourceUserPaste, false, DecisionNeedConfirm},
		{url_source.SourceUserPaste, true, DecisionAllow},
		{url_source.SourceLLMExtracted, true, DecisionNeedConfirm}, // allowlist 也不能自動
		{url_source.SourceRemoteBridge, false, DecisionNeedConfirm},
		{url_source.SourceWebSearchResult, true, DecisionAllow},
		{url_source.SourceSkillManifest, false, DecisionDeny},
		{url_source.SourceSkillManifest, true, DecisionNeedConfirm},
	}
	for _, tt := range tests {
		if got := DecideFetch(tt.src, tt.allowlisted); got != tt.want {
			t.Errorf("DecideFetch(%s, %v)=%s, want %s", tt.src, tt.allowlisted, got, tt.want)
		}
	}
}

func TestFetch_NeedConfirmWithoutUserConfirm(t *testing.T) {
	_, err := FetchURLForLLM(context.Background(), FetchRequest{
		RawURL: "https://example.com",
		Source: url_source.SourceLLMExtracted,
	})
	if !errors.Is(err, ErrNeedConfirm) {
		t.Fatalf("未確認的 llm_extracted 應回 ErrNeedConfirm，得到 %v", err)
	}
}

func TestFetch_HTMLExtraction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, _ := r.Cookie("any"); c != nil {
			t.Error("不應帶 cookie")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>測試頁</title><script>evil()</script></head>` +
			`<body><style>.x{}</style><p>正文內容</p></body></html>`))
	}))
	defer srv.Close()

	old := fetchPolicy
	fetchPolicy = urlsafe.PolicyWebhookDev // 測試允許 http://127.0.0.1
	defer func() { fetchPolicy = old }()

	res, err := FetchURLForLLM(context.Background(), FetchRequest{
		RawURL:        srv.URL,
		Source:        url_source.SourceUserPaste,
		UserConfirmed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Title != "測試頁" {
		t.Errorf("title=%q", res.Title)
	}
	if !strings.Contains(res.Text, "正文內容") {
		t.Errorf("text 應含正文: %q", res.Text)
	}
	if strings.Contains(res.Text, "evil") {
		t.Error("script 內容不應被抽出")
	}
	if res.ContentHash == "" {
		t.Error("應有 content hash")
	}
}

func TestFetch_RejectsBinaryMIME(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x4d, 0x5a})
	}))
	defer srv.Close()

	old := fetchPolicy
	fetchPolicy = urlsafe.PolicyWebhookDev
	defer func() { fetchPolicy = old }()

	_, err := FetchURLForLLM(context.Background(), FetchRequest{
		RawURL: srv.URL, Source: url_source.SourceUserPaste, UserConfirmed: true,
	})
	if !errors.Is(err, ErrUnsupportedMIME) {
		t.Fatalf("binary 應被拒，得到 %v", err)
	}
}

// 正式 policy 下，內網/metadata 目標必須被擋（fetcher 不可成為 SSRF 入口）。
func TestFetch_SSRFBlockedUnderRealPolicy(t *testing.T) {
	for _, bad := range []string{
		"http://127.0.0.1:48765/",                  // http + loopback
		"https://169.254.169.254/latest/meta-data", // metadata
		"https://metadata.google.internal/",        // metadata 主機名
	} {
		_, err := FetchURLForLLM(context.Background(), FetchRequest{
			RawURL: bad, Source: url_source.SourceUserPaste, UserConfirmed: true,
		})
		if err == nil {
			t.Errorf("%s 應被擋", bad)
		}
	}
}
