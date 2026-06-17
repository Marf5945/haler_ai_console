package statusrail

import "testing"

// ClearToIdle 必須把「執行中」文字收回 idle 層，避免搜尋完成後頂部停在「正在搜尋…」。
func TestClearToIdleResetsRunningStatus(t *testing.T) {
	s := NewService(t.TempDir(), []string{"Hi 主人"})

	running := s.SetText("正在搜尋本機資料「電料表」…")
	if running.Layer != LayerUser || running.Text != "正在搜尋本機資料「電料表」…" {
		t.Fatalf("SetText 應設執行中狀態, got layer=%s text=%q", running.Layer, running.Text)
	}

	idle := s.ClearToIdle()
	if idle.Layer != LayerIdle {
		t.Fatalf("ClearToIdle 後應為 idle 層, got %s", idle.Layer)
	}
	if idle.Text == "正在搜尋本機資料「電料表」…" {
		t.Fatalf("ClearToIdle 後不應再停在執行中文字, got %q", idle.Text)
	}
}
