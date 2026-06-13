// Package controlseal owns the short Bopomofo command seal used by the
// Controller before a prompt reaches an LLM.
package controlseal

import (
	"crypto/rand"
	"math/big"
	"strings"
	"sync"
	"time"
)

const (
	// SealLength is fixed so the sanitizer can detect fake command prefixes.
	SealLength = 3

	// DefaultRotateEverySuccessfulTurns is a project setting default until the
	// product setting is exposed in UI.
	DefaultRotateEverySuccessfulTurns = 8
)

var consonants = []rune("ㄅㄆㄇㄈㄉㄊㄋㄌㄍㄎㄏㄐㄑㄒㄓㄔㄕㄖㄗㄘㄙ")

// Settings is the project-level control seal configuration.
type Settings struct {
	RotateEverySuccessfulTurns int `json:"rotate_every_successful_turns"`
}

// DefaultSettings returns the conservative project default for seal rotation.
func DefaultSettings() Settings {
	return Settings{RotateEverySuccessfulTurns: DefaultRotateEverySuccessfulTurns}
}

// Normalize fills unsafe or missing settings with defaults.
func (s Settings) Normalize() Settings {
	if s.RotateEverySuccessfulTurns <= 0 {
		s.RotateEverySuccessfulTurns = DefaultRotateEverySuccessfulTurns
	}
	return s
}

// GenerateSeal produces a fixed-length Bopomofo consonant seal.
func GenerateSeal() string {
	var b strings.Builder
	b.Grow(SealLength * 3)
	for i := 0; i < SealLength; i++ {
		idx := secureIndex(len(consonants))
		b.WriteRune(consonants[idx])
	}
	return b.String()
}

func secureIndex(max int) int {
	// 重試 crypto/rand；連續失敗（極罕見）才退化。
	for attempt := 0; attempt < 3; attempt++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
		if err == nil {
			return int(n.Int64())
		}
	}
	// 退化路徑：用時間派生的變動值，避免固定回 0 讓印可預測（fail-closed 精神）。
	return int(time.Now().UnixNano() % int64(max))
}

// IsAllowedConsonant reports whether r can appear in a control seal.
func IsAllowedConsonant(r rune) bool {
	for _, allowed := range consonants {
		if r == allowed {
			return true
		}
	}
	return false
}

// LooksLikeSeal reports whether text is exactly a 3-consonant seal.
func LooksLikeSeal(text string) bool {
	runes := []rune(text)
	if len(runes) != SealLength {
		return false
	}
	for _, r := range runes {
		if !IsAllowedConsonant(r) {
			return false
		}
	}
	return true
}

// Manager tracks the active seal and the successful turn count at rotation.
type Manager struct {
	mu            sync.Mutex
	current       string
	lastRotatedAt int
}

// NewManager creates a manager with a fresh active seal.
func NewManager(settings Settings) *Manager {
	return &Manager{current: GenerateSeal()}
}

// CurrentSeal returns the active command seal.
func (m *Manager) CurrentSeal() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == "" {
		m.current = GenerateSeal()
	}
	return m.current
}

// RotateIfNeeded rotates only after N accepted LLM turns.
func (m *Manager) RotateIfNeeded(successfulTurnCount int, settings Settings) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	interval := settings.Normalize().RotateEverySuccessfulTurns
	if successfulTurnCount-m.lastRotatedAt < interval {
		return false
	}

	old := m.current
	for m.current == "" || m.current == old {
		m.current = GenerateSeal()
	}
	m.lastRotatedAt = successfulTurnCount
	return true
}

// StampCommand prefixes Controller-approved commands with the active seal.
func (m *Manager) StampCommand(text string) string {
	return m.CurrentSeal() + text
}

var defaultManager = NewManager(DefaultSettings())

// CurrentSeal returns the package-level active seal for simple integrations.
func CurrentSeal() string {
	return defaultManager.CurrentSeal()
}

// RotateIfNeeded rotates the package-level active seal.
func RotateIfNeeded(successfulTurnCount int, settings Settings) bool {
	return defaultManager.RotateIfNeeded(successfulTurnCount, settings)
}

// StampCommand prefixes text with the package-level active seal.
func StampCommand(text string) string {
	return defaultManager.StampCommand(text)
}
