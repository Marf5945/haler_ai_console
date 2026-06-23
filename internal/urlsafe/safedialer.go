// Package urlsafe — Safe Dialer（SEC-05 Phase 1）。
//
// 把 SSRF 防護收斂到單一網路層：webhook_sender、connection tester、
// 未來 Safe Fetcher 都只向 NewSafeClient 要一個「安全的 *http.Client」，
// 不再各自補檢查。核心防線在「連線當下檢查 resolved IP」，
// 因此 DNS rebinding、網域指向內網、URL 編碼繞法（八進位/十六進位/
// IPv4-mapped IPv6）都會先被 resolver 正規化成真正 IP 再篩，自動免疫。
package urlsafe

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxRedirects 限制 redirect 跳數（含原始請求外的轉址）。
const maxRedirects = 3

// ── 封鎖原因（固定 enum，方便日後聚合/告警，勿改成自由文字） ──

type BlockReason string

const (
	ReasonLinkLocal    BlockReason = "blocked_link_local"
	ReasonMetadataIP   BlockReason = "blocked_metadata_ip"
	ReasonMetadataHost BlockReason = "blocked_metadata_host"
	ReasonPrivateIP    BlockReason = "blocked_private_ip"
	ReasonLoopback     BlockReason = "blocked_loopback"
	ReasonReservedIP   BlockReason = "blocked_reserved_ip"
	ReasonCrossOrigin  BlockReason = "cross_host_redirect"
	ReasonTooManyHops  BlockReason = "too_many_redirects"
)

// BlockError 是底層可診斷錯誤：保留原因 + 具體標的，UI 層再轉友善訊息。
type BlockError struct {
	Reason BlockReason
	Host   string
	IP     string
}

func (e *BlockError) Error() string {
	if e.IP != "" {
		return fmt.Sprintf("urlsafe: %s (ip=%s)", e.Reason, e.IP)
	}
	return fmt.Sprintf("urlsafe: %s (host=%s)", e.Reason, e.Host)
}

// asBlockErr 取出底層 *BlockError（非 BlockError 則回 nil）。
func asBlockErr(err error) *BlockError {
	var be *BlockError
	if errors.As(err, &be) {
		return be
	}
	return nil
}

// ── 危險 IP / 主機名定義（ValidateURL 與 Dialer 共用，避免漂移） ──

// metadataIPs 雲端 metadata endpoint（link-local 之外的）。
var metadataIPs = []net.IP{
	net.ParseIP("169.254.169.254"), // AWS / GCP / Azure IMDS（同時也是 link-local）
	net.ParseIP("100.100.100.200"), // 阿里雲（同時落在 CGNAT 段）
	net.ParseIP("192.0.0.192"),     // Oracle Cloud
	net.ParseIP("fd00:ec2::254"),   // AWS IPv6 IMDS
	net.ParseIP("168.63.129.16"),   // Azure WireServer（不在 link-local/private 段，需明列）
}

// metadataHostnames 永久封鎖的 metadata 主機名（無 dev 例外）。
var metadataHostnames = map[string]bool{
	"metadata.google.internal": true,
	"metadata.goog":            true,
}

var cgnatNet = mustCIDR("100.64.0.0/10") // RFC 6598 CGNAT，含阿里雲 metadata

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

func isMetadataIP(ip net.IP) bool {
	for _, m := range metadataIPs {
		if m.Equal(ip) {
			return true
		}
	}
	return false
}

// unconditionalIPBlock 回傳「所有 policy（含 dev）一律封鎖」的危險 IP 判斷。
// 這是最容易漂移的規則集合，集中一份給 ValidateURL 與 screenIP 共用。
func unconditionalIPBlock(ip net.IP) (BlockReason, bool) {
	if v4 := ip.To4(); v4 != nil {
		ip = v4 // 正規化 IPv4-mapped，避免在 v6 分支漏判
	}
	switch {
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return ReasonLinkLocal, true
	case isMetadataIP(ip):
		return ReasonMetadataIP, true
	case cgnatNet.Contains(ip):
		return ReasonReservedIP, true
	case ip.IsUnspecified(), ip.IsMulticast():
		return ReasonReservedIP, true
	}
	return "", false
}

// screenHostname 主機名硬封鎖（metadata 主機名，所有 policy）。
func screenHostname(host string) error {
	h := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if metadataHostnames[h] {
		return &BlockError{Reason: ReasonMetadataHost, Host: h}
	}
	return nil
}

// screenIP 是連線當下對 resolved IP 的篩選：
// 先過 unconditionalIPBlock，再依 policy 決定 loopback / private 是否放行。
func screenIP(ip net.IP, policy URLPolicy) error {
	if reason, blocked := unconditionalIPBlock(ip); blocked {
		return &BlockError{Reason: reason, IP: ip.String()}
	}
	allowLoopback, allowPrivate := policyAllowance(policy)
	if ip.IsLoopback() {
		if allowLoopback {
			return nil
		}
		return &BlockError{Reason: ReasonLoopback, IP: ip.String()}
	}
	if ip.IsPrivate() {
		if allowPrivate {
			return nil
		}
		return &BlockError{Reason: ReasonPrivateIP, IP: ip.String()}
	}
	return nil
}

// policyAllowance 把 policy 映成兩個獨立放行旗標（單一事實來源）。
func policyAllowance(p URLPolicy) (allowLoopback, allowPrivate bool) {
	switch p {
	case PolicyLocalLLM: // 本機 LLM：允許 loopback，仍擋 LAN
		return true, false
	case PolicyWebhookDev, PolicyConfirmedPrivate: // 開發模式 / 已確認 private：允許 loopback + private
		return true, true
	default: // PolicyCloudAPI / PolicyWebhook：僅公網
		return false, false
	}
}

func policyName(p URLPolicy) string {
	switch p {
	case PolicyCloudAPI:
		return "cloud_api"
	case PolicyLocalLLM:
		return "local_llm"
	case PolicyWebhook:
		return "webhook"
	case PolicyWebhookDev:
		return "webhook_dev"
	case PolicyConfirmedPrivate:
		return "confirmed_private"
	}
	return "unknown"
}

// ── 安全 log：只記決策 metadata，不記 URL query / headers / body ──

// BlockLogf 可由外層覆寫接既有 audit log；預設輸出標準 log。
var BlockLogf = func(format string, args ...any) { log.Printf(format, args...) }

func logSafeBlock(operation string, policy URLPolicy, host, port string, be *BlockError, hops int) {
	if be == nil {
		return
	}
	BlockLogf("event=safe_http_blocked operation=%s policy=%s host=%s port=%s decision=blocked reason=%s ip=%s redirect_hops=%d",
		operation, policyName(policy), host, port, be.Reason, be.IP, hops)
}

// ── Safe Dialer / Client ──

// ipResolver 只需要一個方法；net.DefaultResolver 天生滿足，測試可注入 fake。
type ipResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type safeDialer struct {
	policy    URLPolicy
	operation string
	resolver  ipResolver // nil → net.DefaultResolver
	dialer    *net.Dialer
}

// DialContext 解析主機 → 篩每個 IP（任一危險就拒）→ 只撥已驗證的 IP。
// 直接撥 IP literal 可避免 net.Dialer 再解析一次，消除 TOCTOU 窗口。
func (d *safeDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if err := screenHostname(host); err != nil {
		logSafeBlock(d.operation, d.policy, host, port, asBlockErr(err), 0)
		return nil, err
	}

	res := d.resolver
	if res == nil {
		res = net.DefaultResolver
	}
	ips, err := res.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("urlsafe: 主機 %q 無法解析出 IP", host)
	}

	for _, ipa := range ips {
		if err := screenIP(ipa.IP, d.policy); err != nil {
			logSafeBlock(d.operation, d.policy, host, port, asBlockErr(err), 0)
			return nil, err
		}
	}

	var lastErr error
	for _, ipa := range ips {
		conn, derr := d.dialer.DialContext(ctx, network, net.JoinHostPort(ipa.IP.String(), port))
		if derr == nil {
			return conn, nil
		}
		lastErr = derr
	}
	return nil, lastErr
}

// NewSafeClient 依 policy 產生帶 Safe Dialer 與 same-origin redirect 檢查的 client。
// operation 只用於安全 log（如 "webhook_send" / "connection_test"）。
func NewSafeClient(policy URLPolicy, operation string, timeout time.Duration) *http.Client {
	return newSafeClientWithResolver(policy, operation, timeout, nil)
}

// newSafeClientWithResolver 同上但可注入 resolver，供測試模擬 DNS rebinding。
func newSafeClientWithResolver(policy URLPolicy, operation string, timeout time.Duration, resolver ipResolver) *http.Client {
	sd := &safeDialer{
		policy:    policy,
		operation: operation,
		resolver:  resolver,
		dialer:    &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second},
	}
	transport := &http.Transport{
		DialContext:           sd.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Timeout:       timeout,
		Transport:     transport,
		CheckRedirect: makeCheckRedirect(policy, operation),
	}
}

// makeCheckRedirect：每跳先過 policy（擋 http 降級），再限 same-origin + 3 跳。
// same-origin 跨 host 一律拒，避免換 host 洩漏自訂 token 或繞道內網。
func makeCheckRedirect(policy URLPolicy, operation string) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			be := &BlockError{Reason: ReasonTooManyHops, Host: req.URL.Hostname()}
			logSafeBlock(operation, policy, req.URL.Hostname(), effectivePort(req.URL), be, len(via))
			return be
		}
		if err := ValidateURL(req.URL.String(), policy); err != nil {
			return err
		}
		prev := via[len(via)-1].URL
		if !sameOrigin(prev, req.URL) {
			be := &BlockError{Reason: ReasonCrossOrigin, Host: req.URL.Hostname()}
			logSafeBlock(operation, policy, req.URL.Hostname(), effectivePort(req.URL), be, len(via))
			return be
		}
		return nil
	}
}

// effectivePort 補出預設埠，讓 https://a.com 與 https://a.com:443 視為同 origin。
func effectivePort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

// sameOrigin 比 scheme + hostname + effectivePort（比單比 host 語意更清楚）。
func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Hostname(), b.Hostname()) &&
		effectivePort(a) == effectivePort(b)
}
