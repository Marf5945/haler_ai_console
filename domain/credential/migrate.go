// domain/credential/migrate.go — 一次性 legacy credential 遷移。
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ TASKS_1_6_3 Step 2：MigrateLegacyCredentials                       │
// │                                                                     │
// │ 讀取兩處舊格式，合併寫入統一 store：                               │
// │  1. remote_bridge/remote_bridge_credentials.json（base64 明文）    │
// │     → ref = "remote_bridge:<channelID>"                             │
// │  2. data/secrets/credentials.enc（AES-256-GCM，舊 persona_avatar） │
// │     → ref = "persona_avatar:<原始ref>"                              │
// │                                                                     │
// │ 遷移成功後舊檔 rename 為 *.migrated。                              │
// │ 冪等安全：舊檔已 rename 則跳過，不報錯不刪資料。                  │
// └─────────────────────────────────────────────────────────────────────┘
package credential

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
)

// legacyRemoteBridgeEntry 舊 remote_bridge 的 JSON 結構。
type legacyRemoteBridgeEntry struct {
	ChannelID string `json:"channel_id"`
	RawURL    string `json:"raw_url"` // base64 encoded
}

// MigrateLegacyCredentials 將兩處舊格式 credential 合併進統一 store。
// 冪等：舊檔已 rename 為 *.migrated 則自動跳過。
func MigrateLegacyCredentials(root string, store *Store) error {
	// ── 1. remote_bridge base64 舊檔 ──
	rbPath := filepath.Join(root, "remote_bridge", "remote_bridge_credentials.json")
	if err := migrateRemoteBridge(rbPath, store); err != nil {
		return err
	}

	// ── 2. persona_avatar AES-GCM 舊檔 ──
	// 舊 persona_avatar 的 credentials.enc 與新 store 使用相同路徑、
	// 相同 key 派生邏輯。但舊檔的 ref 沒有 namespace prefix。
	// 若新 store 已在使用同一檔案，需從現有 entries 中辨識無 prefix 的舊 ref。
	if err := migratePersonaAvatar(store); err != nil {
		return err
	}

	return nil
}

// migrateRemoteBridge 遷移 base64 舊格式。
func migrateRemoteBridge(rbPath string, store *Store) error {
	data, err := os.ReadFile(rbPath)
	if os.IsNotExist(err) {
		return nil // 無舊檔，跳過
	}
	if err != nil {
		return err
	}

	var entries []legacyRemoteBridgeEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil // 損毀檔案，視為無資料
	}

	for _, e := range entries {
		decoded, err := base64.StdEncoding.DecodeString(e.RawURL)
		if err != nil {
			continue // 單筆解碼失敗跳過
		}
		ref := "remote_bridge:" + e.ChannelID
		if err := store.Store(ref, string(decoded)); err != nil {
			return err
		}
	}

	// 成功後 rename 舊檔
	return os.Rename(rbPath, rbPath+".migrated")
}

// migratePersonaAvatar 遷移舊 persona_avatar 的無 prefix entries。
// 舊檔與新 store 共用同一路徑 + 同一 key，所以直接讀現有 entries，
// 將無 "persona_avatar:" prefix 且無 "remote_bridge:" prefix 的 ref
// 加上 "persona_avatar:" prefix 重新寫入。
func migratePersonaAvatar(store *Store) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	data := store.loadData()

	var toMigrate []string
	for ref := range data.Entries {
		// 已有 namespace prefix → 不需遷移
		if hasNamespace(ref) {
			continue
		}
		toMigrate = append(toMigrate, ref)
	}

	if len(toMigrate) == 0 {
		return nil // 無需遷移
	}

	for _, oldRef := range toMigrate {
		entry := data.Entries[oldRef]
		newRef := "persona_avatar:" + oldRef
		data.Entries[newRef] = entry
		delete(data.Entries, oldRef)
	}

	return store.saveData(data)
}

// hasNamespace 檢查 ref 是否已包含 namespace prefix。
func hasNamespace(ref string) bool {
	for i := 0; i < len(ref); i++ {
		if ref[i] == ':' {
			return true
		}
	}
	return false
}
