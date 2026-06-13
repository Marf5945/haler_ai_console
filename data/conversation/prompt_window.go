// conversation/prompt_window.go — 對話歷史滑動視窗(token 優化熱點 1)。
//
// 只在「組 prompt 時」套用視窗,SentenceStore 一個字都不刪,
// 因此摘要完整性檢查(CheckSummaryIntegrity)與既有摘要流程完全不受影響。
// 規則:
//  1. 最近 KeepRecentTurns 輪(以 user 句為界)無條件完整保留
//  2. 更早的句子由新到舊累計,總預算 RawBudgetRunes,超出者移出視窗
//  3. 重要句豁免(不受預算限制):含「[系統提供:」標記、task-intent 標記、tool-action 句
//  4. 視窗內較早的超長句(> MaxSentenceRunes)保留頭尾,中間以「……(中略)……」標示
//  5. 有句子被省略時,於 H 區開頭插入一行佔位,讓模型知道歷史被裁過
//
// 出問題時把 PromptWindowEnabled 設為 false 即可退回全量模式。
package conversation

import (
	"fmt"
	"strings"
)

// PromptWindowEnabled 全域開關;false 時 ApplyPromptWindow 原樣返回。
var PromptWindowEnabled = true

// PromptWindowConfig 滑動視窗參數。
type PromptWindowConfig struct {
	KeepRecentTurns  int // 最近 N 輪(user 句數)無條件保留
	RawBudgetRunes   int // 較早句子的總 rune 預算
	MaxSentenceRunes int // 單句上限,超過做頭尾保留
	HeadRunes        int // 超長句保留頭部 rune 數
	TailRunes        int // 超長句保留尾部 rune 數
}

// DefaultPromptWindowConfig 預設參數。
func DefaultPromptWindowConfig() PromptWindowConfig {
	return PromptWindowConfig{
		KeepRecentTurns:  4,
		RawBudgetRunes:   3000,
		MaxSentenceRunes: 600,
		HeadRunes:        400,
		TailRunes:        100,
	}
}

// PromptWindowResult 視窗套用結果。
type PromptWindowResult struct {
	Sentences []Sentence // 視窗內句子(時序保持)
	Elided    int        // 被省略的句子數
}

// isWindowProtected 回報句子是否不受預算限制(規則 3)。
func isWindowProtected(sent Sentence) bool {
	if sent.Role == "tool-action" {
		return true
	}
	if strings.Contains(sent.Content, "[系統提供:") {
		return true
	}
	if strings.Contains(sent.ID, "task-intent") || strings.Contains(sent.Content, "task-intent") {
		return true
	}
	return false
}

// truncateSentenceContent 對超長句做頭尾保留(規則 4)。
func truncateSentenceContent(content string, cfg PromptWindowConfig) string {
	runes := []rune(content)
	if cfg.MaxSentenceRunes <= 0 || len(runes) <= cfg.MaxSentenceRunes {
		return content
	}
	head := cfg.HeadRunes
	tail := cfg.TailRunes
	if head <= 0 || head+tail >= len(runes) {
		return content
	}
	return string(runes[:head]) + "……(中略)……" + string(runes[len(runes)-tail:])
}

// ApplyPromptWindow 對未摘要句子套用滑動視窗。
// 輸入須為時序排列(舊→新),輸出維持同序;不修改輸入切片。
func ApplyPromptWindow(raw []Sentence, cfg PromptWindowConfig) PromptWindowResult {
	if !PromptWindowEnabled || len(raw) == 0 {
		return PromptWindowResult{Sentences: raw}
	}

	// 規則 1:找出「最近 N 輪」的起點(從尾端往回數第 N 個 user 句)。
	recentStart := 0
	if cfg.KeepRecentTurns > 0 {
		userSeen := 0
		recentStart = len(raw) // 若 user 句不足 N 輪,全部視為最近
		for i := len(raw) - 1; i >= 0; i-- {
			if raw[i].Role == "user" {
				userSeen++
				if userSeen >= cfg.KeepRecentTurns {
					recentStart = i
					break
				}
			}
		}
		if userSeen < cfg.KeepRecentTurns {
			recentStart = 0
		}
	}

	// 最近輪全保留(不截斷)。
	recent := raw[recentStart:]
	older := raw[:recentStart]

	// 規則 2+3+4:較早句子由新到舊累計預算;豁免句永遠保留。
	kept := make([]bool, len(older))
	budget := cfg.RawBudgetRunes
	for i := len(older) - 1; i >= 0; i-- {
		if isWindowProtected(older[i]) {
			kept[i] = true
			continue
		}
		cost := len([]rune(truncateSentenceContent(older[i].Content, cfg)))
		if budget-cost >= 0 {
			budget -= cost
			kept[i] = true
		}
	}

	var out []Sentence
	elided := 0
	for i, sent := range older {
		if !kept[i] {
			elided++
			continue
		}
		sent.Content = truncateSentenceContent(sent.Content, cfg)
		out = append(out, sent)
	}

	// 規則 5:有省略時插入佔位句,置於 H 區最前。
	if elided > 0 {
		placeholder := Sentence{
			ID:      "[window]",
			Role:    "context-note",
			Content: fmt.Sprintf("(視窗外省略較早 %d 句;更早脈絡見 S 區摘要,如需細節請查對話紀錄)", elided),
		}
		out = append([]Sentence{placeholder}, out...)
	}

	out = append(out, recent...)
	return PromptWindowResult{Sentences: out, Elided: elided}
}
