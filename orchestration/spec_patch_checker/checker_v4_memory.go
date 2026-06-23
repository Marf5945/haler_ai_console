// checker_v4_memory.go — v4.0 Delegated Memory Model Guard 的 24 條 check rule。
//
// 架構：
//   - 純 forbidden keyword rule（16 條）→ strings.Contains
//   - 條件組合 rule（9 條）→ encoding/json Unmarshal 到 contract struct
//   - 通用 guard（2 條）→ CheckNoAbsolutePathInPayload + CheckSchemaVersion
//     （已定義在 v4_payload_contract.go）
//
// JSON parse 失敗 → 回傳 parse error（不靜默 pass）。
// error message 格式：specPatchChecker[v4.0]: <描述>
package spec_patch_checker

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ══════════════════════════════════════════════════════════════════════════════
// Rule 1: talk_full 寫入必須伴隨 memory_ops 記錄
// ══════════════════════════════════════════════════════════════════════════════

// CheckTalkFullWriteHasMemoryOps [Rule 1] 確認對 talk_full.md 的寫入有對應的
// memory_ops.jsonl 記錄。條件組合 rule，使用 JSON parse。
func CheckTalkFullWriteHasMemoryOps(payload string) error {
	var evt MemoryWriteEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 1 JSON parse failed: %w", err)
	}
	if evt.EventType == "memory_write_event" && !evt.MemoryOpsWritten {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 1 violated — talk_full write (target=%s) must have memory_ops_written=true",
			evt.Target)
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 2: memory_ops.jsonl 必須是 append-only
// ══════════════════════════════════════════════════════════════════════════════

// CheckMemoryOpsAppendOnly [Rule 2] 禁止 memory_ops.jsonl 被 overwrite 或 truncate。
// Forbidden keyword rule。
func CheckMemoryOpsAppendOnly(payload string) error {
	if strings.Contains(payload, "memory_ops") {
		if strings.Contains(payload, `"operation":"overwrite"`) ||
			strings.Contains(payload, `"operation":"truncate"`) {
			return fmt.Errorf(
				"specPatchChecker[v4.0]: Rule 2 violated — memory_ops.jsonl must be append-only; found overwrite/truncate")
		}
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 3: main agent 不得直接儲存 talk content
// ══════════════════════════════════════════════════════════════════════════════

// CheckMainAgentDoesNotStoreTalkContent [Rule 3] 條件組合 rule。
// main 的 stored_items 不得包含 talk/conversation/chat 相關項目。
func CheckMainAgentDoesNotStoreTalkContent(payload string) error {
	var evt MainStorageManifest
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 3 JSON parse failed: %w", err)
	}
	forbidden := []string{"talk_full", "talk_content", "conversation_raw", "chat_history"}
	for _, item := range evt.StoredItems {
		for _, f := range forbidden {
			if strings.Contains(item, f) {
				return fmt.Errorf(
					"specPatchChecker[v4.0]: Rule 3 violated — main agent must not store talk content: found %q", item)
			}
		}
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 4: main agent 不得直接儲存 CLI 輸出
// ══════════════════════════════════════════════════════════════════════════════

// CheckMainAgentDoesNotStoreCLIOutput [Rule 4] Forbidden keyword rule。
func CheckMainAgentDoesNotStoreCLIOutput(payload string) error {
	if strings.Contains(payload, `"main_storage_manifest"`) {
		if strings.Contains(payload, "cli_output") ||
			strings.Contains(payload, "raw_cli_response") ||
			strings.Contains(payload, "cli_full_log") {
			return fmt.Errorf(
				"specPatchChecker[v4.0]: Rule 4 violated — main agent must not store CLI output directly")
		}
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 5: sub 不得存取完整 tool database
// ══════════════════════════════════════════════════════════════════════════════

// CheckSubDoesNotAccessFullToolDB [Rule 5] Forbidden keyword rule。
func CheckSubDoesNotAccessFullToolDB(payload string) error {
	if strings.Contains(payload, `"access_scope":"full_tool_database"`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 5 violated — sub must not access full tool database; must use scoped access")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 6: sub 不得改動 global tool registry
// ══════════════════════════════════════════════════════════════════════════════

// CheckSubDoesNotMutateGlobalRegistry [Rule 6] Forbidden keyword rule。
func CheckSubDoesNotMutateGlobalRegistry(payload string) error {
	if strings.Contains(payload, `"mutation_target":"global_tool_registry"`) ||
		strings.Contains(payload, `"mutation_target":"global_risk_policy"`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 6 violated — sub must not mutate global tool registry or risk policy")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 7: sub 的未記錄工具使用必須回報 main
// ══════════════════════════════════════════════════════════════════════════════

// CheckSubReportsUnrecordedToolUse [Rule 7] 條件組合 rule。
func CheckSubReportsUnrecordedToolUse(payload string) error {
	var evt SubToolAccessEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 7 JSON parse failed: %w", err)
	}
	if evt.ToolNotInHistory && !evt.ReportedToMain {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 7 violated — sub %s used unrecorded tool but did not report to main",
			evt.SubID)
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 8: 摘要必須執行 write redaction
// ══════════════════════════════════════════════════════════════════════════════

// CheckSummarizationWriteRedaction [Rule 8] Forbidden keyword rule。
func CheckSummarizationWriteRedaction(payload string) error {
	if strings.Contains(payload, `"summarization_event"`) &&
		strings.Contains(payload, `"write_redaction_applied":false`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 8 violated — summarization must apply write redaction before disk write")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 9: 摘要不得包含敏感 key（secrets）
// ══════════════════════════════════════════════════════════════════════════════

// CheckSummarizationNoSecrets [Rule 9] Forbidden keyword rule。
func CheckSummarizationNoSecrets(payload string) error {
	if strings.Contains(payload, `"summarization_output"`) {
		forbidden := []string{
			"api_key", "password", "secret", "token",
			"credential", "private_key", "local_path",
		}
		for _, f := range forbidden {
			if strings.Contains(payload, f) {
				return fmt.Errorf(
					"specPatchChecker[v4.0]: Rule 9 violated — summarization output must not contain sensitive key: %q", f)
			}
		}
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 10: 摘要不得包含 system prompt 字元
// ══════════════════════════════════════════════════════════════════════════════

// CheckSummarizationNoSystemPrompt [Rule 10] Forbidden keyword rule。
func CheckSummarizationNoSystemPrompt(payload string) error {
	if strings.Contains(payload, `"summarization_event"`) &&
		strings.Contains(payload, `"includes_system_prompt_chars":true`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 10 violated — summarization must not include system prompt characters")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 11: 摘要失敗時不得靜默送出舊 context
// ══════════════════════════════════════════════════════════════════════════════

// CheckSummarizationFailNoSilentSend [Rule 11] 條件組合 rule。
func CheckSummarizationFailNoSilentSend(payload string) error {
	var evt SummarizationEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 11 JSON parse failed: %w", err)
	}
	if evt.SummarizationFailed && evt.OldContextSentSilently {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 11 violated — summarization failed but old context was sent silently")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 12: tool_history.jsonl 必須是 append-only
// ══════════════════════════════════════════════════════════════════════════════

// CheckToolHistoryAppendOnly [Rule 12] Forbidden keyword rule。
func CheckToolHistoryAppendOnly(payload string) error {
	if strings.Contains(payload, "tool_history") {
		if strings.Contains(payload, `"operation":"overwrite"`) ||
			strings.Contains(payload, `"operation":"truncate"`) {
			return fmt.Errorf(
				"specPatchChecker[v4.0]: Rule 12 violated — tool_history.jsonl must be append-only")
		}
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 13: DAG resume guard 不得使用 main_memory_hash
// ══════════════════════════════════════════════════════════════════════════════

// CheckNoMainMemoryHashAsResumeGuard [Rule 13] 條件組合 rule。
func CheckNoMainMemoryHashAsResumeGuard(payload string) error {
	var evt DAGResumeConfig
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 13 JSON parse failed: %w", err)
	}
	if evt.ResumeGuard == "main_memory_hash" {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 13 violated — DAG resume guard must use sub_memory_hash, not main_memory_hash")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 14: delegation_log.jsonl 必須是 append-only
// ══════════════════════════════════════════════════════════════════════════════

// CheckDelegationLogAppendOnly [Rule 14] Forbidden keyword rule。
func CheckDelegationLogAppendOnly(payload string) error {
	if strings.Contains(payload, "delegation_log") {
		if strings.Contains(payload, `"operation":"overwrite"`) ||
			strings.Contains(payload, `"operation":"truncate"`) {
			return fmt.Errorf(
				"specPatchChecker[v4.0]: Rule 14 violated — delegation_log.jsonl must be append-only")
		}
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 15: 摘要輸出不得寫入 talk_full.md
// ══════════════════════════════════════════════════════════════════════════════

// CheckSummarizationNotWriteToTalkFull [Rule 15] 條件組合 rule。
func CheckSummarizationNotWriteToTalkFull(payload string) error {
	var evt SummarizationEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 15 JSON parse failed: %w", err)
	}
	if evt.EventType == "summarization_event" && evt.OutputTarget == "memory/talk_full.md" {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 15 violated — summarization output must not write to talk_full.md; use summaries.md")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 16: 摘要中不得送訊息給 CLI
// ══════════════════════════════════════════════════════════════════════════════

// CheckSummarizationNoMessageToCLI [Rule 16] 條件組合 rule。
func CheckSummarizationNoMessageToCLI(payload string) error {
	var evt SummarizationEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 16 JSON parse failed: %w", err)
	}
	if evt.SummarizationActive && evt.MessageSentToCLI {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 16 violated — must not send messages to CLI during active summarization")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 17: 摘要不得摘要 system prompt
// ══════════════════════════════════════════════════════════════════════════════

// CheckSummarizationNoSystemPromptSummarized [Rule 17] Forbidden keyword rule。
func CheckSummarizationNoSystemPromptSummarized(payload string) error {
	if strings.Contains(payload, `"summarization_event"`) &&
		strings.Contains(payload, `"system_prompt_summarized":true`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 17 violated — system prompt must not be summarized")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 18: 摘要模型不得 hardcoded
// ══════════════════════════════════════════════════════════════════════════════

// CheckModelNotHardcoded [Rule 18] Forbidden keyword rule。
func CheckModelNotHardcoded(payload string) error {
	if strings.Contains(payload, `"model_selection"`) &&
		strings.Contains(payload, `"hardcoded_model":true`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 18 violated — summarization model must not be hardcoded; must be user-configurable")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 19: 模型移除後必須觸發使用者重新選擇
// ══════════════════════════════════════════════════════════════════════════════

// CheckModelRemovedTriggersReselection [Rule 19] 條件組合 rule。
func CheckModelRemovedTriggersReselection(payload string) error {
	var evt ModelSelectionEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 19 JSON parse failed: %w", err)
	}
	if evt.ModelRemoved && !evt.UserReselectionTriggered {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 19 violated — model removed but user reselection was not triggered")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 20: 模型不可用時不得靜默 fallback
// ══════════════════════════════════════════════════════════════════════════════

// CheckModelUnavailableNoSilentFallback [Rule 20] 條件組合 rule。
func CheckModelUnavailableNoSilentFallback(payload string) error {
	var evt ModelSelectionEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 20 JSON parse failed: %w", err)
	}
	if evt.ModelUnavailable && evt.SilentFallback {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 20 violated — model unavailable but silently fell back to another model")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 21: action routing 必須使用 program tag match
// ══════════════════════════════════════════════════════════════════════════════

// CheckActionRoutingUsesProgramTag [Rule 21] Forbidden keyword rule。
func CheckActionRoutingUsesProgramTag(payload string) error {
	if strings.Contains(payload, `"routing_decision"`) &&
		strings.Contains(payload, `"routing_method":"llm_only"`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 21 violated — action routing must use program_tag_match, not llm_only")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 22: LLM prompt 不得注入完整 tool database
// ══════════════════════════════════════════════════════════════════════════════

// CheckNoToolDBInjectedIntoPrompt [Rule 22] 條件組合 rule。
func CheckNoToolDBInjectedIntoPrompt(payload string) error {
	var evt PromptComposition
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("specPatchChecker[v4.0]: Rule 22 JSON parse failed: %w", err)
	}
	if evt.ToolDatabaseInPrompt {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 22 violated — full tool database must not be injected into LLM prompt")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 23: 安全相關 log 不得被 reorganize/compact/delegate
// ══════════════════════════════════════════════════════════════════════════════

// CheckSecurityLogsNotReorganized [Rule 23] Forbidden keyword rule。
func CheckSecurityLogsNotReorganized(payload string) error {
	if strings.Contains(payload, `"log_operation"`) {
		forbidden := []string{
			`"operation":"reorganized"`,
			`"operation":"compacted"`,
			`"operation":"delegated"`,
			`"operation":"truncated"`,
		}
		for _, f := range forbidden {
			if strings.Contains(payload, f) {
				return fmt.Errorf(
					"specPatchChecker[v4.0]: Rule 23 violated — security logs must not be reorganized/compacted/delegated/truncated")
			}
		}
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 24: sub 建立必須經使用者確認
// ══════════════════════════════════════════════════════════════════════════════

// CheckSubNotCreatedWithoutConfirmation [Rule 24] Forbidden keyword rule。
func CheckSubNotCreatedWithoutConfirmation(payload string) error {
	if strings.Contains(payload, `"sub_creation"`) &&
		strings.Contains(payload, `"auto_created":true`) &&
		strings.Contains(payload, `"user_confirmed":false`) {
		return fmt.Errorf(
			"specPatchChecker[v4.0]: Rule 24 violated — sub agent must not be auto-created without user confirmation")
	}
	return nil
}
