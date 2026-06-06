// source_trust/allowlist_store.go — 白名單持久化（project_source_allowlist.json）。
// 所有新增/續期/移除/過期/scope mismatch 寫入 source_trust_log.jsonl。
package source_trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// AllowlistStore 持久化服務
// ──────────────────────────────────────────────

// AllowlistStore 管理 project_source_allowlist.json 的讀寫。
type AllowlistStore struct {
	mu            sync.Mutex
	allowlistPath string
	logPath       string
	entries       []AllowlistEntry
	loaded        bool
}

// NewAllowlistStore 建立白名單儲存服務。
func NewAllowlistStore(projectRoot string) *AllowlistStore {
	return &AllowlistStore{
		allowlistPath: filepath.Join(projectRoot, "source_trust", "project_source_allowlist.json"),
		logPath:       filepath.Join(projectRoot, "source_trust", "source_trust_log.jsonl"),
	}
}

// ──────────────────────────────────────────────
// 讀取 / 寫入
// ──────────────────────────────────────────────

// load 懶載入白名單。
func (s *AllowlistStore) load() error {
	if s.loaded {
		return nil
	}
	data, err := os.ReadFile(s.allowlistPath)
	if os.IsNotExist(err) {
		s.entries = []AllowlistEntry{}
		s.loaded = true
		return nil
	}
	if err != nil {
		return err
	}
	// 空檔案或 "{}" 視為空清單
	if len(data) == 0 || string(data) == "{}" {
		s.entries = []AllowlistEntry{}
		s.loaded = true
		return nil
	}
	if err := json.Unmarshal(data, &s.entries); err != nil {
		s.entries = []AllowlistEntry{}
	}
	s.loaded = true
	return nil
}

// save 寫入白名單到磁碟。
func (s *AllowlistStore) save() error {
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(s.allowlistPath), 0755)
	return os.WriteFile(s.allowlistPath, data, 0o600)
}

// ──────────────────────────────────────────────
// CRUD 操作
// ──────────────────────────────────────────────

// List 回傳所有白名單（含已過期）。
func (s *AllowlistStore) List() ([]AllowlistEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return nil, err
	}
	result := make([]AllowlistEntry, len(s.entries))
	copy(result, s.entries)
	return result, nil
}

// ListActive 回傳未過期的白名單。
func (s *AllowlistStore) ListActive() ([]AllowlistEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return nil, err
	}
	var result []AllowlistEntry
	for _, e := range s.entries {
		if !e.IsExpired() {
			result = append(result, e)
		}
	}
	return result, nil
}

// Add 新增白名單記錄。呼叫端須先通過 Review Card。
func (s *AllowlistStore) Add(entry AllowlistEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return err
	}
	s.entries = append(s.entries, entry)
	s.writeLog("add", entry.ID, entry.CanonicalHostname, "")
	return s.save()
}

// Renew 續期指定白名單。
func (s *AllowlistStore) Renew(entryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return err
	}
	for i, e := range s.entries {
		if e.ID == entryID {
			s.entries[i].Renew()
			s.writeLog("renew", entryID, e.CanonicalHostname, "")
			return s.save()
		}
	}
	return fmt.Errorf("allowlist entry not found: %s", entryID)
}

// Remove 移除指定白名單。
func (s *AllowlistStore) Remove(entryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return err
	}
	var newEntries []AllowlistEntry
	found := false
	for _, e := range s.entries {
		if e.ID == entryID {
			found = true
			s.writeLog("remove", entryID, e.CanonicalHostname, "")
			continue
		}
		newEntries = append(newEntries, e)
	}
	if !found {
		return fmt.Errorf("allowlist entry not found: %s", entryID)
	}
	s.entries = newEntries
	return s.save()
}

// CheckStatus 查詢指定主機名在白名單中的狀態。
func (s *AllowlistStore) CheckStatus(hostname string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.load()
	for _, e := range s.entries {
		if e.CanonicalHostname == hostname {
			if e.IsExpired() {
				return "expired"
			}
			return "active"
		}
	}
	return "not_listed"
}

// ──────────────────────────────────────────────
// 啟動時到期掃描
// ──────────────────────────────────────────────

// ExpiringEntry 描述即將到期的白名單記錄。
type ExpiringEntry struct {
	Entry      AllowlistEntry `json:"entry"`
	DaysLeft   int            `json:"days_left"`
	IsExpired  bool           `json:"is_expired"`
}

// CheckExpiries 掃描所有白名單，回傳已過期或即將到期（3 天內）的記錄。
// 在 app 啟動時呼叫，結果透過 eventbus 通知前端。
func (s *AllowlistStore) CheckExpiries() ([]ExpiringEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return nil, err
	}
	var result []ExpiringEntry
	for _, e := range s.entries {
		if e.IsExpired() {
			result = append(result, ExpiringEntry{
				Entry:     e,
				DaysLeft:  0,
				IsExpired: true,
			})
			s.writeLog("expired", e.ID, e.CanonicalHostname, "auto_scan")
		} else if e.IsExpiringSoon() {
			daysLeft := int(time.Until(e.Expiry).Hours() / 24)
			result = append(result, ExpiringEntry{
				Entry:    e,
				DaysLeft: daysLeft,
			})
		}
	}
	return result, nil
}

// ──────────────────────────────────────────────
// Append-only 日誌
// ──────────────────────────────────────────────

type logEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	EntryID   string `json:"entry_id"`
	Hostname  string `json:"hostname"`
	Detail    string `json:"detail,omitempty"`
}

func (s *AllowlistStore) writeLog(action, entryID, hostname, detail string) {
	entry := logEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Action:    action,
		EntryID:   entryID,
		Hostname:  hostname,
		Detail:    detail,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(s.logPath), 0755)
	f, err := os.OpenFile(s.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	data = append(data, '\n')
	f.Write(data)
}
