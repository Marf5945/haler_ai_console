package localsearch

import "testing"

func TestAuxTermsFromText(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"產出電料bom 要怎麼用", []string{"要怎麼用", "怎麼用"}},
		{"產出電料bom 怎麼做", []string{"怎麼做"}},
		{"產出電料bom", nil},
		// 回歸：內容名詞不可被當問法詞剝掉。
		{"某某教學手冊製作", nil},
		{"電料bom的用途說明", nil},
	}
	for _, c := range cases {
		got := AuxTermsFromText(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("AuxTermsFromText(%q)=%v, want %v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("AuxTermsFromText(%q)=%v, want %v", c.in, got, c.want)
			}
		}
	}
}

// 護欄核心：輔助詞不能單獨讓不相關的 item 成立。
func TestAuxTermsNeverMatchAlone(t *testing.T) {
	item := Item{Source: "skill", Title: "產出電料Bom", Path: "skill:abc",
		Content: "技能 產出電料Bom：根據電料檔產出BOM\n用法 說明"}
	// 主要關鍵詞完全不沾邊，只有輔助詞「用法」命中 -> 必須 not matched。
	if _, ok := matchItem("天氣預報", []string{"用法"}, item); ok {
		t.Fatal("aux term alone should not make an unrelated item match")
	}
}

// 護欄：主要關鍵詞命中時，輔助詞只加分、不改變是否命中，且分數較高排前面。
func TestAuxTermsBoostButLower(t *testing.T) {
	item := Item{Source: "skill", Title: "產出電料Bom", Path: "skill:abc",
		Content: "技能 產出電料Bom：根據電料檔產出BOM\n用法 說明"}
	base, ok := matchItem("產出電料bom", nil, item)
	if !ok {
		t.Fatal("primary term should match")
	}
	boosted, ok := matchItem("產出電料bom", []string{"用法"}, item)
	if !ok {
		t.Fatal("primary term should still match with aux")
	}
	if boosted.Score <= base.Score {
		t.Fatalf("aux should raise score: base=%d boosted=%d", base.Score, boosted.Score)
	}
	if boosted.Score-base.Score > 12 {
		t.Fatalf("aux bonus should be capped at 12, got +%d", boosted.Score-base.Score)
	}
}

// 回歸：judge 附加描述詞（如「使用方式」）不可把標題命中的 skill 擋掉。
func TestTitleHitSurvivesNoiseTerm(t *testing.T) {
	item := Item{Source: "skill", Title: "產出電料Bom", Path: "skill:go-program-產出電料bom",
		Content: "技能 產出電料Bom：根據電料檔產出BOM"}
	// 「使用方式」不在內容/標題，舊邏輯會因 need>=2 卡死回 0。
	if _, ok := matchItem("產出電料Bom 使用方式", nil, item); !ok {
		t.Fatal("title hit should survive an unmatched noise term")
	}
	// 純雜訊（無任何標題詞命中）仍不該命中。
	if _, ok := matchItem("天氣 使用方式", nil, item); ok {
		t.Fatal("pure noise must not match via title fallback")
	}
}
