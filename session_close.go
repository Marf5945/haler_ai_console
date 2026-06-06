// session_close.go — §30 Main/Sub 關閉視窗流程實作。
//
// 當使用者關閉視窗時：
//  1. Wails OnBeforeClose 觸發 → 前端顯示關閉對話框
//  2. 前端呼叫 AnalyzeSessionForSub() 取得操作統計
//  3. 若 main 直接操作 ≥40%，顯示「要不要將這段建立為新的工作流？」
//  4. 使用者選「需要」→ SaveMainAsSub() 建立 sub
//  5. 使用者選「不要」→ 直接關閉
//  6. 無論選什麼，下次啟動 main 對話清空（ClearMainTalk）
//
// v4.0 — 對應 Spec §30.2 Path A。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/data/memory"
	"ui_console/data/storage"
	"ui_console/orchestration/dag"
)

// ──────────────────────────────────────────────
// 操作統計（§30.2 Path A step 2）
// ──────────────────────────────────────────────

// SessionAnalysis 是前端用來決定是否顯示 sub 建立對話框的資料。
type SessionAnalysis struct {
	TotalActions      int     `json:"total_actions"`
	MainDirectActions int     `json:"main_direct_actions"`
	DelegatedActions  int     `json:"delegated_actions"`
	DirectRatio       float64 `json:"direct_ratio"`
	ShouldPrompt      bool    `json:"should_prompt"`
	HasContent        bool    `json:"has_content"`
	SuggestedName     string  `json:"suggested_name"`
	Mode              string  `json:"mode,omitempty"`
	AgentID           string  `json:"agent_id,omitempty"`
}

type CreatedSubagent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SubDir    string `json:"sub_dir"`
	MemoryDir string `json:"memory_dir"`
}

// AnalyzeSessionForSub 分析本次 session 的操作比例。
// 前端在 beforeunload 時呼叫，用來決定是否顯示「存成 sub」對話框。
func (a *App) AnalyzeSessionForSub() SessionAnalysis {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	talk, err := memory.ReadTalkFull(projectRoot)
	if err != nil || strings.TrimSpace(talk) == "" {
		return SessionAnalysis{HasContent: false}
	}

	// 解析對話內容，計算操作統計
	sections := strings.Split(talk, "\n## [")
	totalActions := 0
	mainDirect := 0
	delegated := 0
	var firstUserMsg string

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" || strings.HasPrefix(section, "# Talk Full") {
			continue
		}

		totalActions++

		lines := strings.SplitN(section, "\n", 2)
		if len(lines) < 2 {
			continue
		}
		header := strings.TrimSpace(lines[0])
		body := strings.TrimSpace(lines[1])

		// 提取 role
		role := header
		if idx := strings.LastIndex(role, "]"); idx >= 0 {
			role = strings.TrimSpace(role[idx+1:])
		}

		// 記錄第一條使用者訊息（用於建議名稱）
		if role == "user" && firstUserMsg == "" {
			firstUserMsg = body
			if len(firstUserMsg) > 40 {
				firstUserMsg = firstUserMsg[:40]
			}
		}

		// 簡單判斷：含有 delegate / sub 關鍵字的算 delegated
		lower := strings.ToLower(body)
		if strings.Contains(lower, "delegate_to_sub") ||
			strings.Contains(lower, "delegated to sub") {
			delegated++
		} else if role == "assistant" || role == "ai" {
			// AI 直接回覆的操作算 main direct
			mainDirect++
		}
	}

	if totalActions == 0 {
		return SessionAnalysis{HasContent: false}
	}

	directRatio := float64(mainDirect) / float64(totalActions)

	// 建議名稱：取第一條使用者訊息的前 40 字
	suggestedName := firstUserMsg
	if suggestedName == "" {
		suggestedName = fmt.Sprintf("session_%s", time.Now().Format("0102_1504"))
	}

	return SessionAnalysis{
		TotalActions:      totalActions,
		MainDirectActions: mainDirect,
		DelegatedActions:  delegated,
		DirectRatio:       directRatio,
		ShouldPrompt:      directRatio >= 0.4 && totalActions >= 3,
		HasContent:        true,
		SuggestedName:     suggestedName,
	}
}

func (a *App) SetActiveConversationAgent(agentID string) {
	id := strings.TrimSpace(agentID)
	if id == "" {
		id = "main"
	}
	a.closeMu.Lock()
	a.activeAgentID = id
	a.closeMu.Unlock()
}

func (a *App) activeConversationAgent() string {
	a.closeMu.Lock()
	defer a.closeMu.Unlock()
	if strings.TrimSpace(a.activeAgentID) == "" {
		return "main"
	}
	return a.activeAgentID
}

func (a *App) activeTaskRunForClose() (*dag.DAGRun, bool) {
	a.taskMu.Lock()
	runID := a.activeTaskRunID
	a.taskMu.Unlock()
	if strings.TrimSpace(runID) == "" {
		return nil, false
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	run, err := dag.LoadFullRun(projectRoot, runID)
	if err != nil {
		return nil, false
	}
	switch run.Status {
	case "planning", "running", "waiting_review":
		return run, true
	default:
		return run, false
	}
}

// ──────────────────────────────────────────────
// 存成 Sub（§30.2 Path A step 3 — 使用者選「需要」）
// ──────────────────────────────────────────────

// SaveMainAsSub 將目前 main 的 talk_full 搬移到新 sub 目錄。
// 回傳新建的 sub ID。
func (a *App) SaveMainAsSub(subName string) (string, error) {
	root := appDataRoot()
	projectRoot := storage.ProjectRoot(root, "default")

	// 產生 sub ID
	subID := fmt.Sprintf("sub-%s", time.Now().Format("20060102-150405"))
	if subName == "" {
		subName = subID
	}

	// 建立 sub 目錄：subagents/callable/[sub-id]/memory/
	subDir := filepath.Join(projectRoot, "subagents", "callable", subID)
	subMemDir := filepath.Join(subDir, "memory")
	if err := os.MkdirAll(subMemDir, 0755); err != nil {
		return "", fmt.Errorf("建立 sub 目錄失敗: %w", err)
	}

	// 搬移 talk_full.md → sub 的 memory/
	srcTalk := filepath.Join(projectRoot, "memory", memory.FileTalkFull)
	dstTalk := filepath.Join(subMemDir, memory.FileTalkFull)

	talkData, err := os.ReadFile(srcTalk)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("讀取 talk_full 失敗: %w", err)
	}
	if len(talkData) > 0 {
		if err := os.WriteFile(dstTalk, talkData, 0o600); err != nil {
			return "", fmt.Errorf("寫入 sub talk_full 失敗: %w", err)
		}
	}

	// 複製 summaries.md（若存在）
	srcSummary := filepath.Join(projectRoot, "memory", "summaries.md")
	if data, err := os.ReadFile(srcSummary); err == nil && len(data) > 0 {
		os.WriteFile(filepath.Join(subMemDir, "summaries.md"), data, 0o600)
	}

	// 寫入 sub metadata
	meta := map[string]interface{}{
		"id":           subID,
		"name":         subName,
		"created_at":   time.Now().Format(time.RFC3339),
		"created_from": "session_close",
		"triggers":     []string{},
		"tools_used":   []string{},
		"action_tags":  []string{},
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(filepath.Join(subDir, "sub_meta.json"), metaJSON, 0o600)

	log.Printf("session_close: main talk saved as sub %q (%s)", subName, subID)
	return subID, nil
}

func (a *App) CreateSubagent(subName string) (*CreatedSubagent, error) {
	root := appDataRoot()
	projectRoot := storage.ProjectRoot(root, "default")
	created, err := createSubagentInProject(projectRoot, subName)
	if err != nil {
		return nil, err
	}
	if err := a.getTabOrderManager().Append(created.ID); err != nil {
		return nil, fmt.Errorf("更新 sub tab order 失敗: %w", err)
	}
	log.Printf("session_close: created empty sub %q (%s)", created.Name, created.ID)
	return created, nil
}

func (a *App) RenameSubagent(currentName string, nextName string) (*CreatedSubagent, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	renamed, err := renameSubagentInProject(projectRoot, currentName, nextName)
	if err != nil {
		return nil, err
	}
	log.Printf("session_close: renamed sub %s to %q", renamed.ID, renamed.Name)
	return renamed, nil
}

func createSubagentInProject(projectRoot string, subName string) (*CreatedSubagent, error) {
	subID := fmt.Sprintf("sub-%s", time.Now().Format("20060102-150405"))
	if strings.TrimSpace(subName) == "" {
		subName = fmt.Sprintf("新haㄌer %s", time.Now().Format("15:04"))
	}

	subDir := filepath.Join(projectRoot, "subagents", "callable", subID)
	subMemDir := filepath.Join(subDir, "memory")
	for _, dir := range []string{
		subMemDir,
		filepath.Join(subDir, "dag"),
		filepath.Join(subDir, "tool_history"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("建立 sub 目錄失敗: %w", err)
		}
	}

	talkPath := filepath.Join(subMemDir, memory.FileTalkFull)
	if err := os.WriteFile(talkPath, []byte("# Talk Full\n\n"), 0o600); err != nil {
		return nil, fmt.Errorf("建立 sub talk_full 失敗: %w", err)
	}

	meta := map[string]interface{}{
		"id":           subID,
		"name":         subName,
		"created_at":   time.Now().Format(time.RFC3339),
		"created_from": "manual_create",
		"triggers":     []string{},
		"tools_used":   []string{},
		"action_tags":  []string{},
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(subDir, "sub_meta.json"), metaJSON, 0o600); err != nil {
		return nil, fmt.Errorf("寫入 sub metadata 失敗: %w", err)
	}

	return &CreatedSubagent{
		ID:        subID,
		Name:      subName,
		SubDir:    subDir,
		MemoryDir: subMemDir,
	}, nil
}

func renameSubagentInProject(projectRoot string, currentName string, nextName string) (*CreatedSubagent, error) {
	currentName = strings.TrimSpace(currentName)
	nextName = strings.TrimSpace(nextName)
	if currentName == "" {
		return nil, fmt.Errorf("sub 名稱不可為空")
	}
	if currentName == "主haㄌer" {
		return nil, fmt.Errorf("主haㄌer 不可改名")
	}
	if nextName == "" {
		return nil, fmt.Errorf("新 sub 名稱不可為空")
	}

	callableDir := filepath.Join(projectRoot, "subagents", "callable")
	entries, err := os.ReadDir(callableDir)
	if err != nil {
		return nil, fmt.Errorf("讀取 sub 清單失敗: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		subDir := filepath.Join(callableDir, id)
		metaPath := filepath.Join(subDir, "sub_meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta map[string]interface{}
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		name, _ := meta["name"].(string)
		if name != currentName && id != currentName {
			continue
		}

		meta["name"] = nextName
		if _, ok := meta["id"].(string); !ok {
			meta["id"] = id
		}
		nextData, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("序列化 sub metadata 失敗: %w", err)
		}
		if err := os.WriteFile(metaPath, nextData, 0o600); err != nil {
			return nil, fmt.Errorf("寫入 sub metadata 失敗: %w", err)
		}

		return &CreatedSubagent{
			ID:        id,
			Name:      nextName,
			SubDir:    subDir,
			MemoryDir: filepath.Join(subDir, "memory"),
		}, nil
	}

	return nil, fmt.Errorf("找不到 sub: %s", currentName)
}

// ──────────────────────────────────────────────
// 清除 Main 對話（§30.1 — Main starts fresh each session）
// ──────────────────────────────────────────────

// ClearMainTalk 清空 main 的 talk_full.md。
// 在視窗關閉流程完成後（無論存不存 sub）呼叫。
// 下次啟動 app 時 main 就是乾淨的。
func (a *App) ClearMainTalk() error {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	talkPath := filepath.Join(projectRoot, "memory", memory.FileTalkFull)

	// 清空檔案（保留檔案但內容清空）
	if err := os.WriteFile(talkPath, []byte(""), 0o600); err != nil {
		return fmt.Errorf("清除 main talk_full 失敗: %w", err)
	}

	log.Printf("session_close: main talk_full cleared")
	return nil
}

// ──────────────────────────────────────────────
// Wails OnBeforeClose 回呼
// ──────────────────────────────────────────────

// beforeClose 在 Wails 視窗即將關閉時被呼叫。
// 回傳 true = 阻止關閉（讓前端顯示對話框）。
// 前端完成對話框流程後呼叫 ConfirmClose() 真正關閉。
func (a *App) beforeClose(ctx context.Context) bool {
	// §27: 關閉排程引擎，等待正在執行的 Job 完成
	if a.schedulerService != nil {
		a.schedulerService.Stop()
	}
	a.closeMu.Lock()
	if a.allowClose {
		a.allowClose = false
		a.closeMu.Unlock()
		return false
	}
	a.closeMu.Unlock()

	if run, ok := a.activeTaskRunForClose(); ok {
		a.eventBus.Emit("session:close_prompt", map[string]interface{}{
			"analysis": SessionAnalysis{
				ShouldPrompt:  true,
				HasContent:    true,
				Mode:          "active_task",
				SuggestedName: run.Title,
			},
		})
		return true
	}

	activeAgent := a.activeConversationAgent()
	if activeAgent != "main" {
		a.eventBus.Emit("session:close_prompt", map[string]interface{}{
			"analysis": SessionAnalysis{
				ShouldPrompt: true,
				HasContent:   true,
				Mode:         "active_sub",
				AgentID:      activeAgent,
			},
		})
		return true
	}

	analysis := a.AnalyzeSessionForSub()

	if !analysis.HasContent || !analysis.ShouldPrompt {
		// 沒內容或不需要提示 → 直接清空並關閉
		a.ClearMainTalk()
		return false // 允許關閉
	}

	// 有內容且需要提示 → 阻止關閉，讓前端顯示對話框
	a.eventBus.Emit("session:close_prompt", map[string]interface{}{
		"analysis": analysis,
	})
	return true // 阻止關閉
}

// ConfirmClose 前端對話框完成後呼叫。
// saveAsSub=true 時先存成 sub 再關閉。
func (a *App) ConfirmClose(saveAsSub bool, subName string) error {
	if saveAsSub {
		if _, err := a.SaveMainAsSub(subName); err != nil {
			log.Printf("session_close: save as sub failed: %v", err)
			return err
		}
	}

	a.taskMu.Lock()
	activeTaskRunID := a.activeTaskRunID
	a.taskMu.Unlock()
	if strings.TrimSpace(activeTaskRunID) != "" {
		if _, err := a.CancelTaskProgress(activeTaskRunID, "app_close"); err != nil {
			log.Printf("session_close: cancel active task failed: %v", err)
		}
	}

	// 無論如何都清空 main talk
	a.ClearMainTalk()

	a.closeMu.Lock()
	a.allowClose = true
	a.closeMu.Unlock()

	// 通知前端可以關閉了
	a.eventBus.Emit("session:close_confirmed", nil)
	return nil
}
