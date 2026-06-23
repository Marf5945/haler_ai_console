// memory/redaction.go — 機密遮蔽引擎（§3.4 / §18.3）。
// Layer 1: 已知供應商前綴比對。Layer 2: Shannon entropy 異常偵測。
// 遮蔽 log 只記類型/信心度/來源/時間/hash，不保存原始值。
package memory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ui_console/internal/securitytext"
)

// ──────────────────────────────────────────────
// 信心度 & 記錄結構
// ──────────────────────────────────────────────

// RedactionConfidence 遮蔽信心度等級（§3.4.3）。
type RedactionConfidence string

const (
	ConfHigh   RedactionConfidence = "high"   // Layer 1 pattern 命中
	ConfMedium RedactionConfidence = "medium" // Layer 2 entropy + 上下文關鍵字
	ConfLow    RedactionConfidence = "low"    // Layer 2 entropy 無上下文
)

// RedactionRecord 一次遮蔽操作的元資料（不含原始值）。
type RedactionRecord struct {
	Type       string              `json:"type"`
	Field      string              `json:"field"`
	Source     string              `json:"source"`
	Timestamp  string              `json:"timestamp"`
	Hash       string              `json:"hash"`
	Confidence RedactionConfidence `json:"confidence"`
	Layer      int                 `json:"layer"` // 1 or 2
	Entropy    float64             `json:"entropy,omitempty"`
}

// ──────────────────────────────────────────────
// Layer 1 — 已知前綴 pattern（§3.4.2）
// ──────────────────────────────────────────────

type redactionRule struct {
	provider    string         // 供應商名
	fieldName   string         // 欄位名
	pattern     *regexp.Regexp // 比對正則
	replacement string         // 遮蔽標記
}

// 內建基線——編譯時常數，不可移除（§3.4.4）。
var builtinRules = []redactionRule{
	{provider: "OpenAI", fieldName: "api_key",
		pattern:     regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9]{20,}`),
		replacement: "[REDACTED:pattern:OpenAI]"},
	{provider: "Anthropic", fieldName: "api_key",
		pattern:     regexp.MustCompile(`(?i)\bsk-ant-[A-Za-z0-9\-]{20,}`),
		replacement: "[REDACTED:pattern:Anthropic]"},
	{provider: "OpenRouter", fieldName: "api_key",
		pattern:     regexp.MustCompile(`(?i)\bsk-or-v1-[A-Za-z0-9]{20,}`),
		replacement: "[REDACTED:pattern:OpenRouter]"},
	{provider: "Replicate", fieldName: "api_key",
		pattern:     regexp.MustCompile(`\br8_[A-Za-z0-9]{20,}`),
		replacement: "[REDACTED:pattern:Replicate]"},
	{provider: "GitHub_PAT", fieldName: "token",
		pattern:     regexp.MustCompile(`\bgh[ps]_[A-Za-z0-9]{36,}`),
		replacement: "[REDACTED:pattern:GitHub_PAT]"},
	{provider: "GitHub_OAuth", fieldName: "token",
		pattern:     regexp.MustCompile(`\bgho_[A-Za-z0-9]{36,}`),
		replacement: "[REDACTED:pattern:GitHub_OAuth]"},
	{provider: "GitHub_App", fieldName: "token",
		pattern:     regexp.MustCompile(`\b(ghu|ghs|ghr)_[A-Za-z0-9]{36,}`),
		replacement: "[REDACTED:pattern:GitHub_App]"},
	{provider: "GitLab", fieldName: "token",
		pattern:     regexp.MustCompile(`\bglpat-[A-Za-z0-9]{20,}`),
		replacement: "[REDACTED:pattern:GitLab]"},
	{provider: "Google_AI", fieldName: "api_key",
		pattern:     regexp.MustCompile(`\bAIza[A-Za-z0-9\-_]{35}`),
		replacement: "[REDACTED:pattern:Google_AI]"},
	{provider: "AWS", fieldName: "access_key",
		pattern:     regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`),
		replacement: "[REDACTED:pattern:AWS]"},
	{provider: "Stripe", fieldName: "api_key",
		pattern:     regexp.MustCompile(`\b(sk|pk)_(test|live)_[A-Za-z0-9]{24,}`),
		replacement: "[REDACTED:pattern:Stripe]"},
	{provider: "HuggingFace", fieldName: "token",
		pattern:     regexp.MustCompile(`\bhf_[A-Za-z0-9]{34,}`),
		replacement: "[REDACTED:pattern:HuggingFace]"},
	// Bearer token — \s+ 搭配 NormalizeForSecurityCheck 覆蓋 tab/多空格
	{provider: "Bearer", fieldName: "authorization",
		pattern:     regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`),
		replacement: "[REDACTED:pattern:Bearer]"},
	// PEM 私鑰
	{provider: "PEM", fieldName: "private_key",
		pattern:     regexp.MustCompile(`(?s)-----BEGIN\s+[\w\s]*PRIVATE\s+KEY-----.*?-----END\s+[\w\s]*PRIVATE\s+KEY-----`),
		replacement: "[REDACTED:pattern:PEM]"},
}

// activeRules = builtinRules + user extension（啟動時合併）。
var activeRules []redactionRule

func init() { activeRules = append([]redactionRule{}, builtinRules...) }

// ──────────────────────────────────────────────
// Layer 2 — Shannon Entropy（§3.4.2）
// ──────────────────────────────────────────────

const (
	entropyWindow    = 40  // 滑動視窗寬度
	entropyThreshold = 4.5 // bits/byte 閾值
)

// 上下文關鍵字——高熵 + 有這些關鍵字 → medium confidence。
var contextKeywords = []string{
	"key", "token", "secret", "password",
	"credential", "authorization", "bearer", "api",
}

// 誤報豁免 pattern。
var entropyExemptions = []*regexp.Regexp{
	regexp.MustCompile(`!\[.*?\]\(.*?\)`),                                              // markdown image
	regexp.MustCompile(`data:image/[a-z]+;base64,`),                                    // base64 image
	regexp.MustCompile(`(?i)\bcommit\s+[0-9a-f]{40}\b`),                                // git SHA
	regexp.MustCompile(`(?i)(sha256|sha1|md5|sha512):\s*[0-9a-f]+`),                    // labeled hash
	regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), // UUID
}

// ShannonEntropy 計算 byte 分佈的 Shannon entropy。
func ShannonEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	freq := make(map[byte]int, 256)
	for _, b := range data {
		freq[b]++
	}
	n := float64(len(data))
	h := 0.0
	for _, count := range freq {
		p := float64(count) / n
		if p > 0 {
			h -= p * math.Log2(p)
		}
	}
	return h
}

// entropyCandidate 高熵候選區間。
type entropyCandidate struct {
	start, end int
	entropy    float64
}

// detectHighEntropy 掃描文字，回傳合併後的高熵區間。
func detectHighEntropy(text string) []entropyCandidate {
	if len(text) < entropyWindow {
		return nil
	}
	b := []byte(text)
	var raw []entropyCandidate
	for i := 0; i <= len(b)-entropyWindow; i++ {
		h := ShannonEntropy(b[i : i+entropyWindow])
		if h >= entropyThreshold {
			raw = append(raw, entropyCandidate{start: i, end: i + entropyWindow, entropy: h})
		}
	}
	// 合併相鄰/重疊區間
	return mergeCandidates(raw)
}

// mergeCandidates 合併重疊候選區間。
func mergeCandidates(cs []entropyCandidate) []entropyCandidate {
	if len(cs) == 0 {
		return nil
	}
	merged := []entropyCandidate{cs[0]}
	for _, c := range cs[1:] {
		last := &merged[len(merged)-1]
		if c.start <= last.end {
			if c.end > last.end {
				last.end = c.end
			}
			if c.entropy > last.entropy {
				last.entropy = c.entropy
			}
		} else {
			merged = append(merged, c)
		}
	}
	return merged
}

// isExempt 檢查候選區間是否符合豁免條件。
func isExempt(text string, start, end int) bool {
	// 擴展上下文範圍用於豁免檢查
	ctxStart := start - 30
	if ctxStart < 0 {
		ctxStart = 0
	}
	ctxEnd := end + 30
	if ctxEnd > len(text) {
		ctxEnd = len(text)
	}
	ctx := text[ctxStart:ctxEnd]
	for _, re := range entropyExemptions {
		if re.MatchString(ctx) {
			return true
		}
	}
	return false
}

// hasContextKeyword 檢查候選區間前後是否有上下文關鍵字。
func hasContextKeyword(text string, start, end int) bool {
	ctxStart := start - 40
	if ctxStart < 0 {
		ctxStart = 0
	}
	ctxEnd := end + 20
	if ctxEnd > len(text) {
		ctxEnd = len(text)
	}
	lower := strings.ToLower(text[ctxStart:ctxEnd])
	for _, kw := range contextKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// Key=Value 通用比對
// ──────────────────────────────────────────────

var kvPattern = regexp.MustCompile(
	`(?i)(password|secret|token|api_key|apikey|credential|auth_token|access_token|secret_key)\s*[=:]\s*["']?(\S{8,})["']?`)

// ──────────────────────────────────────────────
// 主遮蔽入口（§3.4）
// ──────────────────────────────────────────────

// RedactBeforeWrite 寫入記憶前執行遮蔽。
// 先正規化（§3.4.1），再 Layer 1 + Layer 2 掃描，回傳清理後內容 + 記錄。
func RedactBeforeWrite(content string) (string, []RedactionRecord) {
	var records []RedactionRecord
	now := time.Now().Format(time.RFC3339)

	// §3.4.1 正規化（工作副本用於比對，原文用於替換定位）
	normalized := securitytext.NormalizeForSecurityCheck(content)

	// ── Layer 1: pattern 比對 ──
	result := normalized
	for _, rule := range activeRules {
		matches := rule.pattern.FindAllString(result, -1)
		for _, m := range matches {
			records = append(records, RedactionRecord{
				Type:       rule.provider,
				Field:      rule.fieldName,
				Source:     "write_pipeline",
				Timestamp:  now,
				Hash:       hashValue(m),
				Confidence: ConfHigh,
				Layer:      1,
			})
		}
		result = rule.pattern.ReplaceAllString(result, rule.replacement)
	}

	// Layer 1: key=value 通用比對
	result = kvPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := kvPattern.FindStringSubmatch(match)
		if len(parts) >= 3 {
			records = append(records, RedactionRecord{
				Type:       "key_value",
				Field:      strings.ToLower(parts[1]),
				Source:     "write_pipeline",
				Timestamp:  now,
				Hash:       hashValue(parts[2]),
				Confidence: ConfHigh,
				Layer:      1,
			})
			return fmt.Sprintf("%s=[REDACTED:pattern:generic_kv]", parts[1])
		}
		return match
	})

	// ── Layer 2: entropy 掃描（僅掃 Layer 1 替換後殘餘文字）──
	candidates := detectHighEntropy(result)
	// 從後往前替換避免 offset 偏移
	for i := len(candidates) - 1; i >= 0; i-- {
		c := candidates[i]
		if c.start >= len(result) || c.end > len(result) {
			continue
		}
		span := result[c.start:c.end]

		// 已被 Layer 1 處理的區間跳過
		if strings.Contains(span, "[REDACTED:") {
			continue
		}
		// 豁免檢查
		if isExempt(result, c.start, c.end) {
			continue
		}

		if hasContextKeyword(result, c.start, c.end) {
			// medium confidence → 直接遮蔽
			records = append(records, RedactionRecord{
				Type:       "entropy+context",
				Field:      "unknown",
				Source:     "write_pipeline",
				Timestamp:  now,
				Hash:       hashValue(span),
				Confidence: ConfMedium,
				Layer:      2,
				Entropy:    c.entropy,
			})
			result = result[:c.start] + "[REDACTED:entropy+context]" + result[c.end:]
		} else {
			// low confidence → 標記保留
			records = append(records, RedactionRecord{
				Type:       "entropy_suspect",
				Field:      "unknown",
				Source:     "write_pipeline",
				Timestamp:  now,
				Hash:       hashValue(span),
				Confidence: ConfLow,
				Layer:      2,
				Entropy:    c.entropy,
			})
			result = result[:c.end] + "[SUSPECT:entropy]" + result[c.end:]
		}
	}

	return result, records
}

// ──────────────────────────────────────────────
// 使用者擴充 pattern 載入（§3.4.4）
// ──────────────────────────────────────────────

// UserPattern 使用者自訂 pattern（JSON schema）。
type UserPattern struct {
	Provider string `json:"provider"`
	Pattern  string `json:"pattern"`
	Enabled  bool   `json:"enabled"`
}

// LoadUserPatterns 從 JSON 載入使用者 pattern，與內建合併。
// 檔案不存在或損壞 → 僅用內建，不 crash。
func LoadUserPatterns(configDir string) {
	path := filepath.Join(configDir, "redaction_patterns.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return // 檔案不存在，靜默使用內建
	}

	var patterns []UserPattern
	if err := json.Unmarshal(data, &patterns); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] redaction_patterns.json 解析失敗: %v\n", err)
		return
	}

	for _, p := range patterns {
		if !p.Enabled {
			continue
		}
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] 跳過無效 regex (provider=%s): %v\n", p.Provider, err)
			continue
		}
		// 檢查是否與內建重複
		dup := false
		for _, b := range builtinRules {
			if b.pattern.String() == re.String() {
				dup = true
				break
			}
		}
		if !dup {
			activeRules = append(activeRules, redactionRule{
				provider:    p.Provider,
				fieldName:   "user_defined",
				pattern:     re,
				replacement: fmt.Sprintf("[REDACTED:pattern:%s]", p.Provider),
			})
		}
	}
}

// ResetActiveRules 重設為內建基線（測試用）。
func ResetActiveRules() {
	activeRules = append([]redactionRule{}, builtinRules...)
}

// ──────────────────────────────────────────────
// 輔助函式
// ──────────────────────────────────────────────

// hashValue SHA-256 前 8 bytes（供驗證，不保存原值）。
func hashValue(value string) string {
	h := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", h[:8])
}
