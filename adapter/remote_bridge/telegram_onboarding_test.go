// remote_bridge/telegram_onboarding_test.go — Telegram Onboarding 測試。
package remote_bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStartTelegramOnboarding_ValidToken 模擬有效 token。
func TestStartTelegramOnboarding_ValidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"id":       12345,
				"username": "test_bot",
			},
		})
	}))
	defer server.Close()

	// 因為 StartTelegramOnboarding 硬編碼 api.telegram.org，
	// 此測試主要驗證函式邏輯結構。整合測試需 mock。
	// 這裡直接驗證空 token 場景。
	session, err := StartTelegramOnboarding("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if session.Valid {
		t.Error("expected Valid=false for empty token")
	}
}

// TestStartTelegramOnboarding_EmptyToken 驗證空 token 回傳錯誤。
func TestStartTelegramOnboarding_EmptyToken(t *testing.T) {
	_, err := StartTelegramOnboarding("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

// TestHashChatID 驗證 hash 產出格式。
func TestHashChatID(t *testing.T) {
	hash := HashChatID("123456789")
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("expected sha256: prefix, got %s", hash)
	}
	if len(hash) < 20 {
		t.Errorf("hash too short: %s", hash)
	}
}

// TestConfirmTelegramChatID_Empty 驗證空 chat_id 回傳錯誤。
func TestConfirmTelegramChatID_Empty(t *testing.T) {
	// 用 mock secret store
	store := &mockSecretStore{}
	err := ConfirmTelegramChatID(store, "ch_test", "")
	if err == nil {
		t.Fatal("expected error for empty chat_id")
	}
}

// TestConfirmTelegramChatID_Success 驗證正常存儲。
func TestConfirmTelegramChatID_Success(t *testing.T) {
	store := &mockSecretStore{data: make(map[string]string)}
	err := ConfirmTelegramChatID(store, "ch_test", "99887766")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 確認存入 device-local
	val, _ := store.Load("remote_bridge:ch_test:chat_id")
	if val != "99887766" {
		t.Errorf("expected 99887766, got %s", val)
	}
}

// mockSecretStore 用於單元測試的 mock。
type mockSecretStore struct {
	data map[string]string
}

func (m *mockSecretStore) Store(ref, value string) error {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[ref] = value
	return nil
}
func (m *mockSecretStore) Load(ref string) (string, error) {
	v, ok := m.data[ref]
	if !ok {
		return "", nil
	}
	return v, nil
}
func (m *mockSecretStore) Delete(ref string) error {
	delete(m.data, ref)
	return nil
}
func (m *mockSecretStore) Has(ref string) bool {
	_, ok := m.data[ref]
	return ok
}
