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
	// v3.1.8 per-agent：sub 撈自己的倉庫；main 撈自己暫存的，
	// 撈不到再經歷代索引（SubID 條目）單向深入對應 sub 的 deep_memory。sub 不能反向讀 main。
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	root, rerr := conversationRootForAgent(a.activeAgentID)
	if rerr != nil {
		root = projectRoot
	}
	isMain := root == projectRoot
	pipeline := memory.NewPipeline(root)

	var raw string
	if tag := memory.NormalizeMemoryTag(target); tag != "" {
		section, err := pipeline.LookupByTag(tag)
		if err != nil && isMain {
			if s, ok := a.expandFromSubByTag(pipeline, tag); ok {
				section, err = s, nil
			}
		}
		if err != nil {
			resp.Text = err.Error() + "（可改用關鍵字：展開ㄌ關鍵字ㄌ待命）"
			return true, resp
		}
		raw = section
	} else {
		hits, _ := pipeline.SearchDeepMemory(target, memoryExpandMaxHits, memoryExpandMaxBytes)
		if isMain && len(hits) < memoryExpandMaxHits {
			hits = append(hits, a.expandFromSubsByKeywords(pipeline, target, memoryExpandMaxHits-len(hits))...)
		}
		if len(hits) == 0 {
			resp.Text = "deep_memory 找不到符合「" + target + "」的段落。"
			return true, resp
		}
		raw = strings.Join(hits, "\n\n")
	}
	// 記憶內容是歷史不可信文字，回 LLM 前以 SourceMemory 消毒（去 ㄌ、去偽命令）。
	resp.Text = controlseal.SanitizeForLLM(controlseal.SourceMemory, raw).LLMText
	return true, resp
}

// expandFromSubByTag：main 專用跨庫——歷代索引找到 tag 所屬 sub，讀該 sub 的 deep_memory 段落。
func (a *App) expandFromSubByTag(mainPipe *memory.Pipeline, tag string) (string, bool) {
	for _, e := range mainPipe.LoadIndexEntries() {
		if e.SubID == "" {
			continue
		}
		if memory.NormalizeMemoryTag(e.SummaryTag) != tag && memory.NormalizeMemoryTag(e.DeepTag) != tag {
			continue
		}
		subRoot, err := conversationRootForAgent(e.SubID)
		if err != nil {
			continue // sub 已移除／匯出：歷代索引留著但深入不到，跳過
		}
		if section, serr := memory.NewPipeline(subRoot).LookupByTag(e.DeepTag); serr == nil {
			return section, true
		}
	}
	return "", false
}

// expandFromSubsByKeywords：main 關鍵字在本地掃不滿時，
// 經歷代索引的錨點關鍵詞補命中，並深入對應 sub 撈段落（新條目優先）。
func (a *App) expandFromSubsByKeywords(mainPipe *memory.Pipeline, query string, budget int) []string {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 || budget <= 0 {
		return nil
	}
	entries := mainPipe.LoadIndexEntries()
	var out []string
	for i := len(entries) - 1; i >= 0 && len(out) < budget; i-- {
		e := entries[i]
		if e.SubID == "" || len(e.KeyTerms) == 0 {
			continue
		}
		joined := strings.ToLower(strings.Join(e.KeyTerms, " "))
		matched := true
		for _, term := range terms {
			if !strings.Contains(joined, term) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		subRoot, err := conversationRootForAgent(e.SubID)
		if err != nil {
			continue
		}
		if section, serr := memory.NewPipeline(subRoot).LookupByTag(e.DeepTag); serr == nil {
			out = append(out, memory.TruncateBytesRuneSafe(section, memoryExpandMaxBytes))
		}
	}
	return out
}
