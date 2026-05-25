// w3a_media/trust_list.go — §9A.5 本機開發者信任清單 + 線上登錄 stub。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 開發者信任清單管理已知 W3A-aware app 的公鑰。               │
// │                                                             │
// │ 本機信任模式：                                              │
// │  - 使用者手動新增信任的 app 開發者                          │
// │  - 儲存 appID + publicKey + displayName                     │
// │  - 驗證操作簽章時查詢此清單                                 │
// │                                                             │
// │ 線上登錄 stub（§9A.5 接口保留）：                           │
// │  - VerifyOnlineRegistry() 目前回傳 registry_unavailable     │
// │  - 未來可接 W3A 線上伺服器，不需改呼叫端                   │
// │                                                             │
// │ 持久化：lazy-load + mutex + JSON（參考 source_trust 模式）  │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// 信任清單
// ──────────────────────────────────────────────

// TrustList 本機開發者信任清單。
type TrustList struct {
	mu       sync.Mutex
	entries  []TrustedDeveloper
	filePath string
	loaded   bool
}

// NewTrustList 建立信任清單管理器。
func NewTrustList(hookRoot string) *TrustList {
	return &TrustList{
		filePath: filepath.Join(hookRoot, "w3a_trust_list.json"),
	}
}

// load 延遲載入（lazy-load）。
func (tl *TrustList) load() {
	if tl.loaded {
		return
	}
	tl.loaded = true
	data, err := os.ReadFile(tl.filePath)
	if err != nil {
		return // 檔案不存在 → 空清單
	}
	json.Unmarshal(data, &tl.entries)
}

// save 持久化到檔案。
func (tl *TrustList) save() error {
	data, err := json.MarshalIndent(tl.entries, "", "  ")
	if err != nil {
		return err
	}
	// SEC-W07（2026-05-24）：trust list 是「本機使用者信任哪些 W3A app」的隱私決策，
	// 預設不對其他帳號 / process 公開（公鑰本身不敏感，但「信任名單」是 metadata）。
	return os.WriteFile(tl.filePath, data, 0o600)
}

// Add 新增信任的開發者。
func (tl *TrustList) Add(appID, pubKey, displayName string) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.load()

	// 檢查重複
	for _, e := range tl.entries {
		if e.AppID == appID && e.PublicKey == pubKey {
			return fmt.Errorf("developer %s already trusted", appID)
		}
	}

	tl.entries = append(tl.entries, TrustedDeveloper{
		AppID:       appID,
		PublicKey:   pubKey,
		DisplayName: displayName,
		AddedAt:     time.Now().Format(time.RFC3339),
	})
	return tl.save()
}

// Remove 移除信任的開發者。
func (tl *TrustList) Remove(appID string) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.load()

	newEntries := make([]TrustedDeveloper, 0, len(tl.entries))
	found := false
	for _, e := range tl.entries {
		if e.AppID == appID {
			found = true
			continue
		}
		newEntries = append(newEntries, e)
	}
	if !found {
		return fmt.Errorf("developer %s not found", appID)
	}

	tl.entries = newEntries
	return tl.save()
}

// IsTrusted 檢查指定 appID + pubKey 是否在信任清單中。
func (tl *TrustList) IsTrusted(appID, pubKey string) bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.load()

	for _, e := range tl.entries {
		if e.AppID == appID && e.PublicKey == pubKey {
			return true
		}
	}
	return false
}

// List 列出所有信任的開發者。
func (tl *TrustList) List() []TrustedDeveloper {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.load()

	result := make([]TrustedDeveloper, len(tl.entries))
	copy(result, tl.entries)
	return result
}

// ──────────────────────────────────────────────
// 線上登錄 stub（§9A.5 接口保留）
// ──────────────────────────────────────────────

// VerifyOnlineRegistry 查詢線上 W3A 開發者登錄系統。
// 目前為 stub：永遠回傳 registry_unavailable。
// 未來接線上伺服器時只需修改此函式實作，呼叫端不需變動。
func VerifyOnlineRegistry(appID string) (RegistryStatus, error) {
	// TODO: 當 W3A 線上登錄伺服器上線後，在此發送 HTTP GET
	//       GET https://registry.w3a.org/v1/apps/{appID}/status
	//       回傳 RegistryVerified / RegistryUnknown
	return RegistryUnavailable, nil
}
