package urlsafe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// fakeResolver 讓測試能模擬 DNS 回任意 IP（含 DNS rebinding 情境）。
type fakeResolver struct{ ips []net.IPAddr }

func (f fakeResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return f.ips, nil
}

func ipAddr(s string) net.IPAddr { return net.IPAddr{IP: net.ParseIP(s)} }

func TestScreenIP(t *testing.T) {
	tests := []struct {
		name   string
		ip     string
		policy URLPolicy
		want   BlockReason // "" 代表放行
	}{
		{"public webhook ok", "1.1.1.1", PolicyWebhook, ""},
		{"link-local blocked all", "169.254.169.254", PolicyWebhookDev, ReasonLinkLocal},
		{"metadata oracle blocked dev", "192.0.0.192", PolicyWebhookDev, ReasonMetadataIP},
		{"metadata azure wireserver blocked dev", "168.63.129.16", PolicyWebhookDev, ReasonMetadataIP},
		{"metadata alibaba blocked dev", "100.100.100.200", PolicyWebhookDev, ReasonMetadataIP},
		{"cgnat blocked dev", "100.64.0.1", PolicyWebhookDev, ReasonReservedIP},
		{"private blocked on webhook", "10.0.0.5", PolicyWebhook, ReasonPrivateIP},
		{"private allowed on dev", "10.0.0.5", PolicyWebhookDev, ""},
		{"loopback blocked on webhook", "127.0.0.1", PolicyWebhook, ReasonLoopback},
		{"loopback allowed on localllm", "127.0.0.1", PolicyLocalLLM, ""},
		{"lan blocked on localllm", "192.168.1.10", PolicyLocalLLM, ReasonPrivateIP},
		{"v4-mapped link-local blocked", "::ffff:169.254.169.254", PolicyWebhookDev, ReasonLinkLocal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := screenIP(net.ParseIP(tt.ip), tt.policy)
			be := asBlockErr(err)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("screenIP(%s,%v)=%v, want nil", tt.ip, tt.policy, err)
				}
				return
			}
			if be == nil || be.Reason != tt.want {
				t.Fatalf("screenIP(%s,%v)=%v, want reason %s", tt.ip, tt.policy, err, tt.want)
			}
		})
	}
}

func TestScreenHostname(t *testing.T) {
	for _, h := range []string{"metadata.google.internal", "METADATA.GOOG", "metadata.google.internal."} {
		if err := screenHostname(h); asBlockErr(err) == nil {
			t.Errorf("screenHostname(%q) 應被封鎖", h)
		}
	}
	if err := screenHostname("api.openai.com"); err != nil {
		t.Errorf("screenHostname(api.openai.com) 不應封鎖: %v", err)
	}
}

// 模擬 DNS rebinding：網域解析回內網 IP，應在 dial 時被擋。
func TestSafeDialer_RebindingBlocked(t *testing.T) {
	client := newSafeClientWithResolver(PolicyWebhook, "test", 3*time.Second,
		fakeResolver{ips: []net.IPAddr{ipAddr("10.0.0.5")}})
	_, err := client.Get("https://looks-legit.example/")
	if be := asBlockErr(err); be == nil || be.Reason != ReasonPrivateIP {
		t.Fatalf("rebinding 應被擋為 private，得到 %v", err)
	}
}

// 多筆 A 記錄混合：任一危險就拒。
func TestSafeDialer_AnyDangerousRejected(t *testing.T) {
	client := newSafeClientWithResolver(PolicyWebhook, "test", 3*time.Second,
		fakeResolver{ips: []net.IPAddr{ipAddr("1.1.1.1"), ipAddr("169.254.169.254")}})
	_, err := client.Get("https://mixed.example/")
	if be := asBlockErr(err); be == nil {
		t.Fatalf("混合危險 IP 應被拒，得到 %v", err)
	}
}

// dev policy 允許 loopback：對本機 httptest server 應連得到。
func TestSafeDialer_AllowsLoopbackOnDev(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, _, _ := net.SplitHostPort(mustHost(t, srv.URL))
	client := newSafeClientWithResolver(PolicyWebhookDev, "test", 3*time.Second,
		fakeResolver{ips: []net.IPAddr{ipAddr(host)}})
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("dev loopback 應連得到: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
}

func TestSameOrigin(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"https://a.com/x", "https://a.com/y", true},
		{"https://a.com", "https://a.com:443/z", true}, // 補預設埠後同 origin
		{"https://a.com", "http://a.com", false},       // 降級
		{"https://a.com", "https://a.com:444", false},  // 換埠
		{"https://a.com", "https://b.com", false},      // 換 host
	}
	for _, tt := range tests {
		a, _ := url.Parse(tt.a)
		b, _ := url.Parse(tt.b)
		if got := sameOrigin(a, b); got != tt.want {
			t.Errorf("sameOrigin(%s,%s)=%v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestValidateURL_MetadataHostBlocked(t *testing.T) {
	for _, p := range []URLPolicy{PolicyCloudAPI, PolicyWebhook, PolicyWebhookDev, PolicyLocalLLM} {
		if err := ValidateURL("http://metadata.google.internal/computeMetadata/v1/", p); err == nil {
			t.Errorf("policy %v 應封鎖 metadata 主機名", policyName(p))
		}
	}
}

// SEC-05 2a: PolicyForLLMEndpoint 集中分流。
func TestPolicyForLLMEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		baseURL    string
		want       URLPolicy
	}{
		{"ollama → LocalLLM", "ollama", "http://localhost:11434", PolicyLocalLLM},
		{"lmstudio → LocalLLM", "LMStudio", "http://127.0.0.1:1234", PolicyLocalLLM},
		{"雲端公網 → CloudAPI", "openai", "https://api.openai.com/v1", PolicyCloudAPI},
		{"雲端 localhost → ConfirmedPrivate", "deepseek", "http://localhost:8080/v1", PolicyConfirmedPrivate},
		{"雲端 LAN IP → ConfirmedPrivate", "openai", "http://192.168.1.50:8080/v1", PolicyConfirmedPrivate},
		{"空 baseURL → CloudAPI", "openai", "", PolicyCloudAPI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PolicyForLLMEndpoint(tt.providerID, tt.baseURL); got != tt.want {
				t.Errorf("PolicyForLLMEndpoint(%q,%q)=%s, want %s",
					tt.providerID, tt.baseURL, policyName(got), policyName(tt.want))
			}
		})
	}
}

// SEC-05 2a: ConfirmedPrivate 放行 LAN/loopback，但 metadata 等仍永久封鎖。
func TestPolicyConfirmedPrivate(t *testing.T) {
	if err := screenIP(net.ParseIP("192.168.1.50"), PolicyConfirmedPrivate); err != nil {
		t.Errorf("ConfirmedPrivate 應放行 LAN: %v", err)
	}
	if err := screenIP(net.ParseIP("127.0.0.1"), PolicyConfirmedPrivate); err != nil {
		t.Errorf("ConfirmedPrivate 應放行 loopback: %v", err)
	}
	for _, bad := range []string{"169.254.169.254", "168.63.129.16", "100.64.0.1"} {
		if err := screenIP(net.ParseIP(bad), PolicyConfirmedPrivate); asBlockErr(err) == nil {
			t.Errorf("ConfirmedPrivate 仍應封鎖 %s", bad)
		}
	}
	if err := ValidateURL("http://192.168.1.50:8080/v1", PolicyConfirmedPrivate); err != nil {
		t.Errorf("ValidateURL ConfirmedPrivate 應放行 LAN http: %v", err)
	}
}

// SEC-05 2b: 外部開頁檢查——loopback 放行（monitor 頁），metadata 擋。
func TestScreenExternalOpenTarget(t *testing.T) {
	for _, ok := range []string{"127.0.0.1", "localhost", "example.com", "192.168.1.20"} {
		if err := ScreenExternalOpenTarget(ok); err != nil {
			t.Errorf("ScreenExternalOpenTarget(%q) 不應封鎖: %v", ok, err)
		}
	}
	for _, bad := range []string{"metadata.google.internal", "169.254.169.254", "168.63.129.16"} {
		if err := ScreenExternalOpenTarget(bad); asBlockErr(err) == nil {
			t.Errorf("ScreenExternalOpenTarget(%q) 應封鎖", bad)
		}
	}
}

func mustHost(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u.Host
}
