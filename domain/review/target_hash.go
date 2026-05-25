// review/target_hash.go — Deterministic TargetHashSet Builder（§4.6 補完）。
// 確保同一批目標無論順序如何都產生相同 hash，避免 false positive invalidation。
package review

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ──────────────────────────────────────────────
// TargetHashSet：deterministic hash builder
// ──────────────────────────────────────────────

// TargetEntry 描述一個操作目標（用於 hash 計算）。
type TargetEntry struct {
	Operation    string `json:"operation"`
	Target       string `json:"target"`
	AffectedScope string `json:"affected_scope"`
}

// ComputeTargetHashSet 對一組 TargetEntry 產生 deterministic hash。
// 步驟：正規化路徑 → 排序 → 拼接 → SHA-256。
// 同一批目標不同順序 hash 一致；內容改變 hash 不同。
func ComputeTargetHashSet(entries []TargetEntry) string {
	if len(entries) == 0 {
		return "empty_target"
	}

	// 正規化並組合為可排序字串
	normalized := make([]string, 0, len(entries))
	for _, e := range entries {
		norm := fmt.Sprintf("%s|%s|%s",
			strings.TrimSpace(e.Operation),
			normalizePath(e.Target),
			strings.TrimSpace(e.AffectedScope),
		)
		normalized = append(normalized, norm)
	}

	// 排序確保順序無關性
	sort.Strings(normalized)

	// 拼接後 SHA-256
	joined := strings.Join(normalized, "\n")
	sum := sha256.Sum256([]byte(joined))
	return fmt.Sprintf("%x", sum)
}

// normalizePath 正規化路徑（清除 ./ ../ trailing slash）。
func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	cleaned := filepath.Clean(p)
	return strings.TrimSuffix(cleaned, "/")
}
