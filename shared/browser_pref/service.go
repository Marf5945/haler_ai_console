// Package browser_pref implements the v3.3.2 P0.4 browser / global preference
// backend contract.
//
// Rules:
//   - browser chip in the top bar is READ-ONLY — it only shows current state.
//   - profile_path is auto-detected by default; manual path goes through validator.
//   - Selecting Safari triggers a one-time notice; every automation run needing
//     profile reuse shows a runtime notice.
//   - If profile reuse is blocked by the browser (e.g. Safari limitation), the
//     Draft Sandbox MUST treat this as an "authorization" stop signal.
//   - No dependency on browser preload.
package browser_pref

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ui_console/data/storage"
)

// BrowserKind identifies the selected browser.
type BrowserKind string

const (
	BrowserChrome  BrowserKind = "chrome"
	BrowserFirefox BrowserKind = "firefox"
	BrowserSafari  BrowserKind = "safari"
	BrowserEdge    BrowserKind = "edge"
)

// BrowserPreference stores the user's browser selection and profile path.
type BrowserPreference struct {
	Browser         BrowserKind `json:"browser"`
	ProfilePath     string      `json:"profile_path"`      // empty = auto-detect
	SafariNoticeSeen bool       `json:"safari_notice_seen"` // one-time notice flag
	UpdatedAt       time.Time  `json:"updated_at"`
}

// RuntimeNoticeResult communicates whether a Safari runtime notice is needed.
type RuntimeNoticeResult struct {
	ShowNotice bool   `json:"show_notice"`
	Reason     string `json:"reason,omitempty"`
}

// Service manages browser preference and validation.
type Service struct {
	mu      sync.Mutex
	store   *storage.JSONStore[BrowserPreference]
	current BrowserPreference
}

// NewService creates (or loads) a browser preference service.
func NewService(dataRoot string) *Service {
	svc := &Service{
		store: storage.NewJSONStore[BrowserPreference](
			filepath.Join(dataRoot, "data", "preferences", "browser_pref.json"),
		),
		current: BrowserPreference{
			Browser: BrowserChrome, // safe default
		},
	}
	if loaded, err := svc.store.Load(); err == nil && loaded.Browser != "" {
		svc.current = loaded
	}
	return svc
}

// Get returns the current browser preference. Read-only — use Set to change.
func (s *Service) Get() BrowserPreference {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// Set stores a new browser preference after validating the profile path.
// Returns a RuntimeNoticeResult for Safari one-time notice.
func (s *Service) Set(browser BrowserKind, profilePath string) (RuntimeNoticeResult, error) {
	if profilePath != "" {
		if err := validateProfilePath(profilePath); err != nil {
			return RuntimeNoticeResult{}, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := RuntimeNoticeResult{}
	if browser == BrowserSafari && !s.current.SafariNoticeSeen {
		result.ShowNotice = true
		result.Reason = "Safari profile reuse requires a runtime notice on each automation run."
		s.current.SafariNoticeSeen = true
	}

	s.current.Browser = browser
	s.current.ProfilePath = profilePath
	s.current.UpdatedAt = time.Now()
	return result, s.store.SaveRaw(s.current)
}

// SafariRuntimeNotice returns a notice result for each automation task that
// needs profile reuse. Must be shown every time (not suppressed after first).
func (s *Service) SafariRuntimeNotice() RuntimeNoticeResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current.Browser != BrowserSafari {
		return RuntimeNoticeResult{}
	}
	return RuntimeNoticeResult{
		ShowNotice: true,
		Reason:     "此任務需要重複使用 Safari profile。若 Safari 不允許 profile 重複使用，將觸發 authorization 阻斷訊號。",
	}
}

// ProfileReuseBlocked returns true when the selected browser cannot share its
// profile (e.g. Safari in some environments). When true, Draft Sandbox must
// treat this as an "authorization" stop signal.
func (s *Service) ProfileReuseBlocked() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	// In production: query OS / browser for actual profile-sharing capability.
	// For now: report Safari as potentially blocked; other browsers assume OK.
	return s.current.Browser == BrowserSafari
}

// --- path validation ---

func validateProfilePath(path string) error {
	// Must be an absolute path.
	if !filepath.IsAbs(path) {
		return fmt.Errorf("browser_pref: profile_path must be absolute, got: %q", path)
	}
	// Must not traverse outside expected browser profile directories.
	// Block obvious traversal patterns.
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("browser_pref: profile_path must not contain traversal sequences")
	}
	return nil
}

// persistence 由 storage.JSONStore 處理。
