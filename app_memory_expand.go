// app_memory_expand.go — 展開 動作：把 deep_memory 細節撈回 LLM context（v3.1.7）。
// 模型在 context 看到 [S-NNN] 摘要標記後，可用 展開ㄌS-NNNㄌ待命 取回被壓掉的原文；
// 也支援 展開ㄌ關鍵字ㄌ待命 在 deep_memory.md 內搜尋（多詞 AND、新段優先）。
// 與 搜尋/網路 相同模式：LLM 只提議，App 持有實際讀取；結果以 SourceMemory 消毒後進 context。
package main

import (
	"strings"

	"ui_console/data/memory"
	"ui_console/data/storage"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/controlseal"
)

const (
	memoryExpandMaxHits     = 3        // 關鍵字搜尋最多回幾段
	memoryExpandMaxBytes    = 2 * 1024 // 每段截斷上限
	memoryExpandActionLabel = "展開"
)

// maybeExpandMemory 是三個動作分派點共用的 展開 處理；非 展開 動作回 handled=false。
func (a *App) maybeExpandMemory(action, target, traceID string) (bool, skill_step.CLIResponse) {
	if strings.TrimSpace(action) != memoryExpandActionLabel {
		return false, skill_step.CLIResponse{}
	}
	target = strings.TrimSpace(target)
	resp := skill_step.CLIResponse{Action: memoryExpandActionLabel, Target: target, Next: "待命"}
	if target == "" {
		resp.Text = "展開需要目標：給記憶標籤（例 S-12345 / D-12345）或關鍵字。"
		return true, resp
	}
	pipeline := memory.NewPipeline(storage.ProjectRoot(appDataRoot(), "default"))

	var raw string
	if tag := memory.NormalizeMemoryTag(target); tag != "" {
		section, err := pipeline.LookupByTag(tag)
		if err != nil {
			resp.Text = err.Error() + "（可改用關鍵字：展開ㄌ關鍵字ㄌ待命）"
			return true, resp
		}
		raw = section
	} else {
		hits, err := pipeline.SearchDeepMemory(target, memoryExpandMaxHits, memoryExpandMaxBytes)
		if err != nil || len(hits) == 0 {
			resp.Text = "deep_memory 找不到符合「" + target + "」的段落。"
			return true, resp
		}
		raw = strings.Join(hits, "\n\n")
	}
	// 記憶內容是歷史不可信文字，回 LLM 前以 SourceMemory 消毒（去 ㄌ、去偽命令）。
	resp.Text = controlseal.SanitizeForLLM(controlseal.SourceMemory, raw).LLMText
	return true, resp
}
