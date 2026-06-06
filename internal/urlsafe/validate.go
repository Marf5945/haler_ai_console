// Package urlsafe 提供 URL 安全驗證，防止 SSRF 攻擊。
// 依據不同用途（雲端 API / 本機 LLM / Webhook）套用不同策略。
// SEC-03 / SEC-04: 共用 ValidateURL，依 URLPolicy 分策略。
package urlsafe

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLPolicy 定義 URL 驗證策略。
type URLPolicy int

const (
	// PolicyCloudAPI 雲端 LLM API：僅 https，封鎖所有 private/localhost。
	PolicyCloudAPI URLPolicy = iota
	// PolicyLocalLLM 本機 LLM（Ollama 等）：允許 http://localhost 與 127.0.0.1，封鎖 LAN。
	PolicyLocalLLM
	// PolicyWebhook Webhook（正式模式）：僅 https 公網。
	PolicyWebhook
	// PolicyWebhookDev Webhook（開發模式）：允許 localhost/private，需前端確認。
	PolicyWebhookDev
	// PolicyConfirmedPrivate 使用者已確認的 private 自架 endpoint（SEC-03 確認流程）：
	// 允許 http/https、localhost、private LAN；metadata/link-local/CGNAT 仍永久封鎖。
	PolicyConfirmedPrivate
)

var (
	ErrEmptyURL        = errors.New("urlsafe: URL 不得為空")
	ErrInvalidURL      = errors.New("urlsafe: URL 格式不正確")
	ErrSchemeNotHTTPS  = errors.New("urlsafe: 此用途僅允許 https")
	ErrPrivateIP       = errors.New("urlsafe: 不允許 private/internal IP 位址")
	ErrLocalhostDenied = errors.New("urlsafe: 此用途不允許 localhost")
	ErrLANDenied       = errors.New("urlsafe: LAN 位址需進階確認，目前不允許")
	ErrLinkLocal       = errors.New("urlsafe: link-local 位址不允許")
	ErrNoHost          = errors.New("urlsafe: URL 缺少 host")
)

// ValidateURL 依據 policy 驗證 URL 是否安全。
// 回傳 nil 代表通過，否則回傳具體的拒絕原因。
func ValidateURL(rawURL string, policy URLPolicy) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ErrEmptyURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	hostname := parsed.Hostname() // 去掉 port 和 []

	if hostname == "" {
		return ErrNoHost
	}

	// metadata 主機名硬封鎖（所有 policy，無 dev 例外）；與 Safe Dialer 共用名單。
	if err := screenHostname(hostname); err != nil {
		return err
	}

	// — Scheme 檢查 —
	switch policy {
	case PolicyCloudAPI, PolicyWebhook:
		// 嚴格模式：僅 https
		if scheme != "https" {
			return ErrSchemeNotHTTPS
		}
	case PolicyLocalLLM:
		// 允許 http（本機）和 https（遠端）
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("urlsafe: 僅允許 http 或 https，收到 %q", scheme)
		}
	case PolicyWebhookDev, PolicyConfirmedPrivate:
		// 開發模式 / 已確認 private：允許 http 和 https
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("urlsafe: 僅允許 http 或 https，收到 %q", scheme)
		}
	}

	// — Host 分類 —
	isLocalhost := isLocalhostHost(hostname)
	ip := net.ParseIP(hostname)

	// 如果 hostname 不是 IP，嘗試判斷是否為 localhost 別名
	if ip == nil && !isLocalhost {
		// 非 IP、非 localhost 的 domain name → 根據 scheme 已驗證，放行
		return nil
	}

	// — Localhost 判斷 —
	if isLocalhost || (ip != nil && ip.IsLoopback()) {
		switch policy {
		case PolicyCloudAPI, PolicyWebhook:
			return ErrLocalhostDenied
		case PolicyLocalLLM, PolicyWebhookDev, PolicyConfirmedPrivate:
			return nil // 允許
		}
	}

	// — IP 位址細分 —
	if ip != nil {
		// 共用 unconditionalIPBlock：link-local / metadata IP / CGNAT / 未指定 /
		// multicast 一律拒絕（含 dev）。與 Safe Dialer 同一份定義，避免漂移。
		if reason, blocked := unconditionalIPBlock(ip); blocked {
			return &BlockError{Reason: reason, IP: ip.String()}
		}

		if isPrivateIP(ip) {
			switch policy {
			case PolicyCloudAPI, PolicyWebhook:
				return ErrPrivateIP
			case PolicyLocalLLM:
				// LAN IP 在本機 LLM 模式下也封鎖（需進階確認流程）
				return ErrLANDenied
			case PolicyWebhookDev, PolicyConfirmedPrivate:
				return nil // 開發模式 / 已確認 private 允許
			}
		}
	}

	return nil
}

// isLocalhostHost 判斷 hostname 是否為 localhost 及其常見別名。
func isLocalhostHost(hostname string) bool {
	h := strings.ToLower(hostname)
	return h == "localhost" || h == "localhost." ||
		h == "127.0.0.1" || h == "::1" || h == "[::1]"
}

// isPrivateIP 判斷 IP 是否為 RFC 1918 / RFC 4193 private 位址。
// 不含 loopback（已由 IsLoopback 處理）。
func isPrivateIP(ip net.IP) bool {
	// net.IP.IsPrivate() 在 Go 1.17+ 可用，涵蓋：
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7
	return ip.IsPrivate()
}

// --- SEC-03: LLM API baseURL 驗證 ---

// localLLMProviders 本機 LLM providerID，允許 http://localhost。
var localLLMProviders = map[string]bool{
	"ollama":   true,
	"lmstudio": true,
}

// ValidateLLMBaseURL 依 providerID 選策略驗證 LLM API baseURL。
// ollama/lmstudio → PolicyLocalLLM（允許 localhost http）
// 其他 provider → PolicyCloudAPI（僅 https、禁 private IP）
// 回傳 (needConfirm, hostname, error)：
//   needConfirm=true 代表 private IP 需前端確認框。
func ValidateLLMBaseURL(providerID, baseURL string) (needConfirm bool, hostname string, err error) {
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	baseURL = strings.TrimSpace(baseURL)

	if baseURL == "" {
		return false, "", nil // 空 URL 由後續自動填入處理
	}

	parsed, pErr := url.Parse(baseURL)
	if pErr != nil {
		return false, "", fmt.Errorf("%w: %v", ErrInvalidURL, pErr)
	}
	hostname = parsed.Hostname()

	// 本機 LLM provider 使用寬鬆策略
	if localLLMProviders[providerID] {
		return false, hostname, ValidateURL(baseURL, PolicyLocalLLM)
	}

	// 雲端 provider 先用嚴格策略
	err = ValidateURL(baseURL, PolicyCloudAPI)
	if err == nil {
		return false, hostname, nil
	}

	// 若目標是 private IP / localhost，無論 scheme 錯誤都改為 needConfirm
	// （使用者可能用 http 連本機自架服務，需確認而非直接拒絕）
	if errors.Is(err, ErrPrivateIP) || errors.Is(err, ErrLocalhostDenied) || errors.Is(err, ErrLANDenied) {
		return true, hostname, nil
	}
	if errors.Is(err, ErrSchemeNotHTTPS) && (isLocalhostHost(hostname) || isPrivateHostname(hostname)) {
		return true, hostname, nil
	}

	return false, hostname, err // 公網 http 等直接拒絕
}


// PolicyForLLMEndpoint 集中決定 LLM / model endpoint 的 dial policy。
// 2a 所有呼叫點一律由此取 policy，不要在呼叫點各自判斷。
// 信任模型：cloud provider 的 private baseURL 能存在於設定中，必然走過
// SEC-03 前端確認流程（RegisterLLMAPIAdapter → need_confirm → Confirm...），
// 故此處視為已確認 → PolicyConfirmedPrivate（仍永擋 metadata/link-local/CGNAT）。
func PolicyForLLMEndpoint(providerID, baseURL string) URLPolicy {
	if localLLMProviders[strings.ToLower(strings.TrimSpace(providerID))] {
		return PolicyLocalLLM
	}
	if parsed, err := url.Parse(strings.TrimSpace(baseURL)); err == nil {
		h := parsed.Hostname()
		if isLocalhostHost(h) || isPrivateHostname(h) {
			return PolicyConfirmedPrivate
		}
	}
	return PolicyCloudAPI
}

// ScreenExternalOpenTarget 供「交給系統瀏覽器開頁」路徑（OpenExternalURL）使用：
// 只擋 metadata 主機名與無條件危險 IP 字面值；loopback/private 放行——
// 開瀏覽器看本機頁面（如 debug trace monitor）是合法用途，風險模型與 App 代抓不同。
func ScreenExternalOpenTarget(hostname string) error {
	if err := screenHostname(hostname); err != nil {
		return err
	}
	if ip := net.ParseIP(hostname); ip != nil {
		if reason, blocked := unconditionalIPBlock(ip); blocked {
			return &BlockError{Reason: reason, IP: ip.String()}
		}
	}
	return nil
}

// isPrivateHostname 判斷 hostname 是否為 private IP（不含 localhost，已由 isLocalhostHost 處理）。
func isPrivateHostname(hostname string) bool {
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback()
}
