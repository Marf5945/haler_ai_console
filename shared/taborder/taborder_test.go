// taborder/taborder_test.go — taborder 套件測試。
package taborder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ──────────────────────────────────────────────
// 基本讀寫測試
// ──────────────────────────────────────────────

func TestNewManager_DefaultState(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	order := m.GetOrder()

	if order.MainHandler != "main" {
		t.Errorf("MainHandler 應為 main，得到: %s", order.MainHandler)
	}
	if len(order.SubOrder) != 0 {
		t.Errorf("初始 SubOrder 應為空，得到 %d 項", len(order.SubOrder))
	}
}

func TestManager_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Append("sub_A")
	m.Append("sub_B")

	// 重新載入
	m2 := NewManager(dir)
	order := m2.GetOrder()

	if len(order.SubOrder) != 2 {
		t.Fatalf("應有 2 項，得到 %d", len(order.SubOrder))
	}
	if order.SubOrder[0] != "sub_A" || order.SubOrder[1] != "sub_B" {
		t.Errorf("順序不符: %v", order.SubOrder)
	}
}

// ──────────────────────────────────────────────
// Append / Remove 測試
// ──────────────────────────────────────────────

func TestManager_Append(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	if err := m.Append("sub_1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Append("sub_2"); err != nil {
		t.Fatal(err)
	}

	order := m.GetOrder()
	if len(order.SubOrder) != 2 {
		t.Errorf("應有 2 項: %v", order.SubOrder)
	}
}

func TestManager_Append_Duplicate(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Append("sub_X")

	if err := m.Append("sub_X"); err == nil {
		t.Error("重複 Append 應回傳錯誤")
	}
}

func TestManager_Remove(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Append("sub_A")
	m.Append("sub_B")
	m.Append("sub_C")

	if err := m.Remove("sub_B"); err != nil {
		t.Fatal(err)
	}

	order := m.GetOrder()
	if len(order.SubOrder) != 2 {
		t.Fatalf("應剩 2 項: %v", order.SubOrder)
	}
	if order.SubOrder[0] != "sub_A" || order.SubOrder[1] != "sub_C" {
		t.Errorf("移除後順序不符: %v", order.SubOrder)
	}
}

func TestManager_Remove_NotFound(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	if err := m.Remove("nonexistent"); err == nil {
		t.Error("移除不存在的 sub 應回傳錯誤")
	}
}

// ──────────────────────────────────────────────
// Move 測試
// ──────────────────────────────────────────────

func TestManager_Move(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Append("A")
	m.Append("B")
	m.Append("C")

	// 將 A (index 0) 移到 index 2
	if err := m.Move(0, 2); err != nil {
		t.Fatal(err)
	}

	order := m.GetOrder()
	expected := []string{"B", "C", "A"}
	for i, v := range expected {
		if order.SubOrder[i] != v {
			t.Errorf("位置 %d: 期望 %s，得到 %s", i, v, order.SubOrder[i])
		}
	}
}

func TestManager_Move_OutOfRange(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Append("X")

	if err := m.Move(0, 5); err == nil {
		t.Error("超出範圍的 Move 應回傳錯誤")
	}
}

// ──────────────────────────────────────────────
// Reorder 測試
// ──────────────────────────────────────────────

func TestManager_Reorder(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Append("A")
	m.Append("B")
	m.Append("C")

	if err := m.Reorder([]string{"C", "A", "B"}); err != nil {
		t.Fatal(err)
	}

	order := m.GetOrder()
	if order.SubOrder[0] != "C" || order.SubOrder[1] != "A" || order.SubOrder[2] != "B" {
		t.Errorf("Reorder 後順序不符: %v", order.SubOrder)
	}
}

// ──────────────────────────────────────────────
// JSON 格式驗證
// ──────────────────────────────────────────────

func TestManager_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)
	m.Append("sub_1")
	m.Save()

	data, err := os.ReadFile(filepath.Join(dir, "tab_order.json"))
	if err != nil {
		t.Fatal(err)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("JSON 格式不正確: %v", err)
	}

	if obj["main_handler"] != "main" {
		t.Errorf("main_handler 不符: %v", obj["main_handler"])
	}
	subOrder, ok := obj["sub_order"].([]interface{})
	if !ok || len(subOrder) != 1 {
		t.Errorf("sub_order 格式不符: %v", obj["sub_order"])
	}
	if _, ok := obj["updated_at"]; !ok {
		t.Error("缺少 updated_at 欄位")
	}
}
