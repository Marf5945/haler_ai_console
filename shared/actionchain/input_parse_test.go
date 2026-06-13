package actionchain

import "testing"

// 核心案例：target 是含 ㄌ 的 JSON，first/last 切法要完整保住中段。
func TestParseInputLineTargetMayContainSeparator(t *testing.T) {
	raw := `輸入ㄌ{"path":"outㄌput.xlsx","note":"aㄌb"}ㄌ待命`
	chain, err := ParseInputLine(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if chain.Target != `{"path":"outㄌput.xlsx","note":"aㄌb"}` {
		t.Fatalf("target mangled: %q", chain.Target)
	}
	if chain.Next != StandbyNext {
		t.Fatalf("next: %q", chain.Next)
	}
}

// 寬鬆切法只保留給 輸入；其他動作不得走此入口（防多命令注入回流）。
func TestParseInputLineRejectsOtherActions(t *testing.T) {
	if _, err := ParseInputLine("寫入ㄌ{...}ㄌ待命"); err == nil {
		t.Fatal("non-input action must be rejected")
	}
}

func TestParseInputLineRejectsBadStructure(t *testing.T) {
	for _, raw := range []string{"輸入", "輸入ㄌ", "輸入ㄌㄌ待命", "輸入ㄌonly-one-sep"} {
		if _, err := ParseInputLine(raw); err == nil {
			t.Fatalf("should reject: %q", raw)
		}
	}
}

func TestIsInputLine(t *testing.T) {
	if !IsInputLine("輸入ㄌ{}ㄌ待命") {
		t.Fatal("expected input line")
	}
	if IsInputLine("搜尋ㄌxㄌ待命") || IsInputLine("plain text") {
		t.Fatal("false positive")
	}
}
