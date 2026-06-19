package localsearch

import (
	"strings"
	"testing"
)

// 跨 root 重複（同檔名＋同內容）應只保留排序後第一筆（分數最高）。
func TestDedupSearchResults(t *testing.T) {
	in := []SearchResult{
		{Title: "Python 教學", Path: "/x/data/documents/py.txt", Snippet: "變數 x=10", Score: 50},
		{Title: "Python 教學", Path: "/x/data/references/files/py.txt", Snippet: "變數 x=10", Score: 40},
		{Title: "別的文件", Path: "/x/other.txt", Snippet: "xyz", Score: 30},
	}
	out := dedupSearchResults(in)
	if len(out) != 2 {
		t.Fatalf("同檔重覆應去重, got %d 筆: %#v", len(out), out)
	}
	if out[0].Score != 50 {
		t.Fatalf("去重應保留分數最高的第一筆, got score=%d", out[0].Score)
	}
}

// 「本機搜尋 X」應被視為直查命令（跳過 routing 兩次模型呼叫）。
func TestParseUserQueryLocalSearchPrefix(t *testing.T) {
	req, ok := ParseUserQuery("本機搜尋 電料表")
	if !ok || !strings.Contains(req.Query, "電料表") {
		t.Fatalf("『本機搜尋 X』應為直查命令, ok=%v query=%q", ok, req.Query)
	}
	if _, ok := ParseUserQuery("搜尋 變數"); !ok {
		t.Fatal("『搜尋 X』仍應為直查命令")
	}
	if _, ok := ParseUserQuery("今天天氣如何"); ok {
		t.Fatal("非搜尋指令不應命中直查")
	}
}

// InvalidateAll 必須清空 root index 快取。
func TestInvalidateAllClearsCache(t *testing.T) {
	rootIndexCache.Lock()
	rootIndexCache.entries["dummy-key"] = cachedRootIndex{}
	rootIndexCache.Unlock()

	InvalidateAll()

	rootIndexCache.Lock()
	n := len(rootIndexCache.entries)
	rootIndexCache.Unlock()
	if n != 0 {
		t.Fatalf("InvalidateAll 後快取應為空, got %d 筆", n)
	}
}
