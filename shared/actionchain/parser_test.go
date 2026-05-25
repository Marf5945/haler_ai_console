package actionchain

import (
	"errors"
	"testing"
)

func TestParseValidChatChain(t *testing.T) {
	chain, err := Parse("聊天ㄌ今天天氣很棒ㄌ輸出")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if chain.Action != "聊天" || chain.Target != "今天天氣很棒" || chain.Next != "輸出" {
		t.Fatalf("unexpected chain: %#v", chain)
	}
}

func TestParseValidExecutableChain(t *testing.T) {
	chain, err := Parse("查詢ㄌ台北天氣ㄌ輸出")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if chain.Action != "查詢" || chain.Target != "台北天氣" || chain.Next != "輸出" {
		t.Fatalf("unexpected chain: %#v", chain)
	}
}

func TestParseRejectsTooManySegments(t *testing.T) {
	_, err := Parse("查詢ㄌ天氣ㄌ複製ㄌ貼上")
	if !errors.Is(err, ErrStructure) {
		t.Fatalf("err = %v, want ErrStructure", err)
	}
}

func TestParseRejectsMissingSeparator(t *testing.T) {
	_, err := Parse("查詢")
	if !errors.Is(err, ErrStructure) {
		t.Fatalf("err = %v, want ErrStructure", err)
	}
}

func TestValidateActionTagUnknown(t *testing.T) {
	chain, err := Parse("剪下ㄌ這一行資料ㄌ輸出")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	result := ValidateActionTag(chain.Action, NewStaticRegistry("查詢"))
	if result.Code != ValidationUnknown || !errors.Is(result.Err, ErrUnknownActionTag) {
		t.Fatalf("validation = %#v", result)
	}
}

func TestResolveBuiltInChatDisplaysOnlyTarget(t *testing.T) {
	chain, err := Parse("聊天ㄌ你好ㄌ待命")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	decision := ResolveBuiltIn(chain)
	if !decision.Handled || decision.DisplayText != "你好" || !decision.Terminal {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestResolveBuiltInNormalizesOldWaitingDirective(t *testing.T) {
	chain, err := Parse("聊天ㄌ你好ㄌ等待指令")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if chain.Next != StandbyNext {
		t.Fatalf("Next = %q, want %q", chain.Next, StandbyNext)
	}
}

func TestNormalizeNextIgnoresTailAfterStandby(t *testing.T) {
	chain, err := Parse("搜尋ㄌ星座學術論文ㄌ待命\n\n需要授權網路搜尋工具")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if chain.Next != StandbyNext {
		t.Fatalf("Next = %q, want %q", chain.Next, StandbyNext)
	}
	decision := ResolveBuiltIn(chain)
	if !decision.Terminal {
		t.Fatalf("standby with tail should remain terminal: %#v", decision)
	}
}

func TestResolveBuiltInQuestionAndClipboardTags(t *testing.T) {
	for _, input := range []string{
		"提問ㄌ你想查哪裡？ㄌ輸出",
		"選項ㄌㄤ台北ㄤ台中ㄌ等待",
		"複製ㄌ今天開會ㄌ待命",
		"貼上ㄌ輸入框ㄌ待命",
	} {
		chain, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", input, err)
		}
		decision := ResolveBuiltIn(chain)
		if !decision.Handled {
			t.Fatalf("expected %q to be built-in", chain.Action)
		}
	}
}

func TestParseStripsLLMOnlyOptionPrefix(t *testing.T) {
	chain, err := Parse("ㄌㄤㄤ選項ㄌㄤ台北ㄤ台中ㄌ等待")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if chain.Action != "選項" || chain.Target != "ㄤ台北ㄤ台中" || chain.Next != StandbyNext {
		t.Fatalf("unexpected chain: %#v", chain)
	}
}

func TestResolveBuiltInLocalSearchTags(t *testing.T) {
	for _, input := range []string{
		"本機搜尋ㄌAPI keyㄌ執行",
		"搜尋ㄌ記憶 API keyㄌ執行",
		"searchㄌnotesㄌ執行",
	} {
		chain, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", input, err)
		}
		decision := ResolveBuiltIn(chain)
		if !decision.Handled || decision.DisplayText != chain.Target {
			t.Fatalf("expected local search builtin decision: %#v", decision)
		}
	}
}

func TestStaticRegistryKeepsReservedTagsControllerOwned(t *testing.T) {
	registry := NewStaticRegistry("聊天", "提問", "選項", "複製", "貼上", "查詢")
	if !registry.HasActionTag("聊天") || !registry.HasActionTag("提問") || !registry.HasActionTag("選項") || !registry.HasActionTag("查詢") {
		t.Fatalf("registry missing expected tags: %#v", registry)
	}
	for _, tag := range []string{"聊天", "提問", "選項", "複製", "貼上"} {
		if !IsReservedTag(tag) {
			t.Fatalf("%s should remain reserved", tag)
		}
	}
}
