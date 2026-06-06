// v4_payload_contract.go — v4.0 Delegated Memory Model Guard 的 Payload Contract。
// 定義所有 event type struct，作為 producer 與 checker 的共同 contract。
// 安全工程師 review 凍結後方可進入 Phase 1。
//
// 設計原則：
//   - 零第三方 dependency
//   - 所有 target 欄位使用 project-relative logical path（禁止絕對路徑）
//   - schema_version 欄位用於版本控制，unknown version → 明確 error
package spec_patch_checker

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ═══════════════════════════��═════════════════════════��════════════════════════
// Contract Version 常數
// ══════════════════════════════════════��═══════════════════════════════════���═══

// ContractVersion 是所有 v4.0 memory guard event 的 schema 版本。
// 版本升級時 bump 為 "v4.0-memory-guard-2" 以此類推。
const ContractVersion = "v4.0-memory-guard-1"

// ═════════════════════════════════════════��════════════════════════════════════
// Event Type Structs（13 個）
// ═══════════════════════════════════���══════════════════════════════════════════

// MemoryWriteEvent — 任何對 talk_full.md 的寫入操作。
// Producer: data/memory/pipeline.go (AppendTalkEntry, Rotate, DeleteSentences)
// Consumer: CheckTalkFullWriteHasMemoryOps, CheckMemoryOpsAppendOnly
type MemoryWriteEvent struct {
	SchemaVersion      string `json:"schema_version"`
	EventType          string `json:"event_type"`            // "memory_write_event"
	Target             string `json:"target"`                // "memory/talk_full.md"
	Operation          string `json:"operation"`             // "append" | "rotate" | "delete_sentences"
	MemoryOpsWritten   bool   `json:"memory_ops_written"`
	TalkFullHashBefore string `json:"talk_full_hash_before"` // 寫入前 SHA-256
	TalkFullHashAfter  string `json:"talk_full_hash_after"`  // 寫入後 SHA-256
	PrevMemoryOpHash   string `json:"prev_memory_op_hash"`   // 前一筆 entry 的 memory_op_hash
	MemoryOpHash       string `json:"memory_op_hash"`        // 當前 entry 全欄位 SHA-256
	Timestamp          string `json:"timestamp"`
}

// FileOperation — 對 append-only 檔案的操作記錄。
// Producer: 任何寫入 memory_ops.jsonl / tool_history.jsonl / delegation_log.jsonl 的函式
// Consumer: CheckMemoryOpsAppendOnly, CheckToolHistoryAppendOnly, CheckDelegationLogAppendOnly
type FileOperation struct {
	SchemaVersion string `json:"schema_version"`
	EventType     string `json:"event_type"` // "file_operation"
	Target        string `json:"target"`     // "memory/memory_ops.jsonl" | "memory/tool_history.jsonl" | ...
	Operation     string `json:"operation"`  // "append" | "overwrite" | "truncate"
	Timestamp     string `json:"timestamp"`
}

// MainStorageManifest — main agent 的 storage 宣告。
// Producer: orchestration/delegation/registry.go
// Consumer: CheckMainAgentDoesNotStoreTalkContent, CheckMainAgentDoesNotStoreCLIOutput
type MainStorageManifest struct {
	SchemaVersion string   `json:"schema_version"`
	EventType     string   `json:"event_type"`   // "main_storage_manifest"
	StoredItems   []string `json:"stored_items"` // 應只含 routing/delegation/registry 類
}

// SubToolAccessEvent — sub 存取工具的事件。
// Producer: orchestration/delegation/handoff.go
// Consumer: CheckSubDoesNotAccessFullToolDB, CheckSubDoesNotMutateGlobalRegistry, CheckSubReportsUnrecordedToolUse
type SubToolAccessEvent struct {
	SchemaVersion    string `json:"schema_version"`
	EventType        string `json:"event_type"`         // "sub_tool_access"
	SubID            string `json:"sub_id"`
	AccessScope      string `json:"access_scope"`       // "scoped" | "full_tool_database"
	MutationTarget   string `json:"mutation_target"`    // "" | "global_tool_registry" | "global_risk_policy"
	ToolNotInHistory bool   `json:"tool_not_in_history"`
	ReportedToMain   bool   `json:"reported_to_main"`
}

// SummarizationEvent — 摘要流程事件。
// Producer: conversation/summarizer.go
// Consumer: Rules 8-11, 15-17
type SummarizationEvent struct {
	SchemaVersion          string `json:"schema_version"`
	EventType              string `json:"event_type"`                    // "summarization_event"
	WriteRedactionApplied  bool   `json:"write_redaction_applied"`
	IncludesSystemPrompt   bool   `json:"includes_system_prompt_chars"`
	SystemPromptSummarized bool   `json:"system_prompt_summarized"`
	SummarizationActive    bool   `json:"summarization_active"`
	MessageSentToCLI       bool   `json:"message_sent_to_cli"`
	SummarizationFailed    bool   `json:"summarization_failed"`
	OldContextSentSilently bool   `json:"old_context_sent_silently"`
	OutputTarget           string `json:"output_target"`                 // "memory/summaries.md" 正確；"memory/talk_full.md" 違規
	OutputOperation        string `json:"output_operation"`              // "append" | "overwrite"
}

// SummarizationOutput — 摘要產出內容的 metadata。
// Producer: conversation/summarizer.go
// Consumer: CheckSummarizationNoSecrets
type SummarizationOutput struct {
	SchemaVersion string   `json:"schema_version"`
	EventType     string   `json:"event_type"`   // "summarization_output"
	ContentKeys   []string `json:"content_keys"` // 欄位名稱列表，用於偵測敏感 key
}

// DAGResumeConfig — DAG resume guard 設定。
// Producer: orchestration/dag/resume_guard.go
// Consumer: CheckNoMainMemoryHashAsResumeGuard
type DAGResumeConfig struct {
	SchemaVersion string `json:"schema_version"`
	EventType     string `json:"event_type"`   // "dag_resume_config"
	ResumeGuard   string `json:"resume_guard"` // "sub_memory_hash"（正確）| "main_memory_hash"（違規）
}

// RoutingDecision — LLM action routing 決策。
// Producer: orchestration/skill_step/router.go
// Consumer: CheckActionRoutingUsesProgramTag
type RoutingDecision struct {
	SchemaVersion   string `json:"schema_version"`
	EventType       string `json:"event_type"`        // "routing_decision"
	RoutingMethod   string `json:"routing_method"`    // "program_tag_match" | "llm_only"
	ProgramTagMatch bool   `json:"program_tag_match"`
}

// PromptComposition — LLM prompt 組成檢查。
// Producer: orchestration/cli_manager/adapter.go
// Consumer: CheckNoToolDBInjectedIntoPrompt
type PromptComposition struct {
	SchemaVersion        string `json:"schema_version"`
	EventType            string `json:"event_type"`            // "prompt_composition"
	SubRegistryInPrompt  bool   `json:"sub_registry_in_prompt"`
	ToolDatabaseInPrompt bool   `json:"tool_database_in_prompt"`
}

// ModelSelectionEvent — 摘要模型選擇事件。
// Producer: conversation/model_list.go
// Consumer: Rules 18-20
type ModelSelectionEvent struct {
	SchemaVersion            string `json:"schema_version"`
	EventType                string `json:"event_type"`                  // "model_selection"
	ModelSource              string `json:"model_source"`                // "configured" | "constant"（後者違規）
	HardcodedModel           bool   `json:"hardcoded_model"`
	ModelRemoved             bool   `json:"model_removed"`
	UserReselectionTriggered bool   `json:"user_reselection_triggered"`
	ModelUnavailable         bool   `json:"model_unavailable"`
	SilentFallback           bool   `json:"silent_fallback"`
}

// LogOperation — 對安全相關 log 的操作。
// Producer: audit_log/audit_log.go, orchestration/*/
// Consumer: CheckSecurityLogsNotReorganized
type LogOperation struct {
	SchemaVersion string `json:"schema_version"`
	EventType     string `json:"event_type"` // "log_operation"
	Target        string `json:"target"`     // "security_log" | "audit_log" | "risk_log" | "review_log"
	Operation     string `json:"operation"`  // "append" | "reorganized" | "compacted" | "delegated" | "truncated"
}

// SubCreationEvent — sub agent 建立事件。
// Producer: orchestration/delegation/lifecycle.go
// Consumer: CheckSubNotCreatedWithoutConfirmation
type SubCreationEvent struct {
	SchemaVersion string `json:"schema_version"`
	EventType     string `json:"event_type"`     // "sub_creation"
	SubID         string `json:"sub_id"`
	AutoCreated   bool   `json:"auto_created"`
	UserConfirmed bool   `json:"user_confirmed"`
}

// ═══════════════════════════════��═════════════════════════════��════════════════
// 通用 Guard 函式
// ══════════════════════════════════════════════════════════════════════════════

// CheckNoAbsolutePathInPayload 偵測 payload 中的絕對路徑。
// 所有 event 的 target/path 欄位應使用 project-relative logical path。
// 違反時回傳描述性 error。
func CheckNoAbsolutePathInPayload(payload string) error {
	// 常見絕對路徑前綴
	absolutePrefixes := []string{
		"/Users/",
		"/home/",
		`C:\Users\`,
		`C:\\Users\\`,
		"/tmp/",
		"/var/",
	}
	for _, prefix := range absolutePrefixes {
		if strings.Contains(payload, prefix) {
			return fmt.Errorf(
				"specPatchChecker[v4.0]: payload contains absolute path (found %q); "+
					"use project-relative logical path instead", prefix)
		}
	}
	return nil
}

// CheckSchemaVersion 驗證 payload 的 schema_version 欄位。
// unknown version → 明確 error；空白 → error。
func CheckSchemaVersion(payload string) error {
	// 嘗試解析 schema_version 欄位
	var partial struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal([]byte(payload), &partial); err != nil {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: failed to parse schema_version from payload: %w", err)
	}

	if partial.SchemaVersion == "" {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: schema_version is empty; expected %q", ContractVersion)
	}

	if partial.SchemaVersion != ContractVersion {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: unknown schema_version %q; expected %q",
			partial.SchemaVersion, ContractVersion)
	}

	return nil
}
