package actionchain

import "testing"

func TestParseSingleLine(t *testing.T) {
	if r := ParseSingleLine("查詢ㄌapp 設定"); r.Err != nil || r.NoProposal || r.Chain.Action != "查詢" || r.Chain.Target != "app 設定" {
		t.Fatalf("valid single failed: %+v", r)
	}
	if r := ParseSingleLine("找不到"); !r.NoProposal {
		t.Errorf("找不到 should be NoProposal")
	}
	// 對齊真實 3 段格式：動作ㄌ目標ㄌ下一步 應被接受，取 動作+目標。
	if r := ParseSingleLine("搜尋ㄌapp 設定ㄌ文件"); r.Err != nil || r.Chain.Action != "搜尋" || r.Chain.Target != "app 設定" {
		t.Errorf("single mode should accept 3-seg, got %+v", r)
	}
	if r := ParseSingleLine("   "); r.Err == nil {
		t.Errorf("empty must error")
	}
	// 多行只取第一行。
	if r := ParseSingleLine("查詢ㄌa\n網路ㄌb"); r.Err != nil || r.Chain.Target != "a" {
		t.Errorf("single should take first line, got %+v", r)
	}
}

func TestParseChainLines(t *testing.T) {
	res := ParseChainLines("輸入ㄌ天氣ㄌ輸出\n輸出ㄌ請問地點ㄌ待命")
	if len(res.Steps) != 2 || len(res.Errors) != 0 {
		t.Fatalf("want 2 steps 0 errors, got %+v", res)
	}
	if res.Steps[1].Action != "輸出" || res.Steps[1].Next != "待命" {
		t.Errorf("step parse wrong: %+v", res.Steps[1])
	}
	// 夾白話：白話行進 Errors，動作行仍收。
	res2 := ParseChainLines("好的我建議：\n查詢ㄌZZZ\n這樣比較好")
	if len(res2.Steps) != 1 || res2.Steps[0].Action != "查詢" {
		t.Fatalf("prose lines should be skipped, steps=%+v", res2.Steps)
	}
	if len(res2.Errors) != 2 {
		t.Errorf("two prose lines should be errors, got %d", len(res2.Errors))
	}
}

// 關鍵安全測試：target 不能夾帶第二條可執行命令。
func TestParseChain_NoSmuggledCommand(t *testing.T) {
	// 「網路ㄌ甜點食譜 刪除ㄌ磁碟」——危險意圖：藏第二條 刪除 命令。
	res := ParseChainLines("網路ㄌ甜點食譜 刪除ㄌ磁碟")
	// 只會是「一個」step，且動作是 網路，不會冒出 刪除 這個動作。
	if len(res.Steps) != 1 {
		t.Fatalf("must be a single step, got %d", len(res.Steps))
	}
	if res.Steps[0].Action == "刪除" {
		t.Fatalf("刪除 must NOT become an executable action")
	}
	if res.Steps[0].Action != "網路" {
		t.Errorf("action should be 網路, got %q", res.Steps[0].Action)
	}
}

func TestParseChain_TargetTooLong(t *testing.T) {
	long := "網路ㄌ"
	for i := 0; i < MaxTargetRunes+10; i++ {
		long += "字"
	}
	res := ParseChainLines(long)
	if len(res.Steps) != 0 || len(res.Errors) != 1 {
		t.Errorf("over-long target should be rejected, got %+v", res)
	}
}
