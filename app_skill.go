// app_skill.go - split out of app.go (same package, codemod).
package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/tools"
)

func (e schedulerSkillExecutor) ExecuteSkill(ctx context.Context, actionTarget string, sessionID string) error {
	if e.app == nil {
		return fmt.Errorf("scheduler skill executor: app is nil")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	adapterID := e.app.defaultSkillExecutionAdapterID()
	if adapterID == "" {
		return fmt.Errorf("scheduler skill executor: no adapter available")
	}
	decision, err := e.app.ExecuteSkillMessage(adapterID, sessionID, actionTarget, fmt.Sprintf("scheduler-skill-%d", time.Now().UnixNano()))
	if err != nil {
		return err
	}
	if !decision.Executed {
		return fmt.Errorf("scheduler skill executor: skill %q was not executed (decision=%s)", actionTarget, decision.Decision)
	}
	return nil
}

// skillManifestToTool 把已歸檔 skill manifest 轉成工具列上的一個 Tool。
func skillManifestToTool(m skill_step.SkillManifest) tools.Tool {
	detail := strings.TrimSpace(m.Description)
	if detail == "" {
		detail = "已安裝 skill"
	}
	return tools.Tool{
		ID:         "skill:" + m.SkillID,
		Icon:       "\u2726", // ✦
		Title:      firstNonEmpty(m.DisplayName, m.SkillID),
		Detail:     detail,
		Kind:       "skill",
		Target:     m.SkillID,
		Enabled:    true,
		Available:  true,
		ActionTags: append([]string(nil), m.Tags.ActionTag...),
	}
}

// registerSkillTool 把單一 skill 登記到工具列；disabled 或不可見者不登記。
// toolsService.AddTool 內含去重，重複呼叫安全。
func (a *App) registerSkillTool(m *skill_step.SkillManifest) {
	if a == nil || a.toolsService == nil || m == nil || m.SkillID == "" {
		return
	}
	skill_step.EnsureLifecycle(m)
	if lc := m.Lifecycle; lc != nil {
		if lc.Status == skill_step.LifecycleDisabled || !lc.VisibleInToolbar {
			return
		}
	}
	a.toolsService.AddTool(skillManifestToTool(*m))
}

// syncArchivedSkillsToToolbar 啟動時把 data/skills 下所有可見 skill 同步進工具列，
// 讓「曾經安裝過」的 skill 重開 App 後仍出現在 工具 > 自動流程。
func (a *App) syncArchivedSkillsToToolbar() {
	if a == nil || a.skillArchive == nil || a.toolsService == nil {
		return
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return
	}
	// 所有已歸檔 skill 都以 skill（✦）顯示在工具列；已長出 skill 的小程式則在
	// ListGoProgramAuthoringCatalog 端被濾掉，確保同一能力只出現一組（統一為 skill）。
	for i := range manifests {
		a.registerSkillTool(&manifests[i])
	}
}

// archivedSkillIDSet 回傳所有已歸檔 skill 的 skill_id 集合，供小程式清單去重用。
func (a *App) archivedSkillIDSet() map[string]bool {
	out := map[string]bool{}
	if a == nil || a.skillArchive == nil {
		return out
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return out
	}
	for _, m := range manifests {
		if m.SkillID != "" {
			out[m.SkillID] = true
		}
	}
	return out
}

// skillSearchContent 把 skill manifest 整理成搜尋語料：第一行是「技能 名稱：摘要」，
// 其後補上 skill_id 與動作／領域／用途標籤，方便以名稱或關鍵詞命中。摘要優先取
// manifest.Description，沒有時用 action/domain 標籤合成，確保搜尋結果可讀。
// skillSummaryLine 取出單行的 skill 功能摘要：優先用 manifest.Description，
// 沒有時用 action／domain 標籤合成，兩者都缺則回空字串。
func skillSummaryLine(m skill_step.SkillManifest) string {
	if s := strings.TrimSpace(m.Description); s != "" {
		return s
	}
	var parts []string
	if len(m.Tags.ActionTag) > 0 {
		parts = append(parts, strings.Join(m.Tags.ActionTag, "、"))
	}
	if len(m.Tags.DomainTag) > 0 {
		parts = append(parts, strings.Join(m.Tags.DomainTag, "、"))
	}
	return strings.TrimSpace(strings.Join(parts, "；"))
}

// synthesizeSkillDescription 在 skill 沒有 Description 時，用現有欄位（名稱、Tags）
// 合成一句保底用法說明。內容不假裝知道細節，只把已知的動作／領域整理出來，並提示
// 使用方式，避免「怎麼用」只看到「已安裝 skill」。
func synthesizeSkillDescription(m skill_step.SkillManifest) string {
	name := firstNonEmpty(m.DisplayName, m.SkillID)
	parts := []string{"「" + name + "」技能"}
	var meaningful []string
	for _, tag := range m.Tags.ActionTag {
		if t := strings.TrimSpace(tag); t != "" {
			meaningful = append(meaningful, t)
		}
	}
	if len(meaningful) > 0 {
		parts = append(parts, "動作："+strings.Join(meaningful, "、"))
	}
	var domains []string
	for _, tag := range m.Tags.DomainTag {
		t := strings.TrimSpace(tag)
		if t != "" && t != name { // DomainTag 常等於名稱，避免贅述
			domains = append(domains, t)
		}
	}
	if len(domains) > 0 {
		parts = append(parts, "領域："+strings.Join(domains, "、"))
	}
	parts = append(parts, "使用時請說明要它做什麼並附上需要的資料或先載入相關檔案")
	return strings.Join(parts, "；")
}

func curatedSkillDescription(m skill_step.SkillManifest) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(firstNonEmpty(m.DisplayName, m.SkillID)))
	switch {
	case strings.Contains(name, "產出電料bom"), strings.Contains(name, "產出電料BOM"):
		return "讀取電料BOM與編碼紀錄，依指定機台交叉比對料號並補齊品名、規格、單價，輸出整合後的電料BOM表；需要輸入：機台、電料BOM、編碼紀錄；資料來源：已載入或匯入的 CSV/XLSX 表格；輸出：指定機台的整合後電料BOM表", true
	case strings.Contains(name, "穿衣建議"):
		return "根據輸入的溫度提供穿衣建議；需要輸入：temperature 溫度數值；輸出：result 穿衣建議文字", true
	default:
		return "", false
	}
}

func fallbackSkillDescription(m skill_step.SkillManifest) string {
	return strings.TrimSpace(synthesizeSkillDescription(m))
}

func isFallbackSkillDescription(m skill_step.SkillManifest) bool {
	desc := strings.TrimSpace(m.Description)
	if desc == "" {
		return true
	}
	if desc == fallbackSkillDescription(m) {
		return true
	}
	return strings.Contains(desc, "使用時請說明要它做什麼並附上需要的資料或先載入相關檔案")
}

func isStaleCuratedSkillDescription(m skill_step.SkillManifest) bool {
	desc := strings.TrimSpace(m.Description)
	if desc == "" {
		return false
	}
	current, ok := curatedSkillDescription(m)
	if !ok || desc == current {
		return false
	}
	for _, stale := range staleCuratedSkillDescriptions(m) {
		if desc == stale {
			return true
		}
	}
	return false
}

func staleCuratedSkillDescriptions(m skill_step.SkillManifest) []string {
	name := strings.ToLower(strings.TrimSpace(firstNonEmpty(m.DisplayName, m.SkillID)))
	switch {
	case strings.Contains(name, "產出電料bom"), strings.Contains(name, "產出電料BOM"):
		return []string{
			"讀取電料BOM與編碼紀錄，交叉比對料號並補齊品名、規格、單價，輸出整合後的電料BOM表；需要輸入：電料BOM、編碼紀錄；資料來源：已載入或匯入的 CSV/XLSX 表格；輸出：整合後電料BOM表",
			"「產出電料Bom」技能；使用時請說明要它做什麼並附上需要的資料或先載入相關檔案",
		}
	default:
		return nil
	}
}

func shouldRefreshSkillDescription(m skill_step.SkillManifest) bool {
	return isFallbackSkillDescription(m) || isStaleCuratedSkillDescription(m)
}

func bestSkillDescription(m skill_step.SkillManifest) string {
	if desc, ok := curatedSkillDescription(m); ok {
		return desc
	}
	if desc, ok := goProgramSkillDescriptionFromDisk(m.SkillID); ok {
		return desc
	}
	return fallbackSkillDescription(m)
}

// backfillSkillDescriptions 啟動時掃過所有已歸檔 skill，把 Description 空白或仍是
// 保底說明者補成更完整的用法並存回（UpdateManifest 會重算 hash）。使用者已填的
// 自訂說明不覆蓋。
func (a *App) backfillSkillDescriptions(traceID string) {
	if a == nil || a.skillArchive == nil {
		return
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		debugtrace.Record("go.skill_backfill.error", traceID, map[string]interface{}{"error": err.Error()})
		return
	}
	filled := 0
	for i := range manifests {
		m := manifests[i]
		if !shouldRefreshSkillDescription(m) {
			continue
		}
		desc := bestSkillDescription(m)
		if desc == "" || desc == strings.TrimSpace(m.Description) {
			continue
		}
		m.Description = desc
		if err := a.skillArchive.UpdateManifest(&m); err != nil {
			debugtrace.Record("go.skill_backfill.update_error", traceID, map[string]interface{}{
				"skill_id": m.SkillID, "error": err.Error(),
			})
			continue
		}
		filled++
	}
	if filled > 0 {
		debugtrace.Record("go.skill_backfill.done", traceID, map[string]interface{}{
			"filled": filled, "total": len(manifests),
		})
	}
}

// isSkillUsageRequest 偵測「某 skill 怎麼用／用法／使用方式」這類詢問特定 skill
// 用法的意圖（問法詞）。是否真有對應 skill 由 matchArchivedSkillByName 決定。
func isSkillUsageRequest(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	return containsAny(t, []string{
		"怎麼用", "要怎麼用", "怎樣用", "如何用", "如何使用", "使用方式",
		"用法", "怎麼操作", "如何操作", "怎麼跑", "怎麼執行", "how to use",
	})
}

// matchArchivedSkillByName 在使用者輸入中找出被提到的已安裝 skill（以 DisplayName
// 子字串比對，取最長命中者，避免短名誤判）。找不到回 false。
func (a *App) matchArchivedSkillByName(userText string) (skill_step.SkillManifest, bool) {
	if a == nil || a.skillArchive == nil {
		return skill_step.SkillManifest{}, false
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return skill_step.SkillManifest{}, false
	}
	lower := strings.ToLower(userText)
	var best skill_step.SkillManifest
	bestLen := 0
	for i := range manifests {
		m := manifests[i]
		skill_step.EnsureLifecycle(&m)
		if lc := m.Lifecycle; lc != nil && lc.Status == skill_step.LifecycleDisabled {
			continue
		}
		name := strings.TrimSpace(m.DisplayName)
		if name == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(name)) && len(name) > bestLen {
			best = m
			bestLen = len(name)
		}
	}
	if bestLen == 0 {
		return skill_step.SkillManifest{}, false
	}
	return best, true
}

// formatSkillUsageCard 把 skill 的「用法欄位」整理成可讀的用法卡：說明、動作/領域、
// 資料存取權限，以及怎麼下指令。Description 為空時誠實告知並指出去哪補。
func (a *App) formatSkillUsageCard(m skill_step.SkillManifest) string {
	name := firstNonEmpty(m.DisplayName, m.SkillID)
	var b strings.Builder
	b.WriteString("技能「" + name + "」用法\n")
	desc := strings.TrimSpace(m.Description)
	if desc != "" {
		b.WriteString("\n說明：" + desc + "\n")
	}
	if len(m.Tags.ActionTag) > 0 {
		b.WriteString("動作：" + strings.Join(m.Tags.ActionTag, "、") + "\n")
	}
	if len(m.Tags.DomainTag) > 0 {
		b.WriteString("領域：" + strings.Join(m.Tags.DomainTag, "、") + "\n")
	}
	var ps []string
	if fs := strings.TrimSpace(m.Permissions.Filesystem); fs != "" {
		ps = append(ps, "檔案("+fs+")")
	}
	if nw := strings.TrimSpace(m.Permissions.Network); nw != "" {
		ps = append(ps, "網路("+nw+")")
	}
	if len(ps) > 0 {
		b.WriteString("資料存取：" + strings.Join(ps, "、") + "\n")
	}
	if desc == "" {
		b.WriteString("\n（此技能尚未填寫用法說明。可在 skill 的 Description／README 補上「做什麼、要輸入什麼資料」，之後這裡就會顯示。）\n")
	}
	b.WriteString("\n使用方式：在輸入框直接說要它做什麼，並附上需要的資料或先載入相關檔案，例如「用「" + name + "」處理我載入的檔案」。")
	return b.String()
}

// maybeHandleSkillUsage 在使用者問「某 skill 怎麼用」且確實有對應已安裝 skill 時，
// 直接回傳用法卡（讀 skill 的用法欄位），否則回 false 交給後續流程。
func (a *App) maybeHandleSkillUsage(userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	_ = sessionID
	if !isSkillUsageRequest(userText) {
		return nil, false
	}
	m, ok := a.matchArchivedSkillByName(userText)
	if !ok {
		return nil, false
	}
	a.pushActionStatus("技能", "說明技能用法")
	resp := skill_step.CLIResponse{Text: a.formatSkillUsageCard(m)}
	debugtrace.Record("go.skill_usage.direct", traceID, map[string]interface{}{
		"skill_id": m.SkillID,
		"has_desc": strings.TrimSpace(m.Description) != "",
	})
	return &resp, true
}

// isListSkillsRequest 判斷使用者是不是在問「你有哪些 skill／列出技能」這類
// 列舉意圖，而不是要「用某個 skill 做事」。必須同時含 skill/技能關鍵詞與列舉動詞。
func isListSkillsRequest(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	if !strings.Contains(t, "skill") && !strings.Contains(t, "技能") {
		return false
	}
	for _, kw := range []string{
		"列出", "哪些", "有什麼", "有甚麼", "清單", "列表", "list", "所有", "全部",
		"可以用", "可用", "能用", "支援", "支持", "你有", "我有", "現有", "有的", "有哪",
	} {
		if strings.Contains(t, kw) {
			return true
		}
	}
	return false
}

// formatInstalledSkills 列出本機已安裝（已歸檔）的 skill，含名稱與功能摘要，
// 讓「列出你有的 skill」由 App 自己回答，而不是交給 CLI 列出它自己的工具。
func (a *App) formatInstalledSkills() string {
	if a == nil || a.skillArchive == nil {
		return "目前沒有已安裝的 skill。"
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return "目前沒有已安裝的 skill。"
	}
	visible := manifests[:0]
	for _, m := range manifests {
		mm := m
		skill_step.EnsureLifecycle(&mm)
		if lc := mm.Lifecycle; lc != nil && lc.Status == skill_step.LifecycleDisabled {
			continue
		}
		visible = append(visible, mm)
	}
	if len(visible) == 0 {
		return "目前沒有已安裝的 skill。"
	}
	sort.SliceStable(visible, func(i, j int) bool {
		return firstNonEmpty(visible[i].DisplayName, visible[i].SkillID) <
			firstNonEmpty(visible[j].DisplayName, visible[j].SkillID)
	})
	var b strings.Builder
	b.WriteString(fmt.Sprintf("目前已安裝 %d 個 skill：", len(visible)))
	for _, m := range visible {
		name := firstNonEmpty(m.DisplayName, m.SkillID)
		b.WriteString("\n- " + name)
		if summary := skillSummaryLine(m); summary != "" {
			b.WriteString("：" + summary)
		}
	}
	return b.String()
}

func skillSearchContent(m skill_step.SkillManifest) string {
	name := firstNonEmpty(m.DisplayName, m.SkillID)
	summary := skillSummaryLine(m)
	if summary == "" {
		summary = "已安裝 skill"
	}
	lines := []string{
		"技能 " + name + "：" + summary,
		"skill 技能 " + m.SkillID,
		"動作 " + strings.Join(m.Tags.ActionTag, " "),
		"領域 " + strings.Join(m.Tags.DomainTag, " "),
		"用途 " + strings.Join(m.Tags.PurposeTag, " "),
	}
	return strings.Join(lines, "\n")
}

// MarkSkillFirstUseExplained 持久化「初次使用說明卡已顯示」旗標。
// 前端在使用者關閉說明卡後呼叫，下次啟動不再顯示。
func (a *App) MarkSkillFirstUseExplained() (interface{}, error) {
	next, err := a.uiSettingsService.MarkSkillFirstUseExplained()
	return frontendDTO(next), err
}

// ScanSkillFolder scans a folder at path, caches the preview, and returns it.
func (a *App) ScanSkillFolder(path string) (interface{}, error) {
	preview, err := a.skillArchive.ScanFolder(path)
	if err != nil {
		return nil, err
	}
	a.cacheMu.Lock()
	a.previewCache[preview.PreviewID] = preview
	a.cacheMu.Unlock()
	return frontendDTO(preview), nil
}

// ConfirmSkillArchive confirms a pending scan preview and archives the skill.
func (a *App) ConfirmSkillArchive(previewID string) (interface{}, error) {
	a.cacheMu.Lock()
	preview, ok := a.previewCache[previewID]
	a.cacheMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("app: ConfirmSkillArchive: preview %q not found", previewID)
	}
	manifest, err := a.skillArchive.ConfirmArchive(preview)
	if err == nil && manifest != nil {
		a.registerSkillTool(manifest)
		a.eventBus.Emit("tools:list_changed", nil)
	}
	return frontendDTO(manifest), err
}

// ListArchivedSkills returns all skill manifests currently in the archive.
func (a *App) ListArchivedSkills() (interface{}, error) {
	manifests, err := a.skillArchive.ListArchived()
	return frontendDTO(manifests), err
}

// ResolveSkillForAction parses actionTarget (format: "動作ㄌ目標"), resolves it
// against the archive, caches the result, and returns it.
func (a *App) ResolveSkillForAction(actionTarget string, sessionID string) (*skill_step.ResolveResult, error) {
	return a.resolveSkillForActionTarget(actionTarget, sessionID)
}

// BuildSkillContext looks up a cached ResolveResult, builds the Injection,
// stores it in the InjectionStore, and records it in the audit log.
func (a *App) BuildSkillContext(resolveID string, sessionID string) (*skill_step.Injection, error) {
	a.cacheMu.Lock()
	result, ok := a.resolveCache[resolveID]
	a.cacheMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("app: BuildSkillContext: resolve result %q not found", resolveID)
	}
	if result.SelectedSkillID == "" || len(result.Candidates) == 0 {
		return nil, fmt.Errorf("app: BuildSkillContext: no selected skill in resolve result %q", resolveID)
	}
	if sessionID == "" {
		sessionID = result.SessionID
	}
	if sessionID == "" {
		sessionID = a.globalSessionID
	}

	// Find the selected candidate.
	var selected skill_step.Candidate
	for _, c := range result.Candidates {
		if c.SkillID == result.SelectedSkillID {
			selected = c
			break
		}
	}

	// Load manifest for resource refs.
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return nil, err
	}
	var manifest skill_step.SkillManifest
	for _, m := range manifests {
		if m.SkillID == result.SelectedSkillID {
			manifest = m
			break
		}
	}

	inj := skill_step.BuildInjection(sessionID, selected, manifest)
	a.skillInjections.Set(sessionID, inj)

	_ = a.skillAudit.Record(skill_step.InjectionAudit{
		Timestamp:     time.Now(),
		SessionID:     sessionID,
		SkillID:       inj.SkillID,
		ResolveStatus: string(result.Status),
		Reason:        inj.Reason,
		SummaryHash:   inj.SummaryHash,
		Risk:          inj.Risk,
	})

	return inj, nil
}

// ClearSkillContext clears the active skill injection for sessionID and records
// the clear event in the audit log.
func (a *App) ClearSkillContext(sessionID string, reason string) error {
	inj := a.skillInjections.Get(sessionID)
	a.skillInjections.Clear(sessionID)

	skillID := ""
	summaryHash := ""
	risk := ""
	if inj != nil {
		skillID = inj.SkillID
		summaryHash = inj.SummaryHash
		risk = inj.Risk
	}

	now := time.Now()
	return a.skillAudit.Record(skill_step.InjectionAudit{
		Timestamp:   now,
		SessionID:   sessionID,
		SkillID:     skillID,
		SummaryHash: summaryHash,
		Risk:        risk,
		ClearedAt:   &now,
		ClearReason: reason,
	})
}

// GetRecentSkillInjections returns the 2 most recent audit entries for sessionID.
func (a *App) GetRecentSkillInjections(sessionID string) (interface{}, error) {
	injections, err := a.skillAudit.Recent(sessionID, 2)
	return frontendDTO(injections), err
}
