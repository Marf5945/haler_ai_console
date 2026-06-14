package main

// routing_candidates.go
//
// 收尾版 Step 4：skill 能力初篩（重用 orchestration/skill_step 的 Router/Scorer，
// 第一版不引入 embedding）。把初篩出的候選 skill 注入 judge prompt，讓 judge
// 不再瞎猜 skill 名稱，而是從受控候選裡挑 SkillID。
//
// 兩個出口共用 RankByQuery（確定性、可 debug）：
//   - formatAvailableSkillsContext：注入 judge prompt 的人類可讀候選清單
//   - routingCandidateSkillIDs：給 Step 5 SkillID 白名單驗證用的 ID 集合
//
// recall 安全：初篩無命中時回空（不注入），judge 仍可走 網路/搜尋/聊天/提問，
// 初篩只是縮小 catalog，不是路由真相來源。

import (
	"fmt"
	"strings"

	"ui_console/data/memory"
)

// routingCandidateLimit 限制注入 judge 的候選數，控制 prompt 大小。
const routingCandidateLimit = 10

// formatAvailableSkillsContext 注入「本輪候選 skill」區塊（SkillID + 簡短觸發語）。
func (a *App) formatAvailableSkillsContext(terms []string) string {
	if a == nil || a.skillRouter == nil || len(terms) == 0 {
		return ""
	}
	cands := a.skillRouter.RankByQuery(terms)
	if len(cands) == 0 {
		return ""
	}
	ids := make([]string, 0, routingCandidateLimit)
	for _, c := range cands {
		if len(ids) >= routingCandidateLimit {
			break
		}
		ids = append(ids, c.SkillID)
	}
	briefs := a.skillRouter.Briefs(ids)
	if len(briefs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n[available_skills]（要走流程時，輸出 流程ㄌ<下面其中一個 SkillID>ㄌ輸出；沒有合適的就別硬選）\n")
	for _, br := range briefs {
		if br.SkillID == "builtin.scheduler" {
			fmt.Fprintf(&b, "- %s（%s：%s；可輸出 builtin.scheduler;title=<短標題>;summary=<一句話摘要>）\n", br.SkillID, br.DisplayName, br.Trigger)
			continue
		}
		if trig := strings.TrimSpace(br.Trigger); trig != "" {
			fmt.Fprintf(&b, "- %s（%s：%s）\n", br.SkillID, br.DisplayName, trig)
		} else {
			fmt.Fprintf(&b, "- %s（%s）\n", br.SkillID, br.DisplayName)
		}
	}
	b.WriteString("[/available_skills]\n")
	return b.String()
}

// routingCandidateSkillIDs 回傳本輪初篩候選的 SkillID 集合（Step 5 白名單用）。
func (a *App) routingCandidateSkillIDs(terms []string) map[string]struct{} {
	out := map[string]struct{}{}
	if a == nil || a.skillRouter == nil {
		return out
	}
	for i, c := range a.skillRouter.RankByQuery(terms) {
		if i >= routingCandidateLimit {
			break
		}
		out[c.SkillID] = struct{}{}
	}
	return out
}

// routingMemoryLimit 限制注入 judge 的記憶摘要條數（低敏、聚焦）。
const routingMemoryLimit = 5

// formatRoutingMemoryContext 注入 3-5 條低敏記憶摘要/偏好（Step 6）。
// 只在能力初篩有候選（matched/ambiguous）時注入；none 不注入（省成本）。
// 雙閘：summaries 寫入時已遮蔽，注入前再 RedactBeforeWrite 一次。
func (a *App) formatRoutingMemoryContext(terms []string) string {
	if a == nil || a.memoryPipeline == nil {
		return ""
	}
	if len(a.routingCandidateSkillIDs(terms)) == 0 {
		return ""
	}
	sums := a.memoryPipeline.ReadRecentSummaries(routingMemoryLimit)
	if len(sums) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n[routing_memory]（僅供判斷參考，勿原樣寫入搜尋 query）\n")
	for _, s := range sums {
		clean, _ := memory.RedactBeforeWrite(s) // 注入前再遮蔽一次（雙閘）
		clean = strings.TrimSpace(strings.ReplaceAll(clean, "\n", " "))
		if clean == "" {
			continue
		}
		if r := []rune(clean); len(r) > 160 {
			clean = strings.TrimSpace(string(r[:160]))
		}
		fmt.Fprintf(&b, "- %s\n", clean)
	}
	b.WriteString("[/routing_memory]\n")
	return b.String()
}
