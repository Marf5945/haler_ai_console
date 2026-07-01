// Package shortcode generates short, human-friendly random codes (e.g. the
// 6-char persona code in shared/settings/persona_code.go, and the per-photo
// code in data/album) with a caller-supplied uniqueness check.
package shortcode

import "crypto/rand"

// Alphabet excludes visually-confusable characters (0/O, 1/I).
const Alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

const defaultLength = 6
const maxAttempts = 50

// Generate returns a random code of defaultLength (6) characters that is
// not present in existing. existing may be nil.
func Generate(existing map[string]bool) string {
	return GenerateN(defaultLength, existing)
}

// GenerateN is like Generate but with a caller-chosen length.
func GenerateN(length int, existing map[string]bool) string {
	if length <= 0 {
		length = defaultLength
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		code := randomCode(length)
		if existing == nil || !existing[code] {
			return code
		}
	}
	// Astronomically unlikely with a 33-char alphabet at length >= 6, but
	// never spin forever: widen instead of looping indefinitely.
	return randomCode(length) + randomCode(length)
}

func randomCode(length int) string {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand practically never fails; fall back to a fixed
		// sequence and let the caller's uniqueness retry do the rest.
		for i := range buf {
			buf[i] = byte(i)
		}
	}
	out := make([]byte, length)
	for i, v := range buf {
		out[i] = Alphabet[int(v)%len(Alphabet)]
	}
	return string(out)
}
