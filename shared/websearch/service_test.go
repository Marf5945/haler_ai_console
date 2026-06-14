package websearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchTavily(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tvly-test" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Wails","url":"https://wails.io/","content":"Desktop app framework.","score":0.9}]}`))
	}))
	defer server.Close()

	service := NewServiceWithClient(rewriteTransportClient(server))
	outcome, err := service.Search(context.Background(), SearchRequest{Query: "wails", Limit: 2}, ProviderConfig{
		ProviderID: ProviderTavily,
		APIKey:     "tvly-test",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if outcome.ProviderID != ProviderTavily || len(outcome.Results) != 1 {
		t.Fatalf("unexpected outcome: %#v", outcome)
	}
	if outcome.Results[0].URL != "https://wails.io/" {
		t.Fatalf("unexpected result: %#v", outcome.Results[0])
	}
}

func TestSearchGoogleCSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "google-key" || r.URL.Query().Get("cx") != "cx-id" {
			t.Fatalf("missing google credentials in query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"title":"Result","link":"https://example.com/","snippet":"Snippet","displayLink":"example.com"}]}`))
	}))
	defer server.Close()

	service := NewServiceWithClient(rewriteTransportClient(server))
	outcome, err := service.Search(context.Background(), SearchRequest{Query: "example"}, ProviderConfig{
		ProviderID: ProviderGoogleCSE,
		APIKey:     "google-key",
		CX:         "cx-id",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Source != "example.com" {
		t.Fatalf("unexpected outcome: %#v", outcome)
	}
}

func TestSearchBrave(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Subscription-Token"); got != "brave-key" {
			t.Fatalf("X-Subscription-Token = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"Brave","url":"https://search.brave.com/","description":"Search API","profile":{"name":"Brave"}}]}}`))
	}))
	defer server.Close()

	service := NewServiceWithClient(rewriteTransportClient(server))
	outcome, err := service.Search(context.Background(), SearchRequest{Query: "brave"}, ProviderConfig{
		ProviderID: ProviderBrave,
		APIKey:     "brave-key",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Title != "Brave" {
		t.Fatalf("unexpected outcome: %#v", outcome)
	}
}

func TestRequestFromAction(t *testing.T) {
	req, ok := RequestFromAction("\u7db2\u8def\u641c\u5c0b", "Wails tutorial")
	if !ok || req.Query != "Wails tutorial" {
		t.Fatalf("RequestFromAction = %#v ok=%v", req, ok)
	}
	req, ok = RequestFromAction("\u7db2\u8def", "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2")
	if !ok || req.Query != "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2" {
		t.Fatalf("RequestFromAction short zh action = %#v ok=%v", req, ok)
	}
	if _, ok := RequestFromAction("search", "local only"); ok {
		t.Fatal("plain local search action should not be web search")
	}
}

func TestParseUserQuery(t *testing.T) {
	req, ok := ParseUserQuery("web_search: Wails")
	if !ok || req.Query != "Wails" {
		t.Fatalf("ParseUserQuery web_search = %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("WEB_SEARCH\u310cWails")
	if !ok || req.Query != "Wails" {
		t.Fatalf("ParseUserQuery uppercase web_search = %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("\u641c\u5c0b\u7db2\u8def Wails")
	if !ok || req.Query != "Wails" {
		t.Fatalf("ParseUserQuery zh = %#v ok=%v", req, ok)
	}
	req, ok = ParseUserQuery("\u7db2\u8def\u310c\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2")
	if !ok || req.Query != "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2" {
		t.Fatalf("ParseUserQuery short zh = %#v ok=%v", req, ok)
	}
}

func TestFormatSearchOutcomeIncludesURLs(t *testing.T) {
	out := FormatSearchOutcome(SearchRequest{Query: "Wails"}, SearchOutcome{
		Query:      "Wails",
		ProviderID: ProviderTavily,
		Provider:   "Tavily",
		Results: []SearchResult{{
			Title:   "Wails",
			URL:     "https://wails.io/",
			Snippet: "Desktop app framework.",
		}},
	})
	if !strings.Contains(out, "https://wails.io/") || !strings.Contains(out, "Web search") {
		t.Fatalf("formatted output missing expected content: %s", out)
	}
}

func rewriteTransportClient(server *httptest.Server) *http.Client {
	baseTransport := server.Client().Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	return &http.Client{Transport: rewriteTransport{base: baseTransport, target: server.URL}}
}

type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target := strings.TrimRight(t.target, "/")
	req.URL.Scheme = strings.Split(target, "://")[0]
	req.URL.Host = strings.TrimPrefix(target, req.URL.Scheme+"://")
	req.URL.Path = "/"
	req.Host = req.URL.Host
	return t.base.RoundTrip(req)
}
