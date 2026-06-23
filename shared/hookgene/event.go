package hookgene

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// RecorderEvent 是寫入 recorder_events.jsonl 的一筆事件。
// 隱私（§3.1.5.18.3）：只記行為摘要，不記輸入內容 / raw output / 檔案 / URL / 完整路徑 /
// credential / token。
type RecorderEvent struct {
	EventID      string    `json:"event_id"`
	SkillID      string    `json:"skill_id"`
	InvocationID string    `json:"invocation_id"`
	Gene         string    `json:"gene"`       // 16 格字串
	HookCount    int       `json:"hook_count"` // 原始 hook 數（補位前）
	BCount       int       `json:"bcount"`     // ㄅ 數量
	Oversized    bool      `json:"oversized"`  // 原始 > 16
	Complete     bool      `json:"complete"`   // 是否完整樣本（incomplete 不入統計分母）
	At           time.Time `json:"at"`
	PrevHash     string    `json:"prev_hash"` // 輕量 hash chain：前一筆 hash
	Hash         string    `json:"hash"`      // 本筆 hash（含 PrevHash + 內容）
}

// hashPayload 是計算 hash 時的內容（刻意排除 Hash 欄位本身）。
type hashPayload struct {
	EventID      string    `json:"event_id"`
	SkillID      string    `json:"skill_id"`
	InvocationID string    `json:"invocation_id"`
	Gene         string    `json:"gene"`
	HookCount    int       `json:"hook_count"`
	BCount       int       `json:"bcount"`
	Oversized    bool      `json:"oversized"`
	Complete     bool      `json:"complete"`
	At           time.Time `json:"at"`
	PrevHash     string    `json:"prev_hash"`
}

// computeHash 以「前一筆 hash + 本筆內容」串成輕量 hash chain（§3.1.5.18.7）。
// 維護：rotate 後新檔第一筆的 PrevHash 接上一檔 tail hash，鏈不中斷。
func (e RecorderEvent) computeHash() string {
	h := sha256.New()
	h.Write([]byte(e.PrevHash))
	payload, _ := json.Marshal(hashPayload{
		EventID: e.EventID, SkillID: e.SkillID, InvocationID: e.InvocationID,
		Gene: e.Gene, HookCount: e.HookCount, BCount: e.BCount,
		Oversized: e.Oversized, Complete: e.Complete, At: e.At, PrevHash: e.PrevHash,
	})
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// geneFromEvent 由事件欄位重建肥大判定所需的最小 Gene（replay 用）。
func geneFromEvent(ev RecorderEvent) Gene {
	return Gene{Oversized: ev.Oversized, BCount: ev.BCount}
}
