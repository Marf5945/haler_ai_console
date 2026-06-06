// remote_bridge/async_dispatch_test.go — 非同步 Dispatch 測試。
package remote_bridge

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"ui_console/shared/eventbus"
)

// TestDispatchAsync_AllSuccess 全部段成功。
func TestDispatchAsync_AllSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	svc, bus := setupAsyncTestService(server.URL)
	ad := NewAsyncDispatcher(svc, bus)

	dispatchID := ad.DispatchAsync(AsyncDispatchRequest{
		ChannelID: "test_ch",
		Content:   "short message",
	})

	// 等待背景完成
	time.Sleep(500 * time.Millisecond)

	if dispatchID == "" {
		t.Fatal("expected non-empty dispatch_id")
	}
}

// TestDispatchAsync_TargetsRequestedChannel 確認暫時測試傳送會使用指定 channel。
func TestDispatchAsync_TargetsRequestedChannel(t *testing.T) {
	var hitA, hitB int
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitA++
		w.WriteHeader(200)
	}))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitB++
		w.WriteHeader(200)
	}))
	defer serverB.Close()

	bus := eventbus.New(nil)
	secrets := &mockSecretStore{data: make(map[string]string)}
	secrets.Store("remote_bridge:active_ch", serverA.URL)
	secrets.Store("remote_bridge:target_ch", serverB.URL)
	now := time.Now()
	expiry := now.Add(24 * time.Hour)
	svc := &Service{
		mu:          sync.Mutex{},
		projectRoot: "/tmp/async_target_test",
		bindings: []ChannelBinding{
			{
				ID:                  "active_ch",
				Channel:             ChannelDiscord,
				Mode:                ModeNotificationOnly,
				Active:              true,
				ExpiresAt:           &expiry,
				TestPassed:          true,
				CreatedAt:           now,
				CustomWebhookConfig: &WebhookRequest{URL: serverA.URL, DevMode: true},
			},
			{
				ID:                  "target_ch",
				Channel:             ChannelDiscord,
				Mode:                ModeNotificationOnly,
				Active:              false,
				ExpiresAt:           &expiry,
				TestPassed:          true,
				CreatedAt:           now,
				CustomWebhookConfig: &WebhookRequest{URL: serverB.URL, DevMode: true},
			},
		},
		loaded:   true,
		secrets:  secrets,
		auditLog: NewAuditLog("/tmp/async_target_test"),
	}

	NewAsyncDispatcher(svc, bus).DispatchAsync(AsyncDispatchRequest{
		ChannelID: "target_ch",
		Content:   "temporary test",
	})
	time.Sleep(500 * time.Millisecond)

	if hitA != 0 {
		t.Fatalf("active channel should not be used, hitA=%d", hitA)
	}
	if hitB == 0 {
		t.Fatal("target channel was not used")
	}
}

// TestDispatchAsync_NoActiveChannel 無啟用通道。
func TestDispatchAsync_NoActiveChannel(t *testing.T) {
	bus := eventbus.New(nil)
	svc := &Service{
		projectRoot: "/tmp/test",
		bindings:    []ChannelBinding{},
		loaded:      true,
		secrets:     &mockSecretStore{data: make(map[string]string)},
		auditLog:    NewAuditLog("/tmp/test"),
	}

	ad := NewAsyncDispatcher(svc, bus)
	dispatchID := ad.DispatchAsync(AsyncDispatchRequest{Content: "test"})

	if dispatchID == "" {
		t.Fatal("expected non-empty dispatch_id even on failure")
	}
}

// setupAsyncTestService 建立測試用 service（含一個啟用的 telegram channel）。
func setupAsyncTestService(serverURL string) (*Service, *eventbus.Bus) {
	bus := eventbus.New(nil)
	secrets := &mockSecretStore{data: make(map[string]string)}
	secrets.Store("remote_bridge:ch_async_test", serverURL)

	now := time.Now()
	expiry := now.Add(24 * time.Hour)
	svc := &Service{
		mu:          sync.Mutex{},
		projectRoot: "/tmp/async_test",
		bindings: []ChannelBinding{{
			ID:         "ch_async_test",
			Channel:    ChannelTelegram,
			Mode:       ModeNotificationOnly,
			Active:     true,
			ExpiresAt:  &expiry,
			TestPassed: true,
			CreatedAt:  now,
		}},
		loaded:   true,
		secrets:  secrets,
		auditLog: NewAuditLog("/tmp/async_test"),
	}

	return svc, bus
}
