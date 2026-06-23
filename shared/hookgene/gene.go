package hookgene

import "strings"

// GeneLength 是固定基因長度（§3.1.5.18.3）：每個 skill invocation 形成 16 格。
const GeneLength = 16

// BloatBCount：ㄅ 比例 > 75%（16 格中 ㄅ >= 13）視為處理型過多（§3.1.5.18.4）。
// 75% * 16 = 12，「> 75%」即 > 12 → >= 13。MVP 寫死，日後可改 per-project config。
const BloatBCount = 13

// Gene 是一次 invocation 的行動基因結果。
type Gene struct {
	RawHooks  []HookCode // 補位前的原始 hook 序列（oversized 時完整保留）
	Padded    []HookCode // 補滿/截斷成 16 格（顯示與比例計算用）
	BCount    int        // Padded 中的 ㄅ 數量
	Oversized bool       // 原始 hook 數 > 16
}

// BuildGene 由原始 hook 序列產生 Gene。判定順序（§3.1.5.18.4）：
//  1. 先看原始 hook 數（> 16 標 Oversized，原始序列保留在 RawHooks）
//  2. 不足 16 以 ㄇ 補滿、超過則截斷成 16 格（僅供顯示/比例）
//  3. ㄅ 比例以補位後的 16 為分母
func BuildGene(raw []HookCode) Gene {
	g := Gene{RawHooks: append([]HookCode(nil), raw...)}
	g.Oversized = len(raw) > GeneLength

	padded := make([]HookCode, 0, GeneLength)
	for i := 0; i < GeneLength; i++ {
		if i < len(raw) {
			padded = append(padded, raw[i])
		} else {
			padded = append(padded, HookStandby) // ㄇ 補位
		}
	}
	g.Padded = padded
	for _, h := range padded {
		if h == HookList {
			g.BCount++
		}
	}
	return g
}

// String 回傳 16 格 gene 字串（僅 debug/review/learning 顯示用）。
func (g Gene) String() string {
	var b strings.Builder
	for _, h := range g.Padded {
		b.WriteRune(rune(h))
	}
	return b.String()
}

// IsBloated：原始 hook > 16 或 ㄅ >= 13（任一成立）即為肥大樣本（§3.1.5.18.4）。
// 註：短的純 ㄅ action（如 ㄅㄅㄅ → 3/16）刻意不算肥大，MVP 只抓真的填滿的肥大。
func (g Gene) IsBloated() bool {
	return g.Oversized || g.BCount >= BloatBCount
}
