// skill_usage_recorder_test.go — skill 使用紀錄 producer 的單元測試。
// 全部走 t.TempDir()，不碰真實 appDataRoot，不需要 App 實例。
//
// 執行：go test -run TestSkillUsage -v
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillUsageAppendAndLoad 驗證 append → load 的來回，
// 以及 append-only：兩次寫入後兩筆都在、順序不變。
func TestSkillUsageAppendAndLoad(t *testing.T) {
	root := t.TempDir()

	rec1 := SkillUsageRecord{
		SchemaVersion: skillUsageSchemaVersion,
		EventType:     "skill_usage",
		SkillID:       "go-program-產出電料bom",
		DisplayName:   "產出電料Bom",
		AgentID:       "main",
		ExecutedAt:    "2026-06-11T19:00:00+08:00",
	}
	rec2 := rec1
	rec2.SkillID = "dressing.advice"
	rec2.DisplayName = "穿衣建議"

	if err := appendSkillUsageRecord(root, rec1); err != nil {
		t.Fatalf("append rec1: %v", err)
	}
	if err := appendSkillUsageRecord(root, rec2); err != nil {
		t.Fatalf("append rec2: %v", err)
	}

	got := loadSkillUsageRecords(root)
	if len(got) != 2 {
		t.Fatalf("want 2 records, got %d", len(got))
	}
	if got[0].SkillID != rec1.SkillID || got[1].SkillID != rec2.SkillID {
		t.Errorf("順序或內容不符: %+v", got)
	}
}

// TestSkillUsageLoadSkipsCorruptLines 驗證壞行（手改/截斷）只跳過該行。
func TestSkillUsageLoadSkipsCorruptLines(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "tool_history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"schema_version":"v1","event_type":"skill_usage","skill_id":"a","display_name":"A","agent_id":"main","executed_at":"x"}
{not valid json
{"schema_version":"v1","event_type":"skill_usage","skill_id":"b","display_name":"B","agent_id":"main","executed_at":"y"}
`
	if err := os.WriteFile(filepath.Join(dir, toolHistoryFileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got := loadSkillUsageRecords(root)
	if len(got) != 2 || got[0].SkillID != "a" || got[1].SkillID != "b" {
		t.Fatalf("want [a b], got %+v", got)
	}
}

// TestUpdateSubMetaToolsUsed 驗證：
//  1. main（無 sub_meta.json）→ no-op、不報錯
//  2. sub → 併入去重，原有欄位保留
func TestUpdateSubMetaToolsUsed(t *testing.T) {
	// case 1: 無 sub_meta.json
	if err := updateSubMetaToolsUsed(t.TempDir(), "x"); err != nil {
		t.Fatalf("main 應為 no-op，卻報錯: %v", err)
	}

	// case 2: 有 sub_meta.json
	root := t.TempDir()
	meta := map[string]interface{}{
		"id":         "sub-test",
		"name":       "測試",
		"tools_used": []string{"existing"},
	}
	data, _ := json.Marshal(meta)
	metaPath := filepath.Join(root, "sub_meta.json")
	if err := os.WriteFile(metaPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := updateSubMetaToolsUsed(root, "go-program-產出電料bom", "產出電料Bom", "existing", ""); err != nil {
		t.Fatalf("update: %v", err)
	}
	out, _ := os.ReadFile(metaPath)
	var loaded map[string]interface{}
	if err := json.Unmarshal(out, &loaded); err != nil {
		t.Fatal(err)
	}
	raw, _ := loaded["tools_used"].([]interface{})
	var tools []string
	for _, v := range raw {
		tools = append(tools, v.(string))
	}
	want := []string{"existing", "go-program-產出電料bom", "產出電料Bom"}
	if strings.Join(tools, ",") != strings.Join(want, ",") {
		t.Errorf("tools_used = %v, want %v", tools, want)
	}
	if loaded["name"] != "測試" {
		t.Errorf("其他欄位應保留, name = %v", loaded["name"])
	}
}

// TestCopySkillUsageRecordsFilter 驗證拉出 sub 時的過濾規則：
//   - 對話有提到（顯示名稱出現在 talk）→ 帶入
//   - 沒提到 → 不帶
//   - talkContent 空 → 全帶
//   - 同 skill 多筆 → 去重只帶一筆
func TestCopySkillUsageRecordsFilter(t *testing.T) {
	src := t.TempDir()
	mk := func(id, name string) SkillUsageRecord {
		return SkillUsageRecord{
			SchemaVersion: skillUsageSchemaVersion, EventType: "skill_usage",
			SkillID: id, DisplayName: name, AgentID: "main", ExecutedAt: "t",
		}
	}
	// bom 寫兩筆（測去重）、dressing 一筆（talk 沒提到，不該帶）
	for _, rec := range []SkillUsageRecord{
		mk("go-program-產出電料bom", "產出電料Bom"),
		mk("go-program-產出電料bom", "產出電料Bom"),
		mk("dressing.advice", "穿衣建議"),
	} {
		if err := appendSkillUsageRecord(src, rec); err != nil {
			t.Fatal(err)
		}
	}

	talk := "## user\n幫我產出電料bom，資料庫為電料編碼紀錄\n## assistant\n已產出電料BOM"

	dst := t.TempDir()
	n, idents := copySkillUsageRecords(src, dst, talk)
	if n != 1 {
		t.Fatalf("want 1 copied (bom 去重、dressing 過濾), got %d", n)
	}
	got := loadSkillUsageRecords(dst)
	if len(got) != 1 || got[0].SkillID != "go-program-產出電料bom" {
		t.Fatalf("dst records = %+v", got)
	}
	if len(idents) != 2 { // skill_id + display_name
		t.Errorf("idents = %v", idents)
	}

	// talkContent 空 → 全帶（bom 一筆 + dressing 一筆，各去重後共 2）
	dst2 := t.TempDir()
	n2, _ := copySkillUsageRecords(src, dst2, "")
	if n2 != 2 {
		t.Errorf("空 talk 應全帶（去重後 2 筆）, got %d", n2)
	}
}

// TestUniqueNonEmptyStrings 驗證去重、去空白、保序、nil 輸入回空切片（非 nil）。
func TestUniqueNonEmptyStrings(t *testing.T) {
	got := uniqueNonEmptyStrings([]string{"a", " ", "b", "a", "", "c", "b"})
	if strings.Join(got, ",") != "a,b,c" {
		t.Errorf("got %v", got)
	}
	empty := uniqueNonEmptyStrings(nil)
	if empty == nil || len(empty) != 0 {
		t.Errorf("nil 輸入應回空切片（JSON 才會是 [] 而非 null）, got %#v", empty)
	}
}
