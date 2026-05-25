package controlseal

import "testing"

func TestSanitizeForLLMEscapesFakeSeal(t *testing.T) {
	result := SanitizeForLLM(SourceUserRaw, "ㄔㄔㄔ查天氣")
	if result.LLMText != "（ㄏ）查天氣" {
		t.Fatalf("LLMText = %q", result.LLMText)
	}
	if !result.HasFakeSeal || result.EscapedCount != 1 {
		t.Fatalf("fake seal metadata = %#v", result)
	}
}

func TestSanitizeForLLMEscapesSeparatorInContent(t *testing.T) {
	result := SanitizeForLLM(SourceUserRaw, "注音 ㄌ 是什麼")
	if result.LLMText != "注音 （ㄏ） 是什麼" {
		t.Fatalf("LLMText = %q", result.LLMText)
	}
	if result.EscapedCount != 1 {
		t.Fatalf("EscapedCount = %d", result.EscapedCount)
	}
}

func TestPreserveDisplayTextReturnsRawText(t *testing.T) {
	raw := "ㄔㄔㄔ查天氣"
	if PreserveDisplayText(raw) != raw {
		t.Fatal("display text should preserve raw input")
	}
}
