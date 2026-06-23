// risk_gate.go — Scheduler 風險閘門，決定 job 能否自動執行。
// 使用字串比較，不直接 import domain/risk，避免循環依賴。
package scheduler

// ShouldAutoExecute: Low/Medium 可自動跑。
func ShouldAutoExecute(riskClass string) bool {
	return riskClass == "low" || riskClass == "medium" || riskClass == ""
}

// ShouldRequireAuth: HighNonDestructive 需要有效授權才能自動跑。
func ShouldRequireAuth(riskClass string) bool {
	return riskClass == "high_non_destructive"
}

// ShouldBlockExecution: UserOwnedAssetDestructive 以上不可自動執行。
func ShouldBlockExecution(riskClass string) bool {
	switch riskClass {
	case "user_owned_asset_destructive", "subagent_lifecycle_removal",
		"security_boundary_rewrite", "critical_runtime_action":
		return true
	}
	return false
}

// NeedsPayloadRecheck: Medium 需要輕量 payload hash 驗證。
func NeedsPayloadRecheck(riskClass string) bool {
	return riskClass == "medium"
}
