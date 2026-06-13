package main

import "ui_console/orchestration/skill_flow"

// dianliao_bom_fix.go — 修正數量語意已泛型化搬入 orchestration/skill_flow（fix.go）。
// 本檔只保留既有測試沿用的轉接函式。

func hasDianliaoFixIntent(text string) bool {
	return skill_flow.HasFixIntent(text)
}

func parseDianliaoFix(text string) (target string, idx int, qty string) {
	return skill_flow.ParseFix(text)
}

func stripDianliaoFixFiller(s string) string {
	return skill_flow.StripFixFiller(s)
}

func containsIntSlice(xs []int, n int) bool {
	for _, x := range xs {
		if x == n {
			return true
		}
	}
	return false
}
