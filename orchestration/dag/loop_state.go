// dag/loop_state.go — chat_route 節點內 tool loop 的輪次級狀態（v3.1.6 M1）。
// 與 DAGRun 分開存 sidecar 檔（dag_runs/<runID>.<nodeID>.loop.json），
// 避免 .full.json 變肥與牽動前端 Wails model；寫入共用 fullRunWriteMu。
package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const suffixLoop = ".loop.json"

// PendingInput 記錄「正在等補參數的工具」。這是 Controller 狀態：
// 模型回 輸入ㄌ{...}ㄌ待命 只是回填，不靠模型重新聲明工具名。
type PendingInput struct {
	Tool          string          `json:"tool"`
	SchemaID      string          `json:"schema_id"`
	PartialArgs   json.RawMessage `json:"partial_args,omitempty"`
	MissingFields []string        `json:"missing_fields,omitempty"`
}

// ObservationRecord 是 loop 內一輪的觀察。三份分工：
// raw 由 RawRef 指向（UI/audit/重建），CompactText 是 deterministic 摘要，
// SanitizedText 是唯一可進下一輪 prompt 的版本。
type ObservationRecord struct {
	Kind          string `json:"kind"` // tool / critic / user_input / system
	Action        string `json:"action,omitempty"`
	Target        string `json:"target,omitempty"`
	CompactText   string `json:"compact_text,omitempty"`
	SanitizedText string `json:"sanitized_text,omitempty"`
	RawRef        string `json:"raw_ref,omitempty"`
	Hash          string `json:"hash,omitempty"`
	Truncated     bool   `json:"truncated,omitempty"`
}

// LoopState 是單一節點的 loop 持久化狀態。
type LoopState struct {
	RunID          string              `json:"run_id"`
	NodeID         string              `json:"node_id"`
	TraceID        string              `json:"trace_id,omitempty"`
	Iteration      int                 `json:"iteration"`
	PendingInput   *PendingInput       `json:"pending_input,omitempty"`
	Observations   []ObservationRecord `json:"observations,omitempty"`
	SeenSignatures map[string]int      `json:"seen_signatures,omitempty"`
	CriticRounds   int                 `json:"critic_rounds"` // M2 用；暫停恢復不歸零
	ExtraRounds    int                 `json:"extra_rounds"`  // 使用者補充後追加的輪數額度
	LoopRevision   int                 `json:"loop_revision"`
	UpdatedAt      string              `json:"updated_at,omitempty"`
}

// HashObservation 計算 observation 內容指紋（丟內容後仍可比對是否處理過）。
func HashObservation(action, target, rawText string) string {
	h := sha256.Sum256([]byte(action + "\x1f" + target + "\x1f" + rawText))
	return hex.EncodeToString(h[:8])
}

// RecordSignature 累計 action 簽名出現次數，回傳累計值（防原地打轉）。
func (s *LoopState) RecordSignature(signature string) int {
	if s.SeenSignatures == nil {
		s.SeenSignatures = map[string]int{}
	}
	s.SeenSignatures[signature]++
	return s.SeenSignatures[signature]
}

// SanitizedBytes 回傳目前累計餵給 LLM 的 observation 大小（雙上限之一）。
func (s *LoopState) SanitizedBytes() int {
	total := 0
	for _, o := range s.Observations {
		total += len(o.SanitizedText)
	}
	return total
}

// TrimToBudget 超預算時丟最舊 observation 的內文，但保留 Hash/RawRef 與短摘要。
func (s *LoopState) TrimToBudget(budget int) {
	for i := range s.Observations {
		if s.SanitizedBytes() <= budget {
			return
		}
		o := &s.Observations[i]
		if o.SanitizedText == "" {
			continue
		}
		// 留前 80 bytes 當線索，其餘靠 Hash/RawRef 追溯；收尾退到 rune 邊界。
		keep := o.SanitizedText
		if len(keep) > 80 {
			cut := 80
			for cut > 0 && (keep[cut]&0xC0) == 0x80 {
				cut--
			}
			keep = keep[:cut]
		}
		o.SanitizedText = "（內容已修剪）" + keep
		o.Truncated = true
	}
}

func loopStatePath(projectRoot, runID, nodeID string) string {
	return filepath.Join(projectRoot, "dag_runs", runID+"."+nodeID+suffixLoop)
}

// LoadLoopState 載入 sidecar；檔案不存在或內容與 run/node 不符 → 回全新狀態
// （recovery policy：對不上就重起，observations 由呼叫端決定是否保留）。
func LoadLoopState(projectRoot, runID, nodeID string) *LoopState {
	fresh := &LoopState{RunID: runID, NodeID: nodeID}
	data, err := os.ReadFile(loopStatePath(projectRoot, runID, nodeID))
	if err != nil {
		return fresh
	}
	var state LoopState
	if json.Unmarshal(data, &state) != nil || state.RunID != runID || state.NodeID != nodeID {
		return fresh
	}
	return &state
}

// SaveLoopStateLocked 原子寫入 sidecar（temp+rename），與 run 寫入共用同一把鎖。
func SaveLoopStateLocked(projectRoot string, state *LoopState) error {
	if state == nil || strings.TrimSpace(state.RunID) == "" || strings.TrimSpace(state.NodeID) == "" {
		return fmt.Errorf("loop state: missing run/node id")
	}
	fullRunWriteMu.Lock()
	defer fullRunWriteMu.Unlock()
	state.LoopRevision++
	state.UpdatedAt = time.Now().Format(time.RFC3339)
	dir := filepath.Join(projectRoot, "dag_runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := loopStatePath(projectRoot, state.RunID, state.NodeID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, loopStatePath(projectRoot, state.RunID, state.NodeID))
}

// DeleteLoopStatesForRun 刪除一個 run 的所有 loop sidecar（run cleanup 一併呼叫）。
func DeleteLoopStatesForRun(projectRoot, runID string) {
	dir := filepath.Join(projectRoot, "dag_runs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasPrefix(name, runID+".") && strings.HasSuffix(name, suffixLoop) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}
