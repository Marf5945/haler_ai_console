// domain/credential/store_test.go — 統一 SecretStore 測試。
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ TASKS_1_6_3 Step 6：Credential 統一測試                            │
// │                                                                     │
// │ 測試清單：                                                         │
// │  1. TestStoreLoadDeleteHas         — 基本 round-trip               │
// │  2. TestNamespacedRefIsolation     — namespace 互不干擾            │
// │  3. TestStoreOverwrite             — 同 ref 覆寫                   │
// │  4. TestPersistenceAcrossInstances — 重建 Store 後 Load 仍正確     │
// │  5. TestEncryptedFileNotPlaintext  — smoke: enc 檔無明文           │
// │  6. TestMigrateLegacyCredentials   — 兩種舊格式合併遷移           │
// │  7. TestMigrateLegacyCredentialsIdempotent — 二次執行冪等          │
// └─────────────────────────────────────────────────────────────────────┘
package credential

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testStore(dir string) *Store {
	var key [32]byte
	copy(key[:], []byte("0123456789abcdef0123456789abcdef"))
	return NewStoreWithProvider(dir, NewStaticMasterKeyProvider("test_provider", key))
}

func legacyStore(dir string) *Store {
	store := NewStoreWithProvider(dir, NewStaticMasterKeyProvider("legacy_test", deriveLegacyDeviceKey()))
	store.legacyDirect = true
	store.key = deriveLegacyDeviceKey()
	store.keyLoaded = true
	return store
}

// ──────────────────────────────────────────────
// 1. 基本 round-trip
// ──────────────────────────────────────────────

func TestStoreLoadDeleteHas(t *testing.T) {
	store := testStore(t.TempDir())

	// Store → Load
	if err := store.Store("test:key1", "secret-value-123"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	val, err := store.Load("test:key1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if val != "secret-value-123" {
		t.Errorf("Load mismatch: got %q", val)
	}

	// Has
	if !store.Has("test:key1") {
		t.Error("Has should return true")
	}
	if store.Has("test:nonexistent") {
		t.Error("Has should return false for nonexistent")
	}

	// Delete → Has false → Load error
	if err := store.Delete("test:key1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.Has("test:key1") {
		t.Error("Has should return false after Delete")
	}
	if _, err := store.Load("test:key1"); err == nil {
		t.Error("Load should error after Delete")
	}
}

// ──────────────────────────────────────────────
// 2. namespace 隔離
// ──────────────────────────────────────────────

func TestNamespacedRefIsolation(t *testing.T) {
	store := testStore(t.TempDir())

	// 兩個 namespace 同時存在
	if err := store.Store("remote_bridge:ch1", "https://hook.slack.com/xxx"); err != nil {
		t.Fatal(err)
	}
	if err := store.Store("persona_avatar:openai_key", "sk-abc-123"); err != nil {
		t.Fatal(err)
	}

	// 各自 Load 正確
	v1, _ := store.Load("remote_bridge:ch1")
	v2, _ := store.Load("persona_avatar:openai_key")
	if v1 != "https://hook.slack.com/xxx" {
		t.Errorf("remote_bridge value mismatch: %q", v1)
	}
	if v2 != "sk-abc-123" {
		t.Errorf("persona_avatar value mismatch: %q", v2)
	}

	// Delete 一方不影響另一方
	if err := store.Delete("remote_bridge:ch1"); err != nil {
		t.Fatal(err)
	}
	if store.Has("remote_bridge:ch1") {
		t.Error("remote_bridge should be gone")
	}
	if !store.Has("persona_avatar:openai_key") {
		t.Error("persona_avatar should still exist after deleting remote_bridge")
	}
}

// ──────────────────────────────────────────────
// 3. 同 ref 覆寫
// ──────────────────────────────────────────────

func TestStoreOverwrite(t *testing.T) {
	store := testStore(t.TempDir())

	_ = store.Store("test:key", "old-value")
	_ = store.Store("test:key", "new-value")

	val, err := store.Load("test:key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "new-value" {
		t.Errorf("expected overwritten value, got %q", val)
	}
}

// ──────────────────────────────────────────────
// 4. 跨 instance 持久化
// ──────────────────────────────────────────────

func TestPersistenceAcrossInstances(t *testing.T) {
	dir := t.TempDir()

	// 第一個 instance 寫入
	store1 := testStore(dir)
	_ = store1.Store("remote_bridge:ch1", "url-1")
	_ = store1.Store("persona_avatar:key1", "secret-1")

	// 第二個 instance 讀取
	store2 := testStore(dir)
	v1, err := store2.Load("remote_bridge:ch1")
	if err != nil {
		t.Fatalf("Load after restart: %v", err)
	}
	if v1 != "url-1" {
		t.Errorf("persistence mismatch: got %q", v1)
	}
	v2, err := store2.Load("persona_avatar:key1")
	if err != nil {
		t.Fatalf("Load after restart: %v", err)
	}
	if v2 != "secret-1" {
		t.Errorf("persistence mismatch: got %q", v2)
	}
}

// ──────────────────────────────────────────────
// 5. smoke: 加密檔案不含明文
// ──────────────────────────────────────────────

func TestEncryptedFileNotPlaintext(t *testing.T) {
	dir := t.TempDir()
	store := testStore(dir)

	secret := "super-secret-api-key-12345"
	_ = store.Store("test:smoke", secret)

	raw, err := os.ReadFile(store.FilePath())
	if err != nil {
		t.Fatalf("read enc file: %v", err)
	}
	if strings.Contains(string(raw), secret) {
		t.Error("encrypted file should NOT contain plaintext secret")
	}
}

func TestStoreWritesV2Metadata(t *testing.T) {
	store := testStore(t.TempDir())
	if err := store.Store("test:key", "secret"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	raw, err := os.ReadFile(store.FilePath())
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	var data storeData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data.SchemaVersion != StoreSchemaV2 {
		t.Fatalf("schema=%q want %q", data.SchemaVersion, StoreSchemaV2)
	}
	if data.KeyProvider != "test_provider" {
		t.Fatalf("provider=%q", data.KeyProvider)
	}
}

func TestLegacyStoreRequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	old := legacyStore(dir)
	if err := old.Store("persona_avatar:old", "legacy-secret"); err != nil {
		t.Fatalf("legacy Store: %v", err)
	}

	store := testStore(dir)
	status := store.MigrationStatus()
	if !status.Required || status.Ready {
		t.Fatalf("status=%+v want migration required", status)
	}
	if _, err := store.Load("persona_avatar:old"); !errors.Is(err, ErrCredentialMigrationRequired) {
		t.Fatalf("Load err=%v want migration required", err)
	}
	if err := store.ConfirmMigration(); err != nil {
		t.Fatalf("ConfirmMigration: %v", err)
	}
	value, err := store.Load("persona_avatar:old")
	if err != nil {
		t.Fatalf("Load after migration: %v", err)
	}
	if value != "legacy-secret" {
		t.Fatalf("value=%q", value)
	}
}

func TestDisableMigrationBlocksCredentialUse(t *testing.T) {
	dir := t.TempDir()
	old := legacyStore(dir)
	if err := old.Store("persona_avatar:old", "legacy-secret"); err != nil {
		t.Fatalf("legacy Store: %v", err)
	}

	store := testStore(dir)
	if err := store.DisableMigration(); err != nil {
		t.Fatalf("DisableMigration: %v", err)
	}
	if err := store.Store("test:new", "secret"); !errors.Is(err, ErrCredentialMigrationDisabled) {
		t.Fatalf("Store err=%v want migration disabled", err)
	}
}

// ──────────────────────────────────────────────
// 6. legacy migration — 兩種舊格式合併
// ──────────────────────────────────────────────

func TestMigrateLegacyCredentials(t *testing.T) {
	dir := t.TempDir()

	// ── 準備舊 remote_bridge 檔案（base64 格式）──
	rbDir := filepath.Join(dir, "remote_bridge")
	_ = os.MkdirAll(rbDir, 0755)

	rbEntries := []legacyRemoteBridgeEntry{
		{ChannelID: "rb_line_001", RawURL: base64.StdEncoding.EncodeToString([]byte("https://line.bot/hook1"))},
		{ChannelID: "rb_discord_002", RawURL: base64.StdEncoding.EncodeToString([]byte("https://discord.bot/hook2"))},
	}
	rbData, _ := json.MarshalIndent(rbEntries, "", "  ")
	rbPath := filepath.Join(rbDir, "remote_bridge_credentials.json")
	_ = os.WriteFile(rbPath, rbData, 0600)

	// ── 準備舊 persona_avatar 檔案（AES-GCM，無 namespace prefix）──
	// 用一個臨時 Store 寫入無 prefix 的 ref（模擬舊行為）
	old := legacyStore(dir)
	_ = old.Store("openai_api_key", "sk-old-persona-secret")

	store := testStore(dir)
	if err := store.ConfirmMigration(); err != nil {
		t.Fatalf("ConfirmMigration: %v", err)
	}

	// ── 執行 migration ──
	if err := MigrateLegacyCredentials(dir, store); err != nil {
		t.Fatalf("MigrateLegacyCredentials: %v", err)
	}

	// ── 驗證 remote_bridge 資料已遷移 ──
	v1, err := store.Load("remote_bridge:rb_line_001")
	if err != nil {
		t.Fatalf("Load remote_bridge:rb_line_001: %v", err)
	}
	if v1 != "https://line.bot/hook1" {
		t.Errorf("remote_bridge value mismatch: %q", v1)
	}

	v2, err := store.Load("remote_bridge:rb_discord_002")
	if err != nil {
		t.Fatalf("Load remote_bridge:rb_discord_002: %v", err)
	}
	if v2 != "https://discord.bot/hook2" {
		t.Errorf("remote_bridge value mismatch: %q", v2)
	}

	// ── 驗證 persona_avatar 舊 ref 已加 prefix ──
	v3, err := store.Load("persona_avatar:openai_api_key")
	if err != nil {
		t.Fatalf("Load persona_avatar:openai_api_key: %v", err)
	}
	if v3 != "sk-old-persona-secret" {
		t.Errorf("persona_avatar value mismatch: %q", v3)
	}

	// ── 驗證舊 ref（無 prefix）已不存在 ──
	if store.Has("openai_api_key") {
		t.Error("old un-namespaced ref should have been removed")
	}

	// ── 驗證合併不覆蓋：兩邊資料都在同一 store ──
	if !store.Has("remote_bridge:rb_line_001") || !store.Has("persona_avatar:openai_api_key") {
		t.Error("migration should merge, not overwrite")
	}

	// ── 驗證舊 remote_bridge 檔案已 rename ──
	if _, err := os.Stat(rbPath); !os.IsNotExist(err) {
		t.Error("old remote_bridge_credentials.json should have been renamed")
	}
	if _, err := os.Stat(rbPath + ".migrated"); os.IsNotExist(err) {
		t.Error("remote_bridge_credentials.json.migrated should exist")
	}
}

// ──────────────────────────────────────────────
// 7. migration 冪等
// ──────────────────────────────────────────────

func TestMigrateLegacyCredentialsIdempotent(t *testing.T) {
	dir := t.TempDir()

	// 準備舊 remote_bridge 檔案
	rbDir := filepath.Join(dir, "remote_bridge")
	_ = os.MkdirAll(rbDir, 0755)
	rbEntries := []legacyRemoteBridgeEntry{
		{ChannelID: "ch1", RawURL: base64.StdEncoding.EncodeToString([]byte("https://example.com"))},
	}
	rbData, _ := json.MarshalIndent(rbEntries, "", "  ")
	_ = os.WriteFile(filepath.Join(rbDir, "remote_bridge_credentials.json"), rbData, 0600)

	store := testStore(dir)

	// 第一次 migration
	if err := MigrateLegacyCredentials(dir, store); err != nil {
		t.Fatalf("first migration: %v", err)
	}

	// 確認資料存在
	v1, _ := store.Load("remote_bridge:ch1")
	if v1 != "https://example.com" {
		t.Fatalf("first migration value wrong: %q", v1)
	}

	// 第二次 migration — 不應報錯，不應刪資料
	if err := MigrateLegacyCredentials(dir, store); err != nil {
		t.Fatalf("second migration should not error: %v", err)
	}

	// 資料仍在
	v2, err := store.Load("remote_bridge:ch1")
	if err != nil {
		t.Fatalf("Load after second migration: %v", err)
	}
	if v2 != "https://example.com" {
		t.Errorf("value after second migration: %q", v2)
	}
}
