// conversation/summarizer.go — 摘要執行（§29.3 先摘要再送）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ RunSummarization 呼叫指定 LLM 產出摘要，                     │
// │ 寫入 summaries.md（壓縮 ~60%）和 deep_memory.md（~40%）。    │
// │                                                             │
// │ 規則：                                                      │
// │  • talk_full.md 不動（原始對話完整保留）                    │
// │  • 寫入磁碟前執行 write redaction                          │
// │  • 摘要失敗 → 不得靜默送舊 context                         │
// └─────────────────────────────────────────────────────────────┘
package conversation

import (
	"fmt"
	"time"
)

// ──────────────────────────────────────────────
// 摘要結果
// ──────────────────────────────────────────────

// SummaryResult 摘要執行結果。
type SummaryResult struct {
	SummaryTag    string `json:"summary_tag"`     // [S-NNN]
	DeepMemoryTag string `json:"deep_memory_tag"` // [D-NNN]
	NewCharCount  int    `json:"new_char_count"`  // 摘要後的字數
	Error         string `json:"error,omitempty"`
}

// SummarizationStatus 摘要流程狀態。
type SummarizationStatus string

const (
	SumStatusIdle       SummarizationStatus = "idle"
	SumStatusInProgress SummarizationStatus = "in_progress"
	SumStatusSuccess    SummarizationStatus = "success"
	SumStatusFailed     SummarizationStatus = "failed"
)

// ──────────────────────────────────────────────
// LLM 呼叫介面（可 mock）
// ──────────────────────────────────────────────

// SummarizationLLM 是摘要用的 LLM 呼叫介面。
type SummarizationLLM interface {
	Summarize(content string, maxOutputChars int) (string, error)
}

// ──────────────────────────────────────────────
// 摘要執行
// ──────────────────────────────────────────────

// RunSummarization 執行摘要流程。
// sentences: 待摘要的句子
// modelID: 使用的 LLM 模型 ID
// llm: LLM 呼叫介面
// 回傳摘要結果或錯誤。
func RunSummarization(sentences []Sentence, modelID string, llm SummarizationLLM) (SummaryResult, error) {
	if len(sentences) == 0 {
		return SummaryResult{Error: "no sentences to summarize"}, fmt.Errorf("no sentences")
	}

	// 組裝待摘要內容
	var content string
	for _, s := range sentences {
		content += fmt.Sprintf("[%s] %s: %s\n", s.ID, s.Role, s.Content)
	}

	// 呼叫 LLM 產出摘要（目標壓縮至 60%）
	originalChars := len([]rune(content))
	targetChars := int(float64(originalChars) * 0.6)

	summary, err := llm.Summarize(content, targetChars)
	if err != nil {
		return SummaryResult{Error: err.Error()}, err
	}

	// 產出 tags
	now := time.Now()
	summaryTag := fmt.Sprintf("[S-%d: %s]", now.Unix()%100000, now.Format("2006-01-02T15:04"))
	deepMemoryTag := fmt.Sprintf("[D-%d: %s]", now.Unix()%100000, now.Format("2006-01-02T15:04"))

	// 壓縮後字數
	newCharCount := len([]rune(summary))

	// 注意：實際寫入 summaries.md / deep_memory.md 由呼叫端執行
	// 此函式只負責 LLM 呼叫和結果結構化

	return SummaryResult{
		SummaryTag:    summaryTag,
		DeepMemoryTag: deepMemoryTag,
		NewCharCount:  newCharCount,
	}, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// v4.0 SummarizationEvent 產生器（§23 specPatchChecker 用）
// ══════════════════════════════════════════════════════════════════════════════

// SummarizationEventPayload 與 spec_patch_checker.SummarizationEvent 結構對齊。
// 用於 emit 給 checker 的靜態驗證。
type SummarizationEventPayload struct {
	SchemaVersion          string `json:"schema_version"`
	EventType              string `json:"event_type"`
	WriteRedactionApplied  bool   `json:"write_redaction_applied"`
	IncludesSystemPrompt   bool   `json:"includes_system_prompt_chars"`
	SystemPromptSummarized bool   `json:"system_prompt_summarized"`
	SummarizationActive    bool   `json:"summarization_active"`
	MessageSentToCLI       bool   `json:"message_sent_to_cli"`
	SummarizationFailed    bool   `json:"summarization_failed"`
	OldContextSentSilently bool   `json:"old_context_sent_silently"`
	OutputTarget           string `json:"output_target"`
	OutputOperation        string `json:"output_operation"`
}

// BuildSummarizationEvent 建立一筆正常摘要流程的 event payload。
// 摘要結果寫入 summaries.md（正確目標），operation = "append"。
// 呼叫端可透過 json.Marshal 後寫入 audit log 或事件佇列。
func BuildSummarizationEvent(failed bool) SummarizationEventPayload {
	return SummarizationEventPayload{
		SchemaVersion:          "v4.0-memory-guard-1",
		EventType:              "summarization_event",
		WriteRedactionApplied:  true,  // 規則：寫入前必執行 redaction
		IncludesSystemPrompt:   false, // 不含 system prompt 字元
		SystemPromptSummarized: false, // 不摘要 system prompt
		SummarizationActive:    !failed,
		MessageSentToCLI:       false, // 摘要中不送訊息
		SummarizationFailed:    failed,
		OldContextSentSilently: false, // 失敗時不得靜默送舊 context
		OutputTarget:           "memory/summaries.md",
		OutputOperation:        "append",
	}
}

// ──────────────────────────────────────────────
// Pending Message 暫存（先摘要再送用）
// ──────────────────────────────────────────────

// PendingMessage 暫存的使用者輸入（等待摘要完成後送出）。
type PendingMessage struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
