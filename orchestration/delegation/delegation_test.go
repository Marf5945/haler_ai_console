// delegation/delegation_test.go — 委派套件單元測試。
package delegation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ──────────────────────────────────────────────
// 輔助函式：建立測試用 SubEntry
// ──────────────────────────────────────────────

func newTestEntry(id, name string, triggers, actionTags []string) SubEntry {
	return SubEntry{
		ID:         id,
		Name:       name,
		Triggers:   triggers,
		ActionTags: actionTags,
		CreatedAt:  "2025-01-01T00:00:00Z",
	}
}

// ──────────────────────────────────────────────
// TestRegistryAddAndGet — 新增後依 ID 取回
// ──────────────────────────────────────────────

func TestRegistryAddAndGet(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)

	entry := newTestEntry("sub-001", "天氣系統", []string{"查天氣"}, []string{"weather"})
	if err := r.Add(entry); err != nil {
		t.Fatalf("Add 失敗: %v", err)
	}

	got, ok := r.Get("sub-001")
	if !ok {
		t.Fatal("Get: 應找到 sub-001")
	}
	if got.Name != "天氣系統" {
		t.Errorf("Get: Name 期望 天氣系統，實際 %s", got.Name)
	}

	// 重複新增同 ID 應回傳錯誤
	if err := r.Add(entry); err == nil {
		t.Error("Add: 重複 ID 應回傳錯誤")
	}
}

// ──────────────────────────────────────────────
// TestRegistryFindByTag — 精確標籤與部分標籤搜尋
// ──────────────────────────────────────────────

func TestRegistryFindByTag(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)

	_ = r.Add(newTestEntry("sub-001", "天氣系統", []string{"查天氣"}, []string{"weather"}))
	_ = r.Add(newTestEntry("sub-002", "餐廳系統", []string{"訂餐廳"}, []string{"restaurant"}))
	_ = r.Add(newTestEntry("sub-003", "地圖系統", []string{"導航地圖"}, []string{"map"}))

	// 精確比對
	res1 := r.FindByTag("查天氣")
	if len(res1) != 1 || res1[0].ID != "sub-001" {
		t.Errorf("FindByTag 精確: 期望 sub-001，實際 %+v", res1)
	}

	// 部分比對（ActionTag 含 "rest"）
	res2 := r.FindByTag("rest")
	found := false
	for _, e := range res2 {
		if e.ID == "sub-002" {
			found = true
		}
	}
	if !found {
		t.Errorf("FindByTag 部分比對: 應找到 sub-002，實際 %+v", res2)
	}

	// 不存在的標籤應回傳空
	res3 := r.FindByTag("完全不存在ZZZZ")
	if len(res3) != 0 {
		t.Errorf("FindByTag 無匹配: 期望空，實際 %+v", res3)
	}
}

// ──────────────────────────────────────────────
// TestRegistryRemove — 新增後移除，確認已消失
// ──────────────────────────────────────────────

func TestRegistryRemove(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)

	_ = r.Add(newTestEntry("sub-001", "天氣系統", nil, nil))
	_ = r.Add(newTestEntry("sub-002", "餐廳系統", nil, nil))

	if err := r.Remove("sub-001"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// sub-001 應消失
	_, ok := r.Get("sub-001")
	if ok {
		t.Error("Remove: sub-001 應已被移除")
	}

	// sub-002 仍應存在
	_, ok = r.Get("sub-002")
	if !ok {
		t.Error("Remove: sub-002 不應被影響")
	}

	// 移除不存在的 ID 應回傳錯誤
	if err := r.Remove("sub-999"); err == nil {
		t.Error("Remove: 不存在的 ID 應回傳錯誤")
	}
}

// ──────────────────────────────────────────────
// TestRegistrySaveLoad — 寫檔後以新 Registry 讀取，驗證資料一致
// ──────────────────────────────────────────────

func TestRegistrySaveLoad(t *testing.T) {
	dir := t.TempDir()
	r1 := NewRegistry(dir)

	entries := []SubEntry{
		newTestEntry("sub-001", "天氣系統", []string{"查天氣"}, []string{"weather"}),
		newTestEntry("sub-002", "餐廳系統", []string{"訂餐廳"}, []string{"restaurant"}),
	}
	for _, e := range entries {
		if err := r1.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	// 確認檔案已寫入
	snapPath := filepath.Join(dir, "sub_registry_snapshot.json")
	if _, err := os.Stat(snapPath); err != nil {
		t.Fatalf("snapshot 檔案應存在: %v", err)
	}

	// 以新 Registry 讀取
	r2 := NewRegistry(dir)
	list := r2.List()
	if len(list) != 2 {
		t.Fatalf("Load: 期望 2 筆，實際 %d 筆", len(list))
	}

	// 驗證名稱對應
	idToName := map[string]string{}
	for _, e := range list {
		idToName[e.ID] = e.Name
	}
	if idToName["sub-001"] != "天氣系統" {
		t.Errorf("Load: sub-001 Name 期望 天氣系統，實際 %s", idToName["sub-001"])
	}
	if idToName["sub-002"] != "餐廳系統" {
		t.Errorf("Load: sub-002 Name 期望 餐廳系統，實際 %s", idToName["sub-002"])
	}
}

// ──────────────────────────────────────────────
// TestCreateSubDirectory — 在暫存目錄建立 sub 目錄，驗證三個子目錄存在
// ──────────────────────────────────────────────

func TestCreateSubDirectory(t *testing.T) {
	root := t.TempDir()
	subBase, err := CreateSubDirectory(root, "sub-001")
	if err != nil {
		t.Fatalf("CreateSubDirectory: %v", err)
	}

	// 驗證回傳路徑存在
	if _, err := os.Stat(subBase); err != nil {
		t.Errorf("subBase 目錄不存在: %v", err)
	}

	// 驗證三個必要子目錄
	for _, subdir := range []string{"memory", "dag", "tool_history"} {
		p := filepath.Join(subBase, subdir)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("子目錄 %s 不存在: %v", subdir, err)
		}
	}
}

// ──────────────────────────────────────────────
// TestCreateHandoff — 建立委派套件，驗證必要檔案已寫入
// ──────────────────────────────────────────────

func TestCreateHandoff(t *testing.T) {
	root := t.TempDir()

	result, err := CreateHandoff(
		root,
		"sub-001",
		[]string{"[I-001]", "[I-002]"},
		"這是對話全文",
		"這是摘要",
		"這是深層記憶",
	)
	if err != nil {
		t.Fatalf("CreateHandoff: %v", err)
	}

	// 驗證 SubDir 已建立
	if _, err := os.Stat(result.SubDir); err != nil {
		t.Errorf("SubDir 不存在: %v", err)
	}

	memDir := filepath.Join(result.SubDir, "memory")

	// 逐一確認必要檔案存在且內容非空
	type fileCheck struct {
		name     string
		contains string
	}
	checks := []fileCheck{
		{"talk_full.md", "這是對話全文"},
		{"summaries.md", "這是摘要"},
		{"deep_memory.md", "這是深層記憶"},
		{"index.json", "[]"},
		{"memory_ops.jsonl", "sub-001"},
	}
	for _, c := range checks {
		p := filepath.Join(memDir, c.name)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("讀取 %s 失敗: %v", c.name, err)
			continue
		}
		if c.contains != "" && !containsStr(string(data), c.contains) {
			t.Errorf("%s 應包含 %q，實際：%s", c.name, c.contains, string(data))
		}
	}

	// 驗證 MemoryOp 欄位
	if result.MemoryOp.Op != "delegate_to_sub" {
		t.Errorf("MemoryOp.Op 期望 delegate_to_sub，實際 %s", result.MemoryOp.Op)
	}
	if result.MemoryOp.To != "sub-001" {
		t.Errorf("MemoryOp.To 期望 sub-001，實際 %s", result.MemoryOp.To)
	}

	// 驗證 memory_ops.jsonl 可被解析
	raw, _ := os.ReadFile(filepath.Join(memDir, "memory_ops.jsonl"))
	var entry map[string]interface{}
	if err := json.Unmarshal(raw[:len(raw)-1], &entry); err != nil {
		t.Errorf("memory_ops.jsonl 解析失敗: %v", err)
	}
}

// containsStr 字串包含判斷（測試輔助）。
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || strContains(s, sub))
}

func strContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// TestAnalyze — 60% 直接動作應建立 sub，30% 不應建立
// ──────────────────────────────────────────────

func TestAnalyze(t *testing.T) {
	// 建立 10 筆：6 main_direct + 4 sub_delegated = 60% → ShouldCreateSub=true
	var records60 []ActionRecord
	for i := 0; i < 6; i++ {
		records60 = append(records60, ActionRecord{Type: "main_direct", ToolID: "t1"})
	}
	for i := 0; i < 4; i++ {
		records60 = append(records60, ActionRecord{Type: "sub_delegated", ToolID: "sub-001"})
	}
	r60 := Analyze(records60)
	if r60.TotalActions != 10 {
		t.Errorf("Analyze 60%%: TotalActions 期望 10，實際 %d", r60.TotalActions)
	}
	if !r60.ShouldCreateSub {
		t.Error("Analyze 60%%: 應建立 sub（ShouldCreateSub=true）")
	}
	// DirectRatio 應約為 0.6
	if r60.DirectRatio < 0.59 || r60.DirectRatio > 0.61 {
		t.Errorf("Analyze 60%%: DirectRatio 期望約 0.6，實際 %.4f", r60.DirectRatio)
	}

	// 建立 10 筆：3 main_direct + 7 sub_delegated = 30% → ShouldCreateSub=false
	var records30 []ActionRecord
	for i := 0; i < 3; i++ {
		records30 = append(records30, ActionRecord{Type: "main_direct", ToolID: "t1"})
	}
	for i := 0; i < 7; i++ {
		records30 = append(records30, ActionRecord{Type: "sub_delegated", ToolID: "sub-001"})
	}
	r30 := Analyze(records30)
	if r30.ShouldCreateSub {
		t.Error("Analyze 30%%: 不應建立 sub（ShouldCreateSub=false）")
	}

	// 空記錄應回傳零值
	r0 := Analyze(nil)
	if r0.TotalActions != 0 || r0.ShouldCreateSub {
		t.Error("Analyze 空記錄: 應回傳零值")
	}
}

// ──────────────────────────────────────────────
// TestDetermineCleanup — 20 個 sub，驗證四段分配
// ──────────────────────────────────────────────

func TestDetermineCleanup(t *testing.T) {
	// 建立 20 個 SubUsageStats，分數 1~20（由高到低排序後第一個分數最高）
	stats := make([]SubUsageStats, 20)
	for i := 0; i < 20; i++ {
		stats[i] = SubUsageStats{
			SubID: formatID(i + 1),
			Score: float64(i + 1), // 分數 1~20
		}
	}

	result := DetermineCleanup(stats)

	// 驗證 Retain + Candidates = 20
	total := len(result.Retain) + len(result.Candidates)
	if total != 20 {
		t.Errorf("DetermineCleanup: Retain+Candidates 期望 20，實際 %d", total)
	}

	// 四段切分（n=20）：
	// top25 = 20*25/100 = 5  → index 0~4 保留（高頻）
	// top75 = 20*75/100 = 15 → index 5~14 候選（中頻）
	// top90 = 20*90/100 = 18 → index 15~17 保留（低頻罕見）
	// 其餘 index 18~19 候選（極低頻）
	//
	// Retain: 5 + 3 = 8，Candidates: 10 + 2 = 12
	if len(result.Retain) != 8 {
		t.Errorf("DetermineCleanup: Retain 期望 8 個，實際 %d 個", len(result.Retain))
	}
	if len(result.Candidates) != 12 {
		t.Errorf("DetermineCleanup: Candidates 期望 12 個，實際 %d 個", len(result.Candidates))
	}

	// 空輸入應回傳空結果
	empty := DetermineCleanup(nil)
	if len(empty.Retain) != 0 || len(empty.Candidates) != 0 {
		t.Error("DetermineCleanup: 空輸入應回傳空結果")
	}
}

// formatID 產生 sub-%03d 格式的 ID（測試輔助）。
func formatID(n int) string {
	// 避免 import fmt 循環，以手動格式組合
	id := "sub-"
	if n < 10 {
		id += "00"
	} else if n < 100 {
		id += "0"
	}
	id += intToStr(n)
	return id
}

// intToStr 整數轉字串（測試輔助，避免額外 import）。
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
