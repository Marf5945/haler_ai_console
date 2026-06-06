// Package storage 定義 AI Console v3.6.1 的雙層儲存架構（§17）。
// Layer 1: data/projects/[project]/ — 專案狀態
// Layer 2: data/personas/[persona-id]/ — 全域資產（跨專案共享）
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// ──────────────────────────────────────────────
// 根路徑函式
// ──────────────────────────────────────────────

// DataRoot 回傳 data/ 的絕對路徑。baseDir 為應用程式根目錄。
func DataRoot(baseDir string) string {
	return filepath.Join(baseDir, "data")
}

// ProjectRoot 回傳指定專案的根路徑（§17.2）。
func ProjectRoot(baseDir, projectID string) string {
	return filepath.Join(baseDir, "data", "projects", projectID)
}

// PersonaRoot 回傳指定 persona 的根路徑（§17.3）。
func PersonaRoot(baseDir, personaID string) string {
	return filepath.Join(baseDir, "data", "personas", personaID)
}

// ──────────────────────────────────────────────
// 專案目錄結構（§17.2 完整樹狀）
// ──────────────────────────────────────────────

// projectDirs 定義專案必要子目錄。
var projectDirs = []string{
	// 記憶管線
	"memory",
	// DAG 執行紀錄
	"dag_runs",
	// 執行時暫存
	"runtime",
	"runtime/temp_sessions",
	"runtime/action_results",
	"runtime/crash_recovery",
	// 控制信任
	"controlled_trust",
	"controlled_trust/draft_sandbox_runs",
	"controlled_trust/pending_digest",
	// Execution Hook
	"execution_hook_runs",
	"execution_hook_runs/new_subagent_candidates",
	// 視覺學習
	"visual_learning",
	"visual_learning/learning_runs",
	"visual_learning/dictionaries",
	"visual_learning/pending",
	"visual_learning/review",
	"visual_learning/export",
	// 子代理
	"subagents",
	"subagents/callable",
	"subagents/candidates",
	"subagents/archived",
	"subagents/lineage",
	// 來源信任
	"source_trust",
	// Review
	"review",
}

// projectFiles 定義專案必要初始檔案（空 JSON 或 JSONL）。
var projectFiles = map[string]string{
	"runtime/purge_manifest.json":              "{}",
	"controlled_trust/trusted_session_scope.json":   "{}",
	"controlled_trust/contextual_risk_overrides.json": "{}",
	"controlled_trust/device_trust_profiles.json":     "{}",
	"controlled_trust/controlled_trust_log.jsonl":     "",
	"execution_hook_runs/encrypted_hook_evidence.jsonl":     "",
	"execution_hook_runs/hook_summary_index.json":          "{}",
	"execution_hook_runs/pending_tag_patch.json":            "{}",
	"execution_hook_runs/tool_registry_patch_proposal.json": "{}",
	"execution_hook_runs/prompt_repair_proposal.json":       "{}",
	"execution_hook_runs/hook_proposal_hash_chain.jsonl":    "",
	"memory/memory_manifest.json": "{}",
	"source_trust/source_trust_reviews.jsonl":        "",
	"source_trust/project_source_allowlist.json":     "{}",
	"source_trust/source_trust_log.jsonl":            "",
	"source_trust/source_trust_cache.json":           "{}",
	"source_trust/source_trust_ui_preferences.json":  "{}",
	"review/review_inbox.json":          "[]",
	"review/review_archive.json":        "[]",
	"review/review_decision_log.jsonl":  "",
}

// EnsureProjectLayout 建立專案目錄結構（§17.2）。
// 如果目錄或檔案已存在，則不覆寫。
func EnsureProjectLayout(baseDir, projectID string) error {
	if err := ValidatePath(projectID, ""); err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	root := ProjectRoot(baseDir, projectID)

	// 建立所有子目錄
	for _, dir := range projectDirs {
		fullPath := filepath.Join(root, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("建立專案目錄 %s 失敗: %w", dir, err)
		}
	}

	// 建立初始檔案（僅在不存在時）
	for relPath, content := range projectFiles {
		fullPath := filepath.Join(root, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
				return fmt.Errorf("建立專案檔案 %s 失敗: %w", relPath, err)
			}
		}
	}

	return nil
}

// ──────────────────────────────────────────────
// Persona 目錄結構（§17.3）
// ──────────────────────────────────────────────

// personaDirs 定義 persona 必要子目錄。
var personaDirs = []string{
	"avatar",
}

// EnsurePersonaLayout 建立 persona 目錄結構（§17.3）。
func EnsurePersonaLayout(baseDir, personaID string) error {
	if err := ValidatePath(personaID, ""); err != nil {
		return fmt.Errorf("invalid persona ID: %w", err)
	}

	root := PersonaRoot(baseDir, personaID)

	for _, dir := range personaDirs {
		fullPath := filepath.Join(root, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("建立 persona 目錄 %s 失敗: %w", dir, err)
		}
	}

	return nil
}

// ──────────────────────────────────────────────
// 目錄存在性檢查
// ──────────────────────────────────────────────

// ProjectExists 檢查專案目錄是否存在。
func ProjectExists(baseDir, projectID string) bool {
	root := ProjectRoot(baseDir, projectID)
	info, err := os.Stat(root)
	return err == nil && info.IsDir()
}

// PersonaExists 檢查 persona 目錄是否存在。
func PersonaExists(baseDir, personaID string) bool {
	root := PersonaRoot(baseDir, personaID)
	info, err := os.Stat(root)
	return err == nil && info.IsDir()
}
