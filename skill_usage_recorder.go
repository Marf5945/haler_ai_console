// skill_usage_recorder.go — skill 使用紀錄 producer。
//
// 背景：sub 匯出包的 tools/ 之所以是空的，是因為整條關聯鏈缺了「源頭」——
// skill 執行成功後沒有任何程式把使用紀錄寫進 tool_history/，sub_meta.json 的
// tools_used 也永遠是空陣列，導致匯出時 resolvePortableSubExportTools 的
// 自動偵測掃不到 skill ID，最後 manifest 寫出 "tools": null。
//
// 本檔補上這個源頭（v4 spec 中 memory/tool_history.jsonl 的 producer）：
//  1. recordSkillUsage    — skill 執行成功後 append 一筆 JSONL 到當前 agent
//     （main 或 sub）的 tool_history/tool_history.jsonl（append-only，符合
//     spec_patch_checker Rule 12），並同步更新 sub_meta.json 的 tools_used。
//  2. copySkillUsageRecords — 「拉出 sub」（SaveMainAsSub）時，把 main 累積的
//     使用紀錄過濾後帶進新 sub，讓匯出偵測鏈接得起來。
//
// 寫入皆為 best-effort：紀錄失敗只記 log，不中斷 skill 執行主流程。
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// toolHistoryFileName 是各 agent tool_history/ 目錄下的使用紀錄檔名。
// 對應 spec_patch_checker v4 contract 的 "memory/tool_history.jsonl" target，
// 此檔必須 append-only（Rule 12），任何修改只能用追加，禁止覆寫或截斷。
const toolHistoryFileName = "tool_history.jsonl"

// skillUsageSchemaVersion 是 SkillUsageRecord 的 schema 版本，升級時 bump。
const skillUsageSchemaVersion = "v1"

// SkillUsageRecord 是一筆 skill 使用紀錄（tool_history.jsonl 的一行）。
// skill_id 與 display_name 都要存：匯出偵測用 ID 比對 data/skills/[id]，
// 顯示名稱則留給人讀與舊資料的 fallback 比對。
type SkillUsageRecord struct {
	SchemaVersion string `json:"schema_version"`        // 固定 skillUsageSchemaVersion
	EventType     string `json:"event_type"`            // 固定 "skill_usage"
	SkillID       string `json:"skill_id"`              // 歸檔 skill ID（如 go-program-產出電料bom）
	DisplayName   string `json:"display_name"`          // 顯示名稱（如 產出電料Bom）
	AgentID       string `json:"agent_id"`              // 執行當下的 agent（main 或 sub-xxx）
	SessionID     string `json:"session_id,omitempty"`  // 對話 session
	TraceID       string `json:"trace_id,omitempty"`    // debugtrace 追蹤 ID
	ExecutedAt    string `json:"executed_at"`           // RFC3339 執行完成時間
}

// recordSkillUsage 是 skill 執行成功後的統一記錄入口。
// skillID / displayName 允許只給其中一個，另一個會從 skill 歸檔反查補齊。
// 寫入目標是「目前 active agent」的對話根目錄（main → projectRoot、
// sub → subagents/callable/[id]），與 talk_full.md 的歸屬規則一致。
func (a *App) recordSkillUsage(skillID, displayName, sessionID, traceID string) {
	if a == nil {
		return
	}
	skillID = strings.TrimSpace(skillID)
	displayName = strings.TrimSpace(displayName)
	if skillID == "" && displayName == "" {
		return // 沒有任何可識別資訊，無從記錄
	}

	// 從歸檔反查補齊缺的那一邊（best-effort，查不到就照原樣記）。
	resolvedID, resolvedName := a.resolveArchivedSkillIdentity(skillID, displayName)
	if skillID == "" {
		skillID = resolvedID
	}
	if displayName == "" {
		displayName = resolvedName
	}

	agentID := a.activeConversationAgent()
	convRoot, err := conversationRootForAgent(agentID)
	if err != nil {
		log.Printf("[SKILL-USAGE] 解析 agent 對話根目錄失敗（agent=%s）: %v", agentID, err)
		return
	}

	rec := SkillUsageRecord{
		SchemaVersion: skillUsageSchemaVersion,
		EventType:     "skill_usage",
		SkillID:       skillID,
		DisplayName:   displayName,
		AgentID:       agentID,
		SessionID:     strings.TrimSpace(sessionID),
		TraceID:       strings.TrimSpace(traceID),
		ExecutedAt:    time.Now().Format(time.RFC3339),
	}
	if err := appendSkillUsageRecord(convRoot, rec); err != nil {
		log.Printf("[SKILL-USAGE] 寫入 tool_history 失敗: %v", err)
		return
	}
	// sub 才有 sub_meta.json；main 沒有此檔，updateSubMetaToolsUsed 內部會略過。
	if err := updateSubMetaToolsUsed(convRoot, rec.SkillID, rec.DisplayName); err != nil {
		log.Printf("[SKILL-USAGE] 更新 sub_meta.tools_used 失敗: %v", err)
	}
	log.Printf("[SKILL-USAGE] 已記錄 skill 使用: id=%q name=%q agent=%s", skillID, displayName, agentID)
}

// resolveArchivedSkillIdentity 從 skill 歸檔反查 (skillID, displayName) 配對。
// 比對規則沿用 normalizeGoProgramLookup（去空白/連字號/底線、轉小寫），
// 與 existingSkillActionChain、findExistingProgramSkill 一致。
func (a *App) resolveArchivedSkillIdentity(skillID, displayName string) (string, string) {
	if a == nil || a.skillArchive == nil {
		return skillID, displayName
	}
	wantID := normalizeGoProgramLookup(skillID)
	wantName := normalizeGoProgramLookup(displayName)
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return skillID, displayName
	}
	for _, m := range manifests {
		mid := normalizeGoProgramLookup(m.SkillID)
		mname := normalizeGoProgramLookup(m.DisplayName)
		if (wantID != "" && (mid == wantID || mname == wantID)) ||
			(wantName != "" && (mid == wantName || mname == wantName)) {
			return m.SkillID, m.DisplayName
		}
	}
	return skillID, displayName
}

// appendSkillUsageRecord 把一筆紀錄 append 到 convRoot/tool_history/tool_history.jsonl。
// 只用 O_APPEND 開檔，維持 append-only 不變量（spec Rule 12）。
func appendSkillUsageRecord(convRoot string, rec SkillUsageRecord) error {
	dir := filepath.Join(convRoot, "tool_history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("建立 tool_history 目錄失敗: %w", err)
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("序列化使用紀錄失敗: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, toolHistoryFileName),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("開啟 tool_history.jsonl 失敗: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("追加 tool_history.jsonl 失敗: %w", err)
	}
	return nil
}

// loadSkillUsageRecords 讀回 convRoot 下全部使用紀錄。
// 檔案不存在回空清單；單行壞掉（手改/截斷）跳過該行，不整檔失敗。
func loadSkillUsageRecords(convRoot string) []SkillUsageRecord {
	f, err := os.Open(filepath.Join(convRoot, "tool_history", toolHistoryFileName))
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []SkillUsageRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 單行上限 1MB，防呆
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec SkillUsageRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue // 壞行跳過
		}
		out = append(out, rec)
	}
	return out
}

// updateSubMetaToolsUsed 把 skill 識別字併入 sub_meta.json 的 tools_used（去重）。
// convRoot 下沒有 sub_meta.json（= main agent）時直接略過，不視為錯誤。
func updateSubMetaToolsUsed(convRoot string, entries ...string) error {
	metaPath := filepath.Join(convRoot, "sub_meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // main 沒有 sub_meta.json，正常情況
		}
		return err
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("解析 sub_meta.json 失敗: %w", err)
	}

	// 還原既有 tools_used（JSON 陣列解出來是 []interface{}）。
	existing := map[string]bool{}
	var tools []string
	if raw, ok := meta["tools_used"].([]interface{}); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				if !existing[s] {
					existing[s] = true
					tools = append(tools, s)
				}
			}
		}
	}
	// 併入新識別字（skill_id 優先，display_name 其次；兩個都記方便日後比對）。
	changed := false
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" || existing[e] {
			continue
		}
		existing[e] = true
		tools = append(tools, e)
		changed = true
	}
	if !changed {
		return nil
	}
	meta["tools_used"] = tools
	out, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, out, 0o600)
}

// copySkillUsageRecords 把 srcRoot 的使用紀錄帶到 dstSubDir（拉出 sub 用）。
//
// 過濾規則：main 的 tool_history 累積了所有對話的紀錄，不該全部照搬；
// 只帶「skill ID 或顯示名稱有出現在這段委派對話內容（talkContent）」的紀錄。
// talkContent 為空時視為整段搬移（全帶）。
// 回傳 (帶入筆數, 帶入的識別字清單)，識別字供呼叫端寫 sub_meta.tools_used。
func copySkillUsageRecords(srcRoot, dstSubDir, talkContent string) (int, []string) {
	records := loadSkillUsageRecords(srcRoot)
	if len(records) == 0 {
		return 0, nil
	}
	talkLower := strings.ToLower(talkContent)

	seen := map[string]bool{} // 以 skill_id+name 去重，同一 skill 只帶一筆
	var copied int
	var idents []string
	for _, rec := range records {
		if strings.TrimSpace(talkLower) != "" && !usageMentionedInTalk(rec, talkLower) {
			continue // 這筆 skill 與委派片段無關，不帶
		}
		key := rec.SkillID + "\x00" + rec.DisplayName
		if seen[key] {
			continue
		}
		seen[key] = true
		if err := appendSkillUsageRecord(dstSubDir, rec); err != nil {
			log.Printf("[SKILL-USAGE] 複製使用紀錄到 sub 失敗: %v", err)
			continue
		}
		copied++
		for _, ident := range []string{rec.SkillID, rec.DisplayName} {
			if strings.TrimSpace(ident) != "" {
				idents = append(idents, ident)
			}
		}
	}
	return copied, idents
}

// uniqueNonEmptyStrings 去重並剔除空白字串，保持原順序。
// 給 SaveMainAsSub 寫 sub_meta.tools_used 用；nil 輸入回空切片
// （JSON 序列化成 [] 而非 null，與既有 sub_meta 格式一致）。
func uniqueNonEmptyStrings(items []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// usageMentionedInTalk 判斷一筆使用紀錄是否與對話內容相關：
// skill ID 或顯示名稱（小寫化後）出現在對話全文即視為相關。
func usageMentionedInTalk(rec SkillUsageRecord, talkLower string) bool {
	for _, ident := range []string{rec.SkillID, rec.DisplayName} {
		ident = strings.ToLower(strings.TrimSpace(ident))
		if ident != "" && strings.Contains(talkLower, ident) {
			return true
		}
	}
	return false
}
