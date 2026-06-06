// Package securitytext contains small, dependency-free helpers for security
// string matching across architectural layers.
package securitytext

import (
	"strings"
	"unicode"
)

// NormalizeForSecurityCheck collapses whitespace before security-sensitive
// string matching. This prevents bypasses such as Bearer<TAB>TOKEN while
// keeping callers independent from domain packages.
func NormalizeForSecurityCheck(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
