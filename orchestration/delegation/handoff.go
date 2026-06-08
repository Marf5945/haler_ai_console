// delegation/handoff.go — 記憶委派打包。
// 當主代理將任務記憶委派給 sub 時，建立標準化的 handoff 套件。
package delegation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// HandoffPackage 委派打包內容。
type HandoffPackage struct {
	SubID           string   `json:"sub_id"`
	SentenceIDs     []string `json:"sentence_ids"`      // 委派的句子 ID 清單
	TalkFullSegment string   `json:"talk_full_segment"` // 對話片段全文
	Summaries       string   `json:"summaries"`         // 摘要內容
	DeepMemoryRefs  []string `json:"deep_memory_refs"`  // 深層記憶參考清單
	CreatedAt       string   `json:"created_at"`        // RFC3339 建立時間
}

// HandoffMemOp 本地複製的記憶操作結構，避免循環 import。
type HandoffMemOp struct {
	OpID                string   `json:"op_id"`
	Op                  string   `json:"op"`
	From                string   `json:"from"`
	To                  string   `json:"to"`
	AffectedSentenceIDs []string `json:"affected_sentence_ids"`
	BeforeHash          string   `json:"before_hash"`
	AfterHash           string   `json:"after_hash"`
	Reason              string   `json:"reason"`
	CreatedAt           string   `json:"created_at"`
}

// HandoffResult 委派建立的結果。
type HandoffResult struct {
	SubDir   string       `json:"sub_dir"`   // 建立的 sub 目錄完整路徑
	MemoryOp HandoffMemOp `json:"memory_op"` // 寫入的記憶操作記錄
}

// ──────────────────────────────────────────────
// 目錄結構建立
// ──────────────────────────────────────────────

// CreateSubDirectory 建立 sub 的完整目錄樹。
// 結構：subagents/callable/[subID]/{memory,dag,tool_history}
func CreateSubDirectory(projectRoot string, subID string) (string, error) {
	subBase := filepath.Join(projectRoot, "subagents", "callable", subID)

	dirs := []string{
		filepath.Join(subBase, "memory"),
		filepath.Join(subBase, "dag"),
		filepath.Join(subBase, "tool_history"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("建立目錄失敗 %s: %w", dir, err)
		}
	}

	return subBase, nil
}

// ──────────────────────────────────────────────
// Handoff 建立
// ──────────────────────────────────────────────

// CreateHandoff 建立委派套件，將主代理的記憶片段交給指定 sub。
// 寫入 talk_full.md、summaries.md、deep_memory.md、index.json 與 memory_ops.jsonl。
func CreateHandoff(
	projectRoot string,
	subID string,
	sentenceIDs []string,
	talkContent string,
	summaryContent string,
	deepMemoryContent string,
) (*HandoffResult, error) {
	// 建立目錄結構
	subBase, err := CreateSubDirectory(projectRoot, subID)
	if err != nil {
		return nil, err
	}

	memDir := filepath.Join(subBase, "memory")
	now := time.Now()
	timestamp := now.Format(time.RFC3339)

	// 寫入 talk_full.md
	if err := writeFile(filepath.Join(memDir, "talk_full.md"), talkContent); err != nil {
		return nil, fmt.Errorf("寫入 talk_full.md 失敗: %w", err)
	}

	// 寫入 summaries.md
	if err := writeFile(filepath.Join(memDir, "summaries.md"), summaryContent); err != nil {
		return nil, fmt.Errorf("寫入 summaries.md 失敗: %w", err)
	}

	// 寫入 deep_memory.md
	if err := writeFile(filepath.Join(memDir, "deep_memory.md"), deepMemoryContent); err != nil {
		return nil, fmt.Errorf("寫入 deep_memory.md 失敗: %w", err)
	}

	// 寫入空的 index.json
	if err := writeFile(filepath.Join(memDir, "index.json"), "[]"); err != nil {
		return nil, fmt.Errorf("寫入 index.json 失敗: %w", err)
	}

	// 建立 HandoffPackage 作為 memory_ops.jsonl 的 delegate 記錄
	pkg := HandoffPackage{
		SubID:           subID,
		SentenceIDs:     sentenceIDs,
		TalkFullSegment: talkContent,
		Summaries:       summaryContent,
		DeepMemoryRefs:  []string{},
		CreatedAt:       timestamp,
	}

	// 建立 memory_ops.jsonl 的操作記錄
	memOp := HandoffMemOp{
		OpID:                fmt.Sprintf("handoff-%s-%d", subID, now.UnixNano()),
		Op:                  "delegate_to_sub",
		From:                "main",
		To:                  subID,
		AffectedSentenceIDs: sentenceIDs,
		BeforeHash:          "",
		AfterHash:           "",
		Reason:              fmt.Sprintf("委派任務記憶至 sub: %s", subID),
		CreatedAt:           timestamp,
	}

	// 寫入 memory_ops.jsonl
	if err := appendMemoryOp(filepath.Join(memDir, "memory_ops.jsonl"), pkg, memOp); err != nil {
		return nil, fmt.Errorf("寫入 memory_ops.jsonl 失敗: %w", err)
	}

	return &HandoffResult{
		SubDir:   subBase,
		MemoryOp: memOp,
	}, nil
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

// writeFile 建立或覆寫指定路徑的檔案內容。
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

// appendMemoryOp 將 HandoffPackage 與操作記錄序列化後 append 到 .jsonl 檔案。
func appendMemoryOp(path string, pkg HandoffPackage, op HandoffMemOp) error {
	// 組合寫入資料：package 摘要 + op 記錄
	type jsonlEntry struct {
		Package HandoffPackage `json:"package"`
		Op      HandoffMemOp   `json:"op"`
	}
	entry := jsonlEntry{Package: pkg, Op: op}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("序列化 memory op 失敗: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("開啟 memory_ops.jsonl 失敗: %w", err)
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}
