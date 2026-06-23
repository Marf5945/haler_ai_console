// scheduler_failover.go — 排程執行的 CLI 故障切換順序（3.1.10 Phase C）。
//
// 規格（使用者確認）：主 CLI → 第二順位 CLI → 本地模型(loc) → 雲端 API 墊底。
// 本檔只負責「排出嘗試順序」這段純邏輯，方便單元測試；實際逐一送訊息、
// 偵測 quota/逾時並往下切，由執行協調器（Phase E）呼叫本函式後執行。
package main

import "sort"

// failoverKind 區分可作為排程執行後端的種類，決定備援優先序。
type failoverKind int

const (
	failoverCLI   failoverKind = iota // 外部 CLI adapter（如 Gemini CLI / Claude CLI）
	failoverLocal                     // 本地模型（Ollama / LM Studio）
	failoverAPI                       // 雲端 API（可能產生費用 → 永遠墊底）
)

// failoverCandidate 描述一個可作為排程執行後端的 adapter。
type failoverCandidate struct {
	AdapterID string
	Kind      failoverKind
	Order     int  // 使用者在模型選單的排序（越小越前）
	Available bool // 目前是否可用（已偵測、未被標記離線）
}

// orderSchedulerFailover 依規格排出排程執行的備援嘗試順序：
//  1. 可用的外部 CLI，依使用者排序（Order 小者先）
//  2. 可用的本地模型（loc 優先於雲端）
//  3. 雲端 API 永遠墊底（最後選擇，因可能產生費用，亦會被風險閘門攔成需確認）
//
// 不可用（Available=false）的候選一律排除；同類型內以 Order 穩定排序，
// Order 相同則維持輸入順序。
func orderSchedulerFailover(cands []failoverCandidate) []failoverCandidate {
	rank := func(k failoverKind) int {
		switch k {
		case failoverCLI:
			return 0
		case failoverLocal:
			return 1
		default: // failoverAPI
			return 2
		}
	}
	out := make([]failoverCandidate, 0, len(cands))
	for _, c := range cands {
		if c.Available {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := rank(out[i].Kind), rank(out[j].Kind)
		if ri != rj {
			return ri < rj
		}
		return out[i].Order < out[j].Order
	})
	return out
}
