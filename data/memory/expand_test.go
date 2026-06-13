package memory

import (
	"strings"
	"testing"
)

// 寫入 → tag 撈回 roundtrip；S-tag 經 index 對到 D-tag。
func TestDeepMemoryLookupByTag(t *testing.T) {
	p := NewPipeline(t.TempDir())
	if _, err := p.AppendDeepMemory("D-111", "甲方報價 3500，含三次修改"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := p.AppendIndexEntry(MemoryIndexEntry{SummaryTag: "S-111", DeepTag: "D-111"}); err != nil {
		t.Fatalf("index: %v", err)
	}
	for _, q := range []string{"D-111", "S-111", "[S-111: 2026-06-12T10:00]"} {
		got, err := p.LookupByTag(q)
		if err != nil || !strings.Contains(got, "3500") {
			t.Fatalf("lookup %q: %v / %q", q, err, got)
		}
	}
	if _, err := p.LookupByTag("D-999"); err == nil {
		t.Fatal("missing tag should error")
	}
}

// index 缺失時 S-tag 用同號 D-tag fallback。
func TestLookupSummaryTagFallbackWithoutIndex(t *testing.T) {
	p := NewPipeline(t.TempDir())
	_, _ = p.AppendDeepMemory("D-222", "細節內容")
	if got, err := p.LookupByTag("S-222"); err != nil || !strings.Contains(got, "細節內容") {
		t.Fatalf("fallback failed: %v / %q", err, got)
	}
}

// 關鍵字搜尋：AND 邏輯、新段優先、截斷不切壞 UTF-8。
func TestSearchDeepMemory(t *testing.T) {
	p := NewPipeline(t.TempDir())
	_, _ = p.AppendDeepMemory("D-1", "舊段：報價 3500 元")
	_, _ = p.AppendDeepMemory("D-2", "新段：報價改為 4200 元，含維護")
	hits, err := p.SearchDeepMemory("報價 維護", 3, 2048)
	if err != nil || len(hits) != 1 || !strings.Contains(hits[0], "4200") {
		t.Fatalf("AND search wrong: %v %v", err, hits)
	}
	hits, _ = p.SearchDeepMemory("報價", 3, 2048)
	if len(hits) != 2 || !strings.Contains(hits[0], "4200") {
		t.Fatalf("newest-first wrong: %v", hits)
	}
	long := strings.Repeat("資料", 2000)
	_, _ = p.AppendDeepMemory("D-3", long)
	hits, _ = p.SearchDeepMemory("資料", 1, 300)
	if len(hits) != 1 || len(hits[0]) > 400 || strings.Contains(hits[0], "�") {
		t.Fatalf("truncate wrong: len=%d", len(hits[0]))
	}
}

// tag 正規化：容忍整段 [S-NNN: ts] 貼入；非 tag 回空走關鍵字。
func TestNormalizeMemoryTag(t *testing.T) {
	if NormalizeMemoryTag("[D-42: 2026-06-12T09:00]") != "D-42" {
		t.Fatal("bracket form")
	}
	if NormalizeMemoryTag("報價細節") != "" {
		t.Fatal("keyword should not parse as tag")
	}
}
