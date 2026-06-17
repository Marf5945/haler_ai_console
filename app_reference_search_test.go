package main

import (
	"testing"

	"ui_console/shared/localsearch"
)

func TestIsReferenceListingQuestion(t *testing.T) {
	listing := []string{
		"我有哪些引用檔",
		"列出已載入的檔案",
		"你看到哪些檔案",
		"拉進來的有幾個檔",
		"what files are loaded",
	}
	for _, in := range listing {
		if !isReferenceListingQuestion(in) {
			t.Fatalf("isReferenceListingQuestion(%q) 應為列檔意圖", in)
		}
	}
	// 內容查詢 / 非檔案問句不算列檔。
	notListing := []string{
		"搜尋引用文件裡的折扣碼",        // 帶「搜尋/內容」→ 查內容
		"生動推薦文字模板.txt 裡面寫什麼", // 問內容
		"引用檔關於 BOM 的部分",      // 帶「關於」
		"今天天氣如何",             // 與檔案無關
		"",
	}
	for _, in := range notListing {
		if isReferenceListingQuestion(in) {
			t.Fatalf("isReferenceListingQuestion(%q) 不應判為列檔意圖", in)
		}
	}
}

func TestIsReferenceSearchSentinel(t *testing.T) {
	if !isReferenceSearchSentinel("引用文件") || !isReferenceSearchSentinel("  引用文件 ") {
		t.Fatal("「引用文件」應為 sentinel")
	}
	for _, in := range []string{"", "引用", "文件", "電料表", "引用文件內容"} {
		if isReferenceSearchSentinel(in) {
			t.Fatalf("%q 不應為 sentinel", in)
		}
	}
}

func TestNamedReferenceFile(t *testing.T) {
	refs := []routingReferenceFile{
		{Name: "生動推薦文字模板.txt", Ext: "txt"},
		{Name: "電料BOM-260327M1.xlsx", Ext: "xlsx"},
	}
	// 含副檔名
	if got := namedReferenceFile("幫我看 生動推薦文字模板.txt 裡的折扣", refs); got != "生動推薦文字模板.txt" {
		t.Fatalf("應命中含副檔名檔名, got %q", got)
	}
	// 不含副檔名（stem）
	if got := namedReferenceFile("生動推薦文字模板 裡寫什麼", refs); got != "生動推薦文字模板.txt" {
		t.Fatalf("應以 stem 命中, got %q", got)
	}
	// 未提及任何檔名
	if got := namedReferenceFile("隨便找點東西", refs); got != "" {
		t.Fatalf("未提檔名應回空, got %q", got)
	}
}

func TestFilterResultsByFileName(t *testing.T) {
	results := []localsearch.SearchResult{
		{Title: "生動推薦文字模板", Path: "/data/references/files/生動推薦文字模板.txt"},
		{Title: "其他文件", Path: "/data/documents/別的.txt"},
	}
	got := filterResultsByFileName(results, "生動推薦文字模板.txt")
	if len(got) != 1 || got[0].Path != "/data/references/files/生動推薦文字模板.txt" {
		t.Fatalf("應只留指定檔, got %#v", got)
	}
}

func TestFilterOutReferenceListingItems(t *testing.T) {
	results := []localsearch.SearchResult{
		{Title: "最近引用文件: 生動推薦文字模板.txt", Path: "/x.txt"},
		{Title: "生動推薦文字模板（內容片段）", Path: "/y.txt"},
	}
	got := filterOutReferenceListingItems(results)
	if len(got) != 1 || got[0].Path != "/y.txt" {
		t.Fatalf("應濾掉合成列檔項，只留真實內容, got %#v", got)
	}
}
