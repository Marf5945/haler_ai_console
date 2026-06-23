// cjklang.go — CJK 語系偵測 helper，供本地模型語系驗收矩陣使用
// （見 lang_matrix_integration_test.go 與前端 dev __runLangMatrix）。
// 純函式、無外部相依；判定刻意採「啟發式」而非完整語言辨識：
//   - presence 判定：字串是否含某語系的特徵 script（用於自由文字，如一般對答）。
//   - foreign-leak 判定：字串是否「混入」與目標語系矛盾的 script
//     （用於選項 label——機器化攔截「韓文角色卻吐日文片假名」這類失敗）。
package main

import "unicode"

// containsKana 回報字串是否含平假名或片假名。
func containsKana(s string) bool {
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0xFF66 && r <= 0xFF9D) { // Halfwidth Katakana
			return true
		}
	}
	return false
}

// containsHangul 回報字串是否含諺文。
func containsHangul(s string) bool {
	for _, r := range s {
		if (r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
			(r >= 0x1100 && r <= 0x11FF) || // Jamo
			(r >= 0x3130 && r <= 0x318F) { // Compatibility Jamo
			return true
		}
	}
	return false
}

// containsHan 回報字串是否含漢字。
func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// containsLongLatinWord 回報字串是否含長拉丁字母詞。
// 韓文輸出允許 CSV/API/UI 這種短技術詞，但不允許用英文句子混過語系驗收。
func containsLongLatinWord(s string) bool {
	run := 0
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			run++
			if run > 3 {
				return true
			}
			continue
		}
		run = 0
	}
	return false
}

// matchesExpectedLanguage 以「presence」判定 text 是否「像」指定語系，
// 適用於自由文字（一般對答、程式/流程用途說明）。
//
//	JA: 必須含假名。
//	KO: 必須含諺文，且不得含假名；也不得用長英文詞混過。
//	ZH: 必須含漢字，且不得含假名或諺文。
//	EN/PT: 不得含任何 CJK script。
//	其他/未知: 一律視為通過（不阻擋）。
func matchesExpectedLanguage(lang, text string) bool {
	hasKana := containsKana(text)
	hasHangul := containsHangul(text)
	hasHan := containsHan(text)
	switch lang {
	case responseLanguageJA:
		return hasKana
	case responseLanguageKO:
		return hasHangul && !hasKana && !containsLongLatinWord(text)
	case responseLanguageZH:
		return hasHan && !hasKana && !hasHangul
	case responseLanguageEN, responseLanguagePT:
		return !hasKana && !hasHangul && !hasHan
	default:
		return true
	}
}

// languageScriptLeak 回報 text 是否「混入」與 lang 矛盾的 script。
// 適用於選項 label 這種 script 易混淆（日期 6月21日 / 6월 21일）但
// 不該跨語系洩漏的場景。回 true = 有洩漏 = 不通過。
//
//	JA: 洩漏 = 含諺文。
//	KO: 洩漏 = 含假名或漢字（攔截「韓文角色吐日文片假名」與日期混成 23日）。
//	ZH: 洩漏 = 含假名或諺文。
//	EN/PT: 洩漏 = 含任何 CJK script。
//	其他/未知: 一律不視為洩漏。
func languageScriptLeak(lang, text string) bool {
	hasKana := containsKana(text)
	hasHangul := containsHangul(text)
	hasHan := containsHan(text)
	switch lang {
	case responseLanguageJA:
		return hasHangul
	case responseLanguageKO:
		return hasKana || hasHan
	case responseLanguageZH:
		return hasKana || hasHangul
	case responseLanguageEN, responseLanguagePT:
		return hasKana || hasHangul || hasHan
	default:
		return false
	}
}
