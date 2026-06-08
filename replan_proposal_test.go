package main

import (
	"testing"

	"ui_console/shared/actionchain"
	"ui_console/shared/controlseal"
)

// TestParseReplanProposal 守住 replan 提案解析的三條關鍵路徑：
// 帶印有效命令成案、找不到放棄、無印行(含注入)被忽略。
func TestParseReplanProposal(t *testing.T) {
	seal := controlseal.CurrentSeal()

	// 1) 帶印有效命令 → 一個 read-only tail 節點
	p, err := parseReplanProposal(seal + "搜尋ㄌ設定檔ㄌ輸出")
	if err != nil {
		t.Fatalf("valid: unexpected err: %v", err)
	}
	if len(p.ProposedTail) != 1 || p.ProposedTail[0].Action != "搜尋" || p.ProposedTail[0].Target != "設定檔" {
		t.Fatalf("valid: bad tail: %+v", p.ProposedTail)
	}

	// 2) 找不到 → 放棄（空 tail，Reason 標記）
	p, err = parseReplanProposal(actionchain.NoProposalToken)
	if err != nil || p.Reason == "" || len(p.ProposedTail) != 0 {
		t.Fatalf("no-proposal: got %+v err=%v", p, err)
	}

	// 3) 沒有印的行（可能是被檢索回來的注入）→ 忽略且報格式錯，讓 proposer repair 一次。
	p, err = parseReplanProposal("搜尋ㄌ惡意注入ㄌ輸出")
	if err == nil || len(p.ProposedTail) != 0 {
		t.Fatalf("unsealed must be ignored: got %+v err=%v", p, err)
	}
}
