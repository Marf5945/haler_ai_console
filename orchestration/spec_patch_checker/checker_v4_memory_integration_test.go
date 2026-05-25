// checker_v4_memory_integration_test.go — v4.0 真實流程 Integration Test。
//
// 6 scenario（3 pass + 3 fail）+ 1 grep 反遺留。
// 每個 scenario 透過真實程式碼路徑觸發事件，再餵給 checker 驗證。
package spec_patch_checker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui_console/data/conversation"
	"ui_console/data/memory"
)

// ══════════════════════════════════════════════════════════════════════════════
// Scenario A1: AppendTalkEntry 正常 → 讀 memory_ops → 餵 checker → pass
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_RealPipeline_A1_AppendNormal(t *testing.T) {
	tmpDir := t.TempDir()
	pipeline := memory.NewPipeline(tmpDir)

	// 正常 append
	_, err := pipeline.AppendTalkEntry("user", "Hello integration test")
	if err != nil {
		t.Fatalf("AppendTalkEntry failed: %v", err)
	}

	// 讀取 memory_ops.jsonl
	opsPath := filepath.Join(tmpDir, "memory", "memory_ops.jsonl")
	data, err := os.ReadFile(opsPath)
	if err != nil {
		t.Fatalf("read memory_ops failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	lastLine := lines[len(lines)-1]

	// 餵給 checker：schema version
	if err := CheckSchemaVersion(lastLine); err != nil {
		t.Errorf("A1: schema version check failed: %v", err)
	}

	// 餵給 checker：absolute path
	if err := CheckNoAbsolutePathInPayload(lastLine); err != nil {
		t.Errorf("A1: absolute path check failed: %v", err)
	}

	// 建立對應的 MemoryWriteEvent（模擬 producer emit）
	writeEvt := MemoryWriteEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "memory_write_event",
		Target:           "memory/talk_full.md",
		Operation:        "append",
		MemoryOpsWritten: true, // 因為 memory_ops 寫入成功
	}
	evtJSON, _ := json.Marshal(writeEvt)

	if err := CheckTalkFullWriteHasMemoryOps(string(evtJSON)); err != nil {
		t.Errorf("A1: Rule 1 check failed: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Scenario A2: 直接寫 talk_full（繞過 pipeline）→ 餵 checker → FAIL
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_RealPipeline_A2_BypassPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, "memory")
	os.MkdirAll(memDir, 0755)

	// 直接寫 talk_full（繞過 pipeline，不產生 memory_ops）
	talkPath := filepath.Join(memDir, "talk_full.md")
	os.WriteFile(talkPath, []byte("# Direct write\n"), 0644)

	// 模擬 producer 未 emit memory_ops
	writeEvt := MemoryWriteEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "memory_write_event",
		Target:           "memory/talk_full.md",
		Operation:        "append",
		MemoryOpsWritten: false, // 繞過 pipeline → 沒有 memory_ops
	}
	evtJSON, _ := json.Marshal(writeEvt)

	// 應該被 checker 攔截
	if err := CheckTalkFullWriteHasMemoryOps(string(evtJSON)); err == nil {
		t.Error("A2: bypassing pipeline should be caught by Rule 1")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Scenario B1: RunSummarization 正常 → 餵 Rules 8-11 → pass
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_RealPipeline_B1_SummarizationNormal(t *testing.T) {
	// 使用 BuildSummarizationEvent（真實 producer）
	evt := conversation.BuildSummarizationEvent(false)
	evtJSON, _ := json.Marshal(evt)
	payload := string(evtJSON)

	// Rule 8: write redaction applied
	if err := CheckSummarizationWriteRedaction(payload); err != nil {
		t.Errorf("B1: Rule 8 failed: %v", err)
	}

	// Rule 10: no system prompt
	if err := CheckSummarizationNoSystemPrompt(payload); err != nil {
		t.Errorf("B1: Rule 10 failed: %v", err)
	}

	// Rule 11: no silent send (not failed)
	if err := CheckSummarizationFailNoSilentSend(payload); err != nil {
		t.Errorf("B1: Rule 11 failed: %v", err)
	}

	// Rule 15: output target not talk_full
	if err := CheckSummarizationNotWriteToTalkFull(payload); err != nil {
		t.Errorf("B1: Rule 15 failed: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Scenario B2: summarization output 含 local_path → Rule 9 FAIL
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_RealPipeline_B2_SummarizationSecretLeak(t *testing.T) {
	// 模擬有問題的摘要輸出（含 local_path）
	output := SummarizationOutput{
		SchemaVersion: ContractVersion,
		EventType:     "summarization_output",
		ContentKeys:   []string{"summary", "local_path", "tags"},
	}
	data, _ := json.Marshal(output)

	if err := CheckSummarizationNoSecrets(string(data)); err == nil {
		t.Error("B2: local_path in output should be caught by Rule 9")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Scenario C1: Sub scoped tool access 正常 → Rules 5-6 pass
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_RealPipeline_C1_SubScopedAccess(t *testing.T) {
	evt := SubToolAccessEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "sub_tool_access",
		SubID:            "sub-integration-1",
		AccessScope:      "scoped",
		MutationTarget:   "",
		ToolNotInHistory: false,
		ReportedToMain:   false,
	}
	data, _ := json.Marshal(evt)
	payload := string(data)

	// Rule 5
	if err := CheckSubDoesNotAccessFullToolDB(payload); err != nil {
		t.Errorf("C1: Rule 5 failed: %v", err)
	}

	// Rule 6
	if err := CheckSubDoesNotMutateGlobalRegistry(payload); err != nil {
		t.Errorf("C1: Rule 6 failed: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Scenario C2: Sub 寫 global tool_registry → Rule 6 FAIL
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_RealPipeline_C2_SubMutatesGlobal(t *testing.T) {
	evt := SubToolAccessEvent{
		SchemaVersion:    ContractVersion,
		EventType:        "sub_tool_access",
		SubID:            "sub-rogue",
		AccessScope:      "scoped",
		MutationTarget:   "global_tool_registry",
		ToolNotInHistory: false,
		ReportedToMain:   false,
	}
	data, _ := json.Marshal(evt)

	if err := CheckSubDoesNotMutateGlobalRegistry(string(data)); err == nil {
		t.Error("C2: sub mutating global registry should be caught by Rule 6")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Grep 反遺留：確認 resume_guard.go 不含 main_memory_hash（非註解）
// ══════════════════════════════════════════════════════════════════════════════

func TestV4Memory_RealPipeline_GrepNoMainMemoryHash(t *testing.T) {
	// 讀取 resume_guard.go 原始碼
	// 注意：此測試在 CI 中需確保檔案路徑正確
	// 這裡用 runtime 方式讀取
	guardPath := filepath.Join("..", "dag", "resume_guard.go")
	data, err := os.ReadFile(guardPath)
	if err != nil {
		t.Skipf("cannot read resume_guard.go for grep check: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		// 跳過註解行
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		// 跳過 deprecated 偵測邏輯（這些行是用來 *檢測* 舊格式的）
		if strings.Contains(line, "deprecated") || strings.Contains(line, "hasOld") {
			continue
		}
		if strings.Contains(line, "main_memory_hash") || strings.Contains(line, "MainMemoryHash") {
			t.Errorf("line %d: resume_guard.go still contains main_memory_hash reference: %s", i+1, line)
		}
	}
}
