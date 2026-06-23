// Package url_source — URL provenance registry（SEC-06 URL Capability Layer）。
//
// 所有 URL 入口（使用者貼上 / LLM 擷取 / remote bridge / skill manifest /
// web search 結果）都進同一條管線：每次「URL 被發現」記一筆 URLRecord
// （occurrence 模型）。查詢風險時以該 URL 出現過的「最低信任來源」為準，
// 避免 LLM 先提、再誘導使用者貼上同一 URL 的洗白攻擊。
//
// 持久化策略：記憶體 registry（session 內查詢）+ append-only JSONL audit。
// audit 只存 metadata（hash / host / source / ids），不存完整 URL——
// query string 可能含 token。
package url_source

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Source URL 的來源類別（固定 enum，新來源需評估 TrustRank）。
type Source string

const (
	SourceUserPaste       Source = "user_paste"
	SourceLLMExtracted    Source = "llm_extracted"
	SourceRemoteBridge    Source = "remote_bridge"
	SourceSkillManifest   Source = "skill_manifest"
	SourceWebSearchResult Source = "web_search_result"
)

// ValidSource 檢查來源字串是否為已知 enum。
func ValidSource(s string) (Source, bool) {
	src := Source(strings.TrimSpace(s))
	switch src {
	case SourceUserPaste, SourceLLMExtracted, SourceRemoteBridge,
		SourceSkillManifest, SourceWebSearchResult:
		return src, true
	}
	return "", false
}

// TrustRank 越小越不可信。用於「最低信任來源」查詢。
func (s Source) TrustRank() int {
	switch s {
	case SourceLLMExtracted:
		return 0 // 可能被 prompt injection 污染
	case SourceRemoteBridge:
		return 1
	case SourceSkillManifest:
		return 2
	case SourceWebSearchResult:
		return 3
	case SourceUserPaste:
		return 4
	}
	return 0 // 未知來源視為最不可信
}

// RiskTier 由來源推導的風險層級（high / medium / low）。
func (s Source) RiskTier() string {
	switch s.TrustRank() {
	case 0, 1:
		return "high"
	case 2, 3:
		return "medium"
	default:
		return "low"
	}
}

// URLRecord 一筆 URL occurrence。不存完整 URL（query 可能含 token）。
type URLRecord struct {
	URLHash          string    `json:"url_hash"`
	NormalizedHost   string    `json:"normalized_host"`
	Source           Source    `json:"source"`
	SessionID        string    `json:"session_id,omitempty"`
	TraceID          string    `json:"trace_id,omitempty"`
	SourceTrustLabel string    `json:"source_trust_label,omitempty"`
	RiskTier         string    `json:"risk_tier"`
	CreatedAt        time.Time `json:"created_at"`
}

// NormalizeURL 正規化供 hash：小寫 scheme/host、去 fragment、保留 path/query。
func NormalizeURL(rawURL string) (normalized, host string, err error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", fmt.Errorf("url_source: %w", err)
	}
	if parsed.Hostname() == "" {
		return "", "", fmt.Errorf("url_source: URL 缺少 host")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	return parsed.String(), parsed.Hostname(), nil
}

// HashURL 回傳正規化後 URL 的 sha256（hex 前 16 bytes，足夠去重且短）。
func HashURL(rawURL string) (hash, host string, err error) {
	normalized, host, err := NormalizeURL(rawURL)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", sum[:16]), host, nil
}

// Registry 記憶體 registry + JSONL audit writer。
type Registry struct {
	mu        sync.Mutex
	records   []URLRecord
	byHash    map[string][]int // url_hash → record indexes
	auditPath string           // 空字串 = 不落盤（測試用）
	maxInMem  int
}

// NewRegistry 建立 registry。auditPath 為 JSONL audit 檔（會自動建目錄）。
func NewRegistry(auditPath string) *Registry {
	return &Registry{
		byHash:    make(map[string][]int),
		auditPath: auditPath,
		maxInMem:  10000, // 記憶體上限，超過丟最舊（audit 檔不受影響）
	}
}

// Record 記一筆 URL occurrence；每次出現都記（occurrence 模型，不去重）。
func (r *Registry) Record(rawURL string, src Source, sessionID, traceID, trustLabel string) (URLRecord, error) {
	hash, host, err := HashURL(rawURL)
	if err != nil {
		return URLRecord{}, err
	}
	rec := URLRecord{
		URLHash:          hash,
		NormalizedHost:   host,
		Source:           src,
		SessionID:        sessionID,
		TraceID:          traceID,
		SourceTrustLabel: trustLabel,
		RiskTier:         src.RiskTier(),
		CreatedAt:        time.Now(),
	}

	r.mu.Lock()
	if len(r.records) >= r.maxInMem {
		r.evictOldestLocked()
	}
	r.records = append(r.records, rec)
	r.byHash[hash] = append(r.byHash[hash], len(r.records)-1)
	r.mu.Unlock()

	r.appendAudit(rec) // 失敗只忽略不阻斷（audit 是輔助，不能擋主流程）
	return rec, nil
}

// LowestTrustSource 回傳該 URL 出現過的最低信任來源。
// 安全語意：被低信任來源提過的 URL，即使之後使用者親貼，仍以低信任對待。
func (r *Registry) LowestTrustSource(rawURL string) (Source, bool) {
	hash, _, err := HashURL(rawURL)
	if err != nil {
		return "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	idxs, ok := r.byHash[hash]
	if !ok || len(idxs) == 0 {
		return "", false
	}
	lowest := r.records[idxs[0]].Source
	for _, i := range idxs[1:] {
		if r.records[i].Source.TrustRank() < lowest.TrustRank() {
			lowest = r.records[i].Source
		}
	}
	return lowest, true
}

// RecordsForSession 回傳某 session 最近的 occurrence（新→舊）。
func (r *Registry) RecordsForSession(sessionID string, limit int) []URLRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]URLRecord, 0, limit)
	for i := len(r.records) - 1; i >= 0 && len(out) < limit; i-- {
		if r.records[i].SessionID == sessionID {
			out = append(out, r.records[i])
		}
	}
	return out
}

// evictOldestLocked 丟掉最舊 10%（呼叫端持鎖）。
func (r *Registry) evictOldestLocked() {
	drop := r.maxInMem / 10
	if drop < 1 {
		drop = 1
	}
	r.records = append([]URLRecord(nil), r.records[drop:]...)
	r.byHash = make(map[string][]int, len(r.records))
	for i, rec := range r.records {
		r.byHash[rec.URLHash] = append(r.byHash[rec.URLHash], i)
	}
}

// appendAudit append-only JSONL；只含 metadata，不含完整 URL。
func (r *Registry) appendAudit(rec URLRecord) {
	if r.auditPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.auditPath), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(r.auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	line, err := json.Marshal(rec)
	if err != nil {
		return
	}
	_, _ = f.Write(append(line, '\n'))
}
