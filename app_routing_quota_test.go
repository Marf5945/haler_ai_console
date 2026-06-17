package main

import (
	"errors"
	"strings"
	"testing"

	"ui_console/orchestration/skill_step"
)

func TestOfflineChatReply(t *testing.T) {
	// 白名單命中：必須回非空定型回覆。
	hits := []string{
		"你好", "您好", "哈囉", "hi", "HELLO",
		"謝謝", "thanks",
		"收到嗎", "在嗎", "還在嗎",
		"收到", "好的", "ok", "OK", "了解",
		"掰掰", "再見", "bye", "晚安",
		"  你好。 ", "收到嗎？", // 正規化：去頭尾空白與標點仍應命中
	}
	for _, in := range hits {
		reply, ok := offlineChatReply(in)
		if !ok || strings.TrimSpace(reply) == "" {
			t.Fatalf("offlineChatReply(%q) 應命中白名單並回非空回覆, got ok=%v reply=%q", in, ok, reply)
		}
	}

	// 非白名單 / 需要上下文的句子：不可命中（避免亂答）。
	misses := []string{
		"",
		"第幾次",
		"請回答",
		"只回答",
		"幫我搜尋生動推薦文字模板.txt", // 長句剛好含「搜尋」也不可被白名單吃掉
		"你好嗎可以幫我做一份報告嗎",    // 超過長度上限的長句
		"今天天氣如何",
	}
	for _, in := range misses {
		if reply, ok := offlineChatReply(in); ok {
			t.Fatalf("offlineChatReply(%q) 不應命中白名單, got reply=%q", in, reply)
		}
	}
}

func TestIsRoutingQuotaHit(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		errText string
		err     error
		want    bool
	}{
		{"429 in text", "RetryableQuotaError: status 429", "", nil, true},
		{"resource_exhausted in error field", "", "RESOURCE_EXHAUSTED", nil, true},
		{"capacity in go error", "", "", errors.New("MODEL_CAPACITY_EXHAUSTED"), true},
		{"exhausted capacity phrase", "You have exhausted your capacity on this model", "", nil, true},
		{"normal result text", "本機搜尋「電料表」有找到 2 筆", "", nil, false},
		{"all empty", "", "", nil, false},
		{"unrelated error", "", "file not found", errors.New("no such file"), false},
	}
	for _, c := range cases {
		if got := isRoutingQuotaHit(c.text, c.errText, c.err); got != c.want {
			t.Fatalf("%s: isRoutingQuotaHit(%q,%q,%v) = %v, want %v",
				c.name, c.text, c.errText, c.err, got, c.want)
		}
	}
}

func TestRouteAfterRoutingQuotaHit_OfflineChat(t *testing.T) {
	app := &App{}
	want, _ := offlineChatReply("你好")
	resp, handled := app.routeAfterRoutingQuotaHit("gemini-cli", "sess-1", "你好", "trace-offline", nil)
	if !handled {
		t.Fatal("白名單寒暄句在 quota 命中後應由 fast-fail 直接回覆")
	}
	if resp == nil || resp.Text != want {
		t.Fatalf("回覆應等於 offlineChatReply 定型回覆 %q, got %q", want, respText(resp))
	}
}

func TestRouteAfterRoutingQuotaHit_SwitchModelWhenNoLocalMatch(t *testing.T) {
	app := &App{}
	// 傳入空 lookup（非 nil）強制走「無本機命中」分支，與檔案系統狀態無關，確保確定性。
	resp, handled := app.routeAfterRoutingQuotaHit("gemini-cli", "sess-2",
		"zzz-unlikely-routing-quota-query-9c1f", "trace-switch", &toolRoutingLookupContext{})
	if !handled {
		t.Fatal("無本機候選時仍應由 fast-fail 處理（回切換模型提示）")
	}
	if resp == nil || !strings.Contains(resp.Text, "gemini-cli") || !strings.Contains(resp.Text, "切換") {
		t.Fatalf("無本機命中應回提示切換模型且點名 adapter, got %q", respText(resp))
	}
}

func TestRouteAfterRoutingQuotaHit_NilLookupRebuildNoPanic(t *testing.T) {
	// keyword 階段命中（lookup 為 nil）：應用本機抽詞補建 lookup，不得 panic，
	// 且一定回傳已處理的非空回覆（本機保底或切換提示）。
	app := &App{}
	resp, handled := app.routeAfterRoutingQuotaHit("gemini-cli", "sess-3",
		"zzz-unlikely-routing-quota-query-3a7e", "trace-rebuild", nil)
	if !handled || resp == nil || strings.TrimSpace(resp.Text) == "" {
		t.Fatalf("nil lookup 補建路徑應回非空已處理回覆, handled=%v resp=%q", handled, respText(resp))
	}
}

func respText(r *skill_step.CLIResponse) string {
	if r == nil {
		return "<nil>"
	}
	return r.Text
}
