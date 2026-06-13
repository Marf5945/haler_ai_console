package main

import (
	"strings"
	"testing"
)

// TestSkillConfirmMatcherInvariants 鎖住迴圈修復所依賴的關鍵不變量：
// 「要/好/可以」算肯定、「取消/不要」算否定，且兩者不重疊（避免「要」被誤判成取消）。
func TestSkillConfirmMatcherInvariants(t *testing.T) {
	for _, yes := range []string{"要", "好", "好的", "可以", "是", "確認", "ok", "yes"} {
		if !confirmRe.MatchString(strings.ToLower(yes)) {
			t.Fatalf("confirmRe 應視 %q 為肯定", yes)
		}
		if isDeclineText(strings.ToLower(yes)) {
			t.Fatalf("%q 不應被當成否定", yes)
		}
	}
	for _, no := range []string{"取消", "不要", "不用", "算了", "cancel"} {
		if !isDeclineText(strings.ToLower(no)) {
			t.Fatalf("isDeclineText 應視 %q 為否定", no)
		}
		if confirmRe.MatchString(strings.ToLower(no)) {
			t.Fatalf("%q 不應被當成肯定", no)
		}
	}
}

// TestPendingSkillConfirm_NoneFallsThrough 沒有待確認狀態時不攔截。
func TestPendingSkillConfirm_NoneFallsThrough(t *testing.T) {
	a := &App{}
	sid := "sess-confirm-none"
	clearPendingSkillConfirm(sid)
	if resp, handled := a.maybeHandlePendingSkillConfirm("要", sid, "trace"); handled || resp != nil {
		t.Fatalf("無 pending 時應回 (nil,false)，got handled=%v resp=%v", handled, resp)
	}
}

// TestPendingSkillConfirm_DeclineClears 否定回覆 → 回「已取消」並清除狀態。
func TestPendingSkillConfirm_DeclineClears(t *testing.T) {
	a := &App{}
	sid := "sess-confirm-decline"
	rememberPendingSkillConfirm(sid, pendingSkillConfirm{
		ResolveID: "r1", SkillID: "s1", Target: "產出電料Bom", AdapterID: "claude-cli", OriginalText: "程式ㄌ產出電料Bom",
	})
	resp, handled := a.maybeHandlePendingSkillConfirm("取消", sid, "trace")
	if !handled || resp == nil {
		t.Fatalf("否定回覆應被攔截，got handled=%v", handled)
	}
	if !strings.Contains(resp.Text, "取消") {
		t.Fatalf("取消回覆文字 = %q", resp.Text)
	}
	// 狀態應已清除：下一句不再被攔截。
	if _, handled2 := a.maybeHandlePendingSkillConfirm("要", sid, "trace"); handled2 {
		t.Fatalf("取消後 pending 應已清除，但仍被攔截")
	}
}

// TestPendingSkillConfirm_UnrelatedKeepsPending 既非肯定也非否定 → 不攔截、保留 pending，
// 避免使用者岔題時被卡在確認；之後仍能用「要」確認。
func TestPendingSkillConfirm_UnrelatedKeepsPending(t *testing.T) {
	a := &App{}
	sid := "sess-confirm-unrelated"
	defer clearPendingSkillConfirm(sid)
	rememberPendingSkillConfirm(sid, pendingSkillConfirm{ResolveID: "r2", Target: "產出電料Bom"})
	if resp, handled := a.maybeHandlePendingSkillConfirm("幫我查天氣", sid, "trace"); handled || resp != nil {
		t.Fatalf("岔題訊息不應被確認閘攔截，got handled=%v", handled)
	}
	// pending 應仍在。
	pendingSkillConfirmMu.Lock()
	_, still := pendingSkillConfirms[sid]
	pendingSkillConfirmMu.Unlock()
	if !still {
		t.Fatalf("岔題後 pending 應保留待後續確認")
	}
}

// TestSkillConfirmPromptNoFiles 沒有已載入引用檔時，提示應請使用者先載入資料（修「沒跟我要資料」）。
func TestSkillConfirmPromptNoFiles(t *testing.T) {
	a := &App{}
	msg := a.skillConfirmPrompt("產出電料Bom")
	if !strings.Contains(msg, "產出電料Bom") {
		t.Fatalf("提示應點名 skill：%q", msg)
	}
	// 測試環境通常無引用檔目錄 → 應帶「先拖入或引用檔案」提醒；若環境剛好有檔案則略過此斷言。
	if !strings.Contains(msg, "回覆「要」") {
		t.Fatalf("提示應包含確認指引：%q", msg)
	}
}
