package main

// SEC-15 驗證：搜尋出境閘門。不依賴 App 初始化，只測純邏輯。

import (
	"strings"
	"testing"
	"time"

	"ui_console/data/memory"
)

func TestEgressDetectsAPIKeyInQuery(t *testing.T) {
	secret := fakeSearchOpenAIKey()
	masked, records := memory.RedactBeforeWrite("幫我查 " + secret + " 是哪家的金鑰")
	if len(records) == 0 {
		t.Fatal("API 金鑰應被偵測")
	}
	if strings.Contains(masked, secret) {
		t.Fatal("遮蔽版不應殘留原始金鑰")
	}
	if !strings.Contains(masked, "[REDACTED:") {
		t.Fatal("遮蔽版應含遮蔽標記")
	}
}

func TestEgressCleanQueryPasses(t *testing.T) {
	_, records := memory.RedactBeforeWrite("台北 明天 天氣")
	if len(records) != 0 {
		t.Fatalf("一般查詢不應命中，got %d", len(records))
	}
}

func TestDescribeEgressHitsNoLeak(t *testing.T) {
	secret := fakeSearchOpenAIKey()
	_, records := memory.RedactBeforeWrite("query with " + secret)
	desc := describeEgressHits(records)
	if desc == "" {
		t.Fatal("描述不應為空")
	}
	if strings.Contains(desc, secret) || strings.Contains(desc, "abcdefghij") {
		t.Fatal("描述洩漏了原始機密值")
	}
	if !strings.Contains(desc, "×") {
		t.Fatalf("描述應含數量，got %q", desc)
	}
}

func fakeSearchOpenAIKey() string {
	return "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
}

func TestPendingSearchEgressLifecycle(t *testing.T) {
	const sid = "sec15-test-session"
	t.Cleanup(func() { clearPendingSearchEgress(sid) })

	// 未存在
	if _, ok := loadPendingSearchEgress(sid); ok {
		t.Fatal("不應有 pending")
	}
	// 存入後可讀，且只存遮蔽版
	storePendingSearchEgress(sid, "查 [REDACTED:pattern:OpenAI] 是哪家", 5)
	p, ok := loadPendingSearchEgress(sid)
	if !ok || !strings.Contains(p.MaskedQuery, "[REDACTED:") {
		t.Fatalf("pending 異常: %+v ok=%v", p, ok)
	}
	// 清除
	clearPendingSearchEgress(sid)
	if _, ok := loadPendingSearchEgress(sid); ok {
		t.Fatal("清除後不應殘留")
	}
	// 過期自動清除
	pendingSearchEgressMu.Lock()
	pendingSearchEgresses[sid] = pendingSearchEgress{MaskedQuery: "x", ExpiresAt: time.Now().Add(-time.Second)}
	pendingSearchEgressMu.Unlock()
	if _, ok := loadPendingSearchEgress(sid); ok {
		t.Fatal("過期 pending 應視為不存在")
	}
	if _, exists := pendingSearchEgresses[sid]; exists {
		t.Fatal("過期 pending 應被順手刪除")
	}
}

func TestEgressConfirmAndDeclineWords(t *testing.T) {
	// 與 URL fetch 共用的確認／取消詞應涵蓋常見口語。
	for _, yes := range []string{"好", "好的", "確認", "ok", "可以"} {
		if !confirmRe.MatchString(yes) {
			t.Errorf("確認詞 %q 應被接受", yes)
		}
	}
	for _, no := range []string{"取消", "不用", "算了", "cancel"} {
		if !isDeclineText(no) {
			t.Errorf("取消詞 %q 應被接受", no)
		}
	}
}
