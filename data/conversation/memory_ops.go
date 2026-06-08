// conversation/memory_ops.go — 記憶操作日誌。
// 以 append-only JSONL 格式記錄所有記憶異動操作，確保可審計性。
//
// 重構：v4.0 遷移至 audit_log.AppendLog[MemoryOp] 共用抽象。
package conversation

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	"ui_console/audit_log"
)

// ──────────────────────────────────────────────
// 資料結構
// ──────────────────────────────────────────────

// MemoryOp 一筆記憶操作日誌記錄。
type MemoryOp struct {
	OpID                string   `json:"op_id"`                 // 唯一操作 ID（UUID）
	Op                  string   `json:"op"`                    // 操作類型（summarize / delete / move / rotate ...）
	From                string   `json:"from"`                  // 來源狀態描述
	To                  string   `json:"to"`                    // 目標狀態描述
	AffectedSentenceIDs []string `json:"affected_sentence_ids"` // 受影響的句子 ID 清單
	BeforeHash          string   `json:"before_hash"`           // 操作前內容 SHA-256
	AfterHash           string   `json:"after_hash"`            // 操作後內容 SHA-256
	Reason              string   `json:"reason"`                // 操作原因說明
	CreatedAt           string   `json:"created_at"`            // RFC3339 時間戳
}

// ──────────────────────────────────────────────
// 建構
// ──────────────────────────────────────────────

// NewMemoryOp 建立一筆新的 MemoryOp，自動產生 OpID 與時間戳。
func NewMemoryOp(opType, from, to string, sentenceIDs []string, beforeHash, afterHash, reason string) MemoryOp {
	return MemoryOp{
		OpID:                uuid.New().String(),
		Op:                  opType,
		From:                from,
		To:                  to,
		AffectedSentenceIDs: sentenceIDs,
		BeforeHash:          beforeHash,
		AfterHash:           afterHash,
		Reason:              reason,
		CreatedAt:           time.Now().Format(time.RFC3339),
	}
}

// ──────────────────────────────────────────────
// 寫入（委託 audit_log）
// ──────────────────────────────────────────────

// WriteMemoryOp 將一筆 MemoryOp 以 JSON 單行格式 append 到指定的 .jsonl 檔案。
// 檔案不存在時自動建立。
func WriteMemoryOp(path string, op MemoryOp) error {
	// SEC-W07（2026-05-24）：拿掉 perm override，回到 audit_log framework 預設
	// （file 0o600 / dir 0o700）。memory ops log 為 AI Console 私有稽核資料，
	// 無外部讀取需求。
	log := audit_log.New[MemoryOp](path)
	return log.Append(op)
}

// ──────────────────────────────────────────────
// 雜湊工具
// ──────────────────────────────────────────────

// ComputeHash 計算字串內容的 SHA-256，回傳十六進位字串。
func ComputeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
