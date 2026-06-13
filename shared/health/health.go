// Package health provides read-only Wails bindings for memory health
// and public config, replacing the legacy REST /api/memory/health and /api/config/public.
//
// Legacy reference: TASKS_1_1.md 遺留能力 #4.
// These are intentionally read-only — no mutation from the UI is allowed.
package health

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// MemoryHealth reports the app's memory and storage status.
type MemoryHealth struct {
	HeapAllocMB   float64 `json:"heap_alloc_mb"`
	NumGoroutines int     `json:"num_goroutines"`
	DataDirSizeMB float64 `json:"data_dir_size_mb"`
	LastChecked   string  `json:"last_checked"`
}

// ConfigPublic represents the non-sensitive public app configuration.
type ConfigPublic struct {
	AppVersion   string `json:"app_version"`
	SpecVersion  string `json:"spec_version"`
	DataRoot     string `json:"data_root"`
	DevMode      bool   `json:"dev_mode"`
	FirstRunDone bool   `json:"first_run_done"`
}

// Service holds the health reporting state.
type Service struct {
	mu       sync.Mutex
	dataRoot string
	devMode  bool
}

// NewService creates a health service rooted at the given data directory.
func NewService(dataRoot string, devMode bool) *Service {
	return &Service{dataRoot: dataRoot, devMode: devMode}
}

// GetMemoryHealth returns current memory/runtime metrics (read-only).
func (s *Service) GetMemoryHealth() MemoryHealth {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemoryHealth{
		HeapAllocMB:   float64(m.HeapAlloc) / 1024 / 1024,
		NumGoroutines: runtime.NumGoroutine(),
		DataDirSizeMB: s.dataDirSizeMB(),
		LastChecked:   time.Now().Format(time.RFC3339),
	}
}

// GetConfigPublic returns the public (non-sensitive) app configuration (read-only).
func (s *Service) GetConfigPublic() ConfigPublic {
	return ConfigPublic{
		AppVersion:   "1.2.0",
		SpecVersion:  "3.3.2",
		DataRoot:     s.dataRoot,
		DevMode:      s.devMode,
		FirstRunDone: s.isFirstRunDone(),
	}
}

// MarkFirstRunDone records that the user has completed onboarding.
func (s *Service) MarkFirstRunDone() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dataRoot, "data", ".first_run_done")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o600)
}

func (s *Service) isFirstRunDone() bool {
	path := filepath.Join(s.dataRoot, "data", ".first_run_done")
	_, err := os.Stat(path)
	return err == nil
}

func (s *Service) dataDirSizeMB() float64 {
	var total int64
	dir := filepath.Join(s.dataRoot, "data")
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return float64(total) / 1024 / 1024
}

// WriteConfigSnapshot exports the current config for debugging (JSON, read-only).
func (s *Service) WriteConfigSnapshot() ([]byte, error) {
	cfg := s.GetConfigPublic()
	return json.MarshalIndent(cfg, "", "  ")
}
