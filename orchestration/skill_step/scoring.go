// scoring.go 實作相關性評分器 Scorer。
// Scorer 是 Router 的核心子元件：它把一個 ActionTarget
// 與一份 SkillManifest 做比對，輸出 [0.0, 1.0] 的相關性分數。
//
// 分數等級定義：
//   - 1.00：動作與目標兩個維度都完全吻合（exact match）
//   - 0.85：動作或目標其中一個維度是別名或部分包含（alias / contains match）
//   - 0.70：兩個維度各有部分吻合，但都未達精確
//   - 0.00：完全沒有任何維度吻合
//
// 這個分數設計刻意簡單：不使用語意向量或 TF-IDF，
// 確保行為完全可預測、可審查，減少不可解釋的路由決策。
package skill_step

import (
	"strings"
)

// Scorer 計算 ActionTarget 對 SkillManifest 的相關性分數。
// 目前為無狀態設計（空結構），未來若需要引入 scoring 設定（如自訂權重）
// 可以在這個結構加入欄位，而不影響呼叫端介面。
type Scorer struct{}

// NewScorer 建立一個 Scorer，目前不需要任何參數。
func NewScorer() *Scorer { return &Scorer{} }

// Score 計算 at 對 manifest 的相關性分數，回傳 [0.0, 1.0] 的浮點數。
//
// 比對對象：
//   - 動作維度（Action）：比對 manifest.Routing.ActionPatterns 與 manifest.Tags.ActionTag
//   - 目標維度（Target）：比對 manifest.Routing.TargetAliases 與 manifest.Tags.DomainTag
//
// 分數矩陣：
//
//	actionScore  targetScore  最終分數
//	1.0          1.0          1.00（雙維度精確吻合）
//	≥0.85        ≥0.85        0.85（雙維度別名吻合）
//	>0           >0           0.70（雙維度各有部分吻合）
//	>0           0 (或反之)   0.70（僅單維度吻合）
//	0            0            0.00（完全不相關）
//
// 注意：單維度吻合（action 或 target 其中一個為 0）仍回傳 0.70，
// 這是刻意的設計——避免因為 manifest 的標籤不完整而完全排除一個合理的 skill。
func (s *Scorer) Score(at ActionTarget, manifest SkillManifest) float64 {
	// 分別計算動作與目標兩個維度的吻合分數
	actionScore := matchScore(at.Action,
		manifest.Routing.ActionPatterns, // 優先比對 routing 動作模式
		manifest.Tags.ActionTag,         // 再比對 tags 動作關鍵字
	)
	targetScore := matchScore(at.Target,
		manifest.Routing.TargetAliases, // 優先比對 routing 目標別名
		manifest.Tags.DomainTag,        // 再比對 tags 領域關鍵字
	)

	// 依分數矩陣決定最終分數
	switch {
	case actionScore == 1.0 && targetScore == 1.0:
		return 1.0 // 雙維度精確吻合：最高信心度
	case actionScore >= 0.85 && targetScore >= 0.85:
		return 0.85 // 雙維度別名吻合：高信心度
	case actionScore > 0 && targetScore > 0:
		return 0.70 // 雙維度各有部分吻合：中等信心度
	case actionScore > 0 || targetScore > 0:
		return 0.70 // 單維度吻合：仍給予中等信心度，避免遺漏
	default:
		return 0.0 // 完全無吻合
	}
}

// matchScore 計算 input 與 primary、secondary 兩個字串清單的最佳吻合分數。
//
// 比對規則（全部不分大小寫）：
//   - 精確吻合（input == pattern）→ 立即回傳 1.0（短路，不再繼續比較）
//   - 包含關係（input 包含 pattern，或 pattern 包含 input）→ best = 0.85
//   - 完全無吻合 → 回傳 0.0
//
// primary 與 secondary 的比對順序：先 primary，再 secondary。
// 找到精確吻合就立即短路，不需要繼續走完所有清單。
//
// 設計注意：
//   - 使用 strings.ToLower 做正規化，不使用 unicode.ToLower，
//     因為 manifest 的標籤預期是 ASCII / 中文混合，不涉及複雜 Unicode 折疊。
//   - 空字串 input 或空清單 primary/secondary 都安全處理（回傳 0.0 或跳過）。
func matchScore(input string, primary []string, secondary []string) float64 {
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "" {
		return 0.0 // 空輸入無法比對任何模式
	}

	best := 0.0
	// 依序比對 primary（ActionPatterns/TargetAliases）與 secondary（ActionTag/DomainTag）
	for _, list := range [][]string{primary, secondary} {
		for _, pattern := range list {
			p := strings.ToLower(strings.TrimSpace(pattern))
			if p == "" {
				continue // 跳過空模式，避免誤觸包含比對
			}
			if lower == p {
				return 1.0 // 精確吻合，直接短路回傳最高分
			}
			// 包含比對：input 包含 pattern，或 pattern 包含 input（互相包含）
			if strings.Contains(lower, p) || strings.Contains(p, lower) {
				if best < 0.85 {
					best = 0.85 // 只更新 best，繼續找是否有精確吻合
				}
			}
		}
	}
	return best
}
