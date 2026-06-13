package controlled_trust

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"ui_console/audit_log"
)

// TrustLogEntry is one immutable record in the append-only controlled trust log.
// Corresponds to schema #54 / spec #36 in TASKS_1_2.md.
//
// INVARIANTS:
//   - final_risk_changed MUST always be false — trust decisions never lower final_risk.
//   - hard_rules_modified MUST always be false — hard rules are never modified.
type TrustLogEntry struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"`
	ScopeHash         string    `json:"scope_hash"`
	EntryHash         string    `json:"entry_hash"`
	PreviousEntryHash string    `json:"previous_entry_hash"`
	FinalRiskChanged  bool      `json:"final_risk_changed"`  // always false
	HardRulesModified bool      `json:"hard_rules_modified"` // always false
	CreatedAt         time.Time `json:"created_at"`
}

const genesisEntryHash = "0000000000000000000000000000000000000000000000000000000000000000"

// TrustLog is the append-only log for all controlled trust decisions.
// 重構：v4.0 委託 audit_log.AppendLog[TrustLogEntry] 共用抽象。
type TrustLog struct {
	inner *audit_log.AppendLog[TrustLogEntry]
}

func NewTrustLog(trustDir string) *TrustLog {
	p := filepath.Join(trustDir, "controlled_trust_log.jsonl")
	return &TrustLog{
		inner: audit_log.New[TrustLogEntry](
			p,
			// 不變量檢查：FinalRiskChanged 和 HardRulesModified 必須為 false
			audit_log.WithBeforeAppend[TrustLogEntry](func(e *TrustLogEntry) error {
				if e.FinalRiskChanged {
					return fmt.Errorf("trust_log: final_risk_changed must be false; trust decisions must never lower final_risk")
				}
				if e.HardRulesModified {
					return fmt.Errorf("trust_log: hard_rules_modified must be false; hard rules are immutable")
				}
				// 自動產生 ID 和時間戳
				e.ID = fmt.Sprintf("tl-%d", time.Now().UnixNano())
				e.CreatedAt = time.Now()
				return nil
			}),
			// Hash chain 連結
			audit_log.WithHashChain[TrustLogEntry](
				genesisEntryHash,
				// beforeWrite: 設定 PreviousEntryHash 並計算 EntryHash
				func(entry *TrustLogEntry, state *audit_log.ChainState) {
					entry.PreviousEntryHash = state.LastHash
					entry.EntryHash = computeEntryHash(*entry)
				},
				// afterWrite: 更新 lastHash
				func(entry *TrustLogEntry, state *audit_log.ChainState) {
					state.LastHash = entry.EntryHash
				},
				// loadTail: 從最後一筆恢復 lastHash
				func(entry *TrustLogEntry, state *audit_log.ChainState) {
					state.LastHash = entry.EntryHash
				},
			),
		),
	}
}

// Append adds a new immutable entry to the trust log.
// final_risk_changed and hard_rules_modified must always be false;
// if they are not, Append returns an error rather than logging invalid state.
func (tl *TrustLog) Append(entry TrustLogEntry) error {
	return tl.inner.Append(entry)
}

func computeEntryHash(e TrustLogEntry) string {
	raw := fmt.Sprintf("%s|%s|%s|%v|%v|%s",
		e.ID, e.Type, e.ScopeHash, e.FinalRiskChanged, e.HardRulesModified,
		e.PreviousEntryHash)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func computeScopeHash(scope OverrideScope) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		scope.DAGRunID, scope.WorkspaceID, scope.OperationFamily,
		scope.TargetHashSet, scope.PlanHash, scope.RiskPolicyHash,
		scope.ToolRegistryHash, scope.DeviceProfileID)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
