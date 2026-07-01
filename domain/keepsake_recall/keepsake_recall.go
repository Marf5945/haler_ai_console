// Package keepsake_recall matches a user's message (in the top "打開互動"
// popover) against the active persona's saved keepsake photos (data/album),
// so the persona can reference an existing photo instead of pretending it
// doesn't exist.
//
// 設計原則：
//   - 相片記憶是永久的：album 本身就落盤、不受關程式/切人格影響，只有使用者
//     主動刪照片（DeleteAlbumPhoto）才會消失。這個套件只負責「這句話有沒有
//     講到某張已保留的照片」，不管保留/刪除的生命週期。
//   - 比對刻意做得很陽春：不用 embedding、不呼叫任何外部服務，純粹看候選
//     照片的場景/描述裡有沒有詞彙原字重新出現在使用者這句話裡。這樣行為
//     可預期、零額外成本，代價是不夠「聰明」——但比起讓人格憑空瞎掰记得
//     一張其實不相關的照片，寧可保守一點、比對不到就算了。
//   - 只在同一個人格底下比對（Photo.PersonaID），不同人格之間的紀念照不會
//     互相看到。
package keepsake_recall

import (
	"strings"
	"unicode"

	"ui_console/data/album"
)

// minKeywordRunes 是視為「有意義」的最短詞長度（以 rune 計）。單一個中文字
// 幾乎都是虛詞（的/了/我...），拿來比對會太雜訊。
const minKeywordRunes = 2

// stopwords 是常見、資訊量太低的詞，比對前先濾掉，避免隨便一句話都命中。
var stopwords = map[string]bool{
	"照片": true, "紀念照": true, "那張": true, "這張": true,
	"我們": true, "一起": true, "那個": true, "這個": true,
	"時候": true, "記得": true, "還記得": true, "看看": true,
}

// Find 在 photos 裡找出屬於 personaID、且跟 userText 有詞彙重疊的最佳一張，
// 完全比對不到（分數 0）就回傳 nil——不勉強選一張最不差的。
func Find(photos []album.Photo, personaID, userText string) *album.Photo {
	personaID = strings.TrimSpace(personaID)
	userText = strings.TrimSpace(userText)
	if personaID == "" || userText == "" || len(photos) == 0 {
		return nil
	}

	var best *album.Photo
	bestScore := 0
	for i := range photos {
		if photos[i].PersonaID != personaID {
			continue
		}
		score := overlapScore(userText, photos[i].Scene+" "+photos[i].Caption)
		if score > bestScore {
			bestScore = score
			best = &photos[i]
		}
	}
	if bestScore <= 0 {
		return nil
	}
	return best
}

// overlapScore 算 candidateText 抽出的關鍵詞裡，有幾個字（依詞長加權）原字
// 出現在 userText 裡。
func overlapScore(userText, candidateText string) int {
	score := 0
	for _, kw := range keywords(candidateText) {
		if strings.Contains(userText, kw) {
			score += len([]rune(kw))
		}
	}
	return score
}

// keywords 用「非字母數字」切段，保留長度足夠、非停用詞的片段。
// 這是單純的 rune-run 切法（沒有斷詞函式庫，對齊本專案「不新增外部依賴」
// 的慣例）——夠用來判斷「這個詞有沒有原字重新出現」，不是真正的 NLP。
func keywords(text string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) >= minKeywordRunes {
			w := string(cur)
			if !stopwords[w] {
				out = append(out, w)
			}
		}
		cur = cur[:0]
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// DescriptionNote 組出要塞進 prompt 的一小段提示：不管這個模型看不看得到
// 圖，文字描述都會給；hasImage 為真時另外提醒「圖已隨附」。
func DescriptionNote(p album.Photo, hasImage bool) string {
	desc := strings.TrimSpace(p.Caption)
	if desc == "" {
		desc = strings.TrimSpace(p.Scene)
	}
	if desc == "" {
		return ""
	}
	label := strings.TrimSpace(p.Code)
	if label == "" {
		label = p.ID
	}
	if hasImage {
		return "[記憶照片 " + label + "] 你們拍過一張照片，描述：" + desc + "（隨附圖片，可直接看）"
	}
	return "[記憶照片 " + label + "] 你們拍過一張照片，描述：" + desc + "（目前這個模型看不到圖，只能靠描述回想）"
}
