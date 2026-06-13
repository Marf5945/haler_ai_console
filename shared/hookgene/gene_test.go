package hookgene

import "testing"

func TestBuildGenePadsTo16(t *testing.T) {
	g := BuildGene([]HookCode{HookInput, HookList, HookList, HookOutput})
	if len(g.Padded) != GeneLength {
		t.Fatalf("padded len = %d, want %d", len(g.Padded), GeneLength)
	}
	if got := g.String(); got != "ㄖㄅㄅㄔㄇㄇㄇㄇㄇㄇㄇㄇㄇㄇㄇㄇ" {
		t.Fatalf("gene string = %q", got)
	}
	if g.Oversized {
		t.Fatal("should not be oversized")
	}
	if g.IsBloated() {
		t.Fatal("4-hook gene should not be bloated")
	}
}

func TestOversizedDetectedBeforePadding(t *testing.T) {
	raw := make([]HookCode, 20) // > 16
	for i := range raw {
		raw[i] = HookInput
	}
	g := BuildGene(raw)
	if !g.Oversized {
		t.Fatal("20 hooks should be oversized")
	}
	if !g.IsBloated() {
		t.Fatal("oversized must be bloated")
	}
	if len(g.RawHooks) != 20 {
		t.Fatalf("raw hooks preserved = %d, want 20", len(g.RawHooks))
	}
}

func TestBloatByBRatio(t *testing.T) {
	raw := make([]HookCode, 13) // 13 個 ㄅ → 13/16 > 75%
	for i := range raw {
		raw[i] = HookList
	}
	g := BuildGene(raw)
	if g.Oversized {
		t.Fatal("13 hooks not oversized")
	}
	if g.BCount != 13 {
		t.Fatalf("bcount = %d, want 13", g.BCount)
	}
	if !g.IsBloated() {
		t.Fatal("13 ㄅ should be bloated")
	}
}

func TestShortPureBNotBloated(t *testing.T) {
	// 刻意設計：短的純 ㄅ action（3/16 ≈ 19%）不算肥大（§3.1.5.18.4）。
	g := BuildGene([]HookCode{HookList, HookList, HookList})
	if g.IsBloated() {
		t.Fatal("ㄅㄅㄅ should NOT be bloated")
	}
}
