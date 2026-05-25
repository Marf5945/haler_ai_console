// delegation/registry.go — Sub Registry 管理。
// 維護 sub_registry_snapshot.json，儲存所有 sub 的結構化元資料與標籤路由。
package delegation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// SubEntry 一筆 sub 的元資料記錄。
type SubEntry struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Triggers     []string          `json:"triggers"`      // 觸發詞清單
	ToolsUsed    []string          `json:"tools_used"`    // 使用工具清單
	InputSchema  map[string]string `json:"input_schema"`  // 輸入欄位定義
	OutputSchema map[string]string `json:"output_schema"` // 輸出欄位定義
	ActionTags   []string          `json:"action_tags"`   // 動作標籤清單
	CreatedAt    string            `json:"created_at"`    // RFC3339 建立時間
	LastUsed     string            `json:"last_used"`     // RFC3339 最後使用時間
}

// Registry 管理所有 sub 的 registry。
type Registry struct {
	mu       sync.Mutex
	entries  []SubEntry
	filePath string // sub_registry_snapshot.json 完整路徑
}

// ──────────────────────────────────────────────
// 建構與載入
// ──────────────────────────────────────────────

// NewRegistry 建立 Registry，從 mainDir 載入 sub_registry_snapshot.json。
func NewRegistry(mainDir string) *Registry {
	r := &Registry{
		filePath: filepath.Join(mainDir, "sub_registry_snapshot.json"),
		entries:  []SubEntry{},
	}
	// 忽略載入錯誤（檔案不存在時從空白開始）
	_ = r.Load()
	return r
}

// ──────────────────────────────────────────────
// 增刪查
// ──────────────────────────────────────────────

// Add 新增一筆 sub 記錄，ID 重複時回傳錯誤。
func (r *Registry) Add(entry SubEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.entries {
		if e.ID == entry.ID {
			return fmt.Errorf("sub ID 已存在: %s", entry.ID)
		}
	}
	r.entries = append(r.entries, entry)
	return r.saveLocked()
}

// Remove 依 ID 刪除 sub 記錄，找不到時回傳錯誤。
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	newEntries := make([]SubEntry, 0, len(r.entries))
	found := false
	for _, e := range r.entries {
		if e.ID == id {
			found = true
			continue
		}
		newEntries = append(newEntries, e)
	}
	if !found {
		return fmt.Errorf("找不到 sub ID: %s", id)
	}
	r.entries = newEntries
	return r.saveLocked()
}

// Get 依 ID 查詢 sub 記錄。
func (r *Registry) Get(id string) (SubEntry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.entries {
		if e.ID == id {
			return e, true
		}
	}
	return SubEntry{}, false
}

// List 回傳所有 sub 記錄的快照。
func (r *Registry) List() []SubEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]SubEntry, len(r.entries))
	copy(result, r.entries)
	return result
}

// FindByTag 依標籤搜尋 sub：先精確比對 Triggers + ActionTags，再進行 contains 比對。
func (r *Registry) FindByTag(tag string) []SubEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	var exact, contains []SubEntry
	tagLower := strings.ToLower(tag)

	for _, e := range r.entries {
		// 精確比對
		if matchExact(e.Triggers, tag) || matchExact(e.ActionTags, tag) {
			exact = append(exact, e)
			continue
		}
		// Contains 比對（小寫）
		if matchContains(e.Triggers, tagLower) || matchContains(e.ActionTags, tagLower) {
			contains = append(contains, e)
		}
	}

	return append(exact, contains...)
}

// ──────────────────────────────────────────────
// 持久化
// ──────────────────────────────────────────────

// Save 將當前 entries 寫入 sub_registry_snapshot.json。
func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveLocked()
}

// Load 從 sub_registry_snapshot.json 讀取 entries。
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.entries = []SubEntry{}
			return nil
		}
		return fmt.Errorf("讀取 registry 失敗: %w", err)
	}

	var entries []SubEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("解析 registry 失敗: %w", err)
	}
	r.entries = entries
	return nil
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

// saveLocked 在已持有鎖的情況下寫入檔案。
func (r *Registry) saveLocked() error {
	data, err := json.MarshalIndent(r.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 registry 失敗: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return fmt.Errorf("建立目錄失敗: %w", err)
	}
	return os.WriteFile(r.filePath, data, 0o600)
}

// matchExact 檢查 slice 是否包含與 target 完全相符的項目。
func matchExact(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// matchContains 檢查 slice 是否包含含有 target（小寫）的項目。
func matchContains(slice []string, target string) bool {
	for _, s := range slice {
		if strings.Contains(strings.ToLower(s), target) {
			return true
		}
	}
	return false
}
