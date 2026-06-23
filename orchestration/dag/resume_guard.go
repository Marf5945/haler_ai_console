// dag/resume_guard.go — Resume Guard 四項 hash 比對（§19.2–§19.3）。
// 偵測 app restart 後環境是否已變更，決定能否安全恢復 DAG 執行。
// Guard 範圍：sub_memory + tool_registry + risk_policy + source_trust_allowlist。
// v4.0 變更：main_memory_hash 已移除，改用 sub_memory_hash（委派式記憶模型）。
package dag

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ──────────────────────────────────────────────
// Guard Hash 結構
// ──────────────────────────────────────────────

// GuardHashes 儲存四項 guard hash。
// v4.0：main_memory_hash 已替換為 sub_memory_hash（委派式記憶模型）。
type GuardHashes struct {
	SubMemoryHash    string `json:"sub_memory_hash"`
	ToolRegistryHash string `json:"tool_registry_hash"`
	RiskPolicyHash   string `json:"risk_policy_hash"`
	SourceTrustHash  string `json:"source_trust_hash"`
}

// GuardCheckResult Resume Guard 檢查結果。
type GuardCheckResult struct {
	Safe          bool     `json:"safe"`
	ChangedFields []string `json:"changed_fields"` // 哪些 hash 不匹配
	BlockReason   string   `json:"block_reason"`
}

// ──────────────────────────────────────────────
// Hash 計算
// ──────────────────────────────────────────────

// ComputeCurrentHashes 計算當前環境的四項 guard hash。
// projectRoot 為專案資料根目錄。
// v4.0：使用 sub_memory_hash（memory_ops.jsonl）取代 main_memory_hash。
func ComputeCurrentHashes(projectRoot string) GuardHashes {
	return GuardHashes{
		SubMemoryHash:    computeFileHash(filepath.Join(projectRoot, "memory", "memory_ops.jsonl")),
		ToolRegistryHash: computeFileHash(filepath.Join(projectRoot, "tool_registry", "registry.json")),
		RiskPolicyHash:   computeFileHash(filepath.Join(projectRoot, "risk_policy", "policy.json")),
		SourceTrustHash:  computeFileHash(filepath.Join(projectRoot, "source_trust", "project_source_allowlist.json")),
	}
}

// CombinedHash 將四項 hash 合併為單一 hash。
// 用於 DAGRun.GuardHash 欄位。
// v4.0：使用 sub_memory_hash 取代 main_memory_hash。
func (g GuardHashes) CombinedHash() string {
	// 固定排序確保一致性
	parts := []string{
		"sub_mem:" + g.SubMemoryHash,
		"tool:" + g.ToolRegistryHash,
		"risk:" + g.RiskPolicyHash,
		"trust:" + g.SourceTrustHash,
	}
	sort.Strings(parts)
	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash)
}

// ──────────────────────────────────────────────
// Guard 檢查
// ──────────────────────────────────────────────

// CheckResumeGuard 比對 DAGRun 建立時的 guard hash 與當前環境。
// 任何一項不匹配 → 不安全，需阻擋恢復。
func CheckResumeGuard(run *DAGRun, current GuardHashes) GuardCheckResult {
	savedHash := run.GuardHash
	currentHash := current.CombinedHash()

	if savedHash == currentHash {
		return GuardCheckResult{Safe: true}
	}

	// 找出哪些欄位變更（需要拆解比對）
	var changed []string

	// 由於 CombinedHash 是合併的，我們做粗粒度比對
	// 實際上只要 combined hash 不同就代表至少一項變更
	changed = append(changed, "guard_hash_mismatch")

	return GuardCheckResult{
		Safe:          false,
		ChangedFields: changed,
		BlockReason:   "RESUME_GUARD_HASH_CHANGED",
	}
}

// CheckResumeGuardDetailed 詳細比對各項 hash。
// 需要 DAGRun 建立時的完整 GuardHashes（從 persisted 檔案載入）。
func CheckResumeGuardDetailed(saved, current GuardHashes) GuardCheckResult {
	var changed []string

	if saved.SubMemoryHash != current.SubMemoryHash {
		changed = append(changed, "sub_memory")
	}
	if saved.ToolRegistryHash != current.ToolRegistryHash {
		changed = append(changed, "tool_registry")
	}
	if saved.RiskPolicyHash != current.RiskPolicyHash {
		changed = append(changed, "risk_policy")
	}
	if saved.SourceTrustHash != current.SourceTrustHash {
		changed = append(changed, "source_trust_allowlist")
	}

	if len(changed) == 0 {
		return GuardCheckResult{Safe: true}
	}

	return GuardCheckResult{
		Safe:          false,
		ChangedFields: changed,
		BlockReason:   fmt.Sprintf("RESUME_GUARD_HASH_CHANGED: %s", strings.Join(changed, ", ")),
	}
}

// ──────────────────────────────────────────────
// Pre-execution Guard（§19.3）
// ──────────────────────────────────────────────

// CheckPreExecutionGuard 在高風險節點執行前再次檢查 guard。
// 僅對 risk_class >= high_non_destructive 的節點執行。
func CheckPreExecutionGuard(node *DAGNode, projectRoot string, savedHashes GuardHashes) GuardCheckResult {
	// 僅高風險節點需要 pre-execution check
	if !isHighRiskClass(node.RiskClass) {
		return GuardCheckResult{Safe: true}
	}

	current := ComputeCurrentHashes(projectRoot)
	return CheckResumeGuardDetailed(savedHashes, current)
}

// ──────────────────────────────────────────────
// Guard Hash 持久化
// ──────────────────────────────────────────────

// SaveGuardHashes 將建立 DAGRun 時的 guard hashes 持久化。
// 用於 resume 時的詳細比對。
func SaveGuardHashes(projectRoot string, runID string, hashes GuardHashes) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, runID+"_guard.json")
	return os.WriteFile(path, data, 0o600)
}

// LoadGuardHashes 載入 DAGRun 建立時的 guard hashes。
// v4.0：若偵測到舊格式（含 main_memory_hash 欄位），回傳 deprecated error，
// 不靜默接受 zero value。呼叫端應視為無效舊資料，需刪除並重建 DAGRun。
func LoadGuardHashes(projectRoot string, runID string) (*GuardHashes, error) {
	path := filepath.Join(projectRoot, "dag_runs", runID+"_guard.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// ── 偵測舊格式：main_memory_hash 欄位存在 → 明確失效 ──
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if _, hasOld := raw["main_memory_hash"]; hasOld {
		return nil, fmt.Errorf(
			"deprecated guard format: file %s contains main_memory_hash (pre-v4.0); "+
				"delete this file and recreate the DAGRun", path)
	}

	var hashes GuardHashes
	if err := json.Unmarshal(data, &hashes); err != nil {
		return nil, err
	}
	return &hashes, nil
}

// ──────────────────────────────────────────────
// 內部輔助
// ──────────────────────────────────────────────

// computeFileHash 計算檔案的 SHA-256 hash。
// 檔案不存在時回傳 "empty"。
func computeFileHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "empty"
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// isHighRiskClass 判斷是否為高風險等級。
var highRiskClasses = map[string]bool{
	"high_non_destructive":         true,
	"user_owned_asset_destructive": true,
	"subagent_lifecycle_removal":   true,
	"security_boundary_rewrite":    true,
	"critical_runtime_action":      true,
}

func isHighRiskClass(riskClass string) bool {
	return highRiskClasses[riskClass]
}
