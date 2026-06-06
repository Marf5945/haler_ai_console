// checker_v4_memory_test.go — v4.0 Delegated Memory Model Guard 單元測試。
// 命名慣例：TestV4Memory_Rule{N}_{Scenario}
// 條件組合 Rules 各 ≥ 3 case（含 JSON parse error case）
// 通用 guard 4 個 case
package spec_patch_checker

import (
	"encoding/json"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════════════
// 通用 Guard 測試
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_SchemaVersion_Valid(t *testing.T) {
	payload := `{"schema_version":"v4.0-memory-guard-1","event_type":"test"}`
	if err := CheckSchemaVersion(payload); err != nil {
		t.Errorf("valid schema version should pass: %v", err)
	}
}

func TestV4Memory_SchemaVersion_Unknown(t *testing.T) {
	payload := `{"schema_version":"v3.0-old","event_type":"test"}`
	if err := CheckSchemaVersion(payload); err == nil {
		t.Error("unknown schema version should fail")
	}
}

func TestV4Memory_SchemaVersion_Empty(t *testing.T) {
	payload := `{"schema_version":"","event_type":"test"}`
	if err := CheckSchemaVersion(payload); err == nil {
		t.Error("empty schema version should fail")
	}
}

func TestV4Memory_SchemaVersion_ParseError(t *testing.T) {
	if err := CheckSchemaVersion("not json"); err == nil {
		t.Error("invalid JSON should return parse error")
	}
}

func TestV4Memory_AbsolutePath_Clean(t *testing.T) {
	payload := `{"target":"memory/talk_full.md"}`
	if err := CheckNoAbsolutePathInPayload(payload); err != nil {
		t.Errorf("relative path should pass: %v", err)
	}
}

func TestV4Memory_AbsolutePath_Users(t *testing.T) {
	payload := `{"target":"/Users/tester/memory/talk_full.md"}`
	if err := CheckNoAbsolutePathInPayload(payload); err == nil {
		t.Error("/Users/ path should fail")
	}
}

func TestV4Memory_AbsolutePath_Home(t *testing.T) {
	payload := `{"target":"/home/user/data"}`
	if err := CheckNoAbsolutePathInPayload(payload); err == nil {
		t.Error("/home/ path should fail")
	}
}

func TestV4Memory_AbsolutePath_Windows(t *testing.T) {
	payload := `{"target":"C:\\Users\\test\\data"}`
	if err := CheckNoAbsolutePathInPayload(payload); err == nil {
		t.Error("C:\\Users\\ path should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 1: talk_full write must have memory_ops
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule1_Pass(t *testing.T) {
	evt := MemoryWriteEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "memory_write_event",
		Target:           "memory/talk_full.md",
		Operation:        "append",
		MemoryOpsWritten: true,
	}
	data, _ := json.Marshal(evt)
	if err := CheckTalkFullWriteHasMemoryOps(string(data)); err != nil {
		t.Errorf("should pass when memory_ops_written=true: %v", err)
	}
}

func TestV4Memory_Rule1_Fail(t *testing.T) {
	evt := MemoryWriteEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "memory_write_event",
		Target:           "memory/talk_full.md",
		Operation:        "append",
		MemoryOpsWritten: false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckTalkFullWriteHasMemoryOps(string(data)); err == nil {
		t.Error("should fail when memory_ops_written=false")
	}
}

func TestV4Memory_Rule1_ParseError(t *testing.T) {
	if err := CheckTalkFullWriteHasMemoryOps("invalid json"); err == nil {
		t.Error("should fail on invalid JSON")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 2: memory_ops must be append-only
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule2_AppendPass(t *testing.T) {
	payload := `{"target":"memory/memory_ops.jsonl","operation":"append"}`
	if err := CheckMemoryOpsAppendOnly(payload); err != nil {
		t.Errorf("append should pass: %v", err)
	}
}

func TestV4Memory_Rule2_OverwriteFail(t *testing.T) {
	payload := `{"target":"memory_ops","operation":"overwrite"}`
	if err := CheckMemoryOpsAppendOnly(payload); err == nil {
		t.Error("overwrite should fail")
	}
}

func TestV4Memory_Rule2_TruncateFail(t *testing.T) {
	payload := `{"target":"memory_ops","operation":"truncate"}`
	if err := CheckMemoryOpsAppendOnly(payload); err == nil {
		t.Error("truncate should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 3: main agent must not store talk content
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule3_Pass(t *testing.T) {
	evt := MainStorageManifest{
		SchemaVersion: ContractVersion,
		EventType:     "main_storage_manifest",
		StoredItems:   []string{"routing_table", "delegation_registry"},
	}
	data, _ := json.Marshal(evt)
	if err := CheckMainAgentDoesNotStoreTalkContent(string(data)); err != nil {
		t.Errorf("routing/delegation should pass: %v", err)
	}
}

func TestV4Memory_Rule3_Fail(t *testing.T) {
	evt := MainStorageManifest{
		SchemaVersion: ContractVersion,
		EventType:     "main_storage_manifest",
		StoredItems:   []string{"routing_table", "talk_full_cache"},
	}
	data, _ := json.Marshal(evt)
	if err := CheckMainAgentDoesNotStoreTalkContent(string(data)); err == nil {
		t.Error("talk_full in stored_items should fail")
	}
}

func TestV4Memory_Rule3_ParseError(t *testing.T) {
	if err := CheckMainAgentDoesNotStoreTalkContent("{bad json"); err == nil {
		t.Error("should fail on parse error")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 4: main agent must not store CLI output
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule4_Pass(t *testing.T) {
	payload := `{"event_type":"main_storage_manifest","stored_items":["routing"]}`
	if err := CheckMainAgentDoesNotStoreCLIOutput(payload); err != nil {
		t.Errorf("should pass without cli_output: %v", err)
	}
}

func TestV4Memory_Rule4_Fail(t *testing.T) {
	payload := `{"event_type":"main_storage_manifest","stored_items":["cli_output"]}`
	if err := CheckMainAgentDoesNotStoreCLIOutput(payload); err == nil {
		t.Error("cli_output should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 5: sub must not access full tool database
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule5_Pass(t *testing.T) {
	payload := `{"access_scope":"scoped"}`
	if err := CheckSubDoesNotAccessFullToolDB(payload); err != nil {
		t.Errorf("scoped should pass: %v", err)
	}
}

func TestV4Memory_Rule5_Fail(t *testing.T) {
	payload := `{"access_scope":"full_tool_database"}`
	if err := CheckSubDoesNotAccessFullToolDB(payload); err == nil {
		t.Error("full_tool_database should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 6: sub must not mutate global registry
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule6_Pass(t *testing.T) {
	payload := `{"mutation_target":""}`
	if err := CheckSubDoesNotMutateGlobalRegistry(payload); err != nil {
		t.Errorf("empty mutation should pass: %v", err)
	}
}

func TestV4Memory_Rule6_FailRegistry(t *testing.T) {
	payload := `{"mutation_target":"global_tool_registry"}`
	if err := CheckSubDoesNotMutateGlobalRegistry(payload); err == nil {
		t.Error("global_tool_registry mutation should fail")
	}
}

func TestV4Memory_Rule6_FailPolicy(t *testing.T) {
	payload := `{"mutation_target":"global_risk_policy"}`
	if err := CheckSubDoesNotMutateGlobalRegistry(payload); err == nil {
		t.Error("global_risk_policy mutation should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 7: sub unrecorded tool use must report to main
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule7_Pass_Recorded(t *testing.T) {
	evt := SubToolAccessEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "sub_tool_access",
		SubID:            "sub-1",
		ToolNotInHistory: false,
		ReportedToMain:   false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSubReportsUnrecordedToolUse(string(data)); err != nil {
		t.Errorf("recorded tool should pass: %v", err)
	}
}

func TestV4Memory_Rule7_Pass_Reported(t *testing.T) {
	evt := SubToolAccessEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "sub_tool_access",
		SubID:            "sub-1",
		ToolNotInHistory: true,
		ReportedToMain:   true,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSubReportsUnrecordedToolUse(string(data)); err != nil {
		t.Errorf("reported should pass: %v", err)
	}
}

func TestV4Memory_Rule7_Fail(t *testing.T) {
	evt := SubToolAccessEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "sub_tool_access",
		SubID:            "sub-1",
		ToolNotInHistory: true,
		ReportedToMain:   false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSubReportsUnrecordedToolUse(string(data)); err == nil {
		t.Error("unrecorded + not reported should fail")
	}
}

func TestV4Memory_Rule7_ParseError(t *testing.T) {
	if err := CheckSubReportsUnrecordedToolUse("bad"); err == nil {
		t.Error("invalid JSON should return error")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 8: summarization must apply write redaction
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule8_Pass(t *testing.T) {
	payload := `{"event_type":"summarization_event","write_redaction_applied":true}`
	if err := CheckSummarizationWriteRedaction(payload); err != nil {
		t.Errorf("redaction applied should pass: %v", err)
	}
}

func TestV4Memory_Rule8_Fail(t *testing.T) {
	payload := `{"event_type":"summarization_event","write_redaction_applied":false}`
	if err := CheckSummarizationWriteRedaction(payload); err == nil {
		t.Error("redaction not applied should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 9: summarization output must not contain secrets
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule9_Pass(t *testing.T) {
	payload := `{"event_type":"summarization_output","content_keys":["summary","tags"]}`
	if err := CheckSummarizationNoSecrets(payload); err != nil {
		t.Errorf("clean keys should pass: %v", err)
	}
}

func TestV4Memory_Rule9_Fail(t *testing.T) {
	payload := `{"event_type":"summarization_output","content_keys":["api_key","summary"]}`
	if err := CheckSummarizationNoSecrets(payload); err == nil {
		t.Error("api_key in output should fail")
	}
}

func TestV4Memory_Rule9_FailLocalPath(t *testing.T) {
	payload := `{"event_type":"summarization_output","content_keys":["local_path"]}`
	if err := CheckSummarizationNoSecrets(payload); err == nil {
		t.Error("local_path in output should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 10: summarization must not include system prompt
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule10_Pass(t *testing.T) {
	payload := `{"event_type":"summarization_event","includes_system_prompt_chars":false}`
	if err := CheckSummarizationNoSystemPrompt(payload); err != nil {
		t.Errorf("no system prompt should pass: %v", err)
	}
}

func TestV4Memory_Rule10_Fail(t *testing.T) {
	payload := `{"event_type":"summarization_event","includes_system_prompt_chars":true}`
	if err := CheckSummarizationNoSystemPrompt(payload); err == nil {
		t.Error("includes system prompt should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 11: summarization failed → no silent send
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule11_Pass_NotFailed(t *testing.T) {
	evt := SummarizationEvent{
		SchemaVersion:          ContractVersion,
		EventType:              "summarization_event",
		SummarizationFailed:    false,
		OldContextSentSilently: false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSummarizationFailNoSilentSend(string(data)); err != nil {
		t.Errorf("not failed should pass: %v", err)
	}
}

func TestV4Memory_Rule11_Pass_FailedButNotSent(t *testing.T) {
	evt := SummarizationEvent{
		SchemaVersion:          ContractVersion,
		EventType:              "summarization_event",
		SummarizationFailed:    true,
		OldContextSentSilently: false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSummarizationFailNoSilentSend(string(data)); err != nil {
		t.Errorf("failed but not silently sent should pass: %v", err)
	}
}

func TestV4Memory_Rule11_Fail(t *testing.T) {
	evt := SummarizationEvent{
		SchemaVersion:          ContractVersion,
		EventType:              "summarization_event",
		SummarizationFailed:    true,
		OldContextSentSilently: true,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSummarizationFailNoSilentSend(string(data)); err == nil {
		t.Error("failed + silent send should fail")
	}
}

func TestV4Memory_Rule11_ParseError(t *testing.T) {
	if err := CheckSummarizationFailNoSilentSend("{invalid"); err == nil {
		t.Error("parse error should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 12: tool_history must be append-only
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule12_Pass(t *testing.T) {
	payload := `{"target":"tool_history","operation":"append"}`
	if err := CheckToolHistoryAppendOnly(payload); err != nil {
		t.Errorf("append should pass: %v", err)
	}
}

func TestV4Memory_Rule12_Fail(t *testing.T) {
	payload := `{"target":"tool_history","operation":"overwrite"}`
	if err := CheckToolHistoryAppendOnly(payload); err == nil {
		t.Error("overwrite should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 13: DAG resume guard must not use main_memory_hash
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule13_Pass(t *testing.T) {
	evt := DAGResumeConfig{
		SchemaVersion: ContractVersion,
		EventType:     "dag_resume_config",
		ResumeGuard:   "sub_memory_hash",
	}
	data, _ := json.Marshal(evt)
	if err := CheckNoMainMemoryHashAsResumeGuard(string(data)); err != nil {
		t.Errorf("sub_memory_hash should pass: %v", err)
	}
}

func TestV4Memory_Rule13_Fail(t *testing.T) {
	evt := DAGResumeConfig{
		SchemaVersion: ContractVersion,
		EventType:     "dag_resume_config",
		ResumeGuard:   "main_memory_hash",
	}
	data, _ := json.Marshal(evt)
	if err := CheckNoMainMemoryHashAsResumeGuard(string(data)); err == nil {
		t.Error("main_memory_hash should fail")
	}
}

func TestV4Memory_Rule13_ParseError(t *testing.T) {
	if err := CheckNoMainMemoryHashAsResumeGuard("not json"); err == nil {
		t.Error("parse error should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 14: delegation_log must be append-only
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule14_Pass(t *testing.T) {
	payload := `{"target":"delegation_log","operation":"append"}`
	if err := CheckDelegationLogAppendOnly(payload); err != nil {
		t.Errorf("append should pass: %v", err)
	}
}

func TestV4Memory_Rule14_Fail(t *testing.T) {
	payload := `{"target":"delegation_log","operation":"truncate"}`
	if err := CheckDelegationLogAppendOnly(payload); err == nil {
		t.Error("truncate should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 15: summarization must not write to talk_full.md
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule15_Pass(t *testing.T) {
	evt := SummarizationEvent{
		SchemaVersion: ContractVersion,
		EventType:     "summarization_event",
		OutputTarget:  "memory/summaries.md",
	}
	data, _ := json.Marshal(evt)
	if err := CheckSummarizationNotWriteToTalkFull(string(data)); err != nil {
		t.Errorf("summaries.md should pass: %v", err)
	}
}

func TestV4Memory_Rule15_Fail(t *testing.T) {
	evt := SummarizationEvent{
		SchemaVersion: ContractVersion,
		EventType:     "summarization_event",
		OutputTarget:  "memory/talk_full.md",
	}
	data, _ := json.Marshal(evt)
	if err := CheckSummarizationNotWriteToTalkFull(string(data)); err == nil {
		t.Error("talk_full.md as output should fail")
	}
}

func TestV4Memory_Rule15_ParseError(t *testing.T) {
	if err := CheckSummarizationNotWriteToTalkFull("xxx"); err == nil {
		t.Error("parse error should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 16: no message to CLI during summarization
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule16_Pass(t *testing.T) {
	evt := SummarizationEvent{
		SchemaVersion:       ContractVersion,
		EventType:           "summarization_event",
		SummarizationActive: true,
		MessageSentToCLI:    false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSummarizationNoMessageToCLI(string(data)); err != nil {
		t.Errorf("no message should pass: %v", err)
	}
}

func TestV4Memory_Rule16_Fail(t *testing.T) {
	evt := SummarizationEvent{
		SchemaVersion:       ContractVersion,
		EventType:           "summarization_event",
		SummarizationActive: true,
		MessageSentToCLI:    true,
	}
	data, _ := json.Marshal(evt)
	if err := CheckSummarizationNoMessageToCLI(string(data)); err == nil {
		t.Error("message during summarization should fail")
	}
}

func TestV4Memory_Rule16_ParseError(t *testing.T) {
	if err := CheckSummarizationNoMessageToCLI("bad"); err == nil {
		t.Error("parse error should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 17: system prompt must not be summarized
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule17_Pass(t *testing.T) {
	payload := `{"event_type":"summarization_event","system_prompt_summarized":false}`
	if err := CheckSummarizationNoSystemPromptSummarized(payload); err != nil {
		t.Errorf("not summarized should pass: %v", err)
	}
}

func TestV4Memory_Rule17_Fail(t *testing.T) {
	payload := `{"event_type":"summarization_event","system_prompt_summarized":true}`
	if err := CheckSummarizationNoSystemPromptSummarized(payload); err == nil {
		t.Error("summarized system prompt should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 18: model must not be hardcoded
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule18_Pass(t *testing.T) {
	payload := `{"event_type":"model_selection","hardcoded_model":false}`
	if err := CheckModelNotHardcoded(payload); err != nil {
		t.Errorf("not hardcoded should pass: %v", err)
	}
}

func TestV4Memory_Rule18_Fail(t *testing.T) {
	payload := `{"event_type":"model_selection","hardcoded_model":true}`
	if err := CheckModelNotHardcoded(payload); err == nil {
		t.Error("hardcoded model should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 19: model removed → must trigger reselection
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule19_Pass(t *testing.T) {
	evt := ModelSelectionEvent{
		SchemaVersion:            ContractVersion,
		EventType:                "model_selection",
		ModelRemoved:             true,
		UserReselectionTriggered: true,
	}
	data, _ := json.Marshal(evt)
	if err := CheckModelRemovedTriggersReselection(string(data)); err != nil {
		t.Errorf("reselection triggered should pass: %v", err)
	}
}

func TestV4Memory_Rule19_Fail(t *testing.T) {
	evt := ModelSelectionEvent{
		SchemaVersion:            ContractVersion,
		EventType:                "model_selection",
		ModelRemoved:             true,
		UserReselectionTriggered: false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckModelRemovedTriggersReselection(string(data)); err == nil {
		t.Error("removed without reselection should fail")
	}
}

func TestV4Memory_Rule19_ParseError(t *testing.T) {
	if err := CheckModelRemovedTriggersReselection("nope"); err == nil {
		t.Error("parse error should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 20: model unavailable → no silent fallback
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule20_Pass(t *testing.T) {
	evt := ModelSelectionEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "model_selection",
		ModelUnavailable: true,
		SilentFallback:   false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckModelUnavailableNoSilentFallback(string(data)); err != nil {
		t.Errorf("no fallback should pass: %v", err)
	}
}

func TestV4Memory_Rule20_Fail(t *testing.T) {
	evt := ModelSelectionEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "model_selection",
		ModelUnavailable: true,
		SilentFallback:   true,
	}
	data, _ := json.Marshal(evt)
	if err := CheckModelUnavailableNoSilentFallback(string(data)); err == nil {
		t.Error("silent fallback should fail")
	}
}

func TestV4Memory_Rule20_ParseError(t *testing.T) {
	if err := CheckModelUnavailableNoSilentFallback("bad"); err == nil {
		t.Error("parse error should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 21: routing must use program_tag_match
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule21_Pass(t *testing.T) {
	payload := `{"event_type":"routing_decision","routing_method":"program_tag_match"}`
	if err := CheckActionRoutingUsesProgramTag(payload); err != nil {
		t.Errorf("program_tag_match should pass: %v", err)
	}
}

func TestV4Memory_Rule21_Fail(t *testing.T) {
	payload := `{"event_type":"routing_decision","routing_method":"llm_only"}`
	if err := CheckActionRoutingUsesProgramTag(payload); err == nil {
		t.Error("llm_only should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 22: no tool DB in prompt
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule22_Pass(t *testing.T) {
	evt := PromptComposition{
		SchemaVersion:        ContractVersion,
		EventType:            "prompt_composition",
		ToolDatabaseInPrompt: false,
	}
	data, _ := json.Marshal(evt)
	if err := CheckNoToolDBInjectedIntoPrompt(string(data)); err != nil {
		t.Errorf("no tool DB should pass: %v", err)
	}
}

func TestV4Memory_Rule22_Fail(t *testing.T) {
	evt := PromptComposition{
		SchemaVersion:        ContractVersion,
		EventType:            "prompt_composition",
		ToolDatabaseInPrompt: true,
	}
	data, _ := json.Marshal(evt)
	if err := CheckNoToolDBInjectedIntoPrompt(string(data)); err == nil {
		t.Error("tool DB in prompt should fail")
	}
}

func TestV4Memory_Rule22_ParseError(t *testing.T) {
	if err := CheckNoToolDBInjectedIntoPrompt("xxx"); err == nil {
		t.Error("parse error should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 23: security logs must not be reorganized
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule23_Pass(t *testing.T) {
	payload := `{"event_type":"log_operation","target":"security_log","operation":"append"}`
	if err := CheckSecurityLogsNotReorganized(payload); err != nil {
		t.Errorf("append should pass: %v", err)
	}
}

func TestV4Memory_Rule23_FailReorganized(t *testing.T) {
	payload := `{"event_type":"log_operation","target":"security_log","operation":"reorganized"}`
	if err := CheckSecurityLogsNotReorganized(payload); err == nil {
		t.Error("reorganized should fail")
	}
}

func TestV4Memory_Rule23_FailCompacted(t *testing.T) {
	payload := `{"event_type":"log_operation","target":"audit_log","operation":"compacted"}`
	if err := CheckSecurityLogsNotReorganized(payload); err == nil {
		t.Error("compacted should fail")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Rule 24: sub must not be auto-created without confirmation
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_Rule24_Pass_UserConfirmed(t *testing.T) {
	payload := `{"event_type":"sub_creation","auto_created":true,"user_confirmed":true}`
	if err := CheckSubNotCreatedWithoutConfirmation(payload); err != nil {
		t.Errorf("user confirmed should pass: %v", err)
	}
}

func TestV4Memory_Rule24_Pass_NotAutoCreated(t *testing.T) {
	payload := `{"event_type":"sub_creation","auto_created":false,"user_confirmed":false}`
	if err := CheckSubNotCreatedWithoutConfirmation(payload); err != nil {
		t.Errorf("not auto created should pass: %v", err)
	}
}

func TestV4Memory_Rule24_Fail(t *testing.T) {
	payload := `{"event_type":"sub_creation","auto_created":true,"user_confirmed":false}`
	if err := CheckSubNotCreatedWithoutConfirmation(payload); err == nil {
		t.Error("auto created without confirmation should fail")
	}
}
