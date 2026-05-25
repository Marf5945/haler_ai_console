// taborder/order.go — Tab 排序持久化（§31.6）。
// 管理 tab_order.json：main 永遠在最左，sub 可拖曳重排。
package taborder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// TabOrder tab 排序狀態。
type TabOrder struct {
	MainHandler string   `json:"main_handler"` // 永遠為 "main"
	SubOrder    []string `json:"sub_order"`     // sub 系統碼的顯示順序（左→右）
	UpdatedAt   string   `json:"updated_at"`    // RFC3339 最後更新時間
}

// Manager tab 排序管理器。
type Manager struct {
	mu       sync.Mutex
	filePath string
	order    TabOrder
}

// ──────────────────────────────────────────────
// 建構與載入
// ──────────────────────────────────────────────

// NewManager 建立管理器，從 projectRoot 載入 tab_order.json。
func NewManager(projectRoot string) *Manager {
	m := &Manager{
		filePath: filepath.Join(projectRoot, "tab_order.json"),
		order: TabOrder{
			MainHandler: "main",
			SubOrder:    []string{},
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
	}
	_ = m.Load()
	return m
}

// ──────────────────────────────────────────────
// 讀寫操作
// ──────────────────────────────────────────────

// Load 從 tab_order.json 載入排序狀態。
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 初次使用，使用預設值
		}
		return fmt.Errorf("讀取 tab_order.json 失敗: %w", err)
	}

	var order TabOrder
	if err := json.Unmarshal(data, &order); err != nil {
		return fmt.Errorf("解析 tab_order.json 失敗: %w", err)
	}
	m.order = order
	return nil
}

// Save 將排序狀態寫入 tab_order.json。
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	m.order.UpdatedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(m.order, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 tab_order 失敗: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return fmt.Errorf("建立目錄失敗: %w", err)
	}
	return os.WriteFile(m.filePath, data, 0o600)
}

// ──────────────────────────────────────────────
// 排序操作
// ──────────────────────────────────────────────

// GetOrder 回傳當前排序快照。
func (m *Manager) GetOrder() TabOrder {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := TabOrder{
		MainHandler: m.order.MainHandler,
		SubOrder:    make([]string, len(m.order.SubOrder)),
		UpdatedAt:   m.order.UpdatedAt,
	}
	copy(result.SubOrder, m.order.SubOrder)
	return result
}

// Append 將新 sub 附加到末尾。
func (m *Manager) Append(systemCode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 檢查重複
	for _, code := range m.order.SubOrder {
		if code == systemCode {
			return fmt.Errorf("sub 已存在於 tab order: %s", systemCode)
		}
	}
	m.order.SubOrder = append(m.order.SubOrder, systemCode)
	return m.saveLocked()
}

// Remove 從排序中移除指定 sub。
func (m *Manager) Remove(systemCode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newOrder := make([]string, 0, len(m.order.SubOrder))
	found := false
	for _, code := range m.order.SubOrder {
		if code == systemCode {
			found = true
			continue
		}
		newOrder = append(newOrder, code)
	}
	if !found {
		return fmt.Errorf("找不到 sub: %s", systemCode)
	}
	m.order.SubOrder = newOrder
	return m.saveLocked()
}

// Move 將 sub 從 fromIndex 移動到 toIndex（0-based，相對於 sub_order）。
func (m *Manager) Move(fromIndex, toIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	n := len(m.order.SubOrder)
	if fromIndex < 0 || fromIndex >= n {
		return fmt.Errorf("fromIndex 超出範圍: %d (共 %d 項)", fromIndex, n)
	}
	if toIndex < 0 || toIndex >= n {
		return fmt.Errorf("toIndex 超出範圍: %d (共 %d 項)", toIndex, n)
	}
	if fromIndex == toIndex {
		return nil
	}

	// 取出元素
	item := m.order.SubOrder[fromIndex]
	// 移除
	m.order.SubOrder = append(m.order.SubOrder[:fromIndex], m.order.SubOrder[fromIndex+1:]...)
	// 插入到新位置
	newOrder := make([]string, 0, n)
	newOrder = append(newOrder, m.order.SubOrder[:toIndex]...)
	newOrder = append(newOrder, item)
	newOrder = append(newOrder, m.order.SubOrder[toIndex:]...)
	m.order.SubOrder = newOrder

	return m.saveLocked()
}

// Reorder 直接設定 sub 順序（用於拖曳重排後的批次更新）。
func (m *Manager) Reorder(newSubOrder []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.order.SubOrder = newSubOrder
	return m.saveLocked()
}
