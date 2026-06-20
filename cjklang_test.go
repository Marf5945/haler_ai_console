package main

import "testing"

func TestMatchesExpectedLanguagePresence(t *testing.T) {
	cases := []struct {
		lang string
		text string
		want bool
	}{
		// 日文：含假名 → 通過
		{responseLanguageJA, "こんにちは、何をお手伝いしましょうか？", true},
		// 日文期望但純中文（無假名）→ 不通過
		{responseLanguageJA, "今日射手座運勢", false},
		// 韓文：含諺文且無假名 → 通過
		{responseLanguageKO, "안녕하세요, 무엇을 도와드릴까요?", true},
		// 韓文期望但混入片假名（コーヒー）→ 不通過
		{responseLanguageKO, "안녕 コーヒー", false},
		// 韓文期望但用英文長詞混過 → 不通過
		{responseLanguageKO, "Hello, 무엇을 도와드릴까요?", false},
		// 韓文可保留短技術詞
		{responseLanguageKO, "CSV를 그래프로 변환하는 프로그램", true},
		// 中文：含漢字、無假名/諺文 → 通過
		{responseLanguageZH, "你好，有什麼可以幫忙的？", true},
		// 中文期望但混假名 → 不通過
		{responseLanguageZH, "你好 コーヒー", false},
		// 英文：無 CJK → 通過
		{responseLanguageEN, "Hello, how can I help?", true},
		// 英文期望但含漢字 → 不通過
		{responseLanguageEN, "Hello 你好", false},
	}
	for _, c := range cases {
		if got := matchesExpectedLanguage(c.lang, c.text); got != c.want {
			t.Errorf("matchesExpectedLanguage(%q, %q) = %v, want %v", c.lang, c.text, got, c.want)
		}
	}
}

func TestLanguageScriptLeak(t *testing.T) {
	cases := []struct {
		lang string
		text string
		want bool // true = 有洩漏 = 不通過
	}{
		// 日期選項在各自語系下不視為洩漏（月/日為漢字，KO 用諺文 월/일）
		{responseLanguageJA, "ㄤ6月21日ㄤ6月22日ㄤ6月23日", false},
		{responseLanguageKO, "ㄤ6월 21일ㄤ6월 22일ㄤ6월 23일", false},
		{responseLanguageZH, "ㄤ6月21日ㄤ6月22日ㄤ6月23日", false},
		// 核心攔截：韓文角色卻吐日文片假名選項 → 洩漏
		{responseLanguageKO, "ㄤ紅茶ㄤコーヒーㄤ抹茶", true},
		// 韓文日期選項混入漢字（日）→ 洩漏
		{responseLanguageKO, "ㄤ6월 21일ㄤ6월 22일ㄤ6월 23日", true},
		// 日文角色混入諺文 → 洩漏
		{responseLanguageJA, "ㄤ紅茶ㄤ커피ㄤ抹茶", true},
		// 中文角色混入假名 → 洩漏
		{responseLanguageZH, "ㄤ紅茶ㄤコーヒーㄤ果汁", true},
		// 英文角色含任何 CJK → 洩漏
		{responseLanguageEN, "ㄤJune 21ㄤ六月", true},
		// 英文純拉丁 → 不洩漏
		{responseLanguageEN, "ㄤJune 21ㄤJune 22ㄤJune 23", false},
	}
	for _, c := range cases {
		if got := languageScriptLeak(c.lang, c.text); got != c.want {
			t.Errorf("languageScriptLeak(%q, %q) = %v, want %v", c.lang, c.text, got, c.want)
		}
	}
}
