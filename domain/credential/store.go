// domain/credential/store.go — 統一加密憑證儲存。
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ TASKS_1_6_3 Step 1：統一 CredentialStore                           │
// │                                                                     │
// │ 合併原本分散在兩處的 credential 儲存：                             │
// │  • adapter/remote_bridge/credential.go（base64，已移除）           │
// │  • adapter/persona_avatar/credential.go（AES-256-GCM，已移除）     │
// │                                                                     │
// │ 統一為單一 SecretStore，所有 ref 必須 namespaced：                  │
// │  • remote_bridge:<channelID>                                        │
// │  • persona_avatar:<ref>                                             │
// │                                                                     │
// │ 唯一加密策略：AES-256-GCM + OS-backed master key。                │
// │ 唯一儲存檔案：data/secrets/credentials.enc（權限 0600）。          │
// │ credential 不得進入 portable export / logs / LLM context。         │
// └─────────────────────────────────────────────────────────────────────┘
package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// SecretStore 統一的加密憑證儲存介面。
// ref 必須 namespaced（如 "remote_bridge:ch1"、"persona_avatar:openai_key"）。
type SecretStore interface {
	Store(ref string, value string) error
	Load(ref string) (string, error)
	Delete(ref string) error
	Has(ref string) bool
}

const (
	StoreSchemaV2       = "credential-store-v2"
	defaultKeyService   = "AI Console"
	defaultKeyAccount   = "credential-master-key"
	migrationMarkerFile = "credential_migration_disabled"
)

// ──────────────────────────────────────────────
// AES-256-GCM 實作
// ──────────────────────────────────────────────

// encryptedEntry 單筆加密記錄。
type encryptedEntry struct {
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// storeData 加密檔案的頂層結構。
type storeData struct {
	SchemaVersion string                    `json:"schema_version,omitempty"`
	KeyProvider   string                    `json:"key_provider,omitempty"`
	Entries       map[string]encryptedEntry `json:"entries"`
}

// Store 是 SecretStore 的 AES-256-GCM 檔案實作。
type Store struct {
	mu             sync.Mutex
	filePath       string // data/secrets/credentials.enc
	migrationPath  string
	provider       MasterKeyProvider
	key            [32]byte // AES-256 master key，由 OS provider 保護。
	keyLoaded      bool
	legacyDirect   bool // 測試/舊格式寫入專用；產品路徑不得啟用。
	migrationBlock bool
}

// NewStore 建立統一加密憑證儲存。
// baseDir 為應用程式根目錄，檔案存於 baseDir/data/secrets/credentials.enc。
func NewStore(baseDir string) *Store {
	return NewStoreWithProvider(baseDir, NewOSMasterKeyProvider(baseDir, defaultKeyService, defaultKeyAccount))
}

// NewStoreWithProvider 讓測試或平台層注入 master key provider。
func NewStoreWithProvider(baseDir string, provider MasterKeyProvider) *Store {
	return &Store{
		filePath:      filepath.Join(baseDir, "data", "secrets", "credentials.enc"),
		migrationPath: filepath.Join(baseDir, "data", "secrets", migrationMarkerFile),
		provider:      provider,
	}
}

// ──────────────────────────────────────────────
// SecretStore 介面實作
// ──────────────────────────────────────────────

// Store 加密並儲存 credential。
func (s *Store) Store(ref string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUsableLocked(); err != nil {
		return err
	}

	data := s.loadData()

	ciphertext, nonce, err := s.encrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	data.Entries[ref] = encryptedEntry{
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}

	return s.saveData(data)
}

// Load 載入並解密 credential。
func (s *Store) Load(ref string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUsableLocked(); err != nil {
		return "", err
	}

	data := s.loadData()

	entry, ok := data.Entries[ref]
	if !ok {
		return "", fmt.Errorf("credential not found: %s", ref)
	}

	plaintext, err := s.decrypt(entry.Ciphertext, entry.Nonce)
	if err != nil {
		return "", fmt.Errorf("decrypt (wrong device?): %w", err)
	}

	return string(plaintext), nil
}

// Delete 刪除指定 credential。
func (s *Store) Delete(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUsableLocked(); err != nil {
		return err
	}

	data := s.loadData()
	delete(data.Entries, ref)
	return s.saveData(data)
}

// Has 檢查指定 credential 是否存在。
func (s *Store) Has(ref string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUsableLocked(); err != nil {
		return false
	}

	data := s.loadData()
	_, ok := data.Entries[ref]
	return ok
}

// ──────────────────────────────────────────────
// 狀態與 migration
// ──────────────────────────────────────────────

// MigrationStatus 回傳前端是否需要提示使用者確認 v1 -> v2 遷移。
type MigrationStatus struct {
	Ready          bool   `json:"ready"`
	Required       bool   `json:"required"`
	ProviderID     string `json:"provider_id"`
	Disabled       bool   `json:"disabled"`
	Error          string `json:"error"`
	CredentialPath string `json:"credential_path"`
}

// MigrationStatus 檢查 credential store 是否可用；舊 v1 檔不會被靜默解密。
func (s *Store) MigrationStatus() MigrationStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := MigrationStatus{
		ProviderID:     s.provider.ProviderID(),
		CredentialPath: s.filePath,
	}
	if s.migrationDisabledLocked() {
		status.Disabled = true
		status.Error = ErrCredentialMigrationDisabled.Error()
		return status
	}
	if s.isLegacyFileLocked() {
		status.Required = true
		status.Error = ErrCredentialMigrationRequired.Error()
		return status
	}
	if !s.fileExistsLocked() {
		status.Ready = true
		return status
	}
	if err := s.ensureKeyLocked(); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Ready = true
	return status
}

// ConfirmMigration performs the only allowed legacy-key read path.
func (s *Store) ConfirmMigration() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isLegacyFileLocked() {
		return s.ensureKeyLocked()
	}

	rawLegacy := s.loadData()
	osKey, err := s.provider.LoadOrCreateKey()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCredentialProviderUnavailable, err)
	}

	legacyKey := deriveLegacyDeviceKey()
	next := storeData{
		SchemaVersion: StoreSchemaV2,
		KeyProvider:   s.provider.ProviderID(),
		Entries:       make(map[string]encryptedEntry, len(rawLegacy.Entries)),
	}

	// 先完整解密再寫新檔；任何一筆失敗都保留舊檔，避免半套 migration。
	for ref, entry := range rawLegacy.Entries {
		plaintext, err := decryptWithKey(legacyKey, entry.Ciphertext, entry.Nonce)
		if err != nil {
			return fmt.Errorf("legacy decrypt %s: %w", ref, err)
		}
		ciphertext, nonce, err := encryptWithKey(osKey, plaintext)
		if err != nil {
			return fmt.Errorf("reencrypt %s: %w", ref, err)
		}
		next.Entries[ref] = encryptedEntry{Nonce: nonce, Ciphertext: ciphertext}
	}

	s.key = osKey
	s.keyLoaded = true
	if err := s.saveDataAtomic(next); err != nil {
		return err
	}
	_ = os.Remove(s.migrationPath)
	s.migrationBlock = false
	return nil
}

// DisableMigration keeps the legacy file untouched but blocks credential use in this session.
func (s *Store) DisableMigration() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.migrationBlock = true
	if err := os.MkdirAll(filepath.Dir(s.migrationPath), 0700); err != nil {
		return err
	}
	return os.WriteFile(s.migrationPath, []byte("disabled\n"), 0600)
}

func (s *Store) ensureUsableLocked() error {
	if s.legacyDirect {
		return nil
	}
	if s.migrationDisabledLocked() {
		return ErrCredentialMigrationDisabled
	}
	if s.isLegacyFileLocked() {
		return ErrCredentialMigrationRequired
	}
	return s.ensureKeyLocked()
}

func (s *Store) ensureKeyLocked() error {
	if s.keyLoaded {
		return nil
	}
	key, err := s.provider.LoadOrCreateKey()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCredentialProviderUnavailable, err)
	}
	s.key = key
	s.keyLoaded = true
	return nil
}

func (s *Store) isLegacyFileLocked() bool {
	raw, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) || len(raw) == 0 {
		return false
	}
	if err != nil {
		return false
	}
	var data storeData
	if err := json.Unmarshal(raw, &data); err != nil {
		return false
	}
	return data.SchemaVersion == "" && len(data.Entries) > 0
}

func (s *Store) fileExistsLocked() bool {
	_, err := os.Stat(s.filePath)
	return err == nil
}

func (s *Store) migrationDisabledLocked() bool {
	if s.migrationBlock {
		return true
	}
	_, err := os.Stat(s.migrationPath)
	return err == nil
}

// deriveLegacyDeviceKey is retained only to migrate v1 stores after user consent.
func deriveLegacyDeviceKey() [32]byte {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}

	seed := fmt.Sprintf("ai-console-credential-v1:%s:%s", hostname, username)
	return sha256.Sum256([]byte(seed))
}

// ──────────────────────────────────────────────
// AES-256-GCM 加密 / 解密
// ──────────────────────────────────────────────

func (s *Store) encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	return encryptWithKey(s.key, plaintext)
}

func (s *Store) decrypt(ciphertext, nonce []byte) ([]byte, error) {
	return decryptWithKey(s.key, ciphertext, nonce)
}

func encryptWithKey(key [32]byte, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

func decryptWithKey(key [32]byte, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ──────────────────────────────────────────────
// 檔案讀寫
// ──────────────────────────────────────────────

func (s *Store) loadData() storeData {
	data := storeData{Entries: make(map[string]encryptedEntry)}

	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		return data
	}

	_ = json.Unmarshal(raw, &data)
	if data.Entries == nil {
		data.Entries = make(map[string]encryptedEntry)
	}
	return data
}

func (s *Store) saveData(data storeData) error {
	if !s.legacyDirect {
		data.SchemaVersion = StoreSchemaV2
		data.KeyProvider = s.provider.ProviderID()
	}
	return s.saveDataAtomic(data)
}

func (s *Store) saveDataAtomic(data storeData) error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// FilePath 回傳 credentials.enc 完整路徑（供 migration 使用）。
func (s *Store) FilePath() string {
	return s.filePath
}

// ProviderID reports the active OS-backed master-key provider.
func (s *Store) ProviderID() string {
	if s == nil || s.provider == nil {
		return ""
	}
	return s.provider.ProviderID()
}
